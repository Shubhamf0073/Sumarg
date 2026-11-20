#!/usr/bin/env python3
"""PreToolUse hook (Read|Edit|Write): zero-exception file rules, enforced without spending model turns.

Blocks (exit 2, reason on stderr → shown to Claude):
  * any access to secrets (.env, db.env, *.pem, *.key, secrets/)
  * reading lock/sum files and big generated files whole (grep them instead)
  * editing generated files (sqlc output, API types/spec), the read-only demo reference, the approved
    dependency list (human-only), and goose migrations already committed
  * editing KiCad sources or released fab packages (humans own them in KiCad), and reading KiCad sources whole
    (review the exports from hardware/scripts/export.sh instead)
"""
import json
import os
import subprocess
import sys
from pathlib import Path

try:
    event = json.load(sys.stdin)
except Exception:
    sys.exit(0)

tool = event.get("tool_name", "")
path_str = (event.get("tool_input") or {}).get("file_path") or ""
if not path_str:
    sys.exit(0)

root = Path(os.environ.get("CLAUDE_PROJECT_DIR") or event.get("cwd") or os.getcwd()).resolve()
path = Path(path_str)
if not path.is_absolute():
    path = (Path(event.get("cwd") or root) / path)
try:
    rel = path.resolve().relative_to(root).as_posix()
except ValueError:
    rel = path.as_posix()  # outside the repo: only the secret rules below apply
name = path.name


def block(reason: str) -> None:
    print(reason, file=sys.stderr)
    sys.exit(2)


# 1. Secrets — never, for any tool.
if name in {".env", "db.env"} or name.startswith(".env.") and name != ".env.example" \
        or name.endswith((".pem", ".key")) or "/secrets/" in f"/{rel}":
    block(f"Blocked: {rel} holds secrets. Use infra/.env.example / infra/db.env.example for variable names.")

# 2. Big generated / lock files — don't read whole.
GENERATED = ("backend/internal/store/",)            # sqlc output
GENERATED_FILES = {"frontend/src/api/schema.d.ts", "frontend/src/api/openapi.json", "mobile/src/api/schema.d.ts"}
if tool == "Read":
    if rel in {"frontend/package-lock.json", "backend/go.sum"}:
        block("Blocked: lock/sum file. Use `npm ls <pkg>` or `go list -m <module>` (via Bash) for what you need.")
    if rel.startswith("reference/demo/source/js/") and rel.endswith(".js"):
        block("Blocked: demo bundles are 30–90 KB each. Grep for the screen you need "
              "(grep -n \"Boarding Verification\" reference/demo/source/js/*.js) and read only those lines.")
    if rel in GENERATED_FILES or (rel.startswith(GENERATED) and rel.endswith("models.go")):
        block("Blocked: generated and large. Grep it for what you need (e.g. `grep -n 'type Student ' "
              f"{rel}`), or read the source: backend/queries/*.sql, docs/schema.sql, the Go handler structs.")
KICAD_SRC = (".kicad_sch", ".kicad_pcb", ".kicad_pro", ".kicad_sym", ".kicad_mod", ".kicad_dru", ".kicad_jobset",
             ".kicad_wks")
if rel.startswith("hardware/") and (name.endswith(KICAD_SRC) or name in {"sym-lib-table", "fp-lib-table"}):
    if tool == "Read":
        block("Blocked: KiCad sources are large S-expressions. Run `bash hardware/scripts/export.sh <board>/<project>` "
              "and read out/netlist-summary.txt, out/pins.txt, out/erc.json, out/drc.json, out/bom.csv instead.")
    block("Blocked: KiCad files are edited by the human designer in KiCad only (hardware/README.md). Write the "
          "change as a review note or a SPEC.md/bom.csv update for the designer.")
if tool in {"Edit", "Write"} and rel.startswith("hardware/") and "/release/" in rel:
    block("Blocked: released fab packages are frozen. Make a new rev folder from fresh exports.")

# 3. Files that must not be edited.
if tool in {"Edit", "Write"}:
    if rel in GENERATED_FILES or rel.startswith(GENERATED):
        block("Blocked: generated file. Change backend/queries/*.sql or the Go handler, then `make sqlc` / "
              "`make gen-api`.")
    if rel.startswith("reference/demo/"):
        block("Blocked: reference/demo is the read-only UI spec. Port into frontend/src instead.")
    if rel == ".claude/approved-deps.txt":
        block("Blocked: the approved dependency list is edited by the human only. Ask them, following "
              "/add-dependency.")
    if rel.startswith("backend/migrations/") and rel.endswith(".sql"):
        tracked = subprocess.run(["git", "cat-file", "-e", f"HEAD:{rel}"], cwd=root,
                                 capture_output=True).returncode == 0
        if tracked:
            block("Blocked: this goose migration is already committed (may be applied somewhere). Add the next "
                  "numbered file instead (/db-migration, expand/contract).")
sys.exit(0)
