package client

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type outputTransport func(*http.Request) (*http.Response, error)

func (f outputTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Generated secrets stay recognizable after JSON/URL encoding. Fuzz data
// changes content, nesting, encoded-container shapes, and response statuses.
func FuzzProtectedOutput(f *testing.F) {
	f.Add([]byte("fixture"), uint8(0))
	f.Add([]byte{0, 255, '\n'}, uint8(31))
	f.Fuzz(func(t *testing.T, data []byte, shape uint8) {
		if len(data) > 2048 {
			t.Skip()
		}
		secret := "SYNTHETIC_" + hex.EncodeToString(data) + "_SECRET"
		var value any = map[string]any{"password": secret, "values": map[string]any{"opaque": secret}}
		for i := 0; i < int(shape%35); i++ {
			if i%3 == 0 {
				b, _ := json.Marshal(value)
				value = string(b)
			} else {
				value = []any{value}
			}
		}
		body, _ := json.Marshal(map[string]any{"nested": value, "ordinary": "preserved", "count": 0, "enabled": false})
		original := bytes.Clone(body)
		calls := 0
		transport := outputTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			actual, _ := io.ReadAll(r.Body)
			if !bytes.Equal(actual, original) {
				t.Fatal("wire payload changed by output protection")
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(original)), Header: http.Header{}}, nil
		})
		var out bytes.Buffer
		cfg := Config{Host: "test.invalid", Token: "token", PrintRequest: true, Transport: transport}
		req := Request{Method: "POST", Path: "commands", Body: body}
		if _, err := New(cfg, &out).Do(req); err != nil {
			t.Fatal(err)
		}
		if calls != 0 || strings.Contains(out.String(), secret) || !bytes.Equal(body, original) {
			t.Fatal("preview side effect or disclosure")
		}
		cfg.PrintRequest = false
		res, err := New(cfg, &out).Do(req)
		if err != nil || calls != 1 {
			t.Fatalf("mutation: %v calls=%d", err, calls)
		}
		if bytes.Contains(res.Body, []byte(secret)) {
			t.Fatal("mutation response disclosed secret")
		}
		var safe map[string]any
		if err := json.Unmarshal(res.Body, &safe); err != nil {
			t.Fatal(err)
		}
		if safe["ordinary"] != "preserved" || safe["count"] != float64(0) || safe["enabled"] != false {
			t.Fatal("ordinary metadata lost")
		}
	})
}

func FuzzHTTPErrorProtection(f *testing.F) {
	f.Add("server echoed submitted password", uint8(0))
	f.Fuzz(func(t *testing.T, message string, status uint8) {
		if len(message) > 4096 {
			t.Skip()
		}
		for _, path := range []string{"inputs", "commands", "executions", "mcp/probe", "bulk/inputs"} {
			cfg := Config{Host: "test.invalid", Token: "token", Transport: outputTransport(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 400 + int(status%200), Body: io.NopCloser(strings.NewReader(message))}, nil
			})}
			res, err := New(cfg, io.Discard).Do(Request{Method: "GET", Path: path})
			he, ok := err.(*HTTPError)
			if !ok || len(res.Body) != 0 || len(he.Body) != 0 {
				t.Fatal("error body escaped protection")
			}
		}
	})
}

func FuzzURLProtection(f *testing.F) {
	f.Add("fixture", uint8(0))
	f.Fuzz(func(t *testing.T, raw string, shape uint8) {
		if len(raw) > 2048 {
			t.Skip()
		}
		secret := "SYNTHETIC_" + hex.EncodeToString([]byte(raw)) + "_SECRET"
		u := &url.URL{Scheme: "https", Host: "example.invalid", Path: "/ordinary"}
		switch shape % 3 {
		case 0:
			u.User = url.UserPassword("user", secret)
		case 1:
			u.RawQuery = url.Values{"access_token": {secret}, "catalog": {"ordinary"}}.Encode()
		case 2:
			u.Fragment = "access_token=" + secret + "&state=/return?ok=1"
		}
		got := redactURL(u.String())
		if strings.Contains(got, secret) || !strings.Contains(got, "/ordinary") {
			t.Fatal("URL protection failed")
		}
	})
}
