//go:build integration

package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/tenant/auth"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/argon2id"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/config"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/db"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/db/platformdb"
)

const (
	testOrigin   = "https://portal.test"
	ownerPass    = "correct horse battery"
	driverPIN    = "482916"
	installOne   = "install-aaaaaaaaaaaaaaaa"
	installTwo   = "install-bbbbbbbbbbbbbbbb"
	ownerMobile  = "9811111111"
	driverMobile = "9822222222"
)

// authEnv is the real API (mountAPI on the sf_app pool) plus the platform pool for seeding and inspecting rows.
type authEnv struct {
	h    http.Handler
	exec func(t *testing.T, sql string, args ...any)
	q    func(t *testing.T, sql string, dst any, args ...any)
}

type school struct {
	id       int64
	code     string
	ownerID  int64
	driverID int64
}

func newAuthEnv(t *testing.T) (*authEnv, school, school) {
	t.Helper()
	appURL, platURL := os.Getenv("TEST_APP_URL"), os.Getenv("TEST_PLATFORM_URL")
	if appURL == "" || platURL == "" {
		t.Skip("TEST_APP_URL / TEST_PLATFORM_URL not set")
	}
	ctx := t.Context()
	plat, err := platformdb.Open(ctx, config.Secret(platURL), db.Options{Name: "test-auth-plat", MaxConns: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(plat.Close)
	app, err := db.OpenApp(ctx, config.Secret(appURL), db.Options{Name: "test-auth-app", MaxConns: 4})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Close)

	pw, _ := argon2id.Hash(ownerPass)
	pinHash, _ := argon2id.Hash(driverPIN)
	var a, b school
	err = platformdb.InPlatform(ctx, plat, func(tx pgx.Tx) error {
		for _, s := range []*school{&a, &b} {
			code := "auth-" + strings.ToLower(rand.Text()[:10])
			id, campus, _, err := seedSchool(ctx, tx, code)
			if err != nil {
				return err
			}
			s.id, s.code = id, code
			if err := tx.QueryRow(ctx, `INSERT INTO users (tenant_id, campus_id, name, mobile, role, password_hash)
				VALUES ($1, NULL, 'Owner', $2, 'owner', $3) RETURNING id`, id, ownerMobile, pw).Scan(&s.ownerID); err != nil {
				return fmt.Errorf("owner: %w", err)
			}
			if err := tx.QueryRow(ctx, `INSERT INTO users (tenant_id, campus_id, name, mobile, role, pin_hash)
				VALUES ($1, $2, 'Driver', $3, 'driver', $4) RETURNING id`, id, campus, driverMobile, pinHash).Scan(&s.driverID); err != nil {
				return fmt.Errorf("driver: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	r := chi.NewRouter()
	mountAPI(r, db.Bind(app), []string{testOrigin})
	env := &authEnv{h: r}
	env.exec = func(t *testing.T, sql string, args ...any) {
		t.Helper()
		if _, err := plat.Exec(t.Context(), sql, args...); err != nil {
			t.Fatalf("exec %q: %v", sql, err)
		}
	}
	env.q = func(t *testing.T, sql string, dst any, args ...any) {
		t.Helper()
		if err := plat.QueryRow(t.Context(), sql, args...).Scan(dst); err != nil {
			t.Fatalf("query %q: %v", sql, err)
		}
	}
	return env, a, b
}

// ── request helpers ──

type call struct {
	method, path string
	body         any
	headers      map[string]string
	cookies      []*http.Cookie
}

type resp struct {
	code    int
	body    map[string]any
	raw     string
	cookies []*http.Cookie
}

func (e *authEnv) do(t *testing.T, c call) resp {
	t.Helper()
	var body *strings.Reader
	if c.body != nil {
		b, _ := json.Marshal(c.body)
		body = strings.NewReader(string(b))
	} else {
		body = strings.NewReader("")
	}
	req := httptest.NewRequest(c.method, c.path, body)
	if c.body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	for _, ck := range c.cookies {
		req.AddCookie(ck)
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	out := resp{code: rec.Code, raw: rec.Body.String(), cookies: rec.Result().Cookies()}
	_ = json.Unmarshal(rec.Body.Bytes(), &out.body)
	return out
}

func portalLoginCall(code, mobile, pass string) call {
	return call{method: http.MethodPost, path: auth.PathLogin, headers: map[string]string{"Origin": testOrigin},
		body: map[string]string{"school_code": code, "mobile": mobile, "password": pass}}
}

func pinCall(code, mobile, pin, install string) call {
	return call{method: http.MethodPost, path: auth.PathPIN,
		body: map[string]string{"school_code": code, "mobile": mobile, "pin": pin, "install_id": install}}
}

// portalSession signs the owner of s in and returns its cookies.
func (e *authEnv) portalSession(t *testing.T, s school) (sid, csrf *http.Cookie) {
	t.Helper()
	r := e.do(t, portalLoginCall(s.code, ownerMobile, ownerPass))
	if r.code != http.StatusOK {
		t.Fatalf("login: %d %s", r.code, r.raw)
	}
	for _, c := range r.cookies {
		switch c.Name {
		case auth.SessionCookie:
			sid = c
		case auth.CSRFCookie:
			csrf = c
		}
	}
	if sid == nil || csrf == nil {
		t.Fatalf("login set cookies %v", r.cookies)
	}
	return sid, csrf
}

func (e *authEnv) pinSession(t *testing.T, s school, install string) string {
	t.Helper()
	r := e.do(t, pinCall(s.code, driverMobile, driverPIN, install))
	if r.code != http.StatusOK {
		t.Fatalf("pin login: %d %s", r.code, r.raw)
	}
	tok, _ := r.body["token"].(string)
	if tok == "" {
		t.Fatalf("pin login: no token in %s", r.raw)
	}
	return tok
}

func meWithCookie(sid *http.Cookie) call {
	return call{method: http.MethodGet, path: auth.PathMe, cookies: []*http.Cookie{sid}}
}

func meWithBearer(tok, install string) call {
	h := map[string]string{"Authorization": "Bearer " + tok}
	if install != "" {
		h[auth.InstallHeader] = install
	}
	return call{method: http.MethodGet, path: auth.PathMe, headers: h}
}

func wantCode(t *testing.T, what string, r resp, code int, errCode string) {
	t.Helper()
	if r.code != code {
		t.Fatalf("%s: got %d %s, want %d", what, r.code, r.raw, code)
	}
	if errCode != "" && r.body["code"] != errCode {
		t.Fatalf("%s: code %v, want %s (%s)", what, r.body["code"], errCode, r.raw)
	}
}

// ── portal ──

func TestPortalLogin(t *testing.T) {
	e, a, b := newAuthEnv(t)

	r := e.do(t, portalLoginCall(strings.ToUpper(a.code), ownerMobile, ownerPass))
	wantCode(t, "login", r, http.StatusOK, "")
	var sid, csrf *http.Cookie
	for _, c := range r.cookies {
		switch c.Name {
		case auth.SessionCookie:
			sid = c
		case auth.CSRFCookie:
			csrf = c
		}
	}
	if sid == nil || !sid.HttpOnly || !sid.Secure || sid.SameSite != http.SameSiteStrictMode || sid.Path != "/" ||
		sid.Domain != "" || !strings.HasPrefix(sid.Value, fmt.Sprint(a.id)+".") {
		t.Fatalf("session cookie %+v", sid)
	}
	if csrf == nil || csrf.HttpOnly || !csrf.Secure || csrf.SameSite != http.SameSiteStrictMode {
		t.Fatalf("csrf cookie %+v", csrf)
	}

	me := e.do(t, meWithCookie(sid))
	wantCode(t, "me", me, http.StatusOK, "")
	user, _ := me.body["user"].(map[string]any)
	school, _ := me.body["school"].(map[string]any)
	if me.body["client"] != "web" || user["role"] != "owner" || user["id"] != float64(a.ownerID) ||
		school["id"] != float64(a.id) || school["code"] != a.code || me.body["read_only"] != false {
		t.Fatalf("me: %s", me.raw)
	}

	// Only the hash is stored.
	var n int
	e.q(t, `SELECT count(*) FROM sessions WHERE tenant_id = $1 AND token_hash = convert_to($2, 'UTF8')`, &n, a.id, sid.Value)
	if n != 0 {
		t.Fatal("raw token stored")
	}

	// School B's owner has the same mobile and password, and gets school B.
	sidB, _ := e.portalSession(t, b)
	meB := e.do(t, meWithCookie(sidB))
	if s, _ := meB.body["school"].(map[string]any); s["id"] != float64(b.id) {
		t.Fatalf("me B: %s", meB.raw)
	}
}

func TestPortalLoginNoOracle(t *testing.T) {
	e, a, _ := newAuthEnv(t)
	cases := map[string]call{
		"unknown school":           portalLoginCall("no-such-school", ownerMobile, ownerPass),
		"unknown mobile":           portalLoginCall(a.code, "9899999999", ownerPass),
		"wrong password":           portalLoginCall(a.code, ownerMobile, "wrong"),
		"driver on the portal":     portalLoginCall(a.code, driverMobile, driverPIN),
		"other school's password?": portalLoginCall("no-such-school", driverMobile, "x"),
	}
	var first string
	for name, c := range cases {
		r := e.do(t, c)
		wantCode(t, name, r, http.StatusUnauthorized, "bad_credentials")
		delete(r.body, "request_id")
		b, _ := json.Marshal(r.body)
		if first == "" {
			first = string(b)
		} else if string(b) != first {
			t.Fatalf("%s: body %s differs from %s", name, b, first)
		}
		if len(r.cookies) != 0 {
			t.Fatalf("%s: set cookies on failure", name)
		}
	}
}

func TestPortalLoginOrigin(t *testing.T) {
	e, a, _ := newAuthEnv(t)
	for name, origin := range map[string]string{"none": "", "other site": "https://evil.example"} {
		c := portalLoginCall(a.code, ownerMobile, ownerPass)
		c.headers = map[string]string{}
		if origin != "" {
			c.headers["Origin"] = origin
		}
		wantCode(t, name, e.do(t, c), http.StatusForbidden, "csrf_failed")
	}
}

func TestPortalBackoff(t *testing.T) {
	e, a, _ := newAuthEnv(t)
	for i := range 5 {
		wantCode(t, fmt.Sprintf("wrong %d", i+1), e.do(t, portalLoginCall(a.code, ownerMobile, "wrong")),
			http.StatusUnauthorized, "bad_credentials")
	}
	// Backing off: even the right password gets the same 401, and doesn't extend the wait.
	wantCode(t, "right during backoff", e.do(t, portalLoginCall(a.code, ownerMobile, ownerPass)),
		http.StatusUnauthorized, "bad_credentials")
	var failed int
	e.q(t, `SELECT failed_logins FROM users WHERE id = $1`, &failed, a.ownerID)
	if failed != 5 {
		t.Fatalf("failed_logins = %d, want 5", failed)
	}
	var secs float64
	e.q(t, `SELECT extract(epoch FROM locked_until - now()) FROM users WHERE id = $1`, &secs, a.ownerID)
	if secs < 20 || secs > 31 {
		t.Fatalf("backoff = %.0f s, want ~30", secs)
	}
	// The window passes: the right password works and the count resets.
	e.exec(t, `UPDATE users SET locked_until = now() - interval '1 second' WHERE id = $1`, a.ownerID)
	wantCode(t, "after backoff", e.do(t, portalLoginCall(a.code, ownerMobile, ownerPass)), http.StatusOK, "")
	e.q(t, `SELECT failed_logins FROM users WHERE id = $1`, &failed, a.ownerID)
	if failed != 0 {
		t.Fatalf("failed_logins after success = %d", failed)
	}
	// Doubling, capped at 15 minutes.
	e.exec(t, `UPDATE users SET failed_logins = 30, locked_until = NULL WHERE id = $1`, a.ownerID)
	e.do(t, portalLoginCall(a.code, ownerMobile, "wrong"))
	e.q(t, `SELECT extract(epoch FROM locked_until - now()) FROM users WHERE id = $1`, &secs, a.ownerID)
	if secs < 890 || secs > 901 {
		t.Fatalf("capped backoff = %.0f s, want 900", secs)
	}
}

func TestSessionTokenTampering(t *testing.T) {
	e, a, b := newAuthEnv(t)
	sid, _ := e.portalSession(t, a)
	_, secret, _ := strings.Cut(sid.Value, ".")
	for name, v := range map[string]string{
		"school B's id with A's token": fmt.Sprintf("%d.%s", b.id, secret),
		"malformed":                    "abc",
		"unknown token":                fmt.Sprintf("%d.%s", a.id, strings.Repeat("A", 43)),
	} {
		c := meWithCookie(&http.Cookie{Name: auth.SessionCookie, Value: v})
		wantCode(t, name, e.do(t, c), http.StatusUnauthorized, "unauthenticated")
	}
	wantCode(t, "no session", e.do(t, call{method: http.MethodGet, path: auth.PathMe}), http.StatusUnauthorized, "unauthenticated")
	// A portal session can't be used as a bearer token (no CSRF on bearer requests).
	wantCode(t, "portal token as bearer", e.do(t, meWithBearer(sid.Value, installOne)), http.StatusUnauthorized, "unauthenticated")
}

func TestPortalCSRFAndLogout(t *testing.T) {
	e, a, _ := newAuthEnv(t)
	sid, csrf := e.portalSession(t, a)
	logout := func(origin, token string) call {
		h := map[string]string{}
		if origin != "" {
			h["Origin"] = origin
		}
		if token != "" {
			h[auth.CSRFHeader] = token
		}
		return call{method: http.MethodPost, path: auth.PathLogout, headers: h, cookies: []*http.Cookie{sid, csrf}}
	}
	wantCode(t, "no csrf header", e.do(t, logout(testOrigin, "")), http.StatusForbidden, "csrf_failed")
	wantCode(t, "wrong csrf header", e.do(t, logout(testOrigin, "x"+csrf.Value[1:])), http.StatusForbidden, "csrf_failed")
	wantCode(t, "other origin", e.do(t, logout("https://evil.example", csrf.Value)), http.StatusForbidden, "csrf_failed")
	wantCode(t, "still signed in", e.do(t, meWithCookie(sid)), http.StatusOK, "")

	r := e.do(t, logout(testOrigin, csrf.Value))
	wantCode(t, "logout", r, http.StatusNoContent, "")
	cleared := 0
	for _, c := range r.cookies {
		if (c.Name == auth.SessionCookie || c.Name == auth.CSRFCookie) && c.MaxAge < 0 {
			cleared++
		}
	}
	if cleared != 2 {
		t.Fatalf("logout cookies %v", r.cookies)
	}
	wantCode(t, "after logout", e.do(t, meWithCookie(sid)), http.StatusUnauthorized, "unauthenticated")
}

func TestPortalSessionExpiry(t *testing.T) {
	e, a, _ := newAuthEnv(t)
	idle, _ := e.portalSession(t, a)
	e.exec(t, `UPDATE sessions SET last_seen_at = now() - interval '12 hours 1 minute'
		WHERE tenant_id = $1 AND user_id = $2 AND revoked_at IS NULL`, a.id, a.ownerID)
	wantCode(t, "idle 12 h", e.do(t, meWithCookie(idle)), http.StatusUnauthorized, "unauthenticated")

	abs, _ := e.portalSession(t, a)
	e.exec(t, `UPDATE sessions SET expires_at = now() - interval '1 second'
		WHERE tenant_id = $1 AND user_id = $2 AND revoked_at IS NULL`, a.id, a.ownerID)
	wantCode(t, "absolute end", e.do(t, meWithCookie(abs)), http.StatusUnauthorized, "unauthenticated")

	// The absolute end is 7 days and use doesn't extend it. (The two dead sessions above are retired first, so the
	// queries below see only the fresh one.)
	e.exec(t, `UPDATE sessions SET revoked_at = now() WHERE tenant_id = $1 AND user_id = $2 AND revoked_at IS NULL`,
		a.id, a.ownerID)
	fresh, _ := e.portalSession(t, a)
	e.exec(t, `UPDATE sessions SET last_seen_at = now() - interval '6 minutes'
		WHERE tenant_id = $1 AND user_id = $2 AND revoked_at IS NULL`, a.id, a.ownerID)
	wantCode(t, "touch", e.do(t, meWithCookie(fresh)), http.StatusOK, "")
	var days float64
	e.q(t, `SELECT extract(epoch FROM expires_at - now()) / 86400 FROM sessions
		WHERE tenant_id = $1 AND user_id = $2 AND revoked_at IS NULL`, &days, a.id, a.ownerID)
	if days > 7.001 || days < 6.99 {
		t.Fatalf("portal session ends in %.3f days, want 7", days)
	}
	var seenAgo float64
	e.q(t, `SELECT extract(epoch FROM now() - last_seen_at) FROM sessions
		WHERE tenant_id = $1 AND user_id = $2 AND revoked_at IS NULL`, &seenAgo, a.id, a.ownerID)
	if seenAgo > 60 {
		t.Fatalf("not touched: last seen %.0f s ago", seenAgo)
	}
}

func TestLoginRevokesPresentedSession(t *testing.T) {
	e, a, _ := newAuthEnv(t)
	old, _ := e.portalSession(t, a)
	c := portalLoginCall(a.code, ownerMobile, ownerPass)
	c.cookies = []*http.Cookie{old}
	r := e.do(t, c)
	wantCode(t, "re-login", r, http.StatusOK, "")
	for _, ck := range r.cookies {
		if ck.Name == auth.SessionCookie && ck.Value == old.Value {
			t.Fatal("session reused")
		}
	}
	wantCode(t, "old session", e.do(t, meWithCookie(old)), http.StatusUnauthorized, "unauthenticated")
}

func TestSchoolStatus(t *testing.T) {
	e, a, _ := newAuthEnv(t)
	e.exec(t, `UPDATE tenants SET status = 'suspended' WHERE id = $1`, a.id)
	sid, csrf := e.portalSession(t, a)
	me := e.do(t, meWithCookie(sid))
	if me.body["read_only"] != true {
		t.Fatalf("suspended: %s", me.raw)
	}
	// A read-only session can still sign out.
	wantCode(t, "logout read-only", e.do(t, call{method: http.MethodPost, path: auth.PathLogout,
		headers: map[string]string{"Origin": testOrigin, auth.CSRFHeader: csrf.Value},
		cookies: []*http.Cookie{sid, csrf}}), http.StatusNoContent, "")

	live, _ := e.portalSession(t, a)
	e.exec(t, `UPDATE tenants SET status = 'cancelled' WHERE id = $1`, a.id)
	wantCode(t, "wrong password, cancelled", e.do(t, portalLoginCall(a.code, ownerMobile, "wrong")),
		http.StatusUnauthorized, "bad_credentials")
	wantCode(t, "right password, cancelled", e.do(t, portalLoginCall(a.code, ownerMobile, ownerPass)),
		http.StatusLocked, "school_closed")
	wantCode(t, "existing session, cancelled", e.do(t, meWithCookie(live)), http.StatusLocked, "school_closed")
}

func TestInactiveUser(t *testing.T) {
	e, a, _ := newAuthEnv(t)
	sid, _ := e.portalSession(t, a)
	e.exec(t, `UPDATE users SET is_active = false WHERE id = $1`, a.ownerID)
	wantCode(t, "session of a deactivated user", e.do(t, meWithCookie(sid)), http.StatusUnauthorized, "unauthenticated")
	wantCode(t, "login of a deactivated user", e.do(t, portalLoginCall(a.code, ownerMobile, ownerPass)),
		http.StatusUnauthorized, "bad_credentials")
}

// ── driver app ──

func TestPINLoginAndInstallBinding(t *testing.T) {
	e, a, _ := newAuthEnv(t)
	r := e.do(t, pinCall(a.code, driverMobile, driverPIN, installOne))
	wantCode(t, "pin", r, http.StatusOK, "")
	tok, _ := r.body["token"].(string)
	user, _ := r.body["user"].(map[string]any)
	if !strings.HasPrefix(tok, fmt.Sprint(a.id)+".") || user["role"] != "driver" || r.body["client"] != "driver_app" {
		t.Fatalf("pin login: %s", r.raw)
	}
	if len(r.cookies) != 0 {
		t.Fatal("app login set cookies")
	}

	wantCode(t, "bearer + install", e.do(t, meWithBearer(tok, installOne)), http.StatusOK, "")
	wantCode(t, "no install header", e.do(t, meWithBearer(tok, "")), http.StatusUnauthorized, "unauthenticated")
	wantCode(t, "other install", e.do(t, meWithBearer(tok, installTwo)), http.StatusUnauthorized, "unauthenticated")
	wantCode(t, "app token as cookie", e.do(t, meWithCookie(&http.Cookie{Name: auth.SessionCookie, Value: tok})),
		http.StatusUnauthorized, "unauthenticated")
	wantCode(t, "basic auth", e.do(t, call{method: http.MethodGet, path: auth.PathMe,
		headers: map[string]string{"Authorization": "Basic " + tok, auth.InstallHeader: installOne}}),
		http.StatusUnauthorized, "unauthenticated")

	// Wrong kind of account each way, and a wrong PIN: the same 401.
	wantCode(t, "owner via pin", e.do(t, pinCall(a.code, ownerMobile, driverPIN, installOne)), http.StatusUnauthorized, "bad_credentials")
	wantCode(t, "wrong pin", e.do(t, pinCall(a.code, driverMobile, "000000", installOne)), http.StatusUnauthorized, "bad_credentials")
	wantCode(t, "bad pin shape", e.do(t, pinCall(a.code, driverMobile, "12345", installOne)), http.StatusBadRequest, "validation_failed")
}

func TestPINOneSessionPerDriver(t *testing.T) {
	e, a, b := newAuthEnv(t)
	first := e.pinSession(t, a, installOne)
	otherSchool := e.pinSession(t, b, installOne) // same mobile in school B: a different user
	second := e.pinSession(t, a, installTwo)

	wantCode(t, "old phone", e.do(t, meWithBearer(first, installOne)), http.StatusUnauthorized, "unauthenticated")
	wantCode(t, "new phone", e.do(t, meWithBearer(second, installTwo)), http.StatusOK, "")
	wantCode(t, "school B untouched", e.do(t, meWithBearer(otherSchool, installOne)), http.StatusOK, "")
}

func TestAppSessionRenewsOnUse(t *testing.T) {
	e, a, _ := newAuthEnv(t)
	tok := e.pinSession(t, a, installOne)
	e.exec(t, `UPDATE sessions SET last_seen_at = now() - interval '10 days', expires_at = now() + interval '80 days'
		WHERE tenant_id = $1 AND user_id = $2 AND revoked_at IS NULL`, a.id, a.driverID)
	wantCode(t, "me", e.do(t, meWithBearer(tok, installOne)), http.StatusOK, "")
	var days float64
	e.q(t, `SELECT extract(epoch FROM expires_at - now()) / 86400 FROM sessions
		WHERE tenant_id = $1 AND user_id = $2 AND revoked_at IS NULL`, &days, a.id, a.driverID)
	if days < 89.99 {
		t.Fatalf("app session not renewed: ends in %.2f days", days)
	}

	// A stolen token from another install neither works nor keeps the session alive.
	e.exec(t, `UPDATE sessions SET last_seen_at = now() - interval '10 days', expires_at = now() + interval '80 days'
		WHERE tenant_id = $1 AND user_id = $2 AND revoked_at IS NULL`, a.id, a.driverID)
	e.do(t, meWithBearer(tok, installTwo))
	e.q(t, `SELECT extract(epoch FROM expires_at - now()) / 86400 FROM sessions
		WHERE tenant_id = $1 AND user_id = $2 AND revoked_at IS NULL`, &days, a.id, a.driverID)
	if days > 80.01 {
		t.Fatalf("wrong install renewed the session (%.2f days)", days)
	}
}

func TestAppLogout(t *testing.T) {
	e, a, _ := newAuthEnv(t)
	tok := e.pinSession(t, a, installOne)
	r := e.do(t, call{method: http.MethodPost, path: auth.PathLogout,
		headers: map[string]string{"Authorization": "Bearer " + tok, auth.InstallHeader: installOne}})
	wantCode(t, "logout", r, http.StatusNoContent, "")
	if len(r.cookies) != 0 {
		t.Fatal("app logout set cookies")
	}
	wantCode(t, "after logout", e.do(t, meWithBearer(tok, installOne)), http.StatusUnauthorized, "unauthenticated")
}

// Parallel wrong guesses can't get past the 5th failure: the attempt is claimed before the hash is verified.
func TestBackoffParallelGuesses(t *testing.T) {
	e, a, _ := newAuthEnv(t)
	const n = 20
	codes := make(chan int, n)
	var wg sync.WaitGroup
	for range n {
		wg.Go(func() {
			b, _ := json.Marshal(map[string]string{"school_code": a.code, "mobile": driverMobile, "pin": "000000",
				"install_id": installOne})
			req := httptest.NewRequest(http.MethodPost, auth.PathPIN, strings.NewReader(string(b)))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			e.h.ServeHTTP(rec, req)
			codes <- rec.Code
		})
	}
	wg.Wait()
	close(codes)
	for c := range codes {
		if c != http.StatusUnauthorized {
			t.Fatalf("parallel guess got %d", c)
		}
	}
	var failed int
	e.q(t, `SELECT failed_logins FROM users WHERE id = $1`, &failed, a.driverID)
	if failed != 5 {
		t.Fatalf("failed_logins = %d after %d parallel guesses, want 5 (the rest refused by backoff)", failed, n)
	}
	wantCode(t, "right PIN during backoff", e.do(t, pinCall(a.code, driverMobile, driverPIN, installOne)),
		http.StatusUnauthorized, "bad_credentials")
	e.exec(t, `UPDATE users SET locked_until = now() - interval '1 second' WHERE id = $1`, a.driverID)
	wantCode(t, "right PIN after backoff", e.do(t, pinCall(a.code, driverMobile, driverPIN, installOne)), http.StatusOK, "")
}

// Two phones signing in at the same moment still leave one live driver app session.
func TestPINConcurrentLogins(t *testing.T) {
	e, a, _ := newAuthEnv(t)
	for round := range 5 {
		var wg sync.WaitGroup
		for _, inst := range []string{installOne, installTwo} {
			wg.Go(func() {
				b, _ := json.Marshal(map[string]string{"school_code": a.code, "mobile": driverMobile, "pin": driverPIN,
					"install_id": inst})
				req := httptest.NewRequest(http.MethodPost, auth.PathPIN, strings.NewReader(string(b)))
				req.Header.Set("Content-Type", "application/json")
				e.h.ServeHTTP(httptest.NewRecorder(), req)
			})
		}
		wg.Wait()
		var live int
		e.q(t, `SELECT count(*) FROM sessions WHERE tenant_id = $1 AND user_id = $2 AND client = 'driver_app'
			AND revoked_at IS NULL`, &live, a.id, a.driverID)
		if live != 1 {
			t.Fatalf("round %d: %d live driver sessions, want 1", round, live)
		}
	}
}
