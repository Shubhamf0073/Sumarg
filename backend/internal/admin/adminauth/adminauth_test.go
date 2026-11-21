package adminauth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/clock"
)

func newFakeStore() (*Ceremonies, *clock.Fake) {
	c := clock.NewFake(time.Date(2026, 11, 21, 8, 0, 0, 0, time.UTC))
	return NewCeremonies(c), c
}

func TestCeremonySingleUse(t *testing.T) {
	s, _ := newFakeStore()
	id, err := s.put(ceremony{kind: kindLogin, session: webauthn.SessionData{Challenge: "c"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.take(id, kindLogin); !ok {
		t.Fatal("first take failed")
	}
	if _, ok := s.take(id, kindLogin); ok {
		t.Fatal("second take succeeded: a ceremony must finish once")
	}
}

func TestCeremonyWrongKindIsSpent(t *testing.T) {
	s, _ := newFakeStore()
	id, _ := s.put(ceremony{kind: kindEnrol})
	if _, ok := s.take(id, kindLogin); ok {
		t.Fatal("an enrolment ceremony finished a sign-in")
	}
	if _, ok := s.take(id, kindEnrol); ok {
		t.Fatal("a ceremony presented to the wrong step must not be usable afterwards")
	}
}

func TestCeremonyExpires(t *testing.T) {
	s, c := newFakeStore()
	id, _ := s.put(ceremony{kind: kindLogin})
	c.Advance(ceremonyTTL)
	if _, ok := s.take(id, kindLogin); ok {
		t.Fatal("expired ceremony was accepted")
	}
}

func TestCeremonyStoreFullRefusesThenRecovers(t *testing.T) {
	s, c := newFakeStore()
	s.max = 3
	for range 3 {
		if _, err := s.put(ceremony{kind: kindLogin}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.put(ceremony{kind: kindLogin}); !errors.Is(err, errCeremoniesFull) {
		t.Fatalf("full store: err = %v, want errCeremoniesFull", err)
	}
	c.Advance(ceremonyTTL)
	if _, err := s.put(ceremony{kind: kindLogin}); err != nil {
		t.Fatalf("after expiry the store should make room: %v", err)
	}
}

func TestHashTokenStrict(t *testing.T) {
	tok, hash, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	got, ok := hashToken(tok)
	if !ok || string(got) != string(hash) {
		t.Fatal("round trip failed")
	}
	for _, bad := range []string{"", tok[:42], tok + "A", strings.Repeat("=", 43), "1." + tok[2:], tok[:42] + "+"} {
		if _, ok := hashToken(bad); ok {
			t.Fatalf("hashToken(%q) accepted", bad)
		}
	}
}

func TestRequireRoles(t *testing.T) {
	ctx := func(role string) context.Context { return withPrincipal(context.Background(), &Principal{Role: role}) }
	for _, tc := range []struct {
		role  string
		allow []string
		want  error
	}{
		{RoleSuperadmin, nil, nil},
		{RoleSuperadmin, []string{RoleBilling}, nil},
		{RoleBilling, []string{RoleBilling}, nil},
		{RoleInstaller, []string{RoleBilling}, ErrForbidden},
		{RoleSupport, nil, ErrForbidden},
	} {
		if _, err := Require(ctx(tc.role), tc.allow...); !errors.Is(err, tc.want) {
			t.Errorf("Require(%s, %v) = %v, want %v", tc.role, tc.allow, err, tc.want)
		}
	}
	if _, err := Require(context.Background()); !errors.Is(err, ErrUnauthenticated) {
		t.Errorf("signed out: %v, want ErrUnauthenticated", err)
	}
}

// Sign-in steps pass through to their handlers; every other admin path needs a session (here: none → 401, before
// any database work, so Deps needs no pool).
func TestMiddlewareGates(t *testing.T) {
	reached := false
	h := Middleware(Deps{Origins: []string{"https://portal.test"}})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		reached = true
	}))
	for path, open := range map[string]bool{
		PathLoginBegin: true, PathLoginFinish: true, PathEnrolBegin: true, PathEnrolFinish: true,
		PathMe: false, PathLogout: false, "/api/admin/schools": false,
	} {
		reached = false
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, nil))
		if reached != open {
			t.Errorf("%s: handler reached = %v, want %v", path, reached, open)
		}
		if !open && rec.Code != http.StatusUnauthorized {
			t.Errorf("%s signed out: %d, want 401", path, rec.Code)
		}
	}
}

// A write with an admin cookie but no CSRF token or a foreign Origin is refused before the session is looked up.
func TestCookieWritesNeedOriginAndCSRF(t *testing.T) {
	d := Deps{Origins: []string{"https://portal.test"}}
	tok, _, _ := NewToken()
	for name, set := range map[string]func(*http.Request){
		"no origin":      func(*http.Request) {},
		"foreign origin": func(r *http.Request) { r.Header.Set("Origin", "https://evil.test") },
		"no csrf":        func(r *http.Request) { r.Header.Set("Origin", "https://portal.test") },
		"csrf mismatch": func(r *http.Request) {
			r.Header.Set("Origin", "https://portal.test")
			r.AddCookie(&http.Cookie{Name: CSRFCookie, Value: "a"})
			r.Header.Set(CSRFHeader, "b")
		},
	} {
		r := httptest.NewRequest(http.MethodPost, PathLogout, nil)
		r.AddCookie(&http.Cookie{Name: SessionCookie, Value: tok})
		set(r)
		if _, err := d.Authenticate(r); !errors.Is(err, ErrCSRF) {
			t.Errorf("%s: %v, want ErrCSRF", name, err)
		}
	}
}
