#!/usr/bin/env bash
# update-cf-ips.sh — refresh Cloudflare ranges so Nginx sees real client IPs (rate limits, audit).
#   sudo bash update-cf-ips.sh && sudo nginx -t && sudo systemctl reload nginx
set -Eeuo pipefail

OUT="/etc/nginx/snippets/schoolfleet-cloudflare-realip.conf"
tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

v4="$(curl -fsS --max-time 10 https://www.cloudflare.com/ips-v4)"
v6="$(curl -fsS --max-time 10 https://www.cloudflare.com/ips-v6)"
[ "$(printf '%s\n' "$v4" | grep -c .)" -ge 10 ] || { echo "unexpected ips-v4 response" >&2; exit 1; }

{
  echo "# Generated $(date -u +%FT%TZ) by update-cf-ips.sh — do not edit by hand"
  printf '%s\n' "$v4" "$v6" | grep -E '^[0-9a-fA-F:.]+/[0-9]+$' | sed 's/^/set_real_ip_from /; s/$/;/'
  echo "real_ip_header CF-Connecting-IP;"
  echo "real_ip_recursive on;"
} > "$tmp"

install -m 644 "$tmp" "$OUT"
echo "Wrote $OUT ($(grep -c set_real_ip_from "$OUT") ranges)."
