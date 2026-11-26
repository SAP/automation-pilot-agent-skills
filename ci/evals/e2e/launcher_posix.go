//go:build !windows

package e2e

import "os/exec"

func runCLI(command *exec.Cmd) error { return command.Run() }
