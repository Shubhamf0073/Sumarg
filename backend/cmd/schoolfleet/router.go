package main

import (
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/admin/adminauth"
	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/admin/schools"
	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/tenant/auth"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/httpx"
)

// registerAPI registers every huma operation. `schoolfleet openapi` calls it too, on a throwaway router, so it
// must not need a database or config: services get their pools from request context, not from here.
// M1.3+: register internal/{tenant,parent,driver}/* and internal/admin/* modules here.
func registerAPI(api huma.API) {
	auth.Register(api)
	adminauth.Register(api)
	schools.Register(api)
}

// mountAPI serves registerAPI's operations on r. main and the isolation walker both use it, so the walker tests
// the real middleware.
//
// Every operation runs signed in, inside one tenant transaction (db.RequestTx with the session resolver), unless
// it is listed in gate: a new route is protected by default. /api/admin/* never gets a tenant transaction: it runs
// behind the admin passkey session instead (adminauth.Middleware), and its services use the platform pool.
func mountAPI(r chi.Router, d auth.Deps, a adminauth.Deps) {
	r.Group(func(r chi.Router) {
		r.Use(auth.Provide(d), gate(d, d.DB.RequestTx(d.Resolve), adminauth.Middleware(a)))
		registerAPI(httpx.NewAPI(r, apiTitle, apiVersion))
	})
}

// gate picks what runs in front of each operation. Sign-in (portal, PIN, parent OTP) resolves its own tenant from
// the school code, and set-password from its one-time link; logout
// needs the session but not the request transaction (a read-only session must still be able to end itself).
func gate(d auth.Deps, requestTx, admin func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		signedIn, sessionOnly, asAdmin := requestTx(next), d.SessionOnly(next), admin(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, adminauth.PathPrefix) {
				asAdmin.ServeHTTP(w, r)
				return
			}
			switch r.URL.Path {
			case auth.PathLogin, auth.PathPIN, auth.PathOTP, auth.PathOTPVerify, auth.PathSetPassword:
				next.ServeHTTP(w, r)
			case auth.PathLogout:
				sessionOnly.ServeHTTP(w, r)
			default:
				signedIn.ServeHTTP(w, r)
			}
		})
	}
}
