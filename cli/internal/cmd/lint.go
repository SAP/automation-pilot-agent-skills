package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/SAP/automation-pilot-agent-skills/cli/internal/lint"
)

func printLintHelp(c *cmdContext) {
	w := c.stdout
	fmt.Fprintln(w, "autopi lint — validate command/input/catalog JSON (mechanical checks)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "USAGE")
	fmt.Fprintln(w, "  autopi lint [--output json] <files...>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Checks structure, naming, the sensitive flag on credential keys, and a few")
	fmt.Fprintln(w, "expression heuristics. Exits 6 when any finding is reported.")
}

func runLint(c *cmdContext, args []string) error {
	fs := flag.NewFlagSet("lint", flag.ContinueOnError)
	fs.SetOutput(c.stderr)
	var output string
	fs.StringVar(&output, "output", "text", "output format: text or json")
	fs.Usage = func() { printLintHelp(c) }
	if err := fs.Parse(args); err != nil {
		return &usageError{msg: err.Error()}
	}
	files := fs.Args()
	if len(files) == 0 {
		return usagef("usage: autopi lint <files...>")
	}

	var all []lint.Finding
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return usagef("cannot read %s: %v", f, err)
		}
		all = append(all, lint.CheckFile(f, b)...)
	}

	if output == "json" {
		enc := json.NewEncoder(c.stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(all); err != nil {
			return err
		}
	} else {
		if len(all) == 0 {
			fmt.Fprintln(c.stdout, "OK — no findings")
		}
		for _, f := range all {
			loc := f.File
			if f.Path != "" {
				loc += " [" + f.Path + "]"
			}
			fmt.Fprintf(c.stdout, "%-8s %s: %s\n", f.Severity, loc, f.Message)
		}
	}

	if len(all) > 0 {
		return &lintFindings{n: len(all)}
	}
	return nil
}
