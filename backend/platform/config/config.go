// Package config loads a binary's configuration from environment variables once, at start-up, and validates it.
// It is the only package that reads the environment (a hook blocks os.Getenv everywhere else), so every setting
// a service uses is declared in one struct and checked before the service starts.
//
// Each binary declares its own struct, embedding Base:
//
//	type Config struct {
//		config.Base
//		GotenbergURL string `env:"GOTENBERG_URL,required,notEmpty"`
//	}
//	cfg, err := config.Load[Config]()
//
// Go note: the text in backticks is a struct tag. The env library reads it at run time to learn which
// variable fills the field and whether it is required.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"reflect"
	"strings"

	"github.com/caarlos0/env/v11"
)

// Env is the deployment environment (APP_ENV).
type Env string

const (
	EnvLocal      Env = "local"
	EnvStaging    Env = "staging"
	EnvProduction Env = "production"
)

// Base holds the settings every binary has. Values come from /srv/schoolfleet/compose/.env (infra/.env.example).
type Base struct {
	AppEnv      Env        `env:"APP_ENV,required,notEmpty"`
	ServiceName string     `env:"SVC_NAME,required,notEmpty"`
	DatabaseURL Secret     `env:"DATABASE_URL,required,notEmpty"` // sf_app (RLS); migrate: sf_owner
	DBMaxConns  int32      `env:"DB_MAX_CONNS" envDefault:"3"`    // per pool; api app pool 8 (docs/06 §4)
	LogLevel    slog.Level `env:"LOG_LEVEL" envDefault:"info"`    // debug | info | warn | error
	LogFormat   string     `env:"LOG_FORMAT" envDefault:"json"`   // json | text
	SentryDSN   Secret     `env:"SENTRY_DSN"`                     // empty = Sentry off
}

// BaseConfig returns the shared settings. A config struct that embeds Base gets it too, which is how the app
// package reads them without knowing the binary's own struct.
func (b Base) BaseConfig() Base { return b }

// One binary (`schoolfleet serve --mode=api|worker|all`) plus the migrator (docs/02 A4).
var services = map[string]bool{"schoolfleet": true, "migrate": true}

// Validate checks what struct tags can't. A struct that embeds Base gets this method too; if it defines its own
// Validate, that one must call c.Base.Validate() first.
func (b Base) Validate() error {
	var errs []error
	switch b.AppEnv {
	case EnvLocal, EnvStaging, EnvProduction:
	default:
		errs = append(errs, fmt.Errorf("APP_ENV must be local, staging or production, got %q", b.AppEnv))
	}
	if !services[b.ServiceName] {
		errs = append(errs, fmt.Errorf("SVC_NAME %q is not a known service", b.ServiceName))
	}
	if b.DBMaxConns < 1 || b.DBMaxConns > 10 {
		errs = append(errs, fmt.Errorf("DB_MAX_CONNS must be between 1 and 10, got %d", b.DBMaxConns))
	}
	if b.LogFormat != "json" && b.LogFormat != "text" {
		errs = append(errs, fmt.Errorf("LOG_FORMAT must be json or text, got %q", b.LogFormat))
	}
	if err := CheckDatabaseURL("DATABASE_URL", b.DatabaseURL, b.AppEnv); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// CheckDatabaseURL validates a Postgres URL held in the environment variable `name`: it must be postgres://, and
// anything that leaves the private Docker network must use TLS. A binary with a second database URL calls it
// from its own Validate. It never includes the URL (or url.Parse's error, which
// quotes it) in its errors: the URL contains the database password.
func CheckDatabaseURL(name string, s Secret, appEnv Env) error {
	u, err := url.Parse(s.Reveal())
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Host == "" {
		return fmt.Errorf("%s must be a postgres:// URL (value not shown: it contains a password)", name)
	}
	if appEnv == EnvLocal || localHosts[u.Hostname()] {
		return nil
	}
	// Anything that leaves the host's private Docker network must use TLS.
	switch u.Query().Get("sslmode") {
	case "require", "verify-ca", "verify-full":
		return nil
	default:
		return fmt.Errorf("%s must set sslmode=require, verify-ca or verify-full when APP_ENV=%s", name, appEnv)
	}
}

// localHosts are database hosts that never leave the machine: the compose service name and loopback. Postgres
// runs in the same compose project on the private `schoolfleet` network (docs/06), so TLS adds nothing there.
var localHosts = map[string]bool{"db": true, "localhost": true, "127.0.0.1": true, "::1": true}

type validator interface{ Validate() error }

// Load reads the process environment into a T and validates it.
func Load[T any]() (T, error) { return load[T](env.Options{}) }

// LoadFrom is Load with an explicit environment instead of the process's. Tests use it so they never depend on
// (or change) real environment variables.
func LoadFrom[T any](environ map[string]string) (T, error) {
	return load[T](env.Options{Environment: environ})
}

func load[T any](opts env.Options) (T, error) {
	cfg, err := env.ParseAsWithOptions[T](opts)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("config: %w", nameEnvVars[T](err))
	}
	if v, ok := any(&cfg).(validator); ok {
		if err := v.Validate(); err != nil {
			var zero T
			return zero, fmt.Errorf("config: %w", err)
		}
	}
	return cfg, nil
}

// nameEnvVars rewrites the env library's parse errors, which name the Go field ("DBMaxConns"), to name the
// environment variable an operator actually sets ("DB_MAX_CONNS"). The library returns every problem at once
// in an AggregateError, so each one is rewritten.
//
// A parse error quotes the bad value. Secret is a plain string type and never fails to parse, so no secret can
// appear here; a future secret type with its own parsing must not echo its input in errors.
func nameEnvVars[T any](err error) error {
	var agg env.AggregateError
	if !errors.As(err, &agg) {
		return err
	}
	vars := map[string]string{}
	for _, f := range reflect.VisibleFields(reflect.TypeFor[T]()) {
		if name, _, _ := strings.Cut(f.Tag.Get("env"), ","); name != "" {
			vars[f.Name] = name
		}
	}
	out := make([]error, 0, len(agg.Errors))
	for _, e := range agg.Errors {
		var pe env.ParseError
		if errors.As(e, &pe) && vars[pe.Name] != "" {
			e = fmt.Errorf("%s: invalid value: %w", vars[pe.Name], pe.Err)
		}
		out = append(out, e)
	}
	return errors.Join(out...)
}
