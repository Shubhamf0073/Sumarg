# hardware/ — SB bus unit PCBs

Decisions and steps: `docs/12-hardware.md`. This folder holds the design sources and everything needed to review,
order and test them.

```
hardware/
  sb-c1/          SB-C1 carrier + front-panel board (first schools)  — SPEC.md, bom.csv, bringup-test.md, KiCad project
  sb-r1/          SB-R1 integrated board (later)                     — SPEC.md (outline), KiCad project later
  lib/            project symbol/footprint/3D libraries (sb.kicad_sym, sb.pretty/)
  checklists/     schematic review, layout review, fab release
  scripts/        export.sh (kicad-cli), netlist_summary.py
  compliance.md   WPC / MTCTE / M2M / AIS-140 notes
```

## Tools
- **KiCad 10.0.x** (latest 10.0 bug-fix release; everyone on the same minor version — KiCad files written by a newer
  version do not open in an older one). Record the version in the board's `SPEC.md` header when you start.
- `kicad-cli` (ships with KiCad) for ERC, DRC, BOM, netlist, PDF, Gerbers, drill and placement files.
- CI runs the same exports in the official KiCad container when a `.kicad_pro` exists (`.github/workflows/ci.yml`,
  job `hardware`).

## Who does what
- **A human designer owns the KiCad files** (in-house or a contracted designer — docs/09 H9). They draw the
  schematic and layout in the KiCad GUI.
- **Claude never edits KiCad files** (a hook blocks it): S-expression edits by text corrupt projects silently.
  Claude helps by writing and maintaining the specs, BOM and checklists, and by reviewing **exports**:
  `/hw-design-review c1` runs `scripts/export.sh`, condenses the netlist with `scripts/netlist_summary.py`, and
  checks it against `SPEC.md`, the checklists and `firmware/include/board_pins.h`.
- **The human approves every gate** (schematic, layout, fab release) and every order.

## Project conventions
- One KiCad project per board family: `sb-c1/sb-c1.kicad_pro` (carrier + panel as two PCBs is fine via two projects
  `sb-c1-carrier` and `sb-c1-panel` if the designer prefers — keep both under `sb-c1/`).
- Every symbol carries fields **MPN**, **Manufacturer**, **LCSC** (or local distributor part no.) and **Alt MPN**;
  the BOM export reads them. A part without an MPN fails the schematic gate.
- Project-local libraries only (`lib/`); stock KiCad libraries may be used, but any downloaded symbol/footprint
  (SnapEDA, LCSC, Ultra Librarian) is copied into `lib/` and checked against the datasheet drawing first.
- Net names for everything the firmware touches match `board_pins.h` names (`SOS_DRIVER`, `IGN_SENSE`,
  `KEEP_ALIVE`, …) so `netlist_summary.py --pins` can produce the pin table.
- Revisions: `revA`, `revB`… on the silkscreen and in the title block; git tag `hw-sb-c1-revA` at fab release.
- Outputs go to `<board>/out/` (git-ignored) except the released fab package `<board>/release/<rev>/` (committed):
  Gerbers + drill (zip), BOM, placement, schematic PDF, assembly drawing, ERC/DRC reports.

## Gates
1. **Spec sign-off** — `SPEC.md` reviewed by the human and the designer; open items listed in docs/09.
2. **Schematic gate** — ERC clean (or every exclusion commented), `checklists/schematic-review.md` done,
   `/hw-design-review` report attached to the PR.
3. **Layout gate** — DRC clean with `--schematic-parity`, `checklists/layout-review.md` done, 3D view and
   enclosure fit checked.
4. **Fab release** — `checklists/fab-release.md`; quotes from at least two fabs/assemblers; order 10 boards first.
5. **Bring-up** — `sb-c1/bringup-test.md` on every board of the first batch; pins into `board_pins.h` via
   `/hw-bringup c1`.

## Fabrication
PCB + SMT assembly can be ordered from Indian fabs/assemblers or overseas (with customs time). Get two quotes per
order; for the first 10 boards prefer a supplier that assembles from your BOM with local substitutions approved by
you. Record the supplier, price and lead time in `release/<rev>/ORDER.md`.
