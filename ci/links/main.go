// links checks local targets and heading anchors in public Markdown guidance.
package main

import (
	"flag"
	"fmt"
	"html"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

var fencePattern = regexp.MustCompile("^ {0,3}(\x60{3,}|~{3,})")
var headingPattern = regexp.MustCompile(`(?m)^ {0,3}#{1,6}[\t ]+(.+?)\s*#*\s*$`)
var tagPattern = regexp.MustCompile(`<[^>]*>`)
var labelPattern = regexp.MustCompile(`\[([^]]+)\]\([^)]*\)`)
var linkPattern = regexp.MustCompile(`!?\[[^]\n]*\]\(\s*(<[^>]+>|[^\s)]+)(?:\s+"[^"]*")?\s*\)`)
var imagePattern = regexp.MustCompile(`<img\b[^>]*\bsrc="([^"]+)"`)

func prose(text string) string {
	var lines []string
	fence := ""
	for _, line := range strings.Split(text, "\n") {
		if m := fencePattern.FindStringSubmatch(line); m != nil {
			marker := m[1]
			if fence == "" {
				fence = marker
			} else if marker[0] == fence[0] && len(marker) >= len(fence) {
				fence = ""
			}
			continue
		}
		if fence == "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}
func stripCode(text string) string {
	var out strings.Builder
	for len(text) > 0 {
		i := strings.IndexByte(text, '\x60')
		if i < 0 {
			out.WriteString(text)
			break
		}
		out.WriteString(text[:i])
		text = text[i:]
		n := 0
		for n < len(text) && text[n] == '\x60' {
			n++
		}
		marker := text[:n]
		rest := text[n:]
		j := strings.Index(rest, marker)
		if j < 0 {
			out.WriteString(text)
			break
		}
		text = rest[j+n:]
	}
	return out.String()
}
func anchors(text string) map[string]bool {
	used := map[string]bool{}
	for _, m := range headingPattern.FindAllStringSubmatch(prose(text), -1) {
		heading := tagPattern.ReplaceAllString(m[1], "")
		heading = labelPattern.ReplaceAllString(heading, "$1")
		heading = strings.ToLower(html.UnescapeString(heading))
		var slug strings.Builder
		for _, r := range heading {
			if r == ' ' {
				slug.WriteByte('-')
			} else if r == '_' || r == '-' || (!unicode.IsPunct(r) && !unicode.IsSymbol(r)) {
				slug.WriteRune(r)
			}
		}
		base := slug.String()
		anchor := base
		for n := 1; used[anchor]; n++ {
			anchor = fmt.Sprintf("%s-%d", base, n)
		}
		used[anchor] = true
	}
	return used
}
func excluded(name string) bool {
	return name == ".local" || name == "runs" || name == "results" || name == "node_modules" || name == "__pycache__" || name == ".git"
}
func check(root string) (int, []string) {
	var problems []string
	var err error
	root, err = filepath.Abs(root)
	if err != nil {
		return 0, []string{err.Error()}
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return 0, []string{err.Error()}
	}
	sources := []string{}
	for _, name := range []string{"README.md", "CONTRIBUTING.md", "SECURITY.md"} {
		sources = append(sources, filepath.Join(root, name))
	}
	for _, name := range []string{"skills", "ci"} {
		err := filepath.WalkDir(filepath.Join(root, name), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if excluded(d.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(path, ".md") {
				if d.Type()&os.ModeSymlink != 0 {
					return fmt.Errorf("symlink guidance: %s", path)
				}
				sources = append(sources, path)
			}
			return nil
		})
		if err != nil {
			problems = append(problems, err.Error())
		}
	}
	count := 0
	cache := map[string]map[string]bool{}
	for _, source := range sources {
		relative, _ := filepath.Rel(root, source)
		raw, err := os.ReadFile(source)
		if err != nil {
			problems = append(problems, relative+": missing guidance file")
			continue
		}
		text := stripCode(prose(string(raw)))
		targets := linkPattern.FindAllStringSubmatch(text, -1)
		targets = append(targets, imagePattern.FindAllStringSubmatch(text, -1)...)
		for _, m := range targets {
			target := html.UnescapeString(strings.Trim(m[1], "<>"))
			u, err := url.Parse(target)
			if err != nil {
				problems = append(problems, relative+": invalid URL "+target)
				continue
			}
			if u.Scheme != "" || u.Host != "" {
				continue
			}
			count++
			dest := source
			if u.Path != "" {
				dest = filepath.Join(filepath.Dir(source), filepath.FromSlash(u.Path))
			}
			dest, err = filepath.EvalSymlinks(dest)
			if err != nil {
				problems = append(problems, relative+": missing target "+target)
				continue
			}
			rel, err := filepath.Rel(root, dest)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				problems = append(problems, relative+": target escapes repository "+target)
				continue
			}
			private := false
			for _, part := range strings.Split(rel, string(filepath.Separator)) {
				if excluded(part) {
					private = true
				}
			}
			if private {
				problems = append(problems, relative+": target is private local evidence "+target)
				continue
			}
			if u.Fragment != "" && filepath.Ext(dest) == ".md" {
				if cache[dest] == nil {
					b, err := os.ReadFile(dest)
					if err != nil {
						problems = append(problems, relative+": cannot read "+target)
						continue
					}
					cache[dest] = anchors(string(b))
				}
				if !cache[dest][u.Fragment] {
					problems = append(problems, relative+": missing anchor "+target)
				}
			}
		}
	}
	return count, problems
}
func main() {
	root := flag.String("root", ".", "repository root")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected arguments")
		os.Exit(1)
	}
	count, problems := check(*root)
	if len(problems) > 0 {
		fmt.Fprintln(os.Stderr, strings.Join(problems, "\n"))
		os.Exit(1)
	}
	fmt.Printf("Verified %d local links and anchors in public guidance\n", count)
}
