# Onboarding schools — the repeatable pipeline

There is no pilot school yet, and the goal is many schools, so onboarding is a **pipeline** run the same way every
time: one checklist per school (`school-checklist.md`), one set of seven CSV templates (`templates/`), one importer, one
installation routine, one training kit. The first school is simply the first run through it, with extra
hypercare.

## Stages

| # | Stage | Who | Typical time | Output | `tenants.onboarding_stage` |
|---|---|---|---|---|---|
| 0 | Lead → demo | Agent + Previa | 1–2 meetings | Demo on the demo school (`demo`), proposal (docs/10) | (no tenant yet) |
| 1 | Agreement | School + Previa | 1–2 weeks | Signed agreement + DPA (docs/07 §3), advance invoice, school code | `setup` |
| 2 | Setup | Previa | 1 day | Tenant, campuses, owner/admin logins, package, agent linked (`01-sales-and-agreement.md`) | `setup` |
| 3 | Data collection | School transport desk | 1–2 weeks | The seven CSVs, dry-run clean (`02-data-collection.md`) | `data` |
| 4 | Route and bus survey | Previa installer + transport manager | 1–3 days | Stops pinned/verified, bus wiring survey (`03-installation.md`) | `survey` |
| 5 | Cards | Previa / card printer | 5–7 days (parallel with 4) | Printed cards, UIDs imported | `survey` |
| 6 | Installation | Installer pair | 4–6 buses/day/pair | Units `installed`, test taps, SOS tests | `install` |
| 7 | Training + invites | Previa trainer | 1 day + 1 week | Crew trained, parent invites sent, posters up (`04-training-and-go-live.md`) | `training` |
| 8 | Go-live + hypercare | Previa support | 2 weeks | Daily review of taps, alerts, offline units, parent activation | `live` |

**Contract → go-live: about 4–5 weeks** for a 20–40 bus school, mostly waiting on the school's data and cards.
Start stage 3 the day the agreement is signed.

## Running several schools at once
- **Stagger go-lives by at least a week**; never two go-live days for one support person.
- Capacity rule of thumb (replace with your own numbers after the first three schools): one installer pair ≈ 5 buses
  a day → a 30-bus school ≈ 6 working days; one trainer ≈ 2 schools a week; hypercare ≈ 1 support person per 2
  schools in their first fortnight.
- Units, SIMs and cards are the long poles: keep **one school's worth of provisioned units in stock ahead** of the
  installation calendar (docs/12 §9, docs/15 §6).
- The `/admin` onboarding board lists tenants by `onboarding_stage` with `go_live_on`; the per-school checklist is
  the detail.

## When to go live
School years in Gujarat start in June; schools change transport arrangements at the start of the year and after
long breaks. Best windows: **the first weeks of the academic year** and **right after a long vacation**. Sign in the
months before, collect data during the break, install while buses are idle.

## Renewals (every renewal pays the agent — docs/10 §5)
- 60 days before package end: usage summary for the school (taps, alerts handled, parent activation, uptime) —
  the agent presents it.
- 30 days before: renewal invoice; student count refreshed from the school's list.
- Agent commission is created when the renewal invoice is paid (`agent_commissions.package_no` ≥ 2).

## Files
| File | Use |
|---|---|
| `school-checklist.md` | Copy per school; tick and date each line |
| `01-sales-and-agreement.md` | Demo script, what the agreement and DPA must say, setup in `/admin` |
| `02-data-collection.md` | What to ask the school for; how to send, validate and import |
| `03-installation.md` | Route survey, bus survey, installation day, sign-off |
| `04-training-and-go-live.md` | Crew training, parent invites, go-live day, hypercare metrics |
| `templates/*.csv` + `templates/README.md` | The seven data templates and their rules |
| `demo/*.csv` | Fictional demo school, loaded with `migrate seed --demo` |
| `validate.py` | Offline pre-check of a school's CSVs: `python3 -I onboarding/validate.py <folder>` (prints row numbers, never personal values) |
| `quick-cards/` | One-page cards for drivers/attendants and parents (en/gu/hi drafts) |
