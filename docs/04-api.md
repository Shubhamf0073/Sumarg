# 04 — API contract

Base `https://sumarg.previasoftwares.com/api`. JSON, UTF-8, times RFC 3339 UTC, money integer paise, ids int64.
Machine contract = OpenAPI 3.1 built by huma from the Go handlers; `make gen-api` writes
`frontend/src/api/openapi.json` + TS types for the portal and the app *(Cargo)*. Keep this page and the code in step.

| Topic | Rule |
|---|---|
| Auth | Portal: cookie `__Host-sf_sid` + CSRF (writes carry `Origin` and `X-CSRF-Token` = cookie `__Host-sf_csrf`). Apps: `Authorization: Bearer <tenant_id>.<token>` **and** `X-Install-Id: <install_id>` on every request (device binding; no cookies, no CSRF). A portal token as bearer, or an app token as cookie → 401. Admin: `__Host-sf_admin` (passkey). |
| Errors | flat `{"code","message","request_id","fields"?}` (`fields` = `[{"path","message"}]`, validation errors only); out-of-scope ids → 404 `not_found` |
| Paging | `?limit=50&after=<cursor>` → `{items, next}` (limit+1) |
| Idempotency | Manual marks, incidents, SOS from the app carry `Idempotency-Key` |
| Live | Live endpoints are served from memory, `Cache-Control: no-store`, polled every 10 s only on live screens |

## Auth (all clients)
| Method | Path | Body → Result |
|---|---|---|
| POST | `/auth/login` | portal: `{school_code, mobile, password}` → cookie |
| POST | `/auth/app/pin` | driver/attendant: `{school_code, mobile, pin, install_id}` → `{token, user, school}` |
| POST | `/auth/app/otp` | parent step 1: `{school_code, mobile}` → 202 always (no oracle: unknown school/mobile, over the per-mobile or per-school limit — the SMS goes only to invited guardians, sent by the worker) |
| POST | `/auth/app/otp/verify` | `{school_code, mobile, code, install_id}` → `{token, client, guardian, school, read_only}`; 90-day session bound to the install, renewed on use; one per guardian per install |
| POST | `/auth/push-token` | parent/driver app session: `{platform, fcm_token, app_version}` → 204; the token moves to this session, the session's older tokens are retired; portal → 403; read-only session → 204, not stored |
| POST | `/auth/set-password` | portal staff, pre-session: `{token, password}` (token `<tenant_id>.<secret>` from the one-time link's `#fragment`; password 10–128 chars; Origin must be the portal's) → 204; the link is spent, other links for the user stop working, every session of the user ends, `audit_log` row; 410 `link_invalid` for a malformed/used/replaced/expired link or an inactive user; 429 shares the per-IP sign-in limit |
| POST | `/auth/logout` · GET `/me` | logout → 204 (works for read-only sessions); `/me` → `{client, user?, guardian?, school, read_only}` |

Auth errors: 401 `bad_credentials` (one body for unknown school, unknown mobile, wrong secret, wrong kind of account, backoff), 401 `unauthenticated` (no/invalid/expired session), 403 `csrf_failed`, 423 `school_closed`, 429 `rate_limited` (per client IP on `/auth/app/otp*`: 30 requests / 60 verifies per 10 min; says nothing about the mobile) (cancelled school, shown only after the secret verifies).

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
Auth (passkeys only: `POST /api/admin/auth/enrol/{begin,finish}` with the one-time link's token, `POST /api/admin/auth/login/{begin,finish}` (discoverable), `POST /api/admin/auth/logout`, `GET /api/admin/me`; writes need Origin + `X-CSRF-Token` = `__Host-sf_admin_csrf`) · schools: `POST /api/admin/schools` (onboarding wizard, one save: `{school, campuses[1–20], settings, owner, package}` → 201 `{id, code, onboarding_stage, period_end, campus_ids, owner_link{user_id, url, expires_at}}` — the link is shown once; 409 `code_taken`, 422 `campus_code_duplicate` / `agent_invalid` / `date_invalid`), `GET /api/admin/schools?after=` (board, 200 per page, `next` cursor), `GET /api/admin/schools/{id}` (details, campuses, owner/admin logins with `password_set`, active package), `PATCH /api/admin/schools/{id}/onboarding` (`{from_stage, stage, go_live_on?}`; stages may move back; 409 `stage_changed` if the stage moved meanwhile; 422 `stage_live` for leaving `live`; to `live` stores at most today IST in the same statement — no date or a future one → today, a past one kept), `POST /api/admin/schools/{id}/users/{user_id}/password-link` (new one-time link, older ones spent; another school's user → 404), `DELETE …/password-link` (cancel the unused links, no new one → 204), `POST /api/admin/schools/{id}/users/{user_id}/deactivate` (owner/admin login off, its sessions ended and links spent → 204; 409 `user_inactive`), `POST …/reactivate` (back on with an unusable password + a new one-time link → 201 `{user_id, url, expires_at}`; 409 `user_active`). Wizard constraint backstops: 409 `duplicate`, 422 `mobile_invalid` / `value_invalid`. GETs run in a READ ONLY platform tx. Roles: write = superadmin, support; read = also billing, installer. Every write → `platform_audit_log` with `tenant_id` · status changes and package edits (billing) → M1.4e · subscriptions · invoices (draft/issue/paid/void)
· agents + commissions · units (provision, allocate, install, replace, retire, send command, status history) ·
firmware releases (upload signed image, channel, staged rollout %, rollout report) · support view · dashboard
(schools, units online, alerts/day, data per unit) · platform audit.

## Units
Units never call HTTP except OTA downloads from `https://devices.<domain>/fw/…`. Everything else is MQTT (docs/13).
