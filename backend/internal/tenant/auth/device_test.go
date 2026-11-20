package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/clock"
)

const testSecretsKey = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"

func TestDeviceCookie(t *testing.T) {
	c := clock.NewFake(time.Date(2026, 11, 21, 8, 0, 0, 0, time.UTC))
	k, err := NewDeviceKeys(testSecretsKey, c)
	if err != nil {
		t.Fatal(err)
	}
	other, err := NewDeviceKeys(strings.Repeat("ab", 32), c)
	if err != nil {
		t.Fatal(err)
	}
	v, ok, err := k.issue(1, 7)
	if err != nil || !ok {
		t.Fatalf("issue = %v, %v", ok, err)
	}
	id, ok := k.known(v, 1, 7)
	if !ok || id == "" {
		t.Fatal("own cookie not known")
	}
	v2, _, _ := k.issue(1, 7)
	if id2, _ := k.known(v2, 1, 7); id2 == id {
		t.Fatal("two browsers share a device id")
	}
	tampered := strings.Replace(v, "1.7.", "1.8.", 1)
	for name, ok := range map[string]bool{
		"other school": func() bool { _, ok := k.known(v, 2, 7); return ok }(),
		"other user":   func() bool { _, ok := k.known(v, 1, 8); return ok }(),
		"tampered":     func() bool { _, ok := k.known(tampered, 1, 8); return ok }(),
		"other key":    func() bool { _, ok := other.known(v, 1, 7); return ok }(),
		"empty":        func() bool { _, ok := k.known("", 1, 7); return ok }(),
		"zero keys":    func() bool { _, ok := DeviceKeys{}.known(v, 1, 7); return ok }(),
	} {
		if ok {
			t.Errorf("%s accepted", name)
		}
	}
	if _, ok, _ := (DeviceKeys{}).issue(1, 7); ok {
		t.Error("zero keys issued a cookie")
	}
	c.Advance(deviceCookieAge + time.Second)
	if _, ok := k.known(v, 1, 7); ok {
		t.Error("expired cookie accepted")
	}
	if _, err := NewDeviceKeys("zz", c); err == nil {
		t.Error("bad key accepted")
	}
}
