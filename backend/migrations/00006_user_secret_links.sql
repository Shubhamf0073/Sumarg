-- M1.4b: one-time "set your password" links for portal staff (docs/schema.sql, docs/05 §10). New table only: no
-- existing rows or constraints change.

-- +goose Up
CREATE TABLE user_secret_links (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id    bigint  NOT NULL REFERENCES tenants(id),
    user_id      bigint  NOT NULL,
    token_hash   bytea   NOT NULL UNIQUE,
    expires_at   timestamptz NOT NULL,
    used_at      timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, id),
    FOREIGN KEY (tenant_id, user_id) REFERENCES users (tenant_id, id)
);
CREATE INDEX user_secret_links_live ON user_secret_links (tenant_id, user_id) WHERE used_at IS NULL;

ALTER TABLE user_secret_links ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_secret_links FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON user_secret_links
    USING (tenant_id = app_tenant()) WITH CHECK (tenant_id = app_tenant());

-- Same grants, in the same order, as schema.sql (platform first), so schema-diff matches.
GRANT SELECT, INSERT, UPDATE, DELETE ON user_secret_links TO sf_platform;
GRANT USAGE, SELECT ON SEQUENCE user_secret_links_id_seq TO sf_platform;
GRANT SELECT, INSERT, UPDATE ON user_secret_links TO sf_app;
GRANT USAGE, SELECT ON SEQUENCE user_secret_links_id_seq TO sf_app;

-- +goose Down
DROP TABLE user_secret_links;
