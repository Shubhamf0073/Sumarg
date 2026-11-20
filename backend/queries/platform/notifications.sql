-- Worker → notification outbox (docs/02 A10). Runs as sf_platform; the claim is cross-tenant by design, every
-- write names the row's tenant.

-- name: ClaimOTPSMS :many
-- Worker → parent login codes due for sending, oldest first. Claimed by pushing next_attempt_at out by a lease, in a
-- short transaction; the SMS is sent after it commits, and each row is settled on its own. A worker that dies
-- mid-send leaves the row to be retried when the lease ends (or skipped as stale). stale: the code (5 min) has
-- expired, so sending it would be useless.
UPDATE notifications n
SET next_attempt_at = now() + make_interval(secs => @lease_secs::double precision)
FROM guardians g, tenants t
WHERE n.id IN (SELECT q.id FROM notifications q
               WHERE q.status = 'queued' AND q.channel = 'sms' AND q.event = 'otp' AND q.next_attempt_at <= now()
               ORDER BY q.next_attempt_at
               LIMIT @row_limit
               FOR UPDATE SKIP LOCKED)
  AND g.tenant_id = n.tenant_id AND g.id = n.guardian_id
  AND t.id = n.tenant_id
RETURNING n.id, n.tenant_id, n.dedupe_key, coalesce(n.data ->> 'box', '')::text AS box, n.attempts,
          (n.created_at < now() - interval '5 minutes')::boolean AS stale,
          g.mobile, g.app_enabled, t.short_name, t.is_demo;

-- name: SettleNotification :execrows
-- Worker → a row is done (sent, skipped or failed for good). The sealed code is wiped either way.
UPDATE notifications
SET status = @status, sent_at = CASE WHEN @status = 'sent' THEN now() END, error = sqlc.narg(err_text),
    attempts = attempts + CASE WHEN @status = 'skipped' THEN 0 ELSE 1 END, data = '{}'
WHERE tenant_id = @tenant_id AND id = @id AND status = 'queued';

-- name: RetryNotification :execrows
-- Worker → a send failed and may be retried after the delay.
UPDATE notifications
SET attempts = attempts + 1, error = @err_text::text, next_attempt_at = now() + make_interval(secs => @delay_secs::double precision)
WHERE tenant_id = @tenant_id AND id = @id AND status = 'queued';

-- name: PurgeOTPChallenges :execrows
-- Worker, hourly → parent-login challenges older than 24 h (mobile and IP, kept for nothing after the 1 h limit
-- window; docs/07 §3). Cross-tenant by design: the rows are expired in every school.
DELETE FROM otp_challenges WHERE created_at < now() - interval '24 hours';

-- name: PurgeOTPNotifications :execrows
-- Worker, hourly → settled login-code SMS rows older than 24 h (the code itself was wiped when it settled).
DELETE FROM notifications
WHERE event = 'otp' AND status <> 'queued' AND created_at < now() - interval '24 hours';
