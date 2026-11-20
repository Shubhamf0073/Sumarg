// Package clock is the source of "now" for business logic. Code takes a Clock instead of calling time.Now, so
// tests can set and move time without sleeping.
//
// For measuring elapsed time (timeouts, durations) use time.Now / time.Since directly: Clock values have the
// monotonic reading stripped, so they shouldn't be subtracted to time an operation.
package clock

import (
	"sync"
	"time"
)

// IST is India Standard Time. India has no daylight saving, so a fixed +05:30 zone is exact and needs no tzdata
// in the distroless image (docs/02 A4). Business dates and financial years are computed in IST.
var IST = time.FixedZone("IST", 5*60*60+30*60)

// Clock tells the time.
type Clock interface {
	Now() time.Time
}

// System is the real clock.
type System struct{}

// Now returns the current time in UTC, truncated to microseconds: Postgres timestamptz precision. A time written
// to the database and read back then compares equal to the original.
func (System) Now() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }

// Fake is a Clock for tests: it only moves when told to. It is safe for concurrent use.
type Fake struct {
	mu  sync.Mutex
	now time.Time
}

// NewFake returns a Fake set to t (normalised the same way as System.Now).
func NewFake(t time.Time) *Fake { return &Fake{now: normalise(t)} }

func (f *Fake) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

// Set moves the clock to t.
func (f *Fake) Set(t time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = normalise(t)
}

// Advance moves the clock forward by d (backward if d is negative).
func (f *Fake) Advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = normalise(f.now.Add(d))
}

func normalise(t time.Time) time.Time { return t.UTC().Truncate(time.Microsecond) }
