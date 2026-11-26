package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

var outcomeOrder = []string{"PASS", "FAIL", "EVALUATOR_ERROR", "UNVERIFIED", "UNRUN"}

type reportSummary struct {
	Selected, Excluded, Pending, Unpriced int
	E2EUnpriced                           int
	Counts, Repairs, RunErrors            map[string]int
	Skills, Tiers                         map[string]map[string]int
	Cost, AgentCost, GraderCost           float64
	AgentUsage, GraderUsage               map[string]any
}

func summarizeReport(report runReport) reportSummary {
	s := reportSummary{Counts: map[string]int{}, Repairs: map[string]int{}, RunErrors: map[string]int{}, Skills: map[string]map[string]int{}, Tiers: map[string]map[string]int{}}
	addCost := func(r caseResult) {
		s.Cost += r.CostUSD
		s.AgentCost += r.AgentCostUSD
		s.GraderCost += r.GraderCostUSD
		s.AgentUsage = addUsage(s.AgentUsage, r.AgentUsage)
		s.GraderUsage = addUsage(s.GraderUsage, r.GraderUsage)
		if r.Generation > 0 && !r.CostKnown {
			s.Unpriced++
		}
	}
	for _, r := range report.Results {
		addCost(r)
		for _, repair := range r.Repairs {
			addCost(repair)
			s.Repairs[repair.Status]++
		}
		if r.Primary == "" {
			if r.Status != "PASS" {
				s.RunErrors[r.ID+": "+r.Reason]++
			}
			continue // Cleanup errors are run errors, not extra test cases.
		}
		if r.Status == "UNRUN" && r.Reason == "excluded by selection" {
			s.Excluded++
			continue
		}
		s.Selected++
		s.Counts[r.Status]++
		if r.Review != "reviewed" {
			s.Pending++
		}
		for _, group := range []struct {
			key  string
			rows map[string]map[string]int
		}{{r.Primary, s.Skills}, {r.Tier, s.Tiers}} {
			if group.rows[group.key] == nil {
				group.rows[group.key] = map[string]int{}
			}
			group.rows[group.key][r.Status]++
		}
	}
	var addE2E func(e2eRunResult)
	addE2E = func(r e2eRunResult) {
		s.Cost += r.CostUSD
		s.AgentCost += r.CostUSD
		s.AgentUsage = addUsage(s.AgentUsage, r.Usage)
		if !r.CostKnown {
			s.E2EUnpriced++
		}
		for _, repair := range r.Repairs {
			addE2E(repair)
		}
	}
	for _, r := range report.E2E {
		addE2E(r)
	}

	return s
}

func outcomeCounts(counts map[string]int) string {
	parts := []string{}
	for _, status := range outcomeOrder {
		parts = append(parts, fmt.Sprintf("%s %d", status, counts[status]))
	}
	return strings.Join(parts, ", ")
}

func markdownCell(value string) string {
	return strings.NewReplacer("|", "\\|", "\n", " ", "\r", " ").Replace(value)
}

func reportMarkdown(report runReport) string {
	s := summarizeReport(report)
	var out strings.Builder
	fmt.Fprintf(&out, "# Evaluation report: %s\n\n", markdownCell(report.ID))
	fmt.Fprintf(&out, "Host: %s. Agent model: %s. Grader model: %s. Suite: %s.\n\n", report.Host, defaultModel(report.Model), defaultModel(report.GraderModel), report.Suite)
	if report.Tenant != "" {
		fmt.Fprintf(&out, "Engine tenant resolved from the Automation Pilot environment: %s.\n\n", markdownCell(report.Tenant))
	}
	fmt.Fprintf(&out, "%d selected case-generations; %d excluded cases. Repairs are counted separately.\n\n", s.Selected, s.Excluded)
	if s.Counts["EVALUATOR_ERROR"]+s.Counts["UNVERIFIED"]+s.Counts["UNRUN"] > 0 || len(s.RunErrors) > 0 || e2eIncomplete(report.E2E) {
		out.WriteString("**Incomplete verification: these results do not establish an overall skill-quality score.**\n\n")
	}
	if s.Pending > 0 {
		fmt.Fprintf(&out, "**Provisional:** %d selected case-generations still require case review/calibration sign-off.\n\n", s.Pending)
	}
	out.WriteString("| Outcome | Case-generations |\n|---|---:|\n")
	for _, status := range outcomeOrder {
		fmt.Fprintf(&out, "| %s | %d |\n", status, s.Counts[status])
	}
	for _, group := range []struct {
		name string
		rows map[string]map[string]int
	}{{"Tier", s.Tiers}, {"Primary skill", s.Skills}} {
		fmt.Fprintf(&out, "\n| %s | Pass | Fail | Evaluator error | Unverified | Unrun |\n|---|---:|---:|---:|---:|---:|\n", group.name)
		for _, key := range sortedKeys(group.rows) {
			fmt.Fprintf(&out, "| %s", markdownCell(key))
			for _, status := range outcomeOrder {
				fmt.Fprintf(&out, " | %d", group.rows[key][status])
			}
			out.WriteString(" |\n")
		}
	}
	out.WriteString(e2eMarkdown(report.E2E))
	if len(s.Repairs) > 0 {
		fmt.Fprintf(&out, "\nSeparate repair outcomes: %s.\n", outcomeCounts(s.Repairs))
	}
	fmt.Fprintf(&out, "\nReported USD cost: **$%.2f** (agent $%.2f; grading $%.2f), including repairs. This is CLI accounting, not a verified bill. %d case-generation/repair rows have incomplete cost data.\n\n", s.Cost, s.AgentCost, s.GraderCost, s.Unpriced+s.E2EUnpriced)
	if report.OriginalReport != "" {
		out.WriteString("This review reuses the original candidate sessions. Costs include the original campaign and subsequent grading; do not add the original report total again. Historical grader usage may omit earlier attempts if captured before cumulative usage accounting was added.\n\n")
	}
	out.WriteString("| Usage reported by CLI | Agent | Grader |\n|---|---:|---:|\n")
	for _, row := range []struct{ label, key string }{
		{"Input tokens", "input_tokens"}, {"Output tokens", "output_tokens"},
		{"Claude cache read tokens", "cache_read_input_tokens"}, {"Claude cache creation tokens", "cache_creation_input_tokens"},
		{"Codex cached input tokens", "cached_input_tokens"},
	} {
		a, _ := s.AgentUsage[row.key].(float64)
		g, _ := s.GraderUsage[row.key].(float64)
		fmt.Fprintf(&out, "| %s | %.0f | %.0f |\n", row.label, a, g)
	}
	out.WriteString("\nMissing usage fields are shown as zero recorded tokens, not proof of zero usage. Claude reports cache usage separately; Codex cached input is part of input usage. Do not sum cache columns into one cross-provider token total.\n")
	if len(s.RunErrors) > 0 {
		out.WriteString("\nRun-level errors:\n\n")
		for _, message := range sortedKeys(s.RunErrors) {
			fmt.Fprintf(&out, "- %s\n", markdownCell(message))
		}
	}
	problems := []caseResult{}
	for _, r := range report.Results {
		if r.Primary != "" && r.Status != "PASS" && !(r.Status == "UNRUN" && r.Reason == "excluded by selection") {
			problems = append(problems, r)
		}
	}
	sort.SliceStable(problems, func(i, j int) bool {
		if problems[i].ID != problems[j].ID {
			return problems[i].ID < problems[j].ID
		}
		return problems[i].Generation < problems[j].Generation
	})
	if len(problems) > 0 {
		out.WriteString("\n| Case | Generation | Outcome | Reason / failed checks | Evidence path relative to run directory |\n|---|---:|---|---|---|\n")
		for _, r := range problems {
			reasons := []string{}
			if r.Reason != "" {
				reasons = append(reasons, r.Reason)
			}
			for _, p := range r.Predicates {
				if p.Status != "PASS" {
					reasons = append(reasons, p.ID+": "+p.Status+" "+p.Reason)
				}
			}
			fmt.Fprintf(&out, "| %s | %d | %s | %s | %s |\n", markdownCell(r.ID), r.Generation, r.Status, markdownCell(strings.Join(reasons, "; ")), markdownCell(r.EvidenceFile))
		}
	}
	return out.String()
}

func defaultModel(model string) string {
	if model == "" {
		return "configured default"
	}
	return model
}

func writeReportSummary(path string, report runReport) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, writeErr := f.WriteString(reportMarkdown(report))
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}
