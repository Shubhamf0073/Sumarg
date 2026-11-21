package adminauth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/clock"
)

const (
	ceremonyTTL    = 5 * time.Minute
	ceremonyWindow = 10 * time.Minute
	// maxCeremonies bounds the store. Ceremonies are begun before sign-in, so the per-address limit (30 per
	// 10 min) and this cap keep one client from filling memory; a full store refuses new ones (429) rather than
	// evicting someone's ceremony in progress.
	maxCeremonies = 2000
)

type ceremonyKind int

const (
	kindLogin ceremonyKind = iota + 1
	kindEnrol
)

// ceremony is a begun WebAuthn registration or sign-in: the challenge the browser must sign, and for enrolment
// the link and user it is for.
type ceremony struct {
	kind        ceremonyKind
	session     webauthn.SessionData
	userID      int64
	enrolmentID int64
	expires     time.Time
}

// Ceremonies keeps begun ceremonies in process memory, keyed by a random id in an HttpOnly cookie. Each one can be
// finished once (take removes it) and lives 5 minutes. The API runs as one process (docs/02 A4); a restart in the
// middle of a ceremony only means trying again.
type Ceremonies struct {
	clock clock.Clock
	mu    sync.Mutex
	m     map[string]ceremony
	max   int
}

// NewCeremonies returns an empty store.
func NewCeremonies(c clock.Clock) *Ceremonies {
	return &Ceremonies{clock: c, m: map[string]ceremony{}, max: maxCeremonies}
}

var errCeremoniesFull = fmt.Errorf("adminauth: ceremony store full")

func (s *Ceremonies) put(c ceremony) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("adminauth: ceremony id: %w", err)
	}
	id := base64.RawURLEncoding.EncodeToString(raw)
	now := s.clock.Now()
	c.expires = now.Add(ceremonyTTL)
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.m) >= s.max {
		for k, v := range s.m {
			if !now.Before(v.expires) {
				delete(s.m, k)
			}
		}
		if len(s.m) >= s.max {
			return "", errCeremoniesFull
		}
	}
	s.m[id] = c
	return id, nil
}

// take removes the ceremony and returns it if it is of kind and still live.
func (s *Ceremonies) take(id string, kind ceremonyKind) (ceremony, bool) {
	if len(id) != tokenB64Len {
		return ceremony{}, false
	}
	s.mu.Lock()
	c, ok := s.m[id]
	delete(s.m, id)
	s.mu.Unlock()
	if !ok || c.kind != kind || !s.clock.Now().Before(c.expires) {
		return ceremony{}, false
	}
	return c, true
}
