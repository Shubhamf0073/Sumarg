---
name: release
description: Tag a release and watch the deploy (human approves the push).
disable-model-invocation: true
argument-hint: "vX.Y.Z"
---
# /release $ARGUMENTS

1. `/preflight` must have passed on this commit. If not, run it first.
2. Check the deploy window: outside 09:00–21:00 IST unless the human says otherwise (docs/06 §6).
3. Summarise the release from `git log --oneline <last tag>..HEAD` in ≤ 8 lines, flag migrations
   (`git diff --stat <last tag> -- backend/migrations`) and anything that needs a manual step.
4. Ask before running `git tag $ARGUMENTS && git push origin $ARGUMENTS` (the push is in the `ask` list).
5. After the push: CI builds the image, uploads the SPA, runs `deploy.sh` over SSH. Tell the human what to watch:
   GitHub Actions run, `/api/healthz`, admin dashboard worker heartbeat. Rollback: `deploy.sh rollback` on the
   server (docs/06 §6) — never down-migrations.
