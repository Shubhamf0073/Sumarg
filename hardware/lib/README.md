# Project libraries

- `sb.kicad_sym` — project symbols (Waveshare header with `Role=MCU`, connectors with our pin names).
- `sb.pretty/` — project footprints; each checked against the datasheet drawing, with the datasheet page noted in the
  footprint description.
- `3d/` — STEP models (Waveshare board outline, enclosure) for fit checks.

Add both libraries to the project's `sym-lib-table` / `fp-lib-table` with `${KIPRJMOD}`-relative paths so the
project opens on any machine.
