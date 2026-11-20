package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/Shubhamf0073/previa-school-fleet/backend/internal/config"
)

// OTPSMS is one login-code SMS: the DLT OTP template (docs/15 §5) with the code and the school's short name.
type OTPSMS struct {
	Mobile string // 10 digits, no country code
	Code   string
	School string
}

// SMSSender sends SMS through a DLT-registered gateway. Errors must not carry the mobile or the code.
type SMSSender interface {
	SendOTP(ctx context.Context, m OTPSMS) error
}

// NewSMSSender builds the sender SMS_PROVIDER names (config has validated the variables it needs).
func NewSMSSender(cfg config.Config, log *slog.Logger) (SMSSender, error) {
	switch cfg.SMSProvider {
	case config.SMSStub:
		return StubSMS{Log: log}, nil
	case config.SMSMsg91:
		return &MSG91{APIKey: cfg.SMSAPIKey.Reveal(), TemplateID: cfg.SMSDLTTemplateOTP,
			Client: &http.Client{Timeout: smsTimeout}}, nil
	default:
		return nil, fmt.Errorf("SMS_PROVIDER=%s is not built yet (use %s)", cfg.SMSProvider, config.SMSMsg91)
	}
}

const smsTimeout = 10 * time.Second

// dltVarMax: DLT templates allow at most 30 characters per {#var#} (docs/15 §5).
const dltVarMax = 30

func dltVar(s string) string {
	if utf8.RuneCountInString(s) <= dltVarMax {
		return s
	}
	return string([]rune(s)[:dltVarMax])
}

// StubSMS logs the code instead of sending it. config refuses SMS_PROVIDER=stub outside APP_ENV=local, which is the
// only reason logging a code is acceptable here.
type StubSMS struct{ Log *slog.Logger }

func (s StubSMS) SendOTP(ctx context.Context, m OTPSMS) error {
	s.Log.InfoContext(ctx, "stub sms: login code (APP_ENV=local only)",
		slog.String("mobile", "******"+m.Mobile[len(m.Mobile)-4:]), slog.String("code", m.Code),
		slog.String("school", dltVar(m.School)))
	return nil
}

// MSG91 sends through MSG91's Flow API (docs/09 V9). The MSG91 template linked to the DLT OTP template must use
// ##var1## for the code and ##var2## for the school; SMS_DLT_TEMPLATE_OTP holds MSG91's template id for it.
// Checked against fake servers only: verify on the live account when R9 is done (docs/15 §5).
type MSG91 struct {
	APIKey     string
	TemplateID string
	Client     *http.Client
	BaseURL    string // tests; empty = MSG91 production
}

const msg91URL = "https://control.msg91.com/api/v5/flow"

type msg91Recipient struct {
	Mobiles string `json:"mobiles"`
	Var1    string `json:"var1"`
	Var2    string `json:"var2"`
}

type msg91Request struct {
	TemplateID string           `json:"template_id"`
	ShortURL   string           `json:"short_url"`
	Recipients []msg91Recipient `json:"recipients"`
}

// errSMSRejected: the gateway answered but did not accept the message.
var errSMSRejected = errors.New("sms rejected")

func (s *MSG91) SendOTP(ctx context.Context, m OTPSMS) error {
	body, err := json.Marshal(msg91Request{TemplateID: s.TemplateID, ShortURL: "0",
		Recipients: []msg91Recipient{{Mobiles: "91" + m.Mobile, Var1: m.Code, Var2: dltVar(m.School)}}})
	if err != nil {
		return fmt.Errorf("msg91: encode: %w", err)
	}
	url := s.BaseURL
	if url == "" {
		url = msg91URL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("msg91: request: %w", err)
	}
	req.Header.Set("authkey", s.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := s.Client.Do(req)
	if err != nil {
		// Transport errors carry the URL only, never the body.
		return fmt.Errorf("msg91: send: %w", err)
	}
	defer resp.Body.Close()
	var out struct {
		Type string `json:"type"`
	}
	_ = json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&out)
	// The response message is not kept: it may echo the number.
	if resp.StatusCode/100 != 2 || out.Type != "success" {
		return fmt.Errorf("msg91: http %d type %q: %w", resp.StatusCode, out.Type, errSMSRejected)
	}
	return nil
}
