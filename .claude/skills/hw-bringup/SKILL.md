---
name: hw-bringup
description: Step-by-step bench bring-up of the SB unit (Waveshare bench prototype, SB-C1 carrier or SB-R1 PCB) with the human — power, rails, flashing a selftest image, modem, GNSS, reader, camera, SD, buttons — and filling board_pins.h from verified results.
disable-model-invocation: true
argument-hint: "[proto|c1|r1]"
---
# /hw-bringup $ARGUMENTS (human at the bench; adapted from Previa Fleet)

Claude never flashes or burns eFuses (hook-blocked). Claude prepares commands and reads logs the human pastes.

1. **Pin proposal first.** proto: the board's silkscreen revision (V1/V2) + Waveshare's tables as hints, then
   continuity checks for the modem UART/power pins (not published). c1/r1: `bash hardware/scripts/export.sh …`
   → `out/pins.txt` (rule check included). Rules: no strapping pins (0, 3, 45, 46) as inputs, nothing on GPIO 33–37
   (octal PSRAM), analog on ADC1 (GPIO 1–10), wake inputs on GPIO 0–21, PWRKEY via transistor. Write the proposal as
   a table — do not edit `board_pins.h` yet.
2. **Power** (human, meter): proto — USB-C 5 V; c1/r1 — `hardware/sb-c1/bringup-test.md` §1 first (12/24/36 V,
   reverse, cranking, parked drain), then rails, modem current spikes, battery charge.
3. **Selftest image** (human flashes `pio run -e sb_<rev> -t upload` with `-DSELFTEST`): installer CLI `selftest`.
4. Per block, human pastes the CLI output: `modem` (AT, CSQ, attach), `gps` (fix, sats), `rfid` (card UID),
   `cam snap` (JPEG size), `sd` (write/read), `sos` (both buttons + key), `ign`, `battery`.
5. Only verified pins go into `include/board_pins.h` (remove that rev's `#error`); unverified stay `#error`.
6. Record results in `docs/12-hardware.md` §4 (table: block, pin, verified on, by) and, for c1, in
   `hardware/sb-c1/release/<rev>/BRINGUP-<serial>.md`; tick the plan task.
