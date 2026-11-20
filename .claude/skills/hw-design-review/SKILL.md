---
name: hw-design-review
description: Review a KiCad board (SB-C1 carrier/panel or SB-R1) at the schematic, layout or fab-release gate from kicad-cli exports — against the board SPEC.md, the checklists and board_pins.h — and write a review note for the designer.
disable-model-invocation: true
argument-hint: "<c1|r1> [schematic|layout|release]"
---
# /hw-design-review $ARGUMENTS

Claude reviews **exports only** (KiCad sources are hook-blocked). The human designer fixes things in KiCad.

1. Board dir: `c1` → `hardware/sb-c1/`, `r1` → `hardware/sb-r1/`. Find the project: `ls hardware/sb-<x>/*.kicad_pro`.
   None → stop: "no KiCad project yet".
2. Run `bash hardware/scripts/export.sh hardware/sb-<x>/<project>` (needs `kicad-cli`; if it is missing here, ask the
   human to run the same command on their machine — `out/` then exists in their checkout). Exit 5 = ERC/DRC errors: list them first.
3. Read, in this order and only these: `out/erc-errors.json` (then `out/erc.json` warnings, briefly),
   `out/netlist-summary.txt`, `out/pins.txt`, `out/bom.csv`; for layout/release also `out/drc-errors.json`.
4. Check against `hardware/sb-<x>/SPEC.md` (by section: `bash .claude/scripts/docsec.sh hardware/sb-<x>/SPEC.md
   "<heading>"`), the gate's checklist in `hardware/checklists/`, and `firmware/include/board_pins.h`:
   - every SPEC block present; power chain order; enable logic; parked-drain budget the designer stated
   - `pins.txt` rule check clean; every firmware-visible net named as in `board_pins.h`
   - field inputs: resistor + TVS + opto; SOS fail-safe wiring
   - BOM: MPN on every line, voltage ratings ≥ 2 × working on input-side parts, locking connectors
5. Write `hardware/sb-<x>/reviews/<YYYY-MM-DD>-<gate>.md`:
   **Blockers** (must fix before the gate) · **Majors** · **Minors/questions** · **Checklist status** (copy the
   checklist with [x]/[ ]/n-a and one-line evidence) · **Pin table** for board_pins.h (proposal only; pins enter
   `board_pins.h` after bring-up, `/hw-bringup`).
6. Reply in ≤ 10 lines: gate verdict (pass / pass with fixes / fail), blocker count, the review file path.
   Never mark the gate passed in docs/08 — the human does that.
