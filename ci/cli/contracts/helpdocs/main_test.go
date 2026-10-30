package main

import (
	"errors"
	"strings"
	"testing"
)

func TestRenderCoversEveryVerbAndSpecificFlags(t *testing.T) {
	calls := 0
	out, err := render(func(args ...string) ([]byte, error) {
		calls++
		if args[0] == "surface" {
			return []byte(`[{"name":"commands","verbs":["get","list"]}]`), nil
		}
		return []byte("CALLS\n  GET /commands | GET /other\n  -output string\n  -f string\n  -tag string\n  -tag string\n"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if calls != 3 || !strings.Contains(s, "| \x60get\x60 | GET /commands \\| GET /other | \x60-f\x60, \x60--tag\x60 |") || !strings.Contains(s, "| \x60list\x60 |") {
		t.Fatalf("incomplete or incorrect help: %s", s)
	}
	if strings.Contains(s, "~BT~") {
		t.Fatal("unrendered template")
	}
}
func TestRenderFailsOnMissingCLIHelp(t *testing.T) {
	_, err := render(func(args ...string) ([]byte, error) {
		if args[0] == "surface" {
			return []byte(`[{"name":"commands","verbs":["get"]}]`), nil
		}
		return nil, errors.New("launcher rejected bundle")
	})
	if err == nil {
		t.Fatal("failed CLI invocation accepted")
	}
}
