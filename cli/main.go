// Command autopi is a single, dependency-free CLI for the SAP Automation Pilot
// REST API. It replaces the hand-written curl blocks previously scattered across
// the automation-pilot-* agent skills with one typed client that runs on macOS,
// Linux, and Windows from a statically linked binary (no runtime dependencies).
package main

import (
	"os"

	"github.com/SAP/automation-pilot-agent-skills/cli/internal/cmd"
)

func main() {
	// cmd.Run resolves the resource/verb from argv, executes it, and returns the
	// process exit code (see internal/cmd/exit.go for the code scheme).
	os.Exit(cmd.Run(os.Args[1:], Version))
}
