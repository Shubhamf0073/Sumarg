# Schematic review checklist

Run before the schematic gate. `/hw-design-review <board>` checks the items marked **(auto)** from the exports and
lists the rest for the human.

## Completeness
- [ ] Every block in the board's `SPEC.md` is present; every open item there is answered or listed in docs/09 **(auto: block list)**
- [ ] Every symbol has MPN, Manufacturer and a distributor part number; no generic "R"/"C" without value, package and voltage **(auto)**
- [ ] ERC clean, or every exclusion has a comment **(auto)**
- [ ] Title block: board, rev, date, designer, KiCad version

## Power
- [ ] Input chain order: connector → (harness fuse) → reverse protection → TVS → regulator
- [ ] TVS standoff above 32 V jump-start; clamp below every downstream part's absolute max
- [ ] Every regulator: input/output caps per datasheet, voltage rating ≥ 2 × working, EN logic defined (no floating EN)
- [ ] Bulk capacitance at the modem/Waveshare 5 V input; inrush acceptable for the reverse-protection FET
- [ ] Parked-drain budget adds up to ≤ 10 mA at 12 V (list every quiescent current)

## ESP32-S3 pins **(auto: pin table from net names)**
- [ ] No input on strapping pins 0, 3, 45, 46; no pin from 33–37 (octal PSRAM)
- [ ] Analog only on ADC1 (GPIO 1–10)
- [ ] Wake inputs (IGN, SOS ×2, TAMPER) on GPIO 0–21
- [ ] Every firmware-visible net is named exactly as in `firmware/include/board_pins.h`
- [ ] SB-C1: only Waveshare header pins proven free in the M7.1 bench table

## Field interfaces
- [ ] Every external line: series resistance, TVS/ESD, opto or buffer; no field line straight to a GPIO
- [ ] SOS wiring fail-safe (cut wire detectable) per SPEC §2.2/§2.6
- [ ] Connector current ratings ≥ 2 × load; locking connectors only

## Radios
- [ ] Antenna feed / pigtail routing kept away from the buck and its inductor
- [ ] Reader antenna keep-out has no copper pour beneath it

## Manufacturability
- [ ] Parts in stock at the chosen assembler, or approved alternates listed
- [ ] No part with lead time > 4 weeks without a second source
