// Package otp is the parent-login code's crypto (docs/02 A22, docs/07 S4), shared by the API, which issues and
// checks codes, and the worker, which sends them by SMS.
//
//   - The code is 6 uniform random digits from crypto/rand.
//   - Only an HMAC of it is stored on the challenge (otp_challenges.code_hash). The HMAC also covers the school and
//     the mobile, so a hash copied to another row matches nothing.
//   - The worker needs the code itself to send it, so the outbox row (notifications.data) holds it sealed with
//     AES-256-GCM, bound to that row; the worker wipes it once the SMS is sent, skipped or given up on.
//
// Both keys are derived from SECRETS_KEY with HKDF under different labels, so neither use weakens the other.
package otp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
)

// Digits is the code length (docs/05 §10).
const Digits = 6

// Keys holds the two derived keys. The zero value is unusable; build it with NewKeys.
type Keys struct {
	mac  []byte
	aead cipher.AEAD
}

// NewKeys derives the OTP keys from SECRETS_KEY (64 hex characters, checked by config at start-up).
func NewKeys(secretsKeyHex string) (Keys, error) {
	secret, err := hex.DecodeString(secretsKeyHex)
	if err != nil || len(secret) != 32 {
		return Keys{}, errors.New("otp: SECRETS_KEY must be 32 bytes as hex") // never echo the value
	}
	mac, err := hkdf.Key(sha256.New, secret, nil, "sumarg otp hmac v1", 32)
	if err != nil {
		return Keys{}, fmt.Errorf("otp: derive hmac key: %w", err)
	}
	boxKey, err := hkdf.Key(sha256.New, secret, nil, "sumarg otp outbox v1", 32)
	if err != nil {
		return Keys{}, fmt.Errorf("otp: derive box key: %w", err)
	}
	block, err := aes.NewCipher(boxKey)
	if err != nil {
		return Keys{}, fmt.Errorf("otp: cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return Keys{}, fmt.Errorf("otp: gcm: %w", err)
	}
	return Keys{mac: mac, aead: aead}, nil
}

var codeSpace = big.NewInt(1_000_000)

// NewCode returns a fresh 6-digit code, uniform over 000000–999999.
func NewCode() (string, error) {
	n, err := rand.Int(rand.Reader, codeSpace)
	if err != nil {
		return "", fmt.Errorf("otp: code: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// Hash is what otp_challenges.code_hash stores: HMAC-SHA256 over the school, the mobile and the code.
func (k Keys) Hash(tenantID int64, mobile, code string) []byte {
	m := hmac.New(sha256.New, k.mac)
	m.Write([]byte(strconv.FormatInt(tenantID, 10) + "|" + mobile + "|" + code))
	return m.Sum(nil)
}

// Matches reports whether code hashes to stored, in constant time.
func (k Keys) Matches(stored []byte, tenantID int64, mobile, code string) bool {
	return hmac.Equal(stored, k.Hash(tenantID, mobile, code))
}

// BoxAAD binds a sealed code to its outbox row: the school and the row's dedupe key.
func BoxAAD(tenantID int64, dedupeKey string) []byte {
	return []byte(strconv.FormatInt(tenantID, 10) + "|" + dedupeKey)
}

// Seal encrypts code for the outbox row aad names. The result is base64url(nonce ‖ ciphertext).
func (k Keys) Seal(code string, aad []byte) (string, error) {
	nonce := make([]byte, k.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("otp: nonce: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(k.aead.Seal(nonce, nonce, []byte(code), aad)), nil
}

// ErrBadBox means a sealed code didn't open: wrong key, wrong row, or tampered.
var ErrBadBox = errors.New("otp: sealed code does not open")

// Open decrypts a Seal result for the row aad names.
func (k Keys) Open(box string, aad []byte) (string, error) {
	raw, err := base64.RawURLEncoding.Strict().DecodeString(box)
	ns := k.aead.NonceSize()
	if err != nil || len(raw) < ns+k.aead.Overhead() {
		return "", ErrBadBox
	}
	pt, err := k.aead.Open(nil, raw[:ns], raw[ns:], aad)
	if err != nil {
		return "", ErrBadBox
	}
	return string(pt), nil
}
