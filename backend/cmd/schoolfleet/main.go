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
	"github.com/go-chi/chi/v5"

	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/app"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/httpx"
)

func main() {
	app.Main(app.Service{
		Name:         "schoolfleet",
		PublicAddr:   ":8000",
		InternalAddr: ":8090",
		Modes:        []string{app.ModeAPI, app.ModeWorker, app.ModeIngest, app.ModeAll},
		Public: func(r chi.Router) {
			r.Get("/api/healthz", httpx.Healthz) // deploy smoke test and uptime monitor call this through Nginx
			// M0.3+: mount internal/api (school portal + apps) and internal/admin (platform) routers here.
		},
	})
}
