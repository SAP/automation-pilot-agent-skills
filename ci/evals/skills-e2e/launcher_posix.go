//go:build !windows

package main

import "os/exec"

func runCLI(command *exec.Cmd) error { return command.Run() }
