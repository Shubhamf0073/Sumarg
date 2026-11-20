// Package auth signs school staff, crew and (M1.3) parents in, and turns a session token back into a principal on
// every request (docs/05 §10, docs/07 S3/S5). Cargo's ADR-0010 is the design it follows: no login oracle, backoff
// instead of lockout, strict token parsing, no session fixation, Origin + double-submit CSRF for the portal.
//
// The tenant is never taken from anything but the token: `<tenant_id>.<token>`, looked up under RLS for that tenant,
// so a forged tenant id finds no row (invariant 1). Only sha256(token) is stored.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

const tokenBytes = 32

// tokenB64Len is the length of 32 bytes in unpadded base64url.
var tokenB64Len = base64.RawURLEncoding.EncodedLen(tokenBytes)

// newToken returns a fresh session token for tenantID and the hash to store.
func newToken(tenantID int64) (token string, hash []byte, err error) {
	raw := make([]byte, tokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("auth: token: %w", err)
	}
	secret := base64.RawURLEncoding.EncodeToString(raw)
	h := sha256.Sum256([]byte(secret))
	return strconv.FormatInt(tenantID, 10) + "." + secret, h[:], nil
}

// parseToken splits `<tenant_id>.<base64url 32 bytes>`. Anything else is rejected before any database work: a
// positive decimal id without sign or leading zero, and exactly 43 base64url characters that decode to 32 bytes.
func parseToken(s string) (tenantID int64, hash []byte, ok bool) {
	idPart, secret, found := strings.Cut(s, ".")
	if !found || len(idPart) == 0 || len(idPart) > 18 || idPart[0] == '0' || len(secret) != tokenB64Len {
		return 0, nil, false
	}
	for _, c := range idPart {
		if c < '0' || c > '9' {
			return 0, nil, false
		}
	}
	id, err := strconv.ParseInt(idPart, 10, 64)
	if err != nil || id <= 0 {
		return 0, nil, false
	}
	raw, err := base64.RawURLEncoding.Strict().DecodeString(secret)
	if err != nil || len(raw) != tokenBytes {
		return 0, nil, false
	}
	h := sha256.Sum256([]byte(secret))
	return id, h[:], true
}
