-- M1.2b: contract step of 00004 (approved): users.locked_until has been neither read nor written since M1.2a.
-- sessions_user lets the driver PIN login ask whether an install has had a session for the user (known device,
-- docs/05 §10); it also serves the per-user revokes. CONCURRENTLY, so sign-ins aren't blocked while it builds.

-- +goose NO TRANSACTION

-- +goose Up
-- NO TRANSACTION: each statement runs on its own, and SET is per connection, so lock_timeout is reset before the
-- index build (CONCURRENTLY waits for old transactions; a timeout there would leave an INVALID index). The index is
-- dropped first so a retry after a failed build rebuilds it instead of skipping an invalid one.
SET lock_timeout = '5s';
ALTER TABLE users DROP COLUMN IF EXISTS locked_until;
RESET lock_timeout;
DROP INDEX CONCURRENTLY IF EXISTS sessions_user;
CREATE INDEX CONCURRENTLY sessions_user ON sessions (tenant_id, user_id, install_id) WHERE user_id IS NOT NULL;

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS sessions_user;
ALTER TABLE users ADD COLUMN IF NOT EXISTS locked_until timestamptz;
