package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"

	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/store/tenantq"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/argon2id"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/httpx"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/ratelimit"
)

// PathSetPassword is pre-session: the one-time link is the credential (docs/05 §10).
const PathSetPassword = "/api/auth/set-password" // #nosec G101 -- a URL path, not a credential

// ErrLinkInvalid: one answer for a link that is malformed, used, replaced, expired, or for a user who can't sign in.
var ErrLinkInvalid = httpx.NewError(http.StatusGone, "link_invalid",
	"This link has expired or was already used. Ask your school administrator or Previa for a new one.")

func registerSetPassword(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "auth-set-password", Method: http.MethodPost, Path: PathSetPassword, Tags: []string{"auth"},
		Summary:       "Portal staff: set a password from a one-time link (72 h, single use); then sign in as usual",
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusForbidden, http.StatusGone, http.StatusTooManyRequests},
	}, setPassword)
}

type setPasswordIn struct {
	Origin string `header:"Origin"`
	Body   struct {
		Token    string `json:"token" pattern:"^[1-9][0-9]{0,17}\\.[A-Za-z0-9_-]{43}$" doc:"from the link's #fragment"`
		Password string `json:"password" minLength:"10" maxLength:"128"`
	}
	client clientInfo
}

func (in *setPasswordIn) Resolve(ctx huma.Context) []error { in.client.resolve(ctx); return nil }

// setPassword consumes the link and sets the user's password in one transaction under the link's tenant (parsed
// from the token, looked up under RLS: a forged tenant id finds no row). Every session of the user ends.
func setPassword(ctx context.Context, in *setPasswordIn) (*struct{}, error) {
	d, err := depsFrom(ctx)
	if err != nil {
		return nil, err
	}
	// No session exists, so SameSite can't protect this request: the Origin must be the portal's.
	if !d.originAllowed(in.Origin) {
		return nil, ErrCSRF
	}
	if !d.Limits.allow(func(l *Limits) *ratelimit.Limiter { return l.login }, in.client.ip) {
		return nil, ErrRateLimited
	}
	tenantID, hash, ok := parseToken(in.Body.Token)
	if !ok {
		return nil, ErrLinkInvalid
	}
	// Look the link up read-only first, so a bogus link never takes an Argon2 slot from real sign-ins. The hash then
	// runs outside any transaction; UseSecretLink below re-checks the link and consumes it once.
	var live bool
	if err := d.DB.InTenantReadOnly(ctx, tenantID, func(tx pgx.Tx) error {
		var err error
		live, err = tenantq.New(tx).SecretLinkIsLive(ctx, hash)
		return err
	}); err != nil {
		return nil, fmt.Errorf("auth: set password: %w", err)
	}
	if !live {
		return nil, ErrLinkInvalid
	}
	encoded, err := hashPassword(ctx, in.Body.Password)
	if err != nil {
		return nil, err
	}
	err = d.DB.InTenant(ctx, tenantID, func(tx pgx.Tx) error {
		q := tenantq.New(tx)
		link, err := q.UseSecretLink(ctx, hash)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrLinkInvalid
		}
		if err != nil {
			return err
		}
		if err := q.SetStaffPassword(ctx, &encoded, link.UserID); err != nil {
			return err
		}
		if err := q.SpendSecretLinks(ctx, link.UserID); err != nil {
			return err
		}
		if err := q.RevokeUserSessions(ctx, &link.UserID); err != nil {
			return err
		}
		rid := httpx.RequestID(ctx)
		return q.InsertAudit(ctx, tenantq.InsertAuditParams{
			UserID: &link.UserID, Action: "user.password_set", Entity: "user", EntityID: &link.UserID,
			Summary: "Set a password from a one-time link", Ip: in.client.ip, RequestID: &rid,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("auth: set password: %w", err)
	}
	return &struct{}{}, nil
}

// hashPassword is Argon2id behind the same two-slot semaphore as verify.
func hashPassword(ctx context.Context, plain string) (string, error) {
	select {
	case hashSem <- struct{}{}:
	case <-ctx.Done():
		return "", fmt.Errorf("auth: wait for hash slot: %w", ctx.Err())
	}
	defer func() { <-hashSem }()
	h, err := argon2id.Hash(plain)
	if err != nil {
		return "", fmt.Errorf("auth: hash: %w", err)
	}
	return h, nil
}
