# Sumarg — Claude Code guide

Previa Softwares' multi-tenant school-transport product: an **SB bus unit** (ESP32-S3, 4G, GNSS, RFID reader,
camera, SOS) in every bus, a **school portal**, and **one Expo app** for parents and drivers, plus Previa's superadmin.
`https://sumarg.previasoftwares.com` · priced per student on a 12-month package (docs/10) · runs on the
**shared** VPS beside Previa Cargo. **Phase 1** = live with the first schools.

Stack: **Go 1.27** (chi, huma, pgx + sqlc, goose, embedded MQTT broker) · PostgreSQL 17 with **forced RLS** · React
portal · Expo/React Native app · **PlatformIO firmware** (Arduino-ESP32 on ESP-IDF) · Docker Compose + Nginx +
Cloudflare. Decisions: docs/02 §2. Protocol: docs/13. Hardware: docs/12 + `hardware/`. Firmware: docs/14.
Accounts/registrations: docs/15. School onboarding: `onboarding/`.

## How to work here (keeps usage low)
- One plan task per session: `/next` → commit → the human runs `/clear`. The SessionStart line names the task.
- Read docs **by section**: `bash .claude/scripts/docsec.sh <file>` then `… <file> "<heading>"`. Folder rules load
  automatically (`.claude/rules/`: backend, sql, frontend, mobile, firmware, hardware, infra).
- Skills instead of re-deriving: `/api-endpoint`, `/db-migration`, `/device-protocol`, `/simulate`, `/mobile-screen`,
  `/push-notifications`, `/ui-port`; hardware: `/hw-bringup`, `/hw-design-review`, `/provision-unit`, `/release-firmware`.
- Long output → `test-runner` subagent; wide searches → `Explore`; protocol changes → `protocol-guardian`.
- **[opus]** tasks need Opus; **[review]** tasks need `/review-task`; **[hw]** tasks need the human and the board.
- Hooks format on save, block secrets/generated files/unapproved packages/board flashing, lint changed code at stop.

## Commands
```
make dev-db migrate seed run sim web app      make test test-integration fw-test lint sqlc gen-api schema-diff
```

## Invariants — breaking any of these is a bug
1. **Tenant only from the session** (`<tenant_id>.<token>` cookie or bearer, looked up under RLS). For units:
   tenant and bus only from `bus_units`, never from the MQTT topic.
2. Tenant requests run in one `db.InTenant`/`InTenantReadOnly` tx on the sf_app pool. Only `/api/admin/*`, the
   worker and ingest use `platformdb` (BYPASSRLS) and they name the tenant in every query. depguard enforces imports.
3. **Parent scope:** every parent request checks the student/route/trip is one of the guardian's children; live
   position only while that child's trip runs. Out of scope → 404.
4. New tenant table: `tenant_id`, `UNIQUE (tenant_id, id)`, composite FKs, forced RLS + policy, no DELETE for
   sf_app. Order: `docs/schema.sql` → goose migration → `make sqlc`.
5. A tap/SOS is never lost: unit journals before sending; ingest is idempotent on `source_key`; boarding +
   notifications + alerts in one transaction. The server decides board/alight; the unit's decision is a hint.
6. Children's data minimal (docs/07 §3): first names only on units, photos and positions expire, no analytics SDKs.
7. Protocol changes go through `/device-protocol`: schema + examples + docs/13 + firmware codec + ingest + simulator
   together; v1 changes are additive only.
8. Firmware: pins only in `board_pins.h` (unknown = `#error`); `src/logic/` pure C++ with host tests; never flash or
   burn eFuses from Claude. Hardware: never read or edit KiCad sources — review `hardware/scripts/export.sh` outputs.
9. Money in paise; Previa invoices via `money.SplitInclusive` (GST-inclusive). No outbound HTTP in requests except
   Routes API on route save.
10. Secrets never in code/logs/tests/app bundle; don't read `.env`/`db.env`; no new dependency without
   `/add-dependency`.

## Working style
Restate acceptance criteria (2–3 lines) → smallest change → tests (two schools; host tests; a simulator scenario
for the tap/live path) → `make lint test` → tick docs/08 → commit `<task id>: <what changed>`.
Ask before: new dependency, ADR change, production `infra/`, deleting data, `git push`, anything on real hardware.
