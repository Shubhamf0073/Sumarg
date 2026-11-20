---
name: release-firmware
description: Build, version, sign and stage a firmware release and roll it out by channel (dev → beta → stable) with rollback monitoring.
disable-model-invocation: true
argument-hint: "vX.Y.Z"
---
# /release-firmware $ARGUMENTS (adapted from Previa Fleet)

1. `make fw-test` green; `pio run -e sb_c1 -e sb_proto` builds (and `-e sb_r1` once HW_REV 2 has pins); CHANGELOG entry in `firmware/README.md`.
2. Bump `FW_VERSION` in `platformio.ini`; tag `fw-$ARGUMENTS` (ask before pushing).
3. CI attaches unsigned `.bin` files. The human signs on the offline machine (Secure Boot v2 signature +
   ECDSA P-256 OTA signature over sha256) — Claude prints the exact commands, never holds the key.
4. Upload in `/admin → Firmware` (version, hw revs, sha256, signature) → channel `dev` (bench units).
5. `beta`: 5 % of installed units, none mid-trip (units defer OTA until no trip), watch 48 h: online count, rollback
   events, boot reasons, tap counts vs. previous week.
6. `stable` only if beta had 0 rollbacks and no new error events. Rollback = set the previous version as target.
