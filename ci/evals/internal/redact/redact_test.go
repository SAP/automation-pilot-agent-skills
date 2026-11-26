package redact

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRedactionProtectsJSONEscapedAndOverlappingSecrets(t *testing.T) {
	secrets := []string{"credential", "credential-longer", "quote\"and\\slash"}
	data := map[string]string{"first": secrets[1], "second": secrets[2], "ordinary": "preserved"}
	raw, _ := json.Marshal(data)
	if !Contains(string(raw), secrets) {
		t.Fatal("escaped disclosure not detected")
	}
	safe, err := Value(data, secrets)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(safe)
	if Contains(string(encoded), secrets) || strings.Contains(string(encoded), "-longer") || !strings.Contains(string(encoded), "preserved") {
		t.Fatal("redaction failed")
	}
}
