package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Capture skill guidance, launchers, checksums, and the evaluation inputs. A Git
// revision alone cannot identify an uncommitted candidate or a resumed run.
func evidenceSources(repository string) (string, map[string]string, error) {
	files := map[string]string{}
	hash := sha256.New()
	for _, dir := range []string{"skills", "ci/evals/skills-e2e"} {
		err := filepath.WalkDir(filepath.Join(repository, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "runs" || d.Name() == "__pycache__" || strings.HasPrefix(d.Name(), "bin.") {
					return filepath.SkipDir
				}
				return nil
			}
			if d.Name() == ".DS_Store" {
				return nil
			}
			if d.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("unexpected symlink in evaluation sources")
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(repository, path)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(b)
			name := filepath.ToSlash(rel)
			files[name] = hex.EncodeToString(sum[:])
			fmt.Fprintf(hash, "%s\x00%s\n", name, files[name])
			return nil
		})
		if err != nil {
			return "", nil, err
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), files, nil
}

func (h *harness) captureSources() error {
	repository := filepath.Clean(filepath.Join(h.root, "..", "..", ".."))
	digest, files, err := evidenceSources(repository)
	if err != nil {
		return err
	}
	h.sourceDigest = digest
	if len(h.state.EvidenceSources) == 0 && h.calls > 0 {
		h.state.EvidenceSources = append(h.state.EvidenceSources, "legacy-unrecorded")
	}
	for _, prior := range h.state.EvidenceSources {
		if prior == digest {
			return nil
		}
	}
	if err := write(filepath.Join(h.dir, "provenance", digest+".json"), map[string]any{
		"sha256": digest, "files": files, "capturedAt": time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		return err
	}
	h.state.EvidenceSources = append(h.state.EvidenceSources, digest)
	return nil
}
