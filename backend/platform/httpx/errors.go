// Package httpx is the HTTP plumbing every service shares: server timeouts, graceful shutdown, request IDs, panic
// recovery, access logs, strict JSON decoding, one error shape, and health endpoints.
package httpx

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

// Error is an error a handler returns to the client. Status, Code and Message are sent; Err is only logged, so it
// may hold SQL errors or other internals.
//
//	return httpx.NewError(http.StatusLocked, "tenant_read_only", "This account is read-only.")
type Error struct {
	Status  int
	Code    string // stable, machine-readable, snake_case: the frontend switches on it
	Message string // safe to show a user; never built from internal error text
	Err     error  // cause, logged for 5xx; never sent
}

// NewError returns an *Error with no cause.
func NewError(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

// Wrap returns a copy of e carrying cause, for the log.
func (e *Error) Wrap(cause error) *Error {
	c := *e
	c.Err = cause
	return &c
}

func (e *Error) Error() string {
	if e == nil {
		return "httpx: nil *Error"
	}
	if e.Err != nil {
		return fmt.Sprintf("%d %s: %v", e.Status, e.Code, e.Err)
	}
	return fmt.Sprintf("%d %s", e.Status, e.Code)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Common errors. Handlers may define their own with NewError.
var (
	ErrBadRequest       = NewError(http.StatusBadRequest, "bad_request", "The request is not valid.")
	ErrNotFound         = NewError(http.StatusNotFound, "not_found", "Not found.")
	ErrMethodNotAllowed = NewError(http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
	ErrTooLarge         = NewError(http.StatusRequestEntityTooLarge, "body_too_large", "The request body is too large.")
	ErrUnsupportedMedia = NewError(http.StatusUnsupportedMediaType, "unsupported_media_type",
		"The request body must be application/json.")
	ErrInternal = NewError(http.StatusInternalServerError, "internal", "Something went wrong. Please try again.")
)

// errorBody is the one error shape every service returns (docs/architecture/services-and-contracts.md).
type errorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
	// Fields is set by huma endpoints for validation errors (see APIError); WriteError never sets it.
	Fields []FieldError `json:"fields,omitempty"`
}

// WriteError sends err as the standard error body. An error that isn't an *Error becomes a generic 500, so
// internal error text never reaches the client. 5xx causes are logged with the request's context.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	var he *Error
	// A nil *Error or an invalid status would make the error path itself panic: send a 500 instead.
	if !errors.As(err, &he) || he == nil || he.Status < 400 || he.Status > 599 {
		he = ErrInternal.Wrap(err)
	}
	if he.Status >= 500 {
		Logger(r.Context()).ErrorContext(r.Context(), "request failed", slog.String("code", he.Code),
			slog.Any("error", err))
	}
	WriteJSON(w, r, he.Status, errorBody{Code: he.Code, Message: he.Message, RequestID: RequestID(r.Context())})
}
