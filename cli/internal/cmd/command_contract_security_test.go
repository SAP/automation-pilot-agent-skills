package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCommandContractProtectsSensitiveDefaults(t *testing.T) {
	const secret = "SYNTHETIC-COMMAND-DEFAULT"
	const body = `{"id":"c:Cred:1","name":"Cred","inputKeys":{"password":{"type":"string","sensitive":true,"defaultValue":"` + secret + `"},"opaque":{"type":"string","sensitive":true,"defaultValue":"` + secret + `"},"clientSecret":{"type":"string","defaultValue":"` + secret + `"},"region":{"type":"string","sensitive":false,"defaultValue":"eu"}},"outputKeys":{"accessToken":{"type":"string","sensitive":true,"defaultValue":"` + secret + `"}},"configuration":{"private":"` + secret + `"}}`
	for _, output := range []string{"json", "raw"} {
		code, out, errb := memoryArgs(t, func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, body)
		}, "", "commands", "contract", "c:Cred:1", "--output", output)
		if code != 0 || strings.Contains(out+errb, secret) {
			t.Fatalf("output=%s code=%d stdout=%s stderr=%s", output, code, out, errb)
		}
		var result struct {
			ID        string
			InputKeys map[string]struct {
				Type         string
				DefaultValue *string
			}
		}
		if json.Unmarshal([]byte(out), &result) != nil || result.ID != "c:Cred:1" {
			t.Fatal("contract identity lost")
		}
		for _, key := range []string{"password", "opaque", "clientSecret"} {
			if schema := result.InputKeys[key]; schema.Type != "string" || schema.DefaultValue != nil {
				t.Fatalf("credential schema changed or secret default retained for %s", key)
			}
		}
		if region := result.InputKeys["region"]; region.DefaultValue == nil || *region.DefaultValue != "eu" {
			t.Fatal("non-sensitive default lost")
		}
	}
}
