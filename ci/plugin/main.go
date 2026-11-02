// plugin stages a self-contained distribution and checks installed skill assets.
package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var links = regexp.MustCompile("!?\\[[^]\\n]*\\]\\(([^ )]+)(?: [^)]*)?\\)")

func main() {
	if len(os.Args) < 3 {
		fatal(fmt.Errorf("usage: plugin stage|stage-evals ROOT DEST | check PLUGIN | version ROOT | zip PLUGIN OUTPUT.zip"))
	}
	var err error
	switch os.Args[1] {
	case "stage", "stage-evals":
		if len(os.Args) != 4 {
			fatal(fmt.Errorf("stage requires root and new destination"))
		}
		err = stage(os.Args[2], os.Args[3], os.Args[1] == "stage-evals")
	case "version":
		var raw []byte
		raw, err = os.ReadFile(filepath.Join(os.Args[2], ".claude-plugin", "plugin.json"))
		if err == nil {
			var manifest struct{ Name, Version string }
			err = json.Unmarshal(raw, &manifest)
			if err == nil {
				if manifest.Name != "automation-pilot-skills" || manifest.Version == "" {
					err = fmt.Errorf("invalid plugin identity")
				} else {
					fmt.Println(manifest.Version)
				}
			}
		}
	case "check":
		err = check(os.Args[2])
	case "zip":
		if len(os.Args) != 4 {
			fatal(fmt.Errorf("zip requires plugin and output path"))
		}
		err = archive(os.Args[2], os.Args[3])
	default:
		err = fmt.Errorf("unknown operation")
	}
	fatal(err)
}

func fatal(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func stage(root, dest string, evaluations bool) error {
	for _, name := range []string{".claude-plugin", "skills", "ci/evals/agent-skills", "LICENSES"} {
		if within(filepath.Join(root, name), dest) {
			return fmt.Errorf("stage destination must be outside copied source directories")
		}
	}
	if err := os.Mkdir(dest, 0755); err != nil {
		return err
	}
	assets := []string{".claude-plugin", "skills", "LICENSE", "LICENSES", "REUSE.toml", "README.md", "SECURITY.md", "CONTRIBUTING.md", "sap-automation-pilot.svg", "overview.svg"}
	if evaluations {
		assets = append(assets, "ci/evals/agent-skills")
	}
	for _, name := range assets {
		source := filepath.Join(root, name)
		if err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.Name() == ".DS_Store" || entry.Name() == "__pycache__" || entry.Name() == "results" || strings.HasPrefix(entry.Name(), "bin.") {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("plugin assets must be regular files: %s", path)
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			target := filepath.Join(dest, rel)
			if entry.IsDir() {
				return os.MkdirAll(target, 0755)
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("non-regular asset: %s", path)
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(target, b, info.Mode().Perm())
		}); err != nil {
			return err
		}
	}
	if evaluations {
		path := filepath.Join(dest, ".claude-plugin", "plugin.json")
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var manifest map[string]any
		if err := json.Unmarshal(raw, &manifest); err != nil {
			return err
		}
		manifest["experimental"] = map[string]string{"evals": "ci/evals/agent-skills"}
		raw, err = json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, append(raw, '\n'), 0644); err != nil {
			return err
		}
	}
	return check(dest)
}

func check(root string) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	manifest, err := os.ReadFile(filepath.Join(root, ".claude-plugin", "plugin.json"))
	if err != nil {
		return err
	}
	var plugin struct{ Name, Version string }
	if err := json.Unmarshal(manifest, &plugin); err != nil {
		return err
	}
	if plugin.Name != "automation-pilot-skills" || plugin.Version == "" {
		return fmt.Errorf("invalid plugin identity")
	}
	var count int
	err = filepath.WalkDir(filepath.Join(root, "skills"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("external/symlink skill asset: %s", path)
		}
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if d.Name() == "SKILL.md" {
			count++
			if !strings.HasPrefix(string(b), "---\nname: "+filepath.Base(filepath.Dir(path))+"\n") || !strings.Contains(string(b), "\ndescription:") {
				return fmt.Errorf("missing skill identity/description: %s", path)
			}
		}
		for _, match := range links.FindAllStringSubmatch(string(b), -1) {
			target := strings.SplitN(match[1], "#", 2)[0]
			if target == "" || strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
				continue
			}
			full := filepath.Clean(filepath.Join(filepath.Dir(path), target))
			if !strings.HasPrefix(full, root+string(filepath.Separator)) {
				return fmt.Errorf("skill reference escapes plugin: %s", target)
			}
			if _, err := os.Stat(full); err != nil {
				return fmt.Errorf("%s: unavailable installed reference %s", path, target)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if count != 12 {
		return fmt.Errorf("expected all 12 skills, found %d", count)
	}
	bin := filepath.Join(root, "skills", "automation-pilot-cli", "scripts", "bin")
	data, err := os.ReadFile(filepath.Join(bin, "SHA256SUMS"))
	if err != nil {
		return err
	}
	expected := map[string]bool{"darwin-amd64/autopi": true, "darwin-arm64/autopi": true, "linux-amd64/autopi": true, "linux-arm64/autopi": true, "windows-amd64/autopi.exe": true}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		parts := strings.Split(line, "  ")
		if len(parts) != 2 || !expected[parts[1]] {
			return fmt.Errorf("unexpected or duplicate checksum entry")
		}
		b, err := os.ReadFile(filepath.Join(bin, filepath.FromSlash(parts[1])))
		if err != nil {
			return err
		}
		hash := sha256.Sum256(b)
		if hex.EncodeToString(hash[:]) != parts[0] {
			return fmt.Errorf("checksum mismatch for %s", parts[1])
		}
		delete(expected, parts[1])
	}
	if len(expected) != 0 {
		return fmt.Errorf("incomplete binary bundle")
	}
	fmt.Println("All 12 skills, local references, and five binary checksums verified")
	return nil
}

// ZIP entries are sorted by WalkDir, preserve executable permissions, and use
// fixed timestamps. Identical plugin contents produce identical archives.
func archive(root, output string) (err error) {
	if within(root, output) {
		return fmt.Errorf("archive output must be outside the plugin directory")
	}
	if err := check(root); err != nil {
		return err
	}
	f, err := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() {
		if e := f.Close(); err == nil {
			err = e
		}
	}()
	z := zip.NewWriter(f)
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("non-regular archive asset")
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		h := &zip.FileHeader{Name: filepath.ToSlash(rel), Method: zip.Deflate}
		h.SetMode(info.Mode().Perm())
		w, err := z.CreateHeader(h)
		if err != nil {
			return err
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		defer input.Close()
		_, err = io.Copy(w, input)
		return err
	})
	if e := z.Close(); err == nil {
		err = e
	}
	return err
}

func within(root, path string) bool {
	root, err := filepath.Abs(root)
	if err != nil {
		return true
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return true
	}
	return path == root || strings.HasPrefix(path, root+string(filepath.Separator))
}
