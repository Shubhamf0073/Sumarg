// Package app is the start-up and shutdown sequence every service binary shares, so cmd/<svc>/main.go only
// declares what is different (name, ports, routes):
//
//	<binary> serve [--mode=api|worker|all]   load config, run the public and internal listeners, stop on SIGTERM
//	<binary> openapi                         print the OpenAPI 3.1 document as JSON (no config needed)
//	<binary> healthcheck                     GET 127.0.0.1:<internal port>/readyz; exit 0 if ready, else 1
//
// healthcheck is what Docker runs (compose: test ["CMD","/app","healthcheck"]): distroless images have no shell or
// curl. It loads no config, so it needs no secrets.
package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/sync/errgroup"

	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/config"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/httpx"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/obs"
)

// ShutdownGrace is how long requests in progress get after SIGTERM, and CloseGrace is how long Runtime.Close then
// gets. They run one after the other, and Docker sends SIGKILL 10 s after SIGTERM by default (compose
// stop_grace_period), so their sum must stay below that.
const (
	ShutdownGrace = 6 * time.Second
	CloseGrace    = 2 * time.Second
)

// HealthcheckTimeout bounds `healthcheck`. Docker's own check timeout is longer, so a hung service reports
// unhealthy from here rather than being killed by Docker.
const HealthcheckTimeout = 3 * time.Second

// Configured is what this package needs from a binary's config struct: the shared settings every service has.
// config.Base satisfies it, and so does any struct that embeds config.Base.
type Configured interface{ BaseConfig() config.Base }

// Service describes one binary. C is the binary's config struct (config.Base for a binary with no extra settings).
type Service[C Configured] struct {
	Name         string // must equal SVC_NAME in the service's environment ("schoolfleet")
	PublicAddr   string // ":8080"; "" = no public listener (docs)
	InternalAddr string // ":8090"; always present, it serves the health endpoints
	// Public mounts the public routes. Only called when PublicAddr is set.
	Public func(r chi.Router)
	// Modes, if set, are the values `serve --mode` accepts; the first is the default.
	Modes []string
	// Setup, if set, runs once in `serve`, after config and logging are ready and before any listener starts. It
	// opens what the service needs (database pools, clients) and returns what to wire in. An error stops the
	// service with exit code 1, and Close is not called, so Setup must clean up after itself when it fails.
	Setup func(ctx context.Context, cfg C, log *slog.Logger) (*Runtime, error)
	// OpenAPI, if set, returns the API's OpenAPI document as JSON for `<binary> openapi` (make gen-api). It runs
	// with no config, database or listener, so it works in CI and on a fresh checkout.
	OpenAPI func() ([]byte, error)
}

// Runtime is what Setup hands back to the shared start-up sequence. Every field is optional.
type Runtime struct {
	// Routes mounts more public routes, after Service.Public. Only called when the public listener runs.
	Routes func(r chi.Router)
	// Ready are extra readiness checks: /readyz answers 503 while any of them fails (e.g. the database ping).
	Ready []httpx.Check
	// Workers are background loops run in worker mode (and all). Each runs until ctx is cancelled and then returns
	// nil; an error stops the whole service, like a failed listener.
	Workers []func(ctx context.Context) error
	// Close releases what Setup opened. It runs once, after the listeners have stopped, with a context that
	// expires after CloseGrace.
	Close func(ctx context.Context) error
}

// Mode values for `schoolfleet serve` (docs/02 A4): production runs api, worker and ingest as three containers from
// one image; dev runs all in one process.
const (
	ModeAPI    = "api"
	ModeWorker = "worker"
	ModeIngest = "ingest" // MQTT broker for the bus units + telemetry pipeline (docs/13)
	ModeAll    = "all"
)

// Main runs the binary and exits with its status. It never returns.
func Main[C Configured](s Service[C]) {
	os.Exit(run(context.Background(), s, os.Args[1:], os.Stdout, os.Stderr, config.Load[C], nil))
}

// run is Main without process globals, for tests. listening, if non-nil, receives each listener's name and bound
// address once it is accepting connections.
func run[C Configured](ctx context.Context, s Service[C], args []string, stdout, stderr io.Writer,
	load func() (C, error), listening func(name, addr string)) int {
	if len(args) == 0 {
		usage(stderr, s)
		return 2
	}
	switch args[0] {
	case "serve":
		return serve(ctx, s, args[1:], stdout, stderr, load, listening)
	case "openapi":
		if s.OpenAPI == nil {
			fmt.Fprintf(stderr, "%s openapi: this binary has no HTTP API\n", s.Name)
			return 1
		}
		doc, err := s.OpenAPI()
		if err != nil {
			fmt.Fprintf(stderr, "%s openapi: %v\n", s.Name, err)
			return 1
		}
		if _, err := stdout.Write(append(doc, '\n')); err != nil {
			fmt.Fprintf(stderr, "%s openapi: %v\n", s.Name, err)
			return 1
		}
		return 0
	case "healthcheck":
		if err := Healthcheck(ctx, s.InternalAddr); err != nil {
			fmt.Fprintf(stderr, "%s healthcheck: %v\n", s.Name, err)
			return 1
		}
		return 0
	default:
		usage(stderr, s)
		return 2
	}
}

func usage[C Configured](w io.Writer, s Service[C]) {
	mode := ""
	if len(s.Modes) > 0 {
		mode = " [--mode=" + strings.Join(s.Modes, "|") + "]"
	}
	fmt.Fprintf(w, "usage: %s serve%s | openapi | healthcheck\n", s.Name, mode)
}

func serve[C Configured](ctx context.Context, s Service[C], args []string, stdout, stderr io.Writer,
	load func() (C, error), listening func(name, addr string)) int {
	fs := flag.NewFlagSet(s.Name+" serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var mode string
	if len(s.Modes) > 0 {
		fs.StringVar(&mode, "mode", s.Modes[0], "what to run: "+strings.Join(s.Modes, "|"))
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "%s serve: unexpected argument %q\n", s.Name, fs.Arg(0))
		return 2
	}
	if len(s.Modes) > 0 && !slices.Contains(s.Modes, mode) {
		fmt.Fprintf(stderr, "%s serve: --mode must be one of %s, got %q\n", s.Name, strings.Join(s.Modes, "|"), mode)
		return 2
	}

	full, err := load()
	if err != nil {
		// Config errors name the variable and never print its value (platform/config).
		fmt.Fprintf(stderr, "%s: %v\n", s.Name, err)
		return 1
	}
	cfg := full.BaseConfig()
	// A service started with another service's env file would run with that service's database role and keys.
	if cfg.ServiceName != s.Name {
		fmt.Fprintf(stderr, "%s: SVC_NAME is %q; this binary is %q (wrong env file?)\n", s.Name, cfg.ServiceName, s.Name)
		return 1
	}
	log := obs.NewLogger(stdout, cfg)
	slog.SetDefault(log)

	// Go note: NotifyContext returns a context that is cancelled on SIGINT/SIGTERM. Everything below watches it,
	// so one signal starts the whole graceful shutdown.
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	var rt Runtime
	if s.Setup != nil {
		got, err := s.Setup(ctx, full, log)
		if err != nil {
			// Config types redact themselves, so a Setup error built from them carries no secret.
			log.Error("setup failed", slog.Any("error", err))
			return 1
		}
		if got != nil {
			rt = *got
		}
	}
	// Runs on every return below: after the listeners stop, or when one fails to start.
	defer func() {
		if rt.Close == nil {
			return
		}
		cctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), CloseGrace)
		defer cancel()
		if err := rt.Close(cctx); err != nil {
			log.Error("close failed", slog.Any("error", err))
		}
	}()

	type listener struct {
		name, addr string
		handler    http.Handler
	}
	ls := []listener{{"internal", s.InternalAddr, httpx.NewInternalRouter(log, rt.Ready...)}}
	runAPI := mode == "" || mode == ModeAPI || mode == ModeAll
	if runAPI && s.PublicAddr != "" {
		r := httpx.NewRouter(log)
		if s.Public != nil {
			s.Public(r)
		}
		if rt.Routes != nil {
			rt.Routes(r)
		}
		ls = append(ls, listener{"public", s.PublicAddr, r})
	}
	runWorkers := mode == ModeWorker || mode == ModeAll
	if runWorkers && len(rt.Workers) == 0 {
		log.Info("worker mode: no job workers are registered")
	}
	if mode == ModeIngest || mode == ModeAll {
		log.Info("ingest mode: the MQTT broker and telemetry pipeline arrive in M3")
	}

	// Go note: an errgroup runs each listener in its own goroutine. If one fails, gctx is cancelled, which shuts the
	// others down too; Wait returns the first error.
	g, gctx := errgroup.WithContext(ctx)
	for _, l := range ls {
		ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", l.addr)
		if err != nil {
			log.Error("listen failed", slog.String("listener", l.name), slog.String("addr", l.addr),
				slog.Any("error", err))
			stop()
			_ = g.Wait()
			return 1
		}
		log.Info("listening", slog.String("listener", l.name), slog.String("addr", ln.Addr().String()),
			slog.String("app_env", string(cfg.AppEnv)), slog.String("mode", mode))
		if listening != nil {
			listening(l.name, ln.Addr().String())
		}
		srv := httpx.NewServer(l.handler, log)
		g.Go(func() error { return httpx.Serve(gctx, srv, ln, ShutdownGrace) })
	}

	if runWorkers {
		for _, w := range rt.Workers {
			g.Go(func() error { return w(gctx) })
		}
	}

	if err := g.Wait(); err != nil {
		log.Error("server stopped with an error", slog.Any("error", err))
		return 1
	}
	log.Info("shut down cleanly")
	return 0
}

// Healthcheck asks the service on this machine whether it is ready. internalAddr is the service's internal
// listen address (":8090"); only the port is used, and the request always goes to 127.0.0.1.
func Healthcheck(ctx context.Context, internalAddr string) error {
	_, port, err := net.SplitHostPort(internalAddr)
	if err != nil {
		return fmt.Errorf("bad internal address %q: %w", internalAddr, err)
	}
	ctx, cancel := context.WithTimeout(ctx, HealthcheckTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+net.JoinHostPort("127.0.0.1", port)+"/readyz", nil)
	if err != nil {
		return err
	}
	client := &http.Client{
		// Never follow a redirect: the answer must come from this service's own /readyz.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<10))
	if resp.StatusCode != http.StatusOK {
		return errors.New("not ready: /readyz returned " + resp.Status)
	}
	return nil
}
