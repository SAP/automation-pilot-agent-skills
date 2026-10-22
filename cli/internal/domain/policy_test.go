package domain

import "testing"

func TestSecretNamePolicy(t *testing.T) {
	for _, name := range []string{"password", "passphrase", "apiKey", "accessToken", "authorizationHeader", "serviceKey", "clientCert", "X-Api-Key", "client_credentials"} {
		if !SecretKey(name) {
			t.Errorf("secret %s missed", name)
		}
	}
	for _, name := range []string{"tokenUrl", "uaaTokenUrl", "clientId", "credentialName", "tokenType", "retryCount"} {
		if SecretKey(name) {
			t.Errorf("metadata %s flagged", name)
		}
	}
}
func TestFeatureRegistry(t *testing.T) {
	for _, tags := range []map[string]string{
		{"feature:priority": "high", "feature:logs": ""},
		{"feature:dryRun": ""},
		{"feature:autoAbort": "168"}, {"feature:retentionPolicy": "delete", "feature:retentionPeriod": "0"},
		{"team": "sre"},
	} {
		if err := ValidateTags(tags, false); err != nil {
			t.Fatal(err)
		}
	}
	for _, tags := range []map[string]string{
		{"feature:dryRun": "true"}, {"feature:priority": "urgent"}, {"feature:autoAbort": "0"},
		{"feature:unknown": ""},
		{"feature:autoAbort": "169"}, {"feature:retentionPeriod": "673"}, {"feature:retentionPeriod": "0"},
		{"feature:retentionPolicy": "forever"}, {"feature:logs": "true"}, {"autopi:status": "x"},
		{"": ""}, {"a=b": "x"}, {"team": "🙂"},
	} {
		if err := ValidateTags(tags, false); err == nil {
			t.Fatalf("accepted %v", tags)
		}
	}
	if err := ValidateTags(map[string]string{"autopi:status": "x"}, true); err != nil {
		t.Fatal(err)
	}
	for _, d := range Durations {
		if !ValidDuration(d) {
			t.Errorf("duration %s rejected", d)
		}
	}
	if ValidDuration("1h") {
		t.Fatal("invalid duration accepted")
	}
}
