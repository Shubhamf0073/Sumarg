-- Platform queries: run as sf_platform (BYPASSRLS) inside platformdb.InPlatform. Every query that touches tenant
-- tables names the tenant explicitly (CLAUDE.md invariant 2).

-- name: UnitForAuth :one
-- Ingest → MQTT CONNECT: the unit's secret hash and allocation. The client id is the unit uid (docs/13 §1).
SELECT id, uid, secret_hash, tenant_id, vehicle_id, status FROM bus_units WHERE uid = @uid;

-- name: ListUnits :many
-- Admin → Units: stock and installed units with their school and bus.
SELECT u.id, u.uid, u.hw_rev, u.fw_version, u.fw_channel, u.status, u.last_seen_at, u.tenant_id,
       t.display_name AS tenant_name, v.reg_no
FROM bus_units u
LEFT JOIN tenants t ON t.id = u.tenant_id
LEFT JOIN vehicles v ON v.tenant_id = u.tenant_id AND v.id = u.vehicle_id
ORDER BY u.uid
LIMIT @row_limit;
