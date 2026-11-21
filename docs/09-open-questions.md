# 09 — Open questions

Defaults are what the code does until someone answers. ⚠ = blocks a milestone.

## Previa (product and business)
| # | Question | Default | Ref | Answer |
|---|---|---|---|---|
| V1 | Product name and URL | Sumarg at `sumarg.previasoftwares.com` | 01 §0 | Name chosen 20 Nov: **Sumarg** (Kundli + numerology, docs/01 §0). Trademark search (classes 9, 42) before store registration |
| V2 | ⚠ Price per student per year, GST included? | ₹1,000 incl. GST; min 20 students per bus | 10 §3 | |
| V3 | Agent commission: first package only, or every renewal? Rate 35/40/45 %? | every renewal, 40 % | 10 §5 | "Every renewal" — 20 Nov. Rate still to confirm per agent |
| V4 | Hardware: included in the price (Previa owns units) or sold to schools? | included | 05 §11 | |
| V5 | Billing: annual in advance, or 2 instalments? | annual | 05 §11 | |
| V6 | Bus unit: add-on now, AIS-140 later | add-on now, AIS-140-ready design | 02 A29 | "Either 1 or 2" — 19 Nov |
| V7 | Attendance method | RFID tap + photo | 02 A9 | RFID + photo — 19 Nov |
| V8 | Mobile apps | One Expo app | 02 A2 | One cross-platform app — 19 Nov |
| V9 | SMS gateway for parent OTP (needs DLT entity + template registration) | MSG91; DLT on Jio TrueConnect | 15 §5 | "Need that as well" — 20 Nov: register (docs/15 R7–R9) |
| V10 | Same shared VPS as Cargo for launch? | yes, separate compose project | 06 §1 | |
| V11 | If an agent stops working with Previa, do renewal commissions for their schools continue? | stop new rows; pay rows already due | 10 §5 | |
| V12 | Firebase, Apple and Google developer accounts | organization accounts (D-U-N-S) | 15 §2–§4 | "Need to register and prepare" — 20 Nov |
| V13 | Admin on its own host (`admin.sumarg.previasoftwares.com`, behind Cloudflare Access) instead of `/admin` on the portal host? Same host means a script injected into the school portal (an unescaped school-entered value) runs with a signed-in admin's cookies; passkeys protect only the sign-in | `/admin` on the portal host (A24) until decided; passkey RP ID is the bare portal host, so a move to `admin.<host>` keeps every passkey (no re-enrolment) | 02 A24, 07 S6 | |

## Hardware and firmware
| # | Question | Default | Ref | Answer |
|---|---|---|---|---|
| H1 | ⚠ Quotes for SB-C1 parts, PCB + assembly (10 and 100 units) and enclosure; assembler for batches | estimates in 12 §6 / `hardware/sb-c1/bom.csv` | 12 §6 | |
| H2 | KiCad source of an earlier board to start from | none exists — design from scratch: bench → SB-C1 carrier → SB-R1 | 12 | "We haven't prepared any KiCad … we need to" — 20 Nov |
| H3 | GNSS: NavIC module on R1 (AIS-140-ready) or the cheaper module first? | NavIC-capable | 12 §1 | |
| H4 | AIS-140 certification budget/timeline from ICAT/ARAI | Phase 3 | 12 §6 | |
| H5 | IoT SIM provider and plan (250 MB/month, pooled) | Airtel/Jio/Vi IoT, ≤ ₹150/month; requirements in docs/15 §6 | 15 §6 | "We can buy that" — 20 Nov. Provider still to pick |
| H6 | Card type: MIFARE Classic 1K printed ID cards now, NTAG 424 DNA in Phase 2? | yes | 07 §2 | |
| H7 | Where is the firmware signing key kept (token/air-gapped laptop) and who signs? | founder + one backup | 14 §9 | |
| H8 | ⚠ Compliance: WPC ETA, MTCTE ("tracking device"), M2MSP registration, EPR — what applies and by when? | consultant before paid installs at scale; M2MSP first (blocks SIMs) | `hardware/compliance.md` | |
| H9 | ⚠ Who designs SB-C1 in KiCad: in-house engineer or contracted designer? Budget? | contracted designer, fixed price, Previa owns the files; ₹1.5–3 lakh incl. first boards | `hardware/README.md`, 10 §1 | |
| H10 | Heat gate: does the Waveshare charger stop charging above 45 °C? | ship SB-C1 without the 18650 until proven | 12 §5 | |

## Per school (collected during onboarding — `onboarding/school-checklist.md`)
No pilot school yet (20 Nov): every school goes through the same pipeline (`onboarding/README.md`); these are the
questions each one answers. The first school gets extra hypercare (plan M9.7).
| # | Question | Default | Ref | Answer |
|---|---|---|---|---|
| C1 | School name, code, campuses, board, GSTIN | — | onboarding/01 | per school |
| C2 | Buses: count, reg nos, existing AIS-140 tracker vendor, power (12/24 V) | bus survey | onboarding/03 | per school |
| C3 | Routes and stops, timings, half-days | `routes_stops.csv` or route survey | onboarding/02 | per school |
| C4 | Students with admission no, class, stop, guardians' mobiles | `students_guardians.csv` | onboarding/02 | per school |
| C5 | Do they already issue RFID/ID cards? Card type? | print ours (MIFARE Classic 1K) | 12 §6 | per school |
| C6 | Attendants on buses? | yes where present | 05 §2 | per school |
| C7 | Languages for parents | en + gu | 05 §6 | per school |
| C8 | DPA signatory and privacy notice approval | — | 07 §3 | per school |
| C9 | Which schools first, and when? Go-live windows: start of the academic year or right after a long vacation | sign before, install during the break | onboarding/README | |

## Legal
| # | Question | Default | Ref | Answer |
|---|---|---|---|---|
| L1 | ⚠ Lawyer review: agreement + DPA templates, privacy notice, app store privacy labels | before M9.5 | 07 §3, onboarding/01 | |
| L2 | Photos of children: retention 30 days OK for schools/lawyer? | 30 days | 05 §9 | |
