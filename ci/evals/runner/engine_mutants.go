package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Test expression mutants on the real engine. Identical artifacts/partitions
// reuse one observed execution result instead of submitting duplicate probes.
func calibrateEngineMutants(ctx context.Context, b bank, suite, repository, tenant, directory string) error {
	if err := validateSuite(suite); err != nil {
		return err
	}
	selected := selectCases(b, suite, "", "engine")
	if len(selected) == 0 {
		return fmt.Errorf("engine mutant selection matches no cases")
	}
	if _, err := os.Stat(directory); err == nil {
		return fmt.Errorf("engine calibration directory exists")
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(directory, "bank.snapshot.json"), b); err != nil {
		return err
	}
	v := engineVerifier{repository: repository, tenant: tenant, runDir: directory}
	if err := v.initialize(ctx); err != nil {
		return err
	}
	rows := []map[string]any{}
	type observed struct {
		outputs map[string]any
		err     error
	}
	cache := map[string]observed{}
	failed := false
	for _, c := range selected {
		// Unsafe/extraneous native executors are tested by the guarded adapter and
		// semantic calibration, rather than deployed as expression probes.
		if c.Engine.Kind == "command" {
			continue
		}
		for _, m := range c.Mutants {
			key := digest([]any{m.Evidence.Artifact, c.Engine})
			result, reused := cache[key]
			if !reused {
				result.outputs, result.err = v.evaluate(ctx, c, m.Evidence.Artifact)
				cache[key] = result
			}
			status, reason := "FAIL", "mutant accepted"
			scores := []predicateResult{}
			for _, a := range c.Assertions {
				if a.Operator == "engine" {
					score := assertionResult(a, evidence{EngineResults: result.outputs})
					scores = append(scores, score)
					if score.Status == "FAIL" && contains(m.Violates, score.ID) {
						status = "PASS"
						reason = "intended failure observed on real engine"
					}
				}
			}
			if result.err != nil && result.outputs == nil {
				status = "EVALUATOR_ERROR"
				reason = result.err.Error()
			}
			if status != "PASS" {
				failed = true
			}
			rows = append(rows, map[string]any{"case": c.ID, "candidate": m.Name, "candidate_digest": digest(m.Evidence.Artifact), "status": status, "reason": reason, "predicates": scores, "engine_results": result.outputs, "reused_observed_result": reused})
			if err := writeJSON(filepath.Join(directory, "calibration.json"), map[string]any{"kind": "real-engine-mutant-calibration", "bank_digest": digest(b), "results": rows}); err != nil {
				return err
			}
			fmt.Printf("%s %s engine challenge: %s\n", c.ID, m.Name, status)
			if v.journal.Pending != "" {
				break
			}
		}
		if v.journal.Pending != "" {
			break
		}
	}
	deadline, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	err := v.cleanup(deadline)
	cancel()
	if err != nil {
		return err
	}
	if failed {
		return fmt.Errorf("engine mutant calibration failed; inspect %s", directory)
	}
	return nil
}
