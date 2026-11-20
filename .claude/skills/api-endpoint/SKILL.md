---
name: api-endpoint
description: Use when adding or changing an HTTP endpoint (portal /school, apps /parent and /crew, admin /api/admin) — huma operation, service, sqlc queries, permissions, parent scope, tests, generated TS types.
---
# Endpoint (huma + service + sqlc)

**Contract first:** the route, permission and shape must be in `docs/04-api.md`
(`bash .claude/scripts/docsec.sh docs/04-api.md "<area>"`). Add/fix it there first.

Files in `backend/internal/{tenant,parent,driver}/<module>/` (admin: `internal/admin/<module>/`):
1. `queries/tenant/<module>.sql` (admin: `queries/platform/`) — one comment line naming the screen/action per
   query; no `tenant_id = $n` as a substitute for RLS; allow-listed sort maps; `limit+1` paging.
   `make sqlc`.
2. `handlers.go` — `huma.Register` with OperationID `<module>-<verb>`, method/path from docs/04, the permission
   (`perms.Require("students.write")`), input/output structs with validation tags (`maxLength`, `pattern`,
   `enum`, `minimum`). Money fields are `money.Paise` (integer paise in JSON). No logic here.
3. `service.go` — the action, using the request tx from the context (`tenantq.New(api.Tx(ctx))`):
   permission/scope check → conditional write (`… WHERE status = 'booked'` / `version = $n`, 0 rows → 409)
   → `audit.Record` → `notifications` rows if parents/staff must hear about it (`/push-notifications`).
   App writes (manual marks, SOS, incidents) honour `Idempotency-Key`.
   **Parent endpoints:** check every student/route/trip id against the guardian's children first (out of scope → 404).
4. Errors: `httpx.NewError(status, code, msg)` with codes from docs/04 "Errors"; other schools'/campuses' rows
   → 404 `not_found`, never 403.
5. Tests (`*_test.go`, `-tags integration` for DB): happy path, permission matrix (each role), the other
   school's id → 404, another campus's id for a campus user → 404, another guardian's child → 404, query budget.
6. `make gen-api` → commit `frontend/src/api/openapi.json` + both `schema.d.ts` (portal and `mobile/`).

GET handlers run in `InTenantReadOnly`; anything that writes must be a non-GET (read-only sessions and
suspended tenants are enforced by Postgres, not by remembering to check).
