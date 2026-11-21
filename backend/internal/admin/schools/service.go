package schools

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/admin/adminauth"
	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/store/platformq"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/argon2id"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/httpx"
)

// LinkHours is how long a set-password link works (docs/05 §10).
const LinkHours = 72

// listPage is the board's page size.
const listPage = 200

var (
	errNotFound     = httpx.NewError(http.StatusNotFound, "not_found", "No such school.")
	errUserNotFound = httpx.NewError(http.StatusNotFound, "not_found",
		"No such owner or admin login in this school (or it is inactive).")
	errCodeTaken = httpx.NewError(http.StatusConflict, "code_taken",
		"That school code is already used. Pick another one.")
	errStageChanged = httpx.NewError(http.StatusConflict, "stage_changed",
		"Someone moved this school meanwhile. Reload and try again.")
	errCampusDup = httpx.NewError(http.StatusUnprocessableEntity, "campus_code_duplicate",
		"Two campuses have the same code.")
	errAgent = httpx.NewError(http.StatusUnprocessableEntity, "agent_invalid",
		"That agent doesn't exist or is inactive.")
	errDate      = httpx.NewError(http.StatusUnprocessableEntity, "date_invalid", "A date is not valid (YYYY-MM-DD).")
	errLeaveLive = httpx.NewError(http.StatusUnprocessableEntity, "stage_live",
		"A live school stays live. Change its status instead.")
	errUserInactive = httpx.NewError(http.StatusConflict, "user_inactive", "This login is already deactivated.")
	errUserActive   = httpx.NewError(http.StatusConflict, "user_active", "This login is already active.")
	errMobile       = httpx.NewError(http.StatusUnprocessableEntity, "mobile_invalid",
		"A mobile number is not valid (10 digits starting 6–9).")
	errValue = httpx.NewError(http.StatusUnprocessableEntity, "value_invalid",
		"One of the values isn't accepted. Check the form and try again.")
	errDuplicate = httpx.NewError(http.StatusConflict, "duplicate", "Something with these details already exists.")
)

// hashSem keeps Argon2 (19 MiB, ~50 ms) to two at a time (.claude/rules/backend.md).
var hashSem = make(chan struct{}, 2)

// unusableHash is the Argon2id hash of a random secret nobody is told: users.password_hash must be set for an owner,
// and the account can't sign in until the one-time link replaces it.
func unusableHash(ctx context.Context) (string, error) {
	select {
	case hashSem <- struct{}{}:
	case <-ctx.Done():
		return "", fmt.Errorf("schools: wait for hash slot: %w", ctx.Err())
	}
	defer func() { <-hashSem }()
	h, err := argon2id.Hash(rand.Text() + rand.Text())
	if err != nil {
		return "", fmt.Errorf("schools: hash: %w", err)
	}
	return h, nil
}

type created struct {
	id        int64
	stage     string
	periodEnd string
	link      LinkBody
	campusIDs []int64
}

// createSchool saves the whole wizard in one platform transaction: any failure leaves nothing behind.
func createSchool(ctx context.Context, p *adminauth.Principal, ip *netip.Addr, b CreateBody) (*created, error) {
	seen := make(map[string]bool, len(b.Campuses))
	for _, c := range b.Campuses {
		if seen[c.Code] {
			return nil, errCampusDup
		}
		seen[c.Code] = true
	}
	goLive, err := optDate(b.School.GoLiveOn)
	if err != nil {
		return nil, err
	}
	start, err := parseDate(b.Package.PeriodStart)
	if err != nil {
		return nil, err
	}
	end := start.AddDate(1, 0, -1) // 12 months − 1 day (docs/05 §11)
	inPlatform, err := adminauth.Platform(ctx)
	if err != nil {
		return nil, err
	}
	origin, err := adminauth.PortalOrigin(ctx)
	if err != nil {
		return nil, err
	}
	pwHash, err := unusableHash(ctx)
	if err != nil {
		return nil, err
	}
	token, tokenHash, err := adminauth.NewToken()
	if err != nil {
		return nil, err
	}

	res := &created{periodEnd: end.Format(time.DateOnly)}
	err = inPlatform(ctx, func(tx pgx.Tx) error {
		q := platformq.New(tx)
		if b.Package.AgentID != nil {
			active, err := q.AgentIsActive(ctx, *b.Package.AgentID)
			if errors.Is(err, pgx.ErrNoRows) || (err == nil && !active) {
				return errAgent
			}
			if err != nil {
				return err
			}
		}
		s := b.School
		t, err := q.CreateTenant(ctx, platformq.CreateTenantParams{
			Code: s.Code, LegalName: s.LegalName, DisplayName: s.DisplayName, ShortName: s.ShortName, Board: s.Board,
			Gstin: s.GSTIN, Address: s.Address, City: s.City, StateCode: s.StateCode, ContactName: s.ContactName,
			ContactMobile: s.ContactMobile, ContactEmail: s.ContactEmail, GoLiveOn: goLive,
			PricePerStudentPaise: b.Package.PricePerStudentPaise, PriceIncludesGst: b.Package.PriceIncludesGST,
			AgentID: b.Package.AgentID, Notes: s.Notes, CreatedBy: &p.UserID,
		})
		if err != nil {
			return dbError(err)
		}
		res.id, res.stage = t.ID, t.OnboardingStage
		st := b.Settings
		if err := q.CreateTenantSettings(ctx, platformq.CreateTenantSettingsParams{
			TenantID: t.ID, ApproachMinutes: st.ApproachMinutes, DelayAlertMinutes: st.DelayAlertMinutes,
			SpeedLimitKmph: st.SpeedLimitKmph, PhotoOnTap: st.PhotoOnTap, PhotoRetentionDays: st.PhotoRetentionDays,
			PositionRetentionDays: st.PositionRetentionDays, SmsFallback: st.SMSFallback,
		}); err != nil {
			return dbError(err)
		}
		for _, c := range b.Campuses {
			id, err := q.CreateCampus(ctx, platformq.CreateCampusParams{TenantID: t.ID, Code: c.Code, Name: c.Name,
				Address: c.Address, Lat: c.Lat, Lon: c.Lon, GeofenceM: c.GeofenceM})
			if err != nil {
				return dbError(err)
			}
			res.campusIDs = append(res.campusIDs, id)
		}
		ownerID, err := q.CreatePortalUser(ctx, platformq.CreatePortalUserParams{TenantID: t.ID,
			Name: b.Owner.Name, Mobile: b.Owner.Mobile, Email: b.Owner.Email, Role: "owner", PasswordHash: &pwHash})
		if err != nil {
			return dbError(err)
		}
		l, err := q.CreateSecretLink(ctx, tokenHash, LinkHours, t.ID, ownerID)
		if err != nil {
			return dbError(err)
		}
		res.link = LinkBody{UserID: ownerID, URL: linkURL(origin, t.ID, token), ExpiresAt: l.ExpiresAt}
		subID, err := q.CreateSubscription(ctx, platformq.CreateSubscriptionParams{TenantID: t.ID,
			PeriodStart: pgDate(start), PeriodEnd: pgDate(end), StudentsCommitted: b.Package.StudentsCommitted,
			PricePerStudentPaise: b.Package.PricePerStudentPaise, CreatedBy: &p.UserID})
		if err != nil {
			return dbError(err)
		}
		changes := map[string]any{"campuses": len(b.Campuses), "subscription_id": subID,
			"students_committed": b.Package.StudentsCommitted, "price_per_student_paise": b.Package.PricePerStudentPaise,
			"price_includes_gst": b.Package.PriceIncludesGST, "agent_id": b.Package.AgentID,
			"period_start": b.Package.PeriodStart, "owner_user_id": ownerID}
		return audit(ctx, q, p, ip, t.ID, "school.created", "tenant", t.ID,
			"Created school "+s.Code+" in the onboarding wizard (owner set-password link issued)", changes)
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func listSchools(ctx context.Context, after int64) ([]SchoolRow, *int64, error) {
	inPlatform, err := adminauth.PlatformReadOnly(ctx)
	if err != nil {
		return nil, nil, err
	}
	var rows []platformq.ListSchoolsRow
	if err := inPlatform(ctx, func(tx pgx.Tx) error {
		var err error
		rows, err = platformq.New(tx).ListSchools(ctx, after, listPage+1)
		return err
	}); err != nil {
		return nil, nil, err
	}
	var next *int64
	if len(rows) > listPage {
		rows = rows[:listPage]
		n := rows[listPage-1].ID
		next = &n
	}
	out := make([]SchoolRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, SchoolRow{ID: r.ID, Code: r.Code, Name: r.DisplayName, Status: r.Status,
			Stage: r.OnboardingStage, GoLiveOn: dateOut(r.GoLiveOn), IsDemo: r.IsDemo, Created: r.CreatedAt})
	}
	return out, next, nil
}

func getSchool(ctx context.Context, id int64) (*SchoolDetail, error) {
	inPlatform, err := adminauth.PlatformReadOnly(ctx)
	if err != nil {
		return nil, err
	}
	var d SchoolDetail
	err = inPlatform(ctx, func(tx pgx.Tx) error {
		q := platformq.New(tx)
		s, err := q.GetSchool(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return errNotFound
		}
		if err != nil {
			return err
		}
		d = SchoolDetail{
			SchoolRow: SchoolRow{ID: s.ID, Code: s.Code, Name: s.DisplayName, Status: s.Status,
				Stage: s.OnboardingStage, GoLiveOn: dateOut(s.GoLiveOn), IsDemo: s.IsDemo, Created: s.CreatedAt},
			LegalName: s.LegalName, ShortName: s.ShortName, Board: s.Board, GSTIN: s.Gstin, City: s.City,
			StateCode: s.StateCode, ContactName: s.ContactName, ContactMobile: s.ContactMobile,
			PriceIncludesGST: s.PriceIncludesGst, AgentID: s.AgentID,
			Campuses: []CampusRow{}, Logins: []LoginRow{},
		}
		campuses, err := q.ListSchoolCampuses(ctx, id)
		if err != nil {
			return err
		}
		for _, c := range campuses {
			d.Campuses = append(d.Campuses, CampusRow{ID: c.ID, Code: c.Code, Name: c.Name, Lat: c.Lat, Lon: c.Lon,
				GeofenceM: c.GeofenceM, IsActive: c.IsActive})
		}
		users, err := q.ListSchoolPortalUsers(ctx, id)
		if err != nil {
			return err
		}
		for _, u := range users {
			d.Logins = append(d.Logins, LoginRow{ID: u.ID, Name: u.Name, Mobile: u.Mobile, Role: u.Role,
				PasswordSet: !u.MustChangeSecret, IsActive: u.IsActive})
		}
		pkg, err := q.GetSchoolPackage(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		d.Package = &PackageRow{ID: pkg.ID, PeriodStart: *dateOut(pkg.PeriodStart), PeriodEnd: *dateOut(pkg.PeriodEnd),
			StudentsCommitted: pkg.StudentsCommitted, PricePerStudentPaise: pkg.PricePerStudentPaise}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// moveStage changes the onboarding stage if it is still from; stages may move back, but a live school stays live
// (moving it back would reopen the parent OTP launch window). Moving to `live` stores at most today (IST) in the same
// statement: that date opens (and 14 days later closes) the launch window (docs/05 §10).
func moveStage(ctx context.Context, p *adminauth.Principal, ip *netip.Addr, id int64, from, to string,
	goLiveIn *string) (string, *string, error) {
	if from == "live" && to != "live" {
		return "", nil, errLeaveLive
	}
	goLive, err := optDate(goLiveIn)
	if err != nil {
		return "", nil, err
	}
	inPlatform, err := adminauth.Platform(ctx)
	if err != nil {
		return "", nil, err
	}
	var row platformq.SetOnboardingStageRow
	err = inPlatform(ctx, func(tx pgx.Tx) error {
		q := platformq.New(tx)
		var err error
		row, err = q.SetOnboardingStage(ctx, to, goLive, id, from)
		if errors.Is(err, pgx.ErrNoRows) {
			if _, err := q.GetSchool(ctx, id); errors.Is(err, pgx.ErrNoRows) {
				return errNotFound
			} else if err != nil {
				return err
			}
			return errStageChanged
		}
		if err != nil {
			return err
		}
		changes := map[string]any{"from": from, "to": to, "go_live_on": dateOut(row.GoLiveOn)}
		return audit(ctx, q, p, ip, id, "school.stage_changed", "tenant", id,
			"Onboarding stage "+from+" → "+to, changes)
	})
	if err != nil {
		return "", nil, err
	}
	return row.OnboardingStage, dateOut(row.GoLiveOn), nil
}

// reissueLink replaces a portal user's set-password links with a new one (the old ones stop working).
func reissueLink(ctx context.Context, p *adminauth.Principal, ip *netip.Addr, tenantID, userID int64) (*LinkBody,
	error) {
	inPlatform, err := adminauth.Platform(ctx)
	if err != nil {
		return nil, err
	}
	origin, err := adminauth.PortalOrigin(ctx)
	if err != nil {
		return nil, err
	}
	token, tokenHash, err := adminauth.NewToken()
	if err != nil {
		return nil, err
	}
	var out LinkBody
	err = inPlatform(ctx, func(tx pgx.Tx) error {
		q := platformq.New(tx)
		if err := q.SpendUserSecretLinks(ctx, tenantID, userID); err != nil {
			return err
		}
		l, err := q.CreateSecretLink(ctx, tokenHash, LinkHours, tenantID, userID)
		if errors.Is(err, pgx.ErrNoRows) {
			return errUserNotFound // rolls back the spend
		}
		if err != nil {
			return err
		}
		out = LinkBody{UserID: userID, URL: linkURL(origin, tenantID, token), ExpiresAt: l.ExpiresAt}
		return audit(ctx, q, p, ip, tenantID, "user.password_link_issued", "user", userID,
			"Issued a one-time set-password link", nil)
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// cancelLinks spends a school login's unused set-password links without issuing a new one (a link sent to the wrong
// person). It returns how many were still live.
func cancelLinks(ctx context.Context, p *adminauth.Principal, ip *netip.Addr, tenantID, userID int64) (int64, error) {
	inPlatform, err := adminauth.Platform(ctx)
	if err != nil {
		return 0, err
	}
	var n int64
	err = inPlatform(ctx, func(tx pgx.Tx) error {
		q := platformq.New(tx)
		if _, err := q.GetSchoolLogin(ctx, tenantID, userID); errors.Is(err, pgx.ErrNoRows) {
			return errUserNotFound
		} else if err != nil {
			return err
		}
		if n, err = q.CancelUserSecretLinks(ctx, tenantID, userID); err != nil {
			return err
		}
		return audit(ctx, q, p, ip, tenantID, "user.password_link_cancelled", "user", userID,
			"Cancelled the set-password link(s) without issuing a new one", map[string]any{"links": n})
	})
	return n, err
}

// deactivateLogin stops a school's owner/admin login signing in: is_active off, every session ended and every unused
// link spent, in one transaction. reactivateLogin brings it back with a new link.
func deactivateLogin(ctx context.Context, p *adminauth.Principal, ip *netip.Addr, tenantID, userID int64) error {
	inPlatform, err := adminauth.Platform(ctx)
	if err != nil {
		return err
	}
	return inPlatform(ctx, func(tx pgx.Tx) error {
		q := platformq.New(tx)
		n, err := q.DeactivateSchoolLogin(ctx, tenantID, userID)
		if err != nil {
			return err
		}
		if n == 0 {
			if _, err := q.GetSchoolLogin(ctx, tenantID, userID); errors.Is(err, pgx.ErrNoRows) {
				return errUserNotFound
			} else if err != nil {
				return err
			}
			return errUserInactive
		}
		sessions, err := q.RevokeUserSessions(ctx, tenantID, &userID)
		if err != nil {
			return err
		}
		links, err := q.CancelUserSecretLinks(ctx, tenantID, userID)
		if err != nil {
			return err
		}
		return audit(ctx, q, p, ip, tenantID, "user.deactivated", "user", userID,
			"Deactivated a school login (sessions ended, links cancelled)",
			map[string]any{"sessions_revoked": sessions, "links_cancelled": links})
	})
}

// reactivateLogin turns a deactivated owner/admin login back on with an unusable password and a new one-time link,
// in one transaction: a school whose only owner was deactivated (a link used by the wrong person) gets a way back
// in, and the password the wrong person set doesn't come back with it.
func reactivateLogin(ctx context.Context, p *adminauth.Principal, ip *netip.Addr, tenantID, userID int64) (*LinkBody,
	error) {
	inPlatform, err := adminauth.Platform(ctx)
	if err != nil {
		return nil, err
	}
	origin, err := adminauth.PortalOrigin(ctx)
	if err != nil {
		return nil, err
	}
	pwHash, err := unusableHash(ctx)
	if err != nil {
		return nil, err
	}
	token, tokenHash, err := adminauth.NewToken()
	if err != nil {
		return nil, err
	}
	var out LinkBody
	err = inPlatform(ctx, func(tx pgx.Tx) error {
		q := platformq.New(tx)
		n, err := q.ReactivateSchoolLogin(ctx, &pwHash, tenantID, userID)
		if err != nil {
			return err
		}
		if n == 0 {
			if _, err := q.GetSchoolLogin(ctx, tenantID, userID); errors.Is(err, pgx.ErrNoRows) {
				return errUserNotFound
			} else if err != nil {
				return err
			}
			return errUserActive
		}
		if err := q.SpendUserSecretLinks(ctx, tenantID, userID); err != nil {
			return err
		}
		l, err := q.CreateSecretLink(ctx, tokenHash, LinkHours, tenantID, userID)
		if err != nil {
			return err
		}
		out = LinkBody{UserID: userID, URL: linkURL(origin, tenantID, token), ExpiresAt: l.ExpiresAt}
		return audit(ctx, q, p, ip, tenantID, "user.reactivated", "user", userID,
			"Reactivated a school login with a new set-password link (old password dropped)", nil)
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// linkURL is the portal's set-password page with `<tenant_id>.<token>` in the fragment (never sent to the server
// or logged by Nginx; the page posts it).
func linkURL(origin string, tenantID int64, token string) string {
	return origin + "/set-password#" + strconv.FormatInt(tenantID, 10) + "." + token
}

func audit(ctx context.Context, q *platformq.Queries, p *adminauth.Principal, ip *netip.Addr, tenantID int64,
	action, entity string, entityID int64, summary string, changes map[string]any) error {
	var raw []byte
	if changes != nil {
		var err error
		if raw, err = json.Marshal(changes); err != nil {
			return fmt.Errorf("schools: audit changes: %w", err)
		}
	}
	rid := httpx.RequestID(ctx)
	return q.InsertPlatformAuditTenant(ctx, platformq.InsertPlatformAuditTenantParams{
		PlatformUserID: &p.UserID, ActorName: p.Name, Action: action, TenantID: &tenantID, Entity: entity,
		EntityID: &entityID, Summary: summary, Changes: raw, Ip: ip, RequestID: &rid,
	})
}

// dbError maps the constraint violations a wizard save can hit to 4xx (the request validation should catch most
// first; Postgres is the backstop). Responses never carry PgError.Detail (it can quote row values) or constraint
// names; anything unexpected keeps only SQLSTATE and constraint, for the 500's log line.
func dbError(err error) error {
	var pg *pgconn.PgError
	if !errors.As(err, &pg) {
		return err
	}
	switch {
	case pg.Code == "23505" && pg.ConstraintName == "tenants_code_key":
		return errCodeTaken
	case pg.Code == "23505" && pg.ConstraintName == "campuses_tenant_id_code_key":
		return errCampusDup
	case pg.Code == "23505":
		return errDuplicate
	case pg.Code == "23503" && pg.ConstraintName == "tenants_agent_id_fkey":
		return errAgent
	case pg.Code == "23514" && (pg.ConstraintName == "users_mobile_check" ||
		pg.ConstraintName == "tenants_contact_mobile_check"):
		return errMobile
	case pg.Code == "23514":
		return errValue
	case pg.Code == "22007" || pg.Code == "22008": // invalid / out-of-range date
		return errDate
	}
	return fmt.Errorf("schools: sqlstate %s constraint %q", pg.Code, pg.ConstraintName)
}

func parseDate(s string) (time.Time, error) {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return time.Time{}, errDate
	}
	return t, nil
}

func optDate(s *string) (pgtype.Date, error) {
	if s == nil {
		return pgtype.Date{}, nil
	}
	t, err := parseDate(*s)
	if err != nil {
		return pgtype.Date{}, err
	}
	return pgDate(t), nil
}

func pgDate(t time.Time) pgtype.Date { return pgtype.Date{Time: t, Valid: true} }

func dateOut(d pgtype.Date) *string {
	if !d.Valid {
		return nil
	}
	s := d.Time.Format(time.DateOnly)
	return &s
}
