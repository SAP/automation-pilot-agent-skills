package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

type noPreflightNetwork struct{ t *testing.T }

func (n noPreflightNetwork) RoundTrip(*http.Request) (*http.Response, error) {
	n.t.Fatal("local preflight attempted a network call")
	return nil, nil
}

func TestPreflightIsLocalAndNeverPrintsConfigurationValues(t *testing.T) {
	for _, tc := range []struct {
		host, token string
		valid       bool
	}{
		{"", "", false},
		{"secret-in-host@host.invalid", "synthetic-token", false},
		{"preflight.invalid", "Bearer synthetic-token", false},
		{"preflight.invalid", "synthetic-token", true},
	} {
		t.Run(tc.host, func(t *testing.T) {
			t.Setenv("AUTOMATION_PILOT_HOSTNAME", tc.host)
			t.Setenv("AUTOMATION_PILOT_TOKEN", tc.token)
			var out, errout bytes.Buffer
			ctx := &cmdContext{stdout: &out, stderr: &errout, version: "test", transport: noPreflightNetwork{t}}
			code := dispatch(ctx, []string{"connection", "preflight"})
			if (code == 0) != tc.valid || (!tc.valid && code != exitConfig) {
				t.Fatalf("exit %d", code)
			}
			var result map[string]any
			if err := json.Unmarshal(out.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result["configurationValid"] != tc.valid || result["connectionChecked"] != false {
				t.Fatal(result)
			}
			for _, value := range []string{tc.host, tc.token} {
				if value != "" && strings.Contains(out.String()+errout.String(), value) {
					t.Fatal("configuration value disclosed")
				}
			}
		})
	}
}
