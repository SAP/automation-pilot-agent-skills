package cmd

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/SAP/automation-pilot-agent-skills/cli/internal/client"
	"github.com/SAP/automation-pilot-agent-skills/cli/internal/contract"
)

// cmdContext carries process-wide I/O and metadata through the dispatch tree.
type cmdContext struct {
	stdin     io.Reader
	stdout    io.Writer
	stderr    io.Writer
	version   string
	transport http.RoundTripper
}

func (c *cmdContext) client(cfg client.Config) *client.Client {
	cfg.Transport = c.transport
	return client.New(cfg, c.stdout)
}

// verb is a single action under a resource (e.g. "list" under "commands").
type verb struct {
	name     string
	summary  string
	endpoint string // shown in help, e.g. "GET /commands"
	run      func(c *cmdContext, args []string) error
}

// resource groups related verbs (e.g. "commands").
type resource struct {
	name    string
	summary string
	verbs   []*verb
}

func (r *resource) verb(name string) *verb {
	for _, v := range r.verbs {
		if v.name == name {
			return v
		}
	}
	return nil
}

// globalFlags are operational options accepted by every verb.
type globalFlags struct {
	catalog, output, timeout                 string
	printRequest, insecure, responseEnvelope bool
	readRetries                              int
}

// bindGlobal registers the global flags on fs and returns the holder.
func bindGlobal(fs *flag.FlagSet) *globalFlags {
	g := &globalFlags{}
	fs.StringVar(&g.output, "output", "", "output format: json (default) or raw")
	fs.StringVar(&g.timeout, "timeout", "", "request timeout, e.g. 30s (default 30s)")
	fs.BoolVar(&g.printRequest, "print-request", false, "render the request (auth masked) without sending it")
	fs.BoolVar(&g.insecure, "insecure", false, "skip TLS verification (escape hatch)")
	fs.BoolVar(&g.responseEnvelope, "response-envelope", false, "emit status, selected headers, and body as JSON")
	fs.IntVar(&g.readRetries, "read-retries", 0, "retry GET/HEAD HTTP 429/503 up to N times (0-5; total timeout unchanged)")
	return g
}

// config reads connection variables and applies operational flags.
func (g *globalFlags) config() (client.Config, error) {
	cfg := client.FromEnv()
	if g.catalog != "" {
		cfg.Catalog = g.catalog
	}
	if g.output != "" {
		if g.output != "json" && g.output != "raw" {
			return cfg, usagef("invalid --output %q (want json or raw)", g.output)
		}
		cfg.Output = g.output
	}
	if g.timeout != "" {
		d, err := time.ParseDuration(g.timeout)
		if err != nil {
			if n, convErr := strconv.Atoi(g.timeout); convErr == nil {
				d = time.Duration(n) * time.Second
			} else {
				return cfg, usagef("invalid --timeout %q", g.timeout)
			}
		}
		if d <= 0 {
			return cfg, usagef("--timeout must be positive")
		}
		cfg.Timeout = d
	}
	cfg.PrintRequest = g.printRequest
	cfg.Insecure = g.insecure
	if g.responseEnvelope && cfg.Output == "raw" {
		return cfg, usagef("--response-envelope cannot be combined with --output raw")
	}
	if g.readRetries < 0 || g.readRetries > 5 {
		return cfg, usagef("--read-retries must be between 0 and 5")
	}
	cfg.ResponseEnvelope = g.responseEnvelope
	cfg.ReadRetries = g.readRetries
	return cfg, nil
}

// newFlagSet returns a FlagSet wired to stderr with global flags bound. Callers
// add verb-specific flags, then call parseArgs.
func newFlagSet(c *cmdContext, name string) (*flag.FlagSet, *globalFlags) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(c.stderr)
	g := bindGlobal(fs)
	return fs, g
}

// errHelpRequested is returned when the user passed -h/--help to a verb; Run
// maps it to a clean exit without an "error:" prefix.
var errHelpRequested = errors.New("help requested")

// parseArgs parses flags that may appear before or after positional arguments
// (the stdlib flag package stops at the first non-flag token; this permutes so
// `commands get <id> --output raw` works). Returns the positional arguments.
func parseArgs(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	var flags []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(a, "-") || a == "-" {
			positional = append(positional, a)
			continue
		}
		flags = append(flags, a)
		name, _, hasValue := strings.Cut(strings.TrimLeft(a, "-"), "=")
		f := fs.Lookup(name)
		if f == nil {
			continue
		}
		if b, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && b.IsBoolFlag() {
			continue
		}
		if !hasValue && i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	if err := fs.Parse(flags); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil, errHelpRequested
		}
		return nil, &usageError{msg: err.Error()}
	}
	return positional, nil
}

func parseNoArgs(fs *flag.FlagSet, args []string) error {
	pos, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 0 {
		return usagef("unexpected positional arguments for %s", fs.Name())
	}
	return nil
}

func flagSet(fs *flag.FlagSet, name string) bool {
	set := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			set = true
		}
	})
	return set
}

// setUsage installs a per-verb --help banner: the endpoint it calls, an optional
// example, then the flag defaults. Keeps the binary the source of truth so skill
// docs can point at `autopi <resource> <verb> --help`.
func setUsage(fs *flag.FlagSet, endpoint, usageLine, example string) {
	fs.Usage = func() {
		w := fs.Output()
		if usageLine != "" {
			fmt.Fprintf(w, "USAGE\n  %s\n\n", usageLine)
		}
		if endpoint != "" {
			fmt.Fprintf(w, "CALLS\n  %s\n\n", endpoint)
		}
		fmt.Fprintln(w, "FLAGS")
		fs.PrintDefaults()
		if example != "" {
			fmt.Fprintf(w, "\nEXAMPLE\n  %s\n", example)
		}
	}
}

// stringList is a repeatable flag value (e.g. --status FAILED --status RUNNING)
// serialized as repeated query keys rather than a comma-joined value.
type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

// addTagFilters serializes repeatable --tag filters as the server expects them:
// a tag query is `key=value`. The CLI accepts `key=value` (matching the wire
// format) so that tag keys containing colons (e.g. `feature:logs`) are not
// split incorrectly. A key-only filter (no `=`) matches any value and is
// forwarded as the bare key.
func addTagFilters(q url.Values, tags stringList) {
	for _, t := range tags {
		if k, v, ok := strings.Cut(t, "="); ok {
			q.Add("tag", k+"="+v)
		} else {
			q.Add("tag", t)
		}
	}
}

// resolveBody returns the request body from either -f/--file (a path) or --data
// (inline JSON, or "@path" curl-style). Exactly one source may be set.
func resolveBody(c *cmdContext, file, data string) ([]byte, error) {
	switch {
	case file != "" && data != "":
		return nil, usagef("use either -f/--file or --data, not both")
	case file != "":
		if file == "-" {
			return readStdin(c)
		}
		return readFile(file)
	case strings.HasPrefix(data, "@"):
		if data == "@-" {
			return readStdin(c)
		}
		return readFile(strings.TrimPrefix(data, "@"))
	case data != "":
		return []byte(data), nil
	default:
		return nil, usagef("a request body is required: pass -f FILE or --data JSON")
	}
}

func readStdin(c *cmdContext) ([]byte, error) {
	r := c.stdin
	if r == nil {
		r = os.Stdin
	}
	b, err := io.ReadAll(io.LimitReader(r, (32<<20)+1))
	if err != nil {
		return nil, fmt.Errorf("cannot read request body from stdin: %w", err)
	}
	if len(b) > 32<<20 {
		return nil, usagef("request body exceeds 32 MiB limit")
	}
	return b, nil
}

func readFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, usagef("cannot read %s: %v", path, err)
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, (32<<20)+1))
	if err != nil {
		return nil, usagef("cannot read request body: %v", err)
	}
	if len(b) > 32<<20 {
		return nil, usagef("request body exceeds 32 MiB limit")
	}
	return b, nil
}

// execute sends req (or renders it under --print-request) and writes the result.
// When statusOnly is set it prints just the HTTP status code (replacing the old
// `curl -o /dev/null -w "%{http_code}"` existence checks) and never errors on a
// non-2xx status.
func execute(c *cmdContext, cfg client.Config, req client.Request, statusOnly bool) error {
	if (statusOnly && req.ShowETag) || (cfg.ResponseEnvelope && (statusOnly || req.ShowETag)) {
		return usagef("--status-only, --show-etag, and --response-envelope are mutually exclusive")
	}
	if !req.SkipValidation {
		if err := contract.ValidateRequest(req.Method, req.Path, req.Query, req.Headers, req.Body); err != nil {
			return err
		}
	}
	cl := c.client(cfg)
	res, err := cl.Do(req)
	if statusOnly {
		var he *client.HTTPError
		if res != nil {
			_, err := fmt.Fprintln(c.stdout, res.Status)
			return err
		}
		if asHTTP(err, &he) {
			_, writeErr := fmt.Fprintln(c.stdout, he.Status)
			return writeErr
		}
		return err // config/network errors still surface
	}
	if err != nil {
		// Still render any error body the API returned before failing.
		if res != nil {
			if writeErr := cl.Render(c.stdout, res); writeErr != nil {
				return writeErr
			}
		}
		return err
	}
	if req.ShowETag && res != nil {
		etag := res.Header.Get("ETag")
		if etag == "" {
			return fmt.Errorf("response has no ETag")
		}
		_, err := fmt.Fprintln(c.stdout, etag)
		return err
	}
	if req.Summarize && res != nil {
		res.Body = client.Summarize(res.Body)
	}
	if req.FilterIDPrefix != "" && res != nil {
		res.Body = client.FilterIDPrefix(res.Body, req.FilterIDPrefix)
	}
	return cl.Render(c.stdout, res)
}

// fetch shares safe API-error rendering with specialized response consumers.
func fetch(c *cmdContext, cl *client.Client, req client.Request) (*client.Result, error) {
	if !req.SkipValidation {
		if err := contract.ValidateRequest(req.Method, req.Path, req.Query, req.Headers, req.Body); err != nil {
			return nil, err
		}
	}
	res, err := cl.Do(req)
	if err != nil && res != nil {
		if writeErr := cl.Render(c.stdout, res); writeErr != nil {
			return nil, writeErr
		}
	}
	return res, err
}

func asHTTP(err error, target **client.HTTPError) bool {
	for err != nil {
		if he, ok := err.(*client.HTTPError); ok {
			*target = he
			return true
		}
		type unwrapper interface{ Unwrap() error }
		if u, ok := err.(unwrapper); ok {
			err = u.Unwrap()
		} else {
			break
		}
	}
	return false
}
