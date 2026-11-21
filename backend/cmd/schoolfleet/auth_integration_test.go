//go:build integration

package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/admin/adminauth"
	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/shared/otp"
	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/tenant/auth"
	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/worker"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/argon2id"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/clock"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/config"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/db"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/db/platformdb"
)

const (
	testOrigin   = "https://portal.test"
	testRPID     = "portal.test"
	ownerPass    = "correct horse battery"
	driverPIN    = "482916"
	installOne   = "install-aaaaaaaaaaaaaaaa"
	installTwo   = "install-bbbbbbbbbbbbbbbb"
	ownerMobile  = "9811111111"
	driverMobile = "9822222222"
)

// authEnv is the real API (mountAPI on the sf_app pool) plus the platform pool for seeding and inspecting rows.
type authEnv struct {
	h     http.Handler
	clock *clock.Fake // the API process's clock (rate limits, sign-in backoff); the database keeps its own
	keys  otp.Keys
	plat  worker.InPlatform
	exec  func(t *testing.T, sql string, args ...any)
	q     func(t *testing.T, sql string, dst any, args ...any)
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

	keys := testOTPKeys(t)
	r := chi.NewRouter()
	fake := clock.NewFake(time.Date(2026, 11, 21, 8, 0, 0, 0, time.UTC))
	device, err := auth.NewDeviceKeys(strings.Repeat("cd", 32), fake)
	if err != nil {
		t.Fatal(err)
	}
	inPlat := func(ctx context.Context, fn func(pgx.Tx) error) error { return platformdb.InPlatform(ctx, plat, fn) }
	admin, err := adminauth.NewDeps(inPlat, testRPID, []string{testOrigin}, 12, nil, fake)
	if err != nil {
		t.Fatal(err)
	}
	mountAPI(r, auth.Deps{DB: db.Bind(app), Origins: []string{testOrigin}, OTP: keys, Device: device,
		Limits: auth.NewLimits(fake)}, admin)
	env := &authEnv{h: r, clock: fake, keys: keys, plat: inPlat}
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

// testOTPKeys derives OTP keys from a key made for this run.
func testOTPKeys(t *testing.T) otp.Keys {
	t.Helper()
	k, err := otp.NewKeys(strings.Repeat("ab", 32))
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// ── request helpers ──

type call struct {
	method, path string
	body         any
	headers      map[string]string
	cookies      []*http.Cookie
	remote       string // the client's address; httptest's 192.0.2.1 by default
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
	if c.remote != "" {
		req.RemoteAddr = c.remote + ":40000"
	}
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
	e, a, b := newAuthEnv(t)
	for i := range 5 {
		wantCode(t, fmt.Sprintf("wrong %d", i+1), e.do(t, portalLoginCall(a.code, ownerMobile, "wrong")),
			http.StatusUnauthorized, "bad_credentials")
	}
	// Backing off: even the right password gets the same 401.
	wantCode(t, "right during backoff", e.do(t, portalLoginCall(a.code, ownerMobile, ownerPass)),
		http.StatusUnauthorized, "bad_credentials")
	// The same mobile in the other school is another account: not slowed down.
	wantCode(t, "other school", e.do(t, portalLoginCall(b.code, ownerMobile, ownerPass)), http.StatusOK, "")
	var failed int
	e.q(t, `SELECT failed_logins FROM users WHERE id = $1`, &failed, a.ownerID)
	if failed != 5 {
		t.Fatalf("failed_logins = %d, want 5 (the try during backoff is not counted)", failed)
	}
	// The 30 s window passes: the right password works and the run of failures resets.
	e.clock.Advance(29 * time.Second)
	wantCode(t, "1 s early", e.do(t, portalLoginCall(a.code, ownerMobile, ownerPass)),
		http.StatusUnauthorized, "bad_credentials")
	e.clock.Advance(time.Second)
	wantCode(t, "after backoff", e.do(t, portalLoginCall(a.code, ownerMobile, ownerPass)), http.StatusOK, "")
	e.q(t, `SELECT failed_logins FROM users WHERE id = $1`, &failed, a.ownerID)
	if failed != 0 {
		t.Fatalf("failed_logins after success = %d", failed)
	}
	// The daily count keeps the 5 failures but not the success.
	var day int
	e.q(t, `SELECT fail_window_count FROM users WHERE id = $1`, &day, a.ownerID)
	if day != 5 {
		t.Fatalf("fail_window_count after success = %d, want 5", day)
	}
}

// An attacker's failures slow down only the attacker's address: the owner, from their own address, still signs in.
func TestBackoffIsPerAddress(t *testing.T) {
	e, a, _ := newAuthEnv(t)
	const attacker, owner = "203.0.113.66", "198.51.100.10"
	for range 8 {
		c := portalLoginCall(a.code, ownerMobile, "wrong")
		c.remote = attacker
		e.do(t, c)
	}
	c := portalLoginCall(a.code, ownerMobile, ownerPass)
	c.remote = attacker
	wantCode(t, "right password from the attacker's address", e.do(t, c), http.StatusUnauthorized, "bad_credentials")
	c.remote = owner
	wantCode(t, "owner's own address", e.do(t, c), http.StatusOK, "")
}

// One address gets at most 10 failures a day per mobile; every address together at most 50 per user, after which
// every try, even the right secret from a clean address, gets the same 401 until the 24 h window ends.
func TestLoginDailyCaps(t *testing.T) {
	e, a, b := newAuthEnv(t)
	guess := func(ip string) resp {
		c := pinCall(a.code, driverMobile, "000000", installOne)
		c.remote = ip
		return e.do(t, c)
	}
	// One address, patiently past every backoff window: 10 failures, then closed for the day.
	for range 30 {
		guess("203.0.113.1")
		e.clock.Advance(15 * time.Minute)
	}
	var count int
	e.q(t, `SELECT fail_window_count FROM users WHERE id = $1`, &count, a.driverID)
	if count != 10 {
		t.Fatalf("one address counted %d failures, want 10", count)
	}
	// Four more addresses reach the user's cap of 50.
	for i := 2; i <= 5; i++ {
		for range 10 {
			guess(fmt.Sprintf("203.0.113.%d", i))
			e.clock.Advance(15 * time.Minute)
		}
	}
	e.q(t, `SELECT fail_window_count FROM users WHERE id = $1`, &count, a.driverID)
	if count != 50 {
		t.Fatalf("fail_window_count = %d, want 50", count)
	}
	c := pinCall(a.code, driverMobile, driverPIN, installOne)
	c.remote = "198.51.100.10"
	wantCode(t, "right PIN after the daily cap", e.do(t, c), http.StatusUnauthorized, "bad_credentials")
	e.q(t, `SELECT fail_window_count FROM users WHERE id = $1`, &count, a.driverID)
	if count != 50 {
		t.Fatalf("a capped try was counted: %d", count)
	}
	// The other school's driver with the same mobile is untouched.
	wantCode(t, "other school", e.do(t, pinCall(b.code, driverMobile, driverPIN, installOne)), http.StatusOK, "")
	// The window ends (database clock): the right PIN works.
	e.exec(t, `UPDATE users SET fail_window_start = now() - interval '24 hours' WHERE id = $1`, a.driverID)
	c.remote = "198.51.100.11"
	wantCode(t, "next day", e.do(t, c), http.StatusOK, "")
}

// closeDailyCap puts a user at the daily cap, as if 50 failures from many addresses had just happened.
func (e *authEnv) closeDailyCap(t *testing.T, userID int64) {
	t.Helper()
	e.exec(t, `UPDATE users SET fail_window_start = now(), fail_window_count = 50 WHERE id = $1`, userID)
}

func (e *authEnv) failWindowCount(t *testing.T, userID int64) int {
	t.Helper()
	var n int
	e.q(t, `SELECT fail_window_count FROM users WHERE id = $1`, &n, userID)
	return n
}

// With the daily cap closed, the driver's own phone (an install that has had a session) still signs in, at a reduced
// rate; a new install doesn't, nor does an install known only to the other school's driver with the same mobile.
func TestKnownInstallPastDailyCap(t *testing.T) {
	e, a, b := newAuthEnv(t)
	e.pinSession(t, a, installOne)
	e.pinSession(t, b, installTwo)
	e.closeDailyCap(t, a.driverID)

	try := func(pin, install, ip string) resp {
		c := pinCall(a.code, driverMobile, pin, install)
		c.remote = ip
		return e.do(t, c)
	}
	wantCode(t, "new install", try(driverPIN, "install-cccccccccccccccc", "198.51.100.1"),
		http.StatusUnauthorized, "bad_credentials")
	wantCode(t, "install of school B's driver", try(driverPIN, installTwo, "198.51.100.2"),
		http.StatusUnauthorized, "bad_credentials")
	wantCode(t, "own phone", try(driverPIN, installOne, "198.51.100.3"), http.StatusOK, "")
	if n := e.failWindowCount(t, a.driverID); n != 50 {
		t.Fatalf("fail_window_count = %d after a known-device success, want 50 (nothing was claimed)", n)
	}

	// Reduced rate: 3 wrong PINs from the phone (different addresses, so only the device's backoff counts), then a
	// 1-minute wait even for the right PIN.
	for i := range 3 {
		wantCode(t, "wrong on own phone", try("000000", installOne, fmt.Sprintf("198.51.100.%d", 10+i)),
			http.StatusUnauthorized, "bad_credentials")
	}
	wantCode(t, "own phone backing off", try(driverPIN, installOne, "198.51.100.20"),
		http.StatusUnauthorized, "bad_credentials")
	if n := e.failWindowCount(t, a.driverID); n != 50 {
		t.Fatalf("known-device failures reached the database count: %d", n)
	}
	e.clock.Advance(time.Minute)
	wantCode(t, "own phone after the wait", try(driverPIN, installOne, "198.51.100.21"), http.StatusOK, "")
}

// The portal's device cookie does the same for a browser the owner has signed in on, and only for that school/user.
func TestDeviceCookiePastDailyCap(t *testing.T) {
	e, a, b := newAuthEnv(t)
	devOf := func(r resp) *http.Cookie {
		for _, c := range r.cookies {
			if c.Name == auth.DeviceCookie {
				return c
			}
		}
		t.Fatalf("no device cookie in %v", r.cookies)
		return nil
	}
	ra := e.do(t, portalLoginCall(a.code, ownerMobile, ownerPass))
	wantCode(t, "login A", ra, http.StatusOK, "")
	devA := devOf(ra)
	if !devA.HttpOnly || !devA.Secure || devA.SameSite != http.SameSiteStrictMode || devA.Path != "/" {
		t.Fatalf("device cookie %+v", devA)
	}
	rb := e.do(t, portalLoginCall(b.code, ownerMobile, ownerPass))
	wantCode(t, "login B", rb, http.StatusOK, "")
	devB := devOf(rb)
	e.closeDailyCap(t, a.ownerID)

	try := func(dev *http.Cookie, ip string) resp {
		c := portalLoginCall(a.code, ownerMobile, ownerPass)
		c.remote = ip
		if dev != nil {
			c.cookies = []*http.Cookie{dev}
		}
		return e.do(t, c)
	}
	wantCode(t, "no cookie", try(nil, "198.51.100.1"), http.StatusUnauthorized, "bad_credentials")
	wantCode(t, "school B's cookie", try(devB, "198.51.100.2"), http.StatusUnauthorized, "bad_credentials")
	forged := &http.Cookie{Name: auth.DeviceCookie, Value: devA.Value + "x"}
	wantCode(t, "forged cookie", try(forged, "198.51.100.3"), http.StatusUnauthorized, "bad_credentials")
	r := try(devA, "198.51.100.4")
	wantCode(t, "own browser", r, http.StatusOK, "")
	devOf(r) // refreshed
	if n := e.failWindowCount(t, a.ownerID); n != 50 {
		t.Fatalf("fail_window_count = %d, want 50", n)
	}
}

// Too many sign-ins from one address are answered 429, which says nothing about the mobile.
func TestLoginPerIPLimit(t *testing.T) {
	e, a, _ := newAuthEnv(t)
	var last resp
	for i := range 61 {
		// A different mobile each time, so the per-mobile backoff never applies.
		c := portalLoginCall(a.code, fmt.Sprintf("97%08d", i), "wrong")
		c.remote = "203.0.113.7"
		last = e.do(t, c)
	}
	wantCode(t, "61st", last, http.StatusTooManyRequests, "rate_limited")
	c := portalLoginCall(a.code, ownerMobile, ownerPass)
	c.remote = "198.51.100.10"
	wantCode(t, "another address", e.do(t, c), http.StatusOK, "")
	e.clock.Advance(10 * time.Minute)
	c.remote = "203.0.113.7"
	wantCode(t, "next window", e.do(t, c), http.StatusOK, "")
}

// A Previa support-view session (platform_user_id only) is refused until support view is built, and isn't renewed.
func TestSupportViewSessionRefused(t *testing.T) {
	e, a, b := newAuthEnv(t)
	sid, _ := e.portalSession(t, a)
	other, _ := e.portalSession(t, b)
	var puID int64
	e.q(t, `INSERT INTO platform_users (name, email, role) VALUES ('Support', $1, 'support') RETURNING id`, &puID,
		"support-"+strings.ToLower(rand.Text()[:8])+"@previa.test")
	e.exec(t, `UPDATE sessions SET user_id = NULL, platform_user_id = $1, read_only = true,
		last_seen_at = now() - interval '1 hour' WHERE tenant_id = $2 AND revoked_at IS NULL`, puID, a.id)
	wantCode(t, "support view", e.do(t, call{method: http.MethodGet, path: auth.PathMe, cookies: []*http.Cookie{sid}}),
		http.StatusUnauthorized, "unauthenticated")
	var stale bool
	e.q(t, `SELECT bool_and(last_seen_at < now() - interval '30 minutes') FROM sessions WHERE platform_user_id = $1`,
		&stale, puID)
	if !stale {
		t.Fatal("a refused support-view session was renewed")
	}
	wantCode(t, "other school's staff", e.do(t, call{method: http.MethodGet, path: auth.PathMe,
		cookies: []*http.Cookie{other}}), http.StatusOK, "")
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
	e.clock.Advance(30 * time.Second)
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
