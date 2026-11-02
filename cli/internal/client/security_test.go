package client

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRedirectsNeverForwardCredentialsOrBody(t *testing.T) {
	for _, status := range []int{301, 302, 303, 307, 308} {
		for _, destination := range []string{"http", "https", "same-origin"} {
			t.Run(http.StatusText(status)+"/"+destination, func(t *testing.T) {
				var received atomic.Int32
				handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { received.Add(1) })
				sink := httptest.NewServer(handler)
				if destination == "https" {
					sink.Close()
					sink = httptest.NewTLSServer(handler)
				}
				defer sink.Close()
				source := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/sink" {
						received.Add(1)
						return
					}
					target := sink.URL + "/sink"
					if destination == "same-origin" {
						target = "/sink"
					}
					w.Header().Set("Location", target)
					w.WriteHeader(status)
				}))
				defer source.Close()
				cl := New(Config{Host: strings.TrimPrefix(source.URL, "https://"), Token: "synthetic-token", Timeout: time.Second}, io.Discard)
				pool := x509.NewCertPool()
				pool.AddCert(source.Certificate())
				if destination == "https" {
					pool.AddCert(sink.Certificate())
				}
				cl.http.Transport.(*http.Transport).TLSClientConfig = &tls.Config{RootCAs: pool}
				res, err := cl.Do(Request{Method: "POST", Path: "inputs", Body: []byte(`{"values":{"password":"synthetic-secret"}}`)})
				var he *HTTPError
				if !errors.As(err, &he) || res.Status != status {
					t.Fatalf("redirect result=%v error=%v", res, err)
				}
				if received.Load() != 0 {
					t.Fatal("redirect destination received a request")
				}
			})
		}
	}
}

func TestProxyDefaultAndRouting(t *testing.T) {
	cl := New(Config{Host: "tenant.example", Token: "synthetic-token", Timeout: time.Second}, io.Discard)
	tr := cl.http.Transport.(*http.Transport)
	if tr.Proxy == nil {
		t.Fatal("environment proxy handling missing")
	}
	// Compare against the default callback under this process's environment.
	req, _ := http.NewRequest("GET", "https://tenant.example", nil)
	got, err := tr.Proxy(req)
	want, werr := http.ProxyFromEnvironment(req)
	if err != werr || (got == nil) != (want == nil) || got != nil && got.String() != want.String() {
		t.Fatal("proxy environment differs from default")
	}
	var routed atomic.Int32
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "CONNECT" || r.Host != "tenant.example:443" {
			t.Errorf("unexpected proxy request %s %s", r.Method, r.Host)
		}
		routed.Add(1)
		w.WriteHeader(502)
	}))
	defer proxy.Close()
	u, _ := url.Parse(proxy.URL)
	tr.Proxy = http.ProxyURL(u)
	_, err = cl.Do(Request{Method: "GET", Path: "commands"})
	if err == nil || routed.Load() != 1 {
		t.Fatalf("proxy was not used: %v", err)
	}
}

func TestInputOutputPolicy(t *testing.T) {
	const secret = "SYNTHETIC-SECRET-UNIQUE"
	for _, path := range []string{"inputs", "inputs/c:I:1", "api/v1/inputs/c:I:1", "bulk/inputs"} {
		for _, method := range []string{"POST", "PUT", "PATCH", "DELETE"} {
			for _, status := range []int{200, 400, 500} {
				srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(status)
					io.WriteString(w, `{"id":"c:I:1","values":{"password":"`+secret+`"},"results":[{"id":"c:I:1","values":{"x":"`+secret+`"}}],"message":"`+secret+`"}`)
				}))
				cl := New(Config{Host: strings.TrimPrefix(srv.URL, "https://"), Token: "synthetic-token", Insecure: true, Timeout: time.Second}, io.Discard)
				res, err := cl.Do(Request{Method: method, Path: path})
				var out bytes.Buffer
				if res != nil {
					if renderErr := cl.Render(&out, res); renderErr != nil {
						t.Fatal(renderErr)
					}
				}
				if strings.Contains(out.String(), secret) || err != nil && strings.Contains(err.Error(), secret) {
					t.Fatalf("secret in %s %s %d output", method, path, status)
				}
				if status == 200 && !strings.Contains(out.String(), "c:I:1") {
					t.Fatal("identity lost")
				}
				srv.Close()
			}
		}
	}
}

func TestPreviewRedactsPayloadAndHeaders(t *testing.T) {
	var out bytes.Buffer
	cl := New(Config{PrintRequest: true}, &out)
	_, err := cl.Do(Request{Method: "POST", Path: "inputs?accessToken=synthetic-token", Headers: map[string]string{"X-Api-Key": "secret-header"}, Body: []byte(`{"values":{"unusual":"secret-value"},"inputKeys":{"opaque":{"type":"string","sensitive":true,"defaultValue":"secret-default"}},"authorizationHeader":"secret-auth"}`)})
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"synthetic-token", "secret-header", "secret-value", "secret-default", "secret-auth"} {
		if strings.Contains(out.String(), secret) {
			t.Fatalf("preview leaked %s", secret)
		}
	}
	if _, err = ProjectObject([]byte(`{} {"values":{"password":"x"}}`), "id"); err == nil {
		t.Fatal("multiple JSON documents accepted")
	}
}

type failedWriter struct{}

func (failedWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
func TestOutputErrorsPropagate(t *testing.T) {
	cl := New(Config{PrintRequest: true}, failedWriter{})
	if _, err := cl.Do(Request{Method: "GET", Path: "commands"}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal("preview write error lost")
	}
	if err := cl.Render(failedWriter{}, &Result{Body: []byte(`{}`)}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal("render write error lost")
	}
}

func TestPreviewRedactsEncodedHeaders(t *testing.T) {
	var out bytes.Buffer
	cl := New(Config{PrintRequest: true}, &out)
	_, err := cl.Do(Request{Method: "POST", Path: "commands", Body: []byte(`{"configuration":{"executors":[{"input":{"headers":"{\"Authorization\":\"encoded-secret\",\"Accept\":\"application/json\"}"}}]}}`)})
	if err != nil || strings.Contains(out.String(), "encoded-secret") || !strings.Contains(out.String(), "application/json") {
		t.Fatal("encoded headers leaked or safe content lost")
	}
}
