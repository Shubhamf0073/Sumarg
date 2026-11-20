#!/usr/bin/env bash
# render-nginx.sh — write /etc/nginx/conf.d/schoolfleet.conf and snippets from the templates.
# Does NOT reload Nginx: run `sudo nginx -t && sudo systemctl reload nginx` yourself.
#   sudo APP_DOMAIN=sumarg.previasoftwares.com API_PORT=8720 bash render-nginx.sh [path/to/infra/nginx]
set -Eeuo pipefail

APP_DOMAIN="${APP_DOMAIN:?set APP_DOMAIN}"
API_PORT="${API_PORT:-8720}"
SRC="${1:-$(cd "$(dirname "$0")/../nginx" && pwd)}"
DEST_SITE="/etc/nginx/conf.d/schoolfleet.conf"
DEST_SNIP="/etc/nginx/snippets"

[[ "$APP_DOMAIN" =~ ^[a-z0-9.-]+$ ]] || { echo "APP_DOMAIN looks wrong: $APP_DOMAIN" >&2; exit 1; }
[[ "$API_PORT" =~ ^[0-9]{2,5}$ ]] || { echo "API_PORT looks wrong: $API_PORT" >&2; exit 1; }

install -d -m 755 "$DEST_SNIP"
install -m 644 "$SRC/snippets/schoolfleet-proxy.conf"            "$DEST_SNIP/schoolfleet-proxy.conf"
install -m 644 "$SRC/snippets/schoolfleet-security-headers.conf" "$DEST_SNIP/schoolfleet-security-headers.conf"
# Don't clobber a configured admin allow-list
[ -f "$DEST_SNIP/schoolfleet-admin-allow.conf" ] || \
  install -m 644 "$SRC/snippets/schoolfleet-admin-allow.conf" "$DEST_SNIP/schoolfleet-admin-allow.conf"
# Don't clobber Cloudflare ranges written by update-cf-ips.sh
[ -f "$DEST_SNIP/schoolfleet-cloudflare-realip.conf" ] || \
  install -m 644 "$SRC/snippets/schoolfleet-cloudflare-realip.conf" "$DEST_SNIP/schoolfleet-cloudflare-realip.conf"

tmp="$(mktemp)"
sed -e "s/__APP_DOMAIN__/${APP_DOMAIN}/g" -e "s/__API_PORT__/${API_PORT}/g" "$SRC/schoolfleet.conf.template" > "$tmp"
if grep -q '__[A-Z_]*__' "$tmp"; then echo "unreplaced token in template" >&2; rm -f "$tmp"; exit 1; fi
# Nginx refuses to start on "listen [::]:…" when the host has IPv6 disabled. Drop those lines then.
# Override with IPV6=1 / IPV6=0.
if [ "${IPV6:-auto}" = "0" ] || { [ "${IPV6:-auto}" = "auto" ] && [ ! -e /proc/net/if_inet6 ]; }; then
  sed -i '/listen \[::\]/d' "$tmp"
  echo "IPv6 not available — rendered IPv4-only listen lines."
fi
if [ -f "$DEST_SITE" ] && ! cmp -s "$tmp" "$DEST_SITE"; then
  cp "$DEST_SITE" "${DEST_SITE}.bak.$(date +%Y%m%d%H%M%S)"
  echo "Backed up previous $DEST_SITE"
fi
install -m 644 "$tmp" "$DEST_SITE"
rm -f "$tmp"

echo "Wrote $DEST_SITE and snippets in $DEST_SNIP."
[ -f /etc/ssl/schoolfleet/origin.pem ] || echo "NOTE: /etc/ssl/schoolfleet/origin.pem missing — add the certificate before reloading (docs/06 §4)."
echo "Next: sudo nginx -t && sudo systemctl reload nginx"
