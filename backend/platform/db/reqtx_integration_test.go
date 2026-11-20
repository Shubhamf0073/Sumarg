//go:build integration

package db_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/config"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/db"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/db/platformdb"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/httpx"
)

// resolveHeader is a test resolver: the tenant comes from a header. Real routes resolve it from the session.
func resolveHeader(r *http.Request) (int64, bool, error) {
	id, err := strconv.ParseInt(r.Header.Get("X-Test-Tenant"), 10, 64)
	if err != nil {
		return 0, false, httpx.ErrBadRequest
	}
	return id, r.Header.Get("X-Test-Readonly") == "1", nil
}

func do(h http.Handler, method string, tenant int64, readOnly bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/api/x", nil)
	req.Header.Set("X-Test-Tenant", strconv.FormatInt(tenant, 10))
	if readOnly {
		req.Header.Set("X-Test-Readonly", "1")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func studentName(t *testing.T, f fixture) string {
	t.Helper()
	var name string
	err := db.InTenantReadOnly(t.Context(), f.app, f.t2, func(tx pgx.Tx) error {
		return tx.QueryRow(t.Context(), "SELECT name FROM students WHERE id = $1", f.c2).Scan(&name)
	})
	if err != nil {
		t.Fatal(err)
	}
	return name
}

// rename writes through the request's transaction.
func rename(w http.ResponseWriter, r *http.Request, name string, status int) {
	tx, ok := db.TxFrom(r.Context())
	if !ok {
		http.Error(w, "no tx", http.StatusInternalServerError)
		return
	}
	if _, err := tx.Exec(r.Context(), "UPDATE students SET name = $1", name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(status)
}

func TestRequestTxIsolatesTenants(t *testing.T) {
	f := setup(t)
	h := db.RequestTx(f.app, resolveHeader)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tx, _ := db.TxFrom(r.Context())
		var n int
		if err := tx.QueryRow(r.Context(), "SELECT count(*) FROM students WHERE id = $1", f.c2).Scan(&n); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(strconv.Itoa(n)))
	}))
	if got := do(h, http.MethodGet, f.t1, false); got.Code != 200 || got.Body.String() != "0" {
		t.Fatalf("tenant 1 sees tenant 2's student: %d %q", got.Code, got.Body.String())
	}
	if got := do(h, http.MethodGet, f.t2, false); got.Code != 200 || got.Body.String() != "1" {
		t.Fatalf("tenant 2 can't see its own student: %d %q", got.Code, got.Body.String())
	}
}

func TestRequestTxCommitsOnSuccessAndRollsBackOnError(t *testing.T) {
	f := setup(t)
	before := studentName(t, f)

	status := http.StatusConflict
	h := db.RequestTx(f.app, resolveHeader)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rename(w, r, "renamed-by-test", status)
	}))

	if got := do(h, http.MethodPost, f.t2, false); got.Code != http.StatusConflict {
		t.Fatalf("status %d", got.Code)
	}
	if name := studentName(t, f); name != before {
		t.Fatalf("a 409 response must roll back, name is now %q", name)
	}

	status = http.StatusOK
	if got := do(h, http.MethodPost, f.t2, false); got.Code != http.StatusOK {
		t.Fatalf("status %d", got.Code)
	}
	if name := studentName(t, f); name != "renamed-by-test" {
		t.Fatalf("a 200 response must commit, name is %q", name)
	}
	// put it back so the database is as the next run expects
	_ = db.InTenant(t.Context(), f.app, f.t2, func(tx pgx.Tx) error {
		_, err := tx.Exec(t.Context(), "UPDATE students SET name = $1 WHERE id = $2", before, f.c2)
		return err
	})
}

func TestRequestTxRollsBackOnPanic(t *testing.T) {
	f := setup(t)
	before := studentName(t, f)
	h := db.RequestTx(f.app, resolveHeader)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rename(w, r, "panic-rename", http.StatusOK)
		panic("boom")
	}))
	func() {
		defer func() { _ = recover() }() // httpx's recoverPanic does this in the real stack
		do(h, http.MethodPost, f.t2, false)
	}()
	if name := studentName(t, f); name != before {
		t.Fatalf("a panic must roll back, name is now %q", name)
	}
}

func TestRequestTxReadOnly(t *testing.T) {
	f := setup(t)
	before := studentName(t, f)
	h := db.RequestTx(f.app, resolveHeader)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rename(w, r, "should-not-happen", http.StatusOK)
	}))
	// GET is read-only whatever the session says; a read-only session is read-only on POST too.
	for _, c := range []struct {
		method   string
		readOnly bool
	}{{http.MethodGet, false}, {http.MethodPost, true}} {
		got := do(h, c.method, f.t2, c.readOnly)
		if got.Code != http.StatusInternalServerError || !strings.Contains(got.Body.String(), "25006") {
			t.Errorf("%s readOnly=%t: want a read-only violation (25006), got %d %q",
				c.method, c.readOnly, got.Code, got.Body.String())
		}
	}
	if name := studentName(t, f); name != before {
		t.Fatalf("name changed to %q", name)
	}
}

func TestRequestTxOversizeResponseRollsBack(t *testing.T) {
	f := setup(t)
	before := studentName(t, f)
	h := db.RequestTx(f.app, resolveHeader)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rename(w, r, "too-big", http.StatusOK)
		_, _ = w.Write(make([]byte, db.MaxBufferedResponse+1))
	}))
	if got := do(h, http.MethodPost, f.t2, false); got.Code != http.StatusInternalServerError {
		t.Fatalf("status %d", got.Code)
	}
	if name := studentName(t, f); name != before {
		t.Fatalf("name changed to %q", name)
	}
}

// Tenant 1's write must not touch tenant 2's rows, even though the request commits.
func TestRequestTxCannotWriteOtherTenant(t *testing.T) {
	f := setup(t)
	before := studentName(t, f)
	h := db.RequestTx(f.app, resolveHeader)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rename(w, r, "cross-tenant", http.StatusOK)
	}))
	if got := do(h, http.MethodPost, f.t1, false); got.Code != http.StatusOK {
		t.Fatalf("status %d", got.Code)
	}
	if name := studentName(t, f); name != before {
		t.Fatalf("tenant 1's request changed tenant 2's student to %q", name)
	}
}

// A handler that answers 200 but whose transaction can no longer commit (here: it rolled back itself, the same state
// a swallowed SQL error leaves) must not reach the client as success.
func TestRequestTxCommitFailureIsNotSuccess(t *testing.T) {
	f := setup(t)
	before := studentName(t, f)
	h := db.RequestTx(f.app, resolveHeader)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rename(w, r, "lost-write", http.StatusOK)
		tx, _ := db.TxFrom(r.Context())
		_ = tx.Rollback(r.Context())
		_, _ = w.Write([]byte("saved!"))
	}))
	got := do(h, http.MethodPost, f.t2, false)
	if got.Code != http.StatusInternalServerError || strings.Contains(got.Body.String(), "saved!") {
		t.Fatalf("want a 500 without the handler's body, got %d %q", got.Code, got.Body.String())
	}
	if name := studentName(t, f); name != before {
		t.Fatalf("name changed to %q", name)
	}
}

func TestRequestTxBadSession(t *testing.T) {
	f := setup(t)
	h := db.RequestTx(f.app, resolveHeader)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler ran without a tenant")
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/x", nil) // no tenant header
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rec.Code)
	}
}

// /readyz fails while a database is unreachable and recovers with it.
func TestReadyzFollowsThePool(t *testing.T) {
	appURL := os.Getenv("TEST_APP_URL")
	if appURL == "" {
		t.Skip("TEST_APP_URL not set")
	}
	pool, err := db.OpenApp(t.Context(), config.Secret(appURL), db.Options{Name: "test-ready", MaxConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	ready := httpx.Readyz(httpx.Check{Name: "db-app", Fn: db.Ping(pool)})
	get := func() int {
		rec := httptest.NewRecorder()
		ready(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
		return rec.Code
	}
	if code := get(); code != http.StatusOK {
		t.Fatalf("healthy pool: %d", code)
	}
	pool.Close()
	if code := get(); code != http.StatusServiceUnavailable {
		t.Fatalf("closed pool: %d, want 503", code)
	}
}

// And the other way round: the platform pool must refuse the RLS role (every admin screen would be silently empty).
func TestPlatformOpenRefusesAppRole(t *testing.T) {
	appURL := os.Getenv("TEST_APP_URL")
	if appURL == "" {
		t.Skip("TEST_APP_URL not set")
	}
	pool, err := platformdb.Open(t.Context(), config.Secret(appURL), db.Options{Name: "test-wrong", MaxConns: 1})
	if err == nil {
		pool.Close()
		t.Fatal("platformdb.Open accepted an RLS role")
	}
}

// A pool with the wrong role must not start (the platform URL in DATABASE_URL would serve tenants without RLS).
func TestOpenAppRefusesPlatformRole(t *testing.T) {
	platURL := os.Getenv("TEST_PLATFORM_URL")
	if platURL == "" {
		t.Skip("TEST_PLATFORM_URL not set")
	}
	pool, err := db.OpenApp(t.Context(), config.Secret(platURL), db.Options{Name: "test-wrong", MaxConns: 1})
	if err == nil {
		pool.Close()
		t.Fatal("OpenApp accepted a BYPASSRLS role")
	}
}
