package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"

	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/store/tenantq"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/db"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/httpx"
)

// Paths, shared with the router in cmd/schoolfleet, which decides what each one runs behind.
const (
	PathLogin  = "/api/auth/login"
	PathPIN    = "/api/auth/app/pin"
	PathLogout = "/api/auth/logout"
	PathMe     = "/api/me"
	// PathOTP and PathOTPVerify are the parent app's sign-in (pre-session, like PathLogin and PathPIN).
	PathOTP       = "/api/auth/app/otp"
	PathOTPVerify = "/api/auth/app/otp/verify"
	PathPushToken = "/api/auth/push-token" // #nosec G101 -- a URL path, not a credential
)

// Register adds the auth operations (docs/04 "Auth (all clients)").
func Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "auth-login", Method: http.MethodPost, Path: PathLogin, Tags: []string{"auth"},
		Summary: "Portal sign-in: school code, mobile and password → session cookie",
		Errors:  []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusLocked},
	}, portalLogin)
	huma.Register(api, huma.Operation{
		OperationID: "auth-pin", Method: http.MethodPost, Path: PathPIN, Tags: []string{"auth"},
		Summary: "Driver/attendant app sign-in: school code, mobile and PIN → bearer token bound to the install",
		Errors:  []int{http.StatusUnauthorized, http.StatusLocked},
	}, pinLogin)
	huma.Register(api, huma.Operation{
		OperationID: "auth-otp", Method: http.MethodPost, Path: PathOTP, Tags: []string{"auth"},
		Summary: "Parent app sign-in, step 1: school code and mobile → an SMS code to invited guardians " +
			"(202 for every mobile)",
		DefaultStatus: http.StatusAccepted,
		Errors:        []int{http.StatusTooManyRequests},
	}, otpRequestHandler)
	huma.Register(api, huma.Operation{
		OperationID: "auth-otp-verify", Method: http.MethodPost, Path: PathOTPVerify, Tags: []string{"auth"},
		Summary: "Parent app sign-in, step 2: the SMS code → bearer token bound to the install (90 days, renewed on use)",
		Errors:  []int{http.StatusUnauthorized, http.StatusLocked, http.StatusTooManyRequests},
	}, otpVerifyHandler)
	huma.Register(api, huma.Operation{
		OperationID: "auth-push-token", Method: http.MethodPost, Path: PathPushToken, Tags: []string{"auth"},
		Summary:       "Parent / driver app: register this install's FCM token for the signed-in session",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusUnauthorized, http.StatusForbidden},
	}, registerPushToken)
	huma.Register(api, huma.Operation{
		OperationID: "auth-logout", Method: http.MethodPost, Path: PathLogout, Tags: []string{"auth"},
		Summary:       "Sign out: ends this session",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusUnauthorized, http.StatusForbidden},
	}, logout)
	huma.Register(api, huma.Operation{
		OperationID: "auth-me", Method: http.MethodGet, Path: PathMe, Tags: []string{"auth"},
		Summary: "Who is signed in (decides the app's tab set and the portal's menu)",
		Errors:  []int{http.StatusUnauthorized},
	}, me)
}

// ── shapes ──

type userBody struct {
	ID               int64  `json:"id"`
	Name             string `json:"name"`
	Role             string `json:"role" enum:"owner,admin,transport_manager,driver,attendant"`
	CampusID         *int64 `json:"campus_id" doc:"null = all campuses"`
	MustChangeSecret bool   `json:"must_change_secret"`
}

type guardianBody struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type schoolBody struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type MeBody struct {
	Client   string        `json:"client" enum:"web,driver_app,parent_app"`
	User     *userBody     `json:"user,omitempty" doc:"school staff or crew"`
	Guardian *guardianBody `json:"guardian,omitempty" doc:"parent"`
	School   schoolBody    `json:"school"`
	ReadOnly bool          `json:"read_only" doc:"suspended school or Previa support view: writes are refused"`
}

func meFrom(p *Principal) MeBody {
	b := MeBody{Client: p.Client, ReadOnly: p.ReadOnly,
		School: schoolBody{ID: p.TenantID, Code: p.TenantCode, Name: p.TenantName}}
	if p.GuardianID != 0 {
		b.Guardian = &guardianBody{ID: p.GuardianID, Name: p.Name}
	} else {
		b.User = &userBody{ID: p.UserID, Name: p.Name, Role: p.Role, CampusID: p.CampusID,
			MustChangeSecret: p.MustChangeSecret}
	}
	return b
}

// clientInfo is filled from the connection, not the body. Nginx sets X-Forwarded-For to the real client IP
// (infra/nginx/snippets/schoolfleet-proxy.conf); it is stored on the session for the audit view only.
type clientInfo struct {
	ip        *netip.Addr
	userAgent string
}

func (c *clientInfo) resolve(ctx huma.Context) {
	c.userAgent = ctx.Header("User-Agent")
	host := ctx.Header("X-Forwarded-For")
	if host == "" {
		host, _, _ = net.SplitHostPort(ctx.RemoteAddr())
	}
	if a, err := netip.ParseAddr(host); err == nil {
		a = a.Unmap()
		c.ip = &a
	}
}

// ── portal login ──

type portalLoginIn struct {
	Origin    string `header:"Origin"`
	Presented string `cookie:"__Host-sf_sid"`
	Body      struct {
		SchoolCode string `json:"school_code" pattern:"^[A-Za-z0-9-]{3,30}$"`
		Mobile     string `json:"mobile" pattern:"^[6-9][0-9]{9}$"`
		Password   string `json:"password" minLength:"1" maxLength:"128"`
	}
	client clientInfo
}

func (in *portalLoginIn) Resolve(ctx huma.Context) []error { in.client.resolve(ctx); return nil }

type portalLoginOut struct {
	SetCookie []http.Cookie `header:"Set-Cookie"`
	Body      MeBody
}

func portalLogin(ctx context.Context, in *portalLoginIn) (*portalLoginOut, error) {
	d, err := depsFrom(ctx)
	if err != nil {
		return nil, err
	}
	// No session cookie exists yet, so SameSite can't protect this request: the Origin must be the portal's.
	if !d.originAllowed(in.Origin) {
		return nil, ErrCSRF
	}
	res, err := login(ctx, d, loginRequest{
		Kind: password, SchoolCode: in.Body.SchoolCode, Mobile: in.Body.Mobile, Secret: in.Body.Password,
		Presented: in.Presented, IP: in.client.ip, UserAgent: in.client.userAgent,
	})
	if err != nil {
		return nil, err
	}
	csrf, err := randomToken()
	if err != nil {
		return nil, err
	}
	return &portalLoginOut{
		SetCookie: []http.Cookie{
			sessionCookie(res.Token, webSessionSecs),
			csrfCookie(csrf, webSessionSecs),
		},
		Body: meFrom(&res.Principal),
	}, nil
}

func sessionCookie(v string, maxAge int) http.Cookie {
	return http.Cookie{Name: SessionCookie, Value: v, Path: "/", MaxAge: maxAge, Secure: true, HttpOnly: true,
		SameSite: http.SameSiteStrictMode}
}

// csrfCookie is readable by the portal's script, which echoes it in X-CSRF-Token (frontend/src/api/client.ts).
func csrfCookie(v string, maxAge int) http.Cookie {
	// #nosec G124 -- not HttpOnly on purpose: double-submit needs the script to read it; it grants nothing alone.
	return http.Cookie{Name: CSRFCookie, Value: v, Path: "/", MaxAge: maxAge, Secure: true,
		SameSite: http.SameSiteStrictMode}
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("auth: csrf token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// ── driver app PIN ──

type pinLoginIn struct {
	Authorization string `header:"Authorization" doc:"an earlier token on this install, if any: it is revoked"`
	Body          struct {
		SchoolCode string `json:"school_code" pattern:"^[A-Za-z0-9-]{3,30}$"`
		Mobile     string `json:"mobile" pattern:"^[6-9][0-9]{9}$"`
		PIN        string `json:"pin" pattern:"^[0-9]{6}$"`
		InstallID  string `json:"install_id" pattern:"^[A-Za-z0-9-]{16,64}$" doc:"random id the app made at install; send it as X-Install-Id on every request"`
	}
	client clientInfo
}

func (in *pinLoginIn) Resolve(ctx huma.Context) []error { in.client.resolve(ctx); return nil }

type pinLoginOut struct {
	Body struct {
		Token string `json:"token" doc:"send as Authorization: Bearer <token>; keep it in expo-secure-store"`
		MeBody
	}
}

func pinLogin(ctx context.Context, in *pinLoginIn) (*pinLoginOut, error) {
	d, err := depsFrom(ctx)
	if err != nil {
		return nil, err
	}
	presented, _ := strings.CutPrefix(in.Authorization, "Bearer ")
	res, err := login(ctx, d, loginRequest{
		Kind: pin, SchoolCode: in.Body.SchoolCode, Mobile: in.Body.Mobile, Secret: in.Body.PIN,
		InstallID: in.Body.InstallID, Presented: presented, IP: in.client.ip, UserAgent: in.client.userAgent,
	})
	if err != nil {
		return nil, err
	}
	out := &pinLoginOut{}
	out.Body.Token, out.Body.MeBody = res.Token, meFrom(&res.Principal)
	return out, nil
}

// ── parent app OTP ──

type otpRequestIn struct {
	Body struct {
		SchoolCode string `json:"school_code" pattern:"^[A-Za-z0-9-]{3,30}$"`
		Mobile     string `json:"mobile" pattern:"^[6-9][0-9]{9}$"`
	}
	client clientInfo
}

func (in *otpRequestIn) Resolve(ctx huma.Context) []error { in.client.resolve(ctx); return nil }

func otpRequestHandler(ctx context.Context, in *otpRequestIn) (*struct{}, error) {
	d, err := depsFrom(ctx)
	if err != nil {
		return nil, err
	}
	return nil, requestOTP(ctx, d, otpRequest{SchoolCode: in.Body.SchoolCode, Mobile: in.Body.Mobile, IP: in.client.ip})
}

type otpVerifyIn struct {
	Authorization string `header:"Authorization" doc:"an earlier token on this install, if any: it is revoked"`
	Body          struct {
		SchoolCode string `json:"school_code" pattern:"^[A-Za-z0-9-]{3,30}$"`
		Mobile     string `json:"mobile" pattern:"^[6-9][0-9]{9}$"`
		Code       string `json:"code" pattern:"^[0-9]{6}$"`
		InstallID  string `json:"install_id" pattern:"^[A-Za-z0-9-]{16,64}$" doc:"random id the app made at install; send it as X-Install-Id on every request"`
	}
	client clientInfo
}

func (in *otpVerifyIn) Resolve(ctx huma.Context) []error { in.client.resolve(ctx); return nil }

func otpVerifyHandler(ctx context.Context, in *otpVerifyIn) (*pinLoginOut, error) {
	d, err := depsFrom(ctx)
	if err != nil {
		return nil, err
	}
	presented, _ := strings.CutPrefix(in.Authorization, "Bearer ")
	res, err := verifyOTP(ctx, d, otpVerify{
		SchoolCode: in.Body.SchoolCode, Mobile: in.Body.Mobile, Code: in.Body.Code, InstallID: in.Body.InstallID,
		Presented: presented, IP: in.client.ip, UserAgent: in.client.userAgent,
	})
	if err != nil {
		return nil, err
	}
	out := &pinLoginOut{}
	out.Body.Token, out.Body.MeBody = res.Token, meFrom(&res.Principal)
	return out, nil
}

// ── push tokens ──

var errAppOnly = httpx.NewError(http.StatusForbidden, "forbidden", "Push notifications are for the mobile app.")

type pushTokenIn struct {
	Body struct {
		Platform   string `json:"platform" enum:"android,ios"`
		FCMToken   string `json:"fcm_token" minLength:"20" maxLength:"1024" pattern:"^[A-Za-z0-9_:.-]+$"`
		AppVersion string `json:"app_version" pattern:"^[0-9A-Za-z.+-]{1,32}$"`
	}
}

// registerPushToken runs in the request transaction. A read-only session (suspended school, support view) is
// answered 204 without storing anything: nothing is pushed to it, and the app needn't retry.
func registerPushToken(ctx context.Context, in *pushTokenIn) (*struct{}, error) {
	p, ok := FromContext(ctx)
	if !ok {
		return nil, ErrUnauthenticated
	}
	if p.Client == ClientWeb {
		return nil, errAppOnly
	}
	if p.ReadOnly {
		return nil, nil
	}
	tx, ok := db.TxFrom(ctx)
	if !ok {
		return nil, errors.New("auth: push token outside the request transaction")
	}
	q := tenantq.New(tx)
	id, err := q.UpsertPushToken(ctx, p.SessionID, in.Body.Platform, in.Body.FCMToken, in.Body.AppVersion)
	if err != nil {
		return nil, fmt.Errorf("auth: push token: %w", err)
	}
	if err := q.RetireOtherPushTokens(ctx, p.SessionID, id); err != nil {
		return nil, fmt.Errorf("auth: push token: %w", err)
	}
	return nil, nil
}

// ── logout and me ──

type logoutOut struct {
	SetCookie []http.Cookie `header:"Set-Cookie"`
}

// logout runs behind SessionOnly, not the request transaction: a read-only session (suspended school, support
// view) must still be able to end itself.
func logout(ctx context.Context, _ *struct{}) (*logoutOut, error) {
	d, err := depsFrom(ctx)
	if err != nil {
		return nil, err
	}
	p, ok := FromContext(ctx)
	if !ok {
		return nil, ErrUnauthenticated
	}
	if err := d.DB.InTenant(ctx, p.TenantID, func(tx pgx.Tx) error {
		return tenantq.New(tx).RevokeSession(ctx, p.SessionID)
	}); err != nil {
		return nil, fmt.Errorf("auth: logout: %w", err)
	}
	out := &logoutOut{}
	if p.Client == ClientWeb {
		out.SetCookie = []http.Cookie{sessionCookie("", -1), csrfCookie("", -1)}
	}
	return out, nil
}

type meOut struct{ Body MeBody }

func me(ctx context.Context, _ *struct{}) (*meOut, error) {
	p, ok := FromContext(ctx)
	if !ok {
		return nil, ErrUnauthenticated
	}
	return &meOut{Body: meFrom(p)}, nil
}

// SessionOnly authenticates without opening the request transaction (logout).
func (d Deps) SessionOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res, err := d.Resolve(r)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(res.Ctx))
	})
}
