-- Parent app queries (docs/04 §parent). Run inside db.InTenantReadOnly for the guardian's session.

-- name: MyChildren :many
-- Parent app → Home: the guardian's children with today's routes. One statement for the whole screen.
SELECT st.id, st.name, st.class_name, st.section, st.to_route_id, st.from_route_id, st.to_stop_id, st.from_stop_id,
       sg.relation, sg.is_primary
FROM student_guardians sg
JOIN students st ON st.tenant_id = sg.tenant_id AND st.id = sg.student_id
WHERE sg.guardian_id = @guardian_id AND st.status = 'active'
ORDER BY st.name;

-- name: LiveForRoute :one
-- Parent app → Tracking (polled every 10 s while the screen is open). The parent may only see a bus that carries
-- one of their children; the caller checks the route is in MyChildren before running this.
SELECT vl.vehicle_id, vl.trip_id, vl.at, vl.lat, vl.lon, vl.speed_kmh, vl.heading, vl.next_stop_id, vl.eta_next_at, vl.online
FROM routes r
JOIN vehicle_live vl ON vl.tenant_id = r.tenant_id AND vl.vehicle_id = r.vehicle_id
WHERE r.id = @route_id;
