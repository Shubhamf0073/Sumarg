---
name: review-task
description: Independent review of the current diff by the reviewer subagent (tenant isolation first).
disable-model-invocation: true
---
# /review-task

Run the `reviewer` subagent on the uncommitted changes (or `$ARGUMENTS` = a commit range). Give it only:
the task id and its acceptance criteria (2–3 lines). It reads the diff itself — don't paste code.

When it returns: fix every **blocker** and **major** finding, re-run `make lint test` (and
`make test-integration` if data access changed), then summarise in ≤ 5 lines what was fixed and what was noted
for later (add those as unchecked items under the milestone in docs/08-plan.md).
