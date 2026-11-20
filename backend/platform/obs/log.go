// Package obs builds the structured logger every binary uses. Every line is JSON (or text locally) on stdout and
// carries `service`; lines logged with a request's context also carry that request's fields (request_id now,
// tenant_id / user_id once tenancy lands), without each call site adding them.
//
// Go note: slog separates the Logger (what call sites use) from the Handler (what turns a record into bytes).
// A Handler is a small interface, so this package wraps the standard JSON/text handler in two layers: one that adds
// fields from the context, and one that redacts secret-looking fields (redact.go).
package obs

import (
	"context"
	"io"
	"log/slog"

	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/config"
)

// NewLogger returns the logger for one binary, configured from LOG_LEVEL / LOG_FORMAT.
func NewLogger(w io.Writer, cfg config.Base) *slog.Logger {
	return New(w, cfg.ServiceName, cfg.LogLevel, cfg.LogFormat)
}

// New is NewLogger without a config.Base, for tests and for code that runs before config is loaded.
func New(w io.Writer, service string, level slog.Level, format string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: level}
	var h slog.Handler
	if format == "text" {
		h = slog.NewTextHandler(w, opts)
	} else {
		h = slog.NewJSONHandler(w, opts)
	}
	// Redaction is the inner layer so it also sees the attrs contextHandler adds.
	return slog.New(contextHandler{redactHandler{h}}).With(slog.String("service", service))
}

type ctxKey struct{}

// With returns a copy of ctx whose log lines also carry attrs. Middleware uses it for request_id (and later
// tenancy for tenant_id / user_id); a handler that logs with this context gets them for free:
//
//	log.InfoContext(ctx, "booking created", "booking_id", id)
func With(ctx context.Context, attrs ...slog.Attr) context.Context {
	prev, _ := ctx.Value(ctxKey{}).([]slog.Attr)
	// Copy instead of appending in place: two requests derived from one parent context must not share a slice.
	next := make([]slog.Attr, 0, len(prev)+len(attrs))
	next = append(append(next, prev...), attrs...)
	return context.WithValue(ctx, ctxKey{}, next)
}

// contextHandler adds the attrs stored by With to every record logged with that context. Only the *Context
// logging methods (InfoContext, ErrorContext, ...) pass a context; plain Info passes context.Background().
type contextHandler struct{ slog.Handler }

func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if attrs, ok := ctx.Value(ctxKey{}).([]slog.Attr); ok {
		r = r.Clone()
		r.AddAttrs(attrs...)
	}
	return h.Handler.Handle(ctx, r)
}

func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{h.Handler.WithAttrs(attrs)}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{h.Handler.WithGroup(name)}
}
