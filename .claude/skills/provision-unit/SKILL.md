---
name: provision-unit
description: Register new SB units — uid from MAC, secret generation, IMEI/ICCID, hw rev, channel — and the provisioning-station steps (secure boot + flash encryption eFuses) that the human runs.
disable-model-invocation: true
argument-hint: "<count> <hw_rev>"
---
# /provision-unit (adapted from Previa Fleet `/provision-device`)

1. Station checklist (human): production signing key present (offline token), `sdkconfig.production` build of the
   provisioning image, labels printer, test card, 12 V supply.
2. Per unit (human runs the station script; Claude never runs it): read MAC → uid `SB-XXXXXX` → generate a 32-char
   secret (`devicesecret.Generate`) → burn Secure Boot v2 + flash encryption (release) → write uid, secret, broker
   host and CA into encrypted NVS → selftest → print the QR label.
3. Register in `/admin → Units` (uid, IMEI, ICCID, hw rev, fw version, channel); the secret is entered once and
   stored only as argon2id. Unit status `stock`.
4. Never write a secret into a file in the repo, a log, a ticket or a chat. A lost secret = re-provision.
5. Allocation and installation happen later in the admin (school) and installer (bus) screens (docs/12 §4).
