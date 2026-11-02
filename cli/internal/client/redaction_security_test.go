package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestPreviewProtectsExecutorSecrets(t *testing.T) {
	const secret = "SYNTHETIC-EXECUTOR-SECRET"
	body := []byte(`{"configuration":{"executors":[{"execute":"scripts-sapcp:ExecuteNodeJsScript:1","input":{"npmrc":"//registry.example/:_authToken=` + secret + `","stdin":"` + secret + `"}}]}}`)
	var out bytes.Buffer
	cl := New(Config{PrintRequest: true}, &out)
	if _, err := cl.Do(Request{Method: "POST", Path: "commands", Body: body}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), secret) {
		t.Fatal("sensitive script inputs leaked into the request preview")
	}
}

func TestRedactionHandlesEncodedContainersAndCredentialObjects(t *testing.T) {
	const secret = "SYNTHETIC-CONTAINER-SECRET"
	cases := []any{
		map[string]any{"headers": `{"Authorization":"` + secret + `"`},
		map[string]any{"items": []any{`{"password":"` + secret + `"}`}},
		map[string]any{"password": map[string]any{"type": "opaque", "value": secret}},
		map[string]any{"url": "https://user:" + secret + "@service.example/path?accessToken=" + secret},
	}
	var nested any = map[string]any{"headers": `{"Authorization":"` + secret + `"}`}
	for i := 0; i < 40; i++ {
		nested = map[string]any{"nested": nested}
	}
	cases = append(cases, nested)
	for i, value := range cases {
		body, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(redactBody(body)), secret) {
			t.Errorf("case %d leaked an encoded or structured credential", i)
		}
	}
}

func TestRedactionPreservesCredentialSchemaMetadata(t *testing.T) {
	body := []byte(`{"keys":{"password":{"type":"string","sensitive":true,"defaultValue":"SYNTHETIC-DEFAULT"},"opaque":{"type":"string","sensitive":true,"defaultValue":"SYNTHETIC-DEFAULT"}}}`)
	safe := redactBody(body)
	var result struct {
		Keys map[string]struct {
			Type         string
			Sensitive    bool
			DefaultValue *string
		}
	}
	if err := json.Unmarshal(safe, &result); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"password", "opaque"} {
		if got := result.Keys[key]; got.Type != "string" || !got.Sensitive || got.DefaultValue != nil {
			t.Fatalf("schema metadata lost or secret default retained for %s", key)
		}
	}
}

func TestSchemaMapsCannotHideCredentialValues(t *testing.T) {
	const secret = "SYNTHETIC-SCHEMA-SECRET"
	for _, field := range []string{"keys", "inputKeys", "outputKeys"} {
		for _, descriptor := range []string{
			`{"value":"` + secret + `"}`,
			`{"type":"opaque","value":"` + secret + `"}`,
			`{"type":{"value":"` + secret + `"}}`,
			`{"type":"string","value":"` + secret + `"}`,
			`{"type":"string","description":{"value":"` + secret + `"}}`,
			`{"type":"string","allowedValues":["` + secret + `"],"suggestedValues":["` + secret + `"]}`,
		} {
			body := []byte(`{"` + field + `":{"password":` + descriptor + `}}`)
			if strings.Contains(string(redactBody(body)), secret) {
				t.Errorf("%s allowed a credential object through the schema exception: %s", field, descriptor)
			}
		}
	}
}

func TestSensitiveSchemaKeepsBindingsAndConstraints(t *testing.T) {
	const metadata = `{"type":"string","sensitive":true,"required":false,"description":"Bound credential","minSize":1,"maxSize":128,"minValue":0,"maxValue":10,"defaultValueFromInput":{"inputReference":"c:Credentials:1","inputKey":"password"},"allowedValuesFromInputKeys":["selectedKey"],"suggestedValuesFromInputKeys":["suggestedKey"]}`
	for _, name := range []string{"password", "opaque"} {
		var schema map[string]any
		if err := json.Unmarshal([]byte(metadata), &schema); err != nil {
			t.Fatal(err)
		}
		schema["defaultValue"] = "SYNTHETIC-DEFAULT"
		schema["unexpected"] = "SYNTHETIC-DEFAULT"
		body, err := json.Marshal(map[string]any{"inputKeys": map[string]any{name: schema}})
		if err != nil {
			t.Fatal(err)
		}
		var got struct {
			InputKeys map[string]json.RawMessage
		}
		if err := json.Unmarshal(redactBody(body), &got); err != nil {
			t.Fatal(err)
		}
		// Compare the complete useful descriptor, including false and zero values.
		var want, actual any
		if err := json.Unmarshal([]byte(metadata), &want); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(got.InputKeys[name], &actual); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(actual, want) {
			t.Fatalf("binding metadata lost or literal values retained: %s", got.InputKeys[name])
		}
	}
}

func TestRedactionProtectsRootEncodedContainers(t *testing.T) {
	const secret = "SYNTHETIC-ROOT-SECRET"
	for _, text := range []string{`{"password":"` + secret + `"}`, `[{"token":"` + secret + `"}]`, `{"password":"` + secret} {
		body, err := json.Marshal(text)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(redactBody(body)), secret) {
			t.Error("root encoded container disclosed a credential")
		}
	}
}

func TestURLRedactionFailsClosed(t *testing.T) {
	const secret = "SYNTHETIC-URL-SECRET"
	for _, raw := range []string{
		"https://user:" + secret + "@tenant.example/api/v1/commands",
		"https://tenant.example/%zz?password=" + secret,
		"https://tenant.example/api/v1/commands?password=" + secret + "%zz",
		"https://tenant.example/callback#access_token=" + secret + "&state=example",
		"https://tenant.example/callback#access_token=" + secret + "&state=/return?ok=1",
		"https://tenant.example/#/callback?access_token=" + secret + "%26suffix&state=example",
		"https://tenant.example/#/callback?authorization=" + secret,
		"https://tenant.example/#/callback?access_token=" + secret + ";state=example",
	} {
		if strings.Contains(redactURL(raw), secret) {
			t.Error("URL redaction leaked credentials")
		}
		err := &NetworkError{err: &url.Error{Op: "Get", URL: raw, Err: errors.New("synthetic transport failure")}}
		if strings.Contains(err.Error(), secret) {
			t.Error("network error leaked URL credentials")
		}
	}
	if got := redactURL("https://tenant.example/api/v1/commands?catalog=demo"); got != "https://tenant.example/api/v1/commands?catalog=demo" {
		t.Error("non-sensitive URL changed")
	}
	for _, raw := range []string{"https://tenant.example/docs#overview", "https://tenant.example/#/commands?catalog=demo"} {
		if got := redactURL(raw); got != raw {
			t.Errorf("non-sensitive fragment changed: %s", got)
		}
	}
}

func TestPreviewProtectsProxyAuthorization(t *testing.T) {
	const secret = "SYNTHETIC-PROXY-SECRET"
	var out bytes.Buffer
	cl := New(Config{PrintRequest: true}, &out)
	if _, err := cl.Do(Request{Method: "GET", Path: "commands", Headers: map[string]string{"Proxy-Authorization": "Basic " + secret}}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), secret) {
		t.Fatal("proxy authentication leaked into the request preview")
	}
}
