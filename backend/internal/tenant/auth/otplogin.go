package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/shared/otp"
	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/store/tenantq"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/clock"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/httpx"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/ratelimit"
)

// Parent OTP limits (docs/05 §10). The per-mobile and per-school limits are counted in the database and answered
// with the same 202 as a sent code (no oracle); the per-IP limit is in-process and answered with 429, which says
// nothing about the mobile.
const (
	otpMaxPerMobileHour   = 3
	otpMaxPerSchoolMinute = 10
	// Launch window (not live yet, or live for under 14 days): every parent signs in in the same few days. SMS
	// only ever go to invited numbers at 3/mobile/hour, so the per-school limit caps cost, not toll fraud.
	otpMaxPerSchoolMinuteLaunch = 60
	otpMaxAttempts              = 5

	// Per client address (CGNAT puts many parents behind one IP, so these are loose).
	otpRequestsPerIP = 30
	otpVerifiesPerIP = 60
	otpIPWindow      = 10 * time.Minute

	// Per app session: the app registers after sign-in and when FCM rotates the token, so this is never reached
	// by a real phone; it stops one session from filling push_tokens with retired rows.
	pushTokensPerSession = 20
	pushTokenWindow      = time.Hour

	// Staff / crew sign-in (docs/05 §10). A school's office or a bus depot puts its staff behind one IP, so the
	// per-IP request limit is loose; it is answered with 429, which says nothing about the mobile.
	loginsPerIP   = 60
	loginIPWindow = 10 * time.Minute
)

// loginBackoff is per school code + mobile + client address, so an attacker's failures slow down only the
// attacker's address. The database adds a per-user cap over all addresses (loginFailuresPerDay), so guessing stays
// bounded however many addresses are used; locking a user out takes loginFailuresPerDay/PerDay addresses at once.
var loginBackoff = ratelimit.BackoffPolicy{
	Free: 5, Base: 30 * time.Second, Max: 15 * time.Minute, PerDay: 10, Day: 24 * time.Hour,
}

// loginFailuresPerDay is the database cap per user over every address (ClaimLoginAttempt). For a 6-digit PIN that
// is one chance in 20,000 a day.
const loginFailuresPerDay = 50

var (
	errBadCode = httpx.NewError(http.StatusUnauthorized, "bad_credentials",
		"The school code, mobile number or code is not right.")
	// ErrRateLimited: too many sign-in requests from one network.
	ErrRateLimited = httpx.NewError(http.StatusTooManyRequests, "rate_limited",
		"Too many tries. Please wait a few minutes and try again.")
)

// Limits are the in-process per-IP limiters of the unauthenticated endpoints and the sign-in backoff. A nil *Limits
// allows all.
type Limits struct {
	otpRequest, otpVerify *ratelimit.Limiter
	login                 *ratelimit.Limiter // portal password and driver PIN, per IP
	loginBackoff          *ratelimit.Backoff // per school code + mobile + IP
	pushToken             *ratelimit.Limiter // keyed by session id
}

// NewLimits returns the per-IP and per-session limits for one API process.
func NewLimits(c clock.Clock) *Limits {
	return &Limits{
		otpRequest:   ratelimit.New(c, otpRequestsPerIP, otpIPWindow),
		otpVerify:    ratelimit.New(c, otpVerifiesPerIP, otpIPWindow),
		login:        ratelimit.New(c, loginsPerIP, loginIPWindow),
		loginBackoff: ratelimit.NewBackoff(c, loginBackoff),
		pushToken:    ratelimit.New(c, pushTokensPerSession, pushTokenWindow),
	}
}

// loginKey: the school code as typed (lower-cased) so an unknown school or mobile backs off exactly like a known one.
func loginKey(schoolCode, mobile string, ip *netip.Addr) string {
	return strings.ToLower(schoolCode) + "|" + mobile + "|" + ratelimit.IPKey(ip)
}

func (l *Limits) claimLogin(key string) bool { return l == nil || l.loginBackoff.Claim(key) }

func (l *Limits) loginSucceeded(key string) {
	if l != nil {
		l.loginBackoff.Succeeded(key)
	}
}

func (l *Limits) allowPushToken(sessionID int64) bool {
	return l == nil || l.pushToken.Allow(strconv.FormatInt(sessionID, 10))
}

func (l *Limits) allow(which func(*Limits) *ratelimit.Limiter, ip *netip.Addr) bool {
	return l == nil || which(l).Allow(ratelimit.IPKey(ip))
}

// schoolByCode is login step 1 for every client. found=false for an unknown code.
func schoolByCode(ctx context.Context, d Deps, code string) (school tenantq.TenantByCodeRow, found bool, err error) {
	err = d.DB.BeforeTenant(ctx, func(tx pgx.Tx) error {
		var err error
		school, err = tenantq.New(tx).TenantByCode(ctx, strings.ToLower(code))
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		found = err == nil
		return err
	})
	if err != nil {
		return school, false, fmt.Errorf("auth: school: %w", err)
	}
	return school, found, nil
}

type otpRequest struct {
	SchoolCode string
	Mobile     string
	IP         *netip.Addr
}

// requestOTP issues a code. Every mobile in a known school gets a challenge, so the work and the answer are the same
// whether or not the mobile is a guardian's; only an invited, app-enabled guardian's challenge gets an SMS, queued
// in the same transaction for the worker (no outbound HTTP in requests, invariant 9). The caller answers 202 to
// every nil return.
//
// Timing (docs/05 §10): within a school every mobile does the same lookups and the same seal; an invited guardian's
// request differs only by one outbox INSERT in the same commit (well under a millisecond, below network jitter, and
// sampled at most 3 times an hour per mobile). An unknown school code returns without a transaction: school codes
// are printed on invite links and QR codes, so they aren't secret (as for staff login).
func requestOTP(ctx context.Context, d Deps, req otpRequest) error {
	if !d.Limits.allow(func(l *Limits) *ratelimit.Limiter { return l.otpRequest }, req.IP) {
		return ErrRateLimited
	}
	school, found, err := schoolByCode(ctx, d, req.SchoolCode)
	if err != nil || !found {
		return err
	}
	code, err := otp.NewCode()
	if err != nil {
		return err
	}
	limited := ""
	err = d.DB.InTenant(ctx, school.TenantID, func(tx pgx.Tx) error {
		q := tenantq.New(tx)
		if err := q.LockOTPIssue(ctx); err != nil {
			return err
		}
		n, err := q.OTPIssueCounts(ctx, req.Mobile)
		if err != nil {
			return err
		}
		schoolMax := otpMaxPerSchoolMinute
		if n.LaunchWindow {
			schoolMax = otpMaxPerSchoolMinuteLaunch
		}
		// Over a limit: nothing is issued, and a code already sent stays valid.
		switch {
		case n.MobileHour >= otpMaxPerMobileHour:
			limited = "mobile"
			return nil
		case int(n.SchoolMinute) >= schoolMax:
			limited = "school"
			return nil
		}
		if err := q.EndLiveOTPs(ctx, req.Mobile); err != nil {
			return err
		}
		id, err := q.CreateOTPChallenge(ctx, req.Mobile, d.OTP.Hash(school.TenantID, req.Mobile, code), req.IP)
		if err != nil {
			return err
		}
		g, err := q.GuardianForLogin(ctx, req.Mobile)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		// Sealed for every mobile, so the crypto work doesn't tell guardians apart; only theirs is queued.
		key := "otp:" + strconv.FormatInt(id, 10)
		box, sealErr := d.OTP.Seal(code, otp.BoxAAD(school.TenantID, key))
		if sealErr != nil {
			return sealErr
		}
		if err != nil || !g.AppEnabled {
			return nil
		}
		return q.QueueOTPSMS(ctx, &g.ID, box, key)
	})
	if err != nil {
		return fmt.Errorf("auth: otp request: %w", err)
	}
	if limited != "" {
		// No mobile in the log: the count is what an operator needs.
		httpx.Logger(ctx).InfoContext(ctx, "otp request over limit", slog.String("limit", limited),
			slog.Int64("tenant_id", school.TenantID))
	}
	return nil
}

type otpVerify struct {
	SchoolCode string
	Mobile     string
	Code       string
	InstallID  string
	Presented  string // a token the client already holds: revoked, never reused (no fixation)
	IP         *netip.Addr
	UserAgent  string
}

// verifyOTP signs a guardian in. One 401 for unknown school, no or expired challenge, too many tries, wrong code and
// a mobile that is not an app-enabled guardian. The try is counted (and committed) before the code is compared, on a
// locked row, so parallel guesses can't exceed 5 per code.
func verifyOTP(ctx context.Context, d Deps, req otpVerify) (*loginResult, error) {
	if !d.Limits.allow(func(l *Limits) *ratelimit.Limiter { return l.otpVerify }, req.IP) {
		return nil, ErrRateLimited
	}
	school, found, err := schoolByCode(ctx, d, req.SchoolCode)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errBadCode
	}
	var res *loginResult
	var outcome error = errBadCode // returned after the commit, so a counted try is never rolled back
	err = d.DB.InTenant(ctx, school.TenantID, func(tx pgx.Tx) error {
		q := tenantq.New(tx)
		ch, err := q.LiveOTPForVerify(ctx, req.Mobile)
		if errors.Is(err, pgx.ErrNoRows) {
			return q.AssignXactID(ctx) // commit costs the same as a counted try
		}
		if err != nil {
			return err
		}
		if !ch.Live || ch.Attempts >= otpMaxAttempts {
			return q.AssignXactID(ctx)
		}
		if err := q.CountOTPAttempt(ctx, ch.ID); err != nil {
			return err
		}
		if !d.OTP.Matches(ch.CodeHash, school.TenantID, req.Mobile, req.Code) {
			return nil
		}
		if err := q.ConsumeOTP(ctx, ch.ID); err != nil {
			return err
		}
		g, err := q.GuardianForLogin(ctx, req.Mobile)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if !g.AppEnabled {
			return nil
		}
		// Only now, with the code proven, does the school's status show (no oracle on school codes).
		if school.Status == "cancelled" {
			outcome = ErrSchoolClosed
			return nil
		}
		res, err = startParentSession(ctx, q, req, school, g)
		if err != nil {
			return err
		}
		outcome = nil
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("auth: otp verify: %w", err)
	}
	if outcome != nil {
		return nil, outcome
	}
	return res, nil
}

// startParentSession runs in verifyOTP's transaction: the code is consumed and the session created together.
func startParentSession(ctx context.Context, q *tenantq.Queries, req otpVerify, school tenantq.TenantByCodeRow,
	g tenantq.GuardianForLoginRow) (*loginResult, error) {
	token, hash, err := newToken(school.TenantID)
	if err != nil {
		return nil, err
	}
	res := &loginResult{Token: token, Principal: Principal{
		TenantID: school.TenantID, Client: ClientParentApp, GuardianID: g.ID, Role: "parent", Name: g.Name,
		ReadOnly: school.Status == "suspended",
	}}
	// No fixation: a token the client already holds for this school is ended (see startSession). A token for
	// another school is left alone: its session is under that school's RLS and can't be fixed onto this one; it
	// expires on its own or by logout.
	if t, oldHash, ok := parseToken(req.Presented); ok && t == school.TenantID {
		if err := q.RevokeSessionByHash(ctx, oldHash); err != nil {
			return nil, err
		}
	}
	// One session per guardian per install; the guardian's other phones stay signed in.
	if err := q.RevokeGuardianInstallSessions(ctx, &g.ID, &req.InstallID); err != nil {
		return nil, err
	}
	id, err := q.CreateSession(ctx, tenantq.CreateSessionParams{
		TokenHash: hash, GuardianID: &g.ID, Client: ClientParentApp, InstallID: &req.InstallID,
		ReadOnly: res.Principal.ReadOnly, LifetimeSecs: float64(appSessionSecs), Ip: req.IP,
		UserAgent: truncate(req.UserAgent, 300),
	})
	if err != nil {
		return nil, err
	}
	res.Principal.SessionID = id
	s, err := q.CurrentSchool(ctx)
	if err != nil {
		return nil, err
	}
	res.Principal.TenantCode, res.Principal.TenantName = s.Code, s.DisplayName
	return res, nil
}
