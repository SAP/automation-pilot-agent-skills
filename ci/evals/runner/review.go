package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Review sealed candidate evidence, preserving the original run. Agent sessions
// and fixture mutations are never replayed. Only grading and run-owned engine
// acceptance are performed, after the source journal has been reconciled.
func reviewCapturedRun(ctx context.Context, b bank, repository, sourceDir, outDir, tenant string, h hostConfig, perGrade float64, parallel int) error {
	if !validCost(perGrade) || parallel < 1 || parallel > 8 {
		return fmt.Errorf("invalid review limits")
	}
	sourceDir, err := filepath.Abs(sourceDir)
	if err != nil {
		return err
	}
	outDir, err = filepath.Abs(outDir)
	if err != nil {
		return err
	}
	if _, err = os.Stat(outDir); err == nil {
		return fmt.Errorf("review output exists; choose a fresh directory")
	}
	var report runReport
	if err = readJSON(filepath.Join(sourceDir, "report.json"), &report); err != nil {
		return err
	}
	oldBank, err := loadBank(filepath.Join(sourceDir, "bank.snapshot.json"))
	if err != nil {
		return err
	}
	if digest(oldBank) != report.BankDigest {
		return fmt.Errorf("saved run bank digest mismatch")
	}
	currentCases := map[string]evalCase{}
	for _, c := range b.Cases {
		currentCases[c.ID] = c
	}
	for _, old := range oldBank.Cases {
		c, ok := currentCases[old.ID]
		if !ok || c.Prompt != old.Prompt || digest(c.Files) != digest(old.Files) {
			return fmt.Errorf("%s public task changed; captured answers cannot be reused", old.ID)
		}
	}
	// Revised hidden references/rubrics may be assessed against the same sealed
	// public task. Keep the original report and record the current bank identity.
	report.BankDigest = digest(b)
	var oldJournal engineJournal
	if err = readJSON(filepath.Join(sourceDir, "engine-journal.json"), &oldJournal); err == nil {
		if oldJournal.Pending != "" || oldJournal.Catalog != "" || len(oldJournal.Commands) > 0 || len(oldJournal.Executions) > 0 {
			return fmt.Errorf("source engine journal must be reconciled and cleaned before review")
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err = os.MkdirAll(outDir, 0700); err != nil {
		return err
	}
	if len(report.E2E) > 0 {
		if err := copyTree(filepath.Join(sourceDir, "e2e"), filepath.Join(outDir, "e2e")); err != nil {
			return err
		}
	}
	report.OriginalReport = filepath.Join(sourceDir, "report.json")
	report.ID = filepath.Base(outDir)
	report.ReviewedAt = time.Now().UTC().Format(time.RFC3339)
	report.GraderModel = h.Model
	report.GraderConfigDigest = hostConfigDigest(h)
	report.GraderDigest = gradingProtocolDigest()
	report.Parallel = parallel
	cases := map[string]evalCase{}
	for _, c := range b.Cases {
		cases[c.ID] = c
	}
	rows := []caseResult{}
	for _, r := range report.Results {
		if _, ok := cases[r.ID]; !ok {
			continue
		}
		if r.Generation > 0 {
			r.OriginalStatus = r.Status
		}
		rows = append(rows, r)
	}
	report.Results = rows
	if err = writeJSON(filepath.Join(outDir, "bank.snapshot.json"), b); err != nil {
		return err
	}
	if err = writeJSON(filepath.Join(outDir, "report.json"), report); err != nil {
		return err
	}
	var verifier *engineVerifier
	for _, r := range rows {
		if r.Generation > 0 && r.Tier == "engine" && r.EvidenceFile != "" {
			verifier = &engineVerifier{repository: repository, runDir: outDir, tenant: tenant}
			if err = verifier.initialize(ctx); err != nil {
				return err
			}
			report.Tenant = verifier.tenant
			break
		}
	}
	work := func(index int) caseResult {
		r := rows[index]
		c := cases[r.ID]
		r.CaseDigest = digest(c)
		if r.Generation == 0 || r.EvidenceFile == "" {
			return r
		}
		var e evidence
		if err := readJSON(filepath.Join(sourceDir, r.EvidenceFile), &e); err != nil {
			r.Status = "EVALUATOR_ERROR"
			r.Reason = err.Error()
			return r
		}
		originalEvidence := filepath.Join(r.ID, fmt.Sprintf("generation-%d", r.Generation), "original-evidence.json")
		if err := writeJSON(filepath.Join(outDir, originalEvidence), e); err != nil {
			r.Status = "EVALUATOR_ERROR"
			r.Reason = err.Error()
			return r
		}
		// Incomplete agent sessions cannot become successful by regrading a fragment.
		if e.Answer == "" || ((r.Status == "UNVERIFIED" || r.Status == "EVALUATOR_ERROR") && !strings.Contains(r.Reason, "engine") && !strings.Contains(r.Reason, "grader") && !strings.Contains(r.Reason, "autopi") && !strings.Contains(r.Reason, "mutation")) {
			r.EvidenceFile = originalEvidence
			return r
		}
		r.Reason = ""
		if c.Engine != nil && e.Artifact != nil {
			if c.Engine.Kind == "command" {
				if err := validateNativeCandidate(e.Artifact); err != nil {
					e.HardFailures = append(e.HardFailures, err.Error())
				}
			}
			deadline, cancel := context.WithTimeout(ctx, 3*time.Minute)
			outputs, err := verifier.evaluate(deadline, c, e.Artifact)
			cancel()
			e.EngineResults = outputs
			if err != nil {
				r.Reason = err.Error()
			}
		}
		r.Predicates = deterministic(c, e)
		if len(c.Criteria) > 0 {
			judgments, meta := externalGrade(ctx, h, c, e, perGrade)
			r.GraderCostUSD += meta.Cost
			r.CostUSD += meta.Cost
			r.CostKnown = r.CostKnown && meta.CostKnown
			r.GraderModel = meta.Model
			r.GraderUsage = addUsage(r.GraderUsage, meta.Usage)
			if meta.CostKnown {
				r.BudgetChargedUSD += meta.Cost
			} else {
				r.BudgetChargedUSD += perGrade
			}
			if meta.Err != nil {
				r.Status = "EVALUATOR_ERROR"
				r.Reason = meta.Err.Error()
			} else {
				scores, err := semanticResults(c, e, judgments)
				if err != nil {
					r.Status = "EVALUATOR_ERROR"
					r.Reason = err.Error()
				} else {
					r.Predicates = append(r.Predicates, scores...)
					r.Status = aggregate(r.Predicates)
				}
			}
		} else {
			r.Status = aggregate(r.Predicates)
		}
		r.EvidenceFile = filepath.Join(r.ID, fmt.Sprintf("generation-%d", r.Generation), "evidence.json")
		if err := writeJSON(filepath.Join(outDir, r.EvidenceFile), e); err != nil {
			r.Status = "EVALUATOR_ERROR"
			r.Reason = err.Error()
		}
		return r
	}
	err = parallelJobs(ctx, len(rows), parallel, 0, perGrade, work,
		func(index int, reason string) caseResult { r := rows[index]; r.Reason = reason; return r },
		func(r caseResult) float64 { return 0 },
		func(index int, r caseResult) error {
			report.Results[index] = r
			fmt.Printf("%s verification review: %s\n", r.ID, r.Status)
			return writeJSON(filepath.Join(outDir, "report.json"), report)
		})
	if verifier != nil {
		deadline, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		cleanupErr := verifier.cleanup(deadline)
		cancel()
		if cleanupErr != nil && err == nil {
			err = cleanupErr
		}
	}
	if err != nil {
		return err
	}
	if err = printReport(report, ""); err != nil {
		return err
	}
	for _, r := range report.Results {
		if r.Generation > 0 && r.Status != "PASS" {
			return fmt.Errorf("review contains nonpassing cases; inspect %s/report.json", outDir)
		}
	}
	return nil
}
