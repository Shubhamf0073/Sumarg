package otp

import (
	"bytes"
	"strings"
	"testing"
)

const keyA = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
const keyB = "1f1e1d1c1b1a191817161514131211100f0e0d0c0b0a09080706050403020100"

func keys(t *testing.T, hex string) Keys {
	t.Helper()
	k, err := NewKeys(hex)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func TestNewKeysRejectsBadKey(t *testing.T) {
	for _, s := range []string{"", "zz", keyA[:62], keyA + "00"} {
		if _, err := NewKeys(s); err == nil {
			t.Errorf("NewKeys(%q) accepted", s)
		}
	}
}

func TestNewCode(t *testing.T) {
	seen := map[string]bool{}
	for range 200 {
		c, err := NewCode()
		if err != nil {
			t.Fatal(err)
		}
		if len(c) != Digits || strings.Trim(c, "0123456789") != "" {
			t.Fatalf("code %q is not 6 digits", c)
		}
		seen[c] = true
	}
	if len(seen) < 190 {
		t.Fatalf("only %d distinct codes in 200", len(seen))
	}
}

func TestHashBindsSchoolMobileAndKey(t *testing.T) {
	a := keys(t, keyA)
	h := a.Hash(1, "9800000001", "123456")
	if !a.Matches(h, 1, "9800000001", "123456") {
		t.Fatal("own hash does not match")
	}
	for name, ok := range map[string]bool{
		"other code":   a.Matches(h, 1, "9800000001", "123457"),
		"other school": a.Matches(h, 2, "9800000001", "123456"),
		"other mobile": a.Matches(h, 1, "9800000002", "123456"),
		"other key":    keys(t, keyB).Matches(h, 1, "9800000001", "123456"),
		// "1|98…" + "|" must not collide with "19|8…": the separator keeps fields apart.
		"shifted digits": a.Matches(a.Hash(19, "800000001", "123456"), 1, "9800000001", "123456"),
	} {
		if ok {
			t.Errorf("%s matched", name)
		}
	}
	if bytes.Contains(h, []byte("123456")) {
		t.Fatal("hash contains the code")
	}
}

func TestSealOpen(t *testing.T) {
	a := keys(t, keyA)
	aad := BoxAAD(7, "otp:42")
	box, err := a.Seal("654321", aad)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(box, "654321") {
		t.Fatal("box shows the code")
	}
	if got, err := a.Open(box, aad); err != nil || got != "654321" {
		t.Fatalf("Open = %q, %v", got, err)
	}
	box2, _ := a.Seal("654321", aad)
	if box == box2 {
		t.Fatal("two seals are equal (nonce reuse)")
	}
	tampered := []byte(box)
	tampered[len(tampered)-1] ^= 1
	for name, c := range map[string]struct {
		k   Keys
		box string
		aad []byte
	}{
		"other row":    {a, box, BoxAAD(7, "otp:43")},
		"other school": {a, box, BoxAAD(8, "otp:42")},
		"other key":    {keys(t, keyB), box, aad},
		"tampered":     {a, string(tampered), aad},
		"garbage":      {a, "!!", aad},
		"short":        {a, "AAAA", aad},
	} {
		if _, err := c.k.Open(c.box, c.aad); err == nil {
			t.Errorf("%s: opened", name)
		}
	}
}
