#!/usr/bin/env bash
# restore.sh — restore a pg_dump (-Fc) file (docs/06 §6).
#   restore.sh <file.dump>                      → into scratch DB sf_restore_test, prints row counts (drill)
#   restore.sh <file.dump> --into-production    → restores into a new DB, then swaps it in for production;
#                                                 the old production DB is kept as <db>_old_<timestamp>
set -Eeuo pipefail

COMPOSE_DIR="/srv/schoolfleet/compose"
dc() { docker compose -f "$COMPOSE_DIR/docker-compose.yml" --project-directory "$COMPOSE_DIR" "$@"; }
DB_ENV_FILE="$COMPOSE_DIR/db.env"
getdbenv() { grep -E "^$1=" "$DB_ENV_FILE" | tail -n1 | cut -d= -f2-; }
PGDB="$(getdbenv SF_DB)"; PGDB="${PGDB:-schoolfleet}"
PGUSER="$(getdbenv POSTGRES_USER)"; PGUSER="${PGUSER:-postgres}"
psql_admin() { dc exec -T db psql -U "$PGUSER" -d postgres -v ON_ERROR_STOP=1 -qAt -c "$1"; }

file="${1:?usage: restore.sh <file.dump> [--into-production]}"
mode="${2:-drill}"
[ -s "$file" ] || { echo "no such dump: $file" >&2; exit 1; }
ts="$(date +%Y%m%d%H%M%S)"

target="sf_restore_test"
if [ "$mode" = "--into-production" ]; then
  target="${PGDB}_restore_${ts}"
  echo "This REPLACES production database '$PGDB' with $file."
  read -r -p "Type the database name ($PGDB) to continue: " answer
  [ "$answer" = "$PGDB" ] || { echo "aborted"; exit 1; }
fi

echo "restore: creating $target"
psql_admin "DROP DATABASE IF EXISTS \"$target\";"
psql_admin "CREATE DATABASE \"$target\" OWNER sf_owner;"
psql_admin "REVOKE ALL ON DATABASE \"$target\" FROM PUBLIC; GRANT CONNECT ON DATABASE \"$target\" TO sf_app, sf_platform;"
# Keep owners/grants/RLS from the dump: the sf_* roles exist in this cluster (01-roles.sh).
dc exec -T db pg_restore -U "$PGUSER" -d "$target" --exit-on-error < "$file"

echo "restore: row counts in $target"
dc exec -T db psql -U "$PGUSER" -d "$target" -qAt -F $'\t' -c "
  SELECT 'tenants', count(*) FROM tenants UNION ALL
  SELECT 'campuses', count(*) FROM campuses UNION ALL
  SELECT 'users', count(*) FROM users UNION ALL
  SELECT 'students', count(*) FROM students UNION ALL
  SELECT 'guardians', count(*) FROM guardians UNION ALL
  SELECT 'routes', count(*) FROM routes UNION ALL
  SELECT 'trips', count(*) FROM trips UNION ALL
  SELECT 'boardings', count(*) FROM boardings UNION ALL
  SELECT 'bus_units', count(*) FROM bus_units UNION ALL
  SELECT 'audit_log', count(*) FROM audit_log UNION ALL
  SELECT 'platform_invoices', count(*) FROM platform_invoices UNION ALL
  SELECT 'latest boarding', max(at)::text FROM boardings;"

if [ "$mode" != "--into-production" ]; then
  echo "drill OK. Drop it when done:  docker compose exec db psql -U $PGUSER -d postgres -c 'DROP DATABASE $target;'"
  exit 0
fi

echo "restore: stopping api + worker"
dc stop api worker
psql_admin "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '$PGDB' AND pid <> pg_backend_pid();" >/dev/null
psql_admin "ALTER DATABASE \"$PGDB\" RENAME TO \"${PGDB}_old_${ts}\";"
psql_admin "ALTER DATABASE \"$target\" RENAME TO \"$PGDB\";"
dc start api worker
echo "restore: production now runs on the restored data. Previous DB kept as ${PGDB}_old_${ts}."
