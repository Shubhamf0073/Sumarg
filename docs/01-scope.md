# 01 — Scope

**Sumarg** is Previa Softwares' multi-tenant product for school transport: a bus unit (4G, GNSS, RFID
reader, camera, SOS buttons) in every bus, a school portal, and one mobile app for parents and drivers. Schools are
tenants (one school, or a trust with several campuses); Previa's superadmin sees everything. Same family and code
base as Previa Cargo and Previa Fleet. Production: `https://sumarg.previasoftwares.com`.

## 0. Name
**Sumarg** — सुमार्ग / સુમાર્ગ: "the right path" — every child on the right path, to school and home. Chosen on 20 Nov
2026 from the product lead's Kundli and numerology (AstroSage Brihat Kundli), as Previa names its products:
- **Letter:** "Su" is one of the Kumbha (Aquarius) rashi syllables (Moon sign and ascendant: Aquarius, Moon in
  Dhanishta pada 3), and **S** is among the report's favourable alphabets (A, J, S).
- **Number:** Chaldean S3 U6 M4 A1 R2 G3 = **19 → 1** (radical number 1; 19 is a fortunate compound); Pythagorean
  = 25 → 7. 1 and 7 are the numbers favourable in both the report's Vedic and numerology pages; 5 and 8 are avoided.
- Shortlisted alternatives: Sukhyatra (25 → 7), Safehop (37 → 1).
- Tagline: *Every child, on the right path.* · ગુજરાતી: *દરેક બાળક, સાચા માર્ગે.* · हिन्दी: *हर बच्चा, सही मार्ग पर।*
- Store title (≤ 30 characters): **Sumarg: School Bus Tracker** · developer: Previa Softwares ("by Previa Softwares"
  stays on the store page, web address and SMS).
- Web: `sumarg.previasoftwares.com`; units: `devices.sumarg.previasoftwares.com`; app id `com.previasoftwares.sumarg`.
- **Internal code name stays `school-fleet`** (repo folder, Go module, `schoolfleet` binary, `sf_*` database roles,
  compose project, systemd units) — renaming code identifiers adds churn and no user value.
- **Before registering the stores and DLT templates:** trademark search on IP India (classes 9 and 42) and consider
  filing. A Delhi company, Sumarg Education Resources Pvt Ltd (education, since 2006), uses the word — check for
  conflicts in our classes.
- Company (not yet incorporated): "Previa Softwares Private Limited" totals 111 → 3 (a good number in the report);
  favourable incorporation date to aim for: Mon 25 Jan 2027 (day 7, date total 19 → 1). The registrar sets the
  certificate date on approval, so file about two weeks ahead.

Sold per student on a **12-month package** (docs/10); agents who bring a school earn a share of its net profit.

## 1. Who uses it

| Role | Where | Does |
|---|---|---|
| Previa superadmin / support / billing / installer | `/admin` (passkeys) | onboard schools, stock and install units, firmware releases, subscriptions, invoices, agents, support view |
| School owner / admin | portal | everything for the school: campuses, buses, crew, routes, students, cards, parents, alerts, reports |
| Transport manager | portal | day-to-day: live board, trips, alerts, absences, requests (campus-scoped) |
| Driver / attendant | app (PIN) | today's trips, inspection, start/end, stop list, manual mark / QR when a card fails, SOS, incidents |
| Parent / guardian | app (OTP) | children, live bus + ETA, boarded/dropped push, absence, requests, announcements |

## 2. Phase 1 — live with the first schools

Platform:
- **P1** Onboarding: school, campuses, settings, owner login; package (students × price, 12 months)
- **P2** Unit inventory: provision (uid, secret, IMEI/ICCID, hw rev), allocate to a school, install on a bus, replace
- **P3** Firmware releases: upload signed image, channels dev/beta/stable, staged rollout, rollback watch
- **P4** Subscriptions + GST-inclusive invoices (`PS/{FY}/{n}`), agent commissions
- **P5** Support view (read-only, time-boxed, audited), platform audit log
- **P6** Previa dashboard: schools, units online/offline, data usage, alerts volume

School (portal + app):
- **S1** Campuses, buses (documents + expiries), crew (licence, verification dates)
- **S2** Routes and stops (map editor, order, planned times; Routes API once on save), route ↔ bus/crew
- **S3** Students (import from Excel/CSV), stop assignment, RFID card issue/revoke, guardians and pickup permissions
- **S4** Parent invites (link + QR with school code), app login with OTP
- **S5** Trips: created daily from routes; start from the driver app (after inspection) or automatically when the bus
  leaves the campus geofence in its window
- **S6** Attendance: RFID tap + photo; manual/QR fallback; board/alight; "not dropped" check at trip end
- **S7** Live board (portal) and live tracking (app): bus position, next stop, ETA, online state
- **S8** Notifications: trip started, approaching my stop, boarded, dropped, arrived at school, delay, SOS
- **S9** SOS: unit buttons and driver app → critical alert to school staff (push + portal siren) with location
- **S10** Alerts: overspeed, route deviation, late start, delay, unit offline, power cut, tamper, unknown card,
  inspection failed
- **S11** Absences (parent marks "not travelling"; the stop is greyed for the driver) and requests (stop change,
  pickup person)
- **S12** Incidents (crew/admin, with photos)
- **S13** Announcements to a route, campus or school
- **S14** Reports (CSV): attendance by day/route/student, trips (planned vs actual), alerts, overspeed, unit health
- **S15** Audit trail
- **S16** Settings: notifications, speed limit, retention, languages (English, Gujarati, Hindi in the app)

Security floor: docs/07 §1. Privacy: docs/07 §3.

## 3. Phase 2 (after launch, as patches)

Maintenance module (work orders, service schedule, parts, vendors — demo "Maintenance Manager"), transport fees and
payments, fuel/expenses, chat (parent ↔ school), scheduled reports, X.509 unit certificates, SMS fallback alerts,
NTAG 424 DNA cards (anti-cloning), more languages.

## 4. Phase 3 / options

Sustainability (fuel, CO₂ scopes — demo "Sustainability"), face match on stored photos (needs consent), AIS-140
certification of the SB unit (docs/12 §6), seat-level SOS buttons, in-bus camera recording.

## 5. Differences from the demo (intentional)

| Demo | Here | Why |
|---|---|---|
| Student "SOS button" at a seat | 2 wired SOS buttons per bus (driver + door) + driver-app SOS | Seat wiring is Phase 3 |
| RFID and QR scan on the driver phone | RFID reader on the bus unit; phone QR/manual only as fallback | Hands-free, works without the driver's phone |
| Maintenance, Finance, Sustainability menus | Phase 2/3 | Launch scope |
| One "Super Admin" for the school | School owner/admin; Previa superadmin is separate | Multi-tenant |
| Driver portal (desktop) | Driver uses the app; portal is for school staff | Drivers don't sit at desks |
| Parent sees bus all day | Live position only while a trip of their child's route is running | Data minimisation (docs/07 §3) |

## 6. Go-live acceptance (first school)

1. Units installed on all buses of the pilot routes; each shows online, GNSS fix and correct bus on the portal.
2. Every transport student has an active card; parents invited; ≥ 70 % logged in on the app before day 1.
3. Five consecutive school days: every tap produced a boarding row and a push (< 5 s p95 when online); no
   cross-route or cross-school data in any screen (two-tenant tests green in CI).
4. SOS from each unit and from the app reaches the transport manager within 10 s.
5. Unit unplugged → "power cut" alert; journal replays after reconnect with no gaps in `seq`.
6. Backups restore-tested; OTA to beta units and rollback tested.
