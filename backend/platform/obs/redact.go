package obs

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"reflect"
	"strings"
	"time"
	"unicode"
)

// Redacted replaces the value of any field whose name looks secret or personal.
const Redacted = "[redacted]"

// Unloggable replaces a struct (or a map/slice holding structs) that isn't a slog.LogValuer. Its field names
// can't be checked, so it is never written: give the type a LogValue method, or log the fields you need.
const Unloggable = "[unloggable: log fields or implement slog.LogValuer]"

// Field names are split into lowercase words at '_', '-', '.', spaces and camelCase boundaries
// ("X-Api-Key" → x api key, "migratorPW" → migrator pw), then matched two ways:
//   - secretSubstrings: anywhere in the joined name ("apikey", "refreshtoken"). Substrings redact too much now and
//     then ("author" contains "auth"); that is the safe direction for a log.
//   - secretWords / personalWords: whole words only, because they are short ("pin" is inside "shipping", "pan" is
//     inside "company"). Phone-like fields keep their last 4 digits; everything else matched is fully redacted.
//
// config.Secret already redacts itself whatever its field is called; this catches values held in plain strings.
var (
	secretSubstrings = []string{
		"password", "passwd", "passphrase", "secret", "token", "auth", "cookie", "session", "otp", "apikey",
		"accesskey", "privatekey", "privkey", "credential", "dsn", "signature", "bearer", "jwt", "hmac", "salt",
	}
	secretWords   = []string{"pw", "pwd", "pass", "pin", "mpin", "key", "sig", "cvv"}
	personalWords = []string{"phone", "mobile", "whatsapp", "msisdn", "email", "aadhaar", "aadhar", "pan", "upi",
		"vpa", "address", "dob", "ifsc"}
	phoneWords = []string{"phone", "mobile", "whatsapp", "msisdn"}
)

// allowedKeys contain a matched word but are known safe. Keep this list short and exact.
var allowedKeys = map[string]bool{"idempotency_key": true, "key_id": true}

func words(key string) []string {
	var out []string
	var cur []rune
	prevLower := false
	flush := func() {
		if len(cur) > 0 {
			out = append(out, strings.ToLower(string(cur)))
			cur = cur[:0]
		}
	}
	for _, r := range key {
		switch {
		case !unicode.IsLetter(r) && !unicode.IsDigit(r):
			flush()
			prevLower = false
			continue
		case unicode.IsUpper(r) && prevLower:
			flush()
		}
		cur = append(cur, r)
		prevLower = unicode.IsLower(r) || unicode.IsDigit(r)
	}
	flush()
	return out
}

func hasWord(ws []string, list []string) bool {
	for _, w := range ws {
		for _, l := range list {
			if w == l {
				return true
			}
		}
	}
	return false
}

// IsSecretKey reports whether a field called key is redacted (secret or personal data).
func IsSecretKey(key string) bool {
	if allowedKeys[strings.ToLower(key)] {
		return false
	}
	ws := words(key)
	joined := strings.Join(ws, "")
	for _, s := range secretSubstrings {
		if strings.Contains(joined, s) {
			return true
		}
	}
	return hasWord(ws, secretWords) || hasWord(ws, personalWords)
}

// redactHandler redacts secret- and personal-named fields in records, in attrs added with Logger.With, inside
// groups, and inside maps (http.Header, url.Values, map[string]...). It looks at field names only: a secret inside
// a message string or under an innocent name is not caught, so never format secrets into messages.
type redactHandler struct{ slog.Handler }

func (h redactHandler) Handle(ctx context.Context, r slog.Record) error {
	out := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	r.Attrs(func(a slog.Attr) bool {
		out.AddAttrs(redact(a))
		return true
	})
	return h.Handler.Handle(ctx, out)
}

func (h redactHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clean := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		clean[i] = redact(a)
	}
	return redactHandler{h.Handler.WithAttrs(clean)}
}

func (h redactHandler) WithGroup(name string) slog.Handler {
	return redactHandler{h.Handler.WithGroup(name)}
}

func redact(a slog.Attr) slog.Attr {
	if IsSecretKey(a.Key) {
		ws := words(a.Key)
		if hasWord(ws, phoneWords) && !hasWord(ws, secretWords) {
			return slog.String(a.Key, maskPhone(a.Value.Resolve()))
		}
		return slog.String(a.Key, Redacted)
	}
	// Resolve LogValuer values first: a value can turn into a group of fields only once resolved.
	v := a.Value.Resolve()
	switch v.Kind() {
	case slog.KindGroup:
		return slog.Attr{Key: a.Key, Value: slog.GroupValue(redactAll(v.Group())...)}
	case slog.KindAny:
		return slog.Attr{Key: a.Key, Value: redactAny(v.Any(), 0)}
	default:
		return slog.Attr{Key: a.Key, Value: v}
	}
}

func redactAll(attrs []slog.Attr) []slog.Attr {
	out := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		out[i] = redact(a)
	}
	return out
}

// redactAny handles a value slog would otherwise write with encoding/json or fmt, unchecked. Errors are written as
// their message; maps with string keys become groups (so their keys are checked); slices of plain values pass;
// anything holding structs fails closed as Unloggable.
func redactAny(x any, depth int) slog.Value {
	if depth > 4 {
		return slog.StringValue(Unloggable)
	}
	switch t := x.(type) {
	case nil:
		return slog.AnyValue(nil)
	case error:
		return slog.StringValue(t.Error())
	case slog.LogValuer:
		// What a LogValuer returns is checked like any other value: it may be a group holding a "password".
		v := slog.AnyValue(t).Resolve()
		switch v.Kind() {
		case slog.KindGroup:
			return slog.GroupValue(redactAll(v.Group())...)
		case slog.KindAny:
			return redactAny(v.Any(), depth+1)
		}
		return v
	case url.URL:
		return slog.StringValue(t.Redacted()) // String() would include the password in user:pass@host
	case *url.URL:
		return slog.StringValue(t.Redacted())
	case time.Time:
		return slog.TimeValue(t)
	}
	rv := reflect.ValueOf(x)
	// A String method on a plain value (uuid.UUID, money.Paise, config.Secret) is safe to call. On a struct it can
	// print any field, so a struct-backed Stringer is treated like any other struct.
	if s, ok := x.(fmt.Stringer); ok && !isStruct(rv) {
		return slog.StringValue(s.String())
	}
	for rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return slog.AnyValue(nil)
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Map:
		if rv.Type().Key().Kind() != reflect.String {
			return slog.StringValue(Unloggable)
		}
		attrs := make([]slog.Attr, 0, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			k := iter.Key().String()
			if IsSecretKey(k) {
				attrs = append(attrs, redact(slog.Any(k, iter.Value().Interface())))
				continue
			}
			attrs = append(attrs, slog.Attr{Key: k, Value: redactAny(iter.Value().Interface(), depth+1)})
		}
		return slog.GroupValue(attrs...)
	case reflect.Slice, reflect.Array:
		if rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() == reflect.Uint8 {
			return slog.StringValue(Unloggable) // []byte: raw bodies, keys
		}
		if isPlain(rv.Type().Elem()) {
			return slog.AnyValue(rv.Interface())
		}
		items := make([]any, rv.Len())
		for i := range rv.Len() {
			v := redactAny(rv.Index(i).Interface(), depth+1)
			if v.Kind() == slog.KindGroup {
				m := map[string]any{}
				for _, a := range v.Group() {
					m[a.Key] = a.Value.Any()
				}
				items[i] = m
			} else {
				items[i] = v.Any()
			}
		}
		return slog.AnyValue(items)
	case reflect.Struct:
		return slog.StringValue(Unloggable)
	default:
		return slog.AnyValue(rv.Interface())
	}
}

func isStruct(rv reflect.Value) bool {
	for rv.Kind() == reflect.Pointer && !rv.IsNil() {
		rv = rv.Elem()
	}
	return rv.Kind() == reflect.Struct
}

func isPlain(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint,
		reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64, reflect.String:
		return true
	}
	return false
}

// maskPhone keeps the last 4 digits so support can match a log line to a customer without the full number.
func maskPhone(v slog.Value) string {
	var digits []rune
	for _, r := range fmt.Sprint(v.Any()) {
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
		}
	}
	if len(digits) < 8 {
		return Redacted
	}
	return "******" + string(digits[len(digits)-4:])
}
