package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/SAP/automation-pilot-agent-skills/cli/internal/client"
)

// printOverview renders the top-level help: global flags, env vars, resources,
// the raw escape hatch, and exit codes. Kept to roughly one screen.
func printOverview(c *cmdContext) {
	w := c.stdout
	fmt.Fprintln(w, "autopi — unified CLI for the SAP Automation Pilot REST API")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "USAGE")
	fmt.Fprintln(w, "  autopi <resource> <verb> [flags]")
	fmt.Fprintln(w, "  autopi api <METHOD> <path> [--data @file|json]   # raw escape hatch")
	fmt.Fprintln(w, "  autopi surface                                  # machine-readable resource verbs")
	fmt.Fprintln(w, "  autopi policy                                   # shared product rule snapshot")
	fmt.Fprintln(w, "  autopi lint <files...>                           # validate command/input JSON")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "RESOURCES")
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	for _, r := range registry() {
		fmt.Fprintf(tw, "  %s\t%s\n", r.name, r.summary)
	}
	tw.Flush()
	fmt.Fprintln(w)
	fmt.Fprintln(w, "ENVIRONMENT")
	fmt.Fprintln(w, "  AUTOMATION_PILOT_HOSTNAME   regional host, e.g. emea.autopilot.cloud.sap")
	fmt.Fprintln(w, "  AUTOMATION_PILOT_TOKEN      Bearer token without the prefix (read from env; never logged)")
	fmt.Fprintln(w, "  Connection configuration is environment-only.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  Supported hosts:")
	for _, h := range client.SupportedHosts {
		fmt.Fprintf(w, "    %s\n", h)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "GLOBAL FLAGS")
	fmt.Fprintln(w, "  --catalog <id>                       filter catalog-scoped list/ids verbs only")
	fmt.Fprintln(w, "  --output json|raw                    pretty JSON (default) or verbatim")
	fmt.Fprintln(w, "  --print-request                      render a redacted request, do not send")
	fmt.Fprintln(w, "  --timeout 30s   --insecure")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "EXIT CODES")
	fmt.Fprintln(w, "  0 ok  1 usage  2 config  3 network  4 http-4xx  5 http-5xx  6 lint-findings")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run 'autopi <resource> --help' for verbs, or 'autopi <resource> <verb> --help' for flags.")
}

func printResourceHelp(c *cmdContext, r *resource) {
	w := c.stdout
	fmt.Fprintf(w, "autopi %s — %s\n\n", r.name, r.summary)
	fmt.Fprintln(w, "VERBS")
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	for _, v := range r.verbs {
		fmt.Fprintf(tw, "  %s\t%s\t%s\n", v.name, v.endpoint, v.summary)
	}
	tw.Flush()
	fmt.Fprintf(w, "\nRun 'autopi %s <verb> --help' for flags.\n", r.name)
}

// runHelp handles `autopi help [resource [verb]]`.
func runHelp(c *cmdContext, args []string) error {
	if len(args) > 2 {
		return usagef("usage: autopi help [resource [verb]]")
	}
	if len(args) == 0 {
		printOverview(c)
		return nil
	}
	r := findResource(args[0])
	if r == nil {
		switch args[0] {
		case "api":
			printAPIHelp(c)
			return nil
		case "lint":
			printLintHelp(c)
			return nil
		}
		return usagef("unknown help topic %q", args[0])
	}
	if len(args) >= 2 {
		if v := r.verb(args[1]); v != nil {
			// Invoke the verb with --help so it prints its own flag usage.
			return v.run(c, []string{"--help"})
		}
		return usagef("unknown %s subcommand %q", r.name, args[1])
	}
	printResourceHelp(c, r)
	return nil
}
