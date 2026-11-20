// Command schoolfleet is Sumarg's API, background worker and device ingest in one binary
// (docs/02 A4).
//
//	schoolfleet serve [--mode=api|worker|ingest|all]   production: three containers; local dev: all
//	schoolfleet openapi                                print the OpenAPI 3.1 document (make gen-api)
//	schoolfleet healthcheck                            Docker's healthcheck (distroless: no shell, no curl)
//
// HTTP routes: /api (school portal + mobile apps), /api/admin (Previa superadmin). The ingest mode listens for
// bus units on MQTT/TLS :8883 (docs/13); it has no public HTTP routes.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/go-chi/chi/v5"

	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/config"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/app"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/db"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/db/platformdb"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/httpx"
)

func main() {
	app.Main(app.Service[config.Config]{
		Name:         "schoolfleet",
		PublicAddr:   ":8000",
		InternalAddr: ":8090",
		Modes:        []string{app.ModeAPI, app.ModeWorker, app.ModeIngest, app.ModeAll},
		Setup:        setup,
		OpenAPI:      openapi,
		Public: func(r chi.Router) {
			r.Get("/api/healthz", httpx.Healthz) // deploy smoke test and uptime monitor call this through Nginx
			// The huma operations are mounted by setup (Runtime.Routes): they need the database pools.
		},
	})
}

const (
	apiTitle   = "Sumarg API"
	apiVersion = "1.0.0"
)

// openapi renders the document `make gen-api` turns into TypeScript types for the portal and the app.
func openapi() ([]byte, error) {
	api := httpx.NewSpecAPI(apiTitle, apiVersion)
	registerAPI(api)
	doc, err := json.MarshalIndent(api.OpenAPI(), "", "  ")
	if err != nil {
		return nil, fmt.Errorf("render openapi: %w", err)
	}
	return doc, nil
}

// setup runs once at start-up, before any listener. It opens the two database pools (sf_app under RLS, sf_platform
// bypassing it), each checked against its expected role, and returns their readiness checks and Close. It also
// records which integrations this process runs with, so a misconfigured server is obvious from its first log lines.
// Nothing secret is logged: only on/off and names.
func setup(ctx context.Context, cfg config.Config, log *slog.Logger) (*app.Runtime, error) {
	log.Info("configuration loaded",
		slog.String("app_env", string(cfg.AppEnv)),
		slog.String("sms_provider", cfg.SMSProvider),
		slog.Bool("push_enabled", cfg.FCMProjectID != ""),
		slog.Bool("maps_enabled", cfg.MapsServerKey != ""),
		slog.Bool("mqtt_tls", cfg.MQTTCert != ""),
		slog.Bool("mqtt_plain_dev_port", cfg.MQTTPlainPort != 0))

	appPool, err := db.OpenApp(ctx, cfg.DatabaseURL, db.Options{Name: "schoolfleet-app", MaxConns: cfg.DBMaxConns})
	if err != nil {
		return nil, fmt.Errorf("open app pool: %w", err)
	}
	platPool, err := platformdb.Open(ctx, cfg.PlatformDatabaseURL,
		db.Options{Name: "schoolfleet-platform", MaxConns: cfg.PlatformDBMaxConns})
	if err != nil {
		appPool.Close() // Close is not called when Setup fails
		return nil, fmt.Errorf("open platform pool: %w", err)
	}
	// platPool is handed to /api/admin (M1.4) and ingest (M3).
	tenants := db.Bind(appPool)
	return &app.Runtime{
		Routes: func(r chi.Router) { mountAPI(r, tenants, cfg.PortalOrigins()) },
		Ready: []httpx.Check{
			{Name: "db-app", Fn: db.Ping(appPool)},
			{Name: "db-platform", Fn: db.Ping(platPool)},
		},
		// pool.Close waits for every connection to be released; don't let a stuck one outlast CloseGrace.
		Close: func(ctx context.Context) error {
			done := make(chan struct{})
			go func() {
				appPool.Close()
				platPool.Close()
				close(done)
			}()
			select {
			case <-done:
				return nil
			case <-ctx.Done():
				return fmt.Errorf("close database pools: %w", ctx.Err())
			}
		},
	}, nil
}
