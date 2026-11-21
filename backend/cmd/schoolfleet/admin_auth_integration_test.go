//go:build integration

package main

import (
	"crypto/rand"
	"net/http"
	"strings"
	"testing"

	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/admin/adminauth"
	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/admin/adminauth/passkeytest"
	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/tenant/auth"
)

// seedAdmin makes a platform user with a live 15-minute enrolment link and returns the link's token.
func (e *authEnv) seedAdmin(t *testing.T, role string) (userID int64, token string) {
	t.Helper()
	email := "admin-" + strings.ToLower(rand.Text()[:10]) + "@previa.test"
	e.q(t, `INSERT INTO platform_users (name, email, role) VALUES ('Asha', $1, $2) RETURNING id`, &userID, email, role)
	return userID, e.issueEnrolment(t, userID, "15 minutes")
}

func (e *authEnv) issueEnrolment(t *testing.T, userID int64, life string) string {
	t.Helper()
	token, hash, err := adminauth.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	e.exec(t, `INSERT INTO platform_enrolments (platform_user_id, token_hash, expires_at)
		VALUES ($1, $2, now() + $3::interval)`, userID, hash, life)
	return token
}

func originHdr() map[string]string { return map[string]string{"Origin": testOrigin} }

func cookie(t *testing.T, r resp, name string) *http.Cookie {
	t.Helper()
	for _, c := range r.cookies {
		if c.Name == name && c.MaxAge >= 0 {
			return c
		}
	}
	t.Fatalf("no %s cookie in %v (%d %s)", name, r.cookies, r.code, r.raw)
	return nil
}

type adminCookies struct{ sid, csrf *http.Cookie }

// enrol runs the enrolment ceremony for token with pk and returns the finish response.
func (e *authEnv) enrol(t *testing.T, token string, pk *passkeytest.Authenticator, presented ...*http.Cookie) resp {
	t.Helper()
	b := e.do(t, call{method: http.MethodPost, path: adminauth.PathEnrolBegin, headers: originHdr(),
		body: map[string]string{"token": token}})
	if b.code != http.StatusOK {
		t.Fatalf("enrol begin: %d %s", b.code, b.raw)
	}
	cred, err := pk.Create(b.body)
	if err != nil {
		t.Fatal(err)
	}
	return e.do(t, call{method: http.MethodPost, path: adminauth.PathEnrolFinish, headers: originHdr(), body: cred,
		cookies: append([]*http.Cookie{cookie(t, b, adminauth.CeremonyCookie)}, presented...)})
}

// login runs the sign-in ceremony with pk and returns the begin response's ceremony cookie, the assertion sent,
// and the finish response.
func (e *authEnv) login(t *testing.T, pk *passkeytest.Authenticator, presented ...*http.Cookie) (*http.Cookie, map[string]any, resp) {
	t.Helper()
	b := e.do(t, call{method: http.MethodPost, path: adminauth.PathLoginBegin, headers: originHdr()})
	if b.code != http.StatusOK {
		t.Fatalf("login begin: %d %s", b.code, b.raw)
	}
	cer := cookie(t, b, adminauth.CeremonyCookie)
	assertion, err := pk.Get(b.body)
	if err != nil {
		t.Fatal(err)
	}
	f := e.do(t, call{method: http.MethodPost, path: adminauth.PathLoginFinish, headers: originHdr(),
		body: assertion, cookies: append([]*http.Cookie{cer}, presented...)})
	return cer, assertion, f
}

func signedIn(t *testing.T, r resp) adminCookies {
	t.Helper()
	if r.code != http.StatusOK {
		t.Fatalf("sign-in: %d %s", r.code, r.raw)
	}
	return adminCookies{sid: cookie(t, r, adminauth.SessionCookie), csrf: cookie(t, r, adminauth.CSRFCookie)}
}

func (e *authEnv) adminMe(t *testing.T, c adminCookies) resp {
	t.Helper()
	return e.do(t, call{method: http.MethodGet, path: adminauth.PathMe, cookies: []*http.Cookie{c.sid}})
}

func newPasskey() *passkeytest.Authenticator {
	return &passkeytest.Authenticator{RPID: testRPID, Origin: testOrigin}
}

// The whole path: one-time link → passkey → session; sign in again with the passkey; sign out.
func TestAdminPasskeyEnrolSignInSignOut(t *testing.T) {
	e, _, _ := newAuthEnv(t)
	userID, token := e.seedAdmin(t, adminauth.RoleSuperadmin)
	pk := newPasskey()

	first := signedIn(t, e.enrol(t, token, pk))
	if me := e.adminMe(t, first); me.code != http.StatusOK || me.body["role"] != "superadmin" ||
		int64(me.body["id"].(float64)) != userID {
		t.Fatalf("me after enrolment: %d %s", me.code, me.raw)
	}
	var n int
	e.q(t, `SELECT count(*) FROM platform_audit_log WHERE platform_user_id = $1 AND action = 'passkey.enrolled'`,
		&n, userID)
	if n != 1 {
		t.Fatalf("enrolment audit rows = %d, want 1", n)
	}

	// The link works once.
	again := e.do(t, call{method: http.MethodPost, path: adminauth.PathEnrolBegin, headers: originHdr(),
		body: map[string]string{"token": token}})
	if again.code != http.StatusGone {
		t.Fatalf("used link: %d %s, want 410", again.code, again.raw)
	}

	// Sign in with the passkey, presenting the first session: it is revoked (no fixation).
	cer, assertion, f := e.login(t, pk, first.sid)
	second := signedIn(t, f)
	if me := e.adminMe(t, first); me.code != http.StatusUnauthorized {
		t.Fatalf("presented session after sign-in: %d, want 401", me.code)
	}
	if me := e.adminMe(t, second); me.code != http.StatusOK {
		t.Fatalf("me after sign-in: %d %s", me.code, me.raw)
	}

	// A ceremony finishes once: replaying the same assertion with the same ceremony cookie fails.
	replay := e.do(t, call{method: http.MethodPost, path: adminauth.PathLoginFinish, headers: originHdr(),
		body: assertion, cookies: []*http.Cookie{cer}})
	if replay.code != http.StatusUnauthorized {
		t.Fatalf("replayed assertion: %d %s, want 401", replay.code, replay.raw)
	}

	// Sign-out is a cookie write: it needs Origin and the CSRF token.
	noCSRF := e.do(t, call{method: http.MethodPost, path: adminauth.PathLogout, headers: originHdr(),
		cookies: []*http.Cookie{second.sid}})
	if noCSRF.code != http.StatusForbidden {
		t.Fatalf("logout without CSRF: %d, want 403", noCSRF.code)
	}
	out := e.do(t, call{method: http.MethodPost, path: adminauth.PathLogout,
		headers: map[string]string{"Origin": testOrigin, adminauth.CSRFHeader: second.csrf.Value},
		cookies: []*http.Cookie{second.sid, second.csrf}})
	if out.code != http.StatusNoContent {
		t.Fatalf("logout: %d %s", out.code, out.raw)
	}
	if me := e.adminMe(t, second); me.code != http.StatusUnauthorized {
		t.Fatalf("me after logout: %d, want 401", me.code)
	}
}

// Admin and school sessions never open each other's routes, for either school.
func TestAdminAndSchoolSessionsAreSeparate(t *testing.T) {
	e, a, b := newAuthEnv(t)
	_, token := e.seedAdmin(t, adminauth.RoleSuperadmin)
	admin := signedIn(t, e.enrol(t, token, newPasskey()))

	if r := e.do(t, call{method: http.MethodGet, path: auth.PathMe, cookies: []*http.Cookie{admin.sid}}); r.code != http.StatusUnauthorized {
		t.Fatalf("admin cookie on /api/me: %d, want 401", r.code)
	}
	for _, s := range []school{a, b} {
		sid, _ := e.portalSession(t, s)
		if r := e.do(t, call{method: http.MethodGet, path: adminauth.PathMe, cookies: []*http.Cookie{sid}}); r.code != http.StatusUnauthorized {
			t.Fatalf("school %s portal cookie on /api/admin/me: %d, want 401", s.code, r.code)
		}
		// A school token as a cookie named like the admin's is just an unknown admin token.
		forged := &http.Cookie{Name: adminauth.SessionCookie, Value: sid.Value}
		if r := e.do(t, call{method: http.MethodGet, path: adminauth.PathMe, cookies: []*http.Cookie{forged}}); r.code != http.StatusUnauthorized {
			t.Fatalf("school %s token as admin cookie: %d, want 401", s.code, r.code)
		}
	}
	// Bearer is never accepted on /api/admin.
	r := e.do(t, call{method: http.MethodGet, path: adminauth.PathMe, cookies: []*http.Cookie{admin.sid},
		headers: map[string]string{"Authorization": "Bearer " + admin.sid.Value}})
	if r.code != http.StatusUnauthorized {
		t.Fatalf("bearer on /api/admin/me: %d, want 401", r.code)
	}
	// Signed out, an admin route answers 401 (every /api/admin route but sign-in is closed by default).
	if r := e.do(t, call{method: http.MethodGet, path: adminauth.PathMe}); r.code != http.StatusUnauthorized {
		t.Fatalf("admin route signed out: %d, want 401", r.code)
	}
}

// A session ends with its user, its passkey or its expiry.
func TestAdminSessionEnds(t *testing.T) {
	e, _, _ := newAuthEnv(t)
	for name, end := range map[string]string{
		"user deactivated": `UPDATE platform_users SET is_active = false WHERE id = $1`,
		"passkey revoked":  `UPDATE platform_passkeys SET revoked_at = now() WHERE platform_user_id = $1`,
		"expired":          `UPDATE platform_sessions SET expires_at = now() - interval '1 second' WHERE platform_user_id = $1`,
	} {
		t.Run(name, func(t *testing.T) {
			userID, token := e.seedAdmin(t, adminauth.RoleSupport)
			c := signedIn(t, e.enrol(t, token, newPasskey()))
			if me := e.adminMe(t, c); me.code != http.StatusOK {
				t.Fatalf("me: %d %s", me.code, me.raw)
			}
			e.exec(t, end, userID)
			if me := e.adminMe(t, c); me.code != http.StatusUnauthorized {
				t.Fatalf("me after %s: %d, want 401", name, me.code)
			}
		})
	}
}

func TestAdminPasskeyRefusals(t *testing.T) {
	e, _, _ := newAuthEnv(t)

	t.Run("no origin", func(t *testing.T) {
		r := e.do(t, call{method: http.MethodPost, path: adminauth.PathLoginBegin})
		if r.code != http.StatusForbidden {
			t.Fatalf("begin without Origin: %d, want 403", r.code)
		}
	})

	t.Run("expired link", func(t *testing.T) {
		userID, _ := e.seedAdmin(t, adminauth.RoleSuperadmin)
		token := e.issueEnrolment(t, userID, "-1 second")
		r := e.do(t, call{method: http.MethodPost, path: adminauth.PathEnrolBegin, headers: originHdr(),
			body: map[string]string{"token": token}})
		if r.code != http.StatusGone {
			t.Fatalf("expired link: %d, want 410", r.code)
		}
	})

	t.Run("credential made for another origin, link still usable", func(t *testing.T) {
		_, token := e.seedAdmin(t, adminauth.RoleSuperadmin)
		evil := &passkeytest.Authenticator{RPID: testRPID, Origin: "https://evil.test"}
		if r := e.enrol(t, token, evil); r.code != http.StatusUnauthorized {
			t.Fatalf("foreign-origin registration: %d %s, want 401", r.code, r.raw)
		}
		signedIn(t, e.enrol(t, token, newPasskey())) // the failed attempt rolled back: the link was not used
	})

	t.Run("revoked passkey", func(t *testing.T) {
		userID, token := e.seedAdmin(t, adminauth.RoleSuperadmin)
		pk := newPasskey()
		signedIn(t, e.enrol(t, token, pk))
		e.exec(t, `UPDATE platform_passkeys SET revoked_at = now() WHERE platform_user_id = $1`, userID)
		if _, _, r := e.login(t, pk); r.code != http.StatusUnauthorized {
			t.Fatalf("revoked passkey sign-in: %d, want 401", r.code)
		}
	})

	t.Run("sign count went backwards", func(t *testing.T) {
		_, token := e.seedAdmin(t, adminauth.RoleSuperadmin)
		pk := newPasskey()
		signedIn(t, e.enrol(t, token, pk))
		_, _, ok := e.login(t, pk) // count 1 stored
		signedIn(t, ok)
		one := uint32(1)
		pk.Counter = &one
		if _, _, r := e.login(t, pk); r.code != http.StatusUnauthorized {
			t.Fatalf("cloned authenticator sign-in: %d, want 401", r.code)
		}
	})

	t.Run("deactivated user's link", func(t *testing.T) {
		userID, token := e.seedAdmin(t, adminauth.RoleSuperadmin)
		e.exec(t, `UPDATE platform_users SET is_active = false WHERE id = $1`, userID)
		r := e.do(t, call{method: http.MethodPost, path: adminauth.PathEnrolBegin, headers: originHdr(),
			body: map[string]string{"token": token}})
		if r.code != http.StatusGone {
			t.Fatalf("deactivated user's link: %d, want 410", r.code)
		}
	})
}
