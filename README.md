# Sumarg

> *Every child, on the right path.* — by Previa Softwares · internal code name `school-fleet` (name: docs/01 §0)

School-transport product by **Previa Softwares**: an SB bus unit (ESP32-S3, 4G, GNSS, RFID reader, camera, SOS buttons)
in every bus, a school portal, and one app for parents and drivers — multi-tenant, with Previa's superadmin.

- App/portal: `https://sumarg.previasoftwares.com` · units: `devices.sumarg.previasoftwares.com:8883`
- Pricing (proposed, docs/10): ₹1,000 per student per year, GST included, 12-month package, hardware included
- Stack: Go 1.27 · PostgreSQL 17 (forced RLS) · React portal · Expo app · PlatformIO firmware · Compose + Nginx + Cloudflare

## Start building with Claude Code
1. New GitHub repo (`previa-school-fleet`) from this folder; `chmod +x .claude/hooks/*.py .claude/scripts/*.sh` if
   your unzip lost the bits.
2. Install Go 1.27, Node 22, Docker, PlatformIO (`pipx install platformio`); `make tools`.
3. `make dev-db migrate test test-integration` — green on day one (isolation tests, protocol tests, firmware logic tests).
4. Open Claude Code, run `/next` (starts at M0.1 in docs/08). Three lanes after M1: backend · app/portal · firmware.
5. Hardware tasks (**[hw]**) need you at the bench: `/hw-bringup proto` walks through the Waveshare prototype.
6. Start the long waits now (no code): registrations in `docs/15` §1 (D-U-N-S, DLT entity, M2MSP) and the SB-C1
   designer (`hardware/README.md`, plan M8.1).

## Working within Claude limits
Same setup as Previa Cargo, plus firmware and mobile:

| Lever | What it does |
|---|---|
| One task per session | `/next` → commit → `/clear` |
| Short always-on context | `CLAUDE.md` ≈ 50 lines; rules for backend, SQL, portal, **mobile**, **firmware**, infra load only in their folders |
| Skills on demand | 16 skills; workflow and hardware ones (`/next`, `/review-task`, `/preflight`, `/release`, `/add-dependency`, `/hw-bringup`, `/hw-design-review`, `/provision-unit`, `/release-firmware`) are user-only, so they cost nothing until typed |
| Docs by section | `bash .claude/scripts/docsec.sh docs/13-device-protocol.md "3. Telemetry"` |
| Right model per task | Sonnet by default; **[opus]** tags in docs/08 for auth, isolation, broker, tap pipeline, OTA |
| Cheap subagents | `test-runner` + `Explore` on Haiku; `reviewer`, `migration-checker`, `protocol-guardian`, `hardware-bringup` on Sonnet |
| Hooks instead of turns | format on save (Go, TS, C++), block secrets/generated/lock files and the 30–90 KB demo bundles, block unapproved packages, **block flashing/eFuses**, **block reading/editing KiCad sources** (review small exports instead), lint only changed code at stop (incl. a host compile of firmware logic) |
| Simulator instead of hardware | virtual buses over MQTT (`/simulate`) test the tap → push path without a board |

## What's decided where
| File | Contents |
|---|---|
| `docs/01-scope.md` | roles, Phase 1 features, Phase 2/3, demo differences, go-live acceptance |
| `docs/02-architecture.md` | system diagram and 32 ADRs (stack, broker, attendance, maps, auth, firmware, hardware path, PCB tooling, AIS-140, children's data, commission) |
| `docs/schema.sql` · `docs/03-data-model.md` | schema with RLS (43 tables), entities, volumes, retention |
| `docs/04-api.md` · `docs/05-domain-rules.md` | API for portal, parent app, crew app, admin; trips, attendance, alerts, ETA, notifications, SOS, billing |
| `docs/06-devops.md` · `docs/07-security.md` | shared server, DNS/certs/firewall, firmware + app releases; security floor, DPDP |
| `docs/08-plan.md` · `docs/09-open-questions.md` | milestones M0–M10 (M8 hardware SB-C1, M9 launch + onboarding, M10 SB-R1) with [opus]/[review]/[hw]/[human] tags; questions with defaults |
| `docs/10-costs.md` | cost per bus/student (SB-C1 and SB-R1), price scenarios vs market, agent commission on every renewal (3-year view) |
| `docs/11-reuse.md` | what came from Previa Cargo, Previa Fleet and eSwasthay |
| `docs/12-hardware.md` · `docs/13-device-protocol.md` · `docs/14-firmware.md` | bus unit (bench → SB-C1 carrier → SB-R1), MQTT protocol v1, firmware design |
| `docs/15-accounts-and-registrations.md` | D-U-N-S, Apple, Google Play, EAS, Firebase, Maps keys, DLT + SMS, M2MSP + IoT SIMs, DNS — order, costs, waits |
| `hardware/` | SB-C1 spec, BOM, bring-up tests, review checklists, compliance notes, kicad-cli export + netlist review scripts |
| `onboarding/` | school onboarding pipeline: checklist, stages, CSV templates + validator, demo school data, quick cards (en/gu/hi) |
| `backend/` `frontend/` `mobile/` `firmware/` `infra/` `reference/` `.claude/` | code, configs, demo spec, Claude setup |

## Needed from outside the code
Registrations in `docs/15` (D-U-N-S → Apple/Google organization accounts; Firebase + APNs key; Maps keys; DLT entity,
header and templates + SMS provider; M2MSP registration → IoT SIM contract; DNS) · a KiCad designer for SB-C1 and
part/PCB quotes (H1, H9) · compliance consultant (H8) · lawyer review of the agreement, DPA and privacy notice (L1) ·
schools signed through the onboarding pipeline (`onboarding/`).
