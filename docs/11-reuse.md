# 11 — Reuse from Previa's GitHub and the Previa Cargo folder

Sources: **`previa-cargo`** (this session's Go folder, itself built on `previalogistic`), **`Shubhamf0073/previafleet`**
(commit `dd33d0e`, 12 Nov 2026 — Previa Fleet: Go services, MQTT protocol v1, firmware design, simulator), the School
Fleet demo (`schoolfleet-mobile.previasoftwares.com`, saved in `reference/demo/source/`), and `eswasthay`
(UI primitives). Nothing was pushed to any repo.

## 1. Code copied

| From | Into | What it gives | Changes |
|---|---|---|---|
| Cargo `backend/platform/{app,clock,config,httpx,ids,money,obs}` | `backend/platform/` | lifecycle, config + Secret, HTTP middleware, money (paise, GST-inclusive split, amount in words), logging redaction | module path; `ingest` mode; service name `schoolfleet` |
| Cargo `platform/db`, `platformdb` + isolation tests | same | `InTenant`, `InTenantReadOnly`, role-checking pools; 6 tests | seeds and queries changed to schools/students — **all 6 pass** on this schema |
| Cargo `cmd/migrate`, `migrations/embed.go`, `.golangci.yml`, Dockerfile, sqlc config | same | goose migrator, lint architecture rules, image | Typst removed; `internal/ingest` added to depguard |
| Cargo `infra/` (compose, roles init, Nginx, scripts, systemd, dev compose) | `infra/` | the whole server setup | renamed `sf_*`; port 8720; ingest service with :8883; devices vhost for OTA; env rewritten |
| Cargo `.claude/` (hooks, docsec, settings, workflow skills, agents) | `.claude/` | usage-efficient Claude setup | firmware + mobile rules/skills/agents added (§3) |
| **Previa Fleet `pkg/protocol`** (types, topic parsing, JSON-schema validation, tests) + `schemas/` + examples | `backend/platform/protocol` | device protocol v1, already tested | fixtures moved to `testdata/`; Sumarg event kinds added in M3.3 |
| **Previa Fleet `pkg/devicesecret`, `pkg/argon2id`** | `backend/platform/` | unit secrets: generate, argon2id hash, constant-time verify | module path |
| eSwasthay `src/ui.tsx` (trimmed) | `reference/reuse/eswasthay-ui.tsx` | Modal, Field, Form, DataTable with cards/table switch | reference only; port in M0.5 |
| School Fleet demo | `reference/demo/source/` | UI spec for portal (admin, maintenance, driver portal) and app (driver, parent) | read-only |

## 2. Designs adopted (port in their milestones)

| From Previa Fleet | Here |
|---|---|
| ADR-002 MQTT 3.1.1/TLS, topics `v1/t/{tk}/d/{id}/…`, persistent sessions, retained config/status | docs/13 (same protocol) |
| docs/07 firmware design: FreeRTOS tasks, SD journal with seq, backlog replay, A/B OTA with signature + rollback, installer mode, pure-C++ logic with host tests, coding rules | docs/14 |
| docs/02 hardware notes (power, BMS, modem, SD, opto blocks — inferred from a board photo; no KiCad exists) | review checklist items in `hardware/checklists/` only; SB-C1/SB-R1 are designed from scratch (docs/12) |
| ADR-003 edge vs cloud split | taps decided on the unit, re-checked in the cloud |
| `services/ingest` pipeline (validation, store, dead-letter) | M3.2 (adapted to the embedded broker) |
| `tools/simulator` (virtual vehicles over MQTT, scenarios, e2e tests) | M3.6 → `tools/simulator` |
| Skills `hw-bringup`, `protocol-change`, `provision-device`, `release-firmware`, `simulate`; agents `hardware-bringup`, `protocol-guardian` | `.claude/skills` + `.claude/agents` (adapted, §3) |

## 3. Not adopted, and why
| Previa Fleet choice | Why not here | Instead |
|---|---|---|
| EMQX broker + HTTP auth hook | ≈ 300 MB+ RAM on a shared box | embedded mochi broker in ingest (same topics, A8) |
| TimescaleDB + PostGIS | extra extensions/ops; we need point-in-radius and polylines only | daily partitions + Go geometry |
| Redis for live state | another container | in-memory map + `NOTIFY` + `vehicle_live` |
| JWT + TOTP auth, Supabase TEST environment | Cargo's session + passkey model is stronger and already written | Cargo auth |
| Mining logic (dump/weight/fuel) | different domain | — (protocol keys simply unused) |

## 4. Worth pulling later
- Previa Fleet `docs/17` prod-migration runbook and `docs/18` launch checklist → adapt for M9.
- Cargo improvements made after this folder (M0.2/M0.3 Setup hook, request tx middleware, auth) → copy across; keep
  `backend/platform/` identical in both repos until it moves to a shared module
  (`github.com/Shubhamf0073/previa-go-platform`) — now three products use it.
