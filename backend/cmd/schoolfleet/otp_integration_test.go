//go:build integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/shared/otp"
	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/tenant/auth"
	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/worker"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/clock"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/config"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/db"
)

const parentMobile = "9833333333"

// otpSignIn signs a guardian in through the real endpoints: request a code, read it back from the outbox the way the
// worker would (open the sealed box), verify. Returns the bearer token.
func otpSignIn(ctx context.Context, h http.Handler, keys otp.Keys, plat worker.InPlatform, tenant, guardian int64,
	install string) (string, error) {
	var mobile, code string
	if err := plat(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT g.mobile, t.code FROM guardians g JOIN tenants t ON t.id = g.tenant_id
			WHERE g.tenant_id = $1 AND g.id = $2`, tenant, guardian).Scan(&mobile, &code)
	}); err != nil {
		return "", fmt.Errorf("guardian: %w", err)
	}
	if rec := serve(h, http.MethodPost, auth.PathOTP, map[string]string{"school_code": code, "mobile": mobile}, nil); rec.Code != http.StatusAccepted {
		return "", fmt.Errorf("otp request: %d %s", rec.Code, rec.Body)
	}
	got, err := sentCode(ctx, keys, plat, tenant, guardian)
	if err != nil {
		return "", err
	}
	rec := serve(h, http.MethodPost, auth.PathOTPVerify, map[string]string{"school_code": code, "mobile": mobile,
		"code": got, "install_id": install}, nil)
	var out struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if rec.Code != http.StatusOK || out.Token == "" {
		return "", fmt.Errorf("otp verify: %d %s", rec.Code, rec.Body)
	}
	return out.Token, nil
}

// sentCode opens the newest queued code for the guardian.
func sentCode(ctx context.Context, keys otp.Keys, plat worker.InPlatform, tenant, guardian int64) (string, error) {
	var box, key string
	if err := plat(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT data->>'box', dedupe_key FROM notifications
			WHERE tenant_id = $1 AND guardian_id = $2 AND event = 'otp' AND status = 'queued'
			ORDER BY id DESC LIMIT 1`, tenant, guardian).Scan(&box, &key)
	}); err != nil {
		return "", fmt.Errorf("queued code: %w", err)
	}
	return keys.Open(box, otp.BoxAAD(tenant, key))
}

func serve(h http.Handler, method, path string, body any, hdr map[string]string) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// ── fixtures ──

func (e *authEnv) guardian(t *testing.T, s school, mobile string, appEnabled bool) int64 {
	t.Helper()
	var id int64
	e.q(t, `INSERT INTO guardians (tenant_id, name, mobile, app_enabled) VALUES ($1, 'Parent', $2, $3) RETURNING id`,
		&id, s.id, mobile, appEnabled)
	return id
}

func otpCall(code, mobile string) call {
	return call{method: http.MethodPost, path: auth.PathOTP, body: map[string]string{"school_code": code, "mobile": mobile}}
}

func verifyCall(code, mobile, otpCode, install string) call {
	return call{method: http.MethodPost, path: auth.PathOTPVerify,
		body: map[string]string{"school_code": code, "mobile": mobile, "code": otpCode, "install_id": install}}
}

func (e *authEnv) code(t *testing.T, s school, guardian int64) string {
	t.Helper()
	c, err := sentCode(t.Context(), e.keys, e.plat, s.id, guardian)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func (e *authEnv) count(t *testing.T, sql string, args ...any) int {
	t.Helper()
	var n int
	e.q(t, sql, &n, args...)
	return n
}

// wrongCode is a code that differs from c.
func wrongCode(c string) string {
	if c == "000000" {
		return "000001"
	}
	return "000000"
}

func (e *authEnv) parentToken(t *testing.T, s school, g int64, install string) string {
	t.Helper()
	tok, err := otpSignIn(t.Context(), e.h, e.keys, e.plat, s.id, g, install)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

// ── sign-in ──

func TestOTPLogin(t *testing.T) {
	e, a, _ := newAuthEnv(t)
	g := e.guardian(t, a, parentMobile, true)
	wantCode(t, "request", e.do(t, otpCall(a.code, parentMobile)), http.StatusAccepted, "")
	code := e.code(t, a, g)

	// Nothing about the code is stored in the clear.
	if n := e.count(t, `SELECT count(*) FROM notifications WHERE tenant_id = $1 AND (body LIKE '%'||$2||'%' OR data::text LIKE '%'||$2||'%')`, a.id, code); n != 0 {
		t.Fatal("code stored in the clear in the outbox")
	}

	r := e.do(t, verifyCall(strings.ToUpper(a.code), parentMobile, code, installOne))
	wantCode(t, "verify", r, http.StatusOK, "")
	tok, _ := r.body["token"].(string)
	gb, _ := r.body["guardian"].(map[string]any)
	if r.body["client"] != auth.ClientParentApp || gb["id"] != float64(g) || r.body["user"] != nil {
		t.Fatalf("verify body %s", r.raw)
	}
	me := e.do(t, meWithBearer(tok, installOne))
	wantCode(t, "me", me, http.StatusOK, "")
	if me.body["client"] != auth.ClientParentApp {
		t.Fatalf("me %s", me.raw)
	}
	wantCode(t, "other install", e.do(t, meWithBearer(tok, installTwo)), http.StatusUnauthorized, "unauthenticated")

	// 90 days, bound to the install.
	if n := e.count(t, `SELECT count(*) FROM sessions WHERE tenant_id = $1 AND guardian_id = $2 AND client = 'parent_app'
		AND install_id = $3 AND expires_at BETWEEN now() + interval '89 days' AND now() + interval '91 days'`, a.id, g, installOne); n != 1 {
		t.Fatalf("sessions: %d", n)
	}
	// A code works once.
	wantCode(t, "reuse", e.do(t, verifyCall(a.code, parentMobile, code, installOne)), http.StatusUnauthorized, "bad_credentials")
}

func TestOTPNoOracle(t *testing.T) {
	e, a, _ := newAuthEnv(t)
	disabled := e.guardian(t, a, "9844444444", false)
	for name, c := range map[string]call{
		"unknown school": otpCall("no-such-school", parentMobile),
		"unknown mobile": otpCall(a.code, "9855555555"),
		"app disabled":   otpCall(a.code, "9844444444"),
	} {
		r := e.do(t, c)
		if r.code != http.StatusAccepted || r.raw != "" {
			t.Fatalf("%s: %d %q", name, r.code, r.raw)
		}
	}
	// Every mobile in a known school gets a challenge; only guardians get an SMS.
	if n := e.count(t, `SELECT count(*) FROM otp_challenges WHERE tenant_id = $1 AND mobile IN ('9855555555','9844444444')`, a.id); n != 2 {
		t.Fatalf("challenges: %d", n)
	}
	if n := e.count(t, `SELECT count(*) FROM notifications WHERE tenant_id = $1 AND event = 'otp'`, a.id); n != 0 {
		t.Fatalf("SMS queued for a non-guardian: %d", n)
	}
	_ = disabled

	g := e.guardian(t, a, parentMobile, true)
	e.do(t, otpCall(a.code, parentMobile))
	real := e.code(t, a, g)
	bodies := map[string]string{}
	for name, c := range map[string]call{
		"unknown school": verifyCall("no-such-school", parentMobile, real, installOne),
		"unknown mobile": verifyCall(a.code, "9855555555", "123456", installOne),
		"app disabled":   verifyCall(a.code, "9844444444", "123456", installOne),
		"no challenge":   verifyCall(a.code, "9866666666", "123456", installOne),
		"wrong code":     verifyCall(a.code, parentMobile, wrongCode(real), installOne),
	} {
		r := e.do(t, c)
		wantCode(t, name, r, http.StatusUnauthorized, "bad_credentials")
		bodies[r.raw] = name
	}
	if len(bodies) != 1 {
		t.Fatalf("401 bodies differ: %v", bodies)
	}
}

func TestOTPAttemptsAndExpiry(t *testing.T) {
	e, a, _ := newAuthEnv(t)
	g := e.guardian(t, a, parentMobile, true)
	e.do(t, otpCall(a.code, parentMobile))
	code := e.code(t, a, g)
	// Parallel wrong guesses: each is counted on the locked row, never more than 5.
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() { e.do(t, verifyCall(a.code, parentMobile, wrongCode(code), installOne)) })
	}
	wg.Wait()
	if n := e.count(t, `SELECT max(attempts) FROM otp_challenges WHERE tenant_id = $1 AND mobile = $2`, a.id, parentMobile); n != 5 {
		t.Fatalf("attempts %d, want 5", n)
	}
	wantCode(t, "right code after 5 tries", e.do(t, verifyCall(a.code, parentMobile, code, installOne)),
		http.StatusUnauthorized, "bad_credentials")

	// A fresh code (the hourly limit allows 3), then expiry.
	e.do(t, otpCall(a.code, parentMobile))
	code = e.code(t, a, g)
	e.exec(t, `UPDATE otp_challenges SET expires_at = now() - interval '1 second' WHERE tenant_id = $1 AND consumed_at IS NULL`, a.id)
	wantCode(t, "expired", e.do(t, verifyCall(a.code, parentMobile, code, installOne)), http.StatusUnauthorized, "bad_credentials")
}

func TestOTPNewCodeEndsOld(t *testing.T) {
	e, a, _ := newAuthEnv(t)
	g := e.guardian(t, a, parentMobile, true)
	e.do(t, otpCall(a.code, parentMobile))
	first := e.code(t, a, g)
	e.do(t, otpCall(a.code, parentMobile))
	second := e.code(t, a, g)
	if first != second {
		wantCode(t, "old code", e.do(t, verifyCall(a.code, parentMobile, first, installOne)), http.StatusUnauthorized, "bad_credentials")
	}
	wantCode(t, "new code", e.do(t, verifyCall(a.code, parentMobile, second, installOne)), http.StatusOK, "")
}

func TestOTPPerMobileLimit(t *testing.T) {
	e, a, b := newAuthEnv(t)
	g := e.guardian(t, a, parentMobile, true)
	gb := e.guardian(t, b, parentMobile, true)
	for range 3 {
		e.do(t, otpCall(a.code, parentMobile))
	}
	third := e.code(t, a, g)
	wantCode(t, "4th", e.do(t, otpCall(a.code, parentMobile)), http.StatusAccepted, "")
	if n := e.count(t, `SELECT count(*) FROM otp_challenges WHERE tenant_id = $1 AND mobile = $2`, a.id, parentMobile); n != 3 {
		t.Fatalf("challenges after the 4th request: %d", n)
	}
	// The code already sent still works, and the same mobile in the other school has its own count.
	e.do(t, otpCall(b.code, parentMobile))
	if e.code(t, b, gb) == "" {
		t.Fatal("other school's code not queued")
	}
	wantCode(t, "3rd code still valid", e.do(t, verifyCall(a.code, parentMobile, third, installOne)), http.StatusOK, "")
}

func TestOTPPerSchoolLimit(t *testing.T) {
	e, a, b := newAuthEnv(t)
	// Live dates are IST, like the query's: A went live 14 days ago (past the launch window), B 13 days ago.
	golive := `UPDATE tenants SET onboarding_stage = 'live', go_live_on = (now() AT TIME ZONE 'Asia/Kolkata')::date - $2::int
		WHERE id = $1`
	e.exec(t, golive, a.id, 14)
	e.exec(t, golive, b.id, 13)
	queued := `SELECT count(*) FROM notifications WHERE tenant_id = $1 AND event = 'otp'`
	next := 0
	request := func(s school, n int) {
		for range n {
			m := fmt.Sprintf("97%08d", next)
			next++
			e.guardian(t, s, m, true)
			c := otpCall(s.code, m)
			c.remote = fmt.Sprintf("203.0.113.%d", next%250) // parents on many networks: only the school limit applies
			wantCode(t, "request", e.do(t, c), http.StatusAccepted, "")
		}
	}

	request(a, 11)
	if n := e.count(t, queued, a.id); n != 10 {
		t.Fatalf("live school A queued %d, want 10", n)
	}
	request(b, 61)
	if n := e.count(t, queued, b.id); n != 60 {
		t.Fatalf("school B (13 days live) queued %d, want 60", n)
	}
	// A school not live yet is in the launch window too; A's 10 still count towards its 60.
	e.exec(t, `UPDATE tenants SET onboarding_stage = 'training', go_live_on = NULL WHERE id = $1`, a.id)
	request(a, 51)
	if n := e.count(t, queued, a.id); n != 60 {
		t.Fatalf("school A in training queued %d, want 60", n)
	}
	// Live with no go-live date: outside the window (10/min), and still a 202, not a 500.
	e.exec(t, `UPDATE tenants SET onboarding_stage = 'live', go_live_on = NULL WHERE id = $1`, b.id)
	e.exec(t, `UPDATE notifications SET created_at = now() - interval '2 minutes' WHERE tenant_id = $1`, b.id)
	request(b, 11)
	if n := e.count(t, queued, b.id); n != 70 {
		t.Fatalf("school B live without a date queued %d more, want 10", n-60)
	}
}

func TestOTPCrossSchool(t *testing.T) {
	e, a, b := newAuthEnv(t)
	ga := e.guardian(t, a, parentMobile, true)
	e.guardian(t, b, parentMobile, true)
	e.do(t, otpCall(a.code, parentMobile))
	e.do(t, otpCall(b.code, parentMobile))
	codeA := e.code(t, a, ga)
	// A's code against B's school (where the same mobile has its own live challenge) is just a wrong code there...
	r := e.do(t, verifyCall(b.code, parentMobile, codeA, installOne))
	codeBMatches := r.code == http.StatusOK // 1 in a million: both schools drew the same code
	if !codeBMatches {
		wantCode(t, "A's code at B", r, http.StatusUnauthorized, "bad_credentials")
	}
	// ...and a session from A only ever sees A.
	r = e.do(t, verifyCall(a.code, parentMobile, codeA, installOne))
	wantCode(t, "A's code at A", r, http.StatusOK, "")
	tok := r.body["token"].(string)
	me := e.do(t, meWithBearer(tok, installOne))
	if sc, _ := me.body["school"].(map[string]any); sc["id"] != float64(a.id) {
		t.Fatalf("me school %s", me.raw)
	}
}

func TestOTPSchoolStatusAndSessions(t *testing.T) {
	e, a, b := newAuthEnv(t)
	ga := e.guardian(t, a, parentMobile, true)
	gb := e.guardian(t, b, parentMobile, true)

	// One session per guardian per install; another install keeps its own; a presented token is ended.
	t1 := e.parentToken(t, a, ga, installOne)
	t2 := e.parentToken(t, a, ga, installTwo)
	wantCode(t, "install one", e.do(t, meWithBearer(t1, installOne)), http.StatusOK, "")
	e.do(t, otpCall(a.code, parentMobile))
	r := e.do(t, call{method: http.MethodPost, path: auth.PathOTPVerify,
		headers: map[string]string{"Authorization": "Bearer " + t2},
		body:    map[string]string{"school_code": a.code, "mobile": parentMobile, "code": e.code(t, a, ga), "install_id": installOne}})
	wantCode(t, "re-sign-in", r, http.StatusOK, "")
	wantCode(t, "replaced on install one", e.do(t, meWithBearer(t1, installOne)), http.StatusUnauthorized, "unauthenticated")
	wantCode(t, "presented token", e.do(t, meWithBearer(t2, installTwo)), http.StatusUnauthorized, "unauthenticated")

	e.exec(t, `UPDATE tenants SET status = 'suspended' WHERE id = $1`, b.id)
	tb := e.parentToken(t, b, gb, installOne)
	if me := e.do(t, meWithBearer(tb, installOne)); me.body["read_only"] != true {
		t.Fatalf("suspended: %s", me.raw)
	}
	e.exec(t, `UPDATE tenants SET status = 'cancelled' WHERE id = $1`, b.id)
	e.do(t, otpCall(b.code, parentMobile))
	code := e.code(t, b, gb)
	wantCode(t, "cancelled, wrong code", e.do(t, verifyCall(b.code, parentMobile, wrongCode(code), installOne)),
		http.StatusUnauthorized, "bad_credentials")
	wantCode(t, "cancelled, right code", e.do(t, verifyCall(b.code, parentMobile, code, installOne)),
		http.StatusLocked, "school_closed")
}

func TestOTPPerIPLimit(t *testing.T) {
	e, a, _ := newAuthEnv(t)
	app, err := db.OpenApp(t.Context(), config.Secret(os.Getenv("TEST_APP_URL")), db.Options{Name: "test-otp-ip", MaxConns: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Close)
	r := chi.NewRouter()
	fake := clock.NewFake(time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC))
	mountAPI(r, auth.Deps{DB: db.Bind(app), OTP: e.keys, Limits: auth.NewLimits(fake), TrustedProxies: testProxy})
	ip := func(addr string) map[string]string { return map[string]string{"X-Forwarded-For": addr} }
	body := map[string]string{"school_code": a.code, "mobile": "9855555555"}
	for i := range 30 {
		if rec := serve(r, http.MethodPost, auth.PathOTP, body, ip("203.0.113.9")); rec.Code != http.StatusAccepted {
			t.Fatalf("request %d: %d", i+1, rec.Code)
		}
	}
	rec := serve(r, http.MethodPost, auth.PathOTP, body, ip("203.0.113.9"))
	if rec.Code != http.StatusTooManyRequests || !strings.Contains(rec.Body.String(), "rate_limited") {
		t.Fatalf("31st: %d %s", rec.Code, rec.Body)
	}
	if rec := serve(r, http.MethodPost, auth.PathOTP, body, ip("203.0.113.10")); rec.Code != http.StatusAccepted {
		t.Fatalf("other IP: %d", rec.Code)
	}
	fake.Advance(10 * time.Minute)
	if rec := serve(r, http.MethodPost, auth.PathOTP, body, ip("203.0.113.9")); rec.Code != http.StatusAccepted {
		t.Fatalf("next window: %d", rec.Code)
	}
}

// ── push tokens ──

func pushCall(tok, install, fcm string) call {
	return call{method: http.MethodPost, path: auth.PathPushToken,
		headers: map[string]string{"Authorization": "Bearer " + tok, auth.InstallHeader: install},
		body:    map[string]string{"platform": "android", "fcm_token": fcm, "app_version": "1.0.0"}}
}

func TestPushToken(t *testing.T) {
	e, a, b := newAuthEnv(t)
	ga := e.guardian(t, a, parentMobile, true)
	gb := e.guardian(t, b, parentMobile, true)
	tokA := e.parentToken(t, a, ga, installOne)
	tokA2 := e.parentToken(t, a, ga, installTwo)
	tokB := e.parentToken(t, b, gb, installOne)
	fcm1, fcm2 := "fcm-token-one-aaaaaaaaaaaa:1", "fcm-token-two-bbbbbbbbbbbb:2"

	wantCode(t, "register", e.do(t, pushCall(tokA, installOne, fcm1)), http.StatusNoContent, "")
	wantCode(t, "rotate", e.do(t, pushCall(tokA, installOne, fcm2)), http.StatusNoContent, "")
	live := `SELECT count(*) FROM push_tokens p JOIN sessions s ON s.tenant_id = p.tenant_id AND s.id = p.session_id
		WHERE p.tenant_id = $1 AND s.install_id = $2 AND p.invalid_at IS NULL`
	if n := e.count(t, live, a.id, installOne); n != 1 {
		t.Fatalf("live tokens after rotation: %d", n)
	}
	// The same FCM token from another session of the school moves there; in the other school it is its own row.
	wantCode(t, "move", e.do(t, pushCall(tokA2, installTwo, fcm2)), http.StatusNoContent, "")
	if n := e.count(t, live, a.id, installOne); n != 0 {
		t.Fatalf("token still on the old session: %d", n)
	}
	wantCode(t, "other school", e.do(t, pushCall(tokB, installOne, fcm2)), http.StatusNoContent, "")
	if n := e.count(t, `SELECT count(*) FROM push_tokens WHERE fcm_token = $1 AND tenant_id IN ($2, $3)`, fcm2, a.id, b.id); n != 2 {
		t.Fatalf("rows for the token across schools: %d", n)
	}
	if n := e.count(t, live, a.id, installTwo); n != 1 {
		t.Fatal("school B's registration touched school A's token")
	}

	// The driver app may register; the portal and anonymous callers may not.
	drv := e.pinSession(t, a, installOne)
	wantCode(t, "driver", e.do(t, pushCall(drv, installOne, "fcm-driver-ccccccccccccc:3")), http.StatusNoContent, "")
	sid, csrf := e.portalSession(t, a)
	wantCode(t, "portal", e.do(t, call{method: http.MethodPost, path: auth.PathPushToken, cookies: []*http.Cookie{sid, csrf},
		headers: map[string]string{"Origin": testOrigin, auth.CSRFHeader: csrf.Value},
		body:    map[string]string{"platform": "ios", "fcm_token": "fcm-portal-dddddddddddd", "app_version": "1.0.0"}}),
		http.StatusForbidden, "forbidden")
	wantCode(t, "anonymous", e.do(t, call{method: http.MethodPost, path: auth.PathPushToken,
		body: map[string]string{"platform": "ios", "fcm_token": "fcm-anon-eeeeeeeeeeeeee", "app_version": "1.0.0"}}),
		http.StatusUnauthorized, "unauthenticated")
}

func TestPushTokenLimit(t *testing.T) {
	e, a, b := newAuthEnv(t)
	app, err := db.OpenApp(t.Context(), config.Secret(os.Getenv("TEST_APP_URL")), db.Options{Name: "test-push-limit", MaxConns: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Close)
	r := chi.NewRouter()
	fake := clock.NewFake(time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC))
	mountAPI(r, auth.Deps{DB: db.Bind(app), OTP: e.keys, Limits: auth.NewLimits(fake), TrustedProxies: testProxy})
	tokA := e.parentToken(t, a, e.guardian(t, a, parentMobile, true), installOne)
	tokB := e.parentToken(t, b, e.guardian(t, b, parentMobile, true), installOne)
	push := func(tok string, i int) int {
		c := pushCall(tok, installOne, fmt.Sprintf("fcm-limit-token-%06d:x", i))
		return serve(r, c.method, c.path, c.body, c.headers).Code
	}
	for i := range 20 {
		if got := push(tokA, i); got != http.StatusNoContent {
			t.Fatalf("registration %d: %d", i+1, got)
		}
	}
	if got := push(tokA, 20); got != http.StatusTooManyRequests {
		t.Fatalf("21st registration: %d, want 429", got)
	}
	if n := e.count(t, `SELECT count(*) FROM push_tokens WHERE tenant_id = $1`, a.id); n != 20 {
		t.Fatalf("school A rows: %d, want 20", n)
	}
	// Per session: school B's session is untouched by A's count; A gets a new allowance next window.
	if got := push(tokB, 100); got != http.StatusNoContent {
		t.Fatalf("other school's session: %d", got)
	}
	fake.Advance(time.Hour)
	if got := push(tokA, 21); got != http.StatusNoContent {
		t.Fatalf("next window: %d", got)
	}
}

// ── worker ──

type fakeSMS struct {
	mu   sync.Mutex
	sent []worker.OTPSMS
	fail error
}

func (f *fakeSMS) SendOTP(_ context.Context, m worker.OTPSMS) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail != nil {
		return f.fail
	}
	f.sent = append(f.sent, m)
	return nil
}

func (f *fakeSMS) to(mobile string) []worker.OTPSMS {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []worker.OTPSMS
	for _, m := range f.sent {
		if m.Mobile == mobile {
			out = append(out, m)
		}
	}
	return out
}

func TestOTPWorker(t *testing.T) {
	e, a, b := newAuthEnv(t)
	// The worker drains every school's outbox; mobiles unique to this run keep other tests' rows apart.
	mA, mB := fmt.Sprintf("96%08d", a.id%100_000_000), fmt.Sprintf("95%08d", b.id%100_000_000)
	ga := e.guardian(t, a, mA, true)
	e.guardian(t, b, mB, true)
	sms := &fakeSMS{}
	w := worker.OTPSender{DB: e.plat, Keys: e.keys, SMS: sms, Log: slog.New(slog.DiscardHandler)}
	status := func(s school) (st string, data string) {
		e.q(t, `SELECT status FROM notifications WHERE tenant_id = $1 AND event = 'otp' ORDER BY id DESC LIMIT 1`, &st, s.id)
		e.q(t, `SELECT data::text FROM notifications WHERE tenant_id = $1 AND event = 'otp' ORDER BY id DESC LIMIT 1`, &data, s.id)
		return st, data
	}

	// Sent: the code that arrives by SMS signs in; the sealed copy is wiped.
	e.do(t, otpCall(a.code, mA))
	if err := w.Drain(t.Context()); err != nil {
		t.Fatal(err)
	}
	got := sms.to(mA)
	if len(got) != 1 || got[0].School != "T" {
		t.Fatalf("sent %+v", got)
	}
	if st, data := status(a); st != "sent" || data != "{}" {
		t.Fatalf("after send: %s %s", st, data)
	}
	wantCode(t, "code from SMS", e.do(t, verifyCall(a.code, mA, got[0].Code, installOne)), http.StatusOK, "")
	_ = ga

	// Stale (older than the code's 5 minutes) and demo schools are skipped, and wiped.
	e.do(t, otpCall(b.code, mB))
	e.exec(t, `UPDATE notifications SET created_at = now() - interval '6 minutes' WHERE tenant_id = $1`, b.id)
	_ = w.Drain(t.Context())
	if st, data := status(b); st != "skipped" || data != "{}" || len(sms.to(mB)) != 0 {
		t.Fatalf("stale: %s %s", st, data)
	}
	e.exec(t, `UPDATE tenants SET is_demo = true WHERE id = $1`, b.id)
	e.do(t, otpCall(b.code, mB))
	_ = w.Drain(t.Context())
	if st, _ := status(b); st != "skipped" || len(sms.to(mB)) != 0 {
		t.Fatalf("demo: %s", st)
	}

	// Gateway failures: retried with a delay, then failed after 3 attempts; the error never carries the code.
	sms.fail = errors.New("msg91: http 500")
	e.do(t, otpCall(a.code, mA))
	for i := range 3 {
		_ = w.Drain(t.Context())
		e.exec(t, `UPDATE notifications SET next_attempt_at = now() WHERE tenant_id = $1 AND status = 'queued'`, a.id)
		if i < 2 {
			if st, _ := status(a); st != "queued" {
				t.Fatalf("attempt %d: %s", i+1, st)
			}
		}
	}
	var attempts int
	e.q(t, `SELECT attempts FROM notifications WHERE tenant_id = $1 AND event = 'otp' ORDER BY id DESC LIMIT 1`, &attempts, a.id)
	if st, data := status(a); st != "failed" || data != "{}" || attempts != 3 {
		t.Fatalf("after 3 failures: %s %s attempts %d", st, data, attempts)
	}
}

func TestOTPPurge(t *testing.T) {
	e, a, b := newAuthEnv(t)
	mA := fmt.Sprintf("94%08d", a.id%100_000_000)
	g := e.guardian(t, a, mA, true)
	e.guardian(t, b, mA, true)
	e.do(t, otpCall(a.code, mA))
	e.do(t, otpCall(b.code, mA))
	w := worker.OTPSender{DB: e.plat, Keys: e.keys, SMS: &fakeSMS{}, Log: slog.New(slog.DiscardHandler)}
	if err := w.Drain(t.Context()); err != nil {
		t.Fatal(err)
	}
	// School A's rows are 25 h old; B's are fresh. A queued row is never purged, however old.
	e.exec(t, `UPDATE otp_challenges SET created_at = now() - interval '25 hours' WHERE tenant_id = $1`, a.id)
	e.exec(t, `UPDATE notifications SET created_at = now() - interval '25 hours' WHERE tenant_id = $1 AND event = 'otp'`, a.id)
	e.exec(t, `INSERT INTO notifications (tenant_id, guardian_id, event, channel, title, body, dedupe_key, created_at, next_attempt_at)
		VALUES ($1, $2, 'otp', 'sms', '', '', 'otp:queued-old', now() - interval '25 hours', now() + interval '1 day')`, a.id, g)
	if err := w.Purge(t.Context()); err != nil {
		t.Fatal(err)
	}
	for what, c := range map[string]struct {
		sql  string
		args []any
		want int
	}{
		"A challenges":    {`SELECT count(*) FROM otp_challenges WHERE tenant_id = $1`, []any{a.id}, 0},
		"A settled SMS":   {`SELECT count(*) FROM notifications WHERE tenant_id = $1 AND event = 'otp' AND status <> 'queued'`, []any{a.id}, 0},
		"A queued SMS":    {`SELECT count(*) FROM notifications WHERE tenant_id = $1 AND event = 'otp' AND status = 'queued'`, []any{a.id}, 1},
		"B challenges":    {`SELECT count(*) FROM otp_challenges WHERE tenant_id = $1`, []any{b.id}, 1},
		"B notifications": {`SELECT count(*) FROM notifications WHERE tenant_id = $1 AND event = 'otp'`, []any{b.id}, 1},
	} {
		if n := e.count(t, c.sql, c.args...); n != c.want {
			t.Errorf("%s: %d, want %d", what, n, c.want)
		}
	}
}

// testProxy stands in for the Nginx hop: httptest's peer address, so X-Forwarded-For names the client.
var testProxy = []netip.Prefix{netip.MustParsePrefix("192.0.2.1/32")}
