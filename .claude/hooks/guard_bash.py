#!/usr/bin/env python3
"""PreToolUse hook (Bash): keep secrets and huge outputs out of the conversation.

Blocks shell commands that touch secret files (.env, db.env, *.pem, *.key), commands that dump whole
lock/generated/demo files, `curl … | sh`, permission-skipping flags, and commands that touch real hardware
(flashing, eFuses, serial monitor) unless .claude/local/hw_flash_allowed exists on this machine. Everything else passes untouched (exit 0, no output).
"""
import json
import os
import re
import sys
from pathlib import Path

try:
    event = json.load(sys.stdin)
except Exception:
    sys.exit(0)

cmd = (event.get("tool_input") or {}).get("command") or ""
root = Path(os.environ.get("CLAUDE_PROJECT_DIR") or event.get("cwd") or os.getcwd())

SECRET = re.compile(r"(?<![\w.\-])(db\.env|\.env|\.env\.(?!example\b)[\w.]+)(?![\w.\-])|[\w\-./]+\.(pem|key)\b")
DUMP = re.compile(r"\b(cat|less|more|bat|head|tail)\s+[^|;&]*(package-lock\.json|go\.sum|schema\.d\.ts|openapi\.json|"
                  r"internal/store/\w+/models\.go|reference/demo/source/js/\S+\.js)")
PIPE_TO_SHELL = re.compile(r"\b(curl|wget)\b[^|]*\|\s*(sudo\s+)?(ba|z)?sh\b")
HW = re.compile(r"\b(esptool(\.py)?|espefuse(\.py)?|espsecure(\.py)?|idf\.py\s+[^|;&]*\b(flash|efuse))\b"
                r"|\bpio\s+run\b[^|;&]*(-t|--target)\s*(upload|uploadfs)\b|\bpio\s+device\s+monitor\b")
KICAD_SRC = re.compile(r"\.kicad_(sch|pcb|pro|sym|mod|dru)\b|\b(sym|fp)-lib-table\b")
KICAD_OK = re.compile(r"^\s*(kicad-cli\b|bash\s+hardware/scripts/export\.sh\b|git\s+(status|diff\s+--stat|log|add|mv|rm)\b"
                      r"|ls\b|find\b|du\b|wc\s+-c\b)")
SKIP_PERMS = re.compile(r"--dangerously-skip-permissions|--permission-mode\s+bypass")

if SECRET.search(cmd):
    print("Blocked: command touches a secrets file. Variable names are in infra/.env.example and "
          "infra/db.env.example.", file=sys.stderr)
    sys.exit(2)
if DUMP.search(cmd):
    print("Blocked: that file is huge or generated. Grep for the line you need, or `npm ls <pkg>` / "
          "`go list -m <module>`.", file=sys.stderr)
    sys.exit(2)
if PIPE_TO_SHELL.search(cmd):
    print("Blocked: never pipe a downloaded script into a shell. Download, show it to the human, and ask.",
          file=sys.stderr)
    sys.exit(2)
if re.search(r"\bespefuse(\.py)?\b[^|;&]*\bburn", cmd):
    print("Blocked: eFuse burning is irreversible and only happens at the provisioning station.", file=sys.stderr)
    sys.exit(2)
if HW.search(cmd) and not (root / ".claude/local/hw_flash_allowed").exists():
    print("Blocked: flashing, eFuse and serial-monitor commands touch real hardware. The human runs them "
          "(/hw-bringup). To allow on this machine: touch .claude/local/hw_flash_allowed (never for espefuse burn).",
          file=sys.stderr)
    sys.exit(2)
if KICAD_SRC.search(cmd) and not all(KICAD_OK.match(part) for part in re.split(r"&&|\|\||;|\|", cmd) if part.strip()):
    print("Blocked: KiCad sources are read and edited in KiCad by the human designer. Use "
          "`bash hardware/scripts/export.sh <board>/<project>` and review its out/ files (hardware/README.md).",
          file=sys.stderr)
    sys.exit(2)
if SKIP_PERMS.search(cmd):
    print("Blocked: no permission-skipping flags in this repo.", file=sys.stderr)
    sys.exit(2)
sys.exit(0)
