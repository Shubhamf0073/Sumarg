-- Tenant table template (docs/02 A19, CLAUDE.md invariant 4). Replace <table>; delete campus_id lines if the
-- table isn't campus-scoped. Keep every RLS/grant line.

-- +goose Up
CREATE TABLE <table> (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id   bigint  NOT NULL REFERENCES tenants(id),
    campus_id   bigint  NOT NULL,
    -- columns: money bigint *_paise CHECK (… >= 0); enums text + CHECK; timestamptz; no float
    version     integer NOT NULL DEFAULT 1,          -- only if humans edit it (optimistic lock)
    created_at  timestamptz NOT NULL DEFAULT now(),
    created_by  bigint  NOT NULL,
    UNIQUE (tenant_id, id),
    FOREIGN KEY (tenant_id, campus_id)  REFERENCES campuses (tenant_id, id),
    FOREIGN KEY (tenant_id, created_by) REFERENCES users (tenant_id, id)
);
CREATE INDEX <table>_tenant_campus_created ON <table> (tenant_id, campus_id, created_at DESC);

ALTER TABLE <table> ENABLE ROW LEVEL SECURITY;
ALTER TABLE <table> FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON <table>
    USING (tenant_id = app_tenant()) WITH CHECK (tenant_id = app_tenant());

GRANT SELECT, INSERT, UPDATE ON <table> TO sf_app;        -- never DELETE; append-only: SELECT, INSERT
GRANT SELECT, INSERT, UPDATE, DELETE ON <table> TO sf_platform;
-- Identity columns need no sequence grant (checked on PG 16/17: INSERT on the table is enough).

-- +goose Down
DROP TABLE <table>;
