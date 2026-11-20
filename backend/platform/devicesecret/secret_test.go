package devicesecret

import (
	"strings"
	"testing"
)

func TestGenerate_LengthAndUniqueness(t *testing.T) {
	a, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := Generate()
	if len(a) != 32 || a == b || strings.ContainsAny(a, "+/=") {
		t.Fatalf("Generate() = %q, %q; want two distinct 32-char base64url secrets", a, b)
	}
}

func TestHashVerify_RoundTrip(t *testing.T) {
	h, err := Hash("s3cret-device")
	if err != nil {
		t.Fatal(err)
	}
	ok, err := Verify("s3cret-device", h)
	if err != nil || !ok {
		t.Fatalf("Verify(right) = %v, %v; want true", ok, err)
	}
}

func TestVerify_WrongSecret(t *testing.T) {
	h, _ := Hash("right")
	ok, err := Verify("wrong", h)
	if err != nil || ok {
		t.Fatalf("Verify(wrong) = %v, %v; want false, nil", ok, err)
	}
}

func TestVerify_RejectsUnknownFormat(t *testing.T) {
	for _, enc := range []string{"", "plain", "argon2i$v=19$m=19456,t=2,p=1$c2FsdA$aGFzaA", "argon2id$v=19$m=1,t=1$x$y"} {
		if ok, err := Verify("x", enc); err == nil || ok {
			t.Errorf("Verify(%q) = %v, %v; want an error", enc, ok, err)
		}
	}
}

func TestHash_ParamsInEncoding(t *testing.T) {
	h, _ := Hash("x")
	if !strings.HasPrefix(h, "argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("Hash() = %q, want the OWASP argon2id parameters in the prefix", h)
	}
}

func TestVerify_RejectsOutOfBoundsParams(t *testing.T) {
	salt := "c2FsdHNhbHRzYWx0c2FsdA"                     // 16 bytes
	key := "a2V5a2V5a2V5a2V5a2V5a2V5a2V5a2V5a2V5a2V5a2U" // 32 bytes
	for _, enc := range []string{
		"argon2id$v=19$m=4294967295,t=2,p=1$" + salt + "$" + key, // memory bomb
		"argon2id$v=19$m=19456,t=99,p=1$" + salt + "$" + key,     // too many passes
		"argon2id$v=19$m=19456,t=2,p=64$" + salt + "$" + key,     // too many lanes
		"argon2id$v=19$m=19456,t=2,p=1$c2FsdA$" + key,            // short salt
		"argon2id$v=19$m=19456,t=2,p=1$" + salt + "$a2V5",        // short key
	} {
		if ok, err := Verify("x", enc); err == nil || ok {
			t.Errorf("Verify(%q) = %v, %v; want ErrFormat", enc, ok, err)
		}
	}
}
