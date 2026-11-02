// helpdocs generates the command reference through the verified CLI launcher.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

type resource struct {
	Name  string
	Verbs []string
}

var common = map[string]bool{"output": true, "timeout": true, "print-request": true, "insecure": true, "response-envelope": true, "read-retries": true}
var callPattern = regexp.MustCompile(`CALLS\n\s+([^\n]+)`)
var flagPattern = regexp.MustCompile(`(?m)^  -([a-z][a-z-]*)`)

const intro = `# autopi command reference

Generated from the bundled CLI. Run
~BT~autopi <resource> <verb> --help~BT~ for usage, defaults, and examples.

For wire types, limits, permissions, HTTP outcomes, and schema exceptions, read
[API contracts](api-contract.md). For workflows, read [recipes](recipes.md).

`
const shared = `## Shared options

| Option | Behavior |
|---|---|
| ~BT~--output json/raw~BT~ | Pretty JSON or verbatim body |
| ~BT~--response-envelope~BT~ | JSON status, selected headers, and body; mutually exclusive with raw/status/ETag selectors |
| ~BT~--print-request~BT~ | Redacted offline preview; no API calls |
| ~BT~--timeout~BT~ | Total request deadline, default 30s |
| ~BT~--read-retries N~BT~ | GET/HEAD 429/503 only, 0–5 retries within the same deadline |
| ~BT~--insecure~BT~ | Explicit TLS-verification escape hatch |

Bodies support ~BT~-f FILE~BT~, ~BT~-f -~BT~, ~BT~--data JSON~BT~, ~BT~--data @file~BT~, and ~BT~--data @-~BT~.
Bearer authentication is the only supported mode; connection credentials are environment-only.
See the [CLI entrypoint](../SKILL.md) for setup and response protection.

`
const ending = `## Raw API and validation

~BT~~BT~~BT~text
autopi api METHOD PATH [-f FILE|- | --data JSON|@file|@-] [--header K:V] [--content-type TYPE]
autopi lint [--kind KIND] [--mode authoring|api] [--read-back] [--output text|json] FILE...
autopi policy
autopi surface
~BT~~BT~~BT~

Raw API preserves the configured authentication and documented output protection,
and bypasses the pinned request validator. Custom Authorization is rejected.
Lint separates API request requirements from naming/sensitivity/expression policy.
Exit 6 means lint findings; exit 7 means one or more bulk action items failed.
`

func render(run func(...string) ([]byte, error)) ([]byte, error) {
	raw, err := run("surface")
	if err != nil {
		return nil, err
	}
	var resources []resource
	if err = json.Unmarshal(raw, &resources); err != nil {
		return nil, err
	}
	if len(resources) == 0 {
		return nil, fmt.Errorf("empty CLI surface")
	}
	var out strings.Builder
	out.WriteString(intro)
	out.WriteString("## Contents\n\n- [Shared options](#shared-options)\n")
	for _, r := range resources {
		fmt.Fprintf(&out, "- [%s](#%s)\n", r.Name, r.Name)
	}
	out.WriteString("- [Raw API and validation](#raw-api-and-validation)\n\n")
	out.WriteString(shared)
	for _, r := range resources {
		fmt.Fprintf(&out, "## %s\n\n| Verb | Calls | Specific flags |\n|---|---|---|\n", r.Name)
		for _, verb := range r.Verbs {
			help, err := run(r.Name, verb, "--help")
			if err != nil {
				return nil, err
			}
			call := "See help"
			if m := callPattern.FindSubmatch(help); len(m) > 1 {
				call = strings.ReplaceAll(string(m[1]), "|", `\|`)
			}
			unique := map[string]bool{}
			for _, m := range flagPattern.FindAllSubmatch(help, -1) {
				f := string(m[1])
				if !common[f] {
					unique[f] = true
				}
			}
			flags := []string{}
			for f := range unique {
				flags = append(flags, f)
			}
			sort.Strings(flags)
			for i, f := range flags {
				prefix := "--"
				if len(f) == 1 {
					prefix = "-"
				}
				flags[i] = "\x60" + prefix + f + "\x60"
			}
			fmt.Fprintf(&out, "| \x60%s\x60 | %s | %s |\n", verb, call, strings.Join(flags, ", "))
		}
		out.WriteString("\n")
	}
	out.WriteString(ending)
	return []byte(strings.ReplaceAll(out.String(), "~BT~", "\x60")), nil
}
func main() {
	root := flag.String("root", ".", "repository or installed plugin root")
	check := flag.Bool("check", false, "fail if generated help differs")
	flag.Parse()
	if flag.NArg() != 0 {
		fatal(fmt.Errorf("unexpected arguments"))
	}
	absolute, err := filepath.Abs(*root)
	fatal(err)
	binary := filepath.Join(absolute, "skills", "automation-pilot-cli", "scripts", "autopi")
	if runtime.GOOS == "windows" {
		fatal(fmt.Errorf("help generation requires a macOS or Linux maintenance host; Windows skill usage uses the prebuilt autopi.cmd launcher"))
	}
	out, err := render(func(args ...string) ([]byte, error) {
		command := exec.Command(binary, args...)
		raw, err := command.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("CLI help failed: %w: %s", err, raw)
		}
		return raw, nil
	})
	fatal(err)
	path := filepath.Join(absolute, "skills", "automation-pilot-cli", "references", "command-reference.md")
	if *check {
		current, err := os.ReadFile(path)
		fatal(err)
		if !bytes.Equal(current, out) {
			fatal(fmt.Errorf("command reference differs from CLI help; regenerate it"))
		}
	} else {
		fatal(os.WriteFile(path, out, 0644))
	}
	fmt.Println("Command reference and complete help surface verified")
}
func fatal(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
