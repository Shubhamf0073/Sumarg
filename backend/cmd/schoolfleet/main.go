// Command schoolfleet is Sumarg's API, background worker and device ingest in one binary
// (docs/02 A4).
//
//	schoolfleet serve [--mode=api|worker|ingest|all]   production: three containers; local dev: all
//	schoolfleet healthcheck                            Docker's healthcheck (distroless: no shell, no curl)
//
// HTTP routes: /api (school portal + mobile apps), /api/admin (Previa superadmin). The ingest mode listens for
// bus units on MQTT/TLS :8883 (docs/13); it has no public HTTP routes.
package main

import (
	"context"
	"log/slog"

	"github.com/go-chi/chi/v5"

	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/config"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/app"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/httpx"
)

func main() {
	app.Main(app.Service[config.Config]{
		Name:         "schoolfleet",
		PublicAddr:   ":8000",
		InternalAddr: ":8090",
		Modes:        []string{app.ModeAPI, app.ModeWorker, app.ModeIngest, app.ModeAll},
		Setup:        setup,
		Public: func(r chi.Router) {
			r.Get("/api/healthz", httpx.Healthz) // deploy smoke test and uptime monitor call this through Nginx
			// M0.3+: mount internal/api (school portal + apps) and internal/admin (platform) routers here.
		},
	})
}

// setup runs once at start-up, before any listener. M0.3 opens the two database pools here (sf_app and sf_platform)
// and returns their readiness checks and Close; for now it records which integrations this process runs with, so a
// misconfigured server is obvious from its first log lines. Nothing secret is logged: only on/off and names.
func setup(_ context.Context, cfg config.Config, log *slog.Logger) (*app.Runtime, error) {
	log.Info("configuration loaded",
		slog.String("app_env", string(cfg.AppEnv)),
		slog.String("sms_provider", cfg.SMSProvider),
		slog.Bool("push_enabled", cfg.FCMProjectID != ""),
		slog.Bool("maps_enabled", cfg.MapsServerKey != ""),
		slog.Bool("mqtt_tls", cfg.MQTTCert != ""),
		slog.Bool("mqtt_plain_dev_port", cfg.MQTTPlainPort != 0))
	return &app.Runtime{}, nil
}
