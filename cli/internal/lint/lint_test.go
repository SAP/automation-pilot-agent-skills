package lint

import "testing"

func TestCredentialKeyMustBeSensitive(t *testing.T) {
	js := []byte(`{"name":"BtpCredentials","keys":{"password":{"type":"string","sensitive":false}}}`)
	findings := CheckFile("BtpCredentials.input.json", js)
	if !hasMessage(findings, Critical, "must set") {
		t.Fatalf("want CRITICAL sensitive finding, got %+v", findings)
	}
}

func TestSensitiveKeyNoDefault(t *testing.T) {
	js := []byte(`{"name":"Cred","keys":{"password":{"type":"string","sensitive":true,"defaultValue":"hunter2"}}}`)
	findings := CheckFile("Cred.input.json", js)
	if !hasMessage(findings, Critical, "hardcoded defaultValue") {
		t.Fatalf("want CRITICAL default finding, got %+v", findings)
	}
}

func TestCommandInputKeyCredentialMustBeSensitive(t *testing.T) {
	// A command's inputKeys map carries the same credential rules as an input's keys.
	js := []byte(`{"name":"CallApi","inputKeys":{"token":{"type":"string","sensitive":false}},"configuration":{}}`)
	findings := CheckFile("CallApi.command.json", js)
	if !hasMessage(findings, Critical, "must set") {
		t.Fatalf("want CRITICAL sensitive finding on command inputKeys, got %+v", findings)
	}
}

func TestCommandWithInputKeysNotFlaggedAsCatalog(t *testing.T) {
	// Regression: a command (name + inputKeys, no executors at top level) must
	// not fall through to the catalog kebab-case rule.
	js := []byte(`{"name":"GetResource","inputKeys":{"region":{"type":"string","sensitive":false}},"outputKeys":{},"configuration":{}}`)
	findings := CheckFile("GetResource.command.json", js)
	if hasMessage(findings, Medium, "kebab-case") {
		t.Fatalf("command misclassified as catalog: %+v", findings)
	}
}

func TestFilenameMismatch(t *testing.T) {
	js := []byte(`{"name":"RestartCfApp","executors":[]}`)
	findings := CheckFile("Wrong.command.json", js)
	if !hasMessage(findings, High, "does not match") {
		t.Fatalf("want filename mismatch, got %+v", findings)
	}
}

func TestCleanCommandHasNoCritical(t *testing.T) {
	js := []byte(`{"name":"RestartCfApp","executors":[{"id":"x"}]}`)
	findings := CheckFile("RestartCfApp.command.json", js)
	for _, f := range findings {
		if f.Severity == Critical {
			t.Fatalf("unexpected critical: %+v", f)
		}
	}
}

func TestInvalidJSON(t *testing.T) {
	findings := CheckFile("broken.command.json", []byte(`{nope`))
	if len(findings) != 1 || findings[0].Severity != Critical {
		t.Fatalf("want one critical for invalid json, got %+v", findings)
	}
}

func TestURLSingleExpressionNotFlagged(t *testing.T) {
	// Finding 6 regression: a url whose entire value is one expression has
	// nothing to encode and must not be flagged.
	js := []byte(`{"name":"CallApi","configuration":{"url":"$(.input.endpoint)"},"executors":[{"id":"x"}]}`)
	findings := CheckFile("CallApi.command.json", js)
	if hasMessage(findings, Medium, "toUrlEncoded") {
		t.Fatalf("single-expression url wrongly flagged: %+v", findings)
	}
}

func TestURLInterpolationFlagged(t *testing.T) {
	// A url that splices an expression into literal text still needs encoding.
	js := []byte(`{"name":"CallApi","configuration":{"url":"https://h/api/$(.input.name)"},"executors":[{"id":"x"}]}`)
	findings := CheckFile("CallApi.command.json", js)
	if !hasMessage(findings, Medium, "toUrlEncoded") {
		t.Fatalf("interpolated url should be flagged, got %+v", findings)
	}
}

func TestURLEncodedNotFlagged(t *testing.T) {
	js := []byte(`{"name":"CallApi","configuration":{"url":"https://h/api/$(.input.name | toUrlEncoded)"},"executors":[{"id":"x"}]}`)
	findings := CheckFile("CallApi.command.json", js)
	if hasMessage(findings, Medium, "toUrlEncoded") {
		t.Fatalf("encoded url wrongly flagged: %+v", findings)
	}
}

func TestInvalidDelayFlagged(t *testing.T) {
	// A delay outside the server enum lints clean today but fails deploy with 400.
	js := []byte(`{"name":"Poll","executors":[{"autoRetry":{"maxCount":3,"delay":"3s"}}]}`)
	findings := CheckFile("Poll.command.json", js)
	if !hasMessage(findings, High, "not an allowed interval") {
		t.Fatalf("want delay enum finding, got %+v", findings)
	}
}

func TestInvalidIntervalFlagged(t *testing.T) {
	js := []byte(`{"name":"Poll","executors":[{"initialDelay":{"interval":"45s"}}]}`)
	findings := CheckFile("Poll.command.json", js)
	if !hasMessage(findings, High, "not an allowed interval") {
		t.Fatalf("want interval enum finding, got %+v", findings)
	}
}

func TestValidDelayNotFlagged(t *testing.T) {
	js := []byte(`{"name":"Poll","executors":[{"autoRetry":{"delay":"5s"},"repeat":{"delay":"15s"},"initialDelay":{"interval":"1m"}}]}`)
	findings := CheckFile("Poll.command.json", js)
	if hasMessage(findings, High, "not an allowed interval") {
		t.Fatalf("valid durations wrongly flagged: %+v", findings)
	}
}

func TestExpressionDelayNotFlagged(t *testing.T) {
	// A dynamic delay can only be resolved at runtime, so it must not be flagged.
	js := []byte(`{"name":"Poll","executors":[{"autoRetry":{"delay":"$(.input.delay)"}}]}`)
	findings := CheckFile("Poll.command.json", js)
	if hasMessage(findings, High, "not an allowed interval") {
		t.Fatalf("expression delay wrongly flagged: %+v", findings)
	}
}

func hasMessage(fs []Finding, sev Severity, substr string) bool {
	for _, f := range fs {
		if f.Severity == sev && contains(f.Message, substr) {
			return true
		}
	}
	return false
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
