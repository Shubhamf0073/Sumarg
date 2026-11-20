package ratelimit

import (
	"sync"
	"time"

	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/clock"
)

// BackoffPolicy shapes a Backoff. After Free failures in a row the next try waits Base, doubling per failure up to
// Max; PerDay failures within one Day close the key until that Day is over.
type BackoffPolicy struct {
	Free   int
	Base   time.Duration
	Max    time.Duration
	PerDay int
	Day    time.Duration
}

// Backoff is an in-process exponential backoff per key (e.g. account + client address), for failed sign-ins: an
// attacker's failures slow down only their own key, never the real user's. Like Limiter it is a first line: a cap
// that must hold across restarts belongs in the database.
//
// Claim counts a try as a failure before the secret is checked (so parallel tries can't all slip past the Free-th
// failure), and Succeeded forgets the key. Memory is bounded the same way as Limiter: entries idle for a Day are
// swept, and once the table holds MaxKeys keys a new key is let through uncounted (fail open).
type Backoff struct {
	clock   clock.Clock
	p       BackoffPolicy
	maxKeys int

	mu        sync.Mutex
	nextSweep time.Time
	entries   map[string]*backoffEntry
}

type backoffEntry struct {
	fails    int       // in a row, since the last success
	until    time.Time // no try before this
	dayStart time.Time
	dayFails int
}

// NewBackoff returns an empty Backoff.
func NewBackoff(c clock.Clock, p BackoffPolicy) *Backoff {
	return &Backoff{clock: c, p: p, maxKeys: DefaultMaxKeys, entries: map[string]*backoffEntry{}}
}

// Claim reports whether key may try now; if so the try is counted as a failure until Succeeded is called.
func (b *Backoff) Claim(key string) bool {
	now := b.clock.Now()
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sweep(now)
	e := b.entries[key]
	if e == nil {
		if len(b.entries) >= b.maxKeys {
			return true
		}
		e = &backoffEntry{dayStart: now}
		b.entries[key] = e
	}
	if now.Sub(e.dayStart) >= b.p.Day || now.Before(e.dayStart) {
		e.dayStart, e.dayFails = now, 0
	}
	if now.Before(e.until) || e.dayFails >= b.p.PerDay {
		return false
	}
	e.fails++
	e.dayFails++
	if e.fails >= b.p.Free {
		wait := b.p.Max
		if shift := e.fails - b.p.Free; shift < 30 {
			if w := b.p.Base << shift; w > 0 && w < wait {
				wait = w
			}
		}
		e.until = now.Add(wait)
	}
	return true
}

// Succeeded forgets key's failures: its try was right.
func (b *Backoff) Succeeded(key string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.entries, key)
}

// sweep drops entries whose day is over and whose wait has passed, at most once per Max.
func (b *Backoff) sweep(now time.Time) {
	if now.Before(b.nextSweep) {
		return
	}
	b.nextSweep = now.Add(b.p.Max)
	for k, e := range b.entries {
		if now.Sub(e.dayStart) >= b.p.Day && !now.Before(e.until) {
			delete(b.entries, k)
		}
	}
}
