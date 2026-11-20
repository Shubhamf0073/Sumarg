---
name: push-notifications
description: Use for parent/staff notifications — enqueueing in the business transaction, FCM HTTP v1 sending from the worker, templates per language, dedupe, token cleanup, deep links.
---
# Notifications (FCM HTTP v1)

Rules: `docsec.sh docs/05-domain-rules.md "6. Notifications"`, decision docs/02 A10.

- **Enqueue** = insert into `notifications` in the same transaction as the boarding/alert/trip change, with
  `dedupe_key` (`<event>:<entity id>:<recipient>`) and `ON CONFLICT DO NOTHING`. Recipients: guardians of the
  student (`student_guardians`) who have app sessions; staff by role/campus for alerts.
- **Worker** (platform pool): claim `status='queued' AND next_attempt_at <= now()` with `FOR UPDATE SKIP LOCKED`;
  send to every valid `push_tokens` row of the recipient's sessions; OAuth2 token from the service account cached
  ~55 min; backoff 30 s → 2 m → 10 m; `UNREGISTERED`/`INVALID_ARGUMENT` → mark token invalid; give up after 4 tries.
- Payload: `notification` (title/body in the guardian's language) + `data.screen` deep link; Android channel per
  severity (SOS channel = high importance, sound); iOS `interruption-level` time-sensitive for SOS only.
- Late updates (event older than 10 min) add "(delayed update)". Never put photos or other children's names in a push.
- SMS only for OTP and the optional fallback (`tenant_settings.sms_fallback`), via the gateway adapter.
- Tests: rollback ⇒ no notification; duplicate event ⇒ one row; language selection; token cleanup; fake FCM server.
