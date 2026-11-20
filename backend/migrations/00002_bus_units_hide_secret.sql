-- sf_app must never read the unit's MQTT secret hash (only ingest, as sf_platform, authenticates units).
-- Replace the table-wide SELECT on bus_units with a column list that leaves out secret_hash.
-- +goose Up
REVOKE SELECT ON bus_units FROM sf_app;
GRANT SELECT (id, uid, hw_rev, imei, iccid, fw_version, fw_channel, tenant_id, vehicle_id, status,
              installed_at, last_seen_at, last_status, created_at) ON bus_units TO sf_app;

-- +goose Down
REVOKE SELECT ON bus_units FROM sf_app;
GRANT SELECT ON bus_units TO sf_app;
