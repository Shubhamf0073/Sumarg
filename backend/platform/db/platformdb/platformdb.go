// Package platformdb is the superadmin/worker side of the database: it connects as sf_platform, which BYPASSES
// Row-Level Security. Only internal/admin, internal/worker and cmd/ may import it — depguard fails the lint if
// tenant code does (CLAUDE.md invariant 3). Every query run here that touches tenant tables names its tenant(s)
// explicitly.
package platformdb

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/config"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/db"
)

// Open opens the platform pool. It refuses a role that does not bypass RLS (the app password by mistake would make
// every admin screen silently empty).
func Open(ctx context.Context, url config.Secret, o db.Options) (*pgxpool.Pool, error) {
	return db.Open(ctx, url, o, true)
}

// InPlatform runs fn in one transaction on the platform pool. Build queries with platformq.New(tx).
func InPlatform(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) error {
	return pgx.BeginTxFunc(ctx, pool, pgx.TxOptions{}, fn)
}
