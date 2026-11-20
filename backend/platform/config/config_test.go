package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"strings"
	"testing"
)

const password = "hunter2-sekret"

func baseEnv() map[string]string {
	return map[string]string{
		"APP_ENV":      "local",
		"SVC_NAME":     "schoolfleet",
		"DATABASE_URL": "postgres://sf_app:" + password + "@pg.example.net:5432/schoolfleet",
	}
}

func with(changes map[string]string) map[string]string {
	e := baseEnv()
	maps.Copy(e, changes)
	for k, v := range changes {
		if v == "<unset>" {
			delete(e, k)
		}
	}
	return e
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := LoadFrom[Base](baseEnv())
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.AppEnv != EnvLocal || cfg.ServiceName != "schoolfleet" || cfg.DBMaxConns != 3 ||
		cfg.LogLevel != slog.LevelInfo || cfg.LogFormat != "json" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if !strings.Contains(cfg.DatabaseURL.Reveal(), password) {
		t.Fatal("Reveal must return the real value")
	}
}

func TestLoadRejects(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr string // a substring the error must contain
	}{
		{"missing APP_ENV", with(map[string]string{"APP_ENV": "<unset>"}), "APP_ENV"},
		{"empty APP_ENV", with(map[string]string{"APP_ENV": ""}), "APP_ENV"},
		{"unknown APP_ENV", with(map[string]string{"APP_ENV": "prod"}), "APP_ENV"},
		{"missing SVC_NAME", with(map[string]string{"SVC_NAME": "<unset>"}), "SVC_NAME"},
		{"unknown SVC_NAME", with(map[string]string{"SVC_NAME": "billing"}), "SVC_NAME"},
		{"missing DATABASE_URL", with(map[string]string{"DATABASE_URL": "<unset>"}), "DATABASE_URL"},
		{"DB_MAX_CONNS zero", with(map[string]string{"DB_MAX_CONNS": "0"}), "DB_MAX_CONNS"},
		{"DB_MAX_CONNS too big", with(map[string]string{"DB_MAX_CONNS": "11"}), "DB_MAX_CONNS"},
		{"DB_MAX_CONNS not a number", with(map[string]string{"DB_MAX_CONNS": "three"}), "DB_MAX_CONNS"},
		{"bad LOG_LEVEL", with(map[string]string{"LOG_LEVEL": "loud"}), "LOG_LEVEL"},
		{"bad LOG_FORMAT", with(map[string]string{"LOG_FORMAT": "xml"}), "LOG_FORMAT"},
		{"not a postgres URL", with(map[string]string{"DATABASE_URL": "mysql://u:" + password + "@h/db"}), "DATABASE_URL"},
		{"unparseable URL", with(map[string]string{"DATABASE_URL": "postgres://u:" + password + "@h:badport/db"}), "DATABASE_URL"},
		{"staging without sslmode", with(map[string]string{"APP_ENV": "staging", "SVC_NAME": "migrate"}), "sslmode"},
		{"staging with sslmode=disable", with(map[string]string{"APP_ENV": "staging", "SVC_NAME": "migrate",
			"DATABASE_URL": "postgres://u:" + password + "@db.example:5432/postgres?sslmode=disable"}), "sslmode"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadFrom[Base](tt.env)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error %q should mention %q", err, tt.wantErr)
			}
			if strings.Contains(err.Error(), password) {
				t.Fatalf("error leaks the password: %q", err)
			}
		})
	}
}

func TestStagingWithTLS(t *testing.T) {
	for _, mode := range []string{"require", "verify-ca", "verify-full"} {
		env := with(map[string]string{"APP_ENV": "staging",
			"DATABASE_URL": "postgres://u:" + password + "@db.example:5432/postgres?sslmode=" + mode})
		if _, err := LoadFrom[Base](env); err != nil {
			t.Errorf("sslmode=%s should be accepted: %v", mode, err)
		}
	}
}

// A service's own struct embeds Base: its fields are loaded and Base's checks still run.
type serviceConfig struct {
	Base
	GotenbergURL string `env:"GOTENBERG_URL,required,notEmpty"`
}

func TestEmbeddedBase(t *testing.T) {
	cfg, err := LoadFrom[serviceConfig](with(map[string]string{"GOTENBERG_URL": "http://gotenberg:3000"}))
	if err != nil || cfg.GotenbergURL != "http://gotenberg:3000" || cfg.ServiceName != "schoolfleet" {
		t.Fatalf("got %+v, %v", cfg, err)
	}
	if _, err := LoadFrom[serviceConfig](with(map[string]string{"GOTENBERG_URL": "x", "APP_ENV": "prod"})); err == nil {
		t.Fatal("Base.Validate must run for structs that embed Base")
	}
	if _, err := LoadFrom[serviceConfig](baseEnv()); err == nil {
		t.Fatal("the service's own required field must be enforced")
	}
}

func TestSecretNeverPrinted(t *testing.T) {
	cfg, err := LoadFrom[Base](with(map[string]string{"SENTRY_DSN": "https://" + password + "@sentry.example/1"}))
	if err != nil {
		t.Fatal(err)
	}
	outputs := map[string]string{}
	for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%q", "%x", "%X", "%d", "%10.3s"} {
		outputs["secret "+verb] = fmt.Sprintf(verb, cfg.DatabaseURL)
		outputs["struct "+verb] = fmt.Sprintf(verb, cfg)
		outputs["pointer "+verb] = fmt.Sprintf(verb, &cfg)
	}
	outputs["String()"] = cfg.DatabaseURL.String()
	outputs["Sprint"] = fmt.Sprint(cfg.DatabaseURL, cfg.SentryDSN)
	outputs["Errorf"] = fmt.Errorf("connecting with %v: %w", cfg, fmt.Errorf("boom")).Error()

	js, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	outputs["json"] = string(js)

	var buf bytes.Buffer
	for _, h := range []slog.Handler{slog.NewJSONHandler(&buf, nil), slog.NewTextHandler(&buf, nil)} {
		slog.New(h).Info("start", "db", cfg.DatabaseURL, "cfg", cfg, "dsn", cfg.SentryDSN)
	}
	outputs["slog"] = buf.String()

	for name, out := range outputs {
		if strings.Contains(out, password) || strings.Contains(out, fmt.Sprintf("%x", password)) {
			t.Errorf("%s leaks the secret: %s", name, out)
		}
	}
	if !strings.Contains(outputs["json"], redacted) || !strings.Contains(outputs["slog"], redacted) {
		t.Errorf("expected %q in output", redacted)
	}
}
