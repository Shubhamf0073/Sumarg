//go:build integration

package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"testing"

	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/admin/adminauth"
	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/admin/schools"
	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/tenant/auth"
)

const newOwnerMobile = "9833333333"

// adminAs enrols a fresh platform user with role and returns their session cookies.
func (e *authEnv) adminAs(t *testing.T, role string) adminCookies {
	t.Helper()
	_, token := e.seedAdmin(t, role)
	return signedIn(t, e.enrol(t, token, newPasskey()))
}

// adminCall is an /api/admin request with the session, and for writes Origin + the CSRF header.
func adminCall(c adminCookies, method, path string, body any) call {
	h := map[string]string{}
	if method != http.MethodGet {
		h["Origin"], h[adminauth.CSRFHeader] = testOrigin, c.csrf.Value
	}
	return call{method: method, path: path, body: body, headers: h, cookies: []*http.Cookie{c.sid, c.csrf}}
}

func wizardBody(code string) map[string]any {
	return map[string]any{
		"school": map[string]any{"code": code, "legal_name": "Trust " + code, "display_name": "School " + code,
			"short_name": "S " + code, "board": "CBSE", "address": "", "city": "Ahmedabad", "state_code": "24",
			"contact_name": "Principal", "contact_mobile": "9844444444", "notes": ""},
		"campuses": []map[string]any{
			{"code": "MAIN", "name": "Main", "address": "", "lat": 23.03, "lon": 72.58},
			{"code": "EAST", "name": "East", "address": "", "lat": 23.05, "lon": 72.62, "geofence_m": 200},
		},
		"settings": map[string]any{"approach_minutes": 7},
		"owner":    map[string]any{"name": "Owner", "mobile": newOwnerMobile},
		"package": map[string]any{"period_start": "2026-06-01", "students_committed": 420,
			"price_per_student_paise": 100000},
	}
}

func randomMobile(t *testing.T) string {
	t.Helper()
	n, err := rand.Int(rand.Reader, big.NewInt(100_000_000))
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("97%08d", n.Int64())
}

func newCode() string { return "w-" + strings.ToLower(rand.Text()[:10]) }

func schoolPath(id int64, rest string) string {
	return strings.Replace(schools.PathSchool, "{id}", fmt.Sprint(id), 1) + rest
}

func linkToken(t *testing.T, r resp) string {
	t.Helper()
	l, _ := r.body["owner_link"].(map[string]any)
	if l == nil {
		l = r.body
	}
	url, _ := l["url"].(string)
	_, tok, ok := strings.Cut(url, "/set-password#")
	if !ok || !strings.HasPrefix(url, testOrigin) {
		t.Fatalf("link url %q", url)
	}
	return tok
}

func setPasswordCall(token, pass string) call {
	return call{method: http.MethodPost, path: auth.PathSetPassword, headers: map[string]string{"Origin": testOrigin},
		body: map[string]string{"token": token, "password": pass}}
}

func id(r resp) int64 { v, _ := r.body["id"].(float64); return int64(v) }

// Two schools through the wizard: each gets its own rows; the owner sets a password from the link and signs in.
func TestSchoolWizardTwoSchools(t *testing.T) {
	e, _, _ := newAuthEnv(t)
	sup := e.adminAs(t, adminauth.RoleSupport)
	codeA, codeB := newCode(), newCode()

	a := e.do(t, adminCall(sup, http.MethodPost, schools.PathSchools, wizardBody(codeA)))
	b := e.do(t, adminCall(sup, http.MethodPost, schools.PathSchools, wizardBody(codeB)))
	for _, r := range []resp{a, b} {
		if r.code != http.StatusCreated || r.body["onboarding_stage"] != "setup" || r.body["period_end"] != "2027-05-31" {
			t.Fatalf("create: %d %s", r.code, r.raw)
		}
	}
	idA, idB := id(a), id(b)

	// Each school's rows carry its own tenant id; settings got the wizard's value and the defaults.
	var n int
	e.q(t, `SELECT count(*) FROM campuses WHERE tenant_id = $1`, &n, idA)
	if n != 2 {
		t.Fatalf("A campuses = %d", n)
	}
	var approach, speed int
	e.q(t, `SELECT approach_minutes FROM tenant_settings WHERE tenant_id = $1`, &approach, idB)
	e.q(t, `SELECT speed_limit_kmph FROM tenant_settings WHERE tenant_id = $1`, &speed, idB)
	if approach != 7 || speed != 40 {
		t.Fatalf("B settings approach=%d speed=%d", approach, speed)
	}
	e.q(t, `SELECT count(*) FROM subscriptions WHERE tenant_id = $1 AND students_committed = 420
		AND period_end = '2027-05-31' AND price_per_student_paise = 100000`, &n, idA)
	if n != 1 {
		t.Fatalf("A subscription rows = %d", n)
	}
	e.q(t, `SELECT count(*) FROM platform_audit_log WHERE tenant_id = $1 AND action = 'school.created'`, &n, idB)
	if n != 1 {
		t.Fatalf("B audit rows = %d", n)
	}

	// The owner can't sign in until the link sets a password.
	if r := e.do(t, portalLoginCall(codeA, newOwnerMobile, "anything-at-all")); r.code != http.StatusUnauthorized {
		t.Fatalf("login before set-password: %d", r.code)
	}
	tokA, tokB := linkToken(t, a), linkToken(t, b)

	// B's secret under A's tenant id finds nothing (tenant only from the token, looked up under RLS).
	_, secretB, _ := strings.Cut(tokB, ".")
	if r := e.do(t, setPasswordCall(fmt.Sprintf("%d.%s", idA, secretB), "a long password")); r.code != http.StatusGone {
		t.Fatalf("B's link with A's id: %d %s", r.code, r.raw)
	}
	if r := e.do(t, setPasswordCall(tokA, "short")); r.code != http.StatusBadRequest {
		t.Fatalf("short password: %d", r.code)
	}
	bad := setPasswordCall(tokA, "a long password")
	bad.headers["Origin"] = "https://evil.test"
	if r := e.do(t, bad); r.code != http.StatusForbidden {
		t.Fatalf("foreign origin: %d", r.code)
	}

	if r := e.do(t, setPasswordCall(tokA, "a long password")); r.code != http.StatusNoContent {
		t.Fatalf("set password: %d %s", r.code, r.raw)
	}
	if r := e.do(t, setPasswordCall(tokA, "another password")); r.code != http.StatusGone {
		t.Fatalf("link reused: %d", r.code)
	}
	if r := e.do(t, portalLoginCall(codeA, newOwnerMobile, "a long password")); r.code != http.StatusOK ||
		r.body["user"].(map[string]any)["must_change_secret"] != false {
		t.Fatalf("owner login: %d %s", r.code, r.raw)
	}
	// B is untouched: its owner still can't sign in with A's password, and its link still works.
	if r := e.do(t, portalLoginCall(codeB, newOwnerMobile, "a long password")); r.code != http.StatusUnauthorized {
		t.Fatalf("B login with A's password: %d", r.code)
	}
	e.q(t, `SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND action = 'user.password_set'`, &n, idA)
	if n != 1 {
		t.Fatalf("A tenant audit rows = %d", n)
	}
	e.q(t, `SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND action = 'user.password_set'`, &n, idB)
	if n != 0 {
		t.Fatalf("B tenant audit rows = %d", n)
	}

	// The school page shows the package and that A's owner has set a password.
	g := e.do(t, adminCall(sup, http.MethodGet, schoolPath(idA, ""), nil))
	if g.code != http.StatusOK {
		t.Fatalf("get: %d %s", g.code, g.raw)
	}
	logins := g.body["logins"].([]any)
	if len(logins) != 1 || logins[0].(map[string]any)["password_set"] != true || g.body["package"] == nil {
		t.Fatalf("get body: %s", g.raw)
	}
}

// A save that fails leaves nothing behind; a taken code is 409.
func TestSchoolWizardRefusals(t *testing.T) {
	e, _, _ := newAuthEnv(t)
	sup := e.adminAs(t, adminauth.RoleSuperadmin)
	code := newCode()
	if r := e.do(t, adminCall(sup, http.MethodPost, schools.PathSchools, wizardBody(code))); r.code != http.StatusCreated {
		t.Fatalf("create: %d %s", r.code, r.raw)
	}
	if r := e.do(t, adminCall(sup, http.MethodPost, schools.PathSchools, wizardBody(code))); r.code != http.StatusConflict ||
		r.body["code"] != "code_taken" {
		t.Fatalf("taken code: %d %s", r.code, r.raw)
	}

	dup := wizardBody(newCode())
	dup["campuses"] = []map[string]any{{"code": "MAIN", "name": "A", "address": "", "lat": 1, "lon": 1},
		{"code": "MAIN", "name": "B", "address": "", "lat": 1, "lon": 1}}
	if r := e.do(t, adminCall(sup, http.MethodPost, schools.PathSchools, dup)); r.code != http.StatusUnprocessableEntity {
		t.Fatalf("duplicate campus: %d %s", r.code, r.raw)
	}

	var agent int64
	e.q(t, `INSERT INTO agents (name, mobile, is_active) VALUES ('Old agent', $1, false) RETURNING id`, &agent,
		randomMobile(t))
	noAgent := wizardBody(newCode())
	noAgent["package"].(map[string]any)["agent_id"] = agent
	if r := e.do(t, adminCall(sup, http.MethodPost, schools.PathSchools, noAgent)); r.code != http.StatusUnprocessableEntity ||
		r.body["code"] != "agent_invalid" {
		t.Fatalf("inactive agent: %d %s", r.code, r.raw)
	}
	var n int
	e.q(t, `SELECT count(*) FROM tenants WHERE code = $1`, &n, noAgent["school"].(map[string]any)["code"])
	if n != 0 {
		t.Fatalf("refused save left %d tenants", n)
	}
}

// Moving to live sets go_live_on to today (IST) only if it is empty; the stage the admin saw must still hold.
func TestSchoolStageLive(t *testing.T) {
	e, _, _ := newAuthEnv(t)
	sup := e.adminAs(t, adminauth.RoleSupport)
	plain := e.do(t, adminCall(sup, http.MethodPost, schools.PathSchools, wizardBody(newCode())))
	plannedBody := wizardBody(newCode())
	plannedBody["school"].(map[string]any)["go_live_on"] = "2026-12-01"
	planned := e.do(t, adminCall(sup, http.MethodPost, schools.PathSchools, plannedBody))
	for _, r := range []resp{plain, planned} {
		if r.code != http.StatusCreated {
			t.Fatalf("create: %d %s", r.code, r.raw)
		}
	}
	move := func(id int64, from, to string) resp {
		return e.do(t, adminCall(sup, http.MethodPatch, schoolPath(id, "/onboarding"),
			map[string]any{"from_stage": from, "stage": to}))
	}

	if r := move(id(plain), "setup", "data"); r.code != http.StatusOK || r.body["go_live_on"] != nil {
		t.Fatalf("setup→data: %d %s", r.code, r.raw)
	}
	if r := move(id(plain), "setup", "survey"); r.code != http.StatusConflict {
		t.Fatalf("stale from_stage: %d %s", r.code, r.raw)
	}
	if r := move(999999999, "setup", "data"); r.code != http.StatusNotFound {
		t.Fatalf("unknown school: %d", r.code)
	}
	var today string
	e.q(t, `SELECT ((now() AT TIME ZONE 'Asia/Kolkata')::date)::text`, &today)
	if r := move(id(plain), "data", "live"); r.code != http.StatusOK || r.body["go_live_on"] != today {
		t.Fatalf("→live with no date: %d %s (today IST %s)", r.code, r.raw, today)
	}
	if r := move(id(planned), "setup", "live"); r.code != http.StatusOK || r.body["go_live_on"] != "2026-12-01" {
		t.Fatalf("→live keeps the set date: %d %s", r.code, r.raw)
	}
	var n int
	e.q(t, `SELECT count(*) FROM platform_audit_log WHERE tenant_id = $1 AND action = 'school.stage_changed'`, &n,
		id(plain))
	if n != 2 {
		t.Fatalf("stage audit rows = %d, want 2", n)
	}
}

// Roles: support/superadmin write; billing and installers read only; signed out is 401.
func TestSchoolRoles(t *testing.T) {
	e, _, _ := newAuthEnv(t)
	sup := e.adminAs(t, adminauth.RoleSupport)
	made := e.do(t, adminCall(sup, http.MethodPost, schools.PathSchools, wizardBody(newCode())))
	if made.code != http.StatusCreated {
		t.Fatalf("create: %d %s", made.code, made.raw)
	}
	for _, role := range []string{adminauth.RoleBilling, adminauth.RoleInstaller} {
		c := e.adminAs(t, role)
		if r := e.do(t, adminCall(c, http.MethodPost, schools.PathSchools, wizardBody(newCode()))); r.code != http.StatusForbidden {
			t.Fatalf("%s create: %d", role, r.code)
		}
		if r := e.do(t, adminCall(c, http.MethodPatch, schoolPath(id(made), "/onboarding"),
			map[string]any{"from_stage": "setup", "stage": "data"})); r.code != http.StatusForbidden {
			t.Fatalf("%s stage: %d", role, r.code)
		}
		if r := e.do(t, adminCall(c, http.MethodGet, schools.PathSchools, nil)); r.code != http.StatusOK {
			t.Fatalf("%s list: %d %s", role, r.code, r.raw)
		}
		if r := e.do(t, adminCall(c, http.MethodGet, schoolPath(id(made), ""), nil)); r.code != http.StatusOK {
			t.Fatalf("%s get: %d", role, r.code)
		}
	}
	if r := e.do(t, call{method: http.MethodGet, path: schools.PathSchools}); r.code != http.StatusUnauthorized {
		t.Fatalf("signed out list: %d", r.code)
	}
	// A school session is not an admin session.
	if r := e.do(t, call{method: http.MethodGet, path: schools.PathSchools,
		headers: map[string]string{"Authorization": "Bearer 1.x"}}); r.code != http.StatusUnauthorized {
		t.Fatalf("bearer on admin: %d", r.code)
	}
}

// A new link spends the old one; another school's user id under this school's id is 404.
func TestSchoolPasswordLinkReissue(t *testing.T) {
	e, _, _ := newAuthEnv(t)
	sup := e.adminAs(t, adminauth.RoleSupport)
	a := e.do(t, adminCall(sup, http.MethodPost, schools.PathSchools, wizardBody(newCode())))
	b := e.do(t, adminCall(sup, http.MethodPost, schools.PathSchools, wizardBody(newCode())))
	ownerB := int64(b.body["owner_link"].(map[string]any)["user_id"].(float64))
	ownerA := int64(a.body["owner_link"].(map[string]any)["user_id"].(float64))

	if r := e.do(t, adminCall(sup, http.MethodPost, schoolPath(id(a), fmt.Sprintf("/users/%d/password-link", ownerB)),
		nil)); r.code != http.StatusNotFound {
		t.Fatalf("B's user under A: %d %s", r.code, r.raw)
	}
	re := e.do(t, adminCall(sup, http.MethodPost, schoolPath(id(a), fmt.Sprintf("/users/%d/password-link", ownerA)), nil))
	if re.code != http.StatusCreated {
		t.Fatalf("reissue: %d %s", re.code, re.raw)
	}
	if r := e.do(t, setPasswordCall(linkToken(t, a), "a long password")); r.code != http.StatusGone {
		t.Fatalf("old link after reissue: %d", r.code)
	}
	if r := e.do(t, setPasswordCall(linkToken(t, re), "a long password")); r.code != http.StatusNoContent {
		t.Fatalf("new link: %d %s", r.code, r.raw)
	}
	// B's link was not spent by A's reissue.
	if r := e.do(t, setPasswordCall(linkToken(t, b), "a long password")); r.code != http.StatusNoContent {
		t.Fatalf("B's link: %d %s", r.code, r.raw)
	}
}

// Using a link ends the user's sessions and clears the failure counters; an expired link is refused.
func TestSetPasswordRevokesAndExpires(t *testing.T) {
	e, _, _ := newAuthEnv(t)
	sup := e.adminAs(t, adminauth.RoleSupport)
	code := newCode()
	a := e.do(t, adminCall(sup, http.MethodPost, schools.PathSchools, wizardBody(code)))
	if a.code != http.StatusCreated {
		t.Fatalf("create: %d %s", a.code, a.raw)
	}
	owner := int64(a.body["owner_link"].(map[string]any)["user_id"].(float64))
	if r := e.do(t, setPasswordCall(linkToken(t, a), "first password")); r.code != http.StatusNoContent {
		t.Fatalf("set: %d %s", r.code, r.raw)
	}
	r := e.do(t, portalLoginCall(code, newOwnerMobile, "first password"))
	if r.code != http.StatusOK {
		t.Fatalf("login: %d %s", r.code, r.raw)
	}
	sid := cookie(t, r, auth.SessionCookie)
	e.exec(t, `UPDATE users SET failed_logins = 7, fail_window_count = 7, fail_window_start = now() WHERE id = $1`, owner)

	re := e.do(t, adminCall(sup, http.MethodPost, schoolPath(id(a), fmt.Sprintf("/users/%d/password-link", owner)), nil))
	if re.code != http.StatusCreated {
		t.Fatalf("reissue: %d %s", re.code, re.raw)
	}
	if r := e.do(t, setPasswordCall(linkToken(t, re), "second password")); r.code != http.StatusNoContent {
		t.Fatalf("set again: %d %s", r.code, r.raw)
	}
	if me := e.do(t, call{method: http.MethodGet, path: auth.PathMe, cookies: []*http.Cookie{sid}}); me.code != http.StatusUnauthorized {
		t.Fatalf("session after password set: %d, want 401", me.code)
	}
	var failed, window int
	e.q(t, `SELECT failed_logins FROM users WHERE id = $1`, &failed, owner)
	e.q(t, `SELECT fail_window_count FROM users WHERE id = $1`, &window, owner)
	if failed != 0 || window != 0 {
		t.Fatalf("counters after set: failed=%d window=%d", failed, window)
	}

	ex := e.do(t, adminCall(sup, http.MethodPost, schoolPath(id(a), fmt.Sprintf("/users/%d/password-link", owner)), nil))
	e.exec(t, `UPDATE user_secret_links SET expires_at = now() - interval '1 second'
		WHERE tenant_id = $1 AND used_at IS NULL`, id(a))
	if r := e.do(t, setPasswordCall(linkToken(t, ex), "third password")); r.code != http.StatusGone {
		t.Fatalf("expired link: %d %s", r.code, r.raw)
	}
}
