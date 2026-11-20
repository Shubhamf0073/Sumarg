package argon2id

import (
	"errors"
	"strings"
	"testing"
)

func TestHashVerify_RoundTrip(t *testing.T) {
	h, err := Hash("correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	ok, err := Verify("correct horse battery", h)
	if err != nil || !ok {
		t.Fatalf("Verify(right) = %v, %v; want true", ok, err)
	}
	if ok, err := Verify("wrong", h); err != nil || ok {
		t.Fatalf("Verify(wrong) = %v, %v; want false, nil", ok, err)
	}
}

func TestHash_SaltedAndParamsInEncoding(t *testing.T) {
	a, _ := Hash("x")
	b, _ := Hash("x")
	if a == b {
		t.Fatal("two hashes of the same input are equal; want a random salt")
	}
	if !strings.HasPrefix(a, "argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("Hash() = %q, want the OWASP argon2id parameters in the prefix", a)
	}
}

func TestVerify_RejectsBadEncodings(t *testing.T) {
	salt := "c2FsdHNhbHRzYWx0c2FsdA"                     // 16 bytes
	key := "a2V5a2V5a2V5a2V5a2V5a2V5a2V5a2V5a2V5a2V5a2U" // 32 bytes
	for _, enc := range []string{
		"", "plain", "argon2i$v=19$m=19456,t=2,p=1$c2FsdA$aGFzaA", "argon2id$v=19$m=1,t=1$x$y",
		"argon2id$v=19$m=4294967295,t=2,p=1$" + salt + "$" + key, // memory bomb
		"argon2id$v=19$m=19456,t=99,p=1$" + salt + "$" + key,     // too many passes
		"argon2id$v=19$m=19456,t=2,p=64$" + salt + "$" + key,     // too many lanes
		"argon2id$v=19$m=19456,t=2,p=1$c2FsdA$" + key,            // short salt
		"argon2id$v=19$m=19456,t=2,p=1$" + salt + "$a2V5",        // short key
	} {
		if ok, err := Verify("x", enc); !errors.Is(err, ErrFormat) || ok {
			t.Errorf("Verify(%q) = %v, %v; want ErrFormat", enc, ok, err)
		}
	}
}
