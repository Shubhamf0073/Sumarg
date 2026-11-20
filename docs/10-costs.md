# 10 — Costs, pricing and agent commission

All figures in ₹. Hardware and SIM prices are planning estimates (to be replaced by quotes, docs/09); server and
maps figures follow Previa Cargo's costing. Market prices are from public listings checked 19 Nov 2026.

## 1. What one bus costs Previa

| Item | Basis | ₹ per bus per month |
|---|---|---|
| Bus unit + installation | ₹6,820 unit (docs/12 §2) + ₹500 install = ₹7,320, written off over 36 months | 203 |
| 4G data SIM | IoT/M2M data plan sized for ~250 MB/month (docs/13 §8) | 150 |
| Server share | Shared VPS; this product's caps are ~1.65 GB of 8 GB; spread over buses | ~10 |
| Maps | App maps (Maps SDK) are free and unlimited; portal maps within the free cap; Routes only on route save | 0 |
| Push notifications | FCM | 0 |
| **Total** | | **≈ 363** |

Per student per year, with the bus's cost shared by the students on it:

| Students per bus | Running cost / student / year | First year incl. card (₹40) |
|---|---|---|
| 20 | 218 | 258 |
| 30 | 145 | 185 |
| 40 | 109 | 149 |

If a school buys the units (`hardware_model = 'purchased'`), Previa's running cost drops to ≈ ₹64 per student per
year at 30 per bus (SIM + server only).

**First schools run on SB-C1** (docs/12): the Waveshare board on our carrier costs ≈ ₹10,700 per unit instead of
≈ ₹6,820 for SB-R1. While a bus runs SB-C1: (₹10,700 + ₹500) ÷ 36 = ₹311 + SIM ₹150 + server ₹10 ≈ **₹471 per bus per
month** → ≈ ₹188 per student per year at 30 per bus (₹228 in year one with the card) → net profit ≈ ₹619 at ₹1,000
(73 %). SB-C1 units stay in service until they fail or a school's renewal brings an SB-R1 swap; returned SB-C1 units
become spares.

One-time hardware development (budget, replace with quotes — docs/09 H9): SB-C1 design by a contracted KiCad
designer, first 10 boards, enclosure samples, test equipment, compliance consultant. Plan for ₹1.5–3 lakh before the
first paid install; SB-R1 (4-layer RF board, pre-compliance scan) later at a multiple of that.

Other costs, product-wide (docs/15): Apple Developer Program $99/year; Google Play one-time $25; DLT entity on Jio
TrueConnect ₹5,900/year; M2MSP registration ₹5,000 once; WPC/MTCTE per the consultant (H8); EAS builds (free tier, or
the paid starter plan during heavy release weeks); SMS OTP ≈ ₹0.25 each × ~2 per parent per year; a firmware signing
token; installer travel.

## 2. Market reference (public listings)
- App + RFID bundles have been sold at **₹1,200–1,500 per child per year** (SkoolSmart, RFID ID cards + readers).
- Per-student SaaS: SafeBus from **₹30/student/month** (≈ ₹360/year); ERP bundles from ₹10/student/month (Schoolites).
- Per-bus tracking-only: ₹199–299 per bus per month (app-only) up to ₹800–1,500 per bus per month (GPS-first).

Sumarg includes the bus unit, RFID cards, photo-backed attendance and the parent app, so it competes with
the RFID bundles, not with app-only trackers.

## 3. Price scenarios (per student, 12-month package, GST-inclusive, 30 students per bus, first year)

| Price / student / year | Net of GST | Previa cost | Net profit | Margin |
|---|---|---|---|---|
| ₹600 | ₹508.47 | ₹185 | ₹323 | 64 % |
| ₹900 | ₹762.71 | ₹185 | ₹577 | 76 % |
| **₹1,000 (default in schema)** | **₹847.46** | **₹185** | **₹662** | **78 %** |
| ₹1,200 | ₹1,016.95 | ₹185 | ₹832 | 82 % |

Recommendation: **₹1,000 per student per year, GST included** (≈ ₹83/month — below the RFID bundles' ₹1,200–1,500,
well above app-only trackers), billed annually in advance, with the **minimum of 20 billable students per bus**
(docs/05 §11) so a half-empty bus still covers its unit and SIM. Year 2 onward the cost falls to ≈ ₹145/student
(cards and installation already paid).

Example school (750 transport students, 30 buses, ₹1,000): gross ₹7,50,000 · net of GST ₹6,35,593 · Previa cost year 1
₹1,60,800 · **net profit ≈ ₹4,74,793**. Hardware outlay before go-live ≈ ₹2,49,600 (30 units + installs + 750 cards),
recovered from the first annual invoice.

## 4. GST on Previa's invoice
Same rule as Previa Cargo (docs/05 §12): ₹1,000 inclusive → taxable ₹847.46 + GST ₹152.54 per student, computed once
on the invoice total.

## 5. Agent commission — paid on every renewal (decided 20 Nov, docs/09 V3)
Net profit per school per package = invoice taxable value − direct costs (unit amortisation, SIM, server share, cards,
installation, SMS). Commission = net profit × agent rate (`agents.commission_bp`, default 40 %), computed when the
invoice is **paid**, stored in `agent_commissions` with `package_no` (1 = first package, 2, 3 … = renewals), and paid
out after the school's payment clears. `agents.commission_mode` defaults to `recurring`; `first_year` stays available
for a one-off deal.

Because the hardware is written off over 36 months, profit **rises** each year, and so does the agent's share.

Per student, ₹1,000 incl. GST, 30 students per bus:

| Package | Previa cost | Net profit | Agent 35 % | Agent 40 % | Agent 45 % | Previa keeps (at 40 %) |
|---|---|---|---|---|---|---|
| Year 1 (cards + install) | ₹185 | ₹662 | ₹232 | ₹265 | ₹298 | ₹397 |
| Year 2 | ₹145 | ₹702 | ₹246 | ₹281 | ₹316 | ₹421 |
| Year 3 | ₹145 | ₹702 | ₹246 | ₹281 | ₹316 | ₹421 |
| Year 4 on (units paid off) | ₹64 | ₹783 | ₹274 | ₹313 | ₹353 | ₹470 |
| **3 years** | | **₹2,067** | | **₹827** | | **₹1,240** |

Example school (750 students, 30 buses = 25 per bus, ₹1,000):

| Package | Net profit | Agent 40 % | Previa keeps |
|---|---|---|---|
| Year 1 | ₹4,74,793 | ₹1,89,917 | ₹2,84,876 |
| Year 2 | ₹5,04,793 | ₹2,01,917 | ₹3,02,876 |
| Year 3 | ₹5,04,793 | ₹2,01,917 | ₹3,02,876 |
| Year 4 on | ₹5,77,993 | ₹2,31,197 | ₹3,46,796 |
| **3 years** | **₹14,84,379** | **₹5,93,752** | **₹8,90,627** |

What "every renewal" costs Previa: over 3 years Previa keeps ₹8.91 lakh from this school instead of ₹12.94 lakh with
first-year-only commission. In return the agent stays involved in renewals, collections and the school relationship.

Rules the code follows (docs/05 §11):
- Commission is computed only on **paid** invoices; a credit note or refund voids the matching commission (`status =
  'void'`).
- Direct cost for each package uses that package's real costs: year 1 includes units, installation and all cards;
  later packages include amortisation still running, SIM, server share, SMS, cards for new students and unit
  replacements.
- A school with negative net profit for a package (for example after many replacements) gives zero commission, never a
  negative one.
- An agent marked inactive stops receiving new commission rows; rows already `due` are still paid (docs/09 V11).

## 6. Guardrails
- Watch data: a SIM plan below 200 MB/month needs `photo:false` or QVGA photos for that school.
- Server: at ~100 schools or 3,000 units, move Sumarg to its own 4 vCPU / 16 GB VPS (≈ ₹3,300/month) — still
  < ₹1/student/month.
- Hardware failures: budget 5 % unit replacements per year (≈ ₹10/student/year at 30 per bus).
