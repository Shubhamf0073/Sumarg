package httpx

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// CheckTimeout bounds each readiness check, so a hung dependency makes /readyz fail instead of hang.
const CheckTimeout = 2 * time.Second

// Check is one readiness dependency (the database ping, M0.3).
type Check struct {
	Name string
	Fn   func(ctx context.Context) error
}

// Healthz is liveness: the process is up and serving HTTP. It checks nothing else, so a database outage doesn't
// make Docker restart every container (docs/06 §4).
func Healthz(w http.ResponseWriter, r *http.Request) {
	writeText(w, http.StatusOK, "ok")
}

// Readyz is readiness: every check passes. `<binary> healthcheck` calls it, so deploy.sh rolls back a release
// that can't reach its dependencies. Failure details go to the log only; the body is just "not ready".
func Readyz(checks ...Check) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, c := range checks {
			ctx, cancel := context.WithTimeout(r.Context(), CheckTimeout)
			err := c.Fn(ctx)
			cancel()
			if err != nil {
				Logger(r.Context()).WarnContext(r.Context(), "readiness check failed",
					slog.String("check", c.Name), slog.Any("error", err))
				writeText(w, http.StatusServiceUnavailable, "not ready")
				return
			}
		}
		writeText(w, http.StatusOK, "ok")
	}
}

func writeText(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body + "\n"))
}
