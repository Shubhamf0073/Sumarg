// Package ratelimit is a small in-process, fixed-window counter per key (e.g. client IP), for slowing down abuse of
// unauthenticated endpoints. It is a first line only: the limits that must hold across restarts and processes live
// in the database (per account, per tenant).
//
// Memory is bounded: the table is cleared every window, and once it holds MaxKeys keys a new key is let through
// uncounted (fail open). Spraying addresses can then only switch this layer off, never lock real users out.
package ratelimit

import (
	"net/netip"
	"sync"
	"time"

	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/clock"
)

// DefaultMaxKeys bounds the table (≈ a few MB).
const DefaultMaxKeys = 50_000

// Limiter allows at most Max hits per key in each Window.
type Limiter struct {
	clock   clock.Clock
	max     int
	window  time.Duration
	maxKeys int

	mu     sync.Mutex
	start  time.Time
	counts map[string]int
}

// New returns a limiter of max hits per key per window.
func New(c clock.Clock, max int, window time.Duration) *Limiter {
	return &Limiter{clock: c, max: max, window: window, maxKeys: DefaultMaxKeys, counts: map[string]int{}}
}

// Allow counts a hit for key and reports whether it is within the limit.
func (l *Limiter) Allow(key string) bool {
	now := l.clock.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	if now.Sub(l.start) >= l.window || now.Before(l.start) {
		l.start, l.counts = now, map[string]int{}
	}
	n, seen := l.counts[key]
	if !seen && len(l.counts) >= l.maxKeys {
		return true
	}
	if n >= l.max {
		return false
	}
	l.counts[key] = n + 1
	return true
}

// IPKey is the key for a client address: the address itself for IPv4, its /64 for IPv6 (one subscriber usually
// holds a whole /64). A missing address shares one key.
func IPKey(a *netip.Addr) string {
	if a == nil || !a.IsValid() {
		return "-"
	}
	ip := a.Unmap()
	if ip.Is6() {
		p, err := ip.Prefix(64)
		if err == nil {
			return p.String()
		}
	}
	return ip.String()
}
