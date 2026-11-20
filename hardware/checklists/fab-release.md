# Fab release checklist

- [ ] Schematic and layout gates passed; review reports linked in the PR
- [ ] `bash hardware/scripts/export.sh <board-dir>/<project>` run on the final commit; outputs copied to
      `<board-dir>/release/<rev>/`: Gerbers + drill (zip), BOM CSV, placement CSV, schematic PDF, ERC/DRC JSON,
      assembly drawing PDF, STEP
- [ ] Gerbers opened in a viewer (KiCad GerbView or the fab's viewer) — every layer present, outline closed, drill
      sizes sane
- [ ] BOM: every line has MPN + supplier part; DNP marked; alternates approved
- [ ] Rev on the silkscreen matches the folder and the git tag `hw-<board>-<rev>`
- [ ] Two quotes (fab + assembly), lead time and shipping/customs time recorded in `release/<rev>/ORDER.md`
- [ ] First order quantity: 10 boards (5 assembled, 5 bare spares) unless the human decides otherwise
- [ ] The human approves the order
