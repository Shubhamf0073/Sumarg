// Package argon2id hashes and verifies secrets (user passwords, device MQTT secrets) with argon2id in the encoding
// "argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash>" (raw standard base64). Moved here from pkg/devicesecret, which keeps
// its API by calling this package, so existing device hashes verify unchanged.
package argon2id

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Parameters: OWASP (2025) minimum — 19 MiB memory, 2 iterations, 1 lane.
const (
	memoryKiB  = 19456
	iterations = 2
	lanes      = 1
	keyLen     = 32
	saltLen    = 16
)

var b64 = base64.RawStdEncoding

// ErrFormat means the stored hash is not in the expected argon2id format (or its parameters are out of bounds).
var ErrFormat = errors.New("argon2id: unsupported hash format")

// Hash returns the encoded argon2id hash of plain with a fresh random salt.
func Hash(plain string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("argon2id: %w", err)
	}
	key := argon2.IDKey([]byte(plain), salt, iterations, memoryKiB, lanes, keyLen)
	return fmt.Sprintf("argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, memoryKiB, iterations, lanes,
		b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// Verify reports whether plain matches encoded, comparing in constant time.
func Verify(plain, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 5 || parts[0] != "argon2id" || parts[1] != fmt.Sprintf("v=%d", argon2.Version) {
		return false, ErrFormat
	}
	var m uint32
	var t uint32
	var p uint8
	// Bounds keep a malformed or tampered hash from exhausting memory or shrinking the comparison.
	if n, err := fmt.Sscanf(parts[2], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil || n != 3 ||
		m < 8192 || m > 65536 || t < 1 || t > 10 || p < 1 || p > 4 {
		return false, ErrFormat
	}
	salt, err := b64.DecodeString(parts[3])
	if err != nil || len(salt) < saltLen {
		return false, ErrFormat
	}
	want, err := b64.DecodeString(parts[4])
	if err != nil || len(want) != keyLen {
		return false, ErrFormat
	}
	got := argon2.IDKey([]byte(plain), salt, t, m, p, uint32(len(want))) // #nosec G115 -- len is a small decoded hash
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
