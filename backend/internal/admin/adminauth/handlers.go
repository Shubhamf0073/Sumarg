package adminauth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/jackc/pgx/v5"

	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/store/platformq"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/httpx"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/ratelimit"
)

// maxCredentialBytes bounds a browser's credential response (a passkey response is a few KB).
const maxCredentialBytes = 64 << 10

// Register adds the admin auth operations (docs/04 "Admin": Auth).
func Register(api huma.API) {
	tags := []string{"admin-auth"}
	huma.Register(api, huma.Operation{
		OperationID: "admin-login-begin", Method: http.MethodPost, Path: PathLoginBegin, Tags: tags,
		Summary: "Admin sign-in, step 1: options for navigator.credentials.get (discoverable passkey)",
		Errors:  []int{http.StatusForbidden, http.StatusTooManyRequests},
	}, loginBegin)
	huma.Register(api, huma.Operation{
		OperationID: "admin-login-finish", Method: http.MethodPost, Path: PathLoginFinish, Tags: tags,
		Summary:      "Admin sign-in, step 2: the browser's assertion → admin session cookie",
		MaxBodyBytes: maxCredentialBytes,
		Errors:       []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusTooManyRequests},
	}, loginFinish)
	huma.Register(api, huma.Operation{
		OperationID: "admin-enrol-begin", Method: http.MethodPost, Path: PathEnrolBegin, Tags: tags,
		Summary: "Passkey enrolment, step 1: the one-time link's token → options for navigator.credentials.create",
		Errors:  []int{http.StatusForbidden, http.StatusGone, http.StatusTooManyRequests},
	}, enrolBegin)
	huma.Register(api, huma.Operation{
		OperationID: "admin-enrol-finish", Method: http.MethodPost, Path: PathEnrolFinish, Tags: tags,
		Summary:      "Passkey enrolment, step 2: the new credential → passkey saved, link used, admin session cookie",
		MaxBodyBytes: maxCredentialBytes,
		Errors:       []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusGone, http.StatusTooManyRequests},
	}, enrolFinish)
	huma.Register(api, huma.Operation{
		OperationID: "admin-logout", Method: http.MethodPost, Path: PathLogout, Tags: tags,
		Summary:       "Admin sign-out: ends this session",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusUnauthorized, http.StatusForbidden},
	}, logout)
	huma.Register(api, huma.Operation{
		OperationID: "admin-me", Method: http.MethodGet, Path: PathMe, Tags: tags,
		Summary: "Which Previa staff member is signed in",
		Errors:  []int{http.StatusUnauthorized},
	}, me)
}

// ── shapes ──

// AdminMeBody is the signed-in staff member.
type AdminMeBody struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role" enum:"superadmin,support,billing,installer"`
}

type client struct {
	ip        *netip.Addr
	userAgent string
}

func (c *client) resolve(ctx huma.Context) {
	c.userAgent = ctx.Header("User-Agent")
	var trusted []netip.Prefix
	if d, err := depsFrom(ctx.Context()); err == nil {
		trusted = d.TrustedProxies
	}
	c.ip = httpx.ClientIP(ctx.RemoteAddr(), ctx.Header("X-Forwarded-For"), trusted)
}

type beginOut struct {
	SetCookie []http.Cookie  `header:"Set-Cookie"`
	Body      map[string]any `doc:"pass .publicKey to navigator.credentials (base64url fields decoded)"`
}

// credentialIn is the browser's PublicKeyCredential as JSON (base64url fields), read raw for the library.
type credentialIn struct {
	Origin    string         `header:"Origin"`
	Ceremony  string         `cookie:"__Host-sf_admin_cer"`
	Presented string         `cookie:"__Host-sf_admin"`
	Body      map[string]any `doc:"PublicKeyCredential.toJSON()"`
	RawBody   []byte
	client    client
}

func (in *credentialIn) Resolve(ctx huma.Context) []error { in.client.resolve(ctx); return nil }

type signedInOut struct {
	SetCookie []http.Cookie `header:"Set-Cookie"`
	Body      AdminMeBody
}

// ── sign-in ──

type loginBeginIn struct {
	Origin string `header:"Origin"`
	client client
}

func (in *loginBeginIn) Resolve(ctx huma.Context) []error { in.client.resolve(ctx); return nil }

func loginBegin(ctx context.Context, in *loginBeginIn) (*beginOut, error) {
	d, err := preSession(ctx, in.Origin, in.client.ip)
	if err != nil {
		return nil, err
	}
	assertion, session, err := d.WebAuthn.BeginDiscoverableLogin(
		webauthn.WithUserVerification(protocol.VerificationRequired))
	if err != nil {
		return nil, fmt.Errorf("adminauth: begin login: %w", err)
	}
	return d.begun(ctx, ceremony{kind: kindLogin, session: *session}, assertion)
}

func loginFinish(ctx context.Context, in *credentialIn) (*signedInOut, error) {
	d, err := preSession(ctx, in.Origin, in.client.ip)
	if err != nil {
		return nil, err
	}
	cer, ok := d.Ceremonies.take(in.Ceremony, kindLogin)
	if !ok {
		return nil, ErrSignInFailed
	}
	parsed, err := protocol.ParseCredentialRequestResponseBytes(in.RawBody)
	if err != nil {
		refused(ctx, "unparseable assertion", err)
		return nil, ErrSignInFailed
	}
	var (
		out    *signedInOut
		cloned bool
	)
	var row platformq.GetPasskeyForLoginRow
	err = d.DB(ctx, func(tx pgx.Tx) error {
		q := platformq.New(tx)
		lookup := func(rawID, handle []byte) (webauthn.User, error) {
			var err error
			row, err = q.GetPasskeyForLogin(ctx, rawID)
			if err != nil {
				return nil, err
			}
			if !bytes.Equal(handle, userHandle(row.UserID)) {
				return nil, errors.New("user handle does not match the passkey's user")
			}
			var cred webauthn.Credential
			if err := json.Unmarshal(row.Credential, &cred); err != nil {
				return nil, fmt.Errorf("stored credential: %w", err)
			}
			return &adminUser{id: row.UserID, name: row.Name, email: row.Email, creds: []webauthn.Credential{cred}}, nil
		}
		_, cred, err := d.WebAuthn.ValidatePasskeyLogin(lookup, cer.session, parsed)
		if err != nil {
			refused(ctx, "assertion not verified", err)
			return ErrSignInFailed
		}
		if cred.Authenticator.CloneWarning {
			// The sign count went backwards: two copies of this authenticator may exist. Refuse and tell a human.
			httpx.Logger(ctx).WarnContext(ctx, "admin passkey sign count went backwards; sign-in refused",
				slog.Int64("platform_user_id", row.UserID), slog.Int64("passkey_id", row.ID))
			cloned = true
			return ErrSignInFailed
		}
		stored, err := json.Marshal(cred)
		if err != nil {
			return fmt.Errorf("adminauth: credential: %w", err)
		}
		if err := q.UsePasskey(ctx, stored, row.ID); err != nil {
			return err
		}
		out, err = d.startSession(ctx, q, in, row.ID, AdminMeBody{ID: row.UserID, Name: row.Name, Email: row.Email,
			Role: row.Role})
		return err
	})
	if cloned {
		// The sign-in transaction rolled back; the refusal is recorded in its own.
		if aerr := d.DB(ctx, func(tx pgx.Tx) error {
			return audit(ctx, platformq.New(tx), in.client, row.UserID, row.Name, "passkey.clone_refused",
				"platform_passkey", row.ID, "Sign-in refused: the passkey's sign count went backwards (possible copy)")
		}); aerr != nil {
			return nil, aerr
		}
	}
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ── enrolment ──

type enrolBeginIn struct {
	Origin string `header:"Origin"`
	Body   struct {
		Token string `json:"token" pattern:"^[A-Za-z0-9_-]{43}$" doc:"from the enrolment link's #fragment"`
	}
	client client
}

func (in *enrolBeginIn) Resolve(ctx huma.Context) []error { in.client.resolve(ctx); return nil }

func enrolBegin(ctx context.Context, in *enrolBeginIn) (*beginOut, error) {
	d, err := preSession(ctx, in.Origin, in.client.ip)
	if err != nil {
		return nil, err
	}
	hash, ok := hashToken(in.Body.Token)
	if !ok {
		return nil, ErrEnrolment
	}
	var (
		user        *adminUser
		enrolmentID int64
	)
	err = d.DB(ctx, func(tx pgx.Tx) error {
		q := platformq.New(tx)
		e, err := q.GetEnrolment(ctx, hash)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrEnrolment
		}
		if err != nil {
			return err
		}
		enrolmentID = e.ID
		user, err = loadUser(ctx, q, e.UserID, e.Name, e.Email)
		return err
	})
	if err != nil {
		return nil, err
	}
	creation, session, err := d.WebAuthn.BeginRegistration(user,
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			ResidentKey:        protocol.ResidentKeyRequirementRequired,
			RequireResidentKey: protocol.ResidentKeyRequired(),
			UserVerification:   protocol.VerificationRequired,
		}),
		webauthn.WithExclusions(user.exclusions()),
		webauthn.WithConveyancePreference(protocol.PreferNoAttestation))
	if err != nil {
		return nil, fmt.Errorf("adminauth: begin registration: %w", err)
	}
	return d.begun(ctx, ceremony{kind: kindEnrol, session: *session, userID: user.id, enrolmentID: enrolmentID},
		creation)
}

func enrolFinish(ctx context.Context, in *credentialIn) (*signedInOut, error) {
	d, err := preSession(ctx, in.Origin, in.client.ip)
	if err != nil {
		return nil, err
	}
	cer, ok := d.Ceremonies.take(in.Ceremony, kindEnrol)
	if !ok {
		return nil, ErrSignInFailed
	}
	parsed, err := protocol.ParseCredentialCreationResponseBytes(in.RawBody)
	if err != nil {
		refused(ctx, "unparseable registration", err)
		return nil, ErrSignInFailed
	}
	var out *signedInOut
	err = d.DB(ctx, func(tx pgx.Tx) error {
		q := platformq.New(tx)
		// Consume the link first: if anything below fails the transaction rolls back and the link still works.
		n, err := q.UseEnrolment(ctx, cer.enrolmentID, cer.userID)
		if err != nil {
			return err
		}
		if n != 1 {
			return ErrEnrolment
		}
		u, err := q.GetPlatformUser(ctx, cer.userID)
		if err != nil {
			return err
		}
		user, err := loadUser(ctx, q, u.ID, u.Name, u.Email)
		if err != nil {
			return err
		}
		cred, err := d.WebAuthn.CreateCredential(user, cer.session, parsed)
		if err != nil {
			refused(ctx, "registration not verified", err)
			return ErrSignInFailed
		}
		stored, err := json.Marshal(cred)
		if err != nil {
			return fmt.Errorf("adminauth: credential: %w", err)
		}
		passkeyID, err := q.CreatePasskey(ctx, user.id, cred.ID, stored, passkeyLabel(in.client.userAgent))
		if err != nil {
			return err
		}
		if err := audit(ctx, q, in.client, user.id, user.name, "passkey.enrolled", "platform_passkey", passkeyID,
			"Registered a passkey from an enrolment link"); err != nil {
			return err
		}
		out, err = d.startSession(ctx, q, in, passkeyID, AdminMeBody{ID: u.ID, Name: u.Name, Email: u.Email,
			Role: u.Role})
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ── session ──

type logoutIn struct {
	Session string `cookie:"__Host-sf_admin"`
	client  client
}

func (in *logoutIn) Resolve(ctx huma.Context) []error { in.client.resolve(ctx); return nil }

type logoutOut struct {
	SetCookie []http.Cookie `header:"Set-Cookie"`
}

func logout(ctx context.Context, in *logoutIn) (*logoutOut, error) {
	d, err := depsFrom(ctx)
	if err != nil {
		return nil, err
	}
	hash, ok := hashToken(in.Session)
	if !ok {
		return nil, ErrUnauthenticated
	}
	if err := d.DB(ctx, func(tx pgx.Tx) error {
		q := platformq.New(tx)
		s, err := q.RevokePlatformSessionForLogout(ctx, hash)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // already ended: nothing to record, the cookies are cleared anyway
		}
		if err != nil {
			return err
		}
		return audit(ctx, q, in.client, s.UserID, s.Name, "admin.signed_out", "platform_session", s.ID,
			"Signed out of /admin")
	}); err != nil {
		return nil, err
	}
	return &logoutOut{SetCookie: []http.Cookie{sessionCookie("", -1), csrfCookie("", -1)}}, nil
}

type meOut struct{ Body AdminMeBody }

func me(ctx context.Context, _ *struct{}) (*meOut, error) {
	p := FromContext(ctx)
	if p == nil {
		return nil, ErrUnauthenticated
	}
	return &meOut{Body: AdminMeBody{ID: p.UserID, Name: p.Name, Email: p.Email, Role: p.Role}}, nil
}

// ── helpers ──

// preSession is the request's Deps after the checks every pre-session step shares: no session cookie exists yet, so
// SameSite can't protect the request and the Origin must be ours; and one address gets 30 steps per 10 minutes.
func preSession(ctx context.Context, origin string, ip *netip.Addr) (Deps, error) {
	d, err := depsFrom(ctx)
	if err != nil {
		return Deps{}, err
	}
	if !d.originAllowed(origin) {
		return Deps{}, ErrCSRF
	}
	if d.Limit != nil && !d.Limit.Allow(ratelimit.IPKey(ip)) {
		return Deps{}, ErrTooMany
	}
	return d, nil
}

// begun stores the ceremony and returns the browser's options with the ceremony cookie.
func (d Deps) begun(ctx context.Context, c ceremony, options any) (*beginOut, error) {
	id, err := d.Ceremonies.put(c)
	if errors.Is(err, errCeremoniesFull) {
		// Real admins get 429 until ceremonies expire (≤ 5 min): docs/06 §9. The alert key is stable for log alerts.
		httpx.Logger(ctx).ErrorContext(ctx, "admin passkey ceremony store full; refusing new ceremonies",
			slog.String("alert", "admin_ceremonies_full"))
		return nil, ErrTooMany
	}
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(options)
	if err != nil {
		return nil, fmt.Errorf("adminauth: options: %w", err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, fmt.Errorf("adminauth: options: %w", err)
	}
	return &beginOut{SetCookie: []http.Cookie{ceremonyCookie(id, int(ceremonyTTL.Seconds()))}, Body: body}, nil
}

// startSession creates the admin session in the ceremony's transaction, revoking any admin session the browser
// presented (no fixation), and returns the cookies.
func (d Deps) startSession(ctx context.Context, q *platformq.Queries, in *credentialIn, passkeyID int64,
	who AdminMeBody) (*signedInOut, error) {
	if old, ok := hashToken(in.Presented); ok {
		if err := q.RevokePlatformSessionByHash(ctx, old); err != nil {
			return nil, err
		}
	}
	token, hash, err := NewToken()
	if err != nil {
		return nil, err
	}
	csrf, _, err := NewToken()
	if err != nil {
		return nil, err
	}
	ua := in.client.userAgent
	sessionID, err := q.CreatePlatformSession(ctx, platformq.CreatePlatformSessionParams{
		TokenHash: hash, PlatformUserID: who.ID, PasskeyID: passkeyID, LifetimeHours: int32(d.SessionHours),
		Ip: in.client.ip, UserAgent: &ua,
	})
	if err != nil {
		return nil, err
	}
	if err := q.MarkPlatformLogin(ctx, who.ID); err != nil {
		return nil, err
	}
	if err := audit(ctx, q, in.client, who.ID, who.Name, "admin.signed_in", "platform_session", sessionID,
		"Signed in to /admin with a passkey"); err != nil {
		return nil, err
	}
	maxAge := d.SessionHours * 3600
	return &signedInOut{
		SetCookie: []http.Cookie{sessionCookie(token, maxAge), csrfCookie(csrf, maxAge), ceremonyCookie("", -1)},
		Body:      who,
	}, nil
}

func loadUser(ctx context.Context, q *platformq.Queries, id int64, name, email string) (*adminUser, error) {
	rows, err := q.ListUserCredentials(ctx, id)
	if err != nil {
		return nil, err
	}
	u := &adminUser{id: id, name: name, email: email, creds: make([]webauthn.Credential, 0, len(rows))}
	for _, raw := range rows {
		var c webauthn.Credential
		if err := json.Unmarshal(raw, &c); err != nil {
			return nil, fmt.Errorf("adminauth: stored credential: %w", err)
		}
		u.creds = append(u.creds, c)
	}
	return u, nil
}

// passkeyLabel names the passkey after the browser it was made in, for the admin users screen.
func passkeyLabel(ua string) string {
	if len(ua) > 120 {
		ua = ua[:120]
	}
	return ua
}

// audit appends a platform_audit_log row for a staff member's own sign-in event.
func audit(ctx context.Context, q *platformq.Queries, c client, userID int64, name, action, entity string,
	entityID int64, summary string) error {
	rid := httpx.RequestID(ctx)
	return q.InsertPlatformAudit(ctx, platformq.InsertPlatformAuditParams{
		PlatformUserID: &userID, ActorName: name, Action: action, Entity: entity, EntityID: &entityID,
		Summary: summary, Ip: c.ip, RequestID: &rid,
	})
}

// refused logs why a ceremony failed (the library's reason, never the credential) and answers one 401 for all.
func refused(ctx context.Context, what string, err error) {
	httpx.Logger(ctx).InfoContext(ctx, "admin passkey "+what, slog.Any("error", err))
}
