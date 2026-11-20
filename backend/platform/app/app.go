// Package app is the start-up and shutdown sequence every service binary shares, so cmd/<svc>/main.go only
// declares what is different (name, ports, routes):
//
//	<binary> serve [--mode=api|worker|all]   load config, run the public and internal listeners, stop on SIGTERM
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

// ShutdownGrace is how long requests in progress get after SIGTERM. Docker sends SIGKILL 10 s after SIGTERM by
// default (compose stop_grace_period), so this must stay below that.
const ShutdownGrace = 8 * time.Second

// HealthcheckTimeout bounds `healthcheck`. Docker's own check timeout is longer, so a hung service reports
// unhealthy from here rather than being killed by Docker.
const HealthcheckTimeout = 3 * time.Second

// Service describes one binary.
type Service struct {
	Name         string // must equal SVC_NAME in .env ("schoolfleet")
	PublicAddr   string // ":8080"; "" = no public listener (docs)
	InternalAddr string // ":8090"; always present, it serves the health endpoints
	// Public mounts the public routes. Only called when PublicAddr is set.
	Public func(r chi.Router)
	// Modes, if set, are the values `serve --mode` accepts; the first is the default.
	Modes []string
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
func Main(s Service) {
	os.Exit(run(context.Background(), s, os.Args[1:], os.Stdout, os.Stderr, config.Load[config.Base], nil))
}

// run is Main without process globals, for tests. listening, if non-nil, receives each listener's name and bound
// address once it is accepting connections.
func run(ctx context.Context, s Service, args []string, stdout, stderr io.Writer,
	load func() (config.Base, error), listening func(name, addr string)) int {
	if len(args) == 0 {
		usage(stderr, s)
		return 2
	}
	switch args[0] {
	case "serve":
		return serve(ctx, s, args[1:], stdout, stderr, load, listening)
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

func usage(w io.Writer, s Service) {
	mode := ""
	if len(s.Modes) > 0 {
		mode = " [--mode=" + strings.Join(s.Modes, "|") + "]"
	}
	fmt.Fprintf(w, "usage: %s serve%s | healthcheck\n", s.Name, mode)
}

func serve(ctx context.Context, s Service, args []string, stdout, stderr io.Writer,
	load func() (config.Base, error), listening func(name, addr string)) int {
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

	cfg, err := load()
	if err != nil {
		// Config errors name the variable and never print its value (platform/config).
		fmt.Fprintf(stderr, "%s: %v\n", s.Name, err)
		return 1
	}
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

	type listener struct {
		name, addr string
		handler    http.Handler
	}
	// TODO(M0.3): add the database ping to the readiness checks.
	ls := []listener{{"internal", s.InternalAddr, httpx.NewInternalRouter(log)}}
	runAPI := mode == "" || mode == ModeAPI || mode == ModeAll
	if runAPI && s.PublicAddr != "" {
		r := httpx.NewRouter(log)
		if s.Public != nil {
			s.Public(r)
		}
		ls = append(ls, listener{"public", s.PublicAddr, r})
	}
	if mode == ModeWorker || mode == ModeAll {
		log.Info("worker mode: no job workers are registered yet (notification sender arrives in M4)")
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
