package httpx

import (
	"net/netip"
	"testing"
)

// X-Forwarded-For is believed only from the trusted Nginx hop, and then only its last entry (what Nginx wrote).
func TestClientIP(t *testing.T) {
	nginx := []netip.Prefix{netip.MustParsePrefix("172.30.231.1/32")}
	for name, tc := range map[string]struct {
		remote, xff string
		trusted     []netip.Prefix
		want        string
	}{
		"direct, no header":             {"198.51.100.7:5000", "", nginx, "198.51.100.7"},
		"direct, spoofed header":        {"198.51.100.7:5000", "203.0.113.9", nginx, "198.51.100.7"},
		"via nginx":                     {"172.30.231.1:41000", "203.0.113.9", nginx, "203.0.113.9"},
		"via nginx, client-sent prefix": {"172.30.231.1:41000", "10.0.0.1, 203.0.113.9", nginx, "203.0.113.9"},
		"via nginx, no header":          {"172.30.231.1:41000", "", nginx, "172.30.231.1"},
		"via nginx, garbage header":     {"172.30.231.1:41000", "nonsense", nginx, "172.30.231.1"},
		"nothing trusted (local)":       {"127.0.0.1:41000", "203.0.113.9", nil, "127.0.0.1"},
		"ipv4-mapped peer":              {"[::ffff:172.30.231.1]:41000", "2001:db8::5", nginx, "2001:db8::5"},
		"unparseable peer":              {"pipe", "203.0.113.9", nginx, ""},
	} {
		t.Run(name, func(t *testing.T) {
			got := ClientIP(tc.remote, tc.xff, tc.trusted)
			s := ""
			if got != nil {
				s = got.String()
			}
			if s != tc.want {
				t.Fatalf("ClientIP = %q, want %q", s, tc.want)
			}
		})
	}
}
