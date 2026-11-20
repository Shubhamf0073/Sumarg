package money

import (
	"math"
	"strings"
	"testing"
)

func TestInWords(t *testing.T) {
	tests := []struct {
		p    Paise
		want string
	}{
		{0, "Rupees Zero Only"},
		{1, "Rupees Zero and One Paise Only"},
		{Rupees(1), "Rupees One Only"},
		{Rupees(19), "Rupees Nineteen Only"},
		{Rupees(20), "Rupees Twenty Only"},
		{Rupees(99), "Rupees Ninety Nine Only"},
		{Rupees(100), "Rupees One Hundred Only"},
		{Rupees(3150), "Rupees Three Thousand One Hundred Fifty Only"},
		{338983, "Rupees Three Thousand Three Hundred Eighty Nine and Eighty Three Paise Only"},
		{Rupees(4000), "Rupees Four Thousand Only"},
		{Rupees(100000), "Rupees One Lakh Only"},
		{Rupees(1250000), "Rupees Twelve Lakh Fifty Thousand Only"},
		{Rupees(10000000), "Rupees One Crore Only"},
		{Rupees(123456789), "Rupees Twelve Crore Thirty Four Lakh Fifty Six Thousand Seven Hundred Eighty Nine Only"},
		{Rupees(1000000000), "Rupees One Hundred Crore Only"},
		{-Rupees(2500) - 50, "Minus Rupees Two Thousand Five Hundred and Fifty Paise Only"},
	}
	for _, tt := range tests {
		if got := InWords(tt.p); got != tt.want {
			t.Errorf("InWords(%d)\n got %q\nwant %q", tt.p, got, tt.want)
		}
	}
	// Extremes must not panic or produce double spaces.
	for _, p := range []Paise{math.MaxInt64, math.MinInt64} {
		if got := InWords(p); strings.Contains(got, "  ") || !strings.HasSuffix(got, " Only") {
			t.Errorf("InWords(%d) = %q", p, got)
		}
	}
}
