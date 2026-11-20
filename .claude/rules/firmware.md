---
paths:
  - "firmware/**"
---
# Firmware rules (ESP32-S3, PlatformIO, Arduino-ESP32 on ESP-IDF)

Read only what the task needs: `docsec.sh docs/14-firmware.md "<section>"`, `docs/13` for payloads, `docs/12` for pins.

- **Pins only in `include/board_pins.h`**, keyed by `HW_REV`. A pin not verified on a powered board stays `#error`.
  Never invent pin numbers (wrong PWRKEY/strapping pins damage hardware).
- `src/logic/` is pure C++17: no Arduino/IDF headers, no heap after construction, host-tested
  (`pio test -e native`, or `make fw-test`). Put every decision you can test there (boarding, report policy, SOS
  latch, trip state).
- Order of operations: sample → SD journal (with seq) → publish → advance the acked pointer. Never publish data
  that isn't journaled. Photos: `/p/<seq>.jpg` first, publish after the TAP event is acked.
- Payload keys exactly as `backend/platform/protocol/schemas/*.v1.json` (short keys). Changes → `/device-protocol`.
- No `delay()` outside drivers; ArduinoJson with fixed capacity; every module has `begin()`, task/`loop()`,
  `selfTest()`; logs via `LOGI/LOGW/LOGE`.
- Security: nothing secret in source or images; Secure Boot/flash-encryption settings live in
  `sdkconfig.production` and are applied only at the provisioning station.
- **Never** run `pio run -t upload`, `esptool`, `espefuse` or `espsecure` — a hook blocks them; the human flashes
  (`/hw-bringup`). Building (`pio run -e …`) and host tests are fine.
