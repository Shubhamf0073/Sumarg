---
paths:
  - "backend/**/*.go"
  - "backend/templates/**"
---
# Go backend rules (loaded only when working under backend/)

Layout and conventions: docs/02 §6–7. Lint enforces most of this (depguard, forbidigo, gosec, errorlint).

- **Packages:** `platform/` is shared with Previa Cargo (keep it product-agnostic); `internal/{tenant,parent,driver}/*`
  run as sf_app under RLS; `internal/admin/*`, `internal/worker`, `internal/ingest` use `platformdb` (BYPASSRLS — name the tenant in every query);
  `internal/shared/*` is imported by both and must not import platform-pool code. `internal/store/*` is sqlc
  output — never edit, never read `models.go` whole (grep).
- **Requests:** the API middleware opens the tx (`db.InTenant`/`InTenantReadOnly`); services take
  `ctx` first and use the tx from it. Handlers: huma structs + validation tags, call one service function,
  no logic.
- **Errors:** wrap `%w`; client errors are `*httpx.Error` via `httpx.NewError(status, code, msg)`; anything
  else is a 500 with the request id. Never return or log `PgError.Detail`, SQL, tokens, passwords, OTPs.
- **Types:** money `money.Paise` / rates `money.BasisPoints` (never float); time via injected `clock.Clock`
  (business dates IST); ids are `int64` (DB) — parse untrusted ids strictly; tokens from `crypto/rand`.
- **Config** only through `platform/config` structs (validated at start). No `os.Getenv`, no `fmt.Print`,
  no `time.Now`, no outbound `http.Get` (lint fails).
- **Logging:** `httpx.Logger(ctx)` / `log.InfoContext(ctx, …)` so `request_id` and `tenant_id` are attached.
- **Concurrency:** no goroutines in request handlers except via errgroup with the request ctx; Argon2 (passwords, unit
  secrets) behind a semaphore of 2.
- **Tests:** table-driven; `-tags integration` for anything touching Postgres, with two tenants; fake clock,
  never `time.Sleep` for timing; fuzz money math. `go test ./...` must pass with no database.
- Go notes for this codebase: `context.Context` first; small interfaces declared where they're used;
  `pgx.BeginTxFunc` handles rollback; named types (`Paise`) don't mix with plain ints without a conversion;
  integer overflow wraps silently — use `money.Sum` / `math/bits`.
