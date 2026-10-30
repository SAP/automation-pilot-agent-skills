package cmd

import (
	"errors"
	"fmt"
	"io"

	"github.com/SAP/automation-pilot-agent-skills/cli/internal/client"
	"github.com/SAP/automation-pilot-agent-skills/cli/internal/contract"
)

// Exit codes. The scheme mirrors curl's success/transport/HTTP split so the
// debugger skill's "no response vs error" guidance maps cleanly onto autopi.
const (
	exitOK       = 0 // success
	exitUsage    = 1 // bad flags / unknown subcommand
	exitConfig   = 2 // missing or invalid credentials
	exitNetwork  = 3 // DNS / TLS / timeout / transport failure
	exitHTTP4xx  = 4 // HTTP 4xx from the API
	exitHTTP5xx  = 5 // HTTP 5xx from the API
	exitLint     = 6 // lint findings present
	exitPartial  = 7 // bulk operation contains failed items
	exitInternal = 1 // unexpected internal error (treated as usage)
)

// usageError marks an argument/flag problem so Run can map it to exitUsage and
// print to stderr.
type usageError struct{ msg string }

func (e *usageError) Error() string { return e.msg }

func usagef(format string, a ...any) error { return &usageError{msg: fmt.Sprintf(format, a...)} }

// lintFindings signals that lint ran successfully but found issues.
type lintFindings struct{ n int }
type partialFailure struct{ n int }

func (e *partialFailure) Error() string {
	return fmt.Sprintf("%d bulk action item(s) failed; inspect results before retrying", e.n)
}

func (e *lintFindings) Error() string { return fmt.Sprintf("%d finding(s)", e.n) }

// codeFor maps an error returned by a verb to a process exit code and writes a
// human-readable message to stderr. nil → exitOK.
func codeFor(err error, stderr io.Writer) int {
	if err == nil {
		return exitOK
	}
	if errors.Is(err, errHelpRequested) {
		return exitOK
	}
	var ce *client.ConfigError
	var he *client.HTTPError
	var ne *client.NetworkError
	var ue *usageError
	var lf *lintFindings
	var invalid *contract.InvalidRequest
	var partial *partialFailure
	switch {
	case errors.As(err, &invalid):
		fmt.Fprintln(stderr, "error:", invalid.Error())
		return exitUsage
	case errors.As(err, &partial):
		fmt.Fprintln(stderr, partial.Error())
		return exitPartial
	case errors.As(err, &ue):
		fmt.Fprintln(stderr, "error:", ue.Error())
		return exitUsage
	case errors.As(err, &ce):
		fmt.Fprintln(stderr, "error:", ce.Error())
		return exitConfig
	case errors.As(err, &ne):
		fmt.Fprintln(stderr, "error:", ne.Error())
		return exitNetwork
	case errors.As(err, &lf):
		fmt.Fprintln(stderr, lf.Error())
		return exitLint
	case errors.As(err, &he):
		fmt.Fprintln(stderr, "error:", he.Error())
		if he.Status >= 500 {
			return exitHTTP5xx
		}
		return exitHTTP4xx
	default:
		fmt.Fprintln(stderr, "error:", err.Error())
		return exitInternal
	}
}
