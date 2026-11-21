//go:build integration

package main

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/admin/adminauth"
	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/admin/schools"
)

// Two schools: status changes are conditional on from_status, follow the allowed transitions, stamp the suspend and
// cancel columns, are audited, and touch only the school named; a cancelled school is final and refuses sign-in.
func TestSchoolStatusTwoSchools(t *testing.T) {
	e, _, _ := newAuthEnv(t)
	sup := e.adminAs(t, adminauth.RoleSupport)
	bill := e.adminAs(t, adminauth.RoleBilling)
	codeA := newCode()
	a := e.do(t, adminCall(sup, http.MethodPost, schools.PathSchools, wizardBody(codeA)))
	b := e.do(t, adminCall(sup, http.MethodPost, schools.PathSchools, wizardBody(newCode())))
	if a.code != http.StatusCreated || b.code != http.StatusCreated {
		t.Fatalf("create: %d %d", a.code, b.code)
	}
	idA, idB := id(a), id(b)
	if r := e.do(t, setPasswordCall(linkToken(t, a), "a long password")); r.code != http.StatusNoContent {
		t.Fatalf("set password: %d %s", r.code, r.raw)
	}
	move := func(sid int64, from, to string, reason any) resp {
		body := map[string]any{"from_status": from, "status": to}
		if reason != nil {
			body["reason"] = reason
		}
		return e.do(t, adminCall(bill, http.MethodPatch, schoolPath(sid, "/status"), body))
	}
	want := func(r resp, code int, errCode string) {
		t.Helper()
		if r.code != code || (errCode != "" && r.body["code"] != errCode) {
			t.Fatalf("got %d %s, want %d %s", r.code, r.raw, code, errCode)
		}
	}

	want(move(idA, "trial", "active", nil), http.StatusOK, "")
	want(move(idA, "trial", "cancelled", "dup"), http.StatusConflict, "status_changed") // A is active now
	want(move(idA, "active", "trial", nil), http.StatusUnprocessableEntity, "status_transition")
	want(move(idA, "active", "suspended", nil), http.StatusUnprocessableEntity, "reason_required")
	want(move(idA, "active", "suspended", "  "), http.StatusUnprocessableEntity, "reason_required")
	want(move(999999999, "active", "past_due", nil), http.StatusNotFound, "not_found")
	want(move(idA, "active", "suspended", " unpaid 45 days "), http.StatusOK, "")

	var status, reason string
	var suspended, cancelled bool
	row := func(sid int64) {
		t.Helper()
		e.q(t, `SELECT status FROM tenants WHERE id = $1`, &status, sid)
		e.q(t, `SELECT coalesce(suspend_reason, '') FROM tenants WHERE id = $1`, &reason, sid)
		e.q(t, `SELECT suspended_at IS NOT NULL FROM tenants WHERE id = $1`, &suspended, sid)
		e.q(t, `SELECT cancelled_at IS NOT NULL FROM tenants WHERE id = $1`, &cancelled, sid)
	}
	row(idA)
	if status != "suspended" || reason != "unpaid 45 days" || !suspended || cancelled {
		t.Fatalf("A after suspend: %s %q %v %v", status, reason, suspended, cancelled)
	}
	g := e.do(t, adminCall(bill, http.MethodGet, schoolPath(idA, ""), nil))
	if g.code != http.StatusOK || g.body["status"] != "suspended" || g.body["suspend_reason"] != "unpaid 45 days" {
		t.Fatalf("get after suspend: %d %s", g.code, g.raw)
	}
	// B never moved.
	row(idB)
	if status != "trial" || suspended || cancelled {
		t.Fatalf("B touched: %s %v %v", status, suspended, cancelled)
	}

	// Leaving suspended clears the stamp; cancelling stamps cancelled_at and is final.
	want(move(idA, "suspended", "active", nil), http.StatusOK, "")
	row(idA)
	if status != "active" || reason != "" || suspended {
		t.Fatalf("A after unsuspend: %s %q %v", status, reason, suspended)
	}
	want(move(idA, "active", "cancelled", "school closed its transport"), http.StatusOK, "")
	row(idA)
	if status != "cancelled" || !cancelled {
		t.Fatalf("A after cancel: %s %v", status, cancelled)
	}
	want(move(idA, "cancelled", "active", "mistake"), http.StatusUnprocessableEntity, "status_transition")
	if r := e.do(t, portalLoginCall(codeA, newOwnerMobile, "a long password")); r.code != http.StatusLocked {
		t.Fatalf("cancelled school login: %d %s", r.code, r.raw)
	}

	var n int
	e.q(t, `SELECT count(*) FROM platform_audit_log WHERE tenant_id = $1 AND action = 'school.status_changed'`, &n, idA)
	if n != 4 {
		t.Fatalf("A status audit rows = %d, want 4", n)
	}
	e.q(t, `SELECT count(*) FROM platform_audit_log WHERE tenant_id = $1 AND action = 'school.status_changed'`, &n, idB)
	if n != 0 {
		t.Fatalf("B status audit rows = %d", n)
	}
}

// Two schools: a package edit is conditional on its version, recomputes period_end, moves the school's price with
// it, is audited with before/after, and another school's package id under this school is 404.
func TestSchoolPackageEditTwoSchools(t *testing.T) {
	e, _, _ := newAuthEnv(t)
	sup := e.adminAs(t, adminauth.RoleSupport)
	bill := e.adminAs(t, adminauth.RoleBilling)
	a := e.do(t, adminCall(sup, http.MethodPost, schools.PathSchools, wizardBody(newCode())))
	b := e.do(t, adminCall(sup, http.MethodPost, schools.PathSchools, wizardBody(newCode())))
	if a.code != http.StatusCreated || b.code != http.StatusCreated {
		t.Fatalf("create: %d %d", a.code, b.code)
	}
	idA, idB := id(a), id(b)
	pkg := func(sid int64) (int64, float64) {
		t.Helper()
		g := e.do(t, adminCall(bill, http.MethodGet, schoolPath(sid, ""), nil))
		p, _ := g.body["package"].(map[string]any)
		if g.code != http.StatusOK || p == nil {
			t.Fatalf("get: %d %s", g.code, g.raw)
		}
		return int64(p["id"].(float64)), p["version"].(float64)
	}
	pkgA, verA := pkg(idA)
	pkgB, _ := pkg(idB)
	if verA != 1 {
		t.Fatalf("version = %v", verA)
	}
	edit := func(sid, pid int64, version float64) resp {
		return e.do(t, adminCall(bill, http.MethodPatch, schoolPath(sid, fmt.Sprintf("/package/%d", pid)),
			map[string]any{"version": version, "period_start": "2026-07-01", "students_committed": 500,
				"price_per_student_paise": 90000}))
	}

	r := edit(idA, pkgA, 1)
	if r.code != http.StatusOK || r.body["period_end"] != "2027-06-30" || r.body["version"] != 2.0 ||
		r.body["students_committed"] != 500.0 || r.body["price_per_student_paise"] != 90000.0 {
		t.Fatalf("edit: %d %s", r.code, r.raw)
	}
	if r := edit(idA, pkgA, 1); r.code != http.StatusConflict || r.body["code"] != "package_changed" {
		t.Fatalf("stale version: %d %s", r.code, r.raw)
	}
	if r := edit(idA, pkgB, 1); r.code != http.StatusNotFound {
		t.Fatalf("B's package under A: %d %s", r.code, r.raw)
	}
	bad := e.do(t, adminCall(bill, http.MethodPatch, schoolPath(idA, fmt.Sprintf("/package/%d", pkgA)),
		map[string]any{"version": 2, "period_start": "2026-02-30", "students_committed": 1,
			"price_per_student_paise": 1}))
	if bad.code != http.StatusBadRequest { // format:"date" is checked by the request validation
		t.Fatalf("bad date: %d %s", bad.code, bad.raw)
	}

	var price, committed int64
	e.q(t, `SELECT price_per_student_paise FROM tenants WHERE id = $1`, &price, idA)
	if price != 90000 {
		t.Fatalf("A school price = %d", price)
	}
	e.q(t, `SELECT price_per_student_paise FROM tenants WHERE id = $1`, &price, idB)
	e.q(t, `SELECT students_committed FROM subscriptions WHERE tenant_id = $1`, &committed, idB)
	if price != 100000 || committed != 420 {
		t.Fatalf("B touched: price %d committed %d", price, committed)
	}
	var n int
	e.q(t, `SELECT count(*) FROM platform_audit_log WHERE tenant_id = $1 AND action = 'package.edited'
		AND (changes->'from'->>'students_committed')::int = 420 AND (changes->'to'->>'students_committed')::int = 500`,
		&n, idA)
	if n != 1 {
		t.Fatalf("A package audit rows = %d", n)
	}
}

// Billing changes: billing and superadmin may; support and installers may not.
func TestSchoolBillingRoles(t *testing.T) {
	e, _, _ := newAuthEnv(t)
	sup := e.adminAs(t, adminauth.RoleSupport)
	made := e.do(t, adminCall(sup, http.MethodPost, schools.PathSchools, wizardBody(newCode())))
	if made.code != http.StatusCreated {
		t.Fatalf("create: %d %s", made.code, made.raw)
	}
	sid := id(made)
	for _, c := range []adminCookies{sup, e.adminAs(t, adminauth.RoleInstaller)} {
		if r := e.do(t, adminCall(c, http.MethodPatch, schoolPath(sid, "/status"),
			map[string]any{"from_status": "trial", "status": "active"})); r.code != http.StatusForbidden {
			t.Fatalf("status: %d", r.code)
		}
		if r := e.do(t, adminCall(c, http.MethodPatch, schoolPath(sid, "/package/1"),
			map[string]any{"version": 1, "period_start": "2026-07-01", "students_committed": 1,
				"price_per_student_paise": 1})); r.code != http.StatusForbidden {
			t.Fatalf("package: %d", r.code)
		}
	}
	root := e.adminAs(t, adminauth.RoleSuperadmin)
	if r := e.do(t, adminCall(root, http.MethodPatch, schoolPath(sid, "/status"),
		map[string]any{"from_status": "trial", "status": "active"})); r.code != http.StatusOK {
		t.Fatalf("superadmin status: %d %s", r.code, r.raw)
	}
}
