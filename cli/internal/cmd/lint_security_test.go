package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLintBoundsFileReads(t *testing.T) {
	for _, size := range []int64{32 << 20, (32 << 20) + 1} {
		path := filepath.Join(t.TempDir(), "large.json")
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		err = f.Truncate(size)
		if closeErr := f.Close(); closeErr != nil {
			t.Fatal(closeErr)
		}
		if err != nil {
			t.Fatal(err)
		}
		out, stderr, code := runCmd("lint", path)
		if size > 32<<20 {
			if code != exitUsage || out != "" || !strings.Contains(stderr, "32 MiB limit") {
				t.Fatalf("oversized file reached parsing: code=%d stderr=%s", code, stderr)
			}
		} else if code != exitLint || strings.Contains(stderr, "32 MiB limit") {
			// The file fits but contains NUL bytes, so JSON lint must reject it.
			t.Fatalf("boundary file did not reach JSON validation: code=%d stderr=%s", code, stderr)
		}
	}
}
