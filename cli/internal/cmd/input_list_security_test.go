package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestInputListFailsClosedOnUnexpectedResponse(t *testing.T) {
	const secret = "SYNTHETIC-INPUT-SECRET"
	for _, body := range []string{
		secret,
		`["` + secret + `"]`,
		`{"items":[{"values":{"token":"` + secret + `"}}]}`,
		`[{"id":{"value":"` + secret + `"}}]`,
		`[{"id":"c:I:1","tags":{"nested":{"value":"` + secret + `"}}}]`,
	} {
		code, out, errb := memoryArgs(t, func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, body)
		}, "", "inputs", "list")
		if code == 0 || out != "" || strings.Contains(errb, secret) {
			t.Fatalf("unexpected response accepted or disclosed: code=%d stdout=%s stderr=%s", code, out, errb)
		}
	}
}

func TestInputListMetadataAndExplicitFullOutput(t *testing.T) {
	const secret = "SYNTHETIC-INPUT-SECRET"
	const body = `[{"id":"c:I:1","catalog":"c","name":"I","version":1,"description":"configuration","owner":"operator","tags":{"team":"ops"},"keys":{"token":{"sensitive":true}},"values":{"token":"` + secret + `"},"unknown":{"payload":"` + secret + `"}}]`
	for _, full := range []bool{false, true} {
		args := []string{"inputs", "list", "--catalog", "c", "--tag", "team=ops"}
		if full {
			args = append(args, "--full")
		}
		code, out, errb := memoryArgs(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("catalog") != "c" || r.URL.Query().Get("tag") != "team=ops" {
				t.Fatal("list filters lost")
			}
			io.WriteString(w, body)
		}, "", args...)
		if code != 0 || strings.Contains(out, secret) != full {
			t.Fatalf("full=%v code=%d stdout=%s stderr=%s", full, code, out, errb)
		}
		var metadata []map[string]any
		if json.Unmarshal([]byte(out), &metadata) != nil || len(metadata) != 1 || metadata[0]["id"] != "c:I:1" || metadata[0]["description"] != "configuration" {
			t.Fatal("metadata lost")
		}
	}
}
