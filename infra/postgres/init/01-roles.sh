#!/usr/bin/env bash
# 01-roles.sh — create Sumarg's database roles and database (docs/02 A20).
# Runs automatically ONCE when the Postgres data directory is empty (docker-entrypoint-initdb.d),
# as the superuser. Also used by CI against a service container (set PGHOST/PGPASSWORD).
#   sf_owner     owns the schema, runs goose migrations (cmd/migrate), BYPASSRLS
#   sf_app       tenant requests, subject to Row-Level Security
#   sf_platform  superadmin API + worker, BYPASSRLS
set -euo pipefail

: "${DB_OWNER_PASSWORD:?}" "${DB_APP_PASSWORD:?}" "${DB_PLATFORM_PASSWORD:?}"
SF_DB="${SF_DB:-schoolfleet}"
SUPERUSER="${POSTGRES_USER:-postgres}"

psql -v ON_ERROR_STOP=1 --username "$SUPERUSER" --dbname postgres \
  -v owner_pw="$DB_OWNER_PASSWORD" -v app_pw="$DB_APP_PASSWORD" -v platform_pw="$DB_PLATFORM_PASSWORD" \
  -v db="$SF_DB" <<'SQL'
CREATE ROLE sf_owner    LOGIN BYPASSRLS   PASSWORD :'owner_pw';
CREATE ROLE sf_app      LOGIN NOBYPASSRLS PASSWORD :'app_pw'      CONNECTION LIMIT 30;
CREATE ROLE sf_platform LOGIN BYPASSRLS   PASSWORD :'platform_pw' CONNECTION LIMIT 10;
CREATE DATABASE :"db" OWNER sf_owner;
REVOKE ALL ON DATABASE :"db" FROM PUBLIC;
GRANT CONNECT ON DATABASE :"db" TO sf_app, sf_platform;
SQL

psql -v ON_ERROR_STOP=1 --username "$SUPERUSER" --dbname "$SF_DB" <<'SQL'
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
ALTER SCHEMA public OWNER TO sf_owner;
SQL

echo "schoolfleet roles and database '$SF_DB' created"
