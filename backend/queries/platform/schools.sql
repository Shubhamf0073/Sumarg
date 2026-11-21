-- /admin schools: the onboarding wizard and board (docs/04 "Admin", onboarding/01 "Setup in /admin"). Runs as
-- sf_platform (BYPASSRLS): every query names the tenant.

-- name: AgentIsActive :one
-- Wizard → package: the agent linked to the school must exist and be active.
SELECT is_active FROM agents WHERE id = @id;

-- name: CreateTenant :one
-- Wizard → save: the school. Code clash → unique violation (409 in Go).
INSERT INTO tenants (code, legal_name, display_name, short_name, board, gstin, address, city, state_code,
                     contact_name, contact_mobile, contact_email, go_live_on, price_per_student_paise,
                     price_includes_gst, agent_id, notes, created_by)
VALUES (@code, @legal_name, @display_name, @short_name, sqlc.narg(board), sqlc.narg(gstin), @address, @city,
        @state_code, @contact_name, @contact_mobile, sqlc.narg(contact_email), sqlc.narg(go_live_on),
        @price_per_student_paise, @price_includes_gst, sqlc.narg(agent_id), @notes, @created_by)
RETURNING id, onboarding_stage;

-- name: CreateTenantSettings :exec
-- Wizard → save: the school's settings (the rest keep their defaults).
INSERT INTO tenant_settings (tenant_id, approach_minutes, delay_alert_minutes, speed_limit_kmph, photo_on_tap,
                             photo_retention_days, position_retention_days, sms_fallback)
VALUES (@tenant_id, @approach_minutes, @delay_alert_minutes, @speed_limit_kmph, @photo_on_tap,
        @photo_retention_days, @position_retention_days, @sms_fallback);

-- name: CreateCampus :one
-- Wizard → save: one campus. Duplicate code within the school → unique violation (409).
INSERT INTO campuses (tenant_id, code, name, address, lat, lon, geofence_m)
VALUES (@tenant_id, @code, @name, @address, @lat, @lon, @geofence_m)
RETURNING id;

-- name: CreatePortalUser :one
-- Wizard → save: the owner. password_hash is the Argon2id hash of a random secret nobody knows: the account can't
-- sign in until the one-time link sets a real password.
INSERT INTO users (tenant_id, campus_id, name, mobile, email, role, password_hash, must_change_secret)
VALUES (@tenant_id, NULL, @name, @mobile, sqlc.narg(email), @role, @password_hash, true)
RETURNING id;

-- name: SpendUserSecretLinks :exec
-- "New password link": earlier unused links for the user stop working (same tx as CreateSecretLink).
UPDATE user_secret_links SET used_at = now()
WHERE tenant_id = @tenant_id AND user_id = @user_id AND used_at IS NULL;

-- name: CreateSecretLink :one
-- Wizard → save, and "new password link": a one-time link for an active portal user of this school. No row → the
-- user isn't an active owner/admin/transport manager of the school.
INSERT INTO user_secret_links (tenant_id, user_id, token_hash, expires_at)
SELECT users.tenant_id, users.id, @token_hash::bytea, now() + make_interval(hours => @lifetime_hours::int)
FROM users
WHERE users.tenant_id = @tenant_id AND users.id = @user_id AND users.is_active
  AND users.role IN ('owner','admin','transport_manager')
RETURNING id, expires_at;

-- name: CreateSubscription :one
-- Wizard → save: the first 12-month package (docs/05 §11). period_end (start + 12 months − 1 day) is computed in Go.
INSERT INTO subscriptions (tenant_id, period_start, period_end, students_committed, price_per_student_paise,
                           created_by)
VALUES (@tenant_id, @period_start, @period_end, @students_committed, @price_per_student_paise, @created_by)
RETURNING id;

-- name: ListSchools :many
-- Onboarding board: schools by id after a cursor (limit+1 pages); the portal groups them by stage.
SELECT id, code, display_name, status, onboarding_stage, go_live_on, is_demo, created_at
FROM tenants
WHERE id > @after
ORDER BY id
LIMIT @lim;

-- name: GetSchool :one
-- School page: the school.
SELECT id, code, legal_name, display_name, short_name, board, gstin, address, city, state_code, contact_name,
       contact_mobile, contact_email, status, onboarding_stage, go_live_on, is_demo, price_per_student_paise,
       price_includes_gst, agent_id, suspend_reason, created_at
FROM tenants
WHERE id = @id;

-- name: GetSchoolPackage :one
-- School page: the current package (no row → none yet).
SELECT id, period_start, period_end, students_committed, price_per_student_paise, version FROM subscriptions
WHERE tenant_id = @tenant_id AND status = 'active'
ORDER BY period_start DESC
LIMIT 1;

-- name: ListSchoolCampuses :many
-- School page: campuses.
SELECT id, code, name, lat, lon, geofence_m, is_active FROM campuses WHERE tenant_id = @tenant_id ORDER BY code;

-- name: ListSchoolPortalUsers :many
-- School page: owner/admin logins and whether they have set a password yet.
SELECT id, name, mobile, role, must_change_secret, is_active FROM users
WHERE tenant_id = @tenant_id AND role IN ('owner','admin')
ORDER BY id;

-- name: SetOnboardingStage :one
-- Onboarding board → move a school, forwards or back (leaving `live` is refused in Go). Conditional on the stage the
-- admin saw (0 rows → 409). go_live_on: the date given, else the one already set. Moving to `live` stores at most
-- today in IST, in this same statement — going live means live now, and that date opens the 14-day OTP launch
-- window (docs/05 §10, OTPIssueCounts): with neither date, or a future one, it is today; a past one is kept.
UPDATE tenants SET
    onboarding_stage = @stage,
    go_live_on = CASE
        WHEN @stage::text = 'live'
            THEN LEAST(COALESCE(sqlc.narg(go_live_on)::date, go_live_on, (now() AT TIME ZONE 'Asia/Kolkata')::date),
                       (now() AT TIME ZONE 'Asia/Kolkata')::date)
        ELSE COALESCE(sqlc.narg(go_live_on)::date, go_live_on)
    END,
    updated_at = now()
WHERE id = @id AND onboarding_stage = @from_stage
RETURNING onboarding_stage, go_live_on;

-- name: InsertPlatformAuditTenant :exec
-- A Previa staff member's change to a school, with what changed.
INSERT INTO platform_audit_log (platform_user_id, actor_name, action, tenant_id, entity, entity_id, summary, changes,
                                ip, request_id)
VALUES (@platform_user_id, @actor_name, @action, @tenant_id, @entity, @entity_id, @summary, sqlc.narg(changes),
        sqlc.narg(ip), sqlc.narg(request_id));

-- name: GetSchoolLogin :one
-- School page → cancel link / deactivate: the owner/admin login of this school (no row → 404).
SELECT id, is_active FROM users
WHERE tenant_id = @tenant_id AND id = @user_id AND role IN ('owner','admin');

-- name: CancelUserSecretLinks :execrows
-- School page → "cancel link": the user's unused links stop working, and no new one is issued (a link sent to the
-- wrong person). Spent links are marked used, so set-password answers 410 like any used link.
UPDATE user_secret_links SET used_at = now()
WHERE tenant_id = @tenant_id AND user_id = @user_id AND used_at IS NULL AND expires_at > now();

-- name: DeactivateSchoolLogin :execrows
-- School page → "deactivate": an active owner/admin login of this school can no longer sign in (0 rows → 404/409).
UPDATE users SET is_active = false, version = version + 1
WHERE tenant_id = @tenant_id AND id = @user_id AND role IN ('owner','admin') AND is_active;

-- name: RevokeUserSessions :execrows
-- School page → "deactivate": end every live session of the user, same transaction.
UPDATE sessions SET revoked_at = now()
WHERE tenant_id = @tenant_id AND user_id = @user_id AND revoked_at IS NULL;

-- name: ReactivateSchoolLogin :execrows
-- School page → "reactivate with a new link": an inactive owner/admin login of this school comes back with an
-- unusable password (whoever set the old one can't use it again) until the new one-time link is used (0 rows →
-- 404/409).
UPDATE users SET is_active = true, password_hash = @password_hash, must_change_secret = true, version = version + 1
WHERE tenant_id = @tenant_id AND id = @user_id AND role IN ('owner','admin') AND NOT is_active;

-- name: SetSchoolStatus :one
-- School page → billing status (docs/05 §11). Conditional on the status the billing user saw (0 rows → 404/409); the
-- allowed transitions are checked in Go. Suspending stamps suspended_at + reason, leaving `suspended` clears them;
-- cancelling stamps cancelled_at (cancelled is final). Sessions read the status on every request: suspended →
-- read-only, cancelled → 423, from the next request on.
UPDATE tenants SET
    status = @status,
    suspended_at = CASE WHEN @status::text = 'suspended' THEN now() END,
    suspend_reason = CASE WHEN @status::text = 'suspended' THEN sqlc.narg(reason)::text END,
    cancelled_at = CASE WHEN @status::text = 'cancelled' THEN now() ELSE cancelled_at END,
    updated_at = now()
WHERE id = @id AND status = @from_status
RETURNING status;

-- name: GetSchoolPackageForEdit :one
-- School page → edit package: the active package of this school as it is now, locked for the edit (no row → 404).
SELECT id, period_start, period_end, students_committed, price_per_student_paise, version FROM subscriptions
WHERE tenant_id = @tenant_id AND id = @id AND status = 'active'
FOR UPDATE;

-- name: UpdateSchoolPackage :one
-- School page → edit package: committed students, price and start (period_end computed in Go), conditional on the
-- version the billing user saw (0 rows → 409).
UPDATE subscriptions SET
    students_committed = @students_committed,
    price_per_student_paise = @price_per_student_paise,
    period_start = @period_start,
    period_end = @period_end,
    version = version + 1
WHERE tenant_id = @tenant_id AND id = @id AND status = 'active' AND version = @version
RETURNING version;

-- name: SetSchoolPrice :exec
-- School page → edit package: the school's price follows its active package's (the next package starts from it).
UPDATE tenants SET price_per_student_paise = @price_per_student_paise, updated_at = now() WHERE id = @id;
