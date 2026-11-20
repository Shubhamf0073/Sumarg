# Layout review checklist

- [ ] DRC clean with `--schematic-parity` (no unrouted nets, no parity errors) **(auto)**
- [ ] Board outline, holes and connector positions match the enclosure drawing; 3D model checked with the Waveshare
      board / modules and the lid closed
- [ ] Buck hot loop minimal, input caps at the IC, feedback trace away from the switch node and inductor
- [ ] Solid ground plane, no slots under high-speed (SPI, camera FPC) or switching nodes; stitching vias at edges
- [ ] Field connector → TVS → resistor → opto order on the board; ≥ 2 mm creepage across the opto boundary
- [ ] Power traces sized (≥ 1.0 mm for 3 A at 1 oz, or polygons); thermal reliefs on hand-soldered pads only
- [ ] Antenna pigtail connectors with clearance; nothing tall next to them
- [ ] Reader antenna area free of copper on all layers
- [ ] Test points reachable with the board in its enclosure base; labelled
- [ ] Silkscreen: name, rev, J1 pin table, polarity marks, date code box; nothing on pads
- [ ] Fiducials and tooling holes if the assembler asks; panelisation agreed with the assembler
- [ ] Conformal-coat keep-out areas marked on the assembly drawing
