-- M1.4e: optimistic lock for /admin package edits (docs/schema.sql). Adds a column with a constant default: no table
-- rewrite, existing rows read version 1.

-- +goose Up
ALTER TABLE subscriptions ADD COLUMN version integer NOT NULL DEFAULT 1;

-- +goose Down
ALTER TABLE subscriptions DROP COLUMN version;
