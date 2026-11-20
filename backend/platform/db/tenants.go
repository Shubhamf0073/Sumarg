package db

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Tenants is the tenant pool handed to code that may not import pgxpool (depguard): login and session lookup,
// which run before or beside the request transaction. Build it with Bind.
type Tenants struct{ pool *pgxpool.Pool }

// Bind wraps the sf_app pool from OpenApp.
func Bind(pool *pgxpool.Pool) Tenants { return Tenants{pool: pool} }

// InTenant is db.InTenant on the bound pool.
func (t Tenants) InTenant(ctx context.Context, tenantID int64, fn func(pgx.Tx) error) error {
	return InTenant(ctx, t.pool, tenantID, fn)
}

// InTenantReadOnly is db.InTenantReadOnly on the bound pool.
func (t Tenants) InTenantReadOnly(ctx context.Context, tenantID int64, fn func(pgx.Tx) error) error {
	return InTenantReadOnly(ctx, t.pool, tenantID, fn)
}

// RequestTx is db.RequestTx on the bound pool.
func (t Tenants) RequestTx(resolve Resolver) func(http.Handler) http.Handler {
	return RequestTx(t.pool, resolve)
}

// BeforeTenant runs fn in a READ ONLY transaction with no tenant set: every tenant table returns zero rows, so the
// only useful thing to do here is the pre-tenant lookup (tenant_by_code, docs/02 A20) at login.
func (t Tenants) BeforeTenant(ctx context.Context, fn func(pgx.Tx) error) error {
	return pgx.BeginTxFunc(ctx, t.pool, pgx.TxOptions{AccessMode: pgx.ReadOnly}, fn)
}
