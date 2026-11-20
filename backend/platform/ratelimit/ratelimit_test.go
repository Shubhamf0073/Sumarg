package ratelimit

import (
	"net/netip"
	"testing"
	"time"

	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/clock"
)

func TestAllowWindow(t *testing.T) {
	c := clock.NewFake(time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC))
	l := New(c, 3, time.Minute)
	for i := range 3 {
		if !l.Allow("a") {
			t.Fatalf("hit %d refused", i+1)
		}
	}
	if l.Allow("a") {
		t.Fatal("4th hit allowed")
	}
	if !l.Allow("b") {
		t.Fatal("other key refused")
	}
	c.Advance(59 * time.Second)
	if l.Allow("a") {
		t.Fatal("allowed before the window ended")
	}
	c.Advance(time.Second)
	if !l.Allow("a") {
		t.Fatal("refused in the next window")
	}
}

func TestFailOpenWhenFull(t *testing.T) {
	c := clock.NewFake(time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC))
	l := New(c, 1, time.Minute)
	l.maxKeys = 2
	l.Allow("a")
	l.Allow("b")
	for range 5 {
		if !l.Allow("c") {
			t.Fatal("new key refused when the table is full (must fail open)")
		}
	}
	if l.Allow("a") {
		t.Fatal("known key no longer limited")
	}
}

func TestIPKey(t *testing.T) {
	p := func(s string) *netip.Addr { a := netip.MustParseAddr(s); return &a }
	for in, want := range map[*netip.Addr]string{
		nil:                         "-",
		p("203.0.113.7"):            "203.0.113.7",
		p("::ffff:203.0.113.7"):     "203.0.113.7",
		p("2001:db8:1:2:3:4:5:6"):   "2001:db8:1:2::/64",
		p("2001:db8:1:2:ffff::abc"): "2001:db8:1:2::/64",
	} {
		if got := IPKey(in); got != want {
			t.Errorf("IPKey(%v) = %q, want %q", in, got, want)
		}
	}
}
