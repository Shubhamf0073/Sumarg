package worker

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/config"
)

func TestMSG91Request(t *testing.T) {
	var got msg91Request
	var hdr http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hdr = r.Header.Clone()
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"type":"success","message":"3763646c"}`))
	}))
	defer srv.Close()
	s := &MSG91{APIKey: "k-123", TemplateID: "tmpl-otp", Client: srv.Client(), BaseURL: srv.URL}
	err := s.SendOTP(t.Context(), OTPSMS{Mobile: "9800000001", Code: "123456",
		School: "Delhi Public School Bopal Ahmedabad West"})
	if err != nil {
		t.Fatal(err)
	}
	if hdr.Get("authkey") != "k-123" || hdr.Get("Content-Type") != "application/json" {
		t.Fatalf("headers %v", hdr)
	}
	if got.TemplateID != "tmpl-otp" || len(got.Recipients) != 1 {
		t.Fatalf("body %+v", got)
	}
	r := got.Recipients[0]
	if r.Mobiles != "919800000001" || r.Var1 != "123456" || len([]rune(r.Var2)) != dltVarMax {
		t.Fatalf("recipient %+v", r)
	}
}

func TestMSG91Failures(t *testing.T) {
	for name, h := range map[string]http.HandlerFunc{
		"http 401": func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"type":"error","message":"9800000001 auth failed"}`))
		},
		"200 error": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"type":"error","message":"invalid mobile 9800000001"}`))
		},
		"not json": func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`<html>`)) },
	} {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(h)
			defer srv.Close()
			err := (&MSG91{Client: srv.Client(), BaseURL: srv.URL}).SendOTP(t.Context(),
				OTPSMS{Mobile: "9800000001", Code: "123456", School: "S"})
			if err == nil {
				t.Fatal("accepted")
			}
			if strings.Contains(err.Error(), "9800000001") || strings.Contains(err.Error(), "123456") {
				t.Fatalf("error carries the mobile or code: %v", err)
			}
		})
	}
}

func TestStubLogsMaskedMobile(t *testing.T) {
	var buf bytes.Buffer
	_ = StubSMS{Log: slog.New(slog.NewJSONHandler(&buf, nil))}.SendOTP(t.Context(),
		OTPSMS{Mobile: "9800001234", Code: "654321", School: "S"})
	out := buf.String()
	if strings.Contains(out, "9800001234") || !strings.Contains(out, "1234") || !strings.Contains(out, "654321") {
		t.Fatalf("stub log: %s", out)
	}
}

func TestNewSMSSender(t *testing.T) {
	log := slog.New(slog.DiscardHandler)
	for provider, ok := range map[string]bool{config.SMSStub: true, config.SMSMsg91: true, config.SMS2Factor: false} {
		_, err := NewSMSSender(config.Config{SMSProvider: provider}, log)
		if (err == nil) != ok {
			t.Errorf("%s: err %v", provider, err)
		}
	}
}

func TestOTPRetryDelay(t *testing.T) {
	if otpRetryDelay(0).Seconds() != 5 || otpRetryDelay(1).Seconds() != 15 || otpRetryDelay(9).Seconds() != 15 {
		t.Fatal("retry delays")
	}
}
