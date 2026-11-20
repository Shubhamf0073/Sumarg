# SB-C1 — carrier and front-panel boards: design specification

Status: **draft for designer kickoff** · KiCad version: _fill in_ · Owner: _designer name_ · Reviewer: Previa
Context: `docs/12-hardware.md` §1–§5. Part numbers below are **candidates** — the designer confirms datasheet,
stock and price at the schematic gate and records the final MPN in each symbol.

## 1. What the boards do
The **carrier** takes the bus supply and every field wire, protects them, and feeds a Waveshare
ESP32-S3-SIM7670G-4G board that plugs onto it. The **front-panel board** sits behind the enclosure lid at the tap
zone and carries the card reader antenna area, display, LED and buzzer. Nothing in the field touches the Waveshare
board directly.

```
 Bus harness ─J1─► fuse(harness) ─► reverse prot. ─► TVS ─► buck 5V/3A ─┬─► Waveshare 5V  (40-pin header + stand-offs)
                                     │                    EN=IGN|KEEP_ALIVE│
 IGN, SOS×2, TAMPER ─J1─► R + D + TVS + opto ───────────────────────────────┼─► Waveshare GPIO (RTC-capable)
 VIN ─► divider ─► ADC1 (power-fail)                                       │
 NTC near 18650 ─► ADC1                                                     │
 Front panel ◄─J2─ SPI (PN532), I²C (OLED), LED, BUZZER, 3V3, 5V, GND ◄─────┘
 IPEX → SMA pigtails ─► external LTE + GNSS antennas (enclosure bulkheads)
```

## 2. Electrical requirements

### 2.1 Supply input (J1 pins 1–4)
| Parameter | Requirement |
|---|---|
| Continuous input | 9–36 V (12 V and 24 V buses) |
| Reverse polarity | −36 V indefinitely, no damage, no current drawn |
| Transients | 24 V suppressed load dump (ISO 16750-2 class, ~58 V clamp level) and jump-start 2 × nominal for 60 s |
| Cranking | Input dips to 6 V for 50 ms: buck may drop out; the Waveshare board rides through on its 18650 |
| Fuse | 3 A slow-blow mini-blade in the harness (not on the PCB); PCB adds nothing that needs replacing |
| Reverse protection | Ideal-diode controller + N-FET (candidate **TI LM74700-Q1**) or P-FET with zener gate clamp |
| TVS | Bidirectional or unidirectional rated for the load-dump pulse after reverse protection (candidate **SMCJ36A**-class; designer to size against the pulse) |
| Buck | ≥ 60 V input rating, 5.0 V / 3 A continuous, EN pin, ≥ 85 % efficiency at 12 V/1 A, spread-spectrum or ≥ 2 MHz preferred (keeps away from the AM band and GNSS harmonics). Candidates: **TI TPS54360B**, **TI LM76005** (or any 60 V+ 3 A buck the designer has used before) |
| Bulk at 5 V | ≥ 470 µF low-ESR polymer/electrolytic + ceramics at the Waveshare 5 V pins (modem bursts) |
| Enable logic | `BUCK_EN = IGN_PRESENT OR KEEP_ALIVE` (diode-OR or a small logic gate powered from a ≤ 10 µA LDO) |
| Parked drain | ≤ 10 mA at 12 V total, measured with IGN off and KEEP_ALIVE low |

### 2.2 Digital inputs (IGN, SOS_DRIVER, SOS_DOOR, TAMPER)
- Field side: series resistor sized for 9–36 V (LED current 1.5–8 mA across the range), reverse diode, TVS
  (SMAJ36A-class), optocoupler (PC817-class through-hole or TLP291-class SMD; CTR ≥ 50 % at 1 mA).
- Logic side: 10 kΩ pull-up to the Waveshare 3V3, 100 nF to GND, to an **RTC-capable GPIO (0–21)** so it wakes
  the ESP32-S3 from deep sleep. Active level documented on the schematic (SOS buttons: closed = active).
- SOS buttons are wired as **closed-to-battery** (each button switches BATT+ to its input) so a cut wire reads as
  "not pressed" and is detected by the 1-per-minute self-test pulse (§2.6).
- TAMPER: lid microswitch, closed when the lid is shut.

### 2.3 Analog sense
- `VIN_SENSE`: protected VIN → divider (e.g. 100 k / 8.2 k, 0.1 % not needed) → 100 nF → **ADC1 pin**
  (GPIO 1–10). Range 0–40 V into ≤ 3.1 V.
- `BATT_NTC`: 10 k NTC (B 3435) glued to the 18650 sleeve via a 2-pin JST lead, divider to 3V3, → ADC1 pin.

### 2.4 Outputs
- `KEEP_ALIVE`: GPIO → buck enable OR (above).
- `BUZZER`: GPIO → NPN/N-FET low-side → 5 V magnetic buzzer (≥ 85 dB at 10 cm) with flyback diode, on the panel.
- `LED`: one WS2812-class RGB LED on the panel (single GPIO, 5 V with 3.3 V-tolerant input, or a level shifter).

### 2.5 Front-panel board (via J2)
- **PN532** on SPI (SCK, MOSI, MISO, CS, RST, IRQ), antenna area 40 × 40 mm keep-out under the marked tap zone,
  ground pour removed beneath the antenna; matching per the NXP PN532 antenna note (or use a pre-tuned module and
  mount its antenna here — designer's choice, state it at the schematic gate).
- **OLED 1.3" SH1106** on I²C (SDA, SCL), 4.7 kΩ pull-ups on the carrier side only.
- Buzzer, RGB LED, 2 mounting holes aligned with the lid bosses.
- J2: locking 10-way, ≤ 30 cm cable: 3V3, 5V, GND ×2, SCK, MOSI, MISO, CS_RFID, SDA, SCL, plus a second 10-way or
  a 12-way if RST/IRQ/LED/BUZZER don't fit — designer chooses, keeps SPI and ground adjacent.

### 2.6 Self-test hooks
- A GPIO-driven test pulse (`SOS_TEST`) injects ~2 mA into each SOS input's opto LED through a diode so firmware
  can prove the input path once a minute without the driver pressing anything. (Optional for revA; mandatory for
  revB if the bus trial shows false alarms or dead buttons.)

## 3. Connectors

### J1 — vehicle harness (locking, 8-way, ≥ 3 A per contact, e.g. Molex Micro-Fit 3.0 2×4 or JST VH-class)
| Pin | Signal | Wire |
|---|---|---|
| 1 | BATT+ (permanent, via 3 A harness fuse) | red 1.0 mm² |
| 2 | BATT+ (second contact for current share) | red |
| 3 | GND | black 1.0 mm² |
| 4 | GND | black |
| 5 | IGN (ACC) | yellow |
| 6 | SOS_DRIVER (switched BATT+) | orange |
| 7 | SOS_DOOR (switched BATT+) | orange/white |
| 8 | TAMPER return / spare | grey |

### J2 — front panel (see §2.5) · J3 — 18650 NTC (2-pin JST-PH) · J4 — lid tamper switch (2-pin JST-PH)
### Antennas — IPEX/U.FL pigtails from the Waveshare board to SMA bulkheads on the enclosure (LTE: main; GNSS: active antenna if the module powers it — confirm at bring-up)

## 4. Mechanical
- Carrier outline set by the chosen enclosure (IP54, ~160 × 110 × 60 mm class); mounting holes M3 at the
  enclosure's boss positions; Waveshare board on M2.5/M3 stand-offs + its 40-pin header into a matching female
  header on the carrier.
- Camera on a longer FPC (24-pin) to a bracket above the tap zone, looking through a clear window.
- USB-C of the Waveshare board reachable with the lid off; DIP switch next to it.
- Keep 10 mm clearance around the Waveshare LTE/GNSS IPEX connectors for the pigtails.
- Conformal coat the carrier (not the Waveshare board connectors, not the SD slot).

## 5. Layout rules
- 2 layers, 1.6 mm FR-4, 1 oz; solid ground on the bottom, no splits under the buck or the SPI lines.
- Buck: follow the datasheet layout; hot loop minimal; input caps at the IC; inductor away from the GNSS pigtail.
- Field inputs enter at the board edge at J1 → TVS → resistor → opto, with the opto as the isolation boundary
  (≥ 2 mm creepage field ↔ logic).
- ≥ 1.0 mm traces for BATT+/5 V paths; via stitching along the board edge.
- Test points (1 mm pads, labelled): VIN_RAW, VIN_PROT, 5V, 3V3, GND ×2, IGN_OUT, SOS_DRIVER_OUT, SOS_DOOR_OUT,
  TAMPER_OUT, KEEP_ALIVE.
- Silkscreen: board name, rev, date code box, J1 pin table, polarity marks, "Previa Softwares".

## 6. Power states (firmware and hardware together)
| State | IGN | KEEP_ALIVE | Buck | ESP32-S3 / modem | Bus battery drain |
|---|---|---|---|---|---|
| Driving | on | any | on | running, reporting | ~0.5–1 A at 12 V |
| Just parked (≤ 30 min, configurable) | off | high | on | uploading backlog, reporting every 5 min | ~0.3 A |
| Parked | off | low | **off** | deep sleep, modem off; runs from 18650; wakes on IGN/SOS/TAMPER | ≤ 10 mA |
| Power cut / cut harness | — | — | off | running on 18650, POWER_CUT event, reporting every 2 min | 0 |

## 7. Deliverables at each gate
Schematic PDF, ERC JSON, BOM CSV with MPNs, netlist (KiCad XML) — then layout: DRC JSON with schematic parity,
Gerbers, drill, placement CSV, 3D STEP of the carrier with the Waveshare board model, assembly drawing.
`hardware/scripts/export.sh sb-c1/<project>` produces all of them.

## 8. Open items for the designer
- Confirm which Waveshare header pins are free (from M7.1 bench table in docs/12 §4) and assign IGN/SOS/TAMPER/
  VIN_SENSE/BATT_NTC/KEEP_ALIVE/BUZZER/LED/SPI/I²C to them.
- Confirm the Waveshare 5 V pin current rating; if < 3 A, feed 5 V on two pins or via the USB-C VBUS pad.
- Enclosure choice (designer proposes two, Previa picks).
