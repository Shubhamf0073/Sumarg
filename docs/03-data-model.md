# 03 — Data model

Authoritative DDL: **`docs/schema.sql`** — 43 tables (incl. the partitioned `positions`), forced RLS on the 34 that
hold tenant data. Baseline migration `backend/migrations/00001_init.sql` = schema.sql verbatim; isolation tests
(`go test -tags integration ./platform/db`) pass on it with two schools.

## 1. Entities

```
tenants (school org) ─┬─ tenant_settings
                      ├─ campuses ─┬─ users (owner, admin, transport_manager, driver, attendant) ── crew
                      │            ├─ vehicles ── bus_units (our hardware; tenant_id null = Previa stock)
                      │            ├─ routes ── stops
                      │            └─ students ─┬─ rfid_cards
                      │                         └─ student_guardians ── guardians (parents)
                      ├─ trips ─┬─ trip_stops · inspections · boardings (+ files: photos)
                      │         └─ positions (daily partitions) · vehicle_live (latest)
                      ├─ alerts · incidents · notifications (outbox) · announcements · requests · absences
                      ├─ sessions · otp_challenges · push_tokens · files · audit_log
                      └─ subscriptions · platform_invoices · agent_commissions        (platform-managed)
platform: platform_users/passkeys/enrolments/sessions/audit_log · agents · platform_counters · firmware_releases ·
          unit_commands · system_state
```

## 2. Isolation layers
1. RLS forced on every tenant table and the `positions` parent (partitions are reached only through the parent;
   sf_app has no grant on partitions).
2. Composite FKs: a boarding can't point at another school's trip, student, photo or unit.
3. `bus_units` is shared inventory: sf_app can only SELECT rows allocated to its tenant; ingest (sf_platform) writes.
4. Parent scope on top of RLS: `student_guardians` decides which students, routes and trips a guardian may see.
5. Ingest never trusts the topic's tenant key; tenant and vehicle come from `bus_units`.

## 3. Volumes (per school, 30 buses, 750 transport students)
| Table | Per day | Kept |
|---|---|---|
| positions | ≈ 45,000 rows (30 buses × 4 h × 360/h) | 30 days (daily partitions) |
| boardings | ≈ 3,000 | package + 1 year |
| files (photos) | ≈ 3,000 × ~35 KB ≈ 100 MB | 30 days |
| notifications | ≈ 6,000 | 90 days (worker deletes sent rows older than 90 days — platform role) |
At 30 schools: positions ≈ 1.35 M rows/day (~40 M kept), photos ≈ 3 GB/day (~90 GB kept at 30 days) — photos are the
disk driver; keep an eye on the shared disk (docs/06 §7, docs/10 §6).

## 4. Derived values
- `vehicle_live`: written by ingest ≤ every 30 s; the API serves live data from memory (fed by `NOTIFY sf_live`).
- `stops.learned_offset_s`: recomputed nightly from `trip_stops` (median of last 10).
- `trips.path`: simplified polyline written at trip end (Douglas-Peucker, 10 m), kept after positions expire.
- Tenants carry `onboarding_stage` / `go_live_on` for the `/admin` onboarding board (onboarding/README.md) and
  `is_demo` (the demo school: no SMS/push to real numbers, never invoiced, no commission).
- `agent_commissions.package_no`: 1 = first package, 2+ = renewals; one row per paid invoice (docs/10 §5).
- Billable students: count of active students with transport and an active card, sampled daily by the worker into
  `system_state` for invoices.
