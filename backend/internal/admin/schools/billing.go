package schools

import (
	"context"
	"errors"
	"net/http"
	"net/netip"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/admin/adminauth"
	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/store/platformq"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/httpx"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/money"
)

var (
	errStatusChanged = httpx.NewError(http.StatusConflict, "status_changed",
		"Someone changed this school's status meanwhile. Reload and try again.")
	errTransition = httpx.NewError(http.StatusUnprocessableEntity, "status_transition",
		"A school can't move to that status from its current one.")
	errReason = httpx.NewError(http.StatusUnprocessableEntity, "reason_required",
		"Give a reason for suspending or cancelling a school.")
	errPackageNotFound = httpx.NewError(http.StatusNotFound, "not_found", "No such active package for this school.")
	errPackageChanged  = httpx.NewError(http.StatusConflict, "package_changed",
		"Someone edited this package meanwhile. Reload and try again.")
)

// transitions are the billing status moves /admin allows (docs/05 §11). cancelled is final: a school that comes back
// is a new package and status flow, not an undo.
var transitions = map[string][]string{
	"trial":     {"active", "cancelled"},
	"active":    {"past_due", "suspended", "cancelled"},
	"past_due":  {"active", "suspended", "cancelled"},
	"suspended": {"active", "past_due", "cancelled"},
}

// checkTransition says whether from → to is allowed and has the reason it needs (suspend and cancel need one).
func checkTransition(from, to string, reason *string) error {
	if !slices.Contains(transitions[from], to) {
		return errTransition
	}
	if (to == "suspended" || to == "cancelled") && (reason == nil || strings.TrimSpace(*reason) == "") {
		return errReason
	}
	return nil
}

// setStatus changes a school's billing status if it is still from. Sessions pick it up on their next request
// (suspended → read-only, cancelled → 423).
func setStatus(ctx context.Context, p *adminauth.Principal, ip *netip.Addr, id int64, from, to string,
	reason *string) (string, error) {
	if err := checkTransition(from, to, reason); err != nil {
		return "", err
	}
	if reason != nil {
		r := strings.TrimSpace(*reason)
		reason = &r
	}
	inPlatform, err := adminauth.Platform(ctx)
	if err != nil {
		return "", err
	}
	var status string
	err = inPlatform(ctx, func(tx pgx.Tx) error {
		q := platformq.New(tx)
		var err error
		status, err = q.SetSchoolStatus(ctx, to, reason, id, from)
		if errors.Is(err, pgx.ErrNoRows) {
			if _, err := q.GetSchool(ctx, id); errors.Is(err, pgx.ErrNoRows) {
				return errNotFound
			} else if err != nil {
				return err
			}
			return errStatusChanged
		}
		if err != nil {
			return err
		}
		return audit(ctx, q, p, ip, id, "school.status_changed", "tenant", id, "Status "+from+" → "+to,
			map[string]any{"from": from, "to": to, "reason": reason})
	})
	if err != nil {
		return "", err
	}
	return status, nil
}

// PackageEdit is a billing user's change to the active package.
type PackageEdit struct {
	Version              int32       `json:"version" minimum:"1" doc:"the version you saw (409 if it changed)"`
	PeriodStart          string      `json:"period_start" format:"date"`
	StudentsCommitted    int32       `json:"students_committed" minimum:"0" maximum:"100000"`
	PricePerStudentPaise money.Paise `json:"price_per_student_paise" minimum:"0" maximum:"100000000" doc:"integer paise; ₹1,000 = 100000"`
}

// editPackage changes the school's active package (and the school's price with it) if it is still the version the
// billing user saw. period_end stays start + 12 months − 1 day.
func editPackage(ctx context.Context, p *adminauth.Principal, ip *netip.Addr, tenantID, pkgID int64,
	e PackageEdit) (*PackageRow, error) {
	start, err := parseDate(e.PeriodStart)
	if err != nil {
		return nil, err
	}
	end := start.AddDate(1, 0, -1)
	inPlatform, err := adminauth.Platform(ctx)
	if err != nil {
		return nil, err
	}
	out := &PackageRow{ID: pkgID, PeriodStart: start.Format(time.DateOnly), PeriodEnd: end.Format(time.DateOnly),
		StudentsCommitted: e.StudentsCommitted, PricePerStudentPaise: e.PricePerStudentPaise}
	err = inPlatform(ctx, func(tx pgx.Tx) error {
		q := platformq.New(tx)
		old, err := q.GetSchoolPackageForEdit(ctx, tenantID, pkgID)
		if errors.Is(err, pgx.ErrNoRows) {
			return errPackageNotFound
		}
		if err != nil {
			return err
		}
		if old.Version != e.Version {
			return errPackageChanged
		}
		out.Version, err = q.UpdateSchoolPackage(ctx, platformq.UpdateSchoolPackageParams{
			StudentsCommitted: e.StudentsCommitted, PricePerStudentPaise: e.PricePerStudentPaise,
			PeriodStart: pgDate(start), PeriodEnd: pgDate(end), TenantID: tenantID, ID: pkgID, Version: e.Version})
		if errors.Is(err, pgx.ErrNoRows) {
			return errPackageChanged // locked above, so only if the row changed under another path
		}
		if err != nil {
			return dbError(err)
		}
		if err := q.SetSchoolPrice(ctx, e.PricePerStudentPaise, tenantID); err != nil {
			return dbError(err)
		}
		changes := map[string]any{
			"from": map[string]any{"period_start": dateOut(old.PeriodStart), "students_committed": old.StudentsCommitted,
				"price_per_student_paise": old.PricePerStudentPaise},
			"to": map[string]any{"period_start": out.PeriodStart, "students_committed": e.StudentsCommitted,
				"price_per_student_paise": e.PricePerStudentPaise},
		}
		return audit(ctx, q, p, ip, tenantID, "package.edited", "subscription", pkgID, "Edited the active package",
			changes)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
