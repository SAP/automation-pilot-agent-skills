package client

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type inMemoryTransport func(*http.Request) (*http.Response, error)

func (f inMemoryTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func fakeResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}
func TestContractFeaturesBearerOnlyAndMaskedPreview(t *testing.T) {
	t.Setenv("AUTOMATION_PILOT_HOSTNAME", "fixture.example")
	t.Setenv("AUTOMATION_PILOT_TOKEN", "fixture-token")
	// Removed authentication variables must neither override Bearer nor provide
	// a fallback when the required token is absent.
	t.Setenv("AUTOMATION_PILOT_AUTH_MODE", "basic")
	t.Setenv("AUTOMATION_PILOT_USERNAME", "legacy")
	t.Setenv("AUTOMATION_PILOT_PASSWORD", "synthetic-password")
	cfg := FromEnv()
	calls := 0
	cfg.Transport = inMemoryTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Header.Get("Authorization") != "Bearer fixture-token" {
			t.Fatal("request did not use the configured Bearer token")
		}
		return fakeResponse(200, `{}`), nil
	})
	if _, err := New(cfg, io.Discard).Do(Request{Method: "GET", Path: "catalogs"}); err != nil {
		t.Fatal(err)
	}
	cfg.PrintRequest = true
	var out bytes.Buffer
	if _, err := New(cfg, &out).Do(Request{Method: "GET", Path: "catalogs"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), cfg.Token) || strings.Contains(out.String(), "synthetic-password") || strings.Contains(out.String(), "Basic") || !strings.Contains(out.String(), "Bearer ***") {
		t.Fatal("unsafe Bearer preview")
	}
	t.Setenv("AUTOMATION_PILOT_TOKEN", "")
	missingToken := FromEnv()
	missingToken.Transport = cfg.Transport
	_, err := New(missingToken, io.Discard).Do(Request{Method: "GET", Path: "catalogs"})
	var configError *ConfigError
	if !errors.As(err, &configError) || !strings.Contains(err.Error(), "AUTOMATION_PILOT_TOKEN") || calls != 1 {
		t.Fatalf("removed credentials provided fallback: calls=%d err=%v", calls, err)
	}
}
func TestContractFeaturesReadRetriesAndMutationAmbiguity(t *testing.T) {
	for _, method := range []string{"GET", "POST"} {
		t.Run(method, func(t *testing.T) {
			calls := 0
			cfg := Config{Host: "fixture.example", Token: "fixture", Timeout: time.Second, ReadRetries: 2}
			cfg.Transport = inMemoryTransport(func(*http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					res := fakeResponse(429, `{"message":"quota"}`)
					res.Header.Set("Retry-After", "0")
					return res, nil
				}
				return fakeResponse(200, `{}`), nil
			})
			_, err := New(cfg, io.Discard).Do(Request{Method: method, Path: "executions"})
			if method == "GET" {
				if err != nil || calls != 2 {
					t.Fatalf("calls=%d err=%v", calls, err)
				}
			} else if err == nil || calls != 1 {
				t.Fatal("mutation was retried")
			}
		})
	}
	calls := 0
	cfg := Config{Host: "fixture.example", Token: "fixture", Timeout: 10 * time.Millisecond, ReadRetries: 2}
	cfg.Transport = inMemoryTransport(func(*http.Request) (*http.Response, error) {
		calls++
		res := fakeResponse(503, `{}`)
		res.Header.Set("Retry-After", "60")
		return res, nil
	})
	start := time.Now()
	_, err := New(cfg, io.Discard).Do(Request{Method: "GET", Path: "catalogs"})
	var network *NetworkError
	if !errors.As(err, &network) || calls != 1 || time.Since(start) > time.Second {
		t.Fatalf("retry escaped deadline calls=%d err=%v", calls, err)
	}
}
func TestContractFeaturesEnvelopeHonorsInputResponseProtection(t *testing.T) {
	for _, status := range []int{200, 400} {
		cfg := Config{Host: "fixture.example", Token: "fixture", Timeout: time.Second, ResponseEnvelope: true}
		cfg.Transport = inMemoryTransport(func(*http.Request) (*http.Response, error) {
			r := fakeResponse(status, `{"id":"c:I:1","values":{"password":"fixture-private"}}`)
			r.Header.Set("Set-Cookie", "fixture-cookie")
			return r, nil
		})
		cl := New(cfg, io.Discard)
		result, _ := cl.Do(Request{Method: "POST", Path: "inputs", Body: []byte(`{}`)})
		var out bytes.Buffer
		if err := cl.Render(&out, result); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out.String(), "fixture-private") || strings.Contains(out.String(), "fixture-cookie") {
			t.Fatal("protected response leaked through envelope")
		}
	}
}

func TestContractFeaturesNetworkErrorsRedactQuerySecrets(t *testing.T) {
	cfg := Config{Host: "fixture.example", Token: "fixture", Timeout: time.Second}
	cfg.Transport = inMemoryTransport(func(*http.Request) (*http.Response, error) {
		return nil, io.ErrUnexpectedEOF
	})
	_, err := New(cfg, io.Discard).Do(Request{Method: "GET", Path: "commands?accessToken=synthetic-private-token&catalog=my-cat"})
	var network *NetworkError
	if !errors.As(err, &network) || !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("transport error classification lost: %v", err)
	}
	if strings.Contains(err.Error(), "synthetic-private-token") || !strings.Contains(err.Error(), "catalog=my-cat") {
		t.Fatalf("network error did not protect secrets while retaining diagnostics: %v", err)
	}
}
