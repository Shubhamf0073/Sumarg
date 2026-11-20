package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// Server limits. Every listener uses them: without timeouts one slow client can hold a connection (and its
// goroutine) forever (threat T23).
const (
	ReadHeaderTimeout = 5 * time.Second
	ReadTimeout       = 15 * time.Second
	WriteTimeout      = 30 * time.Second
	IdleTimeout       = 60 * time.Second
	MaxHeaderBytes    = 64 << 10 // 64 KiB
)

// NewServer returns an http.Server with the limits above. net/http's own errors (bad TLS, malformed requests) go
// to log at warn level.
func NewServer(h http.Handler, log *slog.Logger) *http.Server {
	return &http.Server{
		Handler:           h,
		ReadHeaderTimeout: ReadHeaderTimeout,
		ReadTimeout:       ReadTimeout,
		WriteTimeout:      WriteTimeout,
		IdleTimeout:       IdleTimeout,
		MaxHeaderBytes:    MaxHeaderBytes,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelWarn),
	}
}

// Serve runs srv on ln until ctx is cancelled, then shuts down gracefully: it stops accepting connections and
// waits up to grace for requests in progress to finish. It returns nil after a clean shutdown.
func Serve(ctx context.Context, srv *http.Server, ln net.Listener, grace time.Duration) error {
	// Requests' contexts derive from base, which is cancelled if the grace period runs out, so handlers still
	// running then stop their database work too (Shutdown alone doesn't cancel them).
	base, cancelBase := context.WithCancel(context.WithoutCancel(ctx))
	defer cancelBase()
	srv.BaseContext = func(net.Listener) context.Context { return base }

	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()

	select {
	case err := <-errc:
		return err // the listener failed before any shutdown was asked for
	case <-ctx.Done():
	}

	// ctx is already cancelled, so the shutdown needs a fresh one with its own deadline.
	sctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), grace)
	defer cancel()
	if err := srv.Shutdown(sctx); err != nil {
		cancelBase()
		_ = srv.Close() // grace ran out: drop the remaining connections
		return err
	}
	if err := <-errc; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// NewRouter returns a chi router with the standard middleware and JSON 404 / 405 responses.
func NewRouter(log *slog.Logger) chi.Router {
	r := chi.NewRouter()
	r.Use(Middleware(log)...)
	r.NotFound(func(w http.ResponseWriter, r *http.Request) { WriteError(w, r, ErrNotFound) })
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) { WriteError(w, r, ErrMethodNotAllowed) })
	return r
}

// NewInternalRouter is the router for a service's internal listener. For now it serves only the health endpoints;
// every other path, and every other method on those, is 404. The internal port is never published (compose has
// no `ports:` for it); Docker's healthcheck reaches it on 127.0.0.1 inside the container.
func NewInternalRouter(log *slog.Logger, checks ...Check) chi.Router {
	r := NewRouter(log)
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) { WriteError(w, r, ErrNotFound) })
	r.Get("/healthz", Healthz)
	r.Get("/readyz", Readyz(checks...))
	return r
}
