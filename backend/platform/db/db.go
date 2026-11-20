// Package db opens the tenant pool and runs tenant transactions. It is the only package (with platformdb) allowed
// to import pgxpool (depguard in .golangci.yml).
//
// Tenant isolation is enforced by Postgres (docs/02 A19/A20): the app pool connects as sf_app, which is subject
// to forced Row-Level Security, and every tenant request runs inside InTenant, which sets app.tenant_id for that
// transaction only (set_config(..., true)). Outside InTenant the setting is empty and every tenant table returns zero
// rows: the failure mode is "nothing visible", never "everything visible".
package db

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/config"
)

// ErrNoTenant means InTenant was called without a tenant: a bug, never a user error.
var ErrNoTenant = errors.New("db: no tenant in context")

// ErrWrongRole means a pool connected as a role with the wrong RLS power (e.g. the platform password in
// DATABASE_URL). The service refuses to start rather than serve tenant requests without RLS.
var ErrWrongRole = errors.New("db: database role has the wrong RLS setting for this pool")

// Options are per-pool settings sent as connection parameters, so they hold on every connection the pool opens.
type Options struct {
	Name             string        // application_name: shows in pg_stat_activity and slow-query logs
	MaxConns         int32         // config.Base.DBMaxConns
	StatementTimeout time.Duration // default 10 s: a request never waits on one statement longer
}

// Open connects a pool and checks the role's BYPASSRLS flag matches wantBypassRLS. Tenant code calls OpenApp.
func Open(ctx context.Context, url config.Secret, o Options, wantBypassRLS bool) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url.Reveal())
	if err != nil {
		return nil, errors.New("db: invalid database URL (value not shown)") // pgx errors can quote the URL
	}
	if o.StatementTimeout == 0 {
		o.StatementTimeout = 10 * time.Second
	}
	cfg.MaxConns = o.MaxConns
	cfg.MinConns = 1
	cfg.MaxConnIdleTime = 5 * time.Minute
	rp := cfg.ConnConfig.RuntimeParams
	rp["application_name"] = o.Name
	rp["statement_timeout"] = strconv.FormatInt(o.StatementTimeout.Milliseconds(), 10)
	rp["idle_in_transaction_session_timeout"] = "30000"
	rp["timezone"] = "UTC"

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("db: connect %s: %w", o.Name, err)
	}
	var bypass bool
	if err := pool.QueryRow(ctx, "SELECT rolbypassrls FROM pg_roles WHERE rolname = current_user").Scan(&bypass); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: check role of %s: %w", o.Name, err)
	}
	if bypass != wantBypassRLS {
		pool.Close()
		return nil, fmt.Errorf("%w (%s: bypassrls=%t)", ErrWrongRole, o.Name, bypass)
	}
	return pool, nil
}

// OpenApp opens the tenant pool (sf_app). It refuses a role that bypasses RLS.
func OpenApp(ctx context.Context, url config.Secret, o Options) (*pgxpool.Pool, error) {
	return Open(ctx, url, o, false)
}

// InTenant runs fn in one read-write transaction with app.tenant_id = tenantID. fn gets the transaction; build
// queries with tenantq.New(tx). Commit happens if fn returns nil, rollback otherwise (also on panic).
func InTenant(ctx context.Context, pool *pgxpool.Pool, tenantID int64, fn func(pgx.Tx) error) error {
	return inTenant(ctx, pool, tenantID, pgx.TxOptions{}, fn)
}

// InTenantReadOnly is InTenant in a READ ONLY transaction: Postgres itself rejects any write. GET handlers and every
// request on a read-only session (Previa support view, suspended tenant) use it (docs/05 §10).
func InTenantReadOnly(ctx context.Context, pool *pgxpool.Pool, tenantID int64, fn func(pgx.Tx) error) error {
	return inTenant(ctx, pool, tenantID, pgx.TxOptions{AccessMode: pgx.ReadOnly}, fn)
}

func inTenant(ctx context.Context, pool *pgxpool.Pool, tenantID int64, opts pgx.TxOptions, fn func(pgx.Tx) error) error {
	if tenantID <= 0 {
		return ErrNoTenant
	}
	return pgx.BeginTxFunc(ctx, pool, opts, func(tx pgx.Tx) error {
		// is_local = true: the setting ends with this transaction, so a pooled connection never carries it over.
		if _, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", strconv.FormatInt(tenantID, 10)); err != nil {
			return fmt.Errorf("db: set tenant: %w", err)
		}
		return fn(tx)
	})
}

// Ping is a readiness check for httpx.Readyz.
func Ping(pool *pgxpool.Pool) func(context.Context) error {
	return func(ctx context.Context) error { return pool.Ping(ctx) }
}
