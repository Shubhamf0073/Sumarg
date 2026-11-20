# 02 — Data collection and import

## What to ask the school for
Send the seven templates from `templates/` (Excel or Google Sheets copies are fine; we import CSV or XLSX):

| Template | From whom at the school | Notes |
|---|---|---|
| `campuses.csv` | Office | Usually one row; pin from Google Maps |
| `staff.csv` | Principal / office | Portal users: owner, admins, transport manager |
| `vehicles.csv` | Transport desk | Every bus/van, including contractor buses |
| `crew.csv` | Transport desk | Drivers and attendants; driver licence number and expiry required |
| `routes_stops.csv` | Transport manager | One row per stop; Google Maps link or address is enough — we pin |
| `students_guardians.csv` | Admissions / class teachers | Only students using school transport; up to 2 guardians each |
| `cards.csv` | Card printer / Previa | Card UID per admission number; produced when cards are encoded |

## How data travels (docs/07 S18)
- The school uploads to a link it controls (its Google Drive/OneDrive share to `ops@`), or emails `ops@` — never
  WhatsApp groups.
- Previa downloads, imports, and deletes local copies within 7 days. The school keeps the master.

## Import
0. Optional quick check before upload: `python3 -I onboarding/validate.py <folder>` — same rules as the importer.
1. Portal (Previa or school admin): **Setup → Import** → choose template → upload → **dry run**.
2. The dry-run report lists, by row number: missing required columns, bad formats (mobile, reg no, dates),
   unknown references (route code not in routes, stop name not on that route), duplicates. Nothing is written.
3. Fix in the sheet, re-upload, repeat until clean → **Import**. Import is idempotent on natural keys
   (`campus_code`, `reg_no`, crew `mobile`, `route_code`+`direction`+`shift`, `admission_no`, guardian `mobile`,
   card `uid`) — re-importing updates rows instead of duplicating them.
4. Order: campuses → staff → vehicles → crew → routes_stops → students_guardians → cards.
5. After import: Students → "No stop assigned" and Routes → "Stops without a verified pin" must both be empty
   before installation.

## Common problems
- Mobiles with +91/0/spaces → the importer normalises to 10 digits; landlines are rejected.
- Reg numbers with spaces/hyphens (`GJ 01 AB 1234`) → normalised to `GJ01AB1234`.
- One guardian mobile shared by siblings → fine: one guardian, several students.
- Two schools in one trust → one tenant, several campuses (docs/05).
