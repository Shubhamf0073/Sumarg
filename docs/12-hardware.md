# 12 — Hardware: the SB bus unit

There is **no existing hardware design** to start from (no KiCad, no schematic; the Previa Fleet notes were worked
out from a photo of a board and are only used here as a review checklist). The unit is therefore designed from
scratch, in three steps, so the first schools never wait for a custom radio board:

| Step | What it is | `HW_REV` / PlatformIO env | Who gets it | Why |
|---|---|---|---|---|
| **SB-P0 bench** | Waveshare ESP32-S3-SIM7670G-4G dev board + reader + display + buttons on jumper wires | 0 / `sb_proto` | Firmware developers | Firmware starts now; every pin is measured before any PCB is drawn |
| **SB-C1 carrier** | The same Waveshare board plugged onto a **2-layer carrier PCB** we design: automotive power, protected inputs, harness and panel connectors | 1 / `sb_c1` | The first schools (first 50–100 buses) | No RF or modem layout in our first PCB — those are where first-time designs fail. Simple board, ~2 weeks of design, made in India |
| **SB-R1 integrated** | One 4-layer board: ESP32-S3 module + LTE Cat-1 module + GNSS + everything from SB-C1 | 2 / `sb_r1` | Volume, after ~3 months of SB-C1 field data | Lower unit cost (≈ ₹6,800 vs ≈ ₹10,700), smaller, AIS-140-ready |

The unit sits in one enclosure by the front door: everything a child taps and the driver presses is within reach;
the LTE and GNSS antennas run to the dashboard / roof.

Working files live in `hardware/` (KiCad 10 projects, specs, BOMs, checklists, export scripts — `hardware/README.md`).
This doc holds the decisions; `hardware/sb-c1/SPEC.md` holds the full SB-C1 requirements a designer works from.

## 1. Requirements every step must meet

| Area | Requirement |
|---|---|
| Supply | 12 V and 24 V buses: **9–36 V** continuous; survive reverse connection, jump-start (2 × nominal for 60 s), and 24 V suppressed load dump (ISO 16750-2 class) without damage; brown-out to 6 V on cranking without corrupting the SD |
| Parked drain | Ignition off ≥ 30 min → unit asleep, **≤ 10 mA at 12 V** from the bus battery (summer vacations are 6–7 weeks) |
| Backup | Internal Li-ion keeps reporting after a power cut / cut harness: **≥ 1 h** (SB-C1), **≥ 4 h** (SB-R1, AIS-140 asks 4 h) |
| Temperature | Operate −10 to +70 °C (a parked bus in a Gujarat summer); **no Li-ion charging above 45 °C** |
| Ingress / shock | IP54 enclosure, mounted with vibration-rated fixings; locking connectors on every cable |
| ESD | Tap zone, buttons and every external line survive ±8 kV contact (children touch it daily) |
| Radios | LTE Cat-1 on Indian bands (FDD B1/B3/B5/B8, TDD B40/B41); GNSS (NavIC on SB-R1 for AIS-140); 13.56 MHz reader |
| Wake sources | Ignition, both SOS buttons and tamper wake the ESP32-S3 from deep sleep → these inputs must land on **RTC-capable GPIOs (0–21)** |
| Service | USB-C reachable with the lid off; installer DIP; uid QR label outside |

## 2. Blocks

| Block | SB-P0 / SB-C1 | SB-R1 | Notes |
|---|---|---|---|
| MCU | ESP32-S3R8 on the Waveshare board (16 MB flash, 8 MB octal PSRAM) | **ESP32-S3-WROOM-1-N16R8** module | Secure Boot v2 + flash encryption capable. Octal PSRAM reserves **GPIO 33–37** |
| Cellular | SIM7670G on the Waveshare board | SIMCom A7670C-class or SIM7670G (pick at R1 schematic, H3) | Waveshare lists LTE-FDD B1/B3/B5/B8 and LTE-TDD B40/B41 among the SIM7670G bands — covers Jio, Airtel and Vi |
| GNSS | SIM7670G's GNSS (GPS, BeiDou, GLONASS, Galileo) | NavIC-capable GNSS (separate module or modem variant) | NavIC needed only for AIS-140 (§7) |
| Card reader | PN532 module on SPI (RC522 acceptable on the bench), antenna behind the marked tap zone | PN532 on board, PCB antenna on a panel board | 13.56 MHz ISO 14443A; NTAG 424 DNA support for Phase 2 cards |
| Camera | Waveshare's 24-pin camera port (V2 boards ship OV5640), wide-angle lens on a longer FPC | OV5640 on FPC | One VGA JPEG per accepted tap |
| Display, sound, light | 1.3" SH1106 OLED (I²C), 5 V magnetic buzzer via transistor, RGB LED on the panel | same | Readable at arm's length; buzzer audible over a running diesel |
| SOS | 2 latching buttons (driver, door) + key switch to clear, wired to opto inputs | same | Separate from the school's AIS-140 panic button |
| Ignition | Opto-isolated ACC input | same | Wake + IGN events |
| Power-fail sense | Divider from protected VIN to an ADC1 pin | same | Power-cut event within 1 s |
| Storage | microSD on the Waveshare TF slot, industrial 8 GB | microSD socket | Journal + photos |
| Power | **Carrier:** fuse (harness) → reverse protection → TVS → 60 V-class buck to **5 V / 3 A** → Waveshare 5 V input | Same front end, then 3.8 V for the modem and 3.3 V for the rest | Modem peaks ≈ 2 A; Waveshare quotes ≈ 1.8 A at 5 V (≈ 9 W) while the camera runs continuously |
| Backup | 18650 in the Waveshare holder, Waveshare charger, MAX17048 gauge | 1 × 18650 + charger with NTC (charge inhibit > 45 °C) + gauge | SB-C1 charger temperature behaviour is a test gate (§5) |
| Tamper | Lid switch to an opto/RTC input | + antenna-detect | TAMPER event |
| Installer | Waveshare USB-C + 2-way DIP on the carrier | same | `/hw-bringup` |

## 3. SB-C1 carrier at a glance (full spec: `hardware/sb-c1/SPEC.md`)

- 2-layer FR-4, 1.6 mm, 1 oz; the Waveshare board (30.44 × 110 mm, 40-pin header) plugs in with stand-offs and is
  screwed down; all field wiring lands on the carrier, never on the Waveshare board.
- **Harness connector J1** (locking, 8-way): BATT+ (permanent, 3 A inline fuse), IGN, GND ×2, SOS_DRIVER,
  SOS_DOOR, TAMPER, spare. **Panel connector J2** (locking, 10-way) to the front-panel board: reader SPI, OLED I²C,
  LED, 3V3, 5 V, GND. Antenna pigtails (IPEX → SMA bulkhead) to external LTE and GNSS antennas.
- Power path: harness fuse → ideal-diode or P-FET reverse protection → TVS (24 V load-dump class) → buck 5 V / 3 A
  (≥ 60 V input rating) with bulk capacitance at the Waveshare 5 V pins → **buck enable = IGN OR KEEP_ALIVE**
  (lets firmware finish uploads after ignition off, then drop the bus-battery drain to near zero).
- Inputs (IGN, 2 × SOS, TAMPER): resistor + reverse diode + TVS + optocoupler; outputs pulled up to 3V3 on
  RTC-capable GPIOs so they wake the ESP32-S3.
- A **front-panel board** (also 2-layer, same KiCad project) carries the PN532 antenna area, OLED, LED and buzzer;
  it sits behind the enclosure lid at the tap zone.
- Test points on every rail and input; NTC near the 18650 to an ADC1 pin for temperature logging.

## 4. Pin plan

`firmware/include/board_pins.h` is the only place pins exist. Every HW_REV stays `#error` until its pins are
verified on a powered board (`/hw-bringup`).

- **HW_REV 0 (bench):** camera, TF slot, fuel gauge and RGB LED are fixed by the Waveshare board. Waveshare
  publishes two tables (V1 and V2 boards, V2 from 2026 with OV5640) and they differ — for example V2 moves camera
  XCLK/VSYNC/HREF/PCLK to 39/42/41/46 and the gauge I²C to 15/16. Some V1 entries fall on GPIO 33–37, which the
  octal PSRAM of an S3R8 reserves. **Treat the published tables as hints**: identify the board revision by its
  silkscreen, then measure. The modem's UART and power-control pins are not published at all — find them on the
  board (continuity from the SIM7670G pads) during M7.1.
- **HW_REV 1 (SB-C1):** the carrier uses only Waveshare header pins that M7.1 proved free. The carrier schematic
  cannot be finished before M7.1's pin table exists (plan M8.2).
- **HW_REV 2 (SB-R1):** from its own schematic, exported netlist checked by `/hw-design-review`.

Rules for every revision (from the ESP32-S3 datasheet and Previa Fleet experience): strapping pins **0, 3, 45, 46**
not used as inputs; **GPIO 33–37** unavailable with octal PSRAM; analog only on **ADC1** (GPIO 1–10 — ADC2 is
unusable while Wi-Fi runs); wake inputs on **GPIO 0–21**; modem PWRKEY through a transistor, never direct.

## 5. Gates before the first school installs SB-C1

1. **Power:** 12 V and 24 V bench supplies; reverse polarity; 36 V continuous; load-dump surrogate (TVS clamp
   test at the designer's specified pulse); cranking dip to 6 V — no reset loops, no SD corruption (journal check).
2. **Parked drain** ≤ 10 mA at 12 V after 30 min ignition off; wake on IGN and on each SOS within 2 s.
3. **Heat:** unit in an oven/heat chamber at 45, 55 and 65 °C, measure battery temperature and charge current.
   If the Waveshare charger keeps charging above 45 °C, SB-C1 rev B moves the battery and an NTC-controlled charger
   onto the carrier and feeds the Waveshare board only 5 V. Until this gate passes, units ship **without** the
   18650 (power-cut reporting then relies on the server's offline detection).
4. **ESD** at the tap zone and buttons (borrowed gun or test lab).
5. **Bus trial:** one 12 V and one 24 V bus, two weeks, logging signal, GNSS fix time, enclosure temperature,
   battery temperature, resets.

Procedures: `hardware/sb-c1/bringup-test.md`.

## 6. Bill of materials (estimates, ₹ incl. GST)

**SB-C1 (10–100 units)** — line items with candidate parts in `hardware/sb-c1/bom.csv`:

| Item | ₹ |
|---|---|
| Waveshare ESP32-S3-SIM7670G-4G (Indian distributor ₹4,698 + GST) | 5,544 |
| Carrier + panel PCBs with SMT assembly | 800 |
| Carrier components (buck, protection, optos, connectors, header) | 700 |
| PN532 reader + antenna | 400 |
| OLED 1.3" + buzzer + LED | 320 |
| External LTE + GNSS antennas, 3 m cables, IPEX→SMA pigtails | 600 |
| 18650 cell (high-temperature grade) | 350 |
| 2 SOS buttons + key switch | 300 |
| IP54 enclosure, mounts | 700 |
| Harness, inline fuse, connectors | 350 |
| microSD 8 GB industrial | 300 |
| Wide-angle camera / longer FPC | 300 |
| **Unit total** | **≈ 10,700** |

**SB-R1 (≥ 200 units)** — planning figure **≈ ₹6,820** (modules, PCB 4-layer + assembly, enclosure, antennas,
battery); re-quote at the R1 schematic stage (docs/09 H1).

Installation per bus ≈ ₹500 · student card (printed PVC, MIFARE Classic 1K) ≈ ₹40.

Goods-in check for every Waveshare board: chip marking **ESP32-S3R8** (some resellers list an R2 variant with 2 MB
PSRAM, which the firmware does not support) and the V2 silkscreen.

## 7. Compliance (details: `hardware/compliance.md`)

The unit has Wi-Fi/Bluetooth (ESP32-S3), a 13.56 MHz reader and a cellular modem, and Previa supplies the SIMs. Three
Indian regimes need a consultant's answer **before paid installs at scale** (docs/09 H8):
- **WPC Equipment Type Approval** for the short-range radios (self-declaration route on the Saral Sanchar portal).
- **TEC MTCTE**: "tracking device (with embedded SIM)" and "IoT gateway" are MTCTE product categories — confirm
  whether the SB unit falls under one.
- **M2M service provider registration** with DoT (needed to buy IoT SIMs in Previa's name — docs/15 §6).

## 8. Mechanical and installation
- Enclosure at the front door at 1.0–1.2 m height (children's reach), tap zone marked, camera above the tap zone,
  out of direct sun where possible.
- Power from the bus's permanent and ignition lines through the fused harness; no splicing into the AIS-140
  tracker's wiring.
- Antennas: LTE on the dashboard glass, GNSS with sky view; cable lengths ≤ 3 m.
- Installer checklist (app, `installer` role): unit uid scanned (QR on the label), bus selected, photo of the fitted
  unit, test tap with an installer card, SOS test, GNSS fix, signal ≥ CSQ 12 → unit `installed`
  (`onboarding/03-installation.md`).

## 9. Manufacturing and provisioning (per unit)
1. Goods-in: Waveshare board check (§6), carrier visual + rail test on the jig.
2. Flash the provisioning image; read MAC → uid `SB-XXXXXX`; generate the 32-char secret on the station.
3. Burn eFuses for Secure Boot v2 + flash encryption (release mode) with the production signing key — **irreversible**.
4. Write uid, secret, broker host and CA into encrypted NVS; print the uid QR label.
5. Self-test: modem attach, GNSS, reader, camera, SD, buttons, IGN, battery and temperature.
6. Register in `/admin → Units` (uid, IMEI, ICCID, hw rev); the secret is shown once and stored only as argon2id.

## 10. AIS-140 path (Phase 3, optional)
SB-R1 keeps the AIS-140 hooks: NavIC GNSS, ≥ 4 h battery, wired emergency buttons, tamper input, reporting to a
second server, OTA, and an IMEI/ICCID record. Certification needs type approval at a MoRTH-notified agency
(ARAI/ICAT), conformity-of-production, VAHAN registration of the device model and integration with each state's
backend. Budget and timeline to be quoted by ICAT/ARAI (docs/09 H4). Until then schools keep their existing certified
tracker; our unit does not replace it.
