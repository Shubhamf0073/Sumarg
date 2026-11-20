# 05 — Domain rules

## 1. Trips
- Trips are created by the worker at 00:30 IST for every active route on school days (calendar: tenant holidays,
  Phase 1 = weekdays minus holidays entered by the admin). One trip per route per day (`UNIQUE (route, date)`).
- Start: driver app (after inspection; failed inspection → alert, trip can still start with admin override) or the
  unit (`geo` rule, docs/14 §4). End: driver app, unit, or auto (docs/14 §4). Times are stored in UTC, shown in IST.
- Late start: not started 10 min after `planned_start` → `late_start` alert. Delay: ETA at the next stop later than
  plan by ≥ `delay_alert_minutes` → `delay` alert + parent push for the remaining stops (once per trip per stop).

## 2. Roles and permissions

| Permission | owner | admin | transport_manager | driver | attendant | parent |
|---|---|---|---|---|---|---|
| School settings, users, package view | ✓ | ✓ | — | — | — | — |
| Buses, routes, stops, students, cards, guardians | ✓ | ✓ | campus | — | — | — |
| Live board, alerts, absences, requests | ✓ | ✓ | campus | — | — | — |
| Own trips: start/end, inspection, manual mark, SOS, incidents | — | — | — | assigned | assigned | — |
| Children's live bus, notifications, absence, requests | — | — | — | — | — | own children |
| Reports, audit | ✓ | ✓ | campus | — | — | — |

Campus-scoped users see only their campus (`scope`). Parent access is always through `student_guardians`;
any id outside the guardian's children → 404.

## 3. Attendance (boardings)
- RFID tap → server recomputes direction: no open board on this trip → `board`; open board and ≥ `minRideS`
  since → `alight`; else duplicate (ignored). Unit's `r` is a hint only.
- Each boarding stores trip, student, kind, time, nearest stop within its radius (else null = "off-stop", shown in
  amber), photo, unit, `source_key` = `<uid>:<seq>` (idempotent replays).
- Manual mark (driver app) needs a reason (card forgotten / reader fault / QR) and is audited; QR on the ID card
  encodes `<school code>:<admission no>` and is checked against the trip's roster.
- Unknown card → `unknown_card` alert (normal severity) with the photo.
- Card for a student not on this route → boarding accepted (child is on the bus), `route_mismatch` shown to the
  manager — never block a child.
- End of a from-school trip: any student with `board` and no `alight` → `not_dropped` alert (high) + push to the
  transport manager; parents are not told until a human checks.
- Corrections: admin voids a boarding with a reason (row kept, `voided_at`).

## 4. Alerts
Severity: `sos` critical; `not_dropped`, `power_cut`, `tamper`, `route_deviation` high; others normal.
Overspeed: speed > `speed_limit_kmph` for ≥ 20 s (unit pre-filters, server confirms) — one alert per episode.
Route deviation: > 300 m from the route polyline for ≥ 2 min during a trip. Unit offline: no message for 10 min
during a trip window. Critical/high alerts push to owner/admin/transport manager; acknowledge + resolve with notes.

## 5. ETA
Per stop: `learned_offset_s` = median of the last 10 completed trips (planned offset until 3 trips exist).
ETA(next stops) = now + (learned_offset(stop) − learned_offset(last passed stop)) × slowdown factor (actual/learned
time since trip start, clamped 0.8–2.0). No external API. "Approaching" push when ETA ≤ `approach_minutes` or the bus
is within 1.5 km of the stop, once per trip per student.

## 6. Notifications to parents
| Event | When | Text (en) |
|---|---|---|
| trip_started | trip starts (from-school: when the child boards) | "Bus 12 (R-03) has started" |
| approaching | §5 | "Bus 12 is about 5 min from Maple Street" |
| boarded / alighted | boarding row | "Aarav boarded Bus 12 at 7:42" / "Aarav got off at Maple Street at 2:51" |
| arrived_school | to-school trip ends in campus geofence | "Bus 12 reached school at 8:05" |
| delay | §1 | "Bus 12 is running ~12 min late" |
| absence_ack | parent marked absent | "Noted: Aarav won't travel on 21 Nov" |
Language per guardian (en/gu/hi). Delivered via FCM; dedupe key per event+guardian; late updates (> 10 min old)
say "(delayed update)". SMS only if `sms_fallback` and the guardian has no push token (costs, docs/10).

## 7. SOS
Unit button or driver app → `sos` alert (critical) with live location; push + portal siren to owner, admins and the
campus's transport managers; repeats every 60 s until acknowledged. Clear: key switch at the unit, or by an admin
(command `SOS_CLEAR`). Parents are not notified automatically (a human decides what to tell them).

## 8. Absences and requests
Absence for a date/direction (until 06:00 for morning, 12:00 for afternoon, configurable later): the student is
greyed in the driver's list; a tap still records a boarding. Requests (stop change, pickup person) → manager
approves → change applied from the next day; audited.

## 9. Retention (docs/07 §3)
Boarding photos `photo_retention_days` (default 30); raw positions `position_retention_days` (default 30, dropped by
partition); trip paths, boardings, alerts kept for the package term + 1 year; `left` students anonymised 1 year after
`left_on` (name → "Former student #id", photo deleted, cards revoked).

## 10. Sessions and login
- Portal: school code + mobile + password, `__Host-sf_sid` cookie `<tenant_id>.<token>`, SameSite=Strict, idle 12 h,
  absolute 7 days, CSRF double-submit + Origin *(Cargo)*.
- Apps: `Authorization: Bearer <tenant_id>.<token>`; 90-day session bound to `install_id`, renewed on use; one
  active driver session per user (a new phone logs out the old one).
- Parent OTP: 6 digits, 5 min, max 5 attempts, max 3 OTPs per mobile per hour and 10 per school per minute; same
  response for unknown mobiles (no oracle) — only invited guardians receive an SMS.
- Driver PIN: 6 digits, exponential backoff after 5 failures, reset by an admin.
- Backoff (passwords and PINs): from the 5th failure the next try waits 30 s, doubling to 15 min; during backoff even
  the right secret gets the same 401. A cancelled school → 423 after the secret verifies; a suspended school signs in
  read-only. Login revokes any session the client presents (no fixation).
- Support view: Previa support gets a 30-min read-only session (`InTenantReadOnly`), shown in the school's audit.

## 11. Packages and billing (Previa → school)
- Package = 12 months from `period_start`; `students_committed` × `price_per_student_paise` (GST-inclusive by
  default, docs/10). Billed **annually in advance** (option: 2 instalments) — hardware cost is recovered in year one.
- Billable students = active students with `transport = true` (and an active card) — measured monthly; if the 30-day
  average exceeds the committed number by > 5 %, a top-up invoice for the extra students prorated to the package end.
- Minimum billing per bus: 20 students × price (covers the unit, SIM and install on low-occupancy buses).
- Hardware stays Previa's property (`hardware_model = 'included'`); damage beyond wear is charged at cost.
- Suspension: unpaid 30 days after due → `past_due` banner; 45 days → `suspended` (portal read-only, parent app shows
  "service paused by school", units keep logging to the journal); data is never deleted for non-payment.

## 12. Previa invoices (GST)
GST-inclusive back-calculation exactly as Previa Cargo (`money.SplitInclusive`, `HalveTax`): taxable =
round_half_up(gross × 100/118); intra-state CGST = floor(tax/2), SGST = rest; inter-state IGST = tax. One rounding
per invoice. Numbers `PS/{FY}/{n:05}`. Agent commission is computed when an invoice is paid, on the first package
**and every renewal** (`commission_mode = 'recurring'`), one `agent_commissions` row per paid invoice with its
`package_no`; never negative; voided with a credit note (docs/10 §5).

## 13. Units
Lifecycle `stock → allocated (school) → installed (bus) → repair → retired`. One installed unit per bus. Replacing a
unit moves the bus's routes to the new uid; the old unit's open trip (if any) is closed by the server. Secrets can be
rotated by command + re-provision; a stolen unit is `retired` (connect refused).

## 14. Route setup
Stops placed on the map (or imported); on save the server calls Google Routes once for the ordered stops →
polyline, distance, planned offsets (overridable). Max 25 stops per route in Phase 1. Students assigned to stops;
a stop with no students is skipped in ETA pushes but kept on the map.
