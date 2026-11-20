// Package ids makes and parses public IDs. Every public ID is a UUIDv7: 74 random bits make it unguessable, and
// its leading timestamp makes new rows land at the end of a B-tree index instead of on a random page.
// Human-facing numbers (LR, bill no.) are never used alone to look anything up (CLAUDE.md non-negotiable #3).
package ids

import (
	"errors"

	"github.com/google/uuid"
)

// ErrInvalid is returned by Parse for anything that isn't a canonical, non-nil UUID.
var ErrInvalid = errors.New("ids: invalid id")

// New returns a new UUIDv7. IDs made by one process are strictly increasing.
// It panics only if the OS random source fails, which Go treats as fatal anyway.
func New() uuid.UUID { return uuid.Must(uuid.NewV7()) }

// Parse accepts only the canonical form we emit: 36 characters, lowercase hex, dashes at 8-13-18-23, not the
// nil UUID. uuid.Parse alone also accepts "urn:uuid:...", "{...}", uppercase and 32-hex forms; docs/02 A27's
// cookie and token parsing must be strict, so every untrusted ID goes through here.
func Parse(s string) (uuid.UUID, error) {
	if len(s) != 36 {
		return uuid.Nil, ErrInvalid
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if c != '-' {
				return uuid.Nil, ErrInvalid
			}
		case (c < '0' || c > '9') && (c < 'a' || c > 'f'):
			return uuid.Nil, ErrInvalid
		}
	}
	id, err := uuid.Parse(s)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, ErrInvalid
	}
	return id, nil
}
