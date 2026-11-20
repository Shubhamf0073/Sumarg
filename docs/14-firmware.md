# 14 — Firmware (ESP32-S3)

Built on **Previa Fleet's firmware design** (`previafleet/docs/07-firmware-design.md`): journal-first storage,
A/B OTA with rollback, installer mode, pure-C++ logic tested on the host. Sumarg adds the card reader, camera,
display, SOS and trip/roster handling.

## 1. Goals
1. **Never lose a tap or an SOS.** Every record/event gets a lifetime `seq` and goes to the SD journal before the
   network. Photos are written to `/p/<seq>.jpg` before publishing.
2. **Instant feedback without network.** Board/alight is decided on the unit (`src/logic/boarding.h`), the child
   hears the beep and sees their first name in < 300 ms.
3. **Never brick in the field.** A/B OTA, rollback if the new image doesn't reach MQTT within 10 min, watchdogs.
4. **Locked down.** Secure Boot v2, flash encryption (release), encrypted NVS, ROM download disabled in production.
5. **Testable without hardware.** `src/logic/` compiles on the host (`pio test -e native`); the Go simulator plays
   whole buses against the backend.

## 2. Layout
See `firmware/README.md`. Pins only in `include/board_pins.h`; build constants in `include/build_config.h`;
behaviour from the cloud config (docs/13 §6).

## 3. Tasks
Listed in `firmware/src/main.cpp`: `gps`, `rfid`, `camera`, `logic`, `store`, `net`, `health`. Queues: rfid→logic,
gps→logic, logic→camera (capture request with seq), logic→store, camera→store. `net` only reads from the journal.

## 4. State machine
`BOOT → NO_TRIP ⇄ TRIP_RUNNING`, orthogonal `SOS_LATCHED`, `OFFLINE` (modem down; everything else continues),
`INSTALLER` (DIP). Trip starts on `TRIP_START` command (driver app), or `geo`: leaving the campus geofence within a
route's start window (±30 min); ends on `TRIP_END`, arrival in the campus geofence (to-school), or 30 min after the
last stop with no taps (from-school).

## 5. Taps
1. Reader poll 20 Hz → UID (4/7/10 bytes) → `Roster::find` → `BoardingLedger::on_tap` → result.
2. Board/Alight: beep + "Name ✓"/"Bye Name"; capture request (if `tap.photo`); TAP event with `r` and `ph`.
3. Duplicate: short beep, event `r:"D"` (no photo). Unknown: long beep + red LED, event `r:"U"` with photo (who used
   an unknown card?). No trip: event `r:"N"`, no photo.
4. Roster capacity 128 (`MAX_ROSTER`); roster version in status. Unit reboot mid-trip: ledger restarts empty; the
   server corrects board/alight from history (docs/13 §3).

## 6. Photos
VGA JPEG, quality ≈ 12 (~25–45 KB), taken 150 ms after the tap (child facing the reader). Stored `/p/<seq>.jpg`,
published on `photo/{seq}` after the TAP event is acked; deleted from SD after PUBACK and 24 h. No photo is taken
outside a tap. Camera off when no trip is running.

## 7. Connectivity
TinyGSM (A7670/SIM7670), attach back-off 5 s → 15 s → 60 s → 5 min, modem power-cycle after 3 failed cycles.
TLS on the modem (`AT+CSSL…`) with ISRG Root X1; fallback mbedTLS over a raw socket. Publish backlog oldest-first,
≤ 10 in flight; live SOS jumps the queue (keeps its seq). Photos are sent after all pending telemetry.

## 8. OTA
`OTA` command with url (`https://devices.<domain>/fw/<version>.bin`), sha256, ECDSA P-256 signature, size →
download to the inactive slot with streaming sha256 → verify signature with the embedded Previa public key → set
boot partition → reboot → must connect and publish status within 10 min, else rollback. Channels dev → beta (5 %)
→ stable via `/release-firmware`. Never OTA during a running trip (unit defers until `NO_TRIP`).

## 9. Security
- Production units: Secure Boot v2, flash encryption RELEASE, NVS encryption (`sdkconfig.production`). eFuses are
  burned only at the provisioning station.
- Secrets: unit secret and Wi-Fi-free config in encrypted NVS; nothing secret in the image.
- Firmware signing key lives offline (hardware token or an air-gapped laptop); CI builds unsigned images, a human
  signs releases (`/release-firmware`).
- BLE/Wi-Fi radios off except the installer mode (Phase 2: BLE installer tool).
- The roster holds first names only; photos leave the SD card within 24 h after upload.

## 10. Coding rules
Previa Fleet rules: no `delay()` outside drivers; no heap in hot loops (ArduinoJson with fixed capacity); every
module has `begin()`, a task or `loop()`, and `selfTest()`; logs `LOGI/LOGW/LOGE(tag, …)` to serial + SD;
`src/logic/` is pure C++17 with host tests. Payload keys exactly as `backend/platform/protocol/schemas`.

## 11. Bench and field
- Installer mode (DIP=01): serial CLI `status`, `pins`, `gps`, `modem`, `sd`, `rfid`, `cam snap`, `sos`, `selftest`,
  `set apn …`. Used by `/hw-bringup` and the `hardware-bringup` subagent.
- Flashing from Claude is blocked by a hook unless the human enables it for that machine.
- Field logs: `CAPTURE` command uploads the last N KB of the SD log over MQTT (rate-limited).
