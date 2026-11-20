-- Auth queries (docs/02 A27, docs/05 §10). Tenant queries run inside db.InTenant, so RLS limits them to the
-- session's tenant; never add `tenant_id = $x` filters as a substitute for that.

-- name: TenantByCode :one
-- Login step 1 (portal, driver app, parent app): school code → tenant. SECURITY DEFINER; the only query that runs
-- before app.tenant_id is set. Status is checked only after the secret/OTP verifies (no oracle).
SELECT f.tenant_id::bigint AS tenant_id, f.status::text AS status
FROM tenant_by_code(@code::text) AS f(tenant_id, status);

-- name: StaffForLogin :one
-- Portal / driver app login step 2, inside db.InTenant(tenant from step 1).
SELECT id, name, role, campus_id, password_hash, pin_hash, must_change_secret, failed_logins, locked_until, is_active
FROM users
WHERE mobile = @mobile;

-- name: GuardianForLogin :one
-- Parent app: after the OTP verifies, inside db.InTenant.
SELECT id, name, language, app_enabled FROM guardians WHERE mobile = @mobile;

-- name: GetSession :one
-- Every authenticated request, inside db.InTenant(tenant id parsed from the token): a forged tenant id finds no
-- row under RLS → 401. Validity is enforced here, not in Go.
SELECT s.id, s.client, s.user_id, s.guardian_id, s.platform_user_id, s.install_id, s.read_only, s.last_seen_at,
       s.expires_at, u.role AS user_role, u.campus_id AS user_campus_id, t.status AS tenant_status
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
  AND (s.user_id IS NULL OR u.is_active)
  AND (s.guardian_id IS NULL OR g.app_enabled);

-- name: TouchSession :exec
-- Throttled by the caller: only when last_seen_at is older than 5 minutes.
UPDATE sessions SET last_seen_at = now() WHERE id = @id AND revoked_at IS NULL;
