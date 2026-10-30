package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// The harness sends JSON on stdin. Its small argument vocabulary deliberately
// excludes quotes and expansion characters rather than accepting cmd.exe code.
func runCLI(command *exec.Cmd) error {
	var quoted []string
	for _, arg := range command.Args {
		if strings.ContainsAny(arg, "\"%\r\n\x00") {
			return fmt.Errorf("unsupported Windows harness argument; pass payloads through stdin")
		}
		// Always quote metacharacters. Double trailing slashes for the native
		// Windows argv parser, which otherwise consumes the closing quote.
		suffix := len(arg) - len(strings.TrimRight(arg, "\\"))
		quoted = append(quoted, "\""+arg+strings.Repeat("\\", suffix)+"\"")
	}
	command.Path = filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe")
	command.SysProcAttr = &syscall.SysProcAttr{CmdLine: "cmd.exe /d /v:off /s /c \"" + strings.Join(quoted, " ") + "\""}
	return command.Run()
}
