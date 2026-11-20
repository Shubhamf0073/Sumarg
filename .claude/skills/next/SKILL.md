---
name: next
description: Do the next unchecked task in docs/08-plan.md end to end (plan, build, test, tick, commit).
disable-model-invocation: true
argument-hint: "[task id, e.g. M1.3 — default: first unchecked in the current milestone]"
---
# /next — one plan task, then stop

1. **Pick.** The SessionStart context names the next task. If `$ARGUMENTS` names a task, do that one.
2. **Model check.** Task tagged **[opus]** and you are not Opus → say so in one line and stop (the human runs
   `/model opus`). **[hw]** → confirm the human has the board connected before starting. Untagged → carry on.
3. **Read only what the task needs** — by section, not whole files:
   `bash .claude/scripts/docsec.sh docs/05-domain-rules.md "3. Attendance"`. Load the matching skill
   (`/api-endpoint`, `/db-migration`, `/device-protocol`, `/mobile-screen`, …) instead of re-deriving conventions.
4. **Restate** the acceptance criteria in 2–3 lines. If the docs contradict each other, fix the doc first (same
   commit) or ask.
5. **Build** the smallest change that meets them. Tests with **two schools** for anything touching data; host tests
   for firmware logic; a simulator scenario for the tap/live/alert path.
6. **Check:** `make lint test` (+ `make test-integration` if SQL/auth changed, `make fw-test` if firmware changed). Use the `test-runner`
   subagent for long outputs so failures come back summarised.
7. **[review] tasks:** run `/review-task` and fix findings before ticking.
8. **Finish:** tick `[x]` in docs/08-plan.md; commit `<task id>: <what changed>`; reply in ≤ 5 lines (what
   changed, anything deferred). Then stop — the human runs `/clear` before the next task.
