-- Previa admin sign-in: passkeys only (docs/02 A24, docs/07 S6). Runs as sf_platform; these tables are not tenant
-- data, so no tenant is named. Only sha256 of enrolment and session tokens is stored.

-- ── Staff and enrolment links ──

-- name: UpsertPlatformUser :one
-- migrate create-admin: the staff member by e-mail, created on first use. An existing user keeps their role, and is
-- re-activated: the command is how a locked-out admin is let back in.
INSERT INTO platform_users (name, email, role)
VALUES (@name, @email, @role)
ON CONFLICT (email) DO UPDATE SET is_active = true
RETURNING id, name, email, role;

-- name: CreateEnrolment :one
-- A one-time link that registers a passkey for the user. Earlier unused links for the same user stop working.
WITH spent AS (
    UPDATE platform_enrolments SET used_at = now()
    WHERE platform_user_id = @platform_user_id AND used_at IS NULL
)
INSERT INTO platform_enrolments (platform_user_id, token_hash, expires_at, created_by)
VALUES (@platform_user_id, @token_hash, now() + make_interval(secs => @lifetime_secs::double precision),
        sqlc.narg(created_by))
RETURNING id, expires_at;

-- name: GetEnrolment :one
-- Enrolment step 1: the link is live (unused, unexpired) and its user active. Nothing is consumed yet.
SELECT e.id, u.id AS user_id, u.name, u.email
FROM platform_enrolments e
JOIN platform_users u ON u.id = e.platform_user_id
WHERE e.token_hash = @token_hash AND e.used_at IS NULL AND e.expires_at > now() AND u.is_active;

-- name: UseEnrolment :execrows
-- Enrolment step 2: consumes the link, once. 0 rows → it was used, replaced or expired in between.
UPDATE platform_enrolments e SET used_at = now()
FROM platform_users u
WHERE e.id = @id AND e.platform_user_id = @platform_user_id AND u.id = e.platform_user_id
  AND e.used_at IS NULL AND e.expires_at > now() AND u.is_active;

-- name: GetPlatformUser :one
-- Enrolment step 2: the user the link was for, still active.
SELECT id, name, email, role FROM platform_users WHERE id = @id AND is_active;

-- ── Passkeys ──

-- name: ListUserCredentials :many
-- Registration excludes the user's existing passkeys (an authenticator can't enrol twice).
SELECT credential FROM platform_passkeys
WHERE platform_user_id = @platform_user_id AND revoked_at IS NULL
ORDER BY id;

-- name: CreatePasskey :one
INSERT INTO platform_passkeys (platform_user_id, credential_id, credential, label)
VALUES (@platform_user_id, @credential_id, @credential, @label)
RETURNING id;

-- name: GetPasskeyForLogin :one
-- Sign-in: the passkey the browser offered, live, of an active user.
SELECT p.id, p.credential, u.id AS user_id, u.name, u.email, u.role
FROM platform_passkeys p
JOIN platform_users u ON u.id = p.platform_user_id
WHERE p.credential_id = @credential_id AND p.revoked_at IS NULL AND u.is_active;

-- name: UsePasskey :exec
-- After a sign-in: the credential as updated by the ceremony (sign count, backup state).
UPDATE platform_passkeys SET credential = @credential, last_used_at = now()
WHERE id = @id AND revoked_at IS NULL;

-- ── Sessions ──

-- name: CreatePlatformSession :one
INSERT INTO platform_sessions (token_hash, platform_user_id, passkey_id, expires_at, ip, user_agent)
VALUES (@token_hash, @platform_user_id, @passkey_id, now() + make_interval(hours => @lifetime_hours::int),
        sqlc.narg(ip), @user_agent)
RETURNING id;

-- name: MarkPlatformLogin :exec
UPDATE platform_users SET last_login_at = now() WHERE id = @id;

-- name: GetPlatformSession :one
-- Every admin request. Ends when it expires (ADMIN_SESSION_HOURS, absolute), is revoked, or its user is deactivated
-- or its passkey revoked.
SELECT s.id, u.id AS user_id, u.name, u.email, u.role,
       (s.last_seen_at < now() - interval '5 minutes')::boolean AS touch_due
FROM platform_sessions s
JOIN platform_users u ON u.id = s.platform_user_id
JOIN platform_passkeys p ON p.id = s.passkey_id
WHERE s.token_hash = @token_hash AND s.revoked_at IS NULL AND s.expires_at > now()
  AND u.is_active AND p.revoked_at IS NULL;

-- name: TouchPlatformSession :exec
UPDATE platform_sessions SET last_seen_at = now() WHERE id = @id AND revoked_at IS NULL;

-- name: RevokePlatformSessionByHash :exec
-- Logout, and sign-in presented with an old admin session (no fixation).
UPDATE platform_sessions SET revoked_at = now() WHERE token_hash = @token_hash AND revoked_at IS NULL;

-- ── Audit ──

-- name: InsertPlatformAudit :exec
INSERT INTO platform_audit_log (platform_user_id, actor_name, action, tenant_id, entity, entity_id, summary, ip,
                                request_id)
VALUES (sqlc.narg(platform_user_id), @actor_name, @action, sqlc.narg(tenant_id), @entity, sqlc.narg(entity_id),
        @summary, sqlc.narg(ip), sqlc.narg(request_id));
