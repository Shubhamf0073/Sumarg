package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
)

type thingIn struct {
	Mode string `query:"mode" enum:"a,b" required:"false"`
	Body struct {
		Name string `json:"name" minLength:"2" maxLength:"10"`
	}
}
type thingOut struct {
	Body struct {
		OK bool `json:"ok"`
	}
}

func humaTestRouter(t *testing.T) http.Handler {
	t.Helper()
	log, _ := testLogger()
	r := NewRouter(log)
	api := NewAPI(r, "Test", "0.0.1")
	huma.Register(api, huma.Operation{OperationID: "thing-create", Method: http.MethodPost, Path: "/things"},
		func(_ context.Context, in *thingIn) (*thingOut, error) {
			switch in.Body.Name {
			case "missing":
				return nil, NewError(http.StatusNotFound, "not_found", "Not found.")
			case "boom":
				return nil, errors.New("pq: password authentication failed for sf_app")
			case "wrapped":
				return nil, errors.Join(errors.New("ctx"), NewError(http.StatusConflict, "stale", "Changed elsewhere."))
			}
			out := &thingOut{}
			out.Body.OK = true
			return out, nil
		})
	return r
}

func TestHumaErrorMapping(t *testing.T) {
	h := humaTestRouter(t)
	tests := []struct {
		name, target, body string
		status             int
		code               string
	}{
		{"ok", "/things", `{"name":"alice"}`, 200, ""},
		{"too short", "/things", `{"name":"a"}`, 400, "validation_failed"},
		{"bad enum", "/things?mode=zzz", `{"name":"alice"}`, 400, "validation_failed"},
		{"unknown field", "/things", `{"name":"alice","tenant_id":7}`, 400, "validation_failed"},
		{"handler *Error", "/things", `{"name":"missing"}`, 404, "not_found"},
		{"wrapped *Error", "/things", `{"name":"wrapped"}`, 409, "stale"},
		{"plain error", "/things", `{"name":"boom"}`, 500, "internal"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, h, "POST", tc.target, "application/json", tc.body)
			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d: %s", rec.Code, tc.status, rec.Body)
			}
			if tc.code == "" {
				return
			}
			var b errorBody
			if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
				t.Fatalf("body: %v: %s", err, rec.Body)
			}
			if b.Code != tc.code || b.Message == "" || b.RequestID == "" {
				t.Fatalf("body = %+v, want code %q with message and request_id", b, tc.code)
			}
			if tc.code == "validation_failed" && len(b.Fields) == 0 {
				t.Fatalf("validation error without fields: %s", rec.Body)
			}
			// Internals and the client's own values must never come back.
			for _, leak := range []string{"password", "pq:", "sf_app", "tenant_id\":7", "zzz"} {
				if strings.Contains(rec.Body.String(), leak) {
					t.Fatalf("response leaks %q: %s", leak, rec.Body)
				}
			}
		})
	}
}

func TestHumaUnsupportedMediaType(t *testing.T) {
	rec := do(t, humaTestRouter(t), "POST", "/things", "text/plain", `{"name":"alice"}`)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	if b := errBody(t, rec); b.Code != "unsupported_media_type" {
		t.Fatalf("body = %+v", b)
	}
}

func TestSpecAPIDocumentsErrorShape(t *testing.T) {
	api := NewSpecAPI("Test", "0.0.1")
	huma.Register(api, huma.Operation{OperationID: "thing-create", Method: http.MethodPost, Path: "/things"},
		func(context.Context, *thingIn) (*thingOut, error) { return nil, nil })
	doc, err := json.Marshal(api.OpenAPI())
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		OpenAPI    string `json:"openapi"`
		Components struct {
			Schemas map[string]struct {
				Properties map[string]json.RawMessage `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(doc, &spec); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(spec.OpenAPI, "3.1") {
		t.Fatalf("openapi = %q", spec.OpenAPI)
	}
	e, ok := spec.Components.Schemas["APIError"]
	if !ok {
		t.Fatalf("no APIError schema in %v", spec.Components.Schemas)
	}
	for _, p := range []string{"code", "message", "request_id", "fields"} {
		if _, ok := e.Properties[p]; !ok {
			t.Errorf("APIError schema lacks %q", p)
		}
	}
	if strings.Contains(string(doc), "$schema") {
		t.Error("spec still carries huma's $schema links")
	}
}
