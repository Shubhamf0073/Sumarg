#!/usr/bin/env python3
"""PostToolUse hook (Edit|Write): auto-format the file Claude just changed.
Silent when the file is clean (no tokens spent). If problems remain that the formatter can't fix, prints at
most 15 lines to stderr and exits 2 so Claude sees them right away. Skips quietly when a tool isn't installed.
Go: gofmt -w (+ goimports when installed); a compile/vet check is left to the Stop hook (package-level, slower).
"""
import json
import os
import shutil
import subprocess
import sys
from pathlib import Path

try:
    event = json.load(sys.stdin)
except Exception:
    sys.exit(0)
path_str = (event.get("tool_input") or {}).get("file_path") or ""
if not path_str:
    sys.exit(0)
root = Path(os.environ.get("CLAUDE_PROJECT_DIR") or event.get("cwd") or os.getcwd()).resolve()
path = Path(path_str).resolve()
try:
    rel = path.relative_to(root).as_posix()
except ValueError:
    sys.exit(0)


def run(cmd: list[str], cwd: Path) -> subprocess.CompletedProcess:
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=45)


def tool(name: str) -> str | None:
    local = root / ".bin" / name  # `make tools` installs here
    return str(local) if local.exists() else shutil.which(name)


problems = ""
if rel.startswith("backend/") and rel.endswith(".go") and "/internal/store/" not in f"/{rel}":
    goimports = tool("goimports")
    if goimports:
        res = run([goimports, "-w", "-local", "github.com/Shubhamf0073/previa-school-fleet", str(path)], root)
    elif tool("gofmt"):
        res = run([tool("gofmt"), "-w", str(path)], root)
    else:
        sys.exit(0)
    if res.returncode != 0:  # syntax errors: gofmt can't parse the file
        problems = (res.stdout + res.stderr).strip()
elif rel.startswith(("frontend/", "mobile/")) and rel.endswith((".ts", ".tsx", ".css", ".json", ".js")):
    app = root / rel.split("/", 1)[0]
    if (app / "node_modules/.bin/biome").exists():
        target = str(path.relative_to(app))
        res = run(["./node_modules/.bin/biome", "check", "--write", "--reporter=summary", target], app)
        if res.returncode != 0:
            problems = (res.stdout + res.stderr).strip()
elif rel.startswith("firmware/") and rel.endswith((".cpp", ".h")) and tool("clang-format"):
    res = run([tool("clang-format"), "-i", "--style=file:" + str(root / "firmware/.clang-format"), str(path)], root)
    if res.returncode != 0:
        problems = (res.stdout + res.stderr).strip()

if problems:
    lines = problems.splitlines()
    print("\n".join(lines[:15]) + (f"\n… {len(lines) - 15} more" if len(lines) > 15 else ""), file=sys.stderr)
    sys.exit(2)
sys.exit(0)
