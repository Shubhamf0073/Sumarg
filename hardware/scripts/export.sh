#!/usr/bin/env bash
# export.sh — run KiCad 10 checks and exports for one board, into <board-dir>/out/.
#
#   bash hardware/scripts/export.sh hardware/sb-c1/sb-c1        # project path without extension
#   bash hardware/scripts/export.sh hardware/sb-c1/sb-c1 --check # CI: only ERC/DRC gates + netlist/BOM
#
# Needs kicad-cli (KiCad 10.0.x) on PATH, or KICAD_CLI=/path/to/kicad-cli.
# Exit codes: 0 ok · 5 ERC/DRC errors found (reports still written) · 2 usage/missing files.
# Claude reads out/erc.json, out/drc.json, out/bom.csv and the summary from netlist_summary.py — never the
# .kicad_sch/.kicad_pcb sources (hook-blocked).
set -euo pipefail

proj="${1:-}"
mode="${2:-full}"
if [[ -z "$proj" ]]; then
  echo "usage: export.sh <path/to/project-without-extension> [--check]" >&2
  exit 2
fi
proj="${proj%.kicad_pro}"
sch="$proj.kicad_sch"
pcb="$proj.kicad_pcb"
dir="$(dirname "$proj")"
out="$dir/out"
cli="${KICAD_CLI:-kicad-cli}"

if ! command -v "$cli" >/dev/null 2>&1; then
  echo "kicad-cli not found (install KiCad 10.0.x or set KICAD_CLI)" >&2
  exit 2
fi
if [[ ! -f "$sch" ]]; then
  echo "no schematic at $sch" >&2
  exit 2
fi
mkdir -p "$out"
"$cli" version | sed 's/^/kicad-cli /'

fail=0

# --- Schematic -----------------------------------------------------------------------------------------------
"$cli" sch erc --format json --severity-all --units mm -o "$out/erc.json" "$sch" >/dev/null
if ! "$cli" sch erc --format json --severity-error --exit-code-violations -o "$out/erc-errors.json" "$sch" >/dev/null; then
  echo "ERC: errors found → $out/erc-errors.json" >&2
  fail=1
fi
"$cli" sch export netlist --format kicadxml -o "$out/netlist.xml" "$sch" >/dev/null
# ${QUANTITY} and ${DNP} are KiCad text variables, passed literally (check `kicad-cli sch export bom --help`).
# shellcheck disable=SC2016
"$cli" sch export bom -o "$out/bom.csv" \
  --fields 'Reference,Value,Footprint,${QUANTITY},MPN,Manufacturer,LCSC,Alt MPN,${DNP}' \
  --labels 'Refs,Value,Footprint,Qty,MPN,Manufacturer,LCSC,Alt MPN,DNP' \
  --group-by 'Value,Footprint,MPN' "$sch" >/dev/null
python3 -I "$(dirname "$0")/netlist_summary.py" "$out/netlist.xml" > "$out/netlist-summary.txt"
python3 -I "$(dirname "$0")/netlist_summary.py" --pins "$out/netlist.xml" > "$out/pins.txt"

if [[ "$mode" != "--check" ]]; then
  "$cli" sch export pdf -o "$out/schematic.pdf" "$sch" >/dev/null
fi

# --- Board (after layout has started) -------------------------------------------------------------------------
if [[ -f "$pcb" ]]; then
  "$cli" pcb drc --format json --severity-all --schematic-parity --units mm -o "$out/drc.json" "$pcb" >/dev/null
  if ! "$cli" pcb drc --format json --severity-error --schematic-parity --exit-code-violations \
        -o "$out/drc-errors.json" "$pcb" >/dev/null; then
    echo "DRC: errors found → $out/drc-errors.json" >&2
    fail=1
  fi
  if [[ "$mode" != "--check" ]]; then
    mkdir -p "$out/gerbers"
    "$cli" pcb export gerbers -o "$out/gerbers/" "$pcb" >/dev/null
    "$cli" pcb export drill --format excellon --generate-map --map-format pdf -o "$out/gerbers/" "$pcb" >/dev/null
    "$cli" pcb export pos --format csv --units mm --side both -o "$out/placement.csv" "$pcb" >/dev/null
    "$cli" pcb export step -o "$out/board.step" "$pcb" >/dev/null 2>&1 || echo "STEP export skipped" >&2
    (cd "$out" && rm -f gerbers.zip && zip -qr gerbers.zip gerbers)
  fi
fi

echo "outputs in $out/"
if [[ $fail -ne 0 ]]; then
  exit 5
fi
