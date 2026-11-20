package obs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/config"
)

func decode(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("log line is not JSON: %v\n%s", err, buf)
	}
	return m
}

func TestLoggerAddsServiceAndLevel(t *testing.T) {
	var buf bytes.Buffer
	log := NewLogger(&buf, config.Base{ServiceName: "core", LogLevel: slog.LevelWarn, LogFormat: "json"})
	log.Info("dropped")
	if buf.Len() != 0 {
		t.Fatalf("info logged at warn level: %s", buf.String())
	}
	log.Warn("kept")
	if m := decode(t, &buf); m["service"] != "core" || m["msg"] != "kept" {
		t.Fatalf("got %v", m)
	}
}

func TestTextFormat(t *testing.T) {
	var buf bytes.Buffer
	New(&buf, "core", slog.LevelInfo, "text").Info("hi", "password", "hunter2")
	if s := buf.String(); !strings.Contains(s, "service=core") || strings.Contains(s, "hunter2") {
		t.Fatalf("got %q", s)
	}
}

func TestContextAttrs(t *testing.T) {
	var buf bytes.Buffer
	log := New(&buf, "core", slog.LevelInfo, "json")
	parent := With(context.Background(), slog.String("request_id", "r1"))
	a := With(parent, slog.String("tenant_id", "A"))
	b := With(parent, slog.String("tenant_id", "B"))

	log.InfoContext(a, "x")
	if m := decode(t, &buf); m["request_id"] != "r1" || m["tenant_id"] != "A" {
		t.Fatalf("got %v", m)
	}
	buf.Reset()
	// b was derived from the same parent after a: it must not see a's tenant.
	log.InfoContext(b, "x")
	if m := decode(t, &buf); m["tenant_id"] != "B" {
		t.Fatalf("contexts share attrs: %v", m)
	}
	buf.Reset()
	log.Info("no context")
	if m := decode(t, &buf); m["request_id"] != nil {
		t.Fatalf("request_id without a context: %v", m)
	}
}

type token struct{ v string }

// LogValue makes token a LogValuer that turns into a group once resolved.
func (t token) LogValue() slog.Value {
	return slog.GroupValue(slog.String("kind", "bearer"), slog.String("access_token", t.v))
}

func TestRedaction(t *testing.T) {
	const s = "s3cr3t-value"
	var buf bytes.Buffer
	log := New(&buf, "core", slog.LevelInfo, "json")

	log.With("Authorization", "Bearer "+s, slog.Group("db", slog.String("password", s))). // security:allow test proves a fake secret is redacted

												InfoContext(With(context.Background(), slog.String("session_token", s)), "login",
			"password", s,
			"API-Key", s,
			"otp", s,
			"user", map[string]string{"name": "ok"},
			slog.Group("req", slog.String("path", "/api/x"), slog.Group("headers", slog.String("Cookie", s))),
			"upstream", token{s},
			"wrapped", config.Secret(s),
		)
	out := buf.String()
	if strings.Contains(out, s) {
		t.Fatalf("secret leaked into the log:\n%s", out)
	}
	for _, want := range []string{`"path":"/api/x"`, `"kind":"bearer"`, `"Authorization":"[redacted]"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in\n%s", want, out)
		}
	}
}

func TestIsSecretKey(t *testing.T) {
	for key, want := range map[string]bool{
		"password": true, "db.password": true, "apiKey": true, "api_key": true, "X-Api-Key": true,
		"refresh_token": true, "Set-Cookie": true, "private_key": true, "SENTRY_DSN": true, "otp_code": true,
		"migrator_pw": true, "db_pass": true, "auth_header": true, "bearer": true, "jwt": true, "migratorPW": true,
		"hmac": true, "upi_pin": true, "salt": true, "r2_access_key_id": true,
		"phone": true, "consignee_mobile": true, "email": true, "aadhaar": true, "pan": true, "delivery_address": true,
		"request_id": false, "tenant_id": false, "route": false, "status": false, "company": false, "shipping": false,
		"idempotency_key": false, "key_id": false, "compass": false, "addr": false, "lr_no": false,
	} {
		if got := IsSecretKey(key); got != want {
			t.Errorf("IsSecretKey(%q) = %v, want %v", key, got, want)
		}
	}
}

type creds struct {
	User  string
	Token string
}

// Values slog would otherwise write unchecked with encoding/json: maps are checked key by key, structs fail closed.
func TestRedactionInsideContainers(t *testing.T) {
	const s = "s3cr3t-value"
	var buf bytes.Buffer
	log := New(&buf, "core", slog.LevelInfo, "json")
	log.Info("upstream",
		"headers", http.Header{"Authorization": {"Bearer " + s}, "Content-Type": {"application/json"}},
		"form", url.Values{"otp": {s}, "lang": {"en"}},
		"body", map[string]any{"user": "a", "nested": map[string]string{"password": s}, "list": []any{map[string]any{"token": s}}},
		"creds", creds{"a", s},
		"credsp", &creds{"a", s},
		"many", []creds{{"a", s}},
		"raw", []byte(s),
		"ids", []string{"x", "y"},
		"err", errors.New("boom"),
		"amount", 12,
	)
	out := buf.String()
	if strings.Contains(out, s) {
		t.Fatalf("secret leaked into the log:\n%s", out)
	}
	for _, want := range []string{`"Content-Type":["application/json"]`, `"lang":["en"]`, `"user":"a"`,
		`"ids":["x","y"]`, `"err":"boom"`, `"amount":12`, `"creds":"` + Unloggable} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in\n%s", want, out)
		}
	}
}

func TestPhoneIsMasked(t *testing.T) {
	var buf bytes.Buffer
	New(&buf, "core", slog.LevelInfo, "json").Info("sent", "consignee_phone", "+91 98765 43210", "phone", "123")
	m := decode(t, &buf)
	if m["consignee_phone"] != "******3210" || m["phone"] != Redacted {
		t.Fatalf("got %v", m)
	}
}

type wrapper struct{ inner token }

// LogValue returns a value that is not a group, to check that it is still inspected.
func (w wrapper) LogValue() slog.Value { return slog.AnyValue(map[string]string{"api_key": w.inner.v}) }

type stringerStruct struct{ user, pass string }

func (s stringerStruct) String() string { return s.user + ":" + s.pass }

// What a LogValuer or Stringer produces is checked too (automated review on PR #6).
func TestRedactionOfLogValuersAndStringers(t *testing.T) {
	const s = "s3cr3t-value"
	dbURL, _ := url.Parse("postgres://previa_core:" + s + "@db:5432/postgres")
	var buf bytes.Buffer
	New(&buf, "core", slog.LevelInfo, "json").Info("x",
		"in_map", map[string]any{"up": token{s}, "w": wrapper{token{s}}},
		"w", wrapper{token{s}},
		"db", dbURL,
		"db_value", *dbURL,
		"conn", stringerStruct{"u", s},
		"conns", map[string]any{"c": &stringerStruct{"u", s}},
		"at", map[string]any{"t": time.Date(2027, 3, 31, 0, 0, 0, 0, time.UTC)},
	)
	out := buf.String()
	if strings.Contains(out, s) {
		t.Fatalf("secret leaked into the log:\n%s", out)
	}
	for _, want := range []string{`"kind":"bearer"`, `previa_core:xxxxx@db:5432`, `"conn":"` + Unloggable,
		`"t":"2027-03-31T00:00:00Z"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in\n%s", want, out)
		}
	}
}
