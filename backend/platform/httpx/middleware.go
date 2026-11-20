package httpx

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/ids"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/obs"
)

// RequestIDHeader is the response header that carries the request ID (and, later, the outgoing header on
// service-to-service calls).
const RequestIDHeader = "X-Request-ID"

type requestIDKey struct{}
type loggerKey struct{}

// RequestID returns the current request's ID, or "" outside a request.
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// Logger returns the logger the middleware stored in ctx, or slog.Default() outside a request.
func Logger(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// Middleware returns the standard stack, outermost first: request ID + logger in the context, access log,
// panic recovery. Mount it with router.Use on a chi router so the access log can see the matched route pattern.
//
// Go note: a middleware is a function that takes the next handler and returns a handler wrapping it. Each layer
// runs its code before and/or after calling next.ServeHTTP.
func Middleware(log *slog.Logger) []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{withRequest(log), accessLog, recoverPanic}
}

// RequestTimeout is the deadline on every request's context, a little under the server's WriteTimeout. The server
// timeout only stops the response; this also cancels the database and other work the handler is still doing.
const RequestTimeout = WriteTimeout - 5*time.Second

// withRequest gives every request a fresh UUIDv7 ID. An inbound X-Request-ID is ignored: the edge is public, and a
// client-chosen ID could collide with, or be made to look like, another request in the logs. Nginx logs its own
// $request_id; the API logs both (docs/06 §7).
func withRequest(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := ids.New().String()
			w.Header().Set(RequestIDHeader, id)
			ctx, cancel := context.WithTimeout(r.Context(), RequestTimeout)
			defer cancel()
			ctx = context.WithValue(ctx, requestIDKey{}, id)
			ctx = context.WithValue(ctx, loggerKey{}, log)
			ctx = obs.With(ctx, slog.String("request_id", id))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// accessLog writes one line per request. It logs the route pattern ("/api/bookings/{id}"), never the raw path or
// query string: public tracking links (/b/<token>) and query parameters can carry tokens and personal data.
func accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now() //nolint:forbidigo // request duration for the access log, not business time
		sw := &statusWriter{ResponseWriter: w}
		// Deferred so the line is written even when recoverPanic aborts the response by panicking again.
		defer func() {
			route := "unmatched"
			if rc := chi.RouteContext(r.Context()); rc != nil && rc.RoutePattern() != "" {
				route = rc.RoutePattern()
			}
			level := slog.LevelInfo
			switch {
			case sw.Status() >= 500:
				level = slog.LevelError
			case healthRoutes[route]:
				level = slog.LevelDebug // Docker polls these every 30 s on every service
			}
			Logger(r.Context()).LogAttrs(r.Context(), level, "request",
				slog.String("method", logMethod(r.Method)),
				slog.String("route", route),
				slog.Int("status", sw.Status()),
				slog.Int64("bytes", sw.bytes),
				slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			)
		}()
		next.ServeHTTP(sw, r)
	})
}

var healthRoutes = map[string]bool{"/healthz": true, "/readyz": true, "/api/healthz": true}

var knownMethods = map[string]bool{http.MethodGet: true, http.MethodHead: true, http.MethodPost: true,
	http.MethodPut: true, http.MethodPatch: true, http.MethodDelete: true, http.MethodOptions: true}

// logMethod keeps the client-chosen method out of the log unless it is a standard one.
func logMethod(m string) string {
	if knownMethods[m] {
		return m
	}
	return "OTHER"
}

// recoverPanic turns a panic in a handler into a generic 500 and logs the stack. The stack never reaches the client.
// If the handler had already started the response, a 500 can't be sent any more; the connection is aborted instead,
// so the client sees an error rather than a truncated body that looks complete.
func recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw, ok := w.(*statusWriter)
		if !ok {
			sw = &statusWriter{ResponseWriter: w}
		}
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			if rec == http.ErrAbortHandler { //nolint:errorlint // recover() returns the sentinel itself, never wrapped
				panic(rec) // the server's own signal to abort the response; let net/http handle it
			}
			Logger(r.Context()).ErrorContext(r.Context(), "panic in handler",
				slog.String("panic", fmt.Sprint(rec)), slog.String("stack", string(debug.Stack())))
			if sw.wroteHeader {
				sw.status = http.StatusInternalServerError // for the access log; the client gets a reset connection
				panic(http.ErrAbortHandler)
			}
			WriteError(sw, r, ErrInternal)
		}()
		next.ServeHTTP(sw, r)
	})
}

// statusWriter records the status and size of a response.
type statusWriter struct {
	http.ResponseWriter
	status      int
	bytes       int64
	wroteHeader bool
}

func (w *statusWriter) WriteHeader(code int) {
	// 1xx (e.g. 103 Early Hints) are informational; the final status comes later.
	if code >= 200 && !w.wroteHeader {
		w.status, w.wroteHeader = code, true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += int64(n)
	return n, err
}

// Status is the status sent, or 200 if the handler wrote nothing (net/http sends 200 then).
func (w *statusWriter) Status() int {
	if !w.wroteHeader {
		return http.StatusOK
	}
	return w.status
}

// Flush sends buffered data to the client, for handlers that check w.(http.Flusher).
func (w *statusWriter) Flush() {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	_ = http.NewResponseController(w.ResponseWriter).Flush()
}

// Unwrap lets http.ResponseController reach the real writer (for Flush, deadlines).
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
