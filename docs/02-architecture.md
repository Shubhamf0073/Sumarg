# 02 — Architecture and decisions

## 1. System

```
 Bus (per vehicle)                         Shared VPS (4 vCPU / 8 GB, with Previa Cargo)          People
 ┌───────────────────────────┐   MQTT/TLS  ┌──────────────────────────────────────────────┐
 │ SB unit (ESP32-S3)        │────:8883───▶│ ingest  `schoolfleet serve --mode=ingest`    │
 │ 4G · GNSS · RFID · camera │  DNS-only   │   embedded MQTT broker (mochi) + pipeline    │
 │ SOS buttons · SD journal  │  host       │   positions · taps · photos · SOS · status   │
 │ OTA A/B, secure boot      │◀─HTTPS /fw/─│ Nginx (devices.<domain>, Let's Encrypt)       │
 └───────────────────────────┘             │                                              │
  (school's AIS-140 tracker stays          │ api     `--mode=api`  /api, /api/admin       │◀── Cloudflare ── portal (web)
   for the RTO — A29)                      │ worker  `--mode=worker` push/SMS, retention,  │◀── Cloudflare ── parent + driver app
                                           │         partitions, invoices, offline checks │      (Expo, Android + iOS)
                                           │ Postgres 17 (forced RLS) · files on disk      │──▶ FCM (push) · SMS gateway (OTP)
                                           └──────────────────────────────────────────────┘──▶ Google Routes (route setup only)
```

Live path: unit publishes a fix every 10 s while moving → ingest validates, stores the raw row (daily partition),
updates its in-memory live map, `NOTIFY sf_live` → api keeps the same map in memory → parent app polls
`/api/parent/live/{route}` every 10 s while the tracking screen is open (one map lookup, no SQL). `vehicle_live`
is written at most every 30 s so a restart starts warm.

Tap path: card tap → unit decides board/alight locally (instant beep + first name on the display) → journal →
`TAP` event + photo → ingest re-checks against the trip and roster → `boardings` row + `notifications` rows in one
transaction → worker sends FCM pushes ("Aarav boarded Bus 12 at 7:42 at Maple Street").

## 2. Decisions (ADR log)

Changing a row = edit this table first with the reason. Dependencies only from `.claude/approved-deps.txt`.
Rows marked *(Cargo)* are the same decision as Previa Cargo, kept identical so both products share code and habits.

| # | Area | Decision | Why | Rejected |
|---|---|---|---|---|
| A1 | School portal | React 19 + Vite + TS 5.9, React Router, TanStack Query, demo CSS ported *(Cargo)* | Demo is the spec | Next.js |
| A2 | **Mobile** | **One Expo (React Native) app** for Android + iOS with two experiences (parent, driver/attendant) chosen at login; expo-router, TanStack Query, openapi-fetch with types generated from the same OpenAPI as the portal, `react-native-maps` (Google provider), `expo-notifications` (native FCM/APNs tokens), `expo-secure-store` for the session. EAS builds | One codebase ≈ half the work of Kotlin + Swift; same TS types and query patterns as the portal; native map + push | Two native apps (2× effort; planned earlier, user chose one codebase 19 Nov); PWA (weak iOS background push) |
| A3 | API contract | Code-first OpenAPI 3.1 with huma v2 on chi; generated TS for portal **and** app *(Cargo)* | One source of truth | Hand-written types |
| A4 | Backend | Go 1.27, one binary `schoolfleet serve --mode=api\|worker\|ingest\|all` + `migrate`; pgx + sqlc, goose, slog via `platform/obs` *(Cargo)*. Three containers in production | Small RAM on a shared box; ingest restarts don't touch the API | Python; separate microservices |
| A5 | Database | PostgreSQL 17; positions in **daily partitions** dropped after `position_retention_days` (default 30); trips keep a simplified path forever | ~70k fixes/school/day; dropping a partition is free, `DELETE` is not | TimescaleDB/PostGIS (Previa Fleet's choice — more ops on a shared box; not needed for point-in-radius maths) |
| A6 | Background work | `notifications` table as the outbox + worker (SKIP LOCKED, backoff) *(Cargo pattern)* | Atomic with the boarding row | Redis/queues |
| A7 | **Device link** | **MQTT 3.1.1 over TLS :8883**, protocol **v1 from Previa Fleet** (topics `v1/t/{tk}/d/{id}/…`, JSON, QoS 1, persistent sessions, retained config/status) + Sumarg event kinds and a photo topic (docs/13) | Persistent connection: ~22 MB/month for positions vs. TLS handshakes per HTTPS post; server can push config/OTA; shared with Previa Fleet firmware | HTTPS polling (data + battery); raw TCP |
| A8 | **Broker** | **Embedded broker in the ingest process** (`mochi-mqtt/server/v2`): auth hook = argon2id check of the unit secret (cached per connection), ACL hook = own topics only, publish hook → pipeline. Same topics as EMQX would use, so moving to EMQX later changes no firmware | One process instead of EMQX (≈ 300 MB+ RAM) + auth webhook; device auth lives in our code and DB | EMQX (Previa Fleet's choice — right for a dedicated box, heavy on a shared one); Mosquitto + plugin |
| A9 | **Attendance** | **RFID tap (13.56 MHz card) + a photo per tap** from the unit's camera, stored `photo_retention_days` (default 30). Board/alight decided on the unit for instant feedback, re-checked by the server. Manual mark (and QR on the ID card) in the driver app when a card fails. No face recognition, no biometric templates | Cheap, fast (40 children in minutes), works on ESP32-S3; photo deters card swapping; no biometric consent burden (DPDP) | Face recognition (stronger hardware, parental biometric consent, accuracy with small children) — Phase 3 option on the stored photos |
| A10 | Notifications | Push via **FCM HTTP v1** for Android and iOS (APNs key uploaded to Firebase), sent by the worker; SMS only for OTP and optional fallback (DLT-registered gateway) | FCM is free; one sender path | Expo push service (extra third party for children's data); WhatsApp (per-message cost, template approvals) |
| A11 | Files | Local disk under `<tenant_id>/…`, API authorises → Nginx X-Accel *(Cargo)*; photos expire by `files.expires_at` | No object-store cost at this size | S3/R2 live storage |
| A12 | **Maps** | Google: **Maps SDK in the app (no charge, unlimited)**, Maps JavaScript in the portal (lazy, free monthly cap), Routes API once when stops change (route polyline + planned offsets). ETA computed by us from learned stop-to-stop times — no per-trip API calls | Mobile maps cost ₹0; ETA without per-request cost | Routes/ETA API per poll; Mapbox |
| A13 | Runtime | Docker Compose behind host Nginx *(Cargo)*; separate compose project and Postgres container from Cargo on the same box | Independent deploys/restores per product | Shared Postgres between products (tighter coupling) |
| A14 | Edge | `schoolfleet.<domain>` proxied by Cloudflare (Full strict); **`devices.schoolfleet.<domain>` DNS-only** (grey cloud) for MQTT :8883 and OTA downloads, Let's Encrypt cert, ISRG Root X1 pinned in firmware | Cloudflare proxies HTTP only (Spectrum for TCP is paid) | Cloudflare Spectrum; self-signed device CA (Phase 2 option for mutual TLS) |
| A15 | CI/CD | GitHub Actions: Go lint/test/integration, portal + app typecheck, `pio test -e native` + `pio run`, image to GHCR, SSH deploy; EAS for app builds *(Cargo + firmware)* | | |
| A16 | Logging | slog JSON with redaction; unit `status` messages + last-seen drive the "unit offline" alert *(Cargo)* | | |
| A17 | Lint | golangci-lint with depguard architecture rules *(Cargo)*; Biome; `-Wall -Wextra -Werror` for firmware logic | | |
| A18 | Tests | Go integration with two tenants; firmware logic on the host; **device simulator** (from Previa Fleet `tools/simulator`) drives virtual buses over MQTT for end-to-end tests | Hardware-free testing of the whole tap→push path | |
| A19 | Tenancy | Shared schema + composite FKs + forced RLS; tenant = school organisation, campuses inside *(Cargo)* | | |
| A20 | DB roles | `sf_owner` / `sf_app` / `sf_platform`; pools refuse the wrong role. SECURITY DEFINER only: `tenant_by_code`, and the two partition-maintenance functions (owner-run DDL for the worker) | | |
| A21 | Staff auth | Portal: school code + mobile + password, server sessions *(Cargo A9/A27)*. Driver/attendant app: school code + mobile + 6-digit PIN, session bound to the app install | Drivers share phones rarely; PIN is usable at 6:30 am | OTP for drivers (SMS cost, flaky mornings) |
| A22 | **Parent auth** | School code (from the school's invite link/QR) + mobile + **SMS OTP** (6 digits, 5 min, 5 tries, HMAC-stored) → 90-day session bound to the app install, renewed on use. A parent with children in two schools adds the second school in the app (separate session) | No passwords for thousands of parents; ~2 OTPs/parent/year | Passwords; cross-tenant mobile lookup (oracle, ambiguity — A27 in Cargo) |
| A23 | Unit identity | Unit uid `SB-XXXXXX` from the ESP32 MAC; 32-char random secret generated at provisioning, stored in **encrypted NVS**, argon2id hash in `bus_units` (Previa Fleet `devicesecret`) | Revocable per unit; no shared keys | One fleet-wide key; X.509 per unit (Phase 2 option with flash encryption) |
| A24 | Superadmin | Passkeys only, `/admin` lazy chunk, platform pool *(Cargo)*; plus unit inventory, provisioning, firmware releases, agents and commissions | | |
| A25 | **Firmware** | ESP32-S3 (N16R8), **PlatformIO, Arduino-ESP32 3.x as an ESP-IDF 5 component** (Previa Fleet's toolchain); FreeRTOS tasks; journal-first storage on SD; A/B OTA with rollback, images signed (ECDSA P-256) and verified on the unit; **Secure Boot v2 + flash encryption** at provisioning (docs/14) | Reuses the team's Arduino code style while getting IDF security features | Pure Arduino precompiled (no secure boot control); ESP-IDF only (slower for the team) |
| A26 | **Hardware** | No prior design exists, so three steps: **SB-P0 bench** (Waveshare ESP32-S3-SIM7670G-4G dev board + reader + display + buttons on wires) for firmware now → **SB-C1 carrier** (the same Waveshare board on a 2-layer carrier PCB we design: automotive power, protected inputs, harness/panel connectors) for the first schools → **SB-R1 integrated** 4-layer board once SB-C1 has field data (docs/12) | First PCB has no modem/RF layout, so it can be designed in ~2 weeks and work first time; schools start on hardware we control | Custom integrated board first (RF risk, months); dev board in a box (no automotive protection) |
| A27 | Tenant before RLS | Token `<tenant_id>.<token>` for portal cookie and app bearer token; school code at login *(Cargo A27)* | | |
| A28 | App experiences | One app binary; the session's principal (guardian vs crew user) decides the tab set | One store listing, one review | Two apps |
| A29 | **AIS-140** | Phase 1: our unit is an **add-on**; the bus keeps its certified AIS-140 tracker for the RTO. The SB-R1 design stays **AIS-140-ready** (NavIC GNSS, ≥ 4 h battery, tamper input, wired emergency buttons, second-server reporting in firmware) so certification (ARAI/ICAT type approval + state backend tie-ups) can follow as Phase 3 (docs/12 §6) | Certification takes months and lakhs; schools already have trackers | Certifying before launch |
| A30 | Children's data | Previa is the school's **data processor**; processing limited to location during travel and boarding events/photos for safety (DPDP Rules 2025, Fourth Schedule exemptions for schools and their transport). Short retention, first-name-only on the unit, no ads/analytics SDKs in the app, DPA with each school (docs/07 §3) | Lawful basis without collecting parental consent for location; least data | Tracking outside trips; third-party analytics |
| A31 | **PCB tooling** | **KiCad 10** (open source, `kicad-cli` for ERC/DRC/exports in CI). A human designer owns the KiCad files; Claude reviews **exports** (netlist summary, pin table, ERC/DRC JSON, BOM) via `/hw-design-review` and never edits KiCad sources (hook-blocked) | Free, scriptable, any freelance designer knows it; reviews cost few tokens | Altium/EasyEDA (licence or lock-in); code-first PCB tools (atopile/SKiDL — team unfamiliar, layout still manual) |
| A32 | **Agent commission** | Paid on the first package **and every renewal** (`commission_mode = 'recurring'`), on paid invoices only, never negative (docs/10 §5) | Agent stays engaged in renewals and collections | First-year-only (cheaper, decided against on 20 Nov) |

Approved Go modules beyond Cargo's: `mochi-mqtt/server/v2` (A8), `santhosh-tekuri/jsonschema/v6` (protocol validation,
from Previa Fleet), `golang.org/x/oauth2` (FCM service-account token), `go-webauthn/webauthn` v0.18 (A24 admin
passkeys; Cargo turned out to have none, M1.4a). Firmware libraries: ArduinoJson, TinyGSM,
MFRC522 (or Adafruit PN532), U8g2, esp32-camera. App: see `.claude/approved-deps.txt`.

## 3. Request flows

**Portal / app request:** parse token `<tenant_id>.<token>` → `db.InTenant(Read/Write)` → `GetSession` under RLS →
role/campus scope → handler → commit *(Cargo)*. Parent endpoints additionally check that every student/route/trip id
in the request belongs to one of the guardian's children (`student_guardians`) — out of scope → 404.

**Unit connect:** MQTT CONNECT (client id = uid, username = uid, password = secret) → ingest looks up `bus_units`
(platform pool), argon2id verify (2 at a time), unit must be `installed` or `allocated` → ACL: publish only
`v1/t/{tk}/d/{uid}/{telemetry,status,ack,photo/+}`, subscribe only `…/{config,cmd}` → tenant taken from the DB row,
never from the topic's `{tk}`.

**Tap → push:** see §1. One transaction: boarding + trip_stops (if at a stop) + notifications + alert (unknown card)
+ audit. Pushes go out within ~2 s of the publish when the bus has signal; with no signal the journal replays later
and the notification is marked late (older than 10 min → "Aarav boarded at 7:42 (delayed update)").

## 4. Budgets

| Item | Target |
|---|---|
| Tap → parent push (bus online) | < 5 s p95 |
| Position → parent map | ≤ 15 s (10 s reporting + 10 s polling, worst case) |
| API reads | p95 < 50 ms; ≤ 5 statements; live endpoints served from memory |
| Unit data | ≤ 250 MB/month incl. photos (docs/13 §8) |
| RAM caps (this product) | db 1 GB · api 192 MB · worker 192 MB · ingest 256 MB ≈ **1.65 GB** |
| Connections | api 8 + 3, worker 3, ingest 4 → 18 < `max_connections=30` |

## 5. Layout

```
previa-school-fleet/
├── backend/        Go: cmd/{schoolfleet,migrate} · platform/ (shared with Cargo + protocol, devicesecret, argon2id
│                   from Previa Fleet) · internal/{api,tenant,parent,driver,admin,ingest,worker,shared,store}
│                   · queries/{tenant,platform} · migrations/
├── frontend/       school portal + /admin (React)
├── mobile/         Expo app (parent + driver)
├── firmware/       PlatformIO project for the SB unit
├── tools/simulator Go virtual buses over MQTT (M3, ported from Previa Fleet)
├── infra/ · docs/ · reference/demo/ · .claude/
```
Conventions for Go, SQL, portal and infra are the same as Previa Cargo (`.claude/rules/`); firmware and mobile have
their own rules files.
