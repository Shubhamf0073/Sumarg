-- Auth queries (docs/02 A27, docs/05 §10). Tenant queries run inside db.InTenant, so RLS limits them to the
-- session's tenant; never add `tenant_id = $x` filters as a substitute for that.

-- name: TenantByCode :one
-- Login step 1 (portal, driver app, parent app): school code → tenant. SECURITY DEFINER; the only query that runs
-- before app.tenant_id is set. Status is checked only after the secret/OTP verifies (no oracle).
SELECT f.tenant_id::bigint AS tenant_id, f.status::text AS status
FROM tenant_by_code(@code::text) AS f(tenant_id, status);

-- name: StaffForLogin :one
-- Portal / driver app login step 2, inside db.InTenant(tenant from step 1). backing_off: the backoff window after
-- repeated failures is still open (decided by the database clock, like every other session time).
SELECT id, name, role, campus_id, password_hash, pin_hash, must_change_secret, failed_logins, is_active,
       (locked_until IS NOT NULL AND locked_until > now())::boolean AS backing_off
FROM users
WHERE mobile = @mobile;

-- name: CurrentSchool :one
-- Login response: the school the session belongs to (RLS shows only the current tenant's row).
SELECT id, code, display_name FROM tenants WHERE id = app_tenant();

-- name: ClaimLoginAttempt :one
-- Portal / driver app login, before the secret is verified: count the attempt as a failure up front, atomically, and
-- only if no backoff window is open. No row = backing off. Parallel guesses therefore can't all slip in before the
-- 5th failure is recorded. From the 5th failure on, the next try waits 30 s, doubling up to 15 min (docs/05 §10).
UPDATE users
SET failed_logins = least(failed_logins + 1, 1000),
    locked_until = CASE WHEN failed_logins + 1 >= 5
                        THEN now() + make_interval(secs => least(30 * power(2, least(failed_logins + 1 - 5, 10)), 900))
                   END
WHERE id = @id AND (locked_until IS NULL OR locked_until <= now())
RETURNING id;

-- name: AssignXactID :exec
-- Portal / driver app login, when no attempt was claimed (unknown mobile, wrong kind of account, backing off): give
-- the transaction an id so its commit is flushed like a claim's, and the timing doesn't say whether the mobile exists.
SELECT pg_current_xact_id();

-- name: StaffLoginSucceeded :exec
-- Portal / driver app login: the secret verified; forget the failures (including the one claimed for this attempt).
-- Unconditional on purpose: the row lock it takes serialises two logins of the same user, so "one driver app
-- session per user" holds even when two phones sign in at once.
UPDATE users SET failed_logins = 0, locked_until = NULL WHERE id = @id;

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
