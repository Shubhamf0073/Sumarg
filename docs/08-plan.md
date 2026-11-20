# 08 — Build plan

**Current milestone: M0**

`/next` picks the first unchecked task in the current milestone. Tags: **[opus]** = switch to Opus for this task;
**[review]** = `/review-task` before ticking; **[hw]** = needs the bench board on the desk (human present);
**[human]** = done by a person outside the repo (orders, contracts, registrations) — Claude prepares, never commits spend.
Estimates = focused days for one developer driving Claude Code. Three lanes after M1: backend, app/portal, firmware.

Already done in this folder (docs/11): Go platform packages (from Cargo + Previa Fleet protocol/devicesecret/argon2id),
`platform/db` with 6 passing two-school isolation tests on this schema, goose baseline, sqlc config + first queries,
lint rules, Dockerfile, compose with ingest, Nginx with the devices vhost, firmware project with the boarding ledger
and its host tests, Expo app config.

### M0 — Repo and local loop (0.5 d)
- [x] M0.1 New repo from this folder; `make tools`; `make dev-db migrate test test-integration` green; CI green — backend/Go green locally (sqlc pinned v1.31.1); frontend/mobile typecheck+lint stay red until M0.5 adds tsconfig/biome/src; CI green pending first push
- [x] M0.2 Sumarg config struct (`PLATFORM_DATABASE_URL`, `SECRETS_KEY`, MQTT, FCM, SMS, Maps); `Setup` hook in `platform/app` (same change as Cargo M0.2 — copy it if Cargo did it first) **[review]**
- [x] M0.3 Pools + request tx middleware + readiness (copy from Cargo M0.3) **[review]** — built from platform/db (Cargo repo not on disk); `db.RequestTx` buffers the response and commits before sending
- [x] M0.4 huma + error mapping + `schoolfleet openapi` + `make gen-api` for portal and app — huma v2.40.0; one flat error body with `fields`; spec has no paths until M1 (`/api/healthz` stays plain text for Nginx); biome/tsc stay red until M0.5
- [x] M0.5 Portal skeleton (demo CSS, router, `/admin` lazy) and Expo app skeleton (expo-router, auth stack, two tab sets, API client) — `npx expo install --fix` — portal: tsc/biome/build green, `/admin` lazy chunk; app: tsc/biome/jest green, expo install --fix only bumped react-native 0.81.5; no sign-in or real screens yet (M1.2+)

### M1 — Tenancy, auth, onboarding (2.5 d)
- [x] M1.1 Isolation suite: catalog check (every tenant table forced RLS, no DELETE grant), parent-scope walker (every `/parent/*` route with another guardian's ids → 404) **[opus] [review]** — `internal/isolation`: catalog check (also role not super/bypassrls/owner, UNIQUE (tenant_id, id), composite FKs both sides, SECURITY DEFINER allowlist; verified by injecting faults) + walker with self-tests; real walk in `cmd/schoolfleet` finds 0 parent ops until M1.2 wires `Auth`/middleware
- [x] M1.1a Isolation follow-ups (review minors): sf_app reaches no view/matview without `security_invoker`; partition functions not executable by sf_app + `search_path` set; no FK targets a `noCompositeKey` table; policy `polroles`/`polcmd`/permissive checked; M1.2 must wire walker `Auth` and keep the positive control mandatory
- [x] M1.2 Portal login + sessions (copy Cargo auth), driver PIN login with install binding; wire `isolation.Walk.Auth` in `cmd/schoolfleet/parent_scope_integration_test.go` (real parent session, main's middleware) — never relax the own-id positive control **[opus] [review]** — Cargo (`previalogistic`) has no auth code yet, so `internal/tenant/auth` is new, built to Cargo ADR-0010 + docs/05 §10: login/pin/logout/me, `X-Install-Id` device binding, attempt claimed before verify (no parallel-guess race), one driver session (row lock), `db.Resolver` returns a ctx with the principal, `db.Tenants`, `mountAPI` default-protects every route, `APP_DOMAIN` → portal Origin; portal login page + shell guard; walker signs A/B/C in through `mountAPI` (+ `/api/me` check; fault-injected: leak and all-404 both fail)
- [x] M1.3 Parent OTP login (SMS gateway adapter + stub), rate limits, 90-day app sessions, push-token registration **[opus] [review]** — `internal/tenant/auth` otp + push-token; code HMAC'd (HKDF keys, `internal/shared/otp`), AES-GCM-sealed in the outbox, sent by `internal/worker` (lease claim, send outside tx, 24 h purge; stub + MSG91, 2factor refused at start); per-IP limiter `platform/ratelimit`; `platform/app` Runtime.Workers (copy to Cargo); migration 00003 indexes; app OTP screens + X-Install-Id + push registration; walker signs in via real OTP
- [x] M1.3a OTP follow-ups (review): launch-day burst vs 10 SMS/school/min (raise or lift during onboarding? decide before the first school); even request/verify timing for invited vs unknown mobiles and known vs unknown school; M4.4 must push only to tokens whose session is live (revoked sessions' push_tokens stay unretired); state that a presented token from another school isn't revoked; API reachable only via Nginx (X-Forwarded-For trust, with M1.2a); push-token endpoint has no rate limit; MSG91 request shape to verify on the live account (R9); iOS push token is APNs, not FCM (M4.4/M5.3) **[review]**
- [x] M1.2a Auth follow-ups (review): per-user backoff alone lets an attacker hold one user's login closed — M1.3's per-IP and per-school limits must cover `/auth/login` and `/auth/app/pin` too (or let a correct secret through at a reduced rate); trust `X-Forwarded-For` only from the Nginx hop once per-IP limits use it; `db.Resolved.Ctx` must derive from `r.Context()` (not enforced); support-view sessions (platform user) are refused until built; `.env.example` `PORTAL_SESSION_*`/`APP_SESSION_DAYS` are not read (lifetimes are fixed in code/SQL) **[review]**
- [ ] M1.2b Sign-in follow-ups (M1.2a review): 5 addresses (or a rented IPv6 range's /64s) can still hold a user
  closed for 24 h via the 50/day cap — let a known device (driver install with a past session; a portal device
  cookie) through at a reduced rate; log once when the in-process backoff table is full (fail-open); before deploy,
  `docker network inspect` that 172.30.231.0/24 doesn't overlap Cargo, and after it check sessions record real IPs
  (docs/06 §1) — if the hop isn't `172.30.231.1`, every client shares one per-IP bucket (60 sign-ins/10 min for
  all); the daily-cap warning can log twice in one window (a success takes 1 off the count) — log once per window
  or accept; drop `users.locked_until` (contract, needs approval); M1.4 adds `ADMIN_SESSION_HOURS` back to
  `.env.example` when it is read
- [ ] M1.4 Admin passkeys (copy Cargo), onboarding wizard (school, campuses, settings, owner, package) **[review]** —
  moving a school to `live` sets `go_live_on` to today (IST) if empty, in the same save (the OTP launch window,
  docs/05 §10, M1.3a)
- [ ] M1.5 School setup screens: campuses, vehicles, crew, users

### M2 — Routes, students, cards (2.5 d)
- [ ] M2.1 Routes + stops map editor (portal), Routes API once on save → polyline + offsets **[review]**
- [ ] M2.2 Students CRUD + stop assignment (bulk CSV/XLSX import of all onboarding templates is M9.2)
- [ ] M2.3 Guardians, invites (link + QR), pickup permissions
- [ ] M2.4 RFID cards issue/revoke (desk reader via USB HID in the portal, or type UID), roster versioning per unit
- [ ] M2.5 Daily trip creation job; holidays list

### M3 — Ingest and simulator (3 d)
- [ ] M3.1 Embedded MQTT broker (mochi): TLS with cert reload, auth hook (argon2 semaphore, unknown-uid fast reject), ACL hook, limits **[opus] [review]**
- [ ] M3.2 Pipeline: validate v1 schema (`platform/protocol`), positions into partitions, live map + `NOTIFY`, `vehicle_live` every 30 s, status → `bus_units` **[review]**
- [ ] M3.3 Protocol additions (TAP, SOS, TRIP_*, photo topic, roster/trip config) via `/device-protocol` **[review]**
- [ ] M3.4 TAP handling: server-side direction, boarding + photo file + notifications in one tx; unknown card alert **[opus] [review]**
- [ ] M3.5 Config publisher (retained): roster + routes + tap settings per unit when data changes; commands queue (`unit_commands`)
- [ ] M3.6 Simulator (port Previa Fleet `tools/simulator`): virtual buses follow route polylines, tap roster cards, SOS, power cut, offline replay; `make sim` **[review]**
- [ ] M3.7 Partition jobs (ensure ahead, drop old), photo/position retention, unit-offline detector

### M4 — Live, ETA, alerts, notifications (2 d)
- [ ] M4.1 Live board (portal) + `/parent/live` + `/crew/trips/{id}/live` from memory
- [ ] M4.2 Trip start/end rules (app, unit, geofence), trip_stops arrival/departure, learned offsets, ETA **[review]**
- [ ] M4.3 Alerts engine: overspeed, deviation, late start, delay, offline, power cut, tamper, not dropped **[review]**
- [ ] M4.4 Worker: FCM HTTP v1 sender (service account), dedupe, backoff, token cleanup, language templates en/gu/hi **[review]** — **decide first (human):** iOS rows in `push_tokens` hold the raw **APNs** device token (`expo-notifications` `getDevicePushTokenAsync`), which FCM can't deliver to — either add the Firebase iOS messaging SDK to the app so iOS registers an FCM token (new dependency → `/add-dependency`, ADR A10 note), or send iOS via APNs HTTP/2 directly with the `.p8` key (second sender path, changes A10). Push only to tokens whose session is live (M1.3a).
- [ ] M4.5 SOS end to end (unit + app) with siren on the portal and repeat until ack **[review]**

### M5 — Mobile app (3 d, lane 2 from M1)
- [ ] M5.1 Parent: home, child, live map (react-native-maps, Google), ETA, trip timeline, notifications, absence, requests, settings, language
- [ ] M5.2 Driver/attendant: today, inspection, start/end, stop list with students, manual mark + QR fallback, SOS, incidents with photos
- [ ] M5.3 Push handling (foreground/background, deep links), secure storage, session expiry, offline banners — iOS
  registers whatever token M4.4 decided (APNs device token today, which FCM can't deliver to)
- [ ] M5.4 EAS builds, internal testing tracks, store listings + privacy policy (accounts: docs/15 §2–§4; submission in M9.5)

### M6 — Portal (2 d, lane 2)
- [ ] M6.1 Dashboard, live board, alerts inbox with ack/resolve, incidents, absences, requests, announcements
- [ ] M6.2 Reports (CSV) and audit trail; students' data export for a parent request
- [ ] M6.3 Admin: units (provision, allocate, install, command), firmware releases, subscriptions/invoices, agents/commissions **[review]**

### M7 — Firmware on the bench prototype (6 d, lane 3 from M0) **[hw]**
- [ ] M7.1 Bench bring-up of the Waveshare board (identify V1/V2 silkscreen, check ESP32-S3R8 marking) + reader + OLED + buttons; measure and fill `board_pins.h` HW_REV 0, including the modem UART/power pins Waveshare does not publish; publish the **free header pin table** in docs/12 §4 for the carrier designer (`/hw-bringup proto`) **[hw]**
- [ ] M7.2 Drivers: modem/MQTT (TinyGSM, from Previa Fleet), GNSS, reader, camera, SD journal, buzzer/OLED **[hw]**
- [ ] M7.3 Logic: report policy, trip state, boarding ledger (exists), SOS latch, power states (docs/12 §3, SB-C1 SPEC §6) — host tests **[review]**
- [ ] M7.4 Net: backlog replay oldest-first, photos after telemetry, config/cmd handling, acks **[review]**
- [ ] M7.5 OTA A/B + signature verify + rollback; installer CLI; provisioning image **[opus] [review] [hw]**
- [ ] M7.6 Soak test: 8 h on the bench with the simulator's server, forced power cuts and no-signal periods **[hw]**

### M8 — Hardware: SB-C1 carrier for the first schools (≈ 5–7 weeks elapsed, mostly human + lead times)
Process and gates: `hardware/README.md`; requirements: `hardware/sb-c1/SPEC.md`; decisions: docs/12.
- [ ] M8.1 Designer engaged (docs/09 H9), KiCad 10.0.x installed, SPEC.md walked through and signed off; enclosure shortlist **[human]**
- [ ] M8.2 Carrier + panel schematic using the M7.1 free-pin table; ERC clean; `/hw-design-review c1 schematic` → fixes **[review]**
- [ ] M8.3 Layout (2-layer), DRC with schematic parity, 3D fit with the Waveshare model and enclosure; `/hw-design-review c1 layout` **[review]**
- [ ] M8.4 Fab release (`checklists/fab-release.md`), two quotes, order 10 boards + parts + enclosure/harness samples **[human]**
- [ ] M8.5 Bring-up on every board (`sb-c1/bringup-test.md` §1–§2); fill `board_pins.h` HW_REV 1; `pio run -e sb_c1` in CI green **[hw]**
- [ ] M8.6 Heat gate, parked drain, ESD (`bringup-test.md` §3–§4); decide revB scope (battery/charger move?) **[hw]**
- [ ] M8.7 Bus trial: one 12 V and one 24 V bus, two weeks, daily log **[hw]**
- [ ] M8.8 Batch build for the first schools: assembler, goods-in checks, test jig, provisioning station (docs/12 §9) **[human] [hw]**

### M9 — Launch and the onboarding pipeline (3 d build + per school)
Registrations run from day 1 in parallel (docs/15 §1: start R1 D-U-N-S, R7 DLT entity, R10 M2MSP now).
- [ ] M9.1 Production server: DNS (proxied + DNS-only), certbot, ufw, Nginx, compose, backups, monitoring (docs/15 §7)
- [ ] M9.2 Importer for all seven templates (`onboarding/templates/README.md`): CSV/XLSX, dry-run report by row (no personal values), idempotent on natural keys, Geocoding for address-only stops; rules = `onboarding/validate.py` (shared fixtures) **[review]**
- [ ] M9.3 `migrate seed --demo` from `onboarding/demo/` through the importer; `tenants.is_demo` suppresses SMS/push to real numbers and billing; store-review login (docs/07 S17) **[opus] [review]**
- [ ] M9.4 `/admin` onboarding board (tenants by `onboarding_stage`, `go_live_on`), installer app flow (scan uid → bus → checks → `installed`), crew-app **route survey** mode, invite poster (QR) and invite SMS **[review]**
- [ ] M9.5 Store submissions (organization accounts, docs/15 §2) with the demo school; privacy labels and Data safety from docs/07 §3 **[human]**
- [ ] M9.6 Security floor walk-through (docs/07 §1); DPA and agreement templates reviewed by the lawyer (docs/09 L1) **[opus] [review]**
- [ ] M9.7 First school through `onboarding/school-checklist.md` with extra hypercare; record real durations and update the capacity numbers in `onboarding/README.md`
- [ ] M9.8 Next schools: staggered go-lives (≥ 1 week apart), renewal reminders and agent commission on every paid renewal (docs/10 §5)

### M10 — SB-R1 integrated board (Phase 2, after ~3 months of SB-C1 field data)
- [ ] M10.1 SB-R1 SPEC.md from `hardware/sb-r1/SPEC.md` + SB-C1 field issues; modem/GNSS choice (docs/09 H3)
- [ ] M10.2 Schematic + 4-layer RF layout with an experienced RF reviewer; `/hw-design-review r1` at both gates **[review]**
- [ ] M10.3 Pre-compliance RF/EMC scan; 10 boards; bring-up; HW_REV 2 pins; bus trial **[hw]**
- [ ] M10.4 Compliance filings for R1 per the consultant (hardware/compliance.md); switch new installs to SB-R1

**Estimate:** software M0–M6 ≈ 15.5 focused days (≈ 7–8 calendar days on 3 lanes with 3–4 Claude accounts);
firmware M7 ≈ 6 days in parallel; **hardware M8 ≈ 5–7 weeks elapsed** (designer ~2 weeks, fab + assembly 1–3 weeks,
bring-up and heat/ESD 1 week, bus trial 2 weeks) — start M8.1 now, M8.2 needs M7.1's pin table. Registrations
(docs/15) take 2–4 weeks and start now. The first school installs when M8.7 and M9.1–M9.4 are done.

## Definition of Done
1. Acceptance criteria restated and met; docs updated in the same commit.
2. Tests: two schools in every data-access test; host tests for firmware logic; simulator scenario for anything on
   the tap/live path.
3. `make lint test` (+ `make test-integration`, `make fw-test`) green.
4. `[review]` tasks reviewed; `[hw]` tasks have a bench log in the PR.
5. Ticked here; commit `<task id>: <what changed>`.
