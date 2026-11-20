---
name: preflight
description: Pre-release gate — every check CI runs, locally, in the order that fails fastest.
disable-model-invocation: true
---
# /preflight

Run in order; stop at the first failure and report it (command + ≤ 15 lines of output):
1. `make lint` (golangci-lint incl. depguard architecture rules, Biome)
2. `make sqlc` then `git diff --exit-code backend/internal/store` (generated code committed)
3. `make gen-api` then `git diff --exit-code frontend/src/api` (OpenAPI + TS types committed)
4. `make schema-diff` (goose migrations == docs/schema.sql)
5. `make test` and `make test-integration` (via the `test-runner` subagent)
6. `govulncheck ./...` in backend/ (if installed: `make tools`)
7. `cd frontend && npm run build` (typecheck + bundle; note the main chunk size — admin + maps must stay lazy)
8. `make e2e` if UI flows changed since the last tag

Finish with one line: ready / not ready (+ the first failing step).
