#!/usr/bin/env bash
# docsec.sh — print ONE section of a markdown doc instead of reading the whole file (saves tokens).
#   .claude/scripts/docsec.sh docs/05-domain-rules.md            # table of contents (headings only)
#   .claude/scripts/docsec.sh docs/05-domain-rules.md "12."      # section whose heading contains "12."
#   .claude/scripts/docsec.sh docs/04-api.md "Dispatch"          # first heading containing "Dispatch" (case-insensitive)
# Prints from the matching heading to the line before the next heading of the same or higher level.
# Ignores '#' lines inside fenced code blocks.
set -euo pipefail
file="${1:?usage: docsec.sh <file.md> [heading text]}"
key="${2:-}"
if [ -z "$key" ]; then
  awk '/^```/{f=!f} !f && /^#+ /{print NR": "$0}' "$file"
  exit 0
fi
awk -v key="$key" '
  /^```/ { fence = !fence }
  !fence && /^#+ / {
    lvl = match($0, /[^#]/) - 1
    if (printing && lvl <= start) exit
    if (!printing && index(tolower($0), tolower(key))) { printing = 1; start = lvl }
  }
  printing { print }
' "$file"
