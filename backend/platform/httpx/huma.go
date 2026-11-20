package httpx

import (
	"errors"
	"log/slog"
	"net/http"
	"sync"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

// FieldError is one invalid input in a validation error. Path is where it was ("body.items[3].tags",
// "query.limit"); Message says what is wrong. The offending value is deliberately not echoed back: it may be a
// child's name, a phone number or a token.
type FieldError struct {
	Path    string `json:"path" doc:"Where the problem is, e.g. body.name or query.limit"`
	Message string `json:"message" doc:"What is wrong with it"`
}

// APIError is the one error body of every huma endpoint, in the OpenAPI document and on the wire. It has the same
// shape as WriteError's body, plus Fields for validation errors (docs/04 "Errors").
type APIError struct {
	Status    int          `json:"-"`
	Code      string       `json:"code" doc:"Stable machine-readable code, snake_case; clients switch on it" example:"not_found"`
	Message   string       `json:"message" doc:"Safe to show a user"`
	RequestID string       `json:"request_id,omitempty" doc:"Quote this when reporting a problem"`
	Fields    []FieldError `json:"fields,omitempty" doc:"Validation errors only"`
}

// GetStatus makes APIError a huma.StatusError.
func (e *APIError) GetStatus() int { return e.Status }

func (e *APIError) Error() string { return e.Code + ": " + e.Message }

var installOnce sync.Once

// installHumaErrors points huma's global error constructors at APIError. Everything huma writes itself (request
// validation, an unknown content type, a handler error that is not an *Error) and everything a handler returns as
// *Error then leaves as the same body. huma has no per-API hook for this, so it is done once per process.
func installHumaErrors() {
	installOnce.Do(func() {
		huma.NewError = func(status int, msg string, errs ...error) huma.StatusError {
			return newAPIError(nil, status, msg, errs)
		}
		huma.NewErrorWithContext = func(ctx huma.Context, status int, msg string, errs ...error) huma.StatusError {
			return newAPIError(ctx, status, msg, errs)
		}
	})
}

// newAPIError builds the body for huma. ctx is nil when huma builds an example error for the OpenAPI document.
func newAPIError(ctx huma.Context, status int, msg string, errs []error) *APIError {
	// A handler's *Error arrives wrapped as the one error of a 500 "unexpected error occurred".
	for _, err := range errs {
		var he *Error
		if errors.As(err, &he) && he != nil && he.Status >= 400 && he.Status <= 599 {
			return finishAPIError(ctx, &APIError{Status: he.Status, Code: he.Code, Message: he.Message}, he)
		}
	}
	e := &APIError{Status: status}
	switch {
	case status == http.StatusUnprocessableEntity:
		// huma's validation failure. 400 like every other malformed request (DecodeJSON), never echoing values.
		e.Status, e.Code, e.Message = http.StatusBadRequest, "validation_failed", "Some fields are not valid."
		for _, err := range errs {
			var d huma.ErrorDetailer
			if errors.As(err, &d) {
				if det := d.ErrorDetail(); det != nil {
					e.Fields = append(e.Fields, FieldError{Path: det.Location, Message: det.Message})
				}
			}
		}
	case status >= 500:
		// Never send the message huma was given: it can quote internals. The cause is logged below.
		e.Code, e.Message = ErrInternal.Code, ErrInternal.Message
	default:
		e.Code, e.Message = humaCode(status), humaMessage(status, msg)
	}
	var cause error
	if len(errs) > 0 {
		cause = errs[0]
	}
	return finishAPIError(ctx, e, cause)
}

func finishAPIError(ctx huma.Context, e *APIError, cause error) *APIError {
	if ctx == nil {
		return e
	}
	rc := ctx.Context()
	e.RequestID = RequestID(rc)
	if e.Status >= 500 {
		Logger(rc).ErrorContext(rc, "request failed", slog.String("code", e.Code), slog.Any("error", cause))
	}
	return e
}

// humaCode maps statuses huma raises on its own (unknown content type, bad Accept, malformed parameter) to codes.
func humaCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return ErrBadRequest.Code
	case http.StatusNotFound:
		return ErrNotFound.Code
	case http.StatusMethodNotAllowed:
		return ErrMethodNotAllowed.Code
	case http.StatusRequestEntityTooLarge:
		return ErrTooLarge.Code
	case http.StatusUnsupportedMediaType:
		return ErrUnsupportedMedia.Code
	case http.StatusNotAcceptable:
		return "not_acceptable"
	default:
		return "bad_request"
	}
}

// humaMessage is the safe fixed text for a status huma raised itself; huma's own text can quote the request.
func humaMessage(status int, _ string) string {
	switch status {
	case http.StatusNotFound:
		return ErrNotFound.Message
	case http.StatusMethodNotAllowed:
		return ErrMethodNotAllowed.Message
	case http.StatusRequestEntityTooLarge:
		return ErrTooLarge.Message
	case http.StatusUnsupportedMediaType:
		return ErrUnsupportedMedia.Message
	case http.StatusNotAcceptable:
		return "The server cannot produce a response in the requested format."
	default:
		return ErrBadRequest.Message
	}
}

// NewAPI mounts huma on r and returns the API to register operations on. Operations are written as
// huma.Register(api, huma.Operation{OperationID: "<module>-<verb>", …}, handler); handlers return *Error for client
// errors. The API serves no /openapi.json or /docs: the spec is produced at build time by `<binary> openapi`
// (make gen-api), so production exposes nothing extra.
func NewAPI(r chi.Router, title, version string) huma.API {
	installHumaErrors()
	return humachi.New(r, apiConfig(title, version))
}

// NewSpecAPI is NewAPI on a throwaway router, for `<binary> openapi`: the same configuration and error types, with
// no listener, config or database behind it.
func NewSpecAPI(title, version string) huma.API {
	return NewAPI(chi.NewRouter(), title, version)
}

func apiConfig(title, version string) huma.Config {
	cfg := huma.DefaultConfig(title, version)
	cfg.OpenAPIPath, cfg.DocsPath, cfg.SchemasPath = "", "", ""
	// DefaultConfig adds a "$schema" property to every response and a Link header; generated TS types don't want it.
	cfg.CreateHooks = nil
	return cfg
}
