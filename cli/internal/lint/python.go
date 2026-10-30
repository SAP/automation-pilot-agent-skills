package lint

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// CheckPythonSyntax delegates grammar parsing to an explicitly selected interpreter.
// The submitted source is parsed as an AST, never imported or executed. Normal lint
// stays dependency-free; this optional language check requires Python 3.
func CheckPythonSyntax(file string, fields map[string]any, interpreter string) ([]Finding, error) {
	const parser = `import ast, json, sys
try:
    ast.parse(sys.stdin.read())
except SyntaxError as error:
    print(json.dumps({"valid": False, "line": error.lineno or 0}))
else:
    print(json.dumps({"valid": True}))
`
	var out []Finding
	for _, script := range PythonScripts(fields) {
		if strings.Contains(script.Source, "$(") {
			out = append(out, Finding{Medium, file, script.Path, "dynamic Python source was not syntax-checked; resolve a non-secret fixture first"})
			continue
		}
		// Bound the parser independently of tenant/network deadlines.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		command := exec.CommandContext(ctx, interpreter, "-I", "-S", "-c", parser)
		command.Stdin = strings.NewReader(script.Source)
		command.Env = []string{"PATH=" + os.Getenv("PATH"), "SYSTEMROOT=" + os.Getenv("SYSTEMROOT")}
		var output bytes.Buffer
		command.Stdout = &output
		err := command.Run()
		cancel()
		if err != nil {
			return nil, fmt.Errorf("Python syntax parser failed; check the selected Python 3 interpreter")
		}
		var result struct {
			Valid *bool `json:"valid"`
			Line  *int  `json:"line"`
		}
		if json.Unmarshal(output.Bytes(), &result) != nil || result.Valid == nil || (!*result.Valid && result.Line == nil) {
			return nil, fmt.Errorf("Python syntax parser returned an invalid result")
		}
		if result.Line != nil {
			out = append(out, Finding{High, file, script.Path, fmt.Sprintf("invalid Python syntax at line %d", *result.Line)})
		}
	}
	return out, nil
}
