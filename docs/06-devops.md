# 06 — DevOps

Same server, tooling and habits as Previa Cargo (`previa-cargo/docs/06-devops.md`); this page lists only what differs.

## 1. Where it runs
The shared GoDaddy VPS (4 vCPU / 8 GB / 200 GB), as a **separate compose project** `/srv/schoolfleet/compose`
(api, worker, ingest, db) next to Previa Cargo. Host Nginx serves both products (`limit_req` zones and upstreams
are prefixed per product; both configs pass `nginx -t` together). Ports: API `127.0.0.1:8720`, MQTT `0.0.0.0:8883`.
The API must stay reachable **only through Nginx**: it trusts `X-Forwarded-For` (which Nginx overwrites with the
client address) for session audit and the per-IP OTP limits. Never publish the API port on a public interface.

## 2. DNS, certificates, firewall
| Record | Cloudflare | Points to | Used by |
|---|---|---|---|
| `sumarg.previasoftwares.com` | **proxied** (orange) | server | portal, apps' API, admin |
| `devices.sumarg.previasoftwares.com` | **DNS-only** (grey) | server | units: MQTT :8883, OTA `/fw/` on 443 |

- App vhost: Cloudflare Origin certificate (Full strict) *(Cargo)*.
- `devices.`: Let's Encrypt via `certbot certonly --webroot` (HTTP-01 on port 80 for that host) or DNS-01 with a
  Cloudflare API token limited to the zone; `certbot.timer` renews; a deploy hook touches the cert so the ingest
  process reloads it (it watches the file mtime) and runs `nginx -s reload`.
- Firewall (ufw): 22 (key only), 80, 443, 8883 open; 443 on the app vhost restricted to Cloudflare ranges via Nginx
  `allow` list (`update-cf-ips.sh`) because the origin IP is public through `devices.` (docs/07 S10).

## 3. Images and deploy
One image `ghcr.io/<org>/schoolfleet-backend` (≈ 25 MB, distroless): `schoolfleet` + `migrate`. `deploy.sh`:
unpack portal SPA → pull → `migrate up` as sf_owner → `up -d api worker ingest` → `/api/healthz` → switch SPA or roll
back *(Cargo)*. Ingest restart drops unit connections for a few seconds; units reconnect with persistent sessions and
replay from the journal — deploy outside trip windows (09:00–12:00 or 16:00–21:00 IST) anyway.

## 4. Firmware releases
`/release-firmware`: CI builds unsigned `sb_c1` and `sb_proto` images (`sb_r1` once it exists) on tag `fw-vX.Y.Z` → human signs on the offline
machine (`espsecure sign_data` for Secure Boot + ECDSA P-256 OTA signature) → upload in `/admin → Firmware` → files in
`/srv/schoolfleet/data/files/firmware/` → channel dev → beta (5 % of units, 48 h, no rollbacks) → stable. The admin
screen shows per-version online/rollback counts.

## 5. Mobile releases
EAS Build (Android AAB + iOS IPA) from `mobile/`; EAS Submit to Play internal testing / TestFlight → production.
Secrets in EAS (Maps keys, `google-services.json`, `GoogleService-Info.plist`). OTA JS updates (EAS Update) only for
UI fixes; anything touching auth or the API version ships as a store build. Store listings need the privacy policy
URL and a demo login for reviewers (a demo school tenant with fake data).

## 6. Local development
```
make dev-db migrate seed   # Postgres + roles; seed: Previa admin, 2 demo schools, routes, students, cards
make run                   # api + worker + ingest in one process (MQTT on :1883 plain + :8883 with a dev cert)
make sim                   # tools/simulator: virtual buses drive routes and tap cards over MQTT
make web                   # portal on :5173
make app                   # Expo dev server (scan with Expo Go / dev build)
make fw-test               # firmware logic tests on the host (pio test -e native)
```

## 7. Capacity (shared box)
RAM caps ≈ 1.65 GB (docs/02 §4) + Cargo ≈ 1.45 GB + existing apps ≈ 2.6 GB → ≈ 5.7 GB of 8 GB. Disk: photos
≈ 100 MB/school/day at 30 days' retention → 3 GB per school kept; alert at 70 % disk. Move Sumarg to its own VPS
at ~100 schools or when photos exceed 40 GB (docs/10 §6).

## 8. Backups
Postgres nightly encrypted dump to R2 *(Cargo)*, including `positions` partitions of the last 2 days only (older
positions are not worth backing up; trip paths are in `trips`). Photos are not backed up off-site (30-day evidence,
regenerated never — accepted). Firmware images and signing records are backed up separately.

## 9. Runbook additions
| Symptom | Check | Fix |
|---|---|---|
| Many units offline at once | `docker compose logs ingest`, cert expiry (`openssl s_client -connect devices.…:8883`), ufw 8883 | renew cert / restart ingest |
| One unit offline | admin → unit → last status (CSQ, battery, power cut) | call school; installer visit |
| Taps but no pushes | worker logs, FCM errors (`UNREGISTERED` → token cleanup), `notifications` queue depth | restart worker; check FCM service account |
| Wrong boarding direction | trip boarding history, unit reboot events | void + re-mark; check `minRideS` |
| Data breach suspected | freeze deploys, rotate secrets, preserve logs | notify affected schools within 24 h (docs/07 §3) |
