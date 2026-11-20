---
paths:
  - "infra/**"
  - ".github/**"
  - "backend/Dockerfile"
  - "Makefile"
---
# Infra rules (shared production server — be conservative)

- The VPS is shared with other Previa apps: bind only to 127.0.0.1, keep the memory caps in
  infra/docker-compose.yml (docs/02 §5), never touch other sites' Nginx files, containers, images or volumes.
- Edits under `infra/` and `.github/` need the human's OK (settings `ask`). Never run production compose
  commands, `ssh`, `scp` or deploys yourself.
- Secrets: `.env` (app) and `db.env` (database only) are never read or printed; variable names live in the
  `*.example` files — add new variables there with a comment.
- Images: distroless, non-root, read-only root fs, `no-new-privileges`, healthcheck = `/app/schoolfleet healthcheck`.
  Pin versions (base images, PlatformIO platform/libs, actions by SHA in workflows).
- Shell scripts: `set -euo pipefail`, idempotent, print what they change, pass `shellcheck`.
- Nginx: `nginx -t` before any reload; keep the security-headers and Cloudflare real-IP snippets included.
