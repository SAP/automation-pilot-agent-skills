package cmd

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/SAP/automation-pilot-agent-skills/cli/internal/client"
)

// cmdContext carries process-wide I/O and metadata through the dispatch tree.
type cmdContext struct {
	stdout  io.Writer
	stderr  io.Writer
	version string
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

// globalFlags are accepted by every verb; they override the AUTOPI_* env vars.
type globalFlags struct {
	host, user, password, catalog string
	output, timeout               string
	printRequest, insecure        bool
}

// bindGlobal registers the global flags on fs and returns the holder.
func bindGlobal(fs *flag.FlagSet) *globalFlags {
	g := &globalFlags{}
	fs.StringVar(&g.host, "host", "", "API hostname (env AUTOPI_HOSTNAME)")
	fs.StringVar(&g.user, "user", "", "API username (env AUTOPI_USERNAME)")
	fs.StringVar(&g.password, "password", "", "API password (env AUTOPI_PASSWORD; prefer the env var)")
	fs.StringVar(&g.catalog, "catalog", "", "default catalog (env AUTOPI_DEFAULT_CATALOG)")
	fs.StringVar(&g.output, "output", "", "output format: json (default) or raw")
	fs.StringVar(&g.timeout, "timeout", "", "request timeout, e.g. 30s (env AUTOPI_TIMEOUT)")
	fs.BoolVar(&g.printRequest, "print-request", false, "render the request (auth masked) without sending it")
	fs.BoolVar(&g.insecure, "insecure", false, "skip TLS verification (escape hatch)")
	return g
}

// config merges env defaults with any explicitly set global flags.
func (g *globalFlags) config() (client.Config, error) {
	cfg := client.FromEnv()
	if g.host != "" {
		cfg.Host = g.host
	}
	if g.user != "" {
		cfg.User = g.user
	}
	if g.password != "" {
		cfg.Password = g.password
	}
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
		cfg.Timeout = d
	}
	cfg.PrintRequest = g.printRequest
	cfg.Insecure = g.insecure
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
	for {
		if err := fs.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil, errHelpRequested
			}
			return nil, &usageError{msg: err.Error()}
		}
		if fs.NArg() == 0 {
			break
		}
		positional = append(positional, fs.Arg(0))
		args = fs.Args()[1:]
	}
	return positional, nil
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
// a tag query is `key=value`, while the CLI accepts the `key:value` convention
// used everywhere else. A key-only filter (no colon) matches any value and is
// forwarded as the bare key. Without this translation the raw colon string is
// sent verbatim and the server silently matches nothing.
func addTagFilters(q url.Values, tags stringList) {
	for _, t := range tags {
		if k, v, ok := strings.Cut(t, ":"); ok {
			q.Add("tag", k+"="+v)
		} else {
			q.Add("tag", t)
		}
	}
}

// resolveBody returns the request body from either -f/--file (a path) or --data
// (inline JSON, or "@path" curl-style). Exactly one source may be set.
func resolveBody(file, data string) ([]byte, error) {
	switch {
	case file != "" && data != "":
		return nil, usagef("use either -f/--file or --data, not both")
	case file != "":
		return readFile(file)
	case strings.HasPrefix(data, "@"):
		return readFile(strings.TrimPrefix(data, "@"))
	case data != "":
		return []byte(data), nil
	default:
		return nil, usagef("a request body is required: pass -f FILE or --data JSON")
	}
}

func readFile(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, usagef("cannot read %s: %v", path, err)
	}
	return b, nil
}

// execute sends req (or renders it under --print-request) and writes the result.
// When statusOnly is set it prints just the HTTP status code (replacing the old
// `curl -o /dev/null -w "%{http_code}"` existence checks) and never errors on a
// non-2xx status.
func execute(c *cmdContext, cfg client.Config, req client.Request, statusOnly bool) error {
	cl := client.New(cfg, c.stdout)
	res, err := cl.Do(req)
	if statusOnly {
		var he *client.HTTPError
		if res != nil {
			fmt.Fprintln(c.stdout, res.Status)
			return nil
		}
		if asHTTP(err, &he) {
			fmt.Fprintln(c.stdout, he.Status)
			return nil
		}
		return err // config/network errors still surface
	}
	if err != nil {
		// Still render any error body the API returned before failing.
		if res != nil {
			cl.Render(c.stdout, res)
		}
		return err
	}
	if req.Summarize && res != nil {
		res.Body = client.Summarize(res.Body)
	}
	if req.FilterIDPrefix != "" && res != nil {
		res.Body = client.FilterIDPrefix(res.Body, req.FilterIDPrefix)
	}
	cl.Render(c.stdout, res)
	return nil
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
