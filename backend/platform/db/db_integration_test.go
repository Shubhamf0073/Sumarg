//go:build integration

// Integration tests for tenant isolation through InTenant (docs/02 A19/A20). They need a migrated database and two
// role URLs (Makefile `test-integration` sets them for the dev compose Postgres):
//
//	TEST_APP_URL        sf_app on the test database
//	TEST_PLATFORM_URL   sf_platform on the same database
//
// Two tenants are seeded through the platform pool; every assertion is about what sf_app can and cannot see.
package db_test

import (
	"context"
	"errors"
	"os"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/store/tenantq"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/config"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/db"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/db/platformdb"
)

type fixture struct {
	app, platform *pgxpool.Pool
	t1, t2        int64 // tenant ids
	c2            int64 // a student of tenant 2
}

func setup(t *testing.T) fixture {
	t.Helper()
	appURL, platURL := os.Getenv("TEST_APP_URL"), os.Getenv("TEST_PLATFORM_URL")
	if appURL == "" || platURL == "" {
		t.Skip("TEST_APP_URL / TEST_PLATFORM_URL not set")
	}
	ctx := t.Context()
	app, err := db.OpenApp(ctx, config.Secret(appURL), db.Options{Name: "test-app", MaxConns: 4})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Close)
	plat, err := platformdb.Open(ctx, config.Secret(platURL), db.Options{Name: "test-platform", MaxConns: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(plat.Close)

	f := fixture{app: app, platform: plat}
	err = platformdb.InPlatform(ctx, plat, func(tx pgx.Tx) error {
		// Unique codes per run so the test can run against a database that already has data.
		suffix := randomSuffix(t)
		for i, p := range []*int64{&f.t1, &f.t2} {
			code := []string{"iso-a-", "iso-b-"}[i] + suffix
			if err := tx.QueryRow(ctx, `INSERT INTO tenants (code, legal_name, display_name, short_name, state_code,
				contact_name, contact_mobile) VALUES ($1, $1, $1, 'T', '24', 'Owner', '9800000000') RETURNING id`,
				code).Scan(p); err != nil {
				return err
			}
		}
		var b2, u2 int64
		if err := tx.QueryRow(ctx, `INSERT INTO campuses (tenant_id, code, name, lat, lon)
			VALUES ($1, 'MAIN', 'Main campus', 23.03, 72.58) RETURNING id`, f.t2).Scan(&b2); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `INSERT INTO users (tenant_id, name, mobile, password_hash, role, campus_id)
			VALUES ($1, 'Admin', '9800000009', 'x', 'admin', $2) RETURNING id`, f.t2, b2).Scan(&u2); err != nil {
			return err
		}
		_ = u2
		return tx.QueryRow(ctx, `INSERT INTO students (tenant_id, campus_id, admission_no, name, class_name)
			VALUES ($1, $2, 'A-001', 'Secret child of tenant 2', 'Grade 3') RETURNING id`, f.t2, b2).Scan(&f.c2)
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	return f
}

func TestTenantCannotSeeOtherTenant(t *testing.T) {
	f := setup(t)
	ctx := t.Context()
	err := db.InTenant(ctx, f.app, f.t1, func(tx pgx.Tx) error {
		var n int
		if err := tx.QueryRow(ctx, "SELECT count(*) FROM students WHERE id = $1", f.c2).Scan(&n); err != nil {
			return err
		}
		if n != 0 {
			t.Errorf("tenant 1 sees tenant 2's student")
		}
		var tenants int
		if err := tx.QueryRow(ctx, "SELECT count(*) FROM tenants").Scan(&tenants); err != nil {
			return err
		}
		if tenants != 1 {
			t.Errorf("tenant 1 sees %d tenant rows, want only its own", tenants)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// Same id, right tenant: visible.
	err = db.InTenant(ctx, f.app, f.t2, func(tx pgx.Tx) error {
		var name string
		return tx.QueryRow(ctx, "SELECT name FROM students WHERE id = $1", f.c2).Scan(&name)
	})
	if err != nil {
		t.Fatalf("tenant 2 can't read its own student: %v", err)
	}
}

func TestNoTenantSeesNothing(t *testing.T) {
	f := setup(t)
	var n int
	// Outside InTenant (no app.tenant_id): fail closed.
	if err := f.app.QueryRow(t.Context(), "SELECT count(*) FROM students").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("sf_app without a tenant sees %d students", n)
	}
	if err := db.InTenant(t.Context(), f.app, 0, func(pgx.Tx) error { return nil }); !errors.Is(err, db.ErrNoTenant) {
		t.Fatalf("tenant 0 should be ErrNoTenant, got %v", err)
	}
}

func TestCannotWriteIntoOtherTenant(t *testing.T) {
	f := setup(t)
	ctx := t.Context()
	err := db.InTenant(ctx, f.app, f.t1, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO students (tenant_id, campus_id, admission_no, name, class_name)
			SELECT $1, 1, 'X-1', 'Injected', 'Grade 1'`, f.t2)
		return err
	})
	if !isCode(err, "42501") { // insufficient_privilege: RLS WITH CHECK violation
		t.Fatalf("insert for another tenant: want RLS violation 42501, got %v", err)
	}
	err = db.InTenant(ctx, f.app, f.t2, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "DELETE FROM students WHERE id = $1", f.c2)
		return err
	})
	if !isCode(err, "42501") { // no DELETE grant for sf_app
		t.Fatalf("delete: want permission denied 42501, got %v", err)
	}
}

func TestReadOnlyTransactionRejectsWrites(t *testing.T) {
	f := setup(t)
	ctx := t.Context()
	err := db.InTenantReadOnly(ctx, f.app, f.t2, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE students SET name = 'x' WHERE id = $1", f.c2)
		return err
	})
	if !isCode(err, "25006") { // read_only_sql_transaction
		t.Fatalf("write in read-only tx: want 25006, got %v", err)
	}
	// The platform pool bypasses RLS, so its read-only tx (/api/admin GETs) matters even more.
	err = platformdb.InPlatformReadOnly(ctx, f.platform, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE tenants SET notes = notes WHERE id = $1", f.t2)
		return err
	})
	if !isCode(err, "25006") {
		t.Fatalf("write in read-only platform tx: want 25006, got %v", err)
	}
}

func TestSessionLookupNeedsTheRightTenant(t *testing.T) {
	f := setup(t)
	ctx := t.Context()
	hash := []byte("0123456789abcdef0123456789abcdef-" + randomSuffix(t))
	err := platformdb.InPlatform(ctx, f.platform, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO sessions (tenant_id, token_hash, user_id, client, expires_at)
			SELECT $1, $2, id, 'web', now() + interval '1 day' FROM users WHERE tenant_id = $1 LIMIT 1`, f.t2, hash)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	lookup := func(tenant int64) error {
		return db.InTenantReadOnly(ctx, f.app, tenant, func(tx pgx.Tx) error {
			_, err := tenantq.New(tx).GetSession(ctx, hash)
			return err
		})
	}
	if err := lookup(f.t2); err != nil {
		t.Fatalf("own tenant: %v", err)
	}
	// A cookie that names the wrong tenant id finds nothing (docs/02 A27).
	if err := lookup(f.t1); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("forged tenant id: want no rows, got %v", err)
	}
}

// The unit's MQTT secret hash is read by ingest only; sf_app has a column grant on bus_units that leaves it out.
func TestAppCannotReadUnitSecretHash(t *testing.T) {
	f := setup(t)
	ctx := t.Context()
	err := db.InTenantReadOnly(ctx, f.app, f.t1, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "SELECT secret_hash FROM bus_units")
		return err
	})
	if !isCode(err, "42501") { // insufficient_privilege
		t.Fatalf("secret_hash as sf_app: want 42501, got %v", err)
	}
	err = db.InTenantReadOnly(ctx, f.app, f.t1, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "SELECT id, uid, status, fw_version, last_seen_at FROM bus_units")
		return err
	})
	if err != nil {
		t.Fatalf("granted columns as sf_app: %v", err)
	}
}

func TestPoolsRefuseTheWrongRole(t *testing.T) {
	appURL, platURL := os.Getenv("TEST_APP_URL"), os.Getenv("TEST_PLATFORM_URL")
	if appURL == "" || platURL == "" {
		t.Skip("TEST_APP_URL / TEST_PLATFORM_URL not set")
	}
	ctx := context.Background()
	if _, err := db.OpenApp(ctx, config.Secret(platURL), db.Options{Name: "x", MaxConns: 1}); !errors.Is(err, db.ErrWrongRole) {
		t.Errorf("app pool with platform credentials: want ErrWrongRole, got %v", err)
	}
	if _, err := platformdb.Open(ctx, config.Secret(appURL), db.Options{Name: "x", MaxConns: 1}); !errors.Is(err, db.ErrWrongRole) {
		t.Errorf("platform pool with app credentials: want ErrWrongRole, got %v", err)
	}
}

func isCode(err error, code string) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == code
}

func randomSuffix(t *testing.T) string {
	t.Helper()
	b := make([]byte, 0, 8)
	for _, c := range []byte(t.Name()) {
		if c >= 'A' && c <= 'Z' {
			b = append(b, c+32)
		}
	}
	return string(b) + "-" + strconv.Itoa(os.Getpid())
}
