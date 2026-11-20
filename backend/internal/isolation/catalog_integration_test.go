//go:build integration

// Catalog check (docs/03 §2, docs/07 S1, CLAUDE.md invariant 4): reads pg_catalog as sf_app on the migrated test
// database and fails on any table, grant or function that would let one school's data reach another. It checks the
// schema as migrated, so a new table that forgets RLS, a stray GRANT DELETE or a new SECURITY DEFINER function
// fails here even if no test touches that table yet.
package isolation_test

import (
	"os"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/config"
	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/db"
)

// Tables with tenant_id that need no UNIQUE (tenant_id, id): nothing may reference them by a composite FK.
var noCompositeKey = map[string]string{
	"audit_log":          "append-only log; never a FK target",
	"platform_audit_log": "platform-only (no sf_app grant); append-only",
	"agent_commissions":  "platform-only (no sf_app grant); never a FK target",
}

// The only SECURITY DEFINER functions allowed (rules/sql.md; adding one needs an ADR line in docs/02).
var securityDefiners = []string{"drop_old_position_partitions", "ensure_position_partitions", "tenant_by_code"}

// The SECURITY DEFINER functions sf_app may execute; the rest are sf_platform-only.
var appExecutable = []string{"tenant_by_code"}

// Tables sf_app may reach that have no tenant_id column, and the policy column that scopes them instead.
var scopedWithoutTenantID = map[string]string{"tenants": "(id = app_tenant())"}

const tenantPolicy = "(tenant_id = app_tenant())"

func appConn(t *testing.T) func(sql string, args ...any) [][]string {
	t.Helper()
	url := os.Getenv("TEST_APP_URL")
	if url == "" {
		t.Skip("TEST_APP_URL not set")
	}
	pool, err := db.OpenApp(t.Context(), config.Secret(url), db.Options{Name: "test-catalog", MaxConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return func(sql string, args ...any) [][]string {
		t.Helper()
		rows, err := pool.Query(t.Context(), sql, args...)
		if err != nil {
			t.Fatalf("catalog query: %v", err)
		}
		out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) ([]string, error) {
			vals, err := r.Values()
			s := make([]string, len(vals))
			for i, v := range vals {
				if v != nil {
					s[i] = toString(v)
				}
			}
			return s, err
		})
		if err != nil {
			t.Fatalf("catalog rows: %v", err)
		}
		return out
	}
}

func toString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	default:
		return "?"
	}
}

// tenantTables: base and partitioned-parent tables in public with a tenant_id column.
const tenantTables = `
SELECT c.oid, c.relname FROM pg_class c
JOIN pg_attribute a ON a.attrelid = c.oid AND a.attname = 'tenant_id' AND NOT a.attisdropped
WHERE c.relnamespace = 'public'::regnamespace AND c.relkind IN ('r','p') AND NOT c.relispartition`

func TestAppRoleIsPlain(t *testing.T) {
	q := appConn(t)
	for _, r := range q(`SELECT rolname, rolsuper::text, rolbypassrls::text,
		(SELECT count(*) FROM pg_class WHERE relowner = r.oid)::text
		FROM pg_roles r WHERE rolname = current_user`) {
		if r[1] != "false" || r[2] != "false" || r[3] != "0" {
			t.Errorf("%s: superuser=%s bypassrls=%s owns %s relations — RLS would not bind it", r[0], r[1], r[2], r[3])
		}
	}
}

func TestEveryTenantTableForcesRLS(t *testing.T) {
	q := appConn(t)
	rows := q(`SELECT t.relname, c.relrowsecurity::text, c.relforcerowsecurity::text
		FROM (` + tenantTables + `) t JOIN pg_class c ON c.oid = t.oid ORDER BY 1`)
	if len(rows) < 20 {
		t.Fatalf("found only %d tenant tables — wrong database?", len(rows))
	}
	for _, r := range rows {
		if r[1] != "true" || r[2] != "true" {
			t.Errorf("%s: RLS enabled=%s forced=%s, want both (schema.sql RLS block)", r[0], r[1], r[2])
		}
	}
	for table := range scopedWithoutTenantID {
		r := q(`SELECT relrowsecurity::text, relforcerowsecurity::text FROM pg_class
			WHERE oid = ('public.' || $1)::regclass`, table)
		if len(r) != 1 || r[0][0] != "true" || r[0][1] != "true" {
			t.Errorf("%s: RLS not enabled and forced", table)
		}
	}
}

// Every policy must scope by app_tenant(): permissive policies are OR-ed, so one wide policy opens the table.
func TestEveryPolicyScopesByTenant(t *testing.T) {
	q := appConn(t)
	for _, r := range q(`SELECT t.relname FROM (` + tenantTables + `) t
		WHERE NOT EXISTS (SELECT 1 FROM pg_policy p WHERE p.polrelid = t.oid) ORDER BY 1`) {
		t.Errorf("%s: no RLS policy (forced RLS with no policy hides rows but is a forgotten table)", r[0])
	}
	for _, r := range q(`SELECT tablename, policyname, coalesce(qual, ''), coalesce(with_check, '')
		FROM pg_policies WHERE schemaname = 'public' ORDER BY 1, 2`) {
		table, qual, check := r[0], r[2], r[3]
		want := tenantPolicy
		if w, ok := scopedWithoutTenantID[table]; ok {
			want = w
			check = qual // SELECT-only for sf_app; no WITH CHECK needed
		}
		if qual != want || check != want {
			t.Errorf("%s.%s: USING %q WITH CHECK %q, want %q on both", table, r[1], qual, r[3], want)
		}
	}
	// The USING text alone isn't enough: a policy FOR SELECT only, TO another role, or RESTRICTIVE (with no
	// permissive one) leaves sf_app's other commands with a different — or no — scope. One shape for all.
	for _, r := range q(`SELECT c.relname, p.polname, p.polpermissive::text, p.polcmd::text, p.polroles::text
		FROM pg_policy p JOIN pg_class c ON c.oid = p.polrelid
		WHERE c.relnamespace = 'public'::regnamespace
		  AND NOT (p.polpermissive AND p.polcmd = '*' AND p.polroles = '{0}'::oid[]) ORDER BY 1, 2`) {
		t.Errorf("%s.%s: permissive=%s cmd=%s roles=%s, want a permissive FOR ALL TO PUBLIC policy",
			r[0], r[1], r[2], r[3], r[4])
	}
}

// Views run as their owner unless security_invoker is set, so they would skip RLS; matviews can't have RLS at all.
func TestAppReachesNoRLSBypassingView(t *testing.T) {
	q := appConn(t)
	for _, r := range q(`SELECT c.relname, c.relkind::text FROM pg_class c
		WHERE c.relnamespace = 'public'::regnamespace AND c.relkind IN ('v','m')
		  AND (c.relkind = 'm' OR NOT coalesce('security_invoker=true' = ANY (c.reloptions)
		                                   OR 'security_invoker=on' = ANY (c.reloptions), false))
		  AND (has_table_privilege(c.oid, 'SELECT,INSERT,UPDATE,DELETE')
		       OR has_any_column_privilege(c.oid, 'SELECT,INSERT,UPDATE'))
		ORDER BY 1`) {
		t.Errorf("%s (relkind %s): sf_app can reach it and it bypasses RLS — use a view WITH (security_invoker)",
			r[0], r[1])
	}
}

// sf_app never deletes or truncates (invariant 4), and reaches positions only through the partitioned parent.
func TestAppHasNoDeleteAndNoPartitionGrants(t *testing.T) {
	q := appConn(t)
	for _, r := range q(`SELECT c.relname,
			has_table_privilege(c.oid, 'DELETE')::text, has_table_privilege(c.oid, 'TRUNCATE')::text
		FROM pg_class c WHERE c.relnamespace = 'public'::regnamespace AND c.relkind IN ('r','p')
		  AND (has_table_privilege(c.oid, 'DELETE') OR has_table_privilege(c.oid, 'TRUNCATE')) ORDER BY 1`) {
		t.Errorf("%s: sf_app has DELETE=%s TRUNCATE=%s; it must have neither", r[0], r[1], r[2])
	}
	parts := q(`SELECT c.relname,
			(has_table_privilege(c.oid, 'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
			 OR has_any_column_privilege(c.oid, 'SELECT,INSERT,UPDATE,REFERENCES'))::text
		FROM pg_class c WHERE c.relnamespace = 'public'::regnamespace AND c.relkind = 'r' AND c.relispartition
		ORDER BY 1`)
	if len(parts) == 0 {
		t.Fatal("no positions partitions found — migration should have created some")
	}
	for _, r := range parts {
		if r[1] != "false" {
			t.Errorf("%s: sf_app has a grant on a partition (partitions skip the parent's RLS)", r[0])
		}
	}
}

// Anything sf_app can read without a tenant_id column must be on the reviewed list.
func TestAppReachesNoUnscopedTable(t *testing.T) {
	q := appConn(t)
	for _, r := range q(`SELECT c.relname FROM pg_class c
		WHERE c.relnamespace = 'public'::regnamespace AND c.relkind IN ('r','p','v','m','f')
		  AND NOT EXISTS (SELECT 1 FROM pg_attribute a WHERE a.attrelid = c.oid AND a.attname = 'tenant_id'
		                  AND NOT a.attisdropped)
		  AND (has_table_privilege(c.oid, 'SELECT,INSERT,UPDATE,DELETE')
		       OR has_any_column_privilege(c.oid, 'SELECT,INSERT,UPDATE'))
		ORDER BY 1`) {
		if _, ok := scopedWithoutTenantID[r[0]]; !ok {
			t.Errorf("%s: sf_app can reach it and it has no tenant_id", r[0])
		}
	}
}

// Invariant 4: UNIQUE (tenant_id, id) so children can use composite FKs, and every FK between two tenant tables
// carries tenant_id (a boarding can't point at another school's trip).
func TestCompositeKeys(t *testing.T) {
	q := appConn(t)
	for _, r := range q(`SELECT t.relname FROM (` + tenantTables + `) t
		WHERE EXISTS (SELECT 1 FROM pg_attribute a WHERE a.attrelid = t.oid AND a.attname = 'id')
		  AND NOT EXISTS (SELECT 1 FROM pg_constraint k WHERE k.conrelid = t.oid AND k.contype IN ('u','p')
		      AND (SELECT array_agg(a.attname::text ORDER BY a.attname) FROM pg_attribute a
		           WHERE a.attrelid = t.oid AND a.attnum = ANY (k.conkey)) = ARRAY['id','tenant_id'])
		ORDER BY 1`) {
		if _, ok := noCompositeKey[r[0]]; !ok {
			t.Errorf("%s: no UNIQUE (tenant_id, id)", r[0])
		}
	}
	for _, r := range q(`SELECT k.conrelid::regclass::text, k.conname, k.confrelid::regclass::text
		FROM pg_constraint k
		WHERE k.contype = 'f' AND k.connamespace = 'public'::regnamespace
		  AND EXISTS (SELECT 1 FROM pg_attribute a WHERE a.attrelid = k.conrelid AND a.attname = 'tenant_id')
		  AND EXISTS (SELECT 1 FROM pg_attribute a WHERE a.attrelid = k.confrelid AND a.attname = 'tenant_id')
		  AND (NOT EXISTS (SELECT 1 FROM pg_attribute a
		                   WHERE a.attrelid = k.conrelid AND a.attnum = ANY (k.conkey) AND a.attname = 'tenant_id')
		       OR NOT EXISTS (SELECT 1 FROM pg_attribute a
		                   WHERE a.attrelid = k.confrelid AND a.attnum = ANY (k.confkey) AND a.attname = 'tenant_id'))
		ORDER BY 1, 2`) {
		t.Errorf("%s.%s → %s: FK between tenant tables without tenant_id on both sides", r[0], r[1], r[2])
	}
	// The exemption above holds only while nothing references those tables.
	for _, r := range q(`SELECT k.conrelid::regclass::text, k.conname, k.confrelid::regclass::text
		FROM pg_constraint k WHERE k.contype = 'f' AND k.connamespace = 'public'::regnamespace ORDER BY 1, 2`) {
		if _, ok := noCompositeKey[r[2]]; ok {
			t.Errorf("%s.%s → %s: FK to a table exempt from UNIQUE (tenant_id, id); add the key and a composite FK",
				r[0], r[1], r[2])
		}
	}
}

func TestOnlyApprovedSecurityDefiners(t *testing.T) {
	q := appConn(t)
	var got []string
	for _, r := range q(`SELECT proname FROM pg_proc WHERE prosecdef
		AND pronamespace NOT IN ('pg_catalog'::regnamespace, 'information_schema'::regnamespace) ORDER BY 1`) {
		got = append(got, r[0])
	}
	if !slices.Equal(got, securityDefiners) {
		t.Errorf("SECURITY DEFINER functions = %v, want %v (a new one needs the human's approval + docs/02 ADR)",
			got, securityDefiners)
	}
	// Each pins search_path (else a caller's schema can shadow what it calls) and sf_app may run only the
	// pre-tenant lookup — the partition functions create and drop tables for every school.
	for _, r := range q(`SELECT proname,
			EXISTS (SELECT 1 FROM unnest(proconfig) c WHERE c LIKE 'search_path=%')::text,
			has_function_privilege(oid, 'EXECUTE')::text
		FROM pg_proc WHERE prosecdef
		  AND pronamespace NOT IN ('pg_catalog'::regnamespace, 'information_schema'::regnamespace) ORDER BY 1`) {
		if r[1] != "true" {
			t.Errorf("%s: SECURITY DEFINER without SET search_path", r[0])
		}
		if want := slices.Contains(appExecutable, r[0]); (r[2] == "true") != want {
			t.Errorf("%s: sf_app EXECUTE=%s, want %t", r[0], r[2], want)
		}
	}
}
