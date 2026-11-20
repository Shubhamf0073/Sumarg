package auth

import (
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/clock"
)

// DeviceCookie marks a portal browser that has signed a user in before (docs/05 §10). It is not a credential: it
// only lets that browser keep trying, at a reduced rate, while the user's daily failure cap is closed, so failures
// from strange addresses can't lock the real user out of their own browser.
const DeviceCookie = "__Host-sf_dev"

// deviceCookieAge: a browser that hasn't signed in for this long is a stranger again.
const deviceCookieAge = 180 * 24 * time.Hour

// DeviceKeys signs and checks device cookies. The key is derived from SECRETS_KEY with HKDF under its own label,
// so it can't be confused with the OTP keys. The zero value signs nothing and accepts nothing (tests).
type DeviceKeys struct {
	mac   []byte
	clock clock.Clock
}

// NewDeviceKeys derives the device-cookie key from SECRETS_KEY (64 hex characters, checked by config at start-up).
func NewDeviceKeys(secretsKeyHex string, c clock.Clock) (DeviceKeys, error) {
	secret, err := hex.DecodeString(secretsKeyHex)
	if err != nil || len(secret) != 32 {
		return DeviceKeys{}, errors.New("auth: SECRETS_KEY must be 32 bytes as hex") // never echo the value
	}
	mac, err := hkdf.Key(sha256.New, secret, nil, "sumarg device cookie v1", 32)
	if err != nil {
		return DeviceKeys{}, fmt.Errorf("auth: derive device key: %w", err)
	}
	return DeviceKeys{mac: mac, clock: c}, nil
}

func (k DeviceKeys) sign(payload string) string {
	m := hmac.New(sha256.New, k.mac)
	m.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

// issue returns a cookie value for this browser and user: tenant.user.issued.nonce.mac. ok=false with zero keys.
func (k DeviceKeys) issue(tenantID, userID int64) (string, bool, error) {
	if k.mac == nil {
		return "", false, nil
	}
	n := make([]byte, 16)
	if _, err := rand.Read(n); err != nil {
		return "", false, fmt.Errorf("auth: device nonce: %w", err)
	}
	payload := strconv.FormatInt(tenantID, 10) + "." + strconv.FormatInt(userID, 10) + "." +
		strconv.FormatInt(k.clock.Now().Unix(), 10) + "." + base64.RawURLEncoding.EncodeToString(n)
	return payload + "." + k.sign(payload), true, nil
}

// known returns the device id (the nonce) when v is a live cookie this server issued for this tenant and user.
func (k DeviceKeys) known(v string, tenantID, userID int64) (string, bool) {
	if k.mac == nil || len(v) > 200 {
		return "", false
	}
	i := strings.LastIndexByte(v, '.')
	if i < 0 {
		return "", false
	}
	payload, mac := v[:i], v[i+1:]
	if !hmac.Equal([]byte(mac), []byte(k.sign(payload))) {
		return "", false
	}
	f := strings.Split(payload, ".")
	if len(f) != 4 || f[0] != strconv.FormatInt(tenantID, 10) || f[1] != strconv.FormatInt(userID, 10) {
		return "", false
	}
	issued, err := strconv.ParseInt(f[2], 10, 64)
	if err != nil {
		return "", false
	}
	if age := k.clock.Now().Sub(time.Unix(issued, 0)); age < 0 || age > deviceCookieAge {
		return "", false
	}
	return f[3], true
}

func deviceCookie(v string) http.Cookie {
	return http.Cookie{Name: DeviceCookie, Value: v, Path: "/", MaxAge: int(deviceCookieAge / time.Second),
		Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode}
}
