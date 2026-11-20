package db

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/httpx"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/obs"
)

// MaxBufferedResponse caps what a transactional handler may write. The response is held back until the commit
// succeeds, so it must fit in memory; API responses are small JSON documents (lists are paginated).
const MaxBufferedResponse = 4 << 20

// ErrResponseTooLarge means a handler wrote more than MaxBufferedResponse. The transaction is rolled back.
var ErrResponseTooLarge = errors.New("db: buffered response too large")

type txKey struct{}

// TxFrom returns the request's tenant transaction, or false outside RequestTx. Services call it and build queries
// with tenantq.New(tx).
func TxFrom(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(pgx.Tx)
	return tx, ok
}

// Resolver names the tenant of a request from its session (invariant 1: never from the URL, a header the client
// chose, or the MQTT topic). readOnly is true for sessions that may not write (Previa support view, suspended
// tenant). An error is sent to the client with httpx.WriteError, and no transaction is opened.
//
// A resolver that needs the database (the session lookup under RLS, docs/02 §5) runs its own short InTenantReadOnly
// and finishes it before returning, so it is over before the request transaction begins: one request never holds
// two connections at once. The tenant it uses is the one in the session cookie; never a body, query or path value.
type Resolver func(r *http.Request) (Resolved, error)

// Resolved is what a Resolver found. Ctx, if set, replaces the request's context for the handler: it must derive
// from r.Context() (RequestTx refuses one that doesn't) and is where the resolver leaves what it learned (e.g. the session's principal), so the handler
// needn't look the session up a second time.
type Resolved struct {
	TenantID int64
	ReadOnly bool
	Ctx      context.Context
}

// resolveMarkKey holds a per-request marker that RequestTx looks for in Resolved.Ctx; a Ctx without it is a 500
// and no transaction is opened.
type resolveMarkKey struct{}

var errCtxNotDerived = errors.New("db: Resolved.Ctx does not derive from the request's context")

// RequestTx runs each request in one tenant transaction on pool (docs/02 §5).
//
//   - The tenant comes from resolve. GET, HEAD and OPTIONS always use InTenantReadOnly, so Postgres rejects any
//     write they attempt.
//   - The handler's response is buffered. A 2xx/3xx status commits, and the response is sent only after the commit
//     succeeds, so a failed commit becomes a clean 500 and the client never sees success for data that was not
//     saved. A 4xx/5xx status or a panic rolls back, and the response (an error body) is sent as written.
//   - The handler finds the transaction with TxFrom, and must use only that transaction while it runs: no goroutine
//     may keep it after the handler returns, and the pool must not be used alongside it (a small pool can deadlock).
//
// Mount it only on routes that need a tenant: health checks, login (which resolves its tenant itself) and
// /api/admin do not use it. Keep CORS preflight (OPTIONS carries no cookie) outside it, and keep streaming or SSE
// routes off it: the buffered writer has no Flush or Hijack.
func RequestTx(pool *pgxpool.Pool, resolve Resolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// The marker lets us check that a resolver's Ctx derives from this request's context (so it keeps its
			// cancellation, deadline and request id) rather than from context.Background or another request.
			mark := new(byte)
			r = r.WithContext(context.WithValue(r.Context(), resolveMarkKey{}, mark))
			res, err := resolve(r)
			if err != nil {
				httpx.WriteError(w, r, err)
				return
			}
			if res.Ctx != nil {
				if m, _ := res.Ctx.Value(resolveMarkKey{}).(*byte); m != mark {
					httpx.WriteError(w, r, errCtxNotDerived)
					return
				}
				r = r.WithContext(res.Ctx)
			}
			tenantID, readOnly := res.TenantID, res.ReadOnly
			if safeMethod(r.Method) {
				readOnly = true
			}
			run := InTenant
			if readOnly {
				run = InTenantReadOnly
			}

			buf := &bufferedResponse{header: http.Header{}}
			err = run(r.Context(), pool, tenantID, func(tx pgx.Tx) error {
				ctx := obs.With(context.WithValue(r.Context(), txKey{}, tx), slog.Int64("tenant_id", tenantID))
				next.ServeHTTP(buf, r.WithContext(ctx))
				if buf.tooLarge {
					return ErrResponseTooLarge
				}
				if buf.status() >= 400 {
					return errRollback // handler already produced its error response
				}
				return nil
			})
			switch {
			case err == nil:
				buf.flushTo(w)
			case errors.Is(err, errRollback):
				buf.flushTo(w)
			default: // begin, set tenant, commit, oversize response
				httpx.WriteError(w, r, safeDBError(err))
			}
		})
	}
}

// safeDBError keeps PgError.Detail (it can quote row values) out of the log: only the SQLSTATE and constraint name
// survive. A serialization failure or deadlock is a retryable 503, not a bug.
func safeDBError(err error) error {
	var pg *pgconn.PgError
	if !errors.As(err, &pg) {
		return err
	}
	cause := fmt.Errorf("db: sqlstate %s constraint %q", pg.Code, pg.ConstraintName)
	if pg.Code == "40001" || pg.Code == "40P01" {
		return httpx.NewError(http.StatusServiceUnavailable, "busy", "The server is busy. Please try again.").Wrap(cause)
	}
	return cause
}

// errRollback aborts the transaction because the handler answered with an error status. It never reaches the client.
var errRollback = errors.New("db: handler returned an error status")

func safeMethod(m string) bool {
	return m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions
}

// bufferedResponse holds a handler's response until the transaction's fate is known.
type bufferedResponse struct {
	header   http.Header
	code     int
	body     bytes.Buffer
	tooLarge bool
}

func (b *bufferedResponse) Header() http.Header { return b.header }

func (b *bufferedResponse) WriteHeader(code int) {
	if b.tooLarge {
		return
	}
	if b.code == 0 && code >= 200 { // 1xx are informational; the final status comes later
		b.code = code
	}
}

func (b *bufferedResponse) Write(p []byte) (int, error) {
	if b.tooLarge {
		return 0, ErrResponseTooLarge
	}
	if b.code == 0 {
		b.code = http.StatusOK
	}
	if b.body.Len()+len(p) > MaxBufferedResponse {
		b.tooLarge = true
		return 0, ErrResponseTooLarge
	}
	return b.body.Write(p)
}

func (b *bufferedResponse) status() int {
	if b.code == 0 {
		return http.StatusOK // net/http sends 200 when the handler wrote nothing
	}
	return b.code
}

func (b *bufferedResponse) flushTo(w http.ResponseWriter) {
	dst := w.Header()
	for k, v := range b.header {
		dst[k] = v
	}
	w.WriteHeader(b.status())
	_, _ = w.Write(b.body.Bytes())
}
