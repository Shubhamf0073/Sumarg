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
-- Parent app → Tracking (polled every 10 s while the screen is open). Scoped in SQL (invariant 3): the route must
-- carry one of this guardian's active children, and while tenant_settings.parent_sees_live_only_on_trip is on the
-- bus's current trip must be started. No row (→ 404) otherwise.
SELECT vl.vehicle_id, vl.trip_id, vl.at, vl.lat, vl.lon, vl.speed_kmh, vl.heading, vl.next_stop_id, vl.eta_next_at, vl.online
FROM routes r
JOIN vehicle_live vl ON vl.tenant_id = r.tenant_id AND vl.vehicle_id = r.vehicle_id
JOIN tenant_settings ts ON ts.tenant_id = r.tenant_id
WHERE r.id = @route_id
  AND EXISTS (SELECT 1
              FROM student_guardians sg
              JOIN students st ON st.tenant_id = sg.tenant_id AND st.id = sg.student_id
              WHERE sg.tenant_id = r.tenant_id AND sg.guardian_id = @guardian_id AND st.status = 'active'
                AND r.id IN (st.to_route_id, st.from_route_id))
  AND (NOT ts.parent_sees_live_only_on_trip
       OR EXISTS (SELECT 1 FROM trips tr
                  WHERE tr.tenant_id = vl.tenant_id AND tr.id = vl.trip_id AND tr.status = 'started'));
