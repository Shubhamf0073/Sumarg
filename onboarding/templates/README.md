# Data templates — columns and rules

UTF-8 CSV or XLSX with exactly these headers (order doesn't matter; extra columns are ignored with a warning).
**R** = required. Dates: `DD-MM-YYYY` or `YYYY-MM-DD`. Times: `HH:MM` (24 h). Mobiles: 10 digits starting 6–9
(`+91`, leading `0`, spaces and hyphens are stripped). Full working examples: `../demo/`.

## campuses.csv
| Column | R | Rule |
|---|---|---|
| campus_code | R | 2–8 letters/digits, upper case (`MAIN`) |
| name | R | |
| address | | |
| lat, lon | R* | decimal degrees; *or* give `maps_link` (a Google Maps link to the gate) and we fill them |
| maps_link | | |
| geofence_m | | 30–1000, default 150 (arrival detection radius) |

## staff.csv (portal users)
| Column | R | Rule |
|---|---|---|
| campus_code | | blank = all campuses (owner, trust admin) |
| name, mobile | R | mobile unique per school |
| email | | |
| role | R | `owner`, `admin` or `transport_manager` |

## vehicles.csv
| Column | R | Rule |
|---|---|---|
| campus_code | R | |
| reg_no | R | e.g. `GJ01AB1234` (spaces/hyphens removed) |
| label | R | what the school calls it: `Bus 12` |
| make_model | | |
| seats | R | 4–80 |
| ownership | | `school` (default) or `contractor` |
| contractor_name | | when ownership = contractor |
| fitness_expiry, insurance_expiry, permit_expiry, puc_expiry | | dates; expiry alerts go to the transport desk |
| vltd_vendor | | the bus's existing AIS-140 tracker brand, if known |

## crew.csv
| Column | R | Rule |
|---|---|---|
| campus_code | R | |
| kind | R | `driver` or `attendant` |
| name, mobile | R | mobile unique per school (used for app login) |
| licence_no, licence_expiry | R for drivers | |
| badge_no, police_verified_on, medical_checked_on | | |

## routes_stops.csv — one row per stop
| Column | R | Rule |
|---|---|---|
| campus_code | R | |
| route_code | R | 2–12 of `A–Z 0–9 -` (`R-01`) |
| route_name | R | repeat on every row of the route |
| direction | R | `to_school` or `from_school` (morning and afternoon are separate routes) |
| shift | | `regular` (default), `half_day`, `activity` |
| vehicle_reg_no | | must exist in vehicles.csv |
| driver_mobile, attendant_mobile | | must exist in crew.csv |
| planned_start | R | departure time from the first stop (`to_school`) or from school (`from_school`) |
| stop_seq | R | 1, 2, 3 … in driving order, max 25 per route |
| stop_name | R | unique within the route; students refer to it |
| lat, lon / maps_link / address | R* | at least one; pins without lat/lon are marked "needs pin" until set on the map |
| radius_m | | 20–300, default 60 |
| planned_offset_min | | minutes from planned_start; if blank, computed from the road route when saved |

## students_guardians.csv — one row per student using transport
| Column | R | Rule |
|---|---|---|
| campus_code, admission_no | R | admission_no unique per school |
| student_name | R | full name; units show the first name only |
| class, section | R / | `Grade 5`, `B` |
| to_route_code, to_stop_name | R* | morning; *at least one of morning/afternoon |
| from_route_code, from_stop_name | R* | afternoon |
| guardian1_name, guardian1_mobile | R | |
| guardian1_relation | R | `mother`, `father`, `guardian`, `grandparent`, `sibling`, `other` |
| guardian1_language | | `en` (default), `gu`, `hi` — notification language |
| guardian2_… | | same rules, optional |
| special_needs_note | | shown to crew only (e.g. "needs help on the step") — keep it short |

## cards.csv
| Column | R | Rule |
|---|---|---|
| admission_no | R | must exist |
| card_uid | R | hex, 8–20 characters (from the card encoder file or the portal's desk reader) |

A student can hold one active card; importing a new UID for the same student revokes the old card.
