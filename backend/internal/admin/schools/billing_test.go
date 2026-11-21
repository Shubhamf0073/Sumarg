package schools

import (
	"errors"
	"testing"
)

func TestCheckTransition(t *testing.T) {
	why := "unpaid 45 days"
	blank := "  "
	all := []string{"trial", "active", "past_due", "suspended", "cancelled"}
	allowed := map[[2]string]bool{
		{"trial", "active"}: true, {"trial", "cancelled"}: true,
		{"active", "past_due"}: true, {"active", "suspended"}: true, {"active", "cancelled"}: true,
		{"past_due", "active"}: true, {"past_due", "suspended"}: true, {"past_due", "cancelled"}: true,
		{"suspended", "active"}: true, {"suspended", "past_due"}: true, {"suspended", "cancelled"}: true,
	}
	for _, from := range all {
		for _, to := range all {
			err := checkTransition(from, to, &why)
			if allowed[[2]string{from, to}] != (err == nil) {
				t.Errorf("%s → %s: err %v, want allowed=%v", from, to, err, allowed[[2]string{from, to}])
			}
			if err != nil && !errors.Is(err, errTransition) {
				t.Errorf("%s → %s: err %v, want status_transition", from, to, err)
			}
		}
	}
	for _, tc := range []struct {
		from, to string
		reason   *string
		want     error
	}{
		{"active", "suspended", nil, errReason},
		{"active", "suspended", &blank, errReason},
		{"past_due", "cancelled", nil, errReason},
		{"past_due", "active", nil, nil},
		{"trial", "active", nil, nil},
		{"cancelled", "active", &why, errTransition},
	} {
		if err := checkTransition(tc.from, tc.to, tc.reason); !errors.Is(err, tc.want) {
			t.Errorf("%s → %s: err %v, want %v", tc.from, tc.to, err, tc.want)
		}
	}
}
