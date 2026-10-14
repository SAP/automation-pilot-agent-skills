package cmd

import (
	"fmt"
	"strings"

	"github.com/SAP/automation-pilot-agent-skills/cli/internal/client"
)

func printAPIHelp(c *cmdContext) {
	w := c.stdout
	fmt.Fprintln(w, "autopi api — raw request to any /api/v1 endpoint")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "USAGE")
	fmt.Fprintln(w, "  autopi api <METHOD> <path> [--data @file|json] [--header 'K: V'] [--content-type CT] [--status-only]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "The path is appended to https://<host>/api/v1; a leading /api/v1 is tolerated.")
	fmt.Fprintln(w, "Basic auth is injected automatically. Honors --print-request and --output.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "EXAMPLE")
	fmt.Fprintln(w, "  autopi api GET 'commands?catalog=applm-sapcp'")
	fmt.Fprintln(w, `  autopi api POST executions --data @trigger.json`)
}

func runAPI(c *cmdContext, args []string) error {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		printAPIHelp(c)
		return nil
	}
	fs, g := newFlagSet(c, "api")
	var data, contentType string
	var header stringList
	var statusOnly bool
	fs.StringVar(&data, "data", "", "request body: inline JSON or @file")
	fs.StringVar(&contentType, "content-type", "", "Content-Type header (default application/json when a body is set)")
	fs.Var(&header, "header", "extra header 'Key: Value' (repeatable)")
	fs.BoolVar(&statusOnly, "status-only", false, "print only the HTTP status code")
	fs.Usage = func() { printAPIHelp(c) }
	pos, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(pos) < 2 {
		return usagef("usage: autopi api <METHOD> <path> [flags]")
	}
	method := strings.ToUpper(pos[0])
	path := pos[1]

	var body []byte
	if data != "" {
		body, err = resolveBody("", data)
		if err != nil {
			return err
		}
	}
	headers := map[string]string{}
	for _, h := range header {
		k, v, ok := strings.Cut(h, ":")
		if !ok {
			return usagef("invalid --header %q (want 'Key: Value')", h)
		}
		headers[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}
	return execute(c, cfg, client.Request{
		Method:      method,
		Path:        path,
		Body:        body,
		ContentType: contentType,
		Headers:     headers,
	}, statusOnly)
}
