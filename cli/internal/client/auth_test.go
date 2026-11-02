package client

import (
	"bytes"
	"strings"
	"testing"
)

func TestRequestPreviewMasksToken(t *testing.T) {
	for _, token := range []string{"", "private-test-token"} {
		t.Run("token="+token, func(t *testing.T) {
			var out bytes.Buffer
			cl := New(Config{Token: token, PrintRequest: true}, &out)
			res, err := cl.Do(Request{
				Method: "GET", Path: "catalogs",
				Headers: map[string]string{"If-Match": "etag"},
			})
			if err != nil || res != nil {
				t.Fatalf("offline preview: result=%v error=%v", res, err)
			}
			for _, want := range []string{
				"GET https://<AUTOMATION_PILOT_HOSTNAME>/api/v1/catalogs",
				"Authorization: Bearer ***", "If-Match: etag",
			} {
				if !strings.Contains(out.String(), want) {
					t.Errorf("preview missing %q", want)
				}
			}
			if strings.Contains(out.String(), "private-test-token") || strings.Contains(out.String(), "Basic") {
				t.Fatal("preview exposed token or used Basic auth")
			}
		})
	}
}

func TestClientRejectsAuthorizationOverrides(t *testing.T) {
	for _, printRequest := range []bool{false, true} {
		for _, name := range []string{"Authorization", "authorization", "AUTHORIZATION", "aUtHoRiZaTiOn"} {
			var out bytes.Buffer
			cl := New(Config{Host: "tenant.example", Token: "private-test-token", PrintRequest: printRequest}, &out)
			res, err := cl.Do(Request{
				Method: "GET", Path: "catalogs",
				Headers: map[string]string{name: "Basic private-override"},
			})
			if err == nil || res != nil {
				t.Fatalf("custom %s accepted (preview=%t)", name, printRequest)
			}
			if out.Len() != 0 {
				t.Fatal("rejected request rendered output")
			}
			if strings.Contains(err.Error(), "private-override") || strings.Contains(err.Error(), "private-test-token") {
				t.Fatal("override error exposed credentials")
			}
		}
	}
}
