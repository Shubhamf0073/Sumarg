# 04 — API contract

Base `https://sumarg.previasoftwares.com/api`. JSON, UTF-8, times RFC 3339 UTC, money integer paise, ids int64.
Machine contract = OpenAPI 3.1 built by huma from the Go handlers; `make gen-api` writes
`frontend/src/api/openapi.json` + TS types for the portal and the app *(Cargo)*. Keep this page and the code in step.

| Topic | Rule |
|---|---|
| Auth | Portal: cookie `__Host-sf_sid` + CSRF. Apps: `Authorization: Bearer <tenant_id>.<token>` (no cookies, no CSRF). Admin: `__Host-sf_admin` (passkey). |
| Errors | `{"error":{"code","message","fields"}}`; out-of-scope ids → 404 `not_found` |
| Paging | `?limit=50&after=<cursor>` → `{items, next}` (limit+1) |
| Idempotency | Manual marks, incidents, SOS from the app carry `Idempotency-Key` |
| Live | Live endpoints are served from memory, `Cache-Control: no-store`, polled every 10 s only on live screens |

## Auth (all clients)
| Method | Path | Body → Result |
|---|---|---|
| POST | `/auth/login` | portal: `{school_code, mobile, password}` → cookie |
| POST | `/auth/app/pin` | driver/attendant: `{school_code, mobile, pin, install_id}` → `{token, user, school}` |
| POST | `/auth/app/otp` | parent step 1: `{school_code, mobile}` → 202 always (no oracle) |
| POST | `/auth/app/otp/verify` | `{school_code, mobile, code, install_id}` → `{token, guardian, school}` |
| POST | `/auth/push-token` | `{platform, fcm_token, app_version}` |
| POST | `/auth/logout` · GET `/me` | |

## Parent app (`/parent`)
| Method | Path | Notes |
|---|---|---|
| GET | `/parent/home` | children, today's trips per child (status, boarded/alighted times), unread announcements — one call |
| GET | `/parent/children/{id}` | child card: routes, stops, guardians (names only), recent boardings (14 days) |
| GET | `/parent/live/{route_id}` | `{bus, at, lat, lon, heading, speed, next_stop, eta_my_stop, online}` — only while that child's trip runs, else 409 `no_trip` |
| GET | `/parent/trips/{id}` | trip timeline (stops passed, my child's board/alight, photo thumbnail link for own child) |
| GET | `/parent/boardings/{id}/photo` | own child only; X-Accel; expires with retention |
| POST | `/parent/absences` · DELETE `/parent/absences/{id}` | `{student_id, date, direction, reason}` |
| GET/POST | `/parent/requests` | stop change / pickup person / contact |
| GET | `/parent/notifications` | push history (30 days) |
| PATCH | `/parent/settings` | language, notification toggles per event |

## Driver / attendant app (`/crew`)
| Method | Path | Notes |
|---|---|---|
| GET | `/crew/today` | assigned trips with stops, students per stop (first name, class, photo thumb, absences, special-needs note) |
| POST | `/crew/trips/{id}/inspection` | checklist items → passed/failed |
| POST | `/crew/trips/{id}/start` · `/end` | sends TRIP_START/TRIP_END to the unit |
| POST | `/crew/trips/{id}/boardings` | manual mark `{student_id, kind, reason, qr?}` (Idempotency-Key) |
| POST | `/crew/sos` | `{trip_id?, lat, lon}` → critical alert |
| POST | `/crew/incidents` | multipart: fields + up to 4 photos |
| GET | `/crew/trips/{id}/live` | own bus position + next stop (unit data) |

## School portal (`/school`)
Dashboard (`GET /school/dashboard` — counts, open alerts, trips today, units online), live board
(`GET /school/live` — all buses of the user's scope), campuses, vehicles, crew, routes + stops (`PUT /school/routes/{id}/stops`
→ Routes API once), students (+ `POST /school/students/import` CSV/XLSX), cards (`POST /school/students/{id}/card`,
`DELETE …/card` with reason), guardians + invites (`POST /school/guardians/invite` → link/QR), trips (list, detail,
cancel), boardings (void with reason), alerts (ack/resolve), incidents, absences, requests (approve/reject),
announcements, reports (`GET /school/reports/{kind}.csv?from&to&route`), settings, audit, users. All list endpoints
follow the paging rule; all writes are audited.

## Admin (`/admin`, platform pool)
Auth (passkeys, as Cargo) · schools (onboard wizard, status, package) · subscriptions · invoices (draft/issue/paid/void)
· agents + commissions · units (provision, allocate, install, replace, retire, send command, status history) ·
firmware releases (upload signed image, channel, staged rollout %, rollout report) · support view · dashboard
(schools, units online, alerts/day, data per unit) · platform audit.

## Units
Units never call HTTP except OTA downloads from `https://devices.<domain>/fw/…`. Everything else is MQTT (docs/13).
