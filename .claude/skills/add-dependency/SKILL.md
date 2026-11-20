---
name: add-dependency
description: Vetting checklist for a new Go module or npm package; the human verifies and edits .claude/approved-deps.txt.
disable-model-invocation: true
argument-hint: "<package> <why>"
---
# /add-dependency (human-approved) — reused from previalogistic

Why so strict: LLMs suggest non-existent package names surprisingly often and attackers register those names
(slopsquatting); typosquats have stayed in public module mirrors for years. docs/07 S15.

## Claude does
1. Explain **why** the standard library or an approved package (`.claude/approved-deps.txt`) can't do it —
   often it can; prefer that.
2. Give the exact name and the URL of its **official** repository and registry page (pkg.go.dev / npmjs.com).
   Don't guess; if unsure the package exists, say so.
3. List the checks below with what you could find (stars are not evidence).
4. Stop. Don't install — the hooks block it anyway.

## The human verifies
- [ ] Registry page exists and links to the expected repo; name matches exactly (watch `-go`/`go-`, swapped
      letters, extra scopes).
- [ ] Reputable maintainer; no recent ownership transfer; active repo.
- [ ] Older than ~1 year, real dependents; latest version ≥ 7 days old.
- [ ] Licence MIT/BSD/Apache-2.0/ISC.
- [ ] npm: no install scripts. Go: module path matches the repo; `GOSUMDB` on.
- [ ] Few transitive deps; `govulncheck` clean.

## Then
The human adds one line to `.claude/approved-deps.txt` (`go x/y  # why, verified by <name>, <date>`) and a row
or note in docs/02 §2, in the same commit as the code that uses it.
