package httpx

import (
	"bytes"
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
)

// MaxBody is the default request body limit for DecodeJSON. Endpoints that take more (imports) pass their own.
const MaxBody = 1 << 20 // 1 MiB

// DecodeJSON reads exactly one JSON object from r's body into dst. Anything else is a client error, returned as an
// *Error for WriteError:
//   - Content-Type is not application/json → 415
//   - body is larger than maxBytes → 413 (MaxBody if maxBytes <= 0)
//   - body is empty, not an object (null, [], 1), malformed, has a field dst doesn't declare, a field name in
//     different case ("AMOUNT_PAISE"), the same field twice, a wrong type, or data after the object → 400
//
// It uses encoding/json/v2, which matches field names exactly and rejects duplicate names. The v1 decoder matched
// case-insensitively and let the last duplicate win, so {"amount_paise":100,"AMOUNT_PAISE":1} could mean one amount
// to a WAF or audit view and another to the handler. Unknown fields are rejected so a typo or an attacker-added
// field ("tenant_id", "status") is never silently ignored. Validating the values is the caller's job.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any, maxBytes int64) error {
	if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
		return ErrUnsupportedMedia
	}
	if maxBytes <= 0 {
		maxBytes = MaxBody
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBytes))
	if err != nil {
		return decodeError(err)
	}
	if trimmed := bytes.TrimLeft(body, " \t\r\n"); len(trimmed) == 0 || trimmed[0] != '{' {
		return ErrBadRequest
	}
	if err := jsonv2.Unmarshal(body, dst, jsonv2.RejectUnknownMembers(true)); err != nil {
		return decodeError(err)
	}
	return nil
}

func decodeError(err error) error {
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		return ErrTooLarge
	}
	// The decoder's message quotes the client's own input (field names, types); keep it out of the response so one
	// generic body covers every case, and keep it as the cause for debugging.
	return ErrBadRequest.Wrap(err)
}

// WriteJSON sends v as JSON with the given status. v is encoded before anything is written, so an encoding
// failure still produces a clean 500 instead of a half-written body.
func WriteJSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		Logger(r.Context()).ErrorContext(r.Context(), "encode response", slog.Any("error", err))
		status = http.StatusInternalServerError
		buf.Reset()
		buf.WriteString(`{"code":"internal","message":"Something went wrong. Please try again."}` + "\n")
	}
	h := w.Header()
	h.Set("Content-Type", "application/json; charset=utf-8")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}
