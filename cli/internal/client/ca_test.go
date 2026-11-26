package client

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExplicitCAKeepsTLSVerification(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"ok":true}`)) }))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := Config{Host: strings.TrimPrefix(server.URL, "https://"), Token: "synthetic", CACertFile: path, Timeout: 5 * time.Second}
	c := New(cfg, io.Discard)
	if _, err := c.Do(Request{Method: "GET", Path: "tenant", SkipValidation: true}); err != nil {
		t.Fatal(err)
	}
	wrongHost := cfg
	wrongHost.Host = strings.Replace(cfg.Host, "127.0.0.1", "localhost", 1)
	if wrongHost.Host != cfg.Host {
		_, err := New(wrongHost, io.Discard).Do(Request{Method: "GET", Path: "tenant", SkipValidation: true})
		var hostnameError x509.HostnameError
		if !errors.As(err, &hostnameError) {
			t.Fatal("custom CA did not reject the wrong hostname", err)
		}
	}
	// A configured custom CA must not turn TLS verification off.
	cfg.CACertFile = ""
	if _, err := New(cfg, io.Discard).Do(Request{Method: "GET", Path: "tenant", SkipValidation: true}); err == nil {
		t.Fatal("untrusted certificate accepted")
	}
	cfg.CACertFile = filepath.Join(t.TempDir(), "missing.pem")
	if _, err := New(cfg, io.Discard).Do(Request{Method: "GET", Path: "tenant", SkipValidation: true}); err == nil {
		t.Fatal("missing certificate bundle accepted")
	}
	t.Setenv("SSL_CERT_FILE", path)
	if FromEnv().CACertFile != path {
		t.Fatal("CA environment ignored")
	}
}
