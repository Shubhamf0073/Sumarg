package clock

import (
	"sync"
	"testing"
	"time"
)

func TestSystemNow(t *testing.T) {
	now := System{}.Now()
	if now.Location() != time.UTC {
		t.Fatalf("location = %v, want UTC", now.Location())
	}
	if now.Nanosecond()%1000 != 0 {
		t.Fatalf("not truncated to microseconds: %v", now)
	}
}

func TestFake(t *testing.T) {
	start := time.Date(2027, 3, 31, 18, 29, 59, 999_999_999, time.UTC)
	f := NewFake(start)
	if got, want := f.Now(), start.Truncate(time.Microsecond); !got.Equal(want) {
		t.Fatalf("Now = %v, want %v", got, want)
	}
	f.Advance(time.Second)
	if got := f.Now().In(IST); got.Year() != 2027 || got.Month() != time.April || got.Day() != 1 {
		t.Fatalf("one second later in IST should be 1 Apr 2027 (new financial year), got %v", got)
	}
	f.Set(time.Date(2026, 10, 27, 0, 0, 0, 0, IST))
	if got := f.Now(); got.Location() != time.UTC || got.Hour() != 18 || got.Day() != 26 {
		t.Fatalf("Set should store UTC: got %v", got)
	}
}

func TestIST(t *testing.T) {
	if _, offset := time.Date(2026, 1, 1, 0, 0, 0, 0, IST).Zone(); offset != 19800 {
		t.Fatalf("IST offset = %d, want 19800 (+05:30)", offset)
	}
}

// Run with -race: concurrent Advance/Now must not race.
func TestFakeConcurrent(t *testing.T) {
	f := NewFake(time.Unix(0, 0))
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			f.Advance(time.Second)
			_ = f.Now()
		})
	}
	wg.Wait()
	if got := f.Now().Unix(); got != 50 {
		t.Fatalf("after 50 Advance(1s): %d, want 50", got)
	}
}
