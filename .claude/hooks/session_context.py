#!/usr/bin/env python3
"""SessionStart hook: inject a 4-line orientation so Claude doesn't read the whole plan.

Prints: current milestone, next unchecked task (with its [opus]/[review] tags), git branch and
uncommitted file count, and the working rhythm. Plain-text stdout is added to Claude's context.
"""
import os
import re
import subprocess
import sys
from pathlib import Path

root = Path(os.environ.get("CLAUDE_PROJECT_DIR") or os.getcwd())
plan = root / "docs" / "08-plan.md"


def git(*args: str) -> str:
    try:
        return subprocess.run(["git", *args], cwd=root, capture_output=True, text=True, timeout=5).stdout.strip()
    except Exception:
        return ""


milestone, task = "?", "none found — check docs/08-plan.md"
if plan.exists():
    text = plan.read_text(encoding="utf-8")
    m = re.search(r"\*\*Current milestone:\s*(M\d+)\*\*", text)
    if m:
        milestone = m.group(1)
    # first unchecked task in the current milestone, else first unchecked anywhere in Phase 1
    phase1 = text.split("## Phase 2")[0]
    candidates = re.findall(r"^- \[ \] (M\d+\.\d+.*)$", phase1, re.M)
    in_ms = [c for c in candidates if c.startswith(milestone + ".")]
    pick = (in_ms or candidates or [None])[0]
    if pick:
        task = pick if len(pick) <= 220 else pick[:217] + "..."

branch = git("rev-parse", "--abbrev-ref", "HEAD") or "(no git yet)"
dirty = git("status", "--porcelain")
dirty_n = len([ln for ln in dirty.splitlines() if ln.strip()]) if dirty else 0

print(f"Sumarg — Phase 1, current milestone {milestone}.")
print(f"Next task: {task}")
print(f"Git: branch {branch}, {dirty_n} uncommitted file(s).")
print("Rhythm: /next does one task → commit → user runs /clear. Read docs by section with "
      ".claude/scripts/docsec.sh <file> [heading]. Tags: [opus] needs /model opus; [review] needs the reviewer subagent.")
sys.exit(0)
