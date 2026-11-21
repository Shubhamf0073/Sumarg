// Package adminauth signs Previa staff in to /admin with passkeys only (docs/02 A24, docs/07 S6) and turns the
// admin session cookie back into a principal on every /api/admin request.
//
// There are no passwords and no default credentials. The first passkey is registered from a one-time enrolment
// link (`migrate create-admin`, 15 minutes, single use); after that the staff member signs in with a discoverable
// credential (no e-mail typed). Sessions are absolute (ADMIN_SESSION_HOURS) and stored as sha256(token) only.
// Cookie writes need Origin + a double-submit CSRF token, as the portal's do, but with their own cookies so an
// admin and a school session in the same browser never touch each other.
//
// Everything runs on the platform pool (sf_platform, BYPASSRLS): these tables are not tenant data.
package adminauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"slices"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/jackc/pgx/v5"

	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/clock"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/httpx"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/ratelimit"
)

// Paths, shared with the router in cmd/schoolfleet.
const (
	PathPrefix      = "/api/admin/"
	PathLoginBegin  = "/api/admin/auth/login/begin"
	PathLoginFinish = "/api/admin/auth/login/finish"
	PathEnrolBegin  = "/api/admin/auth/enrol/begin"
	PathEnrolFinish = "/api/admin/auth/enrol/finish"
	PathLogout      = "/api/admin/auth/logout"
	PathMe          = "/api/admin/me"
)

// Cookie names. __Host- keeps them Secure, host-only and on Path=/ (a sibling subdomain can't plant one).
const (
	SessionCookie   = "__Host-sf_admin"
	CSRFCookie      = "__Host-sf_admin_csrf"
	CeremonyCookie  = "__Host-sf_admin_cer"
	CSRFHeader      = "X-CSRF-Token"
	EnrolmentMinute = 15 // an enrolment link lives this many minutes
)

// Roles (platform_users.role). Superadmin can do everything; the others are limited per route with Require.
const (
	RoleSuperadmin = "superadmin"
	RoleSupport    = "support"
	RoleBilling    = "billing"
	RoleInstaller  = "installer"
)

// Client errors. One 401 for every way a sign-in or a session can be wrong.
var (
	ErrUnauthenticated = httpx.NewError(http.StatusUnauthorized, "unauthenticated", "Please sign in.")
	ErrSignInFailed    = httpx.NewError(http.StatusUnauthorized, "sign_in_failed",
		"That passkey could not be verified. Please try again.")
	ErrEnrolment = httpx.NewError(http.StatusGone, "enrolment_invalid",
		"This enrolment link has expired or was already used. Ask a Previa superadmin for a new one.")
	ErrCSRF = httpx.NewError(http.StatusForbidden, "csrf_failed",
		"This request was blocked. Reload the page and try again.")
	ErrForbidden = httpx.NewError(http.StatusForbidden, "forbidden", "Your role can't do this.")
	ErrTooMany   = httpx.NewError(http.StatusTooManyRequests, "too_many_requests",
		"Too many attempts. Please wait a few minutes.")
)

// InPlatform runs fn in one transaction on the platform pool (platformdb.InPlatform bound to the pool).
type InPlatform func(ctx context.Context, fn func(pgx.Tx) error) error

// Deps is what the admin auth endpoints and the session check need. main builds one; Middleware puts it in every
// /api/admin request's context, so registering the operations (and `schoolfleet openapi`) needs no database.
type Deps struct {
	DB InPlatform
	// DBRead is DB in a READ ONLY transaction (platformdb.InPlatformReadOnly) for GET handlers.
	DBRead   InPlatform
	WebAuthn *webauthn.WebAuthn
	// Origins accepted on sign-in and on cookie writes (config.PortalOrigins: /admin is served from the same host).
	Origins []string
	// SessionHours is ADMIN_SESSION_HOURS.
	SessionHours int
	// TrustedProxies: X-Forwarded-For is believed only from these peers (config.TrustedProxyPrefixes).
	TrustedProxies []netip.Prefix
	// Ceremonies holds begun passkey ceremonies until they finish (single use, 5 min).
	Ceremonies *Ceremonies
	// Limit caps ceremonies begun and finished per client address; nil allows all (tests only).
	Limit *ratelimit.Limiter
}

// NewDeps fills the parts of Deps that hold state: the WebAuthn relying party (rpID is the bare host, "localhost"
// locally), the ceremony store and the per-address limit (30 ceremony steps per 10 minutes).
func NewDeps(db, dbRead InPlatform, rpID string, origins []string, sessionHours int, trusted []netip.Prefix,
	c clock.Clock) (Deps, error) {
	wa, err := NewWebAuthn(rpID, origins)
	if err != nil {
		return Deps{}, err
	}
	return Deps{DB: db, DBRead: dbRead, WebAuthn: wa, Origins: origins, SessionHours: sessionHours, TrustedProxies: trusted,
		Ceremonies: NewCeremonies(c), Limit: ratelimit.New(c, 30, ceremonyWindow)}, nil
}

// NewWebAuthn is the relying party: passkeys bound to rpID, accepted only from origins.
func NewWebAuthn(rpID string, origins []string) (*webauthn.WebAuthn, error) {
	wa, err := webauthn.New(&webauthn.Config{
		RPID: rpID, RPDisplayName: "Sumarg Admin", RPOrigins: origins,
		Timeouts: webauthn.TimeoutsConfig{
			Login:        webauthn.TimeoutConfig{Enforce: true, Timeout: ceremonyTTL, TimeoutUVD: ceremonyTTL},
			Registration: webauthn.TimeoutConfig{Enforce: true, Timeout: ceremonyTTL, TimeoutUVD: ceremonyTTL},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("adminauth: webauthn: %w", err)
	}
	return wa, nil
}

type depsKey struct{}

func depsFrom(ctx context.Context) (Deps, error) {
	d, ok := ctx.Value(depsKey{}).(Deps)
	if !ok {
		return Deps{}, errors.New("adminauth: Deps not provided (mount adminauth.Middleware)")
	}
	return d, nil
}

// Platform is the platform-pool transaction runner for /api/admin services (internal/admin/*): they name the tenant
// in every query. Only available behind Middleware.
func Platform(ctx context.Context) (InPlatform, error) {
	d, err := depsFrom(ctx)
	if err != nil {
		return nil, err
	}
	return d.DB, nil
}

// PlatformReadOnly is Platform in a READ ONLY transaction: /api/admin GET services use it.
func PlatformReadOnly(ctx context.Context) (InPlatform, error) {
	d, err := depsFrom(ctx)
	if err != nil {
		return nil, err
	}
	if d.DBRead == nil {
		return nil, errors.New("adminauth: Deps.DBRead not set")
	}
	return d.DBRead, nil
}

// PortalOrigin is where links for school staff point (the first configured portal origin, APP_DOMAIN in production).
func PortalOrigin(ctx context.Context) (string, error) {
	d, err := depsFrom(ctx)
	if err != nil {
		return "", err
	}
	if len(d.Origins) == 0 {
		return "", errors.New("adminauth: no portal origin configured")
	}
	return d.Origins[0], nil
}

// ClientIP is the admin's address for audit rows (X-Forwarded-For only from TrustedProxies).
func ClientIP(ctx huma.Context) *netip.Addr {
	var c client
	c.resolve(ctx)
	return c.ip
}

// ── principal ──

// Principal is the signed-in Previa staff member.
type Principal struct {
	SessionID int64
	UserID    int64
	Name      string
	Email     string
	Role      string
}

type principalKey struct{}

func withPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// FromContext is the request's admin principal; nil outside a signed-in /api/admin route.
func FromContext(ctx context.Context) *Principal {
	p, _ := ctx.Value(principalKey{}).(*Principal)
	return p
}

// Require returns the principal if their role is superadmin or one of roles; otherwise 403 (401 if signed out).
func Require(ctx context.Context, roles ...string) (*Principal, error) {
	p := FromContext(ctx)
	if p == nil {
		return nil, ErrUnauthenticated
	}
	if p.Role != RoleSuperadmin && !slices.Contains(roles, p.Role) {
		return nil, ErrForbidden
	}
	return p, nil
}

// ── tokens ──

const tokenBytes = 32

var tokenB64Len = base64.RawURLEncoding.EncodedLen(tokenBytes)

// NewToken returns 32 random bytes in base64url and their sha256 (what is stored).
func NewToken() (token string, hash []byte, err error) {
	raw := make([]byte, tokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("adminauth: token: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	h := sha256.Sum256([]byte(token))
	return token, h[:], nil
}

// hashToken accepts exactly 43 strict base64url characters decoding to 32 bytes; anything else is refused before
// any database work.
func hashToken(s string) ([]byte, bool) {
	if len(s) != tokenB64Len {
		return nil, false
	}
	raw, err := base64.RawURLEncoding.Strict().DecodeString(s)
	if err != nil || len(raw) != tokenBytes {
		return nil, false
	}
	h := sha256.Sum256([]byte(s))
	return h[:], true
}

// ── the WebAuthn user ──

// adminUser is a platform user as the WebAuthn library sees them. The user handle is the 8-byte id: an opaque
// number, nothing personal is stored on the authenticator beyond the e-mail shown as the account name.
type adminUser struct {
	id          int64
	name, email string
	creds       []webauthn.Credential
}

func userHandle(id int64) []byte { return binary.BigEndian.AppendUint64(nil, uint64(id)) }

func (u *adminUser) WebAuthnID() []byte                         { return userHandle(u.id) }
func (u *adminUser) WebAuthnName() string                       { return u.email }
func (u *adminUser) WebAuthnDisplayName() string                { return u.name }
func (u *adminUser) WebAuthnCredentials() []webauthn.Credential { return u.creds }

func (u *adminUser) exclusions() []protocol.CredentialDescriptor {
	out := make([]protocol.CredentialDescriptor, 0, len(u.creds))
	for i := range u.creds {
		out = append(out, u.creds[i].Descriptor())
	}
	return out
}
