// Command migrate applies the goose migrations embedded from backend/migrations, as sf_owner. It runs as a
// one-off container before each deploy (infra/scripts/deploy.sh) and locally via `make migrate`.
//
//	migrate up                         apply all pending migrations
//	migrate status                     list applied / pending
//	migrate create-admin --email=... [--name=...] [--role=superadmin]
//	                                   a Previa staff member: prints a one-time passkey enrolment link (15 min);
//	                                   for an existing user, a new link (lost passkey) and re-activation
//	migrate seed --sample              M2.1  local/dev only: Previa admin + two sample tenants
//
// Environment: APP_ENV, SVC_NAME=migrate, DATABASE_URL (the sf_owner URL; never the app or platform role);
// create-admin also APP_DOMAIN (the link's host).
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	stdlog "log"
	"log/slog"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver goose needs
	"github.com/pressly/goose/v3"

	"github.com/Shubhamf0073/previa-school-fleet/backend/migrations"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/config"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/obs"
)

var planned = map[string]string{"seed": "M2.1"}

func main() { os.Exit(run(context.Background(), os.Args[1:])) }

func run(ctx context.Context, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: migrate up | status | create-admin --email=... | seed --sample")
		return 2
	}
	if args[0] == "create-admin" {
		return runCreateAdmin(ctx, args[1:], os.Stdout, os.Stderr)
	}
	if m := planned[args[0]]; m != "" {
		fmt.Fprintf(os.Stderr, "migrate %s: not implemented yet (%s)\n", args[0], m)
		return 1
	}
	if args[0] != "up" && args[0] != "status" {
		fmt.Fprintln(os.Stderr, "usage: migrate up | status | create-admin --email=... | seed --sample")
		return 2
	}
	cfg, err := config.Load[config.Base]()
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
		return 1
	}
	if cfg.ServiceName != "migrate" {
		fmt.Fprintf(os.Stderr, "migrate: SVC_NAME is %q, want \"migrate\" (wrong env?)\n", cfg.ServiceName)
		return 1
	}
	log := obs.NewLogger(os.Stdout, cfg)
	if err := migrate(ctx, args[0], cfg.DatabaseURL, log); err != nil {
		log.Error("migrate failed", slog.String("command", args[0]), slog.Any("error", err))
		return 1
	}
	return 0
}

func migrate(ctx context.Context, cmd string, url config.Secret, log *slog.Logger) error {
	db, err := sql.Open("pgx", url.Reveal())
	if err != nil {
		return errors.New("open database (URL not shown)")
	}
	defer func() { _ = db.Close() }()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	// The migrator must be the schema owner: anything else would create objects other roles own, and RLS/grants in
	// the migrations would be wrong (docs/02 A20).
	var role string
	if err := db.QueryRowContext(ctx, "SELECT current_user").Scan(&role); err != nil {
		return err
	}
	if role != "sf_owner" {
		return fmt.Errorf("connected as %q; migrations run only as sf_owner", role)
	}
	goose.SetBaseFS(migrations.FS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	switch cmd {
	case "status":
		goose.SetLogger(stdlog.New(os.Stdout, "", 0)) // status prints its table through the logger
		return goose.StatusContext(ctx, db, ".")
	default:
		before, _ := goose.GetDBVersionContext(ctx, db)
		if err := goose.UpContext(ctx, db, "."); err != nil {
			return err
		}
		after, err := goose.GetDBVersionContext(ctx, db)
		if err != nil {
			return err
		}
		log.Info("migrations applied", slog.Int64("from_version", before), slog.Int64("to_version", after))
		return nil
	}
}
