# School onboarding checklist — copy per school

School: ______________________  Code: ________  Agent: ____________  Planned go-live: ___________

| ✓ | Step | Owner | Date |
|---|---|---|---|
| | **Agreement** | | |
| | Demo given on the demo school; questions noted | Agent | |
| | Proposal: students on transport × ₹1,000 incl. GST (or agreed price), minimum 20 per bus (docs/10 §3) | Agent | |
| | Agreement + DPA signed (school = data fiduciary, Previa = processor) | School / Previa signatory | |
| | Privacy notice for parents approved by the school (docs/07 §3) | School | |
| | Advance invoice raised; payment received | Finance | |
| | **Setup** (`/admin`) | | |
| | Tenant created: legal name, display/short name, board, GSTIN, state code, school code | Previa | |
| | Package: start date, students committed, price, agent linked | Previa | |
| | Owner + admin portal logins sent; transport manager added | Previa | |
| | Data request sent with `templates/` and the secure upload channel (docs/07 S18) | Previa | |
| | **Data** | | |
| | `campuses.csv`, `staff.csv`, `vehicles.csv`, `crew.csv` received; dry-run clean; imported | Previa | |
| | `routes_stops.csv` received; stops pinned on the map; route timings confirmed | Transport manager | |
| | `students_guardians.csv` received; dry-run clean; imported; billable count agreed | Previa + school | |
| | **Survey and cards** | | |
| | Bus survey: voltage (12/24 V), mounting spot, antenna path, existing AIS-140 tracker vendor, per bus | Installer | |
| | Route ride-along (or crew app survey) for stops without verified pins | Installer + driver | |
| | Cards ordered (student name/photo per school's ID-card format); UIDs file received → `cards.csv` imported | Previa | |
| | Units provisioned and allocated to the school (one per bus + 1 spare per 15 buses); SIMs active | Previa | |
| | **Installation** | | |
| | Installation calendar agreed (buses idle: after the morning/evening runs, weekends) | Transport manager | |
| | Every bus: unit installed, uid scanned, test tap, SOS test, GNSS fix, CSQ ≥ 12, photo → `installed` | Installer | |
| | **Training and invites** | | |
| | Crew training done; drivers/attendants logged in with PIN; quick cards handed out | Trainer | |
| | Transport desk training: live board, alerts, absences, requests, reports | Trainer | |
| | Parent invites sent (SMS + class WhatsApp message with the invite link + poster QR) | School + Previa | |
| | **Go-live** | | |
| | Day 1 morning and afternoon runs watched live; issues logged | Support | |
| | Hypercare day 1–10: daily report to the school (activation %, tap coverage, offline units, alerts) | Support | |
| | Hypercare closed; `onboarding_stage = live`; renewal date in the calendar | Previa | |
