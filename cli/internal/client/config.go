package client

import (
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds everything needed to talk to a SAP Automation Pilot tenant.
// Connection fields come only from the environment; operational options use flags.
type Config struct {
	Host    string // AUTOMATION_PILOT_HOSTNAME, e.g. emea.autopilot.cloud.sap
	Token   string // AUTOMATION_PILOT_TOKEN, without the Bearer prefix (never logged)
	Catalog string // --catalog filter for commands/inputs list verbs (no env default)

	Timeout          time.Duration
	Output           string            // "json" (pretty) or "raw" (passthrough)
	PrintRequest     bool              // request preview: render the request instead of sending it
	Insecure         bool              // skip TLS verification (escape hatch, off by default)
	ResponseEnvelope bool              // opt-in status, selected headers, and body
	ReadRetries      int               // bounded retries for GET/HEAD 429/503 only
	Transport        http.RoundTripper // optional in-process transport; never read from environment
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

// FromEnv reads the two connection variables and defaults operational options.
func FromEnv() Config {
	return Config{
		Host:    os.Getenv("AUTOMATION_PILOT_HOSTNAME"),
		Token:   os.Getenv("AUTOMATION_PILOT_TOKEN"),
		Timeout: 30 * time.Second,
		Output:  "json",
	}
}

// Validate checks that the required connection fields are present. It is called
// by the client just before a real request (not for --print-request, which can
// render a request even with a placeholder host so examples work offline).
func (c Config) Validate() error {
	var missing []string
	if strings.TrimSpace(c.Host) == "" {
		missing = append(missing, "AUTOMATION_PILOT_HOSTNAME")
	}
	if strings.TrimSpace(c.Token) == "" {
		missing = append(missing, "AUTOMATION_PILOT_TOKEN")
	}
	if len(missing) > 0 {
		return NewConfigError("missing required configuration: " + strings.Join(missing, ", "))
	}
	u, err := url.Parse("https://" + c.Host)
	if err != nil || u.Host != c.Host || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || strings.ContainsAny(c.Host, " /\\\t\r\n?#") {
		return NewConfigError("AUTOMATION_PILOT_HOSTNAME must be a hostname, optionally with a port, without scheme or path")
	}
	if ip := net.ParseIP(u.Hostname()); ip == nil {
		if len(u.Hostname()) > 253 {
			return NewConfigError("invalid AUTOMATION_PILOT_HOSTNAME")
		}
		for _, label := range strings.Split(u.Hostname(), ".") {
			if label == "" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
				return NewConfigError("invalid AUTOMATION_PILOT_HOSTNAME")
			}
			for _, r := range label {
				if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-') {
					return NewConfigError("invalid AUTOMATION_PILOT_HOSTNAME")
				}
			}
		}
	}
	if port := u.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return NewConfigError("invalid AUTOMATION_PILOT_HOSTNAME port")
		}
	} else if strings.HasSuffix(c.Host, ":") {
		return NewConfigError("invalid AUTOMATION_PILOT_HOSTNAME port")
	}
	if strings.ContainsAny(c.Token, " \t\r\n") || strings.HasPrefix(strings.ToLower(c.Token), "bearer ") {
		return NewConfigError("AUTOMATION_PILOT_TOKEN must be a token without whitespace or the Bearer prefix")
	}
	return nil
}
