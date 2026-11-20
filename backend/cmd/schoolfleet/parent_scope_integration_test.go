//go:build integration

package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/isolation"
	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/tenant/auth"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/config"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/db"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/db/platformdb"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/httpx"
)

// parentNoIDs: parent operations that take no id, with the reason their answer is still only the caller's own
// (docs/04 "Parent app"). Each also needs its own two-guardian test of what it returns.
var parentNoIDs = map[string]string{}

// parentBodies: sample bodies for parent operations whose JSON body carries ids.
var parentBodies = map[string]isolation.Body{}

// parentOwnStatus: the expected status with the caller's own ids where that isn't 2xx (e.g. live → 409 no_trip).
var parentOwnStatus = map[string]int{}

// TestParentScope walks every /parent operation registerAPI registers (CLAUDE.md invariant 3, docs/07 S2): as
// guardian A, each id swapped for guardian B's (same school) or guardian C's (other school) must be a 404.
func TestParentScope(t *testing.T) {
	appURL, platURL := os.Getenv("TEST_APP_URL"), os.Getenv("TEST_PLATFORM_URL")
	if appURL == "" || platURL == "" {
		t.Skip("TEST_APP_URL / TEST_PLATFORM_URL not set")
	}
	ctx := t.Context()
	plat, err := platformdb.Open(ctx, config.Secret(platURL), db.Options{Name: "test-parent-scope", MaxConns: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(plat.Close)

	var a, b, c isolation.Principal
	err = platformdb.InPlatform(ctx, plat, func(tx pgx.Tx) error {
		suffix := rand.Text()[:10] // committed seed: unique per run, never colliding with an earlier one
		t1, campus1, bus1, err := seedSchool(ctx, tx, "scope-a-"+strings.ToLower(suffix))
		if err != nil {
			return err
		}
		t2, campus2, bus2, err := seedSchool(ctx, tx, "scope-b-"+strings.ToLower(suffix))
		if err != nil {
			return err
		}
		if a, err = seedFamily(ctx, tx, "A", t1, campus1, bus1, 1); err != nil {
			return err
		}
		if b, err = seedFamily(ctx, tx, "B (same school)", t1, campus1, bus1, 2); err != nil {
			return err
		}
		c, err = seedFamily(ctx, tx, "C (other school)", t2, campus2, bus2, 1)
		return err
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Each guardian signs in for real, through the OTP endpoints (the code read back from the outbox as the worker
	// would), and the walk goes through main's router: mountAPI on the sf_app pool, so the session resolver, RLS and the request
	// transaction are the production ones.
	app, err := db.OpenApp(ctx, config.Secret(appURL), db.Options{Name: "test-parent-scope-app", MaxConns: 4})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Close)
	keys := testOTPKeys(t)
	r := chi.NewRouter()
	mountAPI(r, auth.Deps{DB: db.Bind(app), OTP: keys})
	inPlat := func(ctx context.Context, fn func(pgx.Tx) error) error { return platformdb.InPlatform(ctx, plat, fn) }
	headers := map[int64]http.Header{}
	for _, p := range []isolation.Principal{a, b, c} {
		install := "walker-" + strings.ToLower(rand.Text())
		tok, err := otpSignIn(ctx, r, keys, inPlat, p.Tenant, p.Guardian, install)
		if err != nil {
			t.Fatalf("sign-in for %s: %v", p.Name, err)
		}
		headers[p.Guardian] = http.Header{"Authorization": {"Bearer " + tok}, auth.InstallHeader: {install}}
	}
	w := isolation.Walk{
		API: httpx.NewSpecAPI(apiTitle, apiVersion), Handler: r, Me: a, Others: []isolation.Principal{b, c},
		NoIDs: parentNoIDs, Bodies: parentBodies, OwnStatus: parentOwnStatus,
		Auth: func(p isolation.Principal) http.Header { return headers[p.Guardian].Clone() },
	}
	registerAPI(w.API)

	// The sign-in itself works: A is A, in A's school. Without this, a broken Auth would turn every walk into
	// 401s, which the own-id positive control catches only once a parent route exists.
	for _, p := range []isolation.Principal{a, c} {
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, auth.PathMe, nil)
		req.Header = w.Auth(p)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		var me struct {
			Client   string `json:"client"`
			Guardian struct {
				ID int64 `json:"id"`
			} `json:"guardian"`
			School struct {
				ID int64 `json:"id"`
			} `json:"school"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &me)
		if rec.Code != http.StatusOK || me.Client != "parent_app" || me.Guardian.ID != p.Guardian || me.School.ID != p.Tenant {
			t.Fatalf("%s: GET /api/me = %d %s", p.Name, rec.Code, rec.Body.String())
		}
	}

	if ops, _ := w.Operations(); len(ops) == 0 {
		t.Log("no /parent operations registered yet; fixture seeded, nothing to walk")
	}
	if problems := w.Check(ctx); len(problems) > 0 {
		t.Fatalf("parent scope:\n%s", strings.Join(problems, "\n"))
	}
}

func seedSchool(ctx context.Context, tx pgx.Tx, code string) (tenant, campus, vehicle int64, err error) {
	if err = tx.QueryRow(ctx, `INSERT INTO tenants (code, legal_name, display_name, short_name, state_code,
		contact_name, contact_mobile) VALUES ($1, $1, $1, 'T', '24', 'Owner', '9800000000') RETURNING id`,
		code).Scan(&tenant); err != nil {
		return 0, 0, 0, fmt.Errorf("tenant: %w", err)
	}
	if err = tx.QueryRow(ctx, `INSERT INTO campuses (tenant_id, code, name, lat, lon)
		VALUES ($1, 'MAIN', 'Main', 23.03, 72.58) RETURNING id`, tenant).Scan(&campus); err != nil {
		return 0, 0, 0, fmt.Errorf("campus: %w", err)
	}
	if err = tx.QueryRow(ctx, `INSERT INTO vehicles (tenant_id, campus_id, reg_no, label, seats)
		VALUES ($1, $2, 'GJ01AB1234', 'Bus 1', 40) RETURNING id`, tenant, campus).Scan(&vehicle); err != nil {
		return 0, 0, 0, fmt.Errorf("vehicle: %w", err)
	}
	return tenant, campus, vehicle, nil
}

// seedFamily: one guardian with one child on its own route, a started trip today, a boarding, an absence and a
// request — one id of every parent-facing kind. n keeps codes unique inside the school.
func seedFamily(ctx context.Context, tx pgx.Tx, name string, tenant, campus, vehicle int64, n int) (isolation.Principal, error) {
	p := isolation.Principal{Name: name, Tenant: tenant, IDs: isolation.IDs{}}
	var route, student, trip, boarding, absence, request int64
	steps := []struct {
		what string
		dst  *int64
		sql  string
		args []any
	}{
		{"route", &route, `INSERT INTO routes (tenant_id, campus_id, code, name, direction, vehicle_id, planned_start)
			VALUES ($1, $2, $3, 'Route', 'to_school', $4, '07:00') RETURNING id`,
			[]any{tenant, campus, fmt.Sprintf("R-%02d", n), vehicle}},
		{"student", &student, `INSERT INTO students (tenant_id, campus_id, admission_no, name, class_name, to_route_id)
			VALUES ($1, $2, $3, 'Child', 'Grade 3', $4) RETURNING id`,
			[]any{tenant, campus, fmt.Sprintf("A-%03d", n), &route}},
		{"guardian", &p.Guardian, `INSERT INTO guardians (tenant_id, name, mobile) VALUES ($1, 'Parent', $2) RETURNING id`,
			[]any{tenant, fmt.Sprintf("98000001%02d", n)}},
		{"trip", &trip, `INSERT INTO trips (tenant_id, route_id, service_date, vehicle_id, status, started_at, started_by)
			VALUES ($1, $2, current_date, $3, 'started', now(), 'unit') RETURNING id`,
			[]any{tenant, &route, vehicle}},
		{"boarding", &boarding, `INSERT INTO boardings (tenant_id, trip_id, student_id, kind, method, at, source_key)
			VALUES ($1, $2, $3, 'board', 'rfid', now(), $4) RETURNING id`,
			[]any{tenant, &trip, &student, fmt.Sprintf("SB-00000%d:%d", n, n)}},
		{"absence", &absence, `INSERT INTO absences (tenant_id, student_id, on_date, direction, guardian_id)
			VALUES ($1, $2, current_date + 1, 'both', $3) RETURNING id`,
			[]any{tenant, &student, &p.Guardian}},
		{"request", &request, `INSERT INTO requests (tenant_id, student_id, guardian_id, kind, details)
			VALUES ($1, $2, $3, 'contact', '{}') RETURNING id`,
			[]any{tenant, &student, &p.Guardian}},
	}
	for _, s := range steps {
		args := make([]any, len(s.args))
		for i, a := range s.args { // ids from earlier steps are passed by pointer; read them now
			if ptr, ok := a.(*int64); ok {
				a = *ptr
			}
			args[i] = a
		}
		if err := tx.QueryRow(ctx, s.sql, args...).Scan(s.dst); err != nil {
			return p, fmt.Errorf("%s %s: %w", name, s.what, err)
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO student_guardians (tenant_id, student_id, guardian_id, relation, is_primary)
		VALUES ($1, $2, $3, 'mother', true)`, tenant, student, p.Guardian); err != nil {
		return p, fmt.Errorf("%s student_guardians: %w", name, err)
	}
	p.IDs = isolation.IDs{
		isolation.Route: route, isolation.Student: student, isolation.Trip: trip,
		isolation.Boarding: boarding, isolation.Absence: absence, isolation.Request: request,
	}
	return p, nil
}
