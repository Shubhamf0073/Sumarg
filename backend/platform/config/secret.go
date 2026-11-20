package config

import (
	"fmt"
	"io"
	"log/slog"
)

const redacted = "[redacted]"

// Secret holds a credential: a password, a database URL with a password in it, a private key, a DSN.
// Every way of printing it shows "[redacted]": every fmt verb (%v, %+v, %#v, %s, %q, %x, %d, ...), slog and
// JSON. Only Reveal returns the value, so a secret can't leak through a log line or an error by accident (T20).
//
// Go note: fmt, slog and encoding/json each look for a method on the value (Format, LogValue, MarshalJSON) and
// call it if present. Implementing them here is what makes redaction automatic everywhere.
type Secret string

// Reveal returns the real value. Call it only at the point of use (e.g. when opening the DB pool).
func (s Secret) Reveal() string { return string(s) }

// Format covers every fmt verb. String alone isn't enough: fmt prints the raw value for verbs that don't
// apply to strings (e.g. %d), and for %#v it calls GoString.
func (Secret) Format(f fmt.State, _ rune) { _, _ = io.WriteString(f, redacted) }

// String is for code that calls .String() directly.
func (Secret) String() string { return redacted }

// LogValue makes slog print "[redacted]" with any handler.
func (Secret) LogValue() slog.Value { return slog.StringValue(redacted) }

// MarshalJSON makes encoding/json print "[redacted]" (e.g. if a config struct is ever dumped as JSON).
func (Secret) MarshalJSON() ([]byte, error) { return []byte(`"` + redacted + `"`), nil }
