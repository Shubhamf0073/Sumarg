package schools

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/httpx"
)

// Constraint violations become 4xx without the constraint name or row values; anything else stays a 500 cause.
func TestDBError(t *testing.T) {
	tests := []struct {
		name, code, constraint string
		status                 int
		apiCode                string
	}{
		{"code taken", "23505", "tenants_code_key", http.StatusConflict, "code_taken"},
		{"campus duplicate", "23505", "campuses_tenant_id_code_key", http.StatusUnprocessableEntity, "campus_code_duplicate"},
		{"owner mobile taken", "23505", "users_tenant_id_mobile_key", http.StatusConflict, "duplicate"},
		{"agent gone", "23503", "tenants_agent_id_fkey", http.StatusUnprocessableEntity, "agent_invalid"},
		{"owner mobile check", "23514", "users_mobile_check", http.StatusUnprocessableEntity, "mobile_invalid"},
		{"contact mobile check", "23514", "tenants_contact_mobile_check", http.StatusUnprocessableEntity, "mobile_invalid"},
		{"other check", "23514", "tenants_gstin_check", http.StatusUnprocessableEntity, "value_invalid"},
		{"bad date", "22008", "", http.StatusUnprocessableEntity, "date_invalid"},
		{"other fk", "23503", "campuses_tenant_id_fkey", 0, ""},
		{"not null", "23502", "", 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg := &pgconn.PgError{Code: tt.code, ConstraintName: tt.constraint, Detail: "Key (mobile)=(9876543210)"}
			err := dbError(fmt.Errorf("wrapped: %w", pg))
			var he *httpx.Error
			if tt.status == 0 {
				if errors.As(err, &he) {
					t.Fatalf("got client error %v, want a 500 cause", he)
				}
				if strings.Contains(err.Error(), "9876543210") {
					t.Fatalf("detail leaked: %v", err)
				}
				return
			}
			if !errors.As(err, &he) || he.Status != tt.status || he.Code != tt.apiCode {
				t.Fatalf("dbError = %#v, want %d %s", err, tt.status, tt.apiCode)
			}
			if (tt.constraint != "" && strings.Contains(he.Message, tt.constraint)) || strings.Contains(he.Message, "9876543210") {
				t.Fatalf("message leaks: %q", he.Message)
			}
		})
	}
	plain := errors.New("boom")
	if dbError(plain) != plain {
		t.Fatal("non-pg error changed")
	}
}
