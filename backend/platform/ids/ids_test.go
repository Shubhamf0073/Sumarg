package ids

import (
	"bytes"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestNewIsV7AndOrdered(t *testing.T) {
	prev := New()
	if prev.Version() != 7 || prev.Variant() != uuid.RFC4122 {
		t.Fatalf("version %d variant %v, want 7 / RFC4122", prev.Version(), prev.Variant())
	}
	for range 1000 {
		next := New()
		if bytes.Compare(prev[:], next[:]) >= 0 {
			t.Fatalf("ids not strictly increasing: %s then %s", prev, next)
		}
		prev = next
	}
}

func TestParse(t *testing.T) {
	id := New()
	got, err := Parse(id.String())
	if err != nil || got != id {
		t.Fatalf("Parse(%s) = %s, %v", id, got, err)
	}

	s := id.String()
	bad := map[string]string{
		"empty":           "",
		"uppercase":       "0192F1C2-3B4A-7C5D-8E6F-0123456789AB",
		"braces":          "{" + s + "}",
		"urn":             "urn:uuid:" + s,
		"no dashes":       "0192f1c23b4a7c5d8e6f0123456789ab",
		"trailing space":  s + " ",
		"dash moved":      "0192f1c-23b4a-7c5d-8e6f-0123456789ab",
		"non-hex":         "0192f1c2-3b4a-7c5d-8e6f-0123456789ag",
		"nil uuid":        "00000000-0000-0000-0000-000000000000",
		"unicode digit":   "0192f1c2-3b4a-7c5d-8e6f-0123456789a١",
		"sql in the tail": "0192f1c2-3b4a-7c5d-8e6f-0123'; drop--",
	}
	for name, in := range bad {
		t.Run(name, func(t *testing.T) {
			if got, err := Parse(in); !errors.Is(err, ErrInvalid) || got != uuid.Nil {
				t.Fatalf("Parse(%q) = %s, %v; want Nil, ErrInvalid", in, got, err)
			}
		})
	}
}

func FuzzParse(f *testing.F) {
	f.Add(New().String())
	f.Add("00000000-0000-0000-0000-000000000000")
	f.Fuzz(func(t *testing.T, s string) {
		id, err := Parse(s)
		if err != nil {
			return
		}
		if id.String() != s { // anything accepted must already be in canonical form
			t.Fatalf("accepted non-canonical %q (canonical %q)", s, id.String())
		}
	})
}
