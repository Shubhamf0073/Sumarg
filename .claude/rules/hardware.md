---
paths:
  - "hardware/**"
---
# Hardware rules (KiCad 10, SB-C1 carrier → SB-R1)

Decisions: `docs/12-hardware.md`. Workflow and gates: `hardware/README.md`. Board requirements: `<board>/SPEC.md`.

- **Never edit or read KiCad sources** (`.kicad_sch/.kicad_pcb/.kicad_pro/.kicad_sym/.kicad_mod`, lib tables) —
  hooks block it. Review exports: `bash hardware/scripts/export.sh hardware/<board>/<project>` →
  `out/netlist-summary.txt`, `out/pins.txt`, `out/erc.json`, `out/drc.json`, `out/bom.csv`.
- What Claude may change: `SPEC.md`, `bom.csv`, checklists, `bringup-test.md`, `compliance.md`, scripts and their
  tests, review notes in `<board>/reviews/`. Released packages (`<board>/release/<rev>/`) are frozen.
- Part numbers are **candidates** until the designer confirms datasheet, stock and price; never present a candidate
  as verified. No made-up MPNs: write "select" when unsure.
- ESP32-S3 pin rules: no inputs on 0/3/45/46; nothing on 33–37 (octal PSRAM); analog on ADC1 (1–10); wake inputs
  on 0–21. Firmware-visible nets use `board_pins.h` names.
- Any change to power, inputs or connectors updates `SPEC.md`, `bom.csv` and `docs/12` together.
- Ordering boards, parts or test-lab work is the human's decision — prepare quotes/notes, don't commit spend.
