package client

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestFromEnvIgnoresLegacyConfiguration(t *testing.T) {
	t.Setenv("AUTOPI_HOSTNAME", "legacy.example")
	t.Setenv("AUTOPI_USERNAME", "legacy-user")
	t.Setenv("AUTOPI_PASSWORD", "legacy-password")
	t.Setenv("AUTOPI_TIMEOUT", "120s")
	t.Setenv("AUTOMATION_PILOT_HOSTNAME", "")
	t.Setenv("AUTOMATION_PILOT_TOKEN", "")

	cfg := FromEnv()
	if cfg.Host != "" || cfg.Token != "" {
		t.Fatal("legacy variables supplied connection configuration")
	}
	if cfg.Timeout != 30*time.Second {
		t.Fatalf("legacy timeout changed default: %s", cfg.Timeout)
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("legacy variables must not satisfy required configuration")
	}

	t.Setenv("AUTOMATION_PILOT_HOSTNAME", "new.example")
	t.Setenv("AUTOMATION_PILOT_TOKEN", "new-token")
	cfg = FromEnv()
	if cfg.Host != "new.example" || cfg.Token != "new-token" {
		t.Fatal("new variables did not supply connection configuration")
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRequiredConnectionFields(t *testing.T) {
	for _, tc := range []struct {
		name, host, token string
		missing           []string
	}{
		{"both missing", "", "", []string{"AUTOMATION_PILOT_HOSTNAME", "AUTOMATION_PILOT_TOKEN"}},
		{"host missing", "", "private-test-token", []string{"AUTOMATION_PILOT_HOSTNAME"}},
		{"token missing", "tenant.example", "", []string{"AUTOMATION_PILOT_TOKEN"}},
		{"blank fields", " \t", " \n", []string{"AUTOMATION_PILOT_HOSTNAME", "AUTOMATION_PILOT_TOKEN"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := (Config{Host: tc.host, Token: tc.token}).Validate()
			var configErr *ConfigError
			if !errors.As(err, &configErr) {
				t.Fatalf("want ConfigError, got %v", err)
			}
			for _, name := range tc.missing {
				if !strings.Contains(err.Error(), name) {
					t.Errorf("missing variable %s absent from error", name)
				}
			}
			if strings.Contains(err.Error(), "private-test-token") {
				t.Fatal("config error exposed token")
			}
		})
	}
}

func TestHostAndTokenShape(t *testing.T) {
	for _, host := range []string{"https://tenant.example", "tenant.example/path", "tenant.example?x=y", "user@tenant.example", "tenant.example:0", "tenant.example:65536", "tenant.example:", "bad host"} {
		if err := (Config{Host: host, Token: "synthetic"}).Validate(); err == nil {
			t.Errorf("invalid host accepted %s", host)
		}
	}
	for _, host := range []string{"tenant.example", "127.0.0.1:443", "[::1]:443"} {
		if err := (Config{Host: host, Token: "synthetic"}).Validate(); err != nil {
			t.Errorf("valid host rejected %s: %v", host, err)
		}
	}
	for _, token := range []string{"Bearer synthetic", "synthetic token", "synthetic\nsecret"} {
		if err := (Config{Host: "tenant.example", Token: token}).Validate(); err == nil || strings.Contains(err.Error(), "synthetic") {
			t.Error("invalid token accepted or disclosed")
		}
	}
}
