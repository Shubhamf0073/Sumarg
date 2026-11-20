package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/store/tenantq"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/db"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/httpx"
)

// Cookie and header names (docs/04 "Auth", docs/05 §10). The __Host- prefix makes the browser refuse a cookie
// that is not Secure, has a Domain, or a Path other than /, so a sibling subdomain can't plant one.
const (
	SessionCookie = "__Host-sf_sid"
	CSRFCookie    = "__Host-sf_csrf"
	CSRFHeader    = "X-CSRF-Token"
	// InstallHeader carries the app install's id on every app request; it must match the session's (device binding).
	InstallHeader = "X-Install-Id"
)

// Client errors. One 401 for every way a session can be wrong, so the answer says nothing about which.
var (
	ErrUnauthenticated = httpx.NewError(http.StatusUnauthorized, "unauthenticated", "Please sign in.")
	ErrCSRF            = httpx.NewError(http.StatusForbidden, "csrf_failed",
		"This request was blocked. Reload the page and try again.")
	ErrSchoolClosed = httpx.NewError(http.StatusLocked, "school_closed",
		"This school's Sumarg account is closed. Please contact Previa.")
)

// Deps is what the auth endpoints and the session resolver need. main builds one and Provide puts it in every
// request's context, so registerAPI (and `schoolfleet openapi`) need no database.
type Deps struct {
	DB db.Tenants
	// Origins accepted on portal login and cookie-authenticated writes (config.PortalOrigins).
	Origins []string
}

type depsKey struct{}

// Provide makes d available to the auth handlers.
func Provide(d Deps) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), depsKey{}, d)))
		})
	}
}

func depsFrom(ctx context.Context) (Deps, error) {
	d, ok := ctx.Value(depsKey{}).(Deps)
	if !ok {
		return Deps{}, errors.New("auth: Deps not provided (mount auth.Provide)")
	}
	return d, nil
}

// Resolve is the db.Resolver for every signed-in route: it finds the session, checks how it was presented, touches
// it, and hands the principal to the handler through the context.
func (d Deps) Resolve(r *http.Request) (db.Resolved, error) {
	p, err := d.Authenticate(r)
	if err != nil {
		return db.Resolved{}, err
	}
	return db.Resolved{TenantID: p.TenantID, ReadOnly: p.ReadOnly, Ctx: withPrincipal(r.Context(), p)}, nil
}

// Authenticate returns the request's principal. Apps send `Authorization: Bearer <token>` plus X-Install-Id; the
// portal sends the session cookie, and on any write also Origin and the CSRF double-submit token. A bearer token for
// a portal session, or a cookie for an app session, is refused: each kind has its own protections.
func (d Deps) Authenticate(r *http.Request) (*Principal, error) {
	var token string
	bearer := false
	if h := r.Header.Get("Authorization"); h != "" {
		t, ok := strings.CutPrefix(h, "Bearer ")
		if !ok {
			return nil, ErrUnauthenticated
		}
		token, bearer = t, true
	} else if c, err := r.Cookie(SessionCookie); err == nil {
		token = c.Value
		if !safeMethod(r.Method) {
			if err := d.checkCSRF(r); err != nil {
				return nil, err
			}
		}
	} else {
		return nil, ErrUnauthenticated
	}
	tenantID, hash, ok := parseToken(token)
	if !ok {
		return nil, ErrUnauthenticated
	}

	ctx := r.Context()
	var row tenantq.GetSessionRow
	err := d.DB.InTenant(ctx, tenantID, func(tx pgx.Tx) error {
		q := tenantq.New(tx)
		var err error
		row, err = q.GetSession(ctx, hash)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUnauthenticated
		}
		if err != nil {
			return err
		}
		// Check how it was presented before renewing it: a token copied off the phone can't keep itself alive.
		if err := checkPresentation(r, row, bearer); err != nil {
			return err
		}
		if row.TouchDue {
			return q.TouchSession(ctx, row.ID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if row.TenantStatus == "cancelled" {
		return nil, ErrSchoolClosed
	}
	return principalFrom(tenantID, row), nil
}

func checkPresentation(r *http.Request, row tenantq.GetSessionRow, bearer bool) error {
	// Previa support-view sessions (platform_user_id only) have no principal here yet: closed until they're built.
	if row.UserID == nil && row.GuardianID == nil {
		return ErrUnauthenticated
	}
	if bearer != (row.Client != ClientWeb) {
		return ErrUnauthenticated
	}
	if bearer {
		got := r.Header.Get(InstallHeader)
		if row.InstallID == nil || got == "" || subtle.ConstantTimeCompare([]byte(got), []byte(*row.InstallID)) != 1 {
			return ErrUnauthenticated
		}
	}
	return nil
}

func principalFrom(tenantID int64, row tenantq.GetSessionRow) *Principal {
	p := &Principal{
		SessionID: row.ID, TenantID: tenantID, Client: row.Client, Name: row.PrincipalName,
		CampusID: row.UserCampusID, TenantCode: row.TenantCode, TenantName: row.TenantName,
		ReadOnly: row.ReadOnly || row.TenantStatus == "suspended",
	}
	if row.UserID != nil {
		p.UserID = *row.UserID
	}
	if row.GuardianID != nil {
		p.GuardianID = *row.GuardianID
		p.Role = "parent"
	}
	if row.UserRole != nil {
		p.Role = *row.UserRole
	}
	if row.UserMustChangeSecret != nil {
		p.MustChangeSecret = *row.UserMustChangeSecret
	}
	return p
}

// checkCSRF: the request comes from the portal's own origin, and echoes the CSRF cookie in a header (a page on
// another site can neither read the cookie nor send the header without a CORS preflight we never answer).
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
