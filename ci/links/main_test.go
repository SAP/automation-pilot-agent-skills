package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnchorsAndCodeExclusion(t *testing.T) {
	s := "# Same\n# Same\n# Same-1\n# Héllo &amp; 世界!\n~~~\n# Hidden\n~~~\n"
	a := anchors(s)
	for _, want := range []string{"same", "same-1", "same-1-1", "héllo--世界"} {
		if !a[want] {
			t.Fatalf("missing anchor %q in %v", want, a)
		}
	}
	if a["hidden"] {
		t.Fatal("fenced heading counted")
	}
	if got := stripCode("before \x60[ignored](x)\x60 after"); got != "before  after" {
		t.Fatal(got)
	}
}
func TestChecksAllPublicGuidanceAndRejectsPrivateLinks(t *testing.T) {
	root := t.TempDir()
	write := func(name, text string) {
		t.Helper()
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"README.md", "CONTRIBUTING.md", "SECURITY.md", "skills/a/SKILL.md"} {
		write(name, "# Guidance\n")
	}
	write("ci/.local/notes.md", "[ignored](missing.md)\n")
	write("ci/guide.md", "[valid](../README.md#guidance)\n![image](<image name.svg>)\n")
	write("ci/image name.svg", "<svg/>")
	count, errs := check(root)
	if count != 2 || len(errs) != 0 {
		t.Fatalf("%d %v", count, errs)
	}
	write("ci/guide.md", "[missing](../README.md#absent)\n[private](.local/notes.md)\n")
	_, errs = check(root)
	if len(errs) != 2 || !strings.Contains(strings.Join(errs, " "), "private local evidence") {
		t.Fatal(errs)
	}
}
