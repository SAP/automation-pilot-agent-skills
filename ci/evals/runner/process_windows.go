//go:build windows

package main

import "os/exec"

// Evaluated sessions are unsupported on Windows until an isolation adapter is
// provided. Keep offline bank/grade/report commands buildable.
func containProcess(cmd *exec.Cmd) {}
