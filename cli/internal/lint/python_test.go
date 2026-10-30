package lint

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func pythonInterpreter(t *testing.T) string {
	t.Helper()
	for _, name := range []string{"python3", "python"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	t.Skip("optional Python syntax and template tests need Python 3")
	return ""
}

func TestPythonSyntaxNeverExecutesSource(t *testing.T) {
	interpreter := pythonInterpreter(t)
	sentinel := filepath.Join(t.TempDir(), "must-not-exist")
	for _, variant := range []string{"ExecutePythonScript", "SensitiveExecutePythonScript"} {
		doc := validCommand()
		ex := doc["configuration"].(map[string]any)["executors"].([]any)[0].(map[string]any)
		ex["execute"] = "scripts-sapcp:" + variant + ":1"
		ex["input"] = map[string]any{"script": "from pathlib import Path\nPath(" + q(sentinel) + ").write_text('unexpected')\n"}
		findings, err := CheckPythonSyntax("CallApi.command.json", doc, interpreter)
		if err != nil || len(findings) != 0 {
			t.Fatalf("syntax check: %v %v", findings, err)
		}
		if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
			t.Fatal("submitted Python was executed")
		}
	}
}

func TestPythonSyntaxRegressionAndDynamicSource(t *testing.T) {
	interpreter := pythonInterpreter(t)
	for _, tc := range []struct {
		source   string
		severity Severity
	}{
		{"reasons = [k for k, fail in [('rain', True)] if fail]", ""},
		{"reasons = [k for k, fail in [('rain', True)] if True else []", High},
		{"$(.execution.input.script)", Medium},
	} {
		doc := validCommand()
		ex := doc["configuration"].(map[string]any)["executors"].([]any)[0].(map[string]any)
		ex["execute"] = "scripts-sapcp:ExecutePythonScript:1"
		ex["input"] = map[string]any{"script": tc.source}
		findings, err := CheckPythonSyntax("CallApi.command.json", doc, interpreter)
		if err != nil {
			t.Fatal(err)
		}
		if tc.severity == "" && len(findings) != 0 {
			t.Fatalf("valid syntax rejected: %v", findings)
		}
		if tc.severity != "" && (len(findings) != 1 || findings[0].Severity != tc.severity) {
			t.Fatalf("findings=%v", findings)
		}
		for _, finding := range findings {
			if strings.Contains(finding.Message, "reasons") {
				t.Fatal("source leaked in syntax diagnostic")
			}
		}
	}
}
