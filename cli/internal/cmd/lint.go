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
	fmt.Fprintln(w, "  autopi lint [--output text|json] [--read-back] <files...>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Checks structure, naming, the sensitive flag on credential keys, and a few")
	fmt.Fprintln(w, "expression heuristics. Exits 6 when any finding is reported.")
}

func runLint(c *cmdContext, args []string) error {
	fs := flag.NewFlagSet("lint", flag.ContinueOnError)
	fs.SetOutput(c.stderr)
	var output string
	var readBack bool
	fs.StringVar(&output, "output", "text", "output format: text or json")
	fs.BoolVar(&readBack, "read-back", false, "validate a retrieved command; permit atomic configuration and service tags")
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

	all := []lint.Finding{}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return usagef("cannot read %s: %v", f, err)
		}
		all = append(all, lint.CheckFileMode(f, b, readBack)...)
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
