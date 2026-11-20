# Bus unit firmware (ESP32-S3)

Design: `docs/14-firmware.md` · hardware: `docs/12-hardware.md` · protocol: `docs/13-device-protocol.md`.
Built on Previa Fleet's firmware design (journal-first, A/B OTA with rollback, installer mode, pure-C++ logic
with host tests) — see `docs/11-reuse.md`.

```
pio test -e native     # logic tests on the host — the CI gate
pio run -e sb_proto    # HW_REV 0 bench (Waveshare ESP32-S3-SIM7670G-4G on wires); needs pins in include/board_pins.h
pio run -e sb_c1       # HW_REV 1 SB-C1 carrier (first schools)
pio run -e sb_r1       # HW_REV 2 SB-R1 integrated board (later)
```

| Path | What |
|---|---|
| `include/board_pins.h` | every GPIO, per `HW_REV`; unknown pins are `#error` on purpose |
| `src/logic/` | pure C++17, no Arduino headers, host-tested (`boarding.h` = tap → board/alight decisions) |
| `src/drivers/` | gnss, modem (TinyGSM), rc522, camera, sd journal, buzzer/display |
| `src/net/` | MQTT client, OTA, command handler |
| `src/app/` | tasks, state machine, config manager, health |
| `partitions_16mb.csv` | nvs, nvs_keys, otadata, ota_0/ota_1 (3 MB each), journal |
| `sdkconfig.defaults` / `sdkconfig.production` | dev settings / secure boot v2 + flash encryption (irreversible eFuses) |
