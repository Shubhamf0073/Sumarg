#!/usr/bin/env bash
# deploy.sh — deploy a tagged release, or roll back to the previous one (docs/06 §5).
#   deploy.sh v0.3.0      # expects /srv/schoolfleet/web/incoming/schoolfleet-web-v0.3.0.tgz (uploaded by CI)
#   deploy.sh rollback    # previous tag's images + SPA (DB stays migrated: expand/contract rule)
# Run as the deploy user (member of the docker group).
set -Eeuo pipefail

ROOT="/srv/schoolfleet"
COMPOSE_DIR="$ROOT/compose"
WEB="$ROOT/web"
STATE="$COMPOSE_DIR/.deploy-state"
ENV_FILE="$COMPOSE_DIR/.env"

die() { echo "deploy: $*" >&2; exit 1; }
log() { echo "deploy: $*"; }
dc()  { docker compose -f "$COMPOSE_DIR/docker-compose.yml" --project-directory "$COMPOSE_DIR" "$@"; }
getenv() { grep -E "^$1=" "$ENV_FILE" | tail -n1 | cut -d= -f2-; }
set_tag() { sed -i -E "s/^IMAGE_TAG=.*/IMAGE_TAG=$1/" "$ENV_FILE"; }

API_PORT="$(getenv API_PORT)"; API_PORT="${API_PORT:-8720}"
healthy() {
  for _ in $(seq 1 30); do
    curl -fsS --max-time 3 "http://127.0.0.1:${API_PORT}/api/healthz" >/dev/null 2>&1 && return 0
    sleep 2
  done
  return 1
}

[ -f "$ENV_FILE" ] || die "missing $ENV_FILE (run bootstrap.sh first)"
mkdir -p "$STATE"
arg="${1:?usage: deploy.sh <tag>|rollback}"
current="$(cat "$STATE/current" 2>/dev/null || true)"
previous="$(cat "$STATE/previous" 2>/dev/null || true)"

if [ "$arg" = "rollback" ]; then
  [ -n "$previous" ] || die "no previous release recorded"
  tag="$previous"
  log "rolling back $current → $tag"
else
  tag="$arg"
fi
[[ "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$ ]] || die "tag must look like v1.2.3, got '$tag'"

# 1. SPA files for this tag
if [ ! -f "$WEB/releases/$tag/index.html" ]; then
  tarball="$WEB/incoming/schoolfleet-web-$tag.tgz"
  [ -f "$tarball" ] || die "missing $tarball"
  rm -rf "$WEB/releases/$tag.tmp"
  mkdir -p "$WEB/releases/$tag.tmp"
  tar -xzf "$tarball" -C "$WEB/releases/$tag.tmp"
  [ -f "$WEB/releases/$tag.tmp/index.html" ] || die "tarball has no index.html at its root"
  chmod -R a+rX "$WEB/releases/$tag.tmp"
  rm -rf "$WEB/releases/$tag"
  mv "$WEB/releases/$tag.tmp" "$WEB/releases/$tag"
  rm -f "$tarball"
  log "unpacked SPA $tag"
fi

# 2. Images
set_tag "$tag"
dc pull api worker
log "pulled images for $tag"

# 3. Migrations (backward-compatible by rule; skipped on rollback)
if [ "$arg" != "rollback" ]; then
  # Only this one-off container gets the schema-owner URL (password from db.env, passed via env — not argv).
  owner_pw="$(grep -E '^DB_OWNER_PASSWORD=' "$COMPOSE_DIR/db.env" | tail -n1 | cut -d= -f2-)"
  [ -n "$owner_pw" ] || die "DB_OWNER_PASSWORD missing in db.env"
  db_name="$(getenv DB_NAME)"
  DATABASE_URL="postgres://sf_owner:${owner_pw}@db:5432/${db_name:-schoolfleet}?sslmode=disable"
  export DATABASE_URL
  dc run --rm -e SVC_NAME=migrate -e DATABASE_URL --entrypoint /app/migrate api up
  unset DATABASE_URL owner_pw
  log "migrations applied"
fi

# 4. Restart app containers
dc up -d api worker

# 5. Verify, then switch the SPA; otherwise restore the previous release
if healthy; then
  ln -sfn "releases/$tag" "$WEB/current.tmp"
  mv -Tf "$WEB/current.tmp" "$WEB/current"
  if [ "$tag" != "$current" ]; then
    [ -n "$current" ] && echo "$current" > "$STATE/previous"
    echo "$tag" > "$STATE/current"
  fi
  # keep the 5 newest SPA releases; prune only Sumarg-labelled dangling images
  find "$WEB/releases" -mindepth 1 -maxdepth 1 -type d -printf '%T@ %p\n' | sort -rn | tail -n +6 | cut -d' ' -f2- | xargs -r rm -rf
  docker image prune -f --filter "label=com.previa.project=schoolfleet" >/dev/null
  log "OK — $tag is live"
else
  log "health check FAILED for $tag"
  dc logs --tail 80 api worker || true
  if [ -n "$current" ] && [ "$current" != "$tag" ]; then
    set_tag "$current"
    dc up -d api worker
    if healthy; then log "restored $current (SPA unchanged)"; else log "previous release $current is ALSO unhealthy — investigate now"; fi
  fi
  exit 1
fi
