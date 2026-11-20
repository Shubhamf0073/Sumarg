---
name: db-migration
description: Use for any schema change — new goose migration with RLS/tenancy rules, sqlc regeneration and schema-diff. Includes the tenant-table template.
---
# Schema change (goose + sqlc)

Order is fixed: **docs/schema.sql → new migration → `make sqlc` → code**. The baseline `00001_init.sql` is
schema.sql verbatim; committed migrations are never edited (hook-enforced).

1. Edit `docs/schema.sql` (the contract) so it describes the end state.
2. New file `backend/migrations/<next 5-digit number>_<verb>_<thing>.sql`:
   ```sql
   -- +goose Up
   ...
   -- +goose Down
   ...            -- local/dev only; production never runs Down (docs/06 §8)
   ```
   Function bodies with `;` inside → wrap that statement in `-- +goose StatementBegin` / `StatementEnd`.
   Big-table index → `-- +goose NO TRANSACTION` + `CREATE INDEX CONCURRENTLY`.
3. **New tenant table:** start from `rls-template.sql` in this folder — every line of it is required
   (tenant_id, `UNIQUE (tenant_id, id)`, composite FKs, ENABLE + FORCE RLS, `tenant_isolation` policy, grants:
   sf_app no DELETE; audit-like tables SELECT/INSERT only). Platform-only table: no tenant_id, no grant to
   sf_app, add it to the "Deliberately NOT granted" comment in schema.sql.
4. Breaking change = expand → backfill (batched) → contract in a later release, and only after the human
   approves the contract step.
5. Money `bigint …_paise` (sqlc maps it to `money.Paise`); enums TEXT + CHECK; no hard deletes (status +
   reason); `version int` on human-edited records.
6. Run: `make migrate && make sqlc && make schema-diff && make test-integration`. The isolation suite fails if a
   table with tenant_id lacks forced RLS, the policy, or has a DELETE grant for sf_app.
7. Touches existing data, constraints or RLS → ask the `migration-checker` subagent before committing.
