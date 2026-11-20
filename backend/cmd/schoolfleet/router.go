package main

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/tenant/auth"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/db"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/httpx"
)

// registerAPI registers every huma operation. `schoolfleet openapi` calls it too, on a throwaway router, so it
// must not need a database or config: services get their pools from request context, not from here.
// M1.3+: register internal/{tenant,parent,driver}/* and internal/admin/* modules here.
func registerAPI(api huma.API) {
	auth.Register(api)
}

// mountAPI serves registerAPI's operations on r. main and the isolation walker both use it, so the walker tests
// the real middleware.
//
// Every operation runs signed in, inside one tenant transaction (db.RequestTx with the session resolver), unless
// it is listed in gate: a new route is protected by default.
func mountAPI(r chi.Router, tenants db.Tenants, origins []string) {
	d := auth.Deps{DB: tenants, Origins: origins}
	r.Group(func(r chi.Router) {
		r.Use(auth.Provide(d), gate(d, tenants.RequestTx(d.Resolve)))
		registerAPI(httpx.NewAPI(r, apiTitle, apiVersion))
	})
}

// gate picks what runs in front of each operation. Login resolves its own tenant from the school code; logout
// needs the session but not the request transaction (a read-only session must still be able to end itself).
func gate(d auth.Deps, requestTx func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		signedIn, sessionOnly := requestTx(next), d.SessionOnly(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case auth.PathLogin, auth.PathPIN:
				next.ServeHTTP(w, r)
			case auth.PathLogout:
				sessionOnly.ServeHTTP(w, r)
			default:
				signedIn.ServeHTTP(w, r)
			}
		})
	}
}
