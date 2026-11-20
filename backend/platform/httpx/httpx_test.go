package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/ids"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/obs"
)

func testLogger() (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	return obs.New(&buf, "test", slog.LevelDebug, "json"), &buf
}

func do(t *testing.T, h http.Handler, method, target, contentType, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func errBody(t *testing.T, rec *httptest.ResponseRecorder) errorBody {
	t.Helper()
	var b errorBody
	dec := json.NewDecoder(rec.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&b); err != nil {
		t.Fatalf("error body is not the standard shape: %v", err)
	}
	return b
}

func TestHealth(t *testing.T) {
	log, logs := testLogger()
	failing := Check{Name: "db", Fn: func(context.Context) error { return errors.New("dial 10.0.0.5: refused") }}

	ok := NewInternalRouter(log)
	if rec := do(t, ok, "GET", "/healthz", "", ""); rec.Code != 200 || rec.Body.String() != "ok\n" {
		t.Fatalf("healthz: %d %q", rec.Code, rec.Body)
	}
	if rec := do(t, ok, "GET", "/readyz", "", ""); rec.Code != 200 {
		t.Fatalf("readyz with no checks: %d", rec.Code)
	}

	bad := NewInternalRouter(log, failing)
	if rec := do(t, bad, "GET", "/healthz", "", ""); rec.Code != 200 {
		t.Fatalf("healthz must not depend on readiness checks: %d", rec.Code)
	}
	rec := do(t, bad, "GET", "/readyz", "", "")
	if rec.Code != 503 || strings.Contains(rec.Body.String(), "10.0.0.5") {
		t.Fatalf("readyz: %d %q", rec.Code, rec.Body)
	}
	if !strings.Contains(logs.String(), "10.0.0.5") || !strings.Contains(logs.String(), `"check":"db"`) {
		t.Fatalf("failure detail not logged: %s", logs)
	}
}

func TestReadyzCheckTimesOut(t *testing.T) {
	log, _ := testLogger()
	hung := Check{Name: "hung", Fn: func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }}
	start := time.Now()
	if rec := do(t, NewInternalRouter(log, hung), "GET", "/readyz", "", ""); rec.Code != 503 {
		t.Fatalf("got %d", rec.Code)
	}
	if d := time.Since(start); d > CheckTimeout+time.Second {
		t.Fatalf("readyz took %v", d)
	}
}

func TestInternalRouterExposesOnlyHealth(t *testing.T) {
	log, _ := testLogger()
	h := NewInternalRouter(log)
	for _, c := range []struct{ method, path string }{
		{"GET", "/internal/v1/usage"}, {"POST", "/readyz"}, {"DELETE", "/healthz"}, {"GET", "/"}, {"GET", "/api/healthz"},
	} {
		rec := do(t, h, c.method, c.path, "", "")
		if rec.Code != 404 || errBody(t, rec).Code != "not_found" {
			t.Errorf("%s %s = %d, want 404 not_found", c.method, c.path, rec.Code)
		}
	}
}

type payload struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func decodeHandler(max int64) http.Handler {
	log, _ := testLogger()
	r := NewRouter(log)
	r.Post("/x", func(w http.ResponseWriter, r *http.Request) {
		var p payload
		if err := DecodeJSON(w, r, &p, max); err != nil {
			WriteError(w, r, err)
			return
		}
		WriteJSON(w, r, http.StatusOK, p)
	})
	return r
}

func TestDecodeJSON(t *testing.T) {
	const js = "application/json"
	cases := []struct {
		name, contentType, body string
		status                  int
		code                    string
	}{
		{"ok", js, `{"name":"a","count":2}`, 200, ""},
		{"ok with charset", "application/json; charset=utf-8", `{"name":"a"}`, 200, ""},
		{"unknown field", js, `{"name":"a","tenant_id":"x"}`, 400, "bad_request"},
		{"wrong type", js, `{"count":"2"}`, 400, "bad_request"},
		{"malformed", js, `{"name":`, 400, "bad_request"},
		{"empty", js, ``, 400, "bad_request"},
		{"two objects", js, `{"name":"a"}{"name":"b"}`, 400, "bad_request"},
		{"trailing garbage", js, `{"name":"a"} x`, 400, "bad_request"},
		{"array", js, `[{"name":"a"}]`, 400, "bad_request"},
		{"null", js, `null`, 400, "bad_request"},
		{"number", js, ` 1`, 400, "bad_request"},
		{"case variant", js, `{"NAME":"a"}`, 400, "bad_request"},
		{"duplicate", js, `{"count":1,"count":2}`, 400, "bad_request"},
		{"invalid utf-8", js, "{\"name\":\"\xff\"}", 400, "bad_request"},
		{"leading space ok", js, ` {"name":"a"}`, 200, ""},
		{"too large", js, `{"name":"` + strings.Repeat("a", 200) + `"}`, 413, "body_too_large"},
		{"no content type", "", `{"name":"a"}`, 415, "unsupported_media_type"},
		{"form", "application/x-www-form-urlencoded", `name=a`, 415, "unsupported_media_type"},
		{"json-ish", "application/jsonp", `{"name":"a"}`, 415, "unsupported_media_type"},
	}
	h := decodeHandler(100)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := do(t, h, "POST", "/x", c.contentType, c.body)
			if rec.Code != c.status {
				t.Fatalf("status %d, want %d: %s", rec.Code, c.status, rec.Body)
			}
			if c.code == "" {
				return
			}
			b := errBody(t, rec)
			if b.Code != c.code || b.RequestID == "" {
				t.Fatalf("body %+v", b)
			}
			// The decoder's own error text (which quotes the input) is not sent.
			if strings.Contains(rec.Body.String(), "json:") || strings.Contains(rec.Body.String(), "tenant_id") {
				t.Fatalf("internal error text leaked: %s", rec.Body)
			}
		})
	}
}

func TestWriteErrorHidesInternalErrors(t *testing.T) {
	log, logs := testLogger()
	r := NewRouter(log)
	r.Get("/plain", func(w http.ResponseWriter, r *http.Request) {
		WriteError(w, r, errors.New(`pq: relation "app.users" does not exist`))
	})
	r.Get("/locked", func(w http.ResponseWriter, r *http.Request) {
		WriteError(w, r, NewError(http.StatusLocked, "tenant_read_only", "This account is read-only.").
			Wrap(errors.New("status=on_hold")))
	})

	rec := do(t, r, "GET", "/plain", "", "")
	if b := errBody(t, rec); rec.Code != 500 || b.Code != "internal" || strings.Contains(rec.Body.String(), "app.users") {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
	if !strings.Contains(logs.String(), "app.users") {
		t.Fatalf("5xx cause not logged: %s", logs)
	}

	rec = do(t, r, "GET", "/locked", "", "")
	if b := errBody(t, rec); rec.Code != 423 || b.Code != "tenant_read_only" || strings.Contains(rec.Body.String(), "on_hold") {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("API responses must not be cached")
	}
}

func TestPanicBecomesGeneric500(t *testing.T) {
	log, logs := testLogger()
	r := NewRouter(log)
	r.Get("/boom", func(http.ResponseWriter, *http.Request) { panic("secret internal state") })

	rec := do(t, r, "GET", "/boom", "", "")
	b := errBody(t, rec)
	if rec.Code != 500 || b.Code != "internal" || b.RequestID != rec.Header().Get(RequestIDHeader) {
		t.Fatalf("got %d %+v", rec.Code, b)
	}
	if strings.Contains(rec.Body.String(), "secret internal state") || strings.Contains(rec.Body.String(), "goroutine") {
		t.Fatalf("panic detail leaked to the client: %s", rec.Body)
	}
	out := logs.String()
	if !strings.Contains(out, "secret internal state") || !strings.Contains(out, "httpx_test.go") {
		t.Fatalf("panic and stack not logged: %s", out)
	}
	if !strings.Contains(out, `"status":500`) {
		t.Fatalf("access log should record the 500: %s", out)
	}
}

func TestAbortHandlerPanicIsNotSwallowed(t *testing.T) {
	log, _ := testLogger()
	r := NewRouter(log)
	r.Get("/abort", func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) })
	defer func() {
		if rec := recover(); rec != http.ErrAbortHandler {
			t.Fatalf("recovered %v, want http.ErrAbortHandler re-panicked", rec)
		}
	}()
	do(t, r, "GET", "/abort", "", "")
}

func TestRequestIDIsServerGenerated(t *testing.T) {
	log, logs := testLogger()
	r := NewRouter(log)
	var seen string
	r.Get("/x", func(w http.ResponseWriter, r *http.Request) {
		seen = RequestID(r.Context())
		Logger(r.Context()).InfoContext(r.Context(), "handler")
	})

	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set(RequestIDHeader, "forged-id")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	got := rec.Header().Get(RequestIDHeader)
	if _, err := ids.Parse(got); err != nil || got != seen {
		t.Fatalf("response id %q, handler saw %q", got, seen)
	}
	if strings.Contains(logs.String(), "forged-id") {
		t.Fatalf("client-supplied request id was used: %s", logs)
	}
	// Both the handler's line and the access line carry the id.
	if n := strings.Count(logs.String(), `"request_id":"`+got+`"`); n != 2 {
		t.Fatalf("request_id on %d lines, want 2: %s", n, logs)
	}
}

func TestAccessLogHasRoutePatternNotPathOrQuery(t *testing.T) {
	log, logs := testLogger()
	r := NewRouter(log)
	r.Get("/b/{token}", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })

	do(t, r, "GET", "/b/tok_abc123?otp=987654&phone=9999999999", "", "")
	do(t, r, "GET", "/nope/tok_def456?x=1", "", "")
	out := logs.String()
	for _, leak := range []string{"tok_abc123", "987654", "9999999999", "tok_def456", "?"} {
		if strings.Contains(out, leak) {
			t.Errorf("access log contains %q:\n%s", leak, out)
		}
	}
	for _, want := range []string{`"route":"/b/{token}"`, `"status":204`, `"route":"unmatched"`, `"status":404`} {
		if !strings.Contains(out, want) {
			t.Errorf("access log missing %s:\n%s", want, out)
		}
	}
}

func TestNewServerSetsLimits(t *testing.T) {
	log, _ := testLogger()
	s := NewServer(http.NotFoundHandler(), log)
	if s.ReadHeaderTimeout == 0 || s.ReadTimeout == 0 || s.WriteTimeout == 0 || s.IdleTimeout == 0 ||
		s.MaxHeaderBytes != MaxHeaderBytes {
		t.Fatalf("limits not set: %+v", s)
	}
}

// A request in progress when shutdown starts still completes, and Serve then returns nil.
func TestServeShutsDownGracefully(t *testing.T) {
	log, _ := testLogger()
	started, release := make(chan struct{}), make(chan struct{})
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		_, _ = io.WriteString(w, "done")
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	served := make(chan error, 1)
	go func() { served <- Serve(ctx, NewServer(h, log), ln, 5*time.Second) }()

	type result struct {
		body string
		err  error
	}
	got := make(chan result, 1)
	go func() {
		resp, err := http.Get("http://" + ln.Addr().String() + "/")
		if err != nil {
			got <- result{err: err}
			return
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		got <- result{string(b), err}
	}()

	<-started
	cancel() // like SIGTERM, while the request is still running
	// Shutdown closes the listener first; wait (bounded) until new connections are refused.
	closed := false
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		c, err := net.DialTimeout("tcp", ln.Addr().String(), 100*time.Millisecond)
		if err != nil {
			closed = true
			break
		}
		_ = c.Close()
		runtime.Gosched()
	}
	if !closed {
		t.Error("listener still accepting connections after shutdown began")
	}
	close(release)

	if r := <-got; r.err != nil || r.body != "done" {
		t.Fatalf("in-flight request: %q %v", r.body, r.err)
	}
	if err := <-served; err != nil {
		t.Fatalf("Serve returned %v", err)
	}
}

func TestPanicAfterWriteAbortsConnection(t *testing.T) {
	log, logs := testLogger()
	r := NewRouter(log)
	r.Get("/half", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `[{"id":1},`)
		w.(http.Flusher).Flush()
		panic("mid-list")
	})
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/half")
	if err == nil {
		_, err = io.ReadAll(resp.Body)
		resp.Body.Close()
	}
	if err == nil {
		t.Fatal("client read a truncated body without an error")
	}
	srv.Close() // waits for the handler (and its deferred access line) to finish before the log is read
	out := logs.String()
	if !strings.Contains(out, "mid-list") || !strings.Contains(out, `"status":500`) || !strings.Contains(out, `"level":"ERROR","msg":"request"`) {
		t.Fatalf("panic or 500 access line missing:\n%s", out)
	}
}

func TestWriteErrorWithBrokenErrors(t *testing.T) {
	log, _ := testLogger()
	r := NewRouter(log)
	r.Get("/nil", func(w http.ResponseWriter, r *http.Request) {
		var e *Error
		WriteError(w, r, e)
	})
	r.Get("/zero", func(w http.ResponseWriter, r *http.Request) { WriteError(w, r, &Error{Code: "x"}) })
	for _, path := range []string{"/nil", "/zero"} {
		rec := do(t, r, "GET", path, "", "")
		if b := errBody(t, rec); rec.Code != 500 || b.Code != "internal" {
			t.Errorf("%s: %d %+v", path, rec.Code, b)
		}
	}
}

func TestAccessLogDetails(t *testing.T) {
	log, logs := testLogger()
	r := NewRouter(log)
	r.Get("/hints", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusEarlyHints)
		w.WriteHeader(http.StatusAccepted)
	})
	r.Get("/healthz", Healthz)
	do(t, r, "GET", "/hints", "", "")
	do(t, r, "BREW", "/hints", "", "")
	do(t, r, "GET", "/healthz", "", "")
	out := logs.String()
	for _, want := range []string{`"status":202`, `"method":"OTHER"`, `"level":"DEBUG","msg":"request","service":"test","method":"GET","route":"/healthz"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in\n%s", want, out)
		}
	}
	if strings.Contains(out, "BREW") || strings.Contains(out, `"status":103`) {
		t.Errorf("unexpected access log content:\n%s", out)
	}
}

func TestRequestContextHasDeadline(t *testing.T) {
	log, _ := testLogger()
	r := NewRouter(log)
	var left time.Duration
	r.Get("/x", func(w http.ResponseWriter, r *http.Request) {
		d, ok := r.Context().Deadline()
		if ok {
			left = time.Until(d)
		}
	})
	do(t, r, "GET", "/x", "", "")
	if left <= 0 || left > RequestTimeout {
		t.Fatalf("request deadline in %v, want (0, %v]", left, RequestTimeout)
	}
}

// When the grace period runs out, handlers still running see their context cancelled.
func TestServeCancelsHandlersAfterGrace(t *testing.T) {
	log, _ := testLogger()
	started, cancelled := make(chan struct{}), make(chan struct{})
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		close(cancelled)
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	served := make(chan error, 1)
	go func() { served <- Serve(ctx, NewServer(h, log), ln, 100*time.Millisecond) }()
	go func() {
		if resp, err := http.Get("http://" + ln.Addr().String() + "/"); err == nil {
			resp.Body.Close()
		}
	}()
	<-started
	cancel()
	select {
	case <-cancelled:
	case <-time.After(5 * time.Second):
		t.Fatal("handler context not cancelled after the grace period")
	}
	if err := <-served; err == nil {
		t.Fatal("Serve should report that the grace period ran out")
	}
}
