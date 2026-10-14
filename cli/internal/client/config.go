package client

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds everything needed to talk to a SAP Automation Pilot tenant.
// Values are resolved flag > environment variable > error (for required fields).
type Config struct {
	Host     string // AUTOPI_HOSTNAME, e.g. emea.autopilot.cloud.sap
	User     string // AUTOPI_USERNAME
	Password string // AUTOPI_PASSWORD (never logged)
	Catalog  string // AUTOPI_DEFAULT_CATALOG, used as default where --catalog is omitted

	Timeout      time.Duration
	Output       string // "json" (pretty) or "raw" (passthrough)
	PrintRequest bool   // dry-run: render the request instead of sending it
	Insecure     bool   // skip TLS verification (escape hatch, off by default)
}

// SupportedHosts is the canonical list of Automation Pilot regional hostnames.
var SupportedHosts = []string{
	"emea.autopilot.cloud.sap",
	"amer.autopilot.cloud.sap",
	"apac.autopilot.cloud.sap",
	"aus.autopilot.cloud.sap",
	"ksa.autopilot.cloud.sap",
}

// ConfigError marks a missing or invalid credential/host so callers can map it
// to the dedicated config exit code.
type ConfigError struct{ msg string }

func (e *ConfigError) Error() string { return e.msg }

// NewConfigError builds a ConfigError with the given message.
func NewConfigError(msg string) *ConfigError { return &ConfigError{msg: msg} }

// FromEnv seeds a Config from the AUTOPI_* environment variables. Flag parsing
// in the cmd layer overrides any field the user set explicitly.
func FromEnv() Config {
	c := Config{
		Host:     os.Getenv("AUTOPI_HOSTNAME"),
		User:     os.Getenv("AUTOPI_USERNAME"),
		Password: os.Getenv("AUTOPI_PASSWORD"),
		Catalog:  os.Getenv("AUTOPI_DEFAULT_CATALOG"),
		Timeout:  30 * time.Second,
		Output:   "json",
	}
	if v := os.Getenv("AUTOPI_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.Timeout = d
		} else if n, err := strconv.Atoi(v); err == nil {
			c.Timeout = time.Duration(n) * time.Second
		}
	}
	return c
}

// Validate checks that the required connection fields are present. It is called
// by the client just before a real request (not for --print-request, which can
// render a request even with a placeholder host so examples work offline).
func (c Config) Validate() error {
	var missing []string
	if strings.TrimSpace(c.Host) == "" {
		missing = append(missing, "host (--host / AUTOPI_HOSTNAME)")
	}
	if strings.TrimSpace(c.User) == "" {
		missing = append(missing, "user (--user / AUTOPI_USERNAME)")
	}
	if strings.TrimSpace(c.Password) == "" {
		missing = append(missing, "password (--password / AUTOPI_PASSWORD)")
	}
	if len(missing) > 0 {
		return NewConfigError("missing required credentials: " + strings.Join(missing, ", "))
	}
	return nil
}
