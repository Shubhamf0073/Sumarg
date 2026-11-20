package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"sync"

	"github.com/jackc/pgx/v5"

	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/store/tenantq"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/argon2id"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/httpx"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/ratelimit"
)

// Session lifetimes (docs/05 §10). Portal idle (12 h) is enforced in GetSession; app sessions renew on use.
const (
	webSessionSecs = 7 * 24 * 60 * 60
	appSessionSecs = 90 * 24 * 60 * 60
)

// One answer for unknown school, unknown mobile, wrong secret, wrong kind of account, inactive user and backoff.
var (
	errBadPassword = httpx.NewError(http.StatusUnauthorized, "bad_credentials",
		"The school code, mobile number or password is not right.")
	errBadPIN = httpx.NewError(http.StatusUnauthorized, "bad_credentials",
		"The school code, mobile number or PIN is not right.")
)

type secretKind int

const (
	password secretKind = iota // portal: owner, admin, transport_manager
	pin                        // driver app: driver, attendant
)

type loginRequest struct {
	Kind       secretKind
	SchoolCode string
	Mobile     string
	Secret     string
	InstallID  string // pin only
	Presented  string // a token the client already holds: revoked, never reused (no fixation)
	IP         *netip.Addr
	UserAgent  string
}

type loginResult struct {
	Token     string
	Principal Principal
}

// hashSem keeps Argon2 (19 MiB, ~50 ms each) to two at a time per process (.claude/rules/backend.md).
var hashSem = make(chan struct{}, 2)

// dummyHash is verified when there is no real hash to check, so an unknown school or mobile costs the same time.
var dummyHash = sync.OnceValues(func() (string, error) { return argon2id.Hash("sumarg-no-such-user") })

func verify(ctx context.Context, secret, encoded string) (bool, error) {
	select {
	case hashSem <- struct{}{}:
	case <-ctx.Done():
		return false, fmt.Errorf("auth: wait for hash slot: %w", ctx.Err())
	}
	defer func() { <-hashSem }()
	ok, err := argon2id.Verify(secret, encoded)
	if err != nil {
		return false, fmt.Errorf("auth: verify: %w", err)
	}
	return ok, nil
}

func roleFits(k secretKind, role string) bool {
	switch role {
	case "owner", "admin", "transport_manager":
		return k == password
	case "driver", "attendant":
		return k == pin
	}
	return false
}

// login signs a staff (password) or crew (PIN) user in. The tenant comes from the school code through
// tenant_by_code; everything after that runs under RLS for that tenant.
func login(ctx context.Context, d Deps, req loginRequest) (*loginResult, error) {
	badCreds := errBadPassword
	if req.Kind == pin {
		badCreds = errBadPIN
	}
	if !d.Limits.allow(func(l *Limits) *ratelimit.Limiter { return l.login }, req.IP) {
		return nil, ErrRateLimited
	}
	// Backoff per school code + mobile + address, claimed before any work (counted as a failure until it succeeds,
	// so parallel guesses can't all get past the 5th). Every mobile, staff or not, backs off the same way, so the
	// quick refusal says nothing about which mobiles are staff.
	key := loginKey(req.SchoolCode, req.Mobile, req.IP)
	if !d.Limits.claimLogin(key) {
		return nil, badCreds
	}

	school, schoolFound, err := schoolByCode(ctx, d, req.SchoolCode)
	if err != nil {
		return nil, err
	}

	// One read-write transaction on every path in a known school: look the user up and, if they may sign in this
	// way, claim the attempt against the user's daily cap over every address (counted as a failure until it
	// succeeds; the check and the count are one statement). When nothing is claimed, the transaction is still given
	// an id so its commit costs the same: the timing doesn't reveal which mobiles are staff. (An unknown school code
	// skips this; school codes are printed on invite links and QR codes, so they aren't secret.)
	var user tenantq.StaffForLoginRow
	var stored *string
	eligible, capReached := false, false
	if schoolFound {
		err = d.DB.InTenant(ctx, school.TenantID, func(tx pgx.Tx) error {
			q := tenantq.New(tx)
			var err error
			user, err = q.StaffForLogin(ctx, req.Mobile)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
			if err == nil {
				stored = user.PasswordHash
				if req.Kind == pin {
					stored = user.PinHash
				}
				eligible = user.IsActive && roleFits(req.Kind, user.Role) && stored != nil
			}
			if eligible {
				n, err := q.ClaimLoginAttempt(ctx, user.ID, loginFailuresPerDay)
				switch {
				case errors.Is(err, pgx.ErrNoRows):
					eligible = false // the daily cap is reached
				case err != nil:
					return err
				case n == loginFailuresPerDay:
					capReached = true // this try is the last one the window allows
				}
			}
			if !eligible {
				return q.AssignXactID(ctx)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("auth: user: %w", err)
		}
	}
	if capReached {
		// Once per window: an operator should know someone is guessing (no mobile in the log).
		httpx.Logger(ctx).WarnContext(ctx, "login daily failure cap reached", slog.Int64("tenant_id", school.TenantID),
			slog.Int64("user_id", user.ID))
	}
	// Not eligible: the dummy hash is verified and the answer is the same 401, after the same work.
	encoded, err := dummyHash()
	if err != nil {
		return nil, fmt.Errorf("auth: dummy hash: %w", err)
	}
	if eligible {
		encoded = *stored
	}
	ok, err := verify(ctx, req.Secret, encoded)
	if err != nil {
		return nil, err
	}
	if !eligible || !ok {
		return nil, badCreds
	}
	d.Limits.loginSucceeded(key)
	// Only now, with the secret proven, does the school's status show (no oracle on school codes).
	if school.Status == "cancelled" {
		return nil, ErrSchoolClosed
	}
	return startSession(ctx, d, req, school, user)
}

func startSession(ctx context.Context, d Deps, req loginRequest, school tenantq.TenantByCodeRow,
	user tenantq.StaffForLoginRow) (*loginResult, error) {
	client, lifetime := ClientWeb, float64(webSessionSecs)
	var installID *string
	if req.Kind == pin {
		client, lifetime, installID = ClientDriverApp, float64(appSessionSecs), &req.InstallID
	}
	token, hash, err := newToken(school.TenantID)
	if err != nil {
		return nil, err
	}
	res := &loginResult{Token: token, Principal: Principal{
		TenantID: school.TenantID, Client: client, UserID: user.ID, Role: user.Role, CampusID: user.CampusID,
		Name: user.Name, MustChangeSecret: user.MustChangeSecret, ReadOnly: school.Status == "suspended",
	}}
	err = d.DB.InTenant(ctx, school.TenantID, func(tx pgx.Tx) error {
		q := tenantq.New(tx)
		if err := q.StaffLoginSucceeded(ctx, user.ID); err != nil {
			return err
		}
		// A session the client already holds is ended, never upgraded (no fixation). Under RLS this only reaches
		// this school's sessions. It may belong to another user of the school: the caller holds its token, so ending
		// it gives them nothing they couldn't do with logout.
		if t, oldHash, ok := parseToken(req.Presented); ok && t == school.TenantID {
			if err := q.RevokeSessionByHash(ctx, oldHash); err != nil {
				return err
			}
		}
		if client == ClientDriverApp { // one app session per crew user: the new phone logs the old one out
			if err := q.RevokeDriverAppSessions(ctx, &user.ID); err != nil {
				return err
			}
		}
		id, err := q.CreateSession(ctx, tenantq.CreateSessionParams{
			TokenHash: hash, UserID: &user.ID, Client: client, InstallID: installID,
			ReadOnly: res.Principal.ReadOnly, LifetimeSecs: lifetime, Ip: req.IP, UserAgent: truncate(req.UserAgent, 300),
		})
		if err != nil {
			return err
		}
		res.Principal.SessionID = id
		s, err := q.CurrentSchool(ctx)
		if err != nil {
			return err
		}
		res.Principal.TenantCode, res.Principal.TenantName = s.Code, s.DisplayName
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("auth: start session: %w", err)
	}
	return res, nil
}

func truncate(s string, n int) *string {
	if len(s) > n {
		s = s[:n]
	}
	return &s
}
