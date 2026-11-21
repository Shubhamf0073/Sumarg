package adminauth

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"slices"

	"github.com/jackc/pgx/v5"

	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/store/platformq"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/httpx"
)

// Middleware runs in front of every /api/admin operation. Sign-in and enrolment are open (they check Origin
// themselves); every other route needs an admin session, so a new admin route is protected by default. Admin
// routes never run inside a tenant transaction: services open their own on the platform pool and name the tenant.
func Middleware(d Deps) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(context.WithValue(r.Context(), depsKey{}, d))
			switch r.URL.Path {
			case PathLoginBegin, PathLoginFinish, PathEnrolBegin, PathEnrolFinish:
				next.ServeHTTP(w, r)
				return
			}
			p, err := d.Authenticate(r)
			if err != nil {
				httpx.WriteError(w, r, err)
				return
			}
			next.ServeHTTP(w, r.WithContext(withPrincipal(r.Context(), p)))
		})
	}
}

// Authenticate returns the admin principal for the session cookie. Writes also need the admin origin and the
// double-submit CSRF token. Bearer tokens are never accepted for /admin.
func (d Deps) Authenticate(r *http.Request) (*Principal, error) {
	c, err := r.Cookie(SessionCookie)
	if err != nil || r.Header.Get("Authorization") != "" {
		return nil, ErrUnauthenticated
	}
	hash, ok := hashToken(c.Value)
	if !ok {
		return nil, ErrUnauthenticated
	}
	if !safeMethod(r.Method) {
		if err := d.checkCSRF(r); err != nil {
			return nil, err
		}
	}
	ctx := r.Context()
	var row platformq.GetPlatformSessionRow
	err = d.DB(ctx, func(tx pgx.Tx) error {
		q := platformq.New(tx)
		var err error
		row, err = q.GetPlatformSession(ctx, hash)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUnauthenticated
		}
		if err != nil {
			return err
		}
		if row.TouchDue {
			return q.TouchPlatformSession(ctx, row.ID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &Principal{SessionID: row.ID, UserID: row.UserID, Name: row.Name, Email: row.Email, Role: row.Role}, nil
}

func (d Deps) checkCSRF(r *http.Request) error {
	if !d.originAllowed(r.Header.Get("Origin")) {
		return ErrCSRF
	}
	c, err := r.Cookie(CSRFCookie)
	hdr := r.Header.Get(CSRFHeader)
	if err != nil || c.Value == "" || subtle.ConstantTimeCompare([]byte(c.Value), []byte(hdr)) != 1 {
		return ErrCSRF
	}
	return nil
}

func (d Deps) originAllowed(origin string) bool {
	return origin != "" && slices.Contains(d.Origins, origin)
}

func safeMethod(m string) bool {
	return m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions
}

func sessionCookie(v string, maxAge int) http.Cookie {
	return http.Cookie{Name: SessionCookie, Value: v, Path: "/", MaxAge: maxAge, Secure: true, HttpOnly: true,
		SameSite: http.SameSiteStrictMode}
}

// csrfCookie is read by the admin pages' script and echoed in X-CSRF-Token (frontend/src/api/client.ts).
func csrfCookie(v string, maxAge int) http.Cookie {
	// #nosec G124 -- not HttpOnly on purpose: double-submit needs the script to read it; it grants nothing alone.
	return http.Cookie{Name: CSRFCookie, Value: v, Path: "/", MaxAge: maxAge, Secure: true,
		SameSite: http.SameSiteStrictMode}
}

func ceremonyCookie(v string, maxAge int) http.Cookie {
	return http.Cookie{Name: CeremonyCookie, Value: v, Path: "/", MaxAge: maxAge, Secure: true, HttpOnly: true,
		SameSite: http.SameSiteStrictMode}
}
