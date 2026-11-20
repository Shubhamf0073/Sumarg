#!/usr/bin/env bash
# bootstrap.sh — one-time, idempotent server setup for Sumarg (docs/06 §3).
# Run from a checkout of this repo (or a copy of infra/) on the server:
#   sudo APP_DOMAIN=sumarg.previasoftwares.com WEB_GROUP=www-data bash infra/scripts/bootstrap.sh
# Prints every change. Does NOT touch firewall rules, other sites, Docker settings,
# and does NOT reload Nginx.
set -Eeuo pipefail

[ "$(id -u)" -eq 0 ] || { echo "run with sudo" >&2; exit 1; }

INFRA="$(cd "$(dirname "$0")/.." && pwd)"
APP_DOMAIN="${APP_DOMAIN:?set APP_DOMAIN, e.g. sumarg.previasoftwares.com}"
API_PORT="${API_PORT:-8720}"
WEB_GROUP="${WEB_GROUP:-www-data}"
DEPLOY_USER="${DEPLOY_USER:-sfdeploy}"
ROOT="/srv/schoolfleet"
APP_UID=10001

say() { printf '  • %s\n' "$*"; }

echo "== Pre-checks"
command -v docker >/dev/null || { echo "Docker is not installed. Install Docker Engine + compose plugin from docs.docker.com, then re-run." >&2; exit 1; }
docker compose version >/dev/null 2>&1 || { echo "docker compose plugin missing." >&2; exit 1; }
command -v nginx >/dev/null || { echo "Nginx not found. See docs/06 §1 for the Apache/panel paths." >&2; exit 1; }
getent group "$WEB_GROUP" >/dev/null || { echo "group $WEB_GROUP not found (Nginx group? try WEB_GROUP=nginx)" >&2; exit 1; }
WEB_GID="$(getent group "$WEB_GROUP" | cut -d: -f3)"
if ss -ltn 2>/dev/null | awk '{print $4}' | grep -Eq "[:.]${API_PORT}\$" && ! docker ps --format '{{.Names}}' | grep -q '^schoolfleet-api-'; then
  echo "port $API_PORT is already used by something else; set API_PORT=<free port>" >&2; exit 1
fi
say "docker ok, nginx ok, web group $WEB_GROUP (gid $WEB_GID), api port $API_PORT"

echo "== Deploy user"
if ! id "$DEPLOY_USER" >/dev/null 2>&1; then
  useradd --system --create-home --shell /bin/bash "$DEPLOY_USER"
  say "created user $DEPLOY_USER"
fi
usermod -aG docker "$DEPLOY_USER"
install -d -m 700 -o "$DEPLOY_USER" -g "$DEPLOY_USER" "/home/$DEPLOY_USER/.ssh"
touch "/home/$DEPLOY_USER/.ssh/authorized_keys"
chown "$DEPLOY_USER:$DEPLOY_USER" "/home/$DEPLOY_USER/.ssh/authorized_keys"
chmod 600 "/home/$DEPLOY_USER/.ssh/authorized_keys"
say "$DEPLOY_USER in group docker; add the GitHub Actions public key to /home/$DEPLOY_USER/.ssh/authorized_keys"

echo "== Directories"
install -d -m 751 -o "$DEPLOY_USER" -g "$DEPLOY_USER" "$ROOT"
install -d -m 750 -o "$DEPLOY_USER" -g "$DEPLOY_USER" "$ROOT/compose" "$ROOT/compose/scripts" "$ROOT/compose/.deploy-state"
install -d -m 755 -o "$DEPLOY_USER" -g "$DEPLOY_USER" "$ROOT/web" "$ROOT/web/releases"
install -d -m 750 -o "$DEPLOY_USER" -g "$DEPLOY_USER" "$ROOT/web/incoming"
install -d -m 751 -o root -g root "$ROOT/data"
install -d -m 700 -o root -g root "$ROOT/data/pg"                      # postgres entrypoint chowns it
install -d -m 750 -o "$APP_UID" -g "$WEB_GID" "$ROOT/data/secrets"    # FCM service-account JSON (chmod 600)
install -d -m 2750 -o "$APP_UID" -g "$WEB_GID" "$ROOT/data/files"      # app writes, nginx reads
install -d -m 700 -o "$DEPLOY_USER" -g "$DEPLOY_USER" "$ROOT/backups" "$ROOT/backups/daily" "$ROOT/backups/weekly" "$ROOT/backups/monthly"
install -d -m 700 -o root -g root /etc/ssl/schoolfleet
say "$ROOT tree ready"

echo "== Compose + scripts"
install -m 640 -o "$DEPLOY_USER" -g "$DEPLOY_USER" "$INFRA/docker-compose.yml" "$ROOT/compose/docker-compose.yml"
install -d -m 755 "$ROOT/compose/postgres-init"
install -m 755 "$INFRA/postgres/init/01-roles.sh" "$ROOT/compose/postgres-init/01-roles.sh"   # read by the db container at first init
for s in deploy.sh backup.sh restore.sh update-cf-ips.sh render-nginx.sh discover.sh; do
  install -m 750 -o "$DEPLOY_USER" -g "$DEPLOY_USER" "$INFRA/scripts/$s" "$ROOT/compose/scripts/$s"
done
say "copied docker-compose.yml and scripts"

echo "== .env and db.env"
ENV="$ROOT/compose/.env"
DBENV="$ROOT/compose/db.env"
if [ -f "$ENV" ] || [ -f "$DBENV" ]; then
  say ".env / db.env exist — left unchanged"
else
  app_pw="$(openssl rand -hex 24)"; platform_pw="$(openssl rand -hex 24)"
  tmp="$(mktemp)"; tmpdb="$(mktemp)"
  cp "$INFRA/.env.example" "$tmp"; cp "$INFRA/db.env.example" "$tmpdb"
  sed -i -E "s/^DB_APP_PASSWORD=.*/DB_APP_PASSWORD=${app_pw}/; s/^DB_PLATFORM_PASSWORD=.*/DB_PLATFORM_PASSWORD=${platform_pw}/" "$tmp" "$tmpdb"
  sed -i "s/__GENERATE_KEY__/$(openssl rand -hex 32)/" "$tmp"
  for f in "$tmp" "$tmpdb"; do
    while grep -q '__GENERATE__' "$f"; do sed -i "0,/__GENERATE__/s//$(openssl rand -hex 24)/" "$f"; done
  done
  sed -i -E "s/^APP_DOMAIN=.*/APP_DOMAIN=${APP_DOMAIN}/; s/^API_PORT=.*/API_PORT=${API_PORT}/; s/^WEB_GID=.*/WEB_GID=${WEB_GID}/; s#^DATA_DIR=.*#DATA_DIR=${ROOT}/data#; s#^BACKUP_DIR=.*#BACKUP_DIR=${ROOT}/backups#" "$tmp"
  install -m 600 -o "$DEPLOY_USER" -g "$DEPLOY_USER" "$tmp" "$ENV"
  install -m 600 -o "$DEPLOY_USER" -g "$DEPLOY_USER" "$tmpdb" "$DBENV"
  rm -f "$tmp" "$tmpdb"
  say "created $ENV and $DBENV with generated secrets — set API_IMAGE, Google keys, PREVIA_* in .env"
  say "store SECRETS_KEY and the db.env passwords in Previa's password manager now"
fi

echo "== Nginx site (not reloaded)"
APP_DOMAIN="$APP_DOMAIN" API_PORT="$API_PORT" bash "$INFRA/scripts/render-nginx.sh" "$INFRA/nginx"

echo "== Timers: backup (installed, not enabled)"
for u in schoolfleet-backup.service schoolfleet-backup.timer; do
  install -m 644 "$INFRA/systemd/$u" "/etc/systemd/system/$u"
done
sed -i "s/^User=.*/User=${DEPLOY_USER}/; s/^Group=.*/Group=${DEPLOY_USER}/" /etc/systemd/system/schoolfleet-backup.service
systemctl daemon-reload

echo "== Units host certificate (devices.${APP_DOMAIN}, DNS-only record must exist first)"
if command -v certbot >/dev/null; then
  say "run: certbot certonly --webroot -w /var/www/html -d devices.${APP_DOMAIN} --deploy-hook 'nginx -s reload; touch /etc/letsencrypt/live/devices.${APP_DOMAIN}/fullchain.pem'"
else
  say "certbot not installed — install it, then issue the devices.${APP_DOMAIN} certificate (docs/06 §2)"
fi
say "firewall: allow 8883/tcp for the bus units (ufw allow 8883/tcp) — docs/06 §2"
command -v rclone >/dev/null || say "rclone not installed — install it before enabling off-site backups"
say "enable after configuring rclone: systemctl enable --now schoolfleet-backup.timer"

cat <<EOF

Bootstrap done. Remaining manual steps (docs/06 §3):
  1. Cloudflare DNS (proxied) + Origin certificate → /etc/ssl/schoolfleet/origin.pem, origin.key (chmod 600)
     then: sudo bash $ROOT/compose/scripts/update-cf-ips.sh
  2. sudo nginx -t && sudo systemctl reload nginx
  3. Edit $ENV: API_IMAGE=ghcr.io/<org>/schoolfleet-backend, GOOGLE_MAPS_*_KEY, PREVIA_* invoice details
  4. sudo -u $DEPLOY_USER docker login ghcr.io -u <github-user>   (read:packages token)
  5. Add the deploy public key to /home/$DEPLOY_USER/.ssh/authorized_keys; push tag v0.1.0
  6. First Previa superadmin (prints a one-time passkey enrolment link, valid 24 h):
       cd $ROOT/compose && sudo -u $DEPLOY_USER docker compose run --rm --entrypoint /app/migrate api create-admin --name "…" --email "…"
     (the command needs SVC_NAME=migrate and the owner DATABASE_URL — same env as the migrate step in deploy.sh)
     Then onboard tenants (MTC first) from https://${APP_DOMAIN}/admin — no tenant data is seeded by CLI.
  7. rclone config (crypt remote) → set RCLONE_REMOTE, BACKUP_PING_URL in .env → systemctl enable --now schoolfleet-backup.timer
     → run once: sudo systemctl start schoolfleet-backup.service → restore drill: scripts/restore.sh <dump>
  8. From another machine: nc -zv <server-ip> 8720 and 5432 must FAIL; https://${APP_DOMAIN}/api/healthz → ok
EOF
