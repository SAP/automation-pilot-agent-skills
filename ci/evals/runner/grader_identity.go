package main

import (
	"embed"
	"github.com/SAP/automation-pilot-agent-skills/ci/evals/internal/redact"
)

// Bind resumed calibration to the grader implementation in this binary. A
// changed prompt, evidence projection or verifier must not reuse old judgments.
//
//go:embed agent.go grade.go host.go isolation.go model.go
var graderSources embed.FS

func gradingProtocolDigest() string {
	sources := map[string]string{"redaction": redact.ProtocolDigest()}
	for _, path := range []string{"agent.go", "grade.go", "host.go", "isolation.go", "model.go"} {
		contents, err := graderSources.ReadFile(path)
		if err != nil {
			panic(err)
		}
		sources[path] = bytesDigest(contents)
	}
	return digest(sources)
}
