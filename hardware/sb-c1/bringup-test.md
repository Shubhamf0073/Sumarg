# SB-C1 bring-up and acceptance tests

Run on **every board of the first batch**, then on a sample of each later batch. Record results in
`release/<rev>/BRINGUP-<serial>.md` (copy the tables). Claude can prepare commands and read pasted logs; the human
handles power, meters and flashing (`/hw-bringup c1`).

Equipment: bench supply 0–40 V / 5 A with current limit, DMM, USB-C cable, a 12 V and a 24 V lead-acid battery (or
supply), heat gun/oven with a thermometer, a test card, the provisioning laptop.

## 1. Bare carrier (Waveshare board NOT fitted)
| # | Test | Pass |
|---|---|---|
| 1.1 | Visual: polarity of TVS/diodes/electrolytics, connector orientation, no bridges | |
| 1.2 | Resistance VIN_RAW–GND and 5V–GND (no short) | > 1 kΩ |
| 1.3 | 12 V, 100 mA limit, IGN high: 5V rail | 4.9–5.1 V |
| 1.4 | 24 V and 36 V, IGN high: 5V rail; input current no-load | 4.9–5.1 V; < 15 mA |
| 1.5 | Reverse 24 V for 60 s | no current, no heat, then 1.3 passes |
| 1.6 | IGN low, KEEP_ALIVE low: 5V rail and input current | 5V off; < 1 mA |
| 1.7 | Load 5 V at 3 A (electronic load / resistors) for 10 min at 12 V and 24 V | 5V ≥ 4.85 V; buck < 85 °C |
| 1.8 | Input steps 12→6→12 V (cranking) with 1 A load | no oscillation; recovers |
| 1.9 | Each input (IGN, SOS ×2, TAMPER) at 9 V and 36 V: logic side reads active; at 0 V / open reads inactive | |
| 1.10 | VIN_SENSE at 12.0 V and 24.0 V | matches divider ±3 % |

## 2. With the Waveshare board (no 18650 yet)
| # | Test | Pass |
|---|---|---|
| 2.1 | Selftest image (human flashes): boots on 12 V and 24 V | CLI prompt |
| 2.2 | `modem`: AT OK, SIM ready, CSQ, attach | CSQ ≥ 12 indoors with external antenna |
| 2.3 | `gps`: cold fix outdoors with external antenna | fix < 120 s |
| 2.4 | `rfid`: test card read at 0–3 cm through the closed lid | 20/20 reads |
| 2.5 | `cam snap`: VGA JPEG, tap zone in frame | size 20–60 KB |
| 2.6 | `sd`: write/read 10 MB, journal append | OK |
| 2.7 | `sos`, `ign`, `tamper`: each input toggles in the CLI; SOS wakes from deep sleep | < 2 s wake |
| 2.8 | `power`: VIN_SENSE trace while pulling the supply | POWER_CUT within 1 s |
| 2.9 | Peak current during LTE attach + camera snap at 12 V | record; buck stays in regulation |

## 3. With the 18650 (only after the heat gate)
| # | Test | Pass |
|---|---|---|
| 3.1 | Charge from 3.3 V cell at 25 °C | charges; MAX17048 SOC rises |
| 3.2 | **Heat gate:** oven at 45, 55, 65 °C, 30 min each, supply on | record cell temperature (NTC) and charge current; **charging must stop ≥ 45 °C** — if not, revB moves the charger (docs/12 §5) |
| 3.3 | Power cut with full cell: reporting every 2 min | ≥ 1 h |
| 3.4 | Parked: IGN off, 30 min, KEEP_ALIVE drops | bus-side current ≤ 10 mA at 12 V |

## 4. ESD and bus trial
- ESD ±8 kV contact on the tap zone, buttons and J1 pins (lab or borrowed gun): unit keeps running or resets cleanly
  with no data loss.
- Bus trial (one 12 V, one 24 V bus, 2 weeks): resets, signal, GNSS fix times, enclosure and cell temperature
  logged daily; failures go to `release/<rev>/ISSUES.md`.
