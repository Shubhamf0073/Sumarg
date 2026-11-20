#!/usr/bin/env bash
# backup.sh — nightly Postgres dump + off-site copy (docs/06 §6). Run by schoolfleet-backup.timer as the deploy user.
set -Eeuo pipefail

COMPOSE_DIR="/srv/schoolfleet/compose"
ENV_FILE="$COMPOSE_DIR/.env"
getenv() { grep -E "^$1=" "$ENV_FILE" | tail -n1 | cut -d= -f2-; }
dc() { docker compose -f "$COMPOSE_DIR/docker-compose.yml" --project-directory "$COMPOSE_DIR" "$@"; }

BACKUP_DIR="$(getenv BACKUP_DIR)"; BACKUP_DIR="${BACKUP_DIR:-/srv/schoolfleet/backups}"
DATA_DIR="$(getenv DATA_DIR)";     DATA_DIR="${DATA_DIR:-/srv/schoolfleet/data}"
DB_ENV_FILE="$COMPOSE_DIR/db.env"
getdbenv() { grep -E "^$1=" "$DB_ENV_FILE" | tail -n1 | cut -d= -f2-; }
PGDB="$(getdbenv SF_DB)"; PGDB="${PGDB:-schoolfleet}"
PGUSER="$(getdbenv POSTGRES_USER)"; PGUSER="${PGUSER:-postgres}"
RCLONE_REMOTE="$(getenv RCLONE_REMOTE)"
PING="$(getenv BACKUP_PING_URL)"

ping_hc() { if [ -n "$PING" ]; then curl -fsS --max-time 10 "$PING$1" >/dev/null || true; fi; }
on_error() { echo "BACKUP FAILED at line $1" >&2; ping_hc "/fail"; exit 1; }
trap 'on_error $LINENO' ERR
umask 077

ts="$(TZ=Asia/Kolkata date +%Y%m%d-%H%M)"
mkdir -p "$BACKUP_DIR/daily" "$BACKUP_DIR/weekly" "$BACKUP_DIR/monthly"
out="$BACKUP_DIR/daily/schoolfleet-$ts.dump"

# 1. Dump (custom format, compressed) and verify it is readable
dc exec -T db pg_dump -U "$PGUSER" -d "$PGDB" -Fc -Z 6 > "$out.partial"
[ -s "$out.partial" ] || { echo "empty dump" >&2; false; }
dc exec -T db pg_restore --list < "$out.partial" > /dev/null
mv "$out.partial" "$out"

# 2. Weekly (Sunday) and monthly (1st) copies, IST calendar
[ "$(TZ=Asia/Kolkata date +%u)" = "7" ]  && cp "$out" "$BACKUP_DIR/weekly/"
[ "$(TZ=Asia/Kolkata date +%d)" = "01" ] && cp "$out" "$BACKUP_DIR/monthly/"

# 3. Local retention: 14 daily, 8 weekly, 6 monthly
find "$BACKUP_DIR/daily"   -name 'schoolfleet-*.dump' -mtime +14  -delete
find "$BACKUP_DIR/weekly"  -name 'schoolfleet-*.dump' -mtime +56  -delete
find "$BACKUP_DIR/monthly" -name 'schoolfleet-*.dump' -mtime +186 -delete
find "$BACKUP_DIR" -name '*.partial' -mmin +120 -delete

# 4. Off-site to an rclone *crypt* remote (encrypted before upload). Copy, never sync:
#    a compromised server must not be able to propagate deletions with one command.
if [ -n "$RCLONE_REMOTE" ]; then
  remote="${RCLONE_REMOTE%/}"
  for d in daily weekly monthly; do
    rclone copy "$BACKUP_DIR/$d" "$remote/db/$d" --include 'schoolfleet-*.dump' --transfers 2
  done
  rclone delete "$remote/db/daily"   --min-age 15d
  rclone delete "$remote/db/weekly"  --min-age 57d
  rclone delete "$remote/db/monthly" --min-age 187d
  rclone copy "$DATA_DIR/files" "$remote/files" --exclude '/pdf/**' --transfers 4
fi

date -u +%FT%TZ > "$BACKUP_DIR/last_success"
ping_hc ""
echo "backup OK: $out ($(du -h "$out" | cut -f1))${RCLONE_REMOTE:+, copied off-site}"
