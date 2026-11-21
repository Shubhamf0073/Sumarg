package main

import (
	"strings"
	"testing"

	"github.com/Shubhamf0073/previa-school-fleet/backend/platform/config"
)

func TestParseAdminArgs(t *testing.T) {
	a, err := parseAdminArgs([]string{"--email= Asha.K@Previa.IN "})
	if err != nil || a.email != "asha.k@previa.in" || a.name != "asha.k" || a.role != "superadmin" {
		t.Fatalf("defaults: %+v %v", a, err)
	}
	a, err = parseAdminArgs([]string{"--email=r@previa.in", "--name=Ravi", "--role=installer"})
	if err != nil || a.name != "Ravi" || a.role != "installer" {
		t.Fatalf("explicit: %+v %v", a, err)
	}
	for _, bad := range [][]string{
		{}, {"--email=not-an-email"}, {"--email=a@b.in", "--role=owner"}, {"--email=a@b.in", "extra"},
	} {
		if _, err := parseAdminArgs(bad); err == nil {
			t.Errorf("parseAdminArgs(%q) accepted", bad)
		}
	}
}

func TestEnrolBase(t *testing.T) {
	for _, tc := range []struct {
		env    config.Env
		domain string
		want   string
	}{
		{config.EnvProduction, "sumarg.example.com", "https://sumarg.example.com"},
		{config.EnvLocal, "", "http://localhost:5173"},
		{config.EnvProduction, "", ""},
	} {
		got, err := enrolBase(adminConfig{Base: config.Base{AppEnv: tc.env}, AppDomain: tc.domain})
		if got != tc.want || (tc.want == "") != (err != nil) {
			t.Errorf("enrolBase(%s, %q) = %q, %v", tc.env, tc.domain, got, err)
		}
		if err != nil && !strings.Contains(err.Error(), "APP_DOMAIN") {
			t.Errorf("error should name APP_DOMAIN: %v", err)
		}
	}
}
