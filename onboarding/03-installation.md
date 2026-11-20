# 03 — Route survey, bus survey and installation

## Route survey (for stops without a verified pin)
- Preferred: the transport manager places/adjusts pins in the portal map editor using the school's Google Maps
  links (Routes → Edit).
- Where the school has no reliable list: a **ride-along** with the driver app's survey mode (crew app → Route
  survey: tap "Stop here" at each stop; the app records the GPS point and time) → the route and planned offsets come
  out of the ride; the transport manager names the stops after.

## Bus survey (one line per bus, in the installer app or a sheet)
| Field | Why |
|---|---|
| Reg no, label, seats | match `vehicles.csv` |
| Supply: 12 V or 24 V; permanent and ignition (ACC) points; fuse box access | harness plan (docs/12 §8) |
| Mounting spot by the front door at 1.0–1.2 m; sun exposure | tap zone reach, heat (docs/12 §5) |
| Antenna path: LTE on the dashboard glass, GNSS with sky view; cable ≤ 3 m | signal |
| Existing AIS-140 tracker vendor and its wiring (do not touch) | docs/02 A29 |
| Signal check with a phone on the same operator as the unit SIM (bars at depot and on route) | CSQ expectations |

## Installation day (per bus, ~60–90 min, installer pair)
1. Unit, harness, antennas, SOS buttons, fuse, cable ties, labels; the installer card; the installer app logged in.
2. Mount the enclosure; route the harness to permanent + ignition through the 3 A inline fuse; ground to chassis.
3. Antennas: LTE on the glass, GNSS with sky view; strain-relieve every cable.
4. SOS buttons: driver's reach and by the door; key switch with the driver's keys.
5. Power on; installer app: **scan unit QR** → select bus → checks run: GNSS fix, CSQ ≥ 12, test tap with the
   installer card (photo arrives), each SOS button (alert arrives on the portal and is cleared with the key),
   ignition on/off seen.
6. Photo of the fitted unit and the harness; installer signs → unit `installed` (docs/05 §13).
7. Leave the bus as found; note anything unusual in the installer app.

## Sign-off
The transport manager signs the installation list (app: Units → Installed today) at the end of each day.
Spare units (1 per 15 buses) stay at the school's transport office, provisioned and allocated.
