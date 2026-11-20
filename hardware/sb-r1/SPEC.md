# SB-R1 — integrated board: outline specification

Status: **outline** — start after SB-C1 has ~3 months of field data (plan M10). Everything SB-C1 proved (power
front end, input circuits, panel board, power states, connectors) is copied over unchanged unless field data says
otherwise. Requirements: `docs/12-hardware.md` §1.

## Changes from SB-C1
| Block | SB-R1 |
|---|---|
| MCU | ESP32-S3-WROOM-1-N16R8 module (PCB antenna variant only if Wi-Fi/BLE is needed for installer tools; else the U.FL variant or keep-out per Espressif guidelines) |
| Cellular | LTE Cat-1 module with Indian bands (B1/B3/B5/B8/B40/B41): SIMCom A7670C-class or SIM7670G — pick by price, GNSS needs and AT compatibility with the SB-C1 firmware (docs/09 H3) |
| GNSS | NavIC-capable receiver (AIS-140 readiness) or the modem's GNSS if NavIC is dropped |
| Modem power | 3.8 V / 2.5 A buck dedicated to the modem, ≥ 1000 µF bulk near the module, PWRKEY via transistor |
| Logic power | 3.3 V buck/LDO for the ESP32-S3, reader, OLED |
| Battery | 1 × 18650 (or 2 in parallel for ≥ 4 h) + charger with NTC charge inhibit > 45 °C + MAX17048-class gauge |
| SIM | Nano-SIM holder with ESD; optional MFF2 eSIM footprint (M2M SIM in solder form, vibration-proof) |
| RF | 4-layer stack-up with 50 Ω controlled-impedance feeds, U.FL connectors, modem RF layout per SIMCom hardware design guide; antenna keep-outs |
| Camera | 24-pin FPC on the board edge facing the window |
| Debug | USB-C to ESP32-S3 native USB; modem USB test pads for firmware updates |

## Process
Same gates as SB-C1 (`hardware/README.md`), plus an RF review by someone who has shipped a cellular board before,
and a pre-compliance RF/EMC scan before the first batch (compliance: `hardware/compliance.md`).
