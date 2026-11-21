package config

import (
	"fmt"
	"maps"
	"strings"
	"testing"

	base "github.com/Shubhamf0073/previa-school-fleet/backend/platform/config"
)

const (
	appURL      = "postgres://sf_app:app-pw-1@db:5432/schoolfleet?sslmode=disable"
	platformURL = "postgres://sf_platform:plat-pw-2@db:5432/schoolfleet?sslmode=disable"
	goodKey     = "9f2c0a47d1b8e35a6c7f0d92e4b1a85c3d6e7f8091a2b3c4d5e6f708192a3b4c"
	smsKey      = "sms-api-key-3"
	mapsSrvKey  = "maps-server-key-4"
)

// production is a complete, valid production environment. Each test case changes one thing.
func production() map[string]string {
	return map[string]string{
		"APP_ENV": "production", "SVC_NAME": "schoolfleet",
		"DATABASE_URL": appURL, "PLATFORM_DATABASE_URL": platformURL, "SECRETS_KEY": goodKey,
		"MQTT_HOST": "devices.example.com", "MQTT_CERT": "/certs/fullchain.pem", "MQTT_KEY": "/certs/privkey.pem",
		"FCM_PROJECT_ID": "sumarg-prod", "FCM_SERVICE_ACCOUNT_FILE": "/data/secrets/fcm.json",
		"SMS_PROVIDER": "msg91", "SMS_API_KEY": smsKey, "SMS_SENDER_ID": "PREVIA", "SMS_DLT_PE_ID": "1101",
		"SMS_DLT_TEMPLATE_OTP":    "1107",
		"GOOGLE_MAPS_BROWSER_KEY": "browser-key", "GOOGLE_MAPS_SERVER_KEY": mapsSrvKey,
		"APP_DOMAIN": "sumarg.example.com", "TRUSTED_PROXIES": "172.30.231.1/32",
	}
}

// local is what `make run` provides: stubs and the dev-only plain MQTT port.
func local() map[string]string {
	return map[string]string{
		"APP_ENV": "local", "SVC_NAME": "schoolfleet",
		"DATABASE_URL":          "postgres://sf_app:dev@localhost:5434/schoolfleet?sslmode=disable",
		"PLATFORM_DATABASE_URL": "postgres://sf_platform:dev@localhost:5434/schoolfleet?sslmode=disable",
		"SECRETS_KEY":           strings.Repeat("0", 64), "MQTT_PLAIN_PORT": "1883", "SMS_PROVIDER": "stub",
	}
}

func with(env map[string]string, kv ...string) map[string]string {
	out := maps.Clone(env)
	for i := 0; i < len(kv); i += 2 {
		if kv[i+1] == "-" {
			delete(out, kv[i])
		} else {
			out[kv[i]] = kv[i+1]
		}
	}
	return out
}

func TestValidEnvironments(t *testing.T) {
	for name, env := range map[string]map[string]string{"production": production(), "local": local()} {
		t.Run(name, func(t *testing.T) {
			cfg, err := base.LoadFrom[Config](env)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.MQTTPort != 8883 || cfg.PlatformDBMaxConns != 3 || cfg.MQTTMaxUnits != 2000 {
				t.Fatalf("defaults not applied: %+v", cfg)
			}
		})
	}
}

func TestInvalidConfigNamesTheVariable(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want string // the variable (or phrase) the error must name
	}{
		{"missing platform url", with(production(), "PLATFORM_DATABASE_URL", "-"), "PLATFORM_DATABASE_URL"},
		{"platform url not postgres", with(production(), "PLATFORM_DATABASE_URL", "mysql://x:y@db/z"), "PLATFORM_DATABASE_URL must be a postgres"},
		{"platform url without tls off-box", with(production(), "PLATFORM_DATABASE_URL",
			"postgres://sf_platform:p@db.example.com:5432/x?sslmode=disable"), "PLATFORM_DATABASE_URL must set sslmode"},
		{"same url for both roles", with(production(), "PLATFORM_DATABASE_URL", appURL), "must differ from DATABASE_URL"},
		{"platform pool too big", with(production(), "PLATFORM_DB_MAX_CONNS", "50"), "PLATFORM_DB_MAX_CONNS"},
		{"admin session zero", with(production(), "ADMIN_SESSION_HOURS", "0"), "ADMIN_SESSION_HOURS"},
		{"admin session over a day", with(production(), "ADMIN_SESSION_HOURS", "48"), "ADMIN_SESSION_HOURS"},
		{"missing secrets key", with(production(), "SECRETS_KEY", "-"), "SECRETS_KEY"},
		{"short secrets key", with(production(), "SECRETS_KEY", "abcd"), "SECRETS_KEY must be 64 hex"},
		{"non-hex secrets key", with(production(), "SECRETS_KEY", strings.Repeat("zz", 32)), "SECRETS_KEY must be 64 hex"},
		{"placeholder secrets key", with(production(), "SECRETS_KEY", "__GENERATE_KEY__"), "SECRETS_KEY must be 64 hex"},
		{"zero secrets key in production", with(production(), "SECRETS_KEY", strings.Repeat("0", 64)), "all zeros"},
		{"zero secrets key in staging", with(production(), "APP_ENV", "staging", "SECRETS_KEY", strings.Repeat("0", 64)), "all zeros"},
		{"missing app domain", with(production(), "APP_DOMAIN", "-"), "APP_DOMAIN is required"},
		{"app domain with scheme", with(production(), "APP_DOMAIN", "https://sumarg.example.com"), "APP_DOMAIN must be a bare host"},
		{"app domain with port", with(production(), "APP_DOMAIN", "sumarg.example.com:443"), "APP_DOMAIN must be a bare host"},
		{"no trusted proxies in production", with(production(), "TRUSTED_PROXIES", "-"), "TRUSTED_PROXIES is required"},
		{"trusted proxy not a prefix", with(production(), "TRUSTED_PROXIES", "nginx"), "TRUSTED_PROXIES must be a comma-separated"},
		{"trusted proxy too wide", with(production(), "TRUSTED_PROXIES", "0.0.0.0/0"), "TRUSTED_PROXIES prefixes must be narrow"},
		{"bad mqtt port", with(production(), "MQTT_PORT", "70000"), "MQTT_PORT"},
		{"plain mqtt in production", with(production(), "MQTT_PLAIN_PORT", "1883"), "MQTT_PLAIN_PORT is a local-development port"},
		{"plain port equals tls port", with(local(), "MQTT_PLAIN_PORT", "8883"), "MQTT_PLAIN_PORT must differ"},
		{"mqtt units zero", with(production(), "MQTT_MAX_UNITS", "0"), "MQTT_MAX_UNITS"},
		{"no mqtt host in production", with(production(), "MQTT_HOST", "-"), "MQTT_HOST is required"},
		{"no mqtt cert in production", with(production(), "MQTT_CERT", "-"), "MQTT_CERT is required"},
		{"no mqtt key in production", with(production(), "MQTT_KEY", "-"), "MQTT_KEY is required"},
		{"missing sms provider", with(production(), "SMS_PROVIDER", "-"), "SMS_PROVIDER"},
		{"unknown sms provider", with(production(), "SMS_PROVIDER", "twilio"), "SMS_PROVIDER must be"},
		{"stub sms in production", with(production(), "SMS_PROVIDER", "stub"), "SMS_PROVIDER=stub"},
		{"sms without api key", with(production(), "SMS_API_KEY", "-"), "SMS_API_KEY is required"},
		{"sms without dlt sender", with(production(), "SMS_SENDER_ID", "-"), "SMS_SENDER_ID is required"},
		{"sms without dlt entity", with(production(), "SMS_DLT_PE_ID", "-"), "SMS_DLT_PE_ID is required"},
		{"sms without otp template", with(production(), "SMS_DLT_TEMPLATE_OTP", "-"), "SMS_DLT_TEMPLATE_OTP is required"},
		{"no fcm project in production", with(production(), "FCM_PROJECT_ID", "-"), "FCM_PROJECT_ID is required"},
		{"no fcm key file in production", with(production(), "FCM_SERVICE_ACCOUNT_FILE", "-"), "FCM_SERVICE_ACCOUNT_FILE is required"},
		{"no browser maps key in production", with(production(), "GOOGLE_MAPS_BROWSER_KEY", "-"), "GOOGLE_MAPS_BROWSER_KEY is required"},
		{"no server maps key in production", with(production(), "GOOGLE_MAPS_SERVER_KEY", "-"), "GOOGLE_MAPS_SERVER_KEY is required"},
		{"base rules still apply", with(production(), "APP_ENV", "prod"), "APP_ENV must be"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := base.LoadFrom[Config](c.env)
			if err == nil {
				t.Fatal("config was accepted")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("error does not mention %q:\n%v", c.want, err)
			}
		})
	}
}

// Errors name variables, never values: every credential in the environment must stay out of every error.
func TestErrorsNeverContainSecrets(t *testing.T) {
	secrets := []string{"app-pw-1", "plat-pw-2", goodKey, smsKey, mapsSrvKey}
	// Break many things at once so every validator produces an error.
	env := with(production(),
		"PLATFORM_DATABASE_URL", "mysql://sf_platform:plat-pw-2@db:5432/x",
		"DATABASE_URL", "postgres://sf_app:app-pw-1@db.example.com:5432/x?sslmode=disable",
		"SECRETS_KEY", goodKey[:60], "SMS_PROVIDER", "twilio", "MQTT_PORT", "0")
	_, err := base.LoadFrom[Config](env)
	if err == nil {
		t.Fatal("config was accepted")
	}
	for _, s := range secrets {
		if strings.Contains(err.Error(), s) {
			t.Fatalf("error leaks %q:\n%v", s, err)
		}
	}
}

// A dumped config (a debug log, a panic message, JSON) must not show a credential.
func TestPrintedConfigRedactsSecrets(t *testing.T) {
	cfg, err := base.LoadFrom[Config](production())
	if err != nil {
		t.Fatal(err)
	}
	for _, verb := range []string{"%v", "%+v", "%#v"} {
		out := fmt.Sprintf(verb, cfg)
		for _, s := range []string{"app-pw-1", "plat-pw-2", goodKey, smsKey, mapsSrvKey} {
			if strings.Contains(out, s) {
				t.Fatalf("%s of Config leaks %q:\n%s", verb, s, out)
			}
		}
	}
}
