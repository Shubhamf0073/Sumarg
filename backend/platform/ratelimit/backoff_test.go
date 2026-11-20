package ratelimit

import (
	"testing"
	"time"

	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/clock"
)

var testPolicy = BackoffPolicy{Free: 5, Base: 30 * time.Second, Max: 15 * time.Minute, PerDay: 10, Day: 24 * time.Hour}

func newTestBackoff() (*Backoff, *clock.Fake) {
	c := clock.NewFake(time.Date(2026, 11, 21, 8, 0, 0, 0, time.UTC))
	return NewBackoff(c, testPolicy), c
}

func TestBackoffDoublesToMax(t *testing.T) {
	b, c := newTestBackoff()
	for i := range 5 {
		if !b.Claim("k") {
			t.Fatalf("try %d refused", i+1)
		}
	}
	// The 5th failure opens a 30 s window, then 60 s, 120 s, ... capped at 15 min.
	for _, wait := range []time.Duration{30 * time.Second, time.Minute, 2 * time.Minute, 4 * time.Minute} {
		c.Advance(wait - time.Second)
		if b.Claim("k") {
			t.Fatalf("allowed 1 s before a %s window ended", wait)
		}
		c.Advance(time.Second)
		if !b.Claim("k") {
			t.Fatalf("refused after a %s window", wait)
		}
	}
	if b.Claim("other") != true {
		t.Fatal("another key was slowed down")
	}
}

func TestBackoffCapsAtMax(t *testing.T) {
	b, c := newTestBackoff()
	b.p.PerDay = 1000
	for range 40 {
		c.Advance(time.Hour)
		b.Claim("k")
	}
	c.Advance(15*time.Minute - time.Second)
	if b.Claim("k") {
		t.Fatal("wait exceeded 15 min cap check: allowed early")
	}
	c.Advance(time.Second)
	if !b.Claim("k") {
		t.Fatal("wait longer than 15 min")
	}
}

func TestBackoffPerDayCap(t *testing.T) {
	b, c := newTestBackoff()
	n := 0
	for range 80 { // 20 h, always past the backoff window
		if b.Claim("k") {
			n++
		}
		c.Advance(15 * time.Minute)
	}
	if n != 10 {
		t.Fatalf("%d tries in a day, want 10", n)
	}
	c.Advance(4 * time.Hour)
	if !b.Claim("k") {
		t.Fatal("still closed the next day")
	}
}

func TestBackoffSucceededForgets(t *testing.T) {
	b, _ := newTestBackoff()
	for range 5 {
		b.Claim("k")
	}
	if b.Claim("k") {
		t.Fatal("not backing off")
	}
	b.Succeeded("k")
	if !b.Claim("k") {
		t.Fatal("still backing off after success")
	}
}

func TestBackoffFailOpenAndSweep(t *testing.T) {
	b, c := newTestBackoff()
	b.maxKeys = 1
	for range 5 {
		b.Claim("a")
	}
	for i := range 2 {
		if !b.Claim("b") {
			t.Fatalf("a new key was refused when the table is full (try %d)", i+1)
		}
	}
	if b.Claim("a") {
		t.Fatal("a tracked key lost its backoff")
	}
	c.Advance(25 * time.Hour)
	b.Claim("b") // sweeps "a"
	if _, ok := b.entries["a"]; ok {
		t.Fatal("idle entry not swept")
	}
}

func TestBackoffOnFullOncePerMax(t *testing.T) {
	b, c := newTestBackoff()
	b.maxKeys = 1
	calls := 0
	b.OnFull(func() { calls++ })
	b.Claim("a")
	for _, k := range []string{"b", "c", "d"} {
		b.Claim(k)
	}
	if calls != 1 {
		t.Fatalf("onFull called %d times in one period, want 1", calls)
	}
	c.Advance(testPolicy.Max)
	b.Claim("e")
	if calls != 2 {
		t.Fatalf("onFull called %d times after a period, want 2", calls)
	}
	b.Claim("a") // a tracked key never reports full
	if calls != 2 {
		t.Fatalf("onFull called for a tracked key")
	}
}
