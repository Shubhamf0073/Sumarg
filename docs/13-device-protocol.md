# 13 — Unit ↔ cloud protocol (v1)

Base: **Previa Fleet protocol v1** (`backend/platform/protocol`, schemas in `backend/platform/protocol/schemas`,
reused unchanged where possible — docs/11). The JSON Schemas win if this page disagrees with them. Change both together
with `/device-protocol` (the `protocol-guardian` review applies).

## 1. Transport

- MQTT 3.1.1 over TLS 1.2+, `devices.sumarg.previasoftwares.com:8883` (provisioning value; overridable by
  `config.broker`). Keep-alive 120 s, clean session **false**, QoS 1.
- Client ID = username = unit uid (`SB-1A2B3C`); password = the unit's 32-char secret (encrypted NVS).
- CA: ISRG Root X1 pinned in firmware (Let's Encrypt). Certificates rotate every 60–90 days; the root does not.
- Last Will: `…/status` `{"v":1,"online":false}`, retained, QoS 1.
- Time: `t` = Unix seconds UTC from GNSS (network time before first fix, `tq`:1).

## 2. Topics

`{tk}` = tenant key (`_pool` until allocated; the server ignores it and uses the DB), `{id}` = unit uid.

| Dir | Topic | QoS | Retained | Payload |
|---|---|---|---|---|
| ↑ | `v1/t/{tk}/d/{id}/telemetry` | 1 | no | `telemetry.v1.json` — position records + events |
| ↑ | `v1/t/{tk}/d/{id}/status` | 1 | yes | `status.v1.json` — birth/health; LWT offline |
| ↑ | `v1/t/{tk}/d/{id}/ack` | 1 | no | `ack.v1.json` |
| ↑ | `v1/t/{tk}/d/{id}/photo/{seq}` | 1 | no | **binary JPEG** (≤ 60 KB) for the TAP event with the same `seq` — Sumarg addition |
| ↓ | `v1/t/{tk}/d/{id}/config` | 1 | yes | `config.v1.json` — full desired config incl. **roster** and **trip** (Sumarg additions) |
| ↓ | `v1/t/{tk}/d/{id}/cmd` | 1 | no | `command.v1.json` — OTA, REBOOT, TRIP_START, TRIP_END, SELFTEST |

## 3. Telemetry

Same envelope as Previa Fleet: `{"v":1,"did","fw","bt","recs":[…],"evs":[…]}`. Records use `s,t,la,lo,sp,hd,al,
sa,hp,ig,mv,vi,vb,cq` (mining-only keys `w,fu,dm,tl` are simply absent). Batching: 6 records, or 60 s while moving,
or at once on a priority event.

Sumarg event kinds (`k`, added to the schema enum before the first SB unit ships — additive, so v1 stays):

| `k` | `d` | Meaning |
|---|---|---|
| `TAP` | `{"uid":"04A1B2C3D4E5F6","r":"B"\|"A"\|"D"\|"U"\|"N","trip":123,"ph":1}` | card tap; `r` = unit decision Board/Alight/Duplicate/Unknown/No-trip; `ph`=1 → a photo follows on `photo/{s}` |
| `SOS` | `{"src":"btn1"\|"btn2"}` | SOS button pressed (latched until cleared) |
| `SOS_CLEAR` | `{"by":"key"\|"cmd"}` | cleared at the unit (key switch) or by command |
| `TRIP_START` / `TRIP_END` | `{"trip":123,"by":"btn"\|"cmd"\|"geo"}` | trip boundaries seen by the unit |
| `IGN_ON`, `IGN_OFF`, `POWER_CUT`, `POWER_RESTORE`, `TAMPER`, `GPS_LOST`, `GPS_FIX`, `BOOT`, `SD_ERROR`, `OVERSPEED_LOCAL` | as Previa Fleet | |

The server is authoritative: it recomputes board/alight from the trip's boarding history and may disagree with `r`
(e.g. the unit rebooted mid-trip). `D`/`N` taps are stored as events, not boardings.

## 4. Status (birth), retained

Previa Fleet `status.v1` plus `"rdr":{"ok":true}` (reader), `"cam":{"ok":true}`, `"roster":{"ver":17,"n":38}`,
`"trip":123`. Sent on connect and every 30 min. Missing reader/camera → unit-health warning on the admin screen.

## 5. Commands

Previa Fleet commands (`OTA`, `REBOOT`, `SELFTEST`, `CAPTURE`) plus `TRIP_START {"trip":id,"route":id}` /
`TRIP_END {"trip":id}` (sent when the driver starts/ends in the app) and `SOS_CLEAR`. Every command has `cid`; the
unit acks with `ack.v1` (`ok`/`error` + detail).

## 6. Config (retained, full document)

Previa Fleet `config.v1` (`cfgVer`, report policy, broker, APN, OTA channel…) plus:

```json
{ "cfgVer": 18,
  "school": { "tz": "Asia/Kolkata", "speedKmh": 40 },
  "routes": [ { "id": 9, "code": "R-03", "dir": "to_school", "start": "06:50", "geo": [[23.04, 72.51, 150]] } ],
  "roster": { "ver": 17, "cards": [ ["04A1B2C3D4E5F6", "Aarav"], ["0A1B2C3D", "Diya"] ] },
  "tap":    { "dedupeS": 5, "minRideS": 90, "photo": true, "jpegQ": 12, "frame": "VGA" },
  "report": { "movingS": 10, "idleS": 120, "distM": 150, "headingDeg": 30 } }
```
`roster.cards` holds card uid + **first name only** (shown on the unit's display). Max 128 cards per unit.

## 7. Versioning

Previa Fleet rules apply (docs reference in `platform/protocol` doc comments): additive optional fields within v1;
never tighten an existing field once units ship; breaking changes → `v2/…` topics, ingest supports N and N-1 for
6 months. Every change updates schema, examples, this page, firmware codec, ingest parser and the simulator.

## 8. Data budget (per bus, 22 school days)

| Traffic | Assumption | MB/month |
|---|---|---|
| Positions | 4 h/day on trips at 10 s, ~170 B/record + overhead | ≈ 12 |
| Idle/heartbeat/status | 2-min idle reports, status every 30 min | ≈ 3 |
| Taps | 30 students × 4 taps/day × ~120 B | < 1 |
| **Photos** | 120 taps/day × ~35 KB (VGA, q≈12) | ≈ 92 |
| OTA | one 1.5 MB image every ~2 months | ≈ 1 |
| **Total** | | **≈ 110 MB** (plan for 250 MB with retries and margin) |

Photos dominate: `tap.photo=false` or `frame:"QVGA"` per school halves the bill if a SIM plan is tight.
