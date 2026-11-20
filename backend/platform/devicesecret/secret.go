// Package devicesecret creates and checks per-device MQTT secrets. Only the argon2id hash is stored
// (fleet.device.secret_hash); the plain secret is shown once at provisioning and lives on the device.
// Hashing lives in pkg/argon2id (shared with user passwords); this package keeps its API by calling it.
package devicesecret

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/argon2id"
)

// Generate returns a new random secret: 24 bytes, base64url without padding (32 characters).
func Generate() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("devicesecret: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Hash returns "argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash>" for plain.
func Hash(plain string) (string, error) { return argon2id.Hash(plain) }

// ErrFormat means the stored hash is not in the expected argon2id format.
var ErrFormat = argon2id.ErrFormat

// Verify reports whether plain matches encoded, comparing in constant time.
func Verify(plain, encoded string) (bool, error) { return argon2id.Verify(plain, encoded) }
