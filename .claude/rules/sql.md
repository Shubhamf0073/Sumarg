---
paths:
  - "backend/queries/**"
  - "backend/migrations/**"
  - "docs/schema.sql"
  - "backend/sqlc.yaml"
---
# SQL rules (queries, migrations, schema)

- Order: `docs/schema.sql` → new goose migration (never edit a committed one) → `make sqlc` → Go. Use
  `/db-migration` for any schema change; the tenant-table template is in that skill.
- Tenant queries (`queries/tenant/`) rely on RLS: no `tenant_id = $n` as a substitute; composite lookups may
  still include tenant_id for index use. Platform queries (`queries/platform/`) bypass RLS: every one that
  touches tenant tables filters by tenant explicitly.
- One leading comment per query naming the screen/action it serves. `-- name: X :one|:many|:exec`.
- Lists: keyset or `limit+1`, never `COUNT(*)`; dynamic sort/filter via allow-listed CASE/params, never
  string-built SQL.
- Writes that change state are conditional (`WHERE status = 'booked'` / `version = @version`); check rows
  affected in Go.
- Money `bigint *_paise`; enums TEXT + CHECK; no hard deletes; `timestamptz` UTC; business dates `date` (IST).
- `SECURITY DEFINER` functions: `tenant_by_code` and the two position-partition functions. Adding another needs the
  human's approval and an ADR line in docs/02.
- `positions` is partitioned by day: always filter by `at` range so Postgres prunes partitions; never index-scan all days.
