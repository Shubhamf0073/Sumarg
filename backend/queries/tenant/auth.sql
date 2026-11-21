-- Auth queries (docs/02 A27, docs/05 §10). Tenant queries run inside db.InTenant, so RLS limits them to the
-- session's tenant; never add `tenant_id = $x` filters as a substitute for that.

-- name: TenantByCode :one
-- Login step 1 (portal, driver app, parent app): school code → tenant. SECURITY DEFINER; the only query that runs
-- before app.tenant_id is set. Status is checked only after the secret/OTP verifies (no oracle).
SELECT f.tenant_id::bigint AS tenant_id, f.status::text AS status
FROM tenant_by_code(@code::text) AS f(tenant_id, status);

-- name: StaffForLogin :one
-- Portal / driver app login step 2, inside db.InTenant(tenant from step 1). The backoff per address is in the API
-- process; the daily cap over every address is decided by ClaimLoginAttempt.
SELECT id, name, role, campus_id, password_hash, pin_hash, must_change_secret, failed_logins, is_active
FROM users
WHERE mobile = @mobile;

-- name: CurrentSchool :one
-- Login response: the school the session belongs to (RLS shows only the current tenant's row).
SELECT id, code, display_name FROM tenants WHERE id = app_tenant();

-- name: ClaimLoginAttempt :one
-- Portal / driver app login, before the secret is verified: count the attempt as a failure up front, atomically, and
-- only while the user's daily cap over every address isn't reached (docs/05 §10). No row = capped: every try,
-- even the right secret, gets the same 401 until the 24 h window ends. Parallel guesses can't exceed the cap.
-- A success takes its own attempt back out (StaffLoginSucceeded) but doesn't reset the window, so a sign-in doesn't
-- hand an attacker a fresh budget.
UPDATE users
SET failed_logins = least(failed_logins + 1, 1000),
    fail_window_start = CASE WHEN fail_window_start IS NULL OR fail_window_start <= now() - interval '24 hours'
                             THEN now() ELSE fail_window_start END,
    fail_window_count = CASE WHEN fail_window_start IS NULL OR fail_window_start <= now() - interval '24 hours'
                             THEN 1 ELSE fail_window_count + 1 END
WHERE id = @id
  AND (fail_window_start IS NULL OR fail_window_start <= now() - interval '24 hours'
       OR fail_window_count < @max_per_day::smallint)
RETURNING fail_window_count;

-- name: AssignXactID :exec
-- Portal / driver app login, when no attempt was claimed (unknown mobile, wrong kind of account, backing off): give
-- the transaction an id so its commit is flushed like a claim's, and the timing doesn't say whether the mobile exists.
SELECT pg_current_xact_id();

-- name: KnownDriverInstall :one
-- Driver app PIN login, only once the daily cap is closed: has this install had a session for this user? Then it is
-- the user's own phone and may keep trying at a reduced rate (docs/05 §10). Revoked and expired sessions count.
SELECT EXISTS (SELECT 1 FROM sessions
               WHERE user_id = @user_id AND client = 'driver_app' AND install_id = @install_id)::boolean;

-- name: StaffLoginSucceeded :exec
-- Portal / driver app login: the secret verified; reset the run of failures and, when this attempt was claimed up
-- front (counted), take it back out of the daily count, so only failures count towards the cap. A known device's try
-- past a closed cap was never counted, so nothing is taken back. The window itself is kept (see ClaimLoginAttempt).
-- Unconditional on purpose: the row lock it takes serialises two logins of the same user, so "one driver app session
-- per user" holds even when two phones sign in at once.
UPDATE users
SET failed_logins = 0,
    fail_window_count = CASE WHEN @counted::boolean THEN greatest(fail_window_count - 1, 0) ELSE fail_window_count END
WHERE id = @id;

-- name: GuardianForLogin :one
-- Parent app: after the OTP verifies, inside db.InTenant.
SELECT id, name, language, app_enabled FROM guardians WHERE mobile = @mobile;

-- name: CreateSession :one
-- Portal / driver app / parent app login: a new session. Only the token's sha256 is stored.
INSERT INTO sessions (tenant_id, token_hash, user_id, guardian_id, client, install_id, read_only, expires_at, ip, user_agent)
VALUES (app_tenant(), @token_hash, sqlc.narg(user_id), sqlc.narg(guardian_id), @client, sqlc.narg(install_id),
        @read_only, now() + make_interval(secs => @lifetime_secs::double precision), sqlc.narg(ip), @user_agent)
RETURNING id;

-- name: RevokeSessionByHash :exec
-- Logout, and login presented with an old session (no fixation).
UPDATE sessions SET revoked_at = now() WHERE token_hash = @token_hash AND revoked_at IS NULL;

-- name: RevokeSession :exec
-- Logout: the signed-in session.
UPDATE sessions SET revoked_at = now() WHERE id = @id AND revoked_at IS NULL;

-- name: RevokeDriverAppSessions :exec
-- Driver app PIN login: one active app session per crew user, so a new phone logs the old one out (docs/05 §10).
UPDATE sessions SET revoked_at = now()
WHERE user_id = @user_id AND client = 'driver_app' AND revoked_at IS NULL;

-- name: GetSession :one
-- Every authenticated request, inside db.InTenant(tenant id parsed from the token): a forged tenant id finds no
-- row under RLS → 401. Validity is enforced here, not in Go: portal sessions also end after 12 h idle.
SELECT s.id, s.client, s.user_id, s.guardian_id, s.platform_user_id, s.install_id, s.read_only,
       (s.last_seen_at < now() - interval '5 minutes')::boolean AS touch_due,
       u.role AS user_role, u.campus_id AS user_campus_id, u.must_change_secret AS user_must_change_secret,
       coalesce(u.name, g.name, '')::text AS principal_name,
       t.status AS tenant_status, t.code AS tenant_code, t.display_name AS tenant_name
FROM sessions s
JOIN tenants t ON t.id = s.tenant_id
LEFT JOIN users u ON u.tenant_id = s.tenant_id AND u.id = s.user_id
LEFT JOIN guardians g ON g.tenant_id = s.tenant_id AND g.id = s.guardian_id
-- TODO(support-sessions): platform_user_id sessions are not checked here (sf_app cannot read platform_users).
-- When support sessions are built: give them a short expires_at and revoke them in the same platform
-- transaction that deactivates the platform user or revokes their platform session.
WHERE s.token_hash = @token_hash
  AND s.revoked_at IS NULL
  AND s.expires_at > now()
  AND (s.client <> 'web' OR s.last_seen_at > now() - interval '12 hours')
  AND (s.user_id IS NULL OR u.is_active)
  AND (s.guardian_id IS NULL OR g.app_enabled);

-- name: TouchSession :exec
-- Every authenticated request, at most once per 5 minutes (touch_due). App sessions are renewed for 90 days on use;
-- portal sessions keep their 7-day absolute end.
UPDATE sessions
SET last_seen_at = now(),
    expires_at = CASE WHEN client = 'web' THEN expires_at ELSE now() + interval '90 days' END
WHERE id = @id AND revoked_at IS NULL;

-- ── Parent OTP (docs/05 §10, docs/07 S4) ──

-- name: LockOTPIssue :exec
-- Parent app → request code: serialise code requests within the school, so the per-mobile and per-school counts
-- below can't be raced past (at ≤ 10 SMS a minute per school, the wait is nothing).
SELECT pg_advisory_xact_lock(hashtextextended('sumarg:otp-issue:' || app_tenant()::text, 0));

-- name: OTPIssueCounts :one
-- Parent app → request code: codes issued to this mobile in the last hour (max 3) and code SMS queued by the
-- school in the last minute (max 10, or 60 in the launch window). Unknown mobiles get a challenge too, so they count
-- the same. launch_window: the school isn't live yet, or went live in the last 14 days (IST dates) — every parent
-- signs in then, so the per-school limit is raised (docs/05 §10).
SELECT (SELECT count(*) FROM otp_challenges c
        WHERE c.mobile = @mobile AND c.created_at > now() - interval '1 hour')::integer AS mobile_hour,
       (SELECT count(*) FROM notifications n
        WHERE n.event = 'otp' AND n.created_at > now() - interval '1 minute')::integer AS school_minute,
       -- COALESCE: a live school with no go_live_on (or no row) is outside the window, never NULL.
       COALESCE((SELECT t.onboarding_stage <> 'live'
                        OR t.go_live_on > (now() AT TIME ZONE 'Asia/Kolkata')::date - 14
                 FROM tenants t WHERE t.id = app_tenant()), false)::boolean AS launch_window;

-- name: EndLiveOTPs :exec
-- Parent app → request code: one live challenge per mobile; a new code ends the older ones.
UPDATE otp_challenges SET consumed_at = now() WHERE mobile = @mobile AND consumed_at IS NULL;

-- name: CreateOTPChallenge :one
-- Parent app → request code: valid 5 minutes. Only the HMAC is stored.
INSERT INTO otp_challenges (tenant_id, mobile, code_hash, expires_at, ip)
VALUES (app_tenant(), @mobile, @code_hash, now() + interval '5 minutes', sqlc.narg(ip))
RETURNING id;

-- name: QueueOTPSMS :exec
-- Parent app → request code, invited guardians only: the SMS the worker sends. data.box holds the code sealed for
-- this row (internal/shared/otp); the worker wipes it once the row is settled.
INSERT INTO notifications (tenant_id, guardian_id, event, channel, title, body, data, dedupe_key)
VALUES (app_tenant(), @guardian_id, 'otp', 'sms', '', '', jsonb_build_object('box', @box::text), @dedupe_key);

-- name: LiveOTPForVerify :one
-- Parent app → verify code: the mobile's newest open challenge, locked so parallel guesses are counted one by one.
SELECT id, code_hash, attempts, (expires_at > now())::boolean AS live
FROM otp_challenges
WHERE mobile = @mobile AND consumed_at IS NULL
ORDER BY created_at DESC
LIMIT 1
FOR UPDATE;

-- name: CountOTPAttempt :exec
-- Parent app → verify code: the try is counted before the code is compared (max 5 per challenge).
UPDATE otp_challenges SET attempts = attempts + 1 WHERE id = @id AND attempts < 5;

-- name: ConsumeOTP :exec
-- Parent app → verify code: the code matched; it can't be used again.
UPDATE otp_challenges SET consumed_at = now() WHERE id = @id;

-- name: RevokeGuardianInstallSessions :exec
-- Parent app sign-in: a new session on an install replaces the guardian's older one there (other phones stay).
UPDATE sessions SET revoked_at = now()
WHERE guardian_id = @guardian_id AND client = 'parent_app' AND install_id = @install_id AND revoked_at IS NULL;

-- ── Push tokens (docs/04 /auth/push-token) ──

-- name: UpsertPushToken :one
-- Parent / driver app → after sign-in and whenever FCM rotates the token: the token belongs to this session now.
INSERT INTO push_tokens (tenant_id, session_id, platform, fcm_token, app_version)
VALUES (app_tenant(), @session_id, @platform, @fcm_token, @app_version)
ON CONFLICT (tenant_id, fcm_token) DO UPDATE
SET session_id = excluded.session_id, platform = excluded.platform, app_version = excluded.app_version,
    updated_at = now(), invalid_at = NULL
RETURNING id;

-- name: RetireOtherPushTokens :exec
-- Parent / driver app → push-token registration: one live token per session, so a rotated token isn't pushed twice.
UPDATE push_tokens SET invalid_at = now()
WHERE session_id = @session_id AND id <> @keep_id AND invalid_at IS NULL;

-- ── Set-password links (docs/05 §10) ──

-- name: SecretLinkIsLive :one
-- Set password, read-only, before the new password is hashed: a bogus link costs no Argon2 work.
SELECT EXISTS (SELECT 1 FROM user_secret_links l JOIN users u ON u.id = l.user_id
               WHERE l.token_hash = @token_hash AND l.used_at IS NULL AND l.expires_at > now()
                 AND u.is_active AND u.role IN ('owner','admin','transport_manager'))::boolean;

-- name: UseSecretLink :one
-- Set password (link from the wizard or a "new password link"): consumes the link once, for an active portal user.
-- No row → used, replaced, expired, or the user can't sign in to the portal.
UPDATE user_secret_links l SET used_at = now()
FROM users u
WHERE l.token_hash = @token_hash AND l.used_at IS NULL AND l.expires_at > now()
  AND u.id = l.user_id AND u.is_active AND u.role IN ('owner','admin','transport_manager')
RETURNING l.id, u.id AS user_id;

-- name: SetStaffPassword :exec
-- Set password: the new hash; the run of failures and the daily window are cleared (the link proved who it is).
UPDATE users
SET password_hash = @password_hash, must_change_secret = false, failed_logins = 0, fail_window_start = NULL,
    fail_window_count = 0, version = version + 1
WHERE id = @id;

-- name: SpendSecretLinks :exec
-- Set password: any other live link for the user stops working.
UPDATE user_secret_links SET used_at = now() WHERE user_id = @user_id AND used_at IS NULL;

-- name: RevokeUserSessions :exec
-- Set password: every session of the user ends (a new password signs everyone out).
UPDATE sessions SET revoked_at = now() WHERE user_id = @user_id AND revoked_at IS NULL;

-- name: InsertAudit :exec
-- A school-side change, by a user or (no user) by the holder of a one-time link.
INSERT INTO audit_log (tenant_id, user_id, action, entity, entity_id, summary, ip, request_id)
VALUES (app_tenant(), sqlc.narg(user_id), @action, @entity, sqlc.narg(entity_id), @summary, sqlc.narg(ip),
        sqlc.narg(request_id));
