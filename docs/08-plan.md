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
- [ ] M0.2 Sumarg config struct (`PLATFORM_DATABASE_URL`, `SECRETS_KEY`, MQTT, FCM, SMS, Maps); `Setup` hook in `platform/app` (same change as Cargo M0.2 — copy it if Cargo did it first) **[review]**
- [ ] M0.3 Pools + request tx middleware + readiness (copy from Cargo M0.3) **[review]**
- [ ] M0.4 huma + error mapping + `schoolfleet openapi` + `make gen-api` for portal and app
- [ ] M0.5 Portal skeleton (demo CSS, router, `/admin` lazy) and Expo app skeleton (expo-router, auth stack, two tab sets, API client) — `npx expo install --fix`

### M1 — Tenancy, auth, onboarding (2.5 d)
- [ ] M1.1 Isolation suite: catalog check (every tenant table forced RLS, no DELETE grant), parent-scope walker (every `/parent/*` route with another guardian's ids → 404) **[opus] [review]**
- [ ] M1.2 Portal login + sessions (copy Cargo auth), driver PIN login with install binding **[opus] [review]**
- [ ] M1.3 Parent OTP login (SMS gateway adapter + stub), rate limits, 90-day app sessions, push-token registration **[opus] [review]**
- [ ] M1.4 Admin passkeys (copy Cargo), onboarding wizard (school, campuses, settings, owner, package) **[review]**
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
- [ ] M4.4 Worker: FCM HTTP v1 sender (service account), dedupe, backoff, token cleanup, language templates en/gu/hi **[review]**
- [ ] M4.5 SOS end to end (unit + app) with siren on the portal and repeat until ack **[review]**

### M5 — Mobile app (3 d, lane 2 from M1)
- [ ] M5.1 Parent: home, child, live map (react-native-maps, Google), ETA, trip timeline, notifications, absence, requests, settings, language
- [ ] M5.2 Driver/attendant: today, inspection, start/end, stop list with students, manual mark + QR fallback, SOS, incidents with photos
- [ ] M5.3 Push handling (foreground/background, deep links), secure storage, session expiry, offline banners
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
