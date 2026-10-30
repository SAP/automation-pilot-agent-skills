package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/SAP/automation-pilot-agent-skills/cli/internal/domain"
)

// Request describes a single API call in resource-agnostic terms. Path is the
// portion after /api/v1 (with or without a leading slash); the client builds the
// full URL, injects Bearer auth, and applies Query and Headers.
type Request struct {
	Method      string
	Path        string
	Query       url.Values
	Body        []byte
	ContentType string            // defaults to application/json when Body is set
	Headers     map[string]string // extra headers (e.g. If-Match)
	Summarize   bool              // strip heavy blocks from the response (list views)
	// FilterIDPrefix, when set, keeps only the string elements of a top-level
	// JSON array that start with the prefix. The /commands/ids endpoint ignores
	// its ?catalog= filter server-side, so the CLI scopes the id list itself.
	FilterIDPrefix string
	Project        func([]byte) ([]byte, error)
	SkipValidation bool // raw api deliberately bypasses the pinned contract
	ShowETag       bool
}

// Result is the outcome of a sent request.
type Result struct {
	Status int
	Body   []byte
	Header http.Header
}

// HTTPError classifies non-2xx responses by status. Do suppresses server error
// bodies because free-form diagnostics can echo submitted credentials.
type HTTPError struct {
	Status int
	Body   []byte
}

func (e *HTTPError) Error() string {
	msg := strings.TrimSpace(string(e.Body))
	if msg == "" {
		msg = http.StatusText(e.Status)
	}
	return fmt.Sprintf("HTTP %d: %s", e.Status, msg)
}

// NetworkError wraps transport-level failures (DNS, TLS, timeout) so they map to
// a distinct exit code from HTTP errors — mirroring curl's exit-status split.
type NetworkError struct{ err error }

func (e *NetworkError) Error() string {
	var requestError *url.Error
	if errors.As(e.err, &requestError) {
		safe := *requestError
		safe.URL = redactURL(requestError.URL)
		return "request failed: " + safe.Error()
	}
	return "request failed: " + e.err.Error()
}
func (e *NetworkError) Unwrap() error { return e.err }

// Client is a thin typed wrapper over net/http for the Automation Pilot API.
type Client struct {
	cfg  Config
	http *http.Client
	out  io.Writer
}

// New builds a Client. out receives --print-request output; pass os.Stdout.
func New(cfg Config, out io.Writer) *Client {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.Insecure {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // #nosec G402 — opt-in escape hatch
	}
	var transport http.RoundTripper = tr
	if cfg.Transport != nil {
		transport = cfg.Transport
	}
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout, Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		out:  out,
	}
}

// baseURL returns https://<host>/api/v1 with the host defaulted to a placeholder
// for --print-request so examples render offline without credentials.
func (c *Client) baseURL() string {
	host := c.cfg.Host
	if strings.TrimSpace(host) == "" {
		host = "<AUTOMATION_PILOT_HOSTNAME>"
	}
	return "https://" + host + "/api/v1"
}

// buildURL joins the base, the (normalized) path, and the query string. Path
// segments are supplied by callers; command/Input IDs retain their raw colons.
func (c *Client) buildURL(req Request) string {
	p := strings.TrimPrefix(req.Path, "/")
	p = strings.TrimPrefix(p, "api/v1/") // tolerate callers passing the full path
	u := c.baseURL() + "/" + p
	if len(req.Query) > 0 {
		u += "?" + req.Query.Encode()
	}
	return u
}

// Do executes req. When PrintRequest is set it renders the request (auth masked)
// and returns nil without touching the network. Otherwise it validates config,
// sends the request, and returns a Result or a typed error.
func (c *Client) Do(req Request) (*Result, error) {
	if len(req.Body) > 32<<20 {
		return nil, NewConfigError("request body exceeds 32 MiB limit")
	}
	for k := range req.Headers {
		if strings.EqualFold(k, "Authorization") {
			return nil, NewConfigError("Authorization is managed by AUTOMATION_PILOT_TOKEN; custom Authorization headers are unsupported")
		}
	}
	fullURL := c.buildURL(req)
	ctype := req.ContentType
	if ctype == "" && len(req.Body) > 0 {
		ctype = "application/json"
	}

	if c.cfg.PrintRequest {
		return nil, c.renderRequest(req.Method, fullURL, ctype, req.Headers, req.Body)
	}

	if err := c.cfg.Validate(); err != nil {
		return nil, err
	}

	var bodyReader io.Reader
	if len(req.Body) > 0 {
		bodyReader = bytes.NewReader(req.Body)
	}
	httpReq, err := http.NewRequest(req.Method, fullURL, bodyReader)
	if err != nil {
		return nil, &NetworkError{err: err}
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.cfg.Token)
	if ctype != "" {
		httpReq.Header.Set("Content-Type", ctype)
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.cfg.Timeout)
	defer cancel()
	httpReq = httpReq.WithContext(ctx)
	resp, err := c.send(httpReq)
	if err != nil {
		return nil, &NetworkError{err: err}
	}
	defer resp.Body.Close()

	const maxResponse = 32 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse+1))
	if err != nil {
		return nil, &NetworkError{err: err}
	}
	if len(body) > maxResponse {
		return nil, &NetworkError{err: fmt.Errorf("response exceeds 32 MiB limit")}
	}
	res := &Result{Status: resp.StatusCode, Body: body, Header: resp.Header}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Free-form errors can echo arbitrary submitted credentials. Retain
		// status and headers; field-name masking cannot protect error prose.
		res.Body = nil
		return res, &HTTPError{Status: resp.StatusCode}
	}
	if req.Method != "GET" && req.Method != "HEAD" {
		if protectedInput(req.Path) {
			res.Body = inputMetadata(body)
		} else if len(body) > 0 && !mcpProtocol(req) {
			res.Body = redactBody(body)
		}
	}
	if req.Project != nil {
		res.Body, err = req.Project(res.Body)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

// renderRequest prints a human-readable, credential-safe view of the request for
// --print-request. The Authorization header is always masked.
func (c *Client) renderRequest(method, fullURL, ctype string, headers map[string]string, body []byte) error {
	var out bytes.Buffer
	fmt.Fprintf(&out, "%s %s\n", method, redactURL(fullURL))
	fmt.Fprintln(&out, "Authorization: Bearer ***")
	if ctype != "" {
		fmt.Fprintf(&out, "Content-Type: %s\n", ctype)
	}
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		value := headers[k]
		if domain.SecretKey(k) {
			value = "***"
		}
		fmt.Fprintf(&out, "%s: %s\n", k, value)
	}
	if len(body) > 0 {
		fmt.Fprintln(&out)
		out.Write(indentJSON(redactBody(body)))
		fmt.Fprintln(&out)
	}
	_, err := c.out.Write(out.Bytes())
	return err
}

// Render writes a Result's body to w according to the output mode: pretty-printed
// JSON ("json") or verbatim ("raw"). Non-JSON bodies pass through unchanged.
func (c *Client) Render(w io.Writer, res *Result) error {
	if res != nil && c.cfg.ResponseEnvelope {
		var body any
		if len(res.Body) > 0 {
			if value, err := decodeJSON(res.Body); err == nil {
				body = value
			} else {
				body = string(res.Body)
			}
		}
		headers := map[string]string{}
		for _, key := range []string{"ETag", "Preference-Applied", "Retry-After", "Content-Type"} {
			if value := res.Header.Get(key); value != "" {
				headers[key] = value
			}
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]any{"status": res.Status, "headers": headers, "body": body})
	}
	if res == nil || len(res.Body) == 0 {
		return nil
	}
	if c.cfg.Output == "raw" {
		if _, err := w.Write(res.Body); err != nil {
			return err
		}
		if !bytes.HasSuffix(res.Body, []byte("\n")) {
			_, err := fmt.Fprintln(w)
			return err
		}
		return nil
	}
	if _, err := w.Write(indentJSON(res.Body)); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w)
	return err
}

// send retries only read-side quota/availability failures within one total deadline.
func (c *Client) send(req *http.Request) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		resp, err := c.http.Do(req)
		if err != nil || (req.Method != "GET" && req.Method != "HEAD") || attempt >= c.cfg.ReadRetries || (resp.StatusCode != 429 && resp.StatusCode != 503) {
			return resp, err
		}
		delay := time.Duration(1<<attempt) * 200 * time.Millisecond
		if value := resp.Header.Get("Retry-After"); value != "" {
			if seconds, err := strconv.ParseInt(value, 10, 64); err == nil && seconds >= 0 {
				if seconds > 86400 {
					seconds = 86400
				}
				delay = time.Duration(seconds) * time.Second
			} else if date, err := http.ParseTime(value); err == nil {
				delay = time.Until(date)
				if delay < 0 {
					delay = 0
				}
			}
		}
		resp.Body.Close()
		timer := time.NewTimer(delay)
		select {
		case <-req.Context().Done():
			timer.Stop()
			return nil, req.Context().Err()
		case <-timer.C:
		}
	}
}

// indentJSON pretty-prints JSON; if the input is not valid JSON it is returned
// unchanged (trailing newline trimmed) so arbitrary bodies still render.
func indentJSON(b []byte) []byte {
	var buf bytes.Buffer
	if json.Indent(&buf, b, "", "  ") == nil {
		return buf.Bytes()
	}
	return bytes.TrimRight(b, "\n")
}

// heavyKeys are the large, resource-specific blocks stripped from list responses
// in summary mode. They cover the user-facing resources: commands
// (configuration/executors/steps/inputKeys/outputKeys), inputs (keys/values),
// mcp-servers (mcpTools/instructions/inputReferences), and scheduled-executions
// (schedule/inputValues/inputReferences). The remaining fields — name, id,
// catalog, description, tags, version, enabled, commandId — are the metadata a
// listing is for. Stripping values/input/output also keeps secret values out of
// the default view. Projection is fail-safe: resources lacking these keys are
// left untouched, so an unknown shape is never broken, only less-trimmed.
var heavyKeys = []string{
	"configuration", "executors", "steps", "listeners",
	"inputKeys", "outputKeys", "keys", "values",
	"input", "output", "tools",
	"mcpTools", "instructions", "inputReferences",
	"inputValues", "schedule",
}

// Summarize strips the heavy blocks (see heavyKeys) from a list response so the
// default view carries only lightweight metadata. A top-level array has each
// object element trimmed; a top-level object is trimmed directly. Non-JSON or a
// parse error returns the bytes unchanged, so arbitrary bodies pass through.
func Summarize(body []byte) []byte {
	var v any
	if json.Unmarshal(body, &v) != nil {
		return body
	}
	switch t := v.(type) {
	case []any:
		for _, el := range t {
			if m, ok := el.(map[string]any); ok {
				stripHeavy(m)
			}
		}
	case map[string]any:
		stripHeavy(t)
	default:
		return body
	}
	out, err := json.Marshal(v)
	if err != nil {
		return body
	}
	return out
}

func stripHeavy(m map[string]any) {
	for _, k := range heavyKeys {
		delete(m, k)
	}
}

// contractKeys are the fields kept when projecting a command down to its
// contract (the caller-facing shape). It is a keep-list, not a delete-list, so
// large or unknown blocks (configuration/executors/steps/...) can never leak
// into a contract view. This is distinct from Summarize/heavyKeys, which also
// strips inputKeys/outputKeys — the contract must keep them.
var contractKeys = []string{
	"id", "catalog", "name", "description", "version",
	"inputKeys", "outputKeys", "tags",
}

// Contract projects a single command object down to its contract fields (see
// contractKeys), dropping configuration and other heavy blocks. A non-JSON or
// non-object body is returned unchanged, matching Summarize's fail-safe rule.
func Contract(body []byte) []byte {
	var v any
	if json.Unmarshal(body, &v) != nil {
		return body
	}
	m, ok := v.(map[string]any)
	if !ok {
		return body
	}
	kept := make(map[string]any, len(contractKeys))
	for _, k := range contractKeys {
		if val, present := m[k]; present {
			kept[k] = val
		}
	}
	out, err := json.Marshal(kept)
	if err != nil {
		return body
	}
	return out
}

// StripKeys removes the named keys from a top-level JSON object (or from each
// object in a top-level array). Used to keep secret material (e.g. an Input's
// values block) out of a get response. Non-JSON or an unexpected shape passes
// through unchanged, so arbitrary bodies are safe.
func StripKeys(body []byte, keys ...string) []byte {
	var v any
	if json.Unmarshal(body, &v) != nil {
		return body
	}
	strip := func(m map[string]any) {
		for _, k := range keys {
			delete(m, k)
		}
	}
	switch t := v.(type) {
	case []any:
		for _, el := range t {
			if m, ok := el.(map[string]any); ok {
				strip(m)
			}
		}
	case map[string]any:
		strip(t)
	default:
		return body
	}
	out, err := json.Marshal(v)
	if err != nil {
		return body
	}
	return out
}

// FilterIDPrefix keeps only the string elements of a top-level JSON array that
// start with prefix. A non-array body, or any non-string element, passes
// through unchanged so arbitrary bodies are safe.
func FilterIDPrefix(body []byte, prefix string) []byte {
	var v any
	if json.Unmarshal(body, &v) != nil {
		return body
	}
	arr, ok := v.([]any)
	if !ok {
		return body
	}
	kept := make([]any, 0, len(arr))
	for _, el := range arr {
		if s, ok := el.(string); ok && !strings.HasPrefix(s, prefix) {
			continue
		}
		kept = append(kept, el)
	}
	out, err := json.Marshal(kept)
	if err != nil {
		return body
	}
	return out
}
