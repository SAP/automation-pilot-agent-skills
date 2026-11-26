package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func checkReferenceArtifacts(ctx context.Context, b bank, repository string) error {
	dir, err := os.MkdirTemp("", "autopi-reference-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	for _, c := range b.Cases {
		if c.Artifact == "" || c.Reference.Artifact == nil {
			continue
		}
		kind := ""
		switch filepath.Ext(c.Artifact) {
		case ".json":
			if filepath.Ext(c.Artifact[:len(c.Artifact)-5]) == ".command" {
				kind = "command"
			} else if filepath.Ext(c.Artifact[:len(c.Artifact)-5]) == ".input" {
				kind = "input"
			} else if filepath.Ext(c.Artifact[:len(c.Artifact)-5]) == ".mcp" {
				kind = "mcp-server"
			}
		}
		if kind == "" {
			continue
		}
		path := filepath.Join(dir, c.ID, c.Artifact)
		if err = writeJSON(path, c.Reference.Artifact); err != nil {
			return err
		}
		command := exec.CommandContext(ctx, filepath.Join(repository, "skills/automation-pilot-cli/scripts/autopi"), "lint", "--kind", kind, "--output", "json", path)
		command.Env = agentEnvironment(dir, "", "", "")
		if output, err := command.CombinedOutput(); err != nil {
			return fmt.Errorf("reference %s rejected by actual CLI: %w: %s", c.ID, err, output)
		}
	}
	return nil
}
func calibrateEngine(ctx context.Context, b bank, suite, repository, tenant, directory string) error {
	if err := validateSuite(suite); err != nil {
		return err
	}
	selected := selectCases(b, suite, "", "engine")
	if len(selected) == 0 {
		return fmt.Errorf("engine calibration selection matches no cases")
	}
	if _, err := os.Stat(directory); err == nil {
		return fmt.Errorf("calibration directory exists; choose a fresh directory")
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		return err
	}
	v := engineVerifier{repository: repository, runDir: directory, tenant: tenant}
	if err := v.initialize(ctx); err != nil {
		return err
	}
	results := []caseResult{}
	failed := false
	for _, c := range selected {
		result := caseResult{ID: c.ID, CaseDigest: digest(c), Primary: c.Primary, Family: c.Family, Tier: c.Tier, Review: c.Review, Status: "EVALUATOR_ERROR"}
		output, err := v.evaluate(ctx, c, c.Reference.Artifact)
		if err != nil {
			result.Reason = err.Error()
			failed = true
		} else {
			for _, a := range c.Assertions {
				if a.Operator == "engine" {
					result.Predicates = append(result.Predicates, assertionResult(a, evidence{EngineResults: output}))
				}
			}
			result.Status = aggregate(result.Predicates)
			if result.Status != "PASS" {
				failed = true
			}
		}
		results = append(results, result)
		if err := writeJSON(filepath.Join(directory, "calibration.json"), map[string]any{"kind": "real-engine-reference-calibration", "bank_digest": digest(b), "results": results}); err != nil {
			return err
		}
		fmt.Printf("%s reference: %s\n", c.ID, result.Status)
		if v.journal.Pending != "" {
			break
		}
	}
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	cleanupErr := v.cleanup(cleanupCtx)
	cancel()
	if cleanupErr != nil {
		return cleanupErr
	}
	if failed {
		return fmt.Errorf("reference engine calibration failed; inspect %s", directory)
	}
	return nil
}
