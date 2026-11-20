#!/usr/bin/env python3
"""Stop hook: before Claude finishes, check only what changed in the working tree.
  * Go: `go vet` + `golangci-lint` on the changed packages only (fast: seconds, cached)
  * go.mod: every direct requirement must be on .claude/approved-deps.txt
  * sqlc: if queries/ or migrations/ changed, generated code must be up to date (`sqlc diff`)
  * frontend + mobile: Biome on changed TS files
  * firmware: host compile check of src/logic (g++ -fsyntax-only)
Silent when clean. On problems prints ≤ 20 lines to stderr and exits 2 so Claude fixes them instead of ending
the turn. Never re-blocks a stop it already blocked (no loops) and never runs tests (test-runner's job).
"""
import json
import os
import re
import shutil
import subprocess
import sys
from pathlib import Path

try:
    event = json.load(sys.stdin)
except Exception:
    event = {}
if event.get("stop_hook_active"):
    sys.exit(0)
root = Path(os.environ.get("CLAUDE_PROJECT_DIR") or event.get("cwd") or os.getcwd()).resolve()
backend = root / "backend"


def run(cmd: list[str], cwd: Path) -> subprocess.CompletedProcess:
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=55)


def tool(name: str) -> str | None:
    local = root / ".bin" / name
    return str(local) if local.exists() else shutil.which(name)


try:
    changed = run(["git", "status", "--porcelain", "--untracked-files=all"], root).stdout.splitlines()
except Exception:
    sys.exit(0)
files = [ln[3:].split(" -> ")[-1] for ln in changed if ln[:2].strip() != "D"]
out: list[str] = []

go_files = [f for f in files if f.startswith("backend/") and f.endswith(".go")]
if go_files and tool("go"):
    pkgs = sorted({"./" + str(Path(f).parent.relative_to("backend")) for f in go_files})
    res = run([tool("go"), "vet", "-tags", "integration", *pkgs], backend)
    if res.returncode != 0:
        out += (res.stdout + res.stderr).strip().splitlines()
    elif tool("golangci-lint"):
        res = run([tool("golangci-lint"), "run", "--output.text.print-issued-lines=false", *pkgs], backend)
        if res.returncode != 0:
            out += [ln for ln in (res.stdout + res.stderr).strip().splitlines() if ln.strip()]

if "backend/go.mod" in files:
    approved = set()
    try:
        for line in (root / ".claude/approved-deps.txt").read_text().splitlines():
            parts = line.split("#", 1)[0].split()
            if len(parts) >= 2 and parts[0] in ("go", "tool"):
                approved.add(parts[1])
    except FileNotFoundError:
        pass
    text = (backend / "go.mod").read_text()
    direct = re.findall(r"^\s*([\w.\-/]+\.[\w.\-/]+)\s+v\S+\s*$", text, re.M)  # lines without "// indirect"
    bad = [m for m in direct if not any(m == a or m.startswith(a.rstrip("/") + "/") for a in approved)]
    if bad:
        out.append("go.mod has modules not on .claude/approved-deps.txt: " + ", ".join(bad)
                   + " — remove them or ask the human (/add-dependency).")

if any(f.startswith(("backend/queries/", "backend/migrations/")) for f in files) and tool("sqlc"):
    res = run([tool("sqlc"), "diff"], backend)
    if res.returncode != 0:
        out.append("sqlc output is stale: run `make sqlc` and commit the generated code.")

for app in ("frontend", "mobile"):
    ts = [f[len(app) + 1:] for f in files if f.startswith(app + "/") and f.endswith((".ts", ".tsx"))]
    if ts and (root / app / "node_modules/.bin/biome").exists():
        res = run(["./node_modules/.bin/biome", "check", "--reporter=summary", *ts], root / app)
        if res.returncode != 0:
            out += (res.stdout + res.stderr).strip().splitlines()

# Firmware logic must keep compiling on the host (fast g++ syntax check; full tests are `make fw-test`).
logic = [f for f in files if f.startswith("firmware/src/logic/") and f.endswith((".cpp", ".h"))]
if logic and tool("g++"):
    cpps = [str(root / f) for f in files if f.startswith("firmware/src/logic/") and f.endswith(".cpp")] or \
           [str(p) for p in (root / "firmware/src/logic").glob("*.cpp")]
    res = run([tool("g++"), "-std=c++17", "-Wall", "-Wextra", "-Werror", "-DUNIT_TEST", "-fsyntax-only", *cpps], root)
    if res.returncode != 0:
        out += (res.stdout + res.stderr).strip().splitlines()

if out:
    print("Problems in changed files (fix before finishing):\n" + "\n".join(out[:20])
          + (f"\n… {len(out) - 20} more" if len(out) > 20 else ""), file=sys.stderr)
    sys.exit(2)
sys.exit(0)
