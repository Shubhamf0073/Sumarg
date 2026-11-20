// Package config is Sumarg's configuration: the shared settings every service has (platform/config.Base) plus the
// ones only Sumarg needs. The values come from the compose environment (infra/.env.example) and are checked once,
// at start-up, so a missing or unsafe setting stops the service with a message that names the variable.
//
// Rules for this file:
//   - Every credential is a config.Secret, so it prints as "[redacted]" everywhere (logs, errors, JSON).
//   - Validate errors name the variable and never include its value.
//   - Outside APP_ENV=local, anything that would silently degrade the product (stub SMS, no push, no maps, a
//     placeholder encryption key, dev-only plain MQTT) is a start-up error rather than a warning.
package config

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/netip"
	"regexp"
	"slices"
	"strings"

	base "github.com/Shubhamf0073/previa-school-fleet/backend/platform/config"
)

// SMS providers (SMS_PROVIDER). "stub" logs the OTP instead of sending it and exists for local development only.
const (
	SMSMsg91   = "msg91"
	SMS2Factor = "2factor"
	SMSStub    = "stub"
)

// secretsKeyBytes is the length of SECRETS_KEY once decoded: 32 bytes (AES-256 / the box key), written as 64 hex
// characters.
const secretsKeyBytes = 32

// Config is the whole configuration of the schoolfleet binary.
type Config struct {
	base.Base

	// Databases. DATABASE_URL (in Base) is sf_app, the RLS role every tenant request runs as. The platform pool is
	// sf_platform (BYPASSRLS): only /api/admin, the worker and ingest use it (invariant 2).
	PlatformDatabaseURL base.Secret `env:"PLATFORM_DATABASE_URL,required,notEmpty"`
	PlatformDBMaxConns  int32       `env:"PLATFORM_DB_MAX_CONNS" envDefault:"3"`

	// AppDomain is the portal's host (sumarg.previasoftwares.com). Cookie-authenticated writes and the portal login
	// must carry Origin https://<AppDomain> (CSRF, docs/05 §10). Locally the Vite dev origin is accepted too.
	AppDomain string `env:"APP_DOMAIN"`

	// TrustedProxies are the addresses (CIDR) of the reverse-proxy hop in front of the API: Nginx, as seen from
	// inside the container (the compose network's gateway, infra/docker-compose.yml). X-Forwarded-For is believed
	// only on a connection from one of these; otherwise the connection's own address is the client's. Required
	// outside APP_ENV=local: without it every client would share the gateway address and its per-IP limits.
	TrustedProxies []string `env:"TRUSTED_PROXIES" envSeparator:","`

	// SecretsKey encrypts secrets stored in the database (64 hex characters = 32 bytes).
	SecretsKey base.Secret `env:"SECRETS_KEY,required,notEmpty"`

	// MQTT broker for the bus units (docs/13). The plain port is for the local simulator only.
	MQTTHost      string `env:"MQTT_HOST"`
	MQTTPort      int    `env:"MQTT_PORT" envDefault:"8883"`
	MQTTPlainPort int    `env:"MQTT_PLAIN_PORT" envDefault:"0"` // 0 = off
	MQTTCert      string `env:"MQTT_CERT"`
	MQTTKey       string `env:"MQTT_KEY"`
	MQTTMaxUnits  int    `env:"MQTT_MAX_UNITS" envDefault:"2000"`

	// Push notifications (FCM HTTP v1). Empty project = pushes are logged, not sent (local only).
	FCMProjectID          string `env:"FCM_PROJECT_ID"`
	FCMServiceAccountFile string `env:"FCM_SERVICE_ACCOUNT_FILE"`

	// SMS: parent OTP and fallback alerts. India requires a DLT-registered sender and templates (docs/15 §5).
	SMSProvider           string      `env:"SMS_PROVIDER,required,notEmpty"`
	SMSAPIKey             base.Secret `env:"SMS_API_KEY"`
	SMSSenderID           string      `env:"SMS_SENDER_ID"`
	SMSDLTPEID            string      `env:"SMS_DLT_PE_ID"`
	SMSDLTTemplateOTP     string      `env:"SMS_DLT_TEMPLATE_OTP"`
	SMSDLTTemplateInvite  string      `env:"SMS_DLT_TEMPLATE_INVITE"`
	SMSDLTTemplateBoarded string      `env:"SMS_DLT_TEMPLATE_BOARDED"`
	SMSDLTTemplateDropped string      `env:"SMS_DLT_TEMPLATE_DROPPED"`
	SMSDLTTemplateDelay   string      `env:"SMS_DLT_TEMPLATE_DELAY"`

	// Google Maps. The browser key ships in the portal (referrer-restricted); the server key is used only by the
	// Routes API call on route save (invariant 9).
	MapsBrowserKey string      `env:"GOOGLE_MAPS_BROWSER_KEY"`
	MapsServerKey  base.Secret `env:"GOOGLE_MAPS_SERVER_KEY"`
}

// Validate checks what struct tags can't. Go note: Config embeds Base, which has its own Validate; defining
// Validate here replaces it for Config, so it must call the Base one first.
func (c Config) Validate() error {
	errs := []error{c.Base.Validate()}
	local := c.AppEnv == base.EnvLocal

	if c.PlatformDBMaxConns < 1 || c.PlatformDBMaxConns > 10 {
		errs = append(errs, fmt.Errorf("PLATFORM_DB_MAX_CONNS must be between 1 and 10, got %d", c.PlatformDBMaxConns))
	}
	if err := base.CheckDatabaseURL("PLATFORM_DATABASE_URL", c.PlatformDatabaseURL, c.AppEnv); err != nil {
		errs = append(errs, err)
	}
	// The two URLs must not be the same credentials: the app role is subject to RLS, the platform role is not.
	if c.PlatformDatabaseURL == c.DatabaseURL {
		errs = append(errs, errors.New("PLATFORM_DATABASE_URL must differ from DATABASE_URL (sf_platform vs sf_app)"))
	}

	errs = append(errs, c.validateSecretsKey(local), c.validateMQTT(local), c.validateSMS(local),
		c.validatePush(local), c.validateMaps(local), c.validateAppDomain(local), c.validateTrustedProxies(local))
	return errors.Join(errs...)
}

// TrustedProxyPrefixes parses TRUSTED_PROXIES (already checked by Validate).
func (c Config) TrustedProxyPrefixes() []netip.Prefix {
	out := make([]netip.Prefix, 0, len(c.TrustedProxies))
	for _, s := range c.TrustedProxies {
		if p, err := netip.ParsePrefix(strings.TrimSpace(s)); err == nil {
			out = append(out, p.Masked())
		}
	}
	return out
}

func (c Config) validateTrustedProxies(local bool) error {
	if len(c.TrustedProxies) == 0 {
		if local {
			return nil
		}
		return fmt.Errorf("TRUSTED_PROXIES is required when APP_ENV=%s (the Nginx hop as seen by the API, e.g. 172.30.231.1/32)", c.AppEnv)
	}
	for _, s := range c.TrustedProxies {
		p, err := netip.ParsePrefix(strings.TrimSpace(s))
		if err != nil {
			return errors.New("TRUSTED_PROXIES must be a comma-separated list of CIDR prefixes such as 172.30.231.1/32")
		}
		// A wide prefix would let a client on the internet pick its own address for the per-IP limits.
		if (p.Addr().Is4() && p.Bits() < 16) || (p.Addr().Is6() && p.Bits() < 64) {
			return errors.New("TRUSTED_PROXIES prefixes must be narrow (at least /16 for IPv4, /64 for IPv6)")
		}
	}
	return nil
}

// PortalOrigins are the Origin values accepted on portal login and cookie-authenticated writes.
func (c Config) PortalOrigins() []string {
	var out []string
	if c.AppDomain != "" {
		out = append(out, "https://"+c.AppDomain)
	}
	if c.AppEnv == base.EnvLocal {
		out = append(out, "http://localhost:5173", "http://127.0.0.1:5173") // make web (Vite)
	}
	return out
}

func (c Config) validateAppDomain(local bool) error {
	if c.AppDomain == "" {
		if local {
			return nil
		}
		return fmt.Errorf("APP_DOMAIN is required when APP_ENV=%s", c.AppEnv)
	}
	if !appDomainRE.MatchString(c.AppDomain) {
		return errors.New("APP_DOMAIN must be a bare host name such as sumarg.example.com (no scheme, port or path)")
	}
	return nil
}

var appDomainRE = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$`)

func (c Config) validateSecretsKey(local bool) error {
	raw := c.SecretsKey.Reveal()
	key, err := hex.DecodeString(raw)
	if err != nil || len(key) != secretsKeyBytes {
		// Never echo the value, or even how it was wrong.
		return fmt.Errorf("SECRETS_KEY must be %d hex characters (generate one with: openssl rand -hex %d)",
			secretsKeyBytes*2, secretsKeyBytes)
	}
	// The placeholder in infra/.env.example ("__GENERATE_KEY__") is not hex, so the check above already rejects it.
	if !local && allZero(key) {
		return fmt.Errorf("SECRETS_KEY is all zeros, which is only allowed when APP_ENV=local (APP_ENV=%s)", c.AppEnv)
	}
	return nil
}

func allZero(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}

func (c Config) validateMQTT(local bool) error {
	var errs []error
	if c.MQTTPort < 1 || c.MQTTPort > 65535 {
		errs = append(errs, fmt.Errorf("MQTT_PORT must be between 1 and 65535, got %d", c.MQTTPort))
	}
	if c.MQTTPlainPort < 0 || c.MQTTPlainPort > 65535 {
		errs = append(errs, fmt.Errorf("MQTT_PLAIN_PORT must be 0 (off) or between 1 and 65535, got %d", c.MQTTPlainPort))
	}
	if c.MQTTPlainPort != 0 && c.MQTTPlainPort == c.MQTTPort {
		errs = append(errs, errors.New("MQTT_PLAIN_PORT must differ from MQTT_PORT"))
	}
	if c.MQTTMaxUnits < 1 || c.MQTTMaxUnits > 10000 {
		errs = append(errs, fmt.Errorf("MQTT_MAX_UNITS must be between 1 and 10000, got %d", c.MQTTMaxUnits))
	}
	if !local {
		// Units authenticate with a secret over MQTT; without TLS it would cross the internet in the clear.
		if c.MQTTPlainPort != 0 {
			errs = append(errs, fmt.Errorf("MQTT_PLAIN_PORT is a local-development port and must be 0 when APP_ENV=%s", c.AppEnv))
		}
		errs = append(errs, requireAll(c.AppEnv, map[string]string{
			"MQTT_HOST": c.MQTTHost, "MQTT_CERT": c.MQTTCert, "MQTT_KEY": c.MQTTKey,
		})...)
	}
	return errors.Join(errs...)
}

func (c Config) validateSMS(local bool) error {
	switch c.SMSProvider {
	case SMSMsg91, SMS2Factor:
		// Only the OTP template is required: login depends on it. The boarded/dropped/delay/invite templates serve
		// the optional SMS fallback (tenant_settings.sms_fallback, off by default), so a school that never turns
		// it on needs no DLT approval for them; the task that sends those messages must check its own template.
		return errors.Join(requireAll(c.AppEnv, map[string]string{
			"SMS_API_KEY": c.SMSAPIKey.Reveal(), "SMS_SENDER_ID": c.SMSSenderID, "SMS_DLT_PE_ID": c.SMSDLTPEID,
			"SMS_DLT_TEMPLATE_OTP": c.SMSDLTTemplateOTP,
		})...)
	case SMSStub:
		if !local {
			return fmt.Errorf("SMS_PROVIDER=stub only logs OTP codes and is not allowed when APP_ENV=%s", c.AppEnv)
		}
		return nil
	default:
		return fmt.Errorf("SMS_PROVIDER must be %s, %s or %s, got %q", SMSMsg91, SMS2Factor, SMSStub, c.SMSProvider)
	}
}

func (c Config) validatePush(local bool) error {
	if local {
		return nil // empty = pushes are logged, not sent
	}
	return errors.Join(requireAll(c.AppEnv, map[string]string{
		"FCM_PROJECT_ID": c.FCMProjectID, "FCM_SERVICE_ACCOUNT_FILE": c.FCMServiceAccountFile,
	})...)
}

func (c Config) validateMaps(local bool) error {
	if local {
		return nil
	}
	return errors.Join(requireAll(c.AppEnv, map[string]string{
		"GOOGLE_MAPS_BROWSER_KEY": c.MapsBrowserKey, "GOOGLE_MAPS_SERVER_KEY": c.MapsServerKey.Reveal(),
	})...)
}

// requireAll returns one error per empty variable, in a stable order, naming the variable and the environment.
func requireAll(appEnv base.Env, vars map[string]string) []error {
	names := make([]string, 0, len(vars))
	for name, v := range vars {
		if strings.TrimSpace(v) == "" {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	errs := make([]error, 0, len(names))
	for _, name := range names {
		errs = append(errs, fmt.Errorf("%s is required when APP_ENV=%s", name, appEnv))
	}
	return errs
}
