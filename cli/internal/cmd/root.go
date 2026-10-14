package cmd

import (
	"fmt"
	"os"
)

// registry is the ordered list of API resources. api, lint, version, and help
// are handled directly in Run.
func registry() []*resource {
	return []*resource{
		catalogsResource(),
		commandsResource(),
		inputsResource(),
		mcpServersResource(),
		a2aAdaptersResource(),
		executionsResource(),
		scheduledResource(),
		triggersResource(),
	}
}

func findResource(name string) *resource {
	for _, r := range registry() {
		if r.name == name {
			return r
		}
	}
	return nil
}

// Run is the CLI entrypoint. It returns the process exit code.
func Run(args []string, version string) int {
	c := &cmdContext{stdout: os.Stdout, stderr: os.Stderr, version: version}
	return dispatch(c, args)
}

// dispatch resolves the resource/verb and runs it against the given context.
// Separated from Run so tests can supply buffers.
func dispatch(c *cmdContext, args []string) int {
	if len(args) == 0 {
		printOverview(c)
		return exitOK
	}

	switch args[0] {
	case "-h", "--help", "help":
		return codeFor(runHelp(c, args[1:]), c.stderr)
	case "-v", "--version", "version":
		fmt.Fprintf(c.stdout, "autopi %s\n", c.version)
		return exitOK
	case "api":
		return codeFor(runAPI(c, args[1:]), c.stderr)
	case "lint":
		return codeFor(runLint(c, args[1:]), c.stderr)
	}

	r := findResource(args[0])
	if r == nil {
		fmt.Fprintf(c.stderr, "error: unknown command %q\n\n", args[0])
		printOverview(c)
		return exitUsage
	}

	if len(args) == 1 {
		printResourceHelp(c, r)
		return exitUsage
	}
	verbName := args[1]
	if verbName == "-h" || verbName == "--help" {
		printResourceHelp(c, r)
		return exitOK
	}
	v := r.verb(verbName)
	if v == nil {
		fmt.Fprintf(c.stderr, "error: unknown %s subcommand %q\n\n", r.name, verbName)
		printResourceHelp(c, r)
		return exitUsage
	}
	return codeFor(v.run(c, args[2:]), c.stderr)
}
