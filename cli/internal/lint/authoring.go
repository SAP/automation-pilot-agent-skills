package lint

import (
	"fmt"
	"regexp"
	"strings"
)

func commandExecutors(fields map[string]any) []any {
	config, _ := fields["configuration"].(map[string]any)
	if config != nil {
		executors, _ := config["executors"].([]any)
		return executors
	}
	executors, _ := fields["executors"].([]any)
	return executors
}

func checkAuthoringControls(file string, fields map[string]any) []Finding {
	var out []Finding
	for i, raw := range commandExecutors(fields) {
		executor, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		prefix := fmt.Sprintf("configuration.executors[%d]", i)
		if retry, ok := executor["autoRetry"].(map[string]any); ok && retry["logic"] != nil {
			if logic, ok := retry["logic"].(string); !ok || (logic != "FIXED" && logic != "INCREMENTAL") {
				out = append(out, Finding{High, file, prefix + ".autoRetry.logic", "retry logic must be FIXED or INCREMENTAL"})
			}
		}
		alias, _ := executor["alias"].(string)
		if alias == "" {
			continue
		}
		counter := regexp.MustCompile(`\.` + regexp.QuoteMeta(alias) + `\.(retriesCount|repetitionsCount)\b`)
		inputs, _ := executor["input"].(map[string]any)
		for _, key := range sortedKeys(inputs) {
			text, ok := inputs[key].(string)
			if !ok || key == "script" {
				continue
			}
			for _, expr := range Expressions(text) {
				if counter.MatchString(expr) {
					out = append(out, Finding{Medium, file, prefix + ".input." + key, "own retry/repeat counters do not provide changing executor input; use a changing response and observe counters in a later step"})
					break
				}
			}
		}
	}
	return out
}

// PythonScripts returns embedded Python source without running or interpolating it.
func PythonScripts(fields map[string]any) []PythonScript {
	var scripts []PythonScript
	for i, raw := range commandExecutors(fields) {
		executor, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id, _ := executor["execute"].(string)
		parts := strings.Split(id, ":")
		if len(parts) != 3 || (parts[1] != "ExecutePythonScript" && parts[1] != "SensitiveExecutePythonScript") {
			continue
		}
		input, _ := executor["input"].(map[string]any)
		if source, ok := input["script"].(string); ok && source != "" {
			scripts = append(scripts, PythonScript{fmt.Sprintf("configuration.executors[%d].input.script", i), source})
		}
	}
	return scripts
}

type PythonScript struct {
	Path   string
	Source string
}
