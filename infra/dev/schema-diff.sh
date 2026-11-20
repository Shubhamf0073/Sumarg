#!/usr/bin/env bash
# Proves the goose migrations and docs/schema.sql describe the same database (docs/06 §8).
# Builds two scratch databases on the dev Postgres (make dev-db): A = `migrate up`, B = docs/schema.sql loaded as
# sf_owner; diffs `pg_dump --schema-only` of both (goose's version table excluded); drops both.
# Usage: infra/dev/schema-diff.sh            (needs Go and Docker; run from the repo root via `make schema-diff`)
set -euo pipefail

compose=(docker compose -f infra/docker-compose.dev.yml)
psql_su() { "${compose[@]}" exec -T db psql -v ON_ERROR_STOP=1 -q -U postgres "$@"; }
a=sf_diff_migrated b=sf_diff_schema
tmp=$(mktemp -d)
cleanup() { psql_su -d postgres -c "DROP DATABASE IF EXISTS $a" -c "DROP DATABASE IF EXISTS $b" >/dev/null 2>&1 || true; rm -rf "$tmp"; }
trap cleanup EXIT

for d in "$a" "$b"; do
  psql_su -d postgres -c "DROP DATABASE IF EXISTS $d" -c "CREATE DATABASE $d OWNER sf_owner" \
    -c "REVOKE ALL ON DATABASE $d FROM PUBLIC" -c "GRANT CONNECT ON DATABASE $d TO sf_app, sf_platform"
  psql_su -d "$d" -c "REVOKE CREATE ON SCHEMA public FROM PUBLIC" -c "ALTER SCHEMA public OWNER TO sf_owner"
done

(cd backend && APP_ENV=local SVC_NAME=migrate LOG_FORMAT=text \
  DATABASE_URL="postgres://sf_owner:${DB_OWNER_PASSWORD:-dev}@localhost:${DB_PORT:-5434}/$a?sslmode=disable" \
  go run ./cmd/migrate up)
{ echo "SET ROLE sf_owner;"; cat docs/schema.sql; } | psql_su -d "$b"

dump() { "${compose[@]}" exec -T db pg_dump -U postgres --schema-only --exclude-table=goose_db_version \
  --exclude-table=goose_db_version_id_seq "$1" | grep -v -E '^(--|SET |SELECT pg_catalog\.set_config|\\(un)?restrict )' | sed '/^$/d'; }
dump "$a" >"$tmp/migrated.sql"
dump "$b" >"$tmp/schema.sql"
if diff -u "$tmp/schema.sql" "$tmp/migrated.sql" >"$tmp/diff"; then
  echo "schema-diff: migrations == docs/schema.sql"
else
  head -60 "$tmp/diff"
  echo "schema-diff: DIFFERENT (left = docs/schema.sql, right = migrations). Fix one of them." >&2
  exit 1
fi
