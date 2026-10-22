// buildmeta records and verifies the CLI's source and binary provenance.
package main

import (
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

type artifact struct {
	Target string `json:"target"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}
type manifest struct {
	Version      string     `json:"version"`
	Revision     string     `json:"revision"`
	SourceSHA256 string     `json:"sourceSHA256"`
	Toolchain    string     `json:"toolchain"`
	CGOEnabled   bool       `json:"cgoEnabled"`
	Flags        []string   `json:"flags"`
	Artifacts    []artifact `json:"artifacts"`
}

var targets = []string{"darwin-arm64", "darwin-amd64", "linux-amd64", "linux-arm64", "windows-amd64"}

func fail(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func sourceDigest(root string) (string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		name := filepath.Base(path)
		if strings.HasSuffix(name, ".go") || name == "go.mod" || name == "go.sum" || name == "build.sh" || name == ".go-version" || name == "Makefile" {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(paths)
	h := sha256.New()
	for _, path := range paths {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return "", err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(h, "%s\x00%d\x00", filepath.ToSlash(rel), len(b))
		h.Write(b)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func main() {
	if len(os.Args) < 3 {
		fail(fmt.Errorf("usage: buildmeta source MODULE | manifest MODULE BIN VERSION REVISION | verify MODULE BIN"))
	}
	mode, root := os.Args[1], os.Args[2]
	source, err := sourceDigest(root)
	fail(err)
	if mode == "source" {
		fmt.Println(source)
		return
	}
	if len(os.Args) < 4 {
		fail(fmt.Errorf("binary directory required"))
	}
	bin := os.Args[3]
	if mode == "verify" {
		var m manifest
		b, err := os.ReadFile(filepath.Join(bin, "build-manifest.json"))
		fail(err)
		fail(json.Unmarshal(b, &m))
		if m.SourceSHA256 != source {
			fail(fmt.Errorf("bundled binaries do not match CLI source digest"))
		}
		if m.CGOEnabled || len(m.Artifacts) != len(targets) {
			fail(fmt.Errorf("unexpected manifest build settings"))
		}
		for i, a := range m.Artifacts {
			if a.Target != targets[i] {
				fail(fmt.Errorf("unexpected target %s", a.Target))
			}
			b, err := os.ReadFile(filepath.Join(bin, a.Path))
			fail(err)
			info, err := buildinfo.ReadFile(filepath.Join(bin, a.Path))
			fail(err)
			settings := map[string]string{}
			for _, setting := range info.Settings {
				settings[setting.Key] = setting.Value
			}
			parts := strings.SplitN(a.Target, "-", 2)
			if info.GoVersion != m.Toolchain || settings["CGO_ENABLED"] != "0" || settings["-trimpath"] != "true" || settings["GOOS"] != parts[0] || settings["GOARCH"] != parts[1] {
				fail(fmt.Errorf("build metadata mismatch: %s", a.Path))
			}
			if digest(b) != a.SHA256 {
				fail(fmt.Errorf("checksum mismatch: %s", a.Path))
			}
		}
		fmt.Println("source and all five binary checksums match manifest")
		return
	}
	if mode != "manifest" || len(os.Args) != 6 {
		fail(fmt.Errorf("invalid buildmeta invocation"))
	}
	m := manifest{Version: os.Args[4], Revision: os.Args[5], SourceSHA256: source, Toolchain: runtime.Version(), Flags: []string{"-trimpath", "-buildvcs=false", "-s", "-w", "-X main.Version=" + os.Args[4]}}
	var sums strings.Builder
	for _, target := range targets {
		path := target + "/autopi"
		if strings.HasPrefix(target, "windows") {
			path += ".exe"
		}
		b, err := os.ReadFile(filepath.Join(bin, path))
		fail(err)
		hash := digest(b)
		m.Artifacts = append(m.Artifacts, artifact{target, path, hash})
		fmt.Fprintf(&sums, "%s  %s\n", hash, path)
	}
	b, err := json.MarshalIndent(m, "", "  ")
	fail(err)
	fail(os.WriteFile(filepath.Join(bin, "build-manifest.json"), append(b, '\n'), 0644))
	fail(os.WriteFile(filepath.Join(bin, "SHA256SUMS"), []byte(sums.String()), 0644))
}
