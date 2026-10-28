package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/SAP/automation-pilot-agent-skills/cli/internal/contract"
	"github.com/SAP/automation-pilot-agent-skills/cli/internal/lint"
)

func printLintHelp(c *cmdContext) {
	w := c.stdout
	fmt.Fprintln(w, "autopi lint — API request schemas and repository authoring policy")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "USAGE")
	fmt.Fprintln(w, "  autopi lint [--kind KIND] [--mode authoring|api] [--output text|json] [--read-back] [--python PATH] <files...>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Checks structure, naming, the sensitive flag on credential keys, and a few")
	fmt.Fprintln(w, "catalog prefixes, retry enums, collection bounds, and expression heuristics.")
	fmt.Fprintln(w, "--python PATH optionally parses inline Python without executing it.")
	fmt.Fprintln(w, "--kind selects full definitions, catalog-update, input-patch, bulk envelopes, actions, schedules, or webhooks.")
	fmt.Fprintln(w, "--mode api validates the request contract; default authoring adds naming/sensitivity/expression policy for content.")
	fmt.Fprintln(w, "Exits 6 when any finding is reported. Normal lint has no runtime dependencies.")
}

func runLint(c *cmdContext, args []string) error {
	fs := flag.NewFlagSet("lint", flag.ContinueOnError)
	fs.SetOutput(c.stderr)
	var output string
	var readBack bool
	var python string
	var kind, mode string
	fs.StringVar(&output, "output", "text", "output format: text or json")
	fs.BoolVar(&readBack, "read-back", false, "validate a retrieved command; permit atomic configuration and service tags")
	fs.StringVar(&python, "python", "", "optional Python 3 interpreter for inline syntax checking")
	fs.StringVar(&kind, "kind", "", "explicit request kind: "+strings.Join(contract.Kinds(), ", ")+", mcp-server")
	fs.StringVar(&mode, "mode", "authoring", "authoring (API checks plus repository policy) or api (request schema only)")
	fs.Usage = func() { printLintHelp(c) }
	files, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if output != "text" && output != "json" {
		return usagef("invalid --output (want text or json)")
	}
	if len(files) == 0 {
		return usagef("usage: autopi lint <files...>")
	}
	if kind != "" && !contract.KnownKind(kind) && kind != "mcp-server" {
		return usagef("unknown --kind")
	}
	if mode != "api" && mode != "authoring" {
		return usagef("--mode must be authoring or api")
	}
	if mode == "api" && kind == "mcp-server" {
		return usagef("MCP has authoring/binding validation; it is outside the pinned API contracts")
	}
	if python != "" {
		python, err = exec.LookPath(python)
		if err != nil {
			return usagef("cannot find the selected Python 3 interpreter")
		}
	}

	all := []lint.Finding{}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return usagef("cannot read %s: %v", f, err)
		}
		all = append(all, lint.CheckFileOptions(f, b, readBack, kind, mode)...)
		if python != "" {
			if doc, _, parseErr := lint.Parse(b); parseErr == nil {
				findings, err := lint.CheckPythonSyntax(f, doc.Fields, python)
				if err != nil {
					return usagef("%s: %v", f, err)
				}
				all = append(all, findings...)
			}
		}
	}

	if output == "json" {
		enc := json.NewEncoder(c.stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(all); err != nil {
			return err
		}
	} else {
		if len(all) == 0 {
			if _, err := fmt.Fprintln(c.stdout, "OK — no findings"); err != nil {
				return err
			}
		}
		for _, f := range all {
			loc := f.File
			if f.Path != "" {
				loc += " [" + f.Path + "]"
			}
			if _, err := fmt.Fprintf(c.stdout, "%-8s %s: %s\n", f.Severity, loc, f.Message); err != nil {
				return err
			}
		}
	}

	if len(all) > 0 {
		return &lintFindings{n: len(all)}
	}
	return nil
}
