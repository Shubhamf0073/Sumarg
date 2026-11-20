-- M1.2a: sign-in backoff moves to the API process, per school code + mobile + client address, so an attacker's
-- failures no longer close the real user's login. The database keeps a per-user cap over every address: at most 50
-- failures in a 24 h window. Expand only: locked_until is no longer written or read; dropping it is a later contract
-- step (docs/schema.sql). Adding nullable / constant-default columns is a metadata-only change (no rewrite).

-- +goose Up
ALTER TABLE users
    ADD COLUMN fail_window_start timestamptz,
    ADD COLUMN fail_window_count smallint NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE users
    DROP COLUMN fail_window_count,
    DROP COLUMN fail_window_start;
