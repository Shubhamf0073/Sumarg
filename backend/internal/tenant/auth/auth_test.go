package auth

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/store/tenantq"
)

func TestTokenRoundTrip(t *testing.T) {
	tok, hash, err := newToken(42)
	if err != nil {
		t.Fatal(err)
	}
	id, got, ok := parseToken(tok)
	if !ok || id != 42 || !bytes.Equal(got, hash) {
		t.Fatalf("parseToken(newToken(42)) = %d, %x, %v", id, got, ok)
	}
	tok2, hash2, _ := newToken(42)
	if tok == tok2 || bytes.Equal(hash, hash2) {
		t.Fatal("two tokens are equal")
	}
}

func TestParseTokenRejects(t *testing.T) {
	good, _, _ := newToken(7)
	_, secret, _ := strings.Cut(good, ".")
	for name, s := range map[string]string{
		"empty":           "",
		"no dot":          "7" + secret,
		"no tenant":       "." + secret,
		"zero tenant":     "0." + secret,
		"leading zero":    "07." + secret,
		"negative":        "-7." + secret,
		"plus":            "+7." + secret,
		"tenant overflow": "99999999999999999999." + secret,
		"short secret":    "7." + secret[:42],
		"long secret":     "7." + secret + "A",
		"padded":          "7." + secret[:42] + "=",
		"std alphabet":    "7." + strings.NewReplacer("-", "+", "_", "/").Replace(secret[:41]) + "+/",
		"space":           "7. " + secret[1:],
		"two dots":        "7.1." + secret[2:],
	} {
		if _, _, ok := parseToken(s); ok {
			t.Errorf("%s: parseToken(%q) accepted", name, s)
		}
	}
}

func TestParseTokenNonCanonicalTail(t *testing.T) {
	// 43 base64url chars carry 258 bits; the last 2 must be zero. Strict decoding refuses a second spelling of the
	// same bytes, so one token has exactly one string (and one hash).
	good, _, _ := newToken(7)
	last := good[len(good)-1]
	alt := map[byte]byte{'A': 'B', 'E': 'F', 'I': 'J', 'M': 'N', 'Q': 'R', 'U': 'V', 'Y': 'Z', 'c': 'd', 'g': 'h',
		'k': 'l', 'o': 'p', 's': 't', 'w': 'x', '0': '1', '4': '5', '8': '9'}[last]
	if alt == 0 {
		t.Fatalf("last char %q is not canonical", last)
	}
	if _, _, ok := parseToken(good[:len(good)-1] + string(alt)); ok {
		t.Fatal("non-canonical encoding accepted")
	}
}

func TestCheckPresentation(t *testing.T) {
	install := "install-0123456789abcdef"
	uid, pid := int64(1), int64(9)
	web := tenantq.GetSessionRow{Client: ClientWeb, UserID: &uid}
	app := tenantq.GetSessionRow{Client: ClientDriverApp, UserID: &uid, InstallID: &install}
	cases := []struct {
		name    string
		row     tenantq.GetSessionRow
		bearer  bool
		install string
		ok      bool
	}{
		{"cookie for a portal session", web, false, "", true},
		{"bearer for a portal session", web, true, install, false},
		{"cookie for an app session", app, false, install, false},
		{"bearer with its install", app, true, install, true},
		{"bearer without install header", app, true, "", false},
		{"bearer from another install", app, true, "install-fedcba9876543210", false},
		{"app session with no install", tenantq.GetSessionRow{Client: ClientParentApp, UserID: &uid}, true, "", false},
		{"support session (not built yet)", tenantq.GetSessionRow{Client: ClientWeb, PlatformUserID: &pid}, false, "", false},
	}
	for _, c := range cases {
		r := httptest.NewRequest(http.MethodGet, "/api/me", nil)
		if c.install != "" {
			r.Header.Set(InstallHeader, c.install)
		}
		if err := checkPresentation(r, c.row, c.bearer); (err == nil) != c.ok {
			t.Errorf("%s: err = %v, want ok=%v", c.name, err, c.ok)
		}
	}
}

func TestCheckCSRF(t *testing.T) {
	d := Deps{Origins: []string{"https://sumarg.example.com"}}
	req := func(origin, cookie, header string) *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/api/x", nil)
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		if cookie != "" {
			r.AddCookie(&http.Cookie{Name: CSRFCookie, Value: cookie})
		}
		if header != "" {
			r.Header.Set(CSRFHeader, header)
		}
		return r
	}
	cases := []struct {
		name string
		r    *http.Request
		ok   bool
	}{
		{"same origin, token echoed", req("https://sumarg.example.com", "abc", "abc"), true},
		{"no origin", req("", "abc", "abc"), false},
		{"other origin", req("https://evil.example", "abc", "abc"), false},
		{"origin with path trick", req("https://sumarg.example.com.evil.example", "abc", "abc"), false},
		{"no cookie", req("https://sumarg.example.com", "", "abc"), false},
		{"no header", req("https://sumarg.example.com", "abc", ""), false},
		{"mismatch", req("https://sumarg.example.com", "abc", "abd"), false},
	}
	for _, c := range cases {
		if err := d.checkCSRF(c.r); (err == nil) != c.ok {
			t.Errorf("%s: err = %v, want ok=%v", c.name, err, c.ok)
		}
	}
}

func TestRoleFits(t *testing.T) {
	for role, k := range map[string]secretKind{"owner": password, "admin": password, "transport_manager": password,
		"driver": pin, "attendant": pin} {
		other := pin
		if k == pin {
			other = password
		}
		if !roleFits(k, role) || roleFits(other, role) {
			t.Errorf("roleFits(%s) wrong", role)
		}
	}
	if roleFits(password, "parent") || roleFits(pin, "") {
		t.Error("unknown role fits")
	}
}
