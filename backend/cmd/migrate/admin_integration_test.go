//go:build integration

package main

import (
	"crypto/rand"
	"crypto/sha256"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// create-admin makes the user once, issues a 15-minute link whose token is only stored hashed, replaces an earlier
// unused link, re-activates a deactivated user without changing their role, and audits each link.
// (Run here on the platform role; runCreateAdmin itself insists on sf_owner.)
func TestCreateAdmin(t *testing.T) {
	url := os.Getenv("TEST_PLATFORM_URL")
	if url == "" {
		t.Skip("TEST_PLATFORM_URL not set")
	}
	ctx := t.Context()
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })
	email := "ca-" + strings.ToLower(rand.Text()[:10]) + "@previa.test"
	args := adminArgs{email: email, name: "Asha", role: "billing"}

	link1, _, err := createAdmin(ctx, conn, args, "https://sumarg.example.com")
	if err != nil {
		t.Fatal(err)
	}
	token1, ok := strings.CutPrefix(link1, "https://sumarg.example.com/admin/enrol#")
	if !ok || len(token1) != 43 {
		t.Fatalf("link %q", link1)
	}
	var userID int64
	var live int
	if err := conn.QueryRow(ctx, `SELECT u.id, count(*) FILTER (WHERE e.used_at IS NULL
			AND e.expires_at BETWEEN now() + interval '14 minutes' AND now() + interval '15 minutes')
		FROM platform_users u JOIN platform_enrolments e ON e.platform_user_id = u.id
		WHERE u.email = $1 AND e.token_hash = $2 GROUP BY u.id`, email, hash(token1)).Scan(&userID, &live); err != nil {
		t.Fatalf("stored link: %v", err)
	}
	if live != 1 {
		t.Fatalf("live links for the token = %d, want 1", live)
	}

	// Deactivated, asked again with another role: re-activated, role kept, first link replaced.
	if _, err := conn.Exec(ctx, `UPDATE platform_users SET is_active = false WHERE id = $1`, userID); err != nil {
		t.Fatal(err)
	}
	args.role = "superadmin"
	link2, u2, err := createAdmin(ctx, conn, args, "https://sumarg.example.com")
	if err != nil || link2 == link1 || u2.Role != "billing" {
		t.Fatalf("second link: %q %v", link2, err)
	}
	var role string
	var active bool
	var unused, audits int
	if err := conn.QueryRow(ctx, `SELECT role, is_active,
			(SELECT count(*) FROM platform_enrolments WHERE platform_user_id = $1 AND used_at IS NULL),
			(SELECT count(*) FROM platform_audit_log WHERE entity = 'platform_user' AND entity_id = $1
			   AND action = 'enrolment.issued')
		FROM platform_users WHERE id = $1`, userID).Scan(&role, &active, &unused, &audits); err != nil {
		t.Fatal(err)
	}
	if role != "billing" || !active || unused != 1 || audits != 2 {
		t.Fatalf("after second link: role=%s active=%v unused links=%d audits=%d", role, active, unused, audits)
	}
}

func hash(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}
