package main

import (
	"os"
	"path/filepath"
	"testing"
)

func testBank(t *testing.T) bank {
	t.Helper()
	b, err := loadBank("../cases/bank.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func findCase(t *testing.T, id string) evalCase {
	t.Helper()
	for _, c := range testBank(t).Cases {
		if c.ID == id {
			return c
		}
	}
	t.Fatal("missing case", id)
	return evalCase{}
}
func TestCorpusCoverageAndFamilySplits(t *testing.T) {
	b := testBank(t)
	if len(b.Cases) != 160 {
		t.Fatal(len(b.Cases))
	}
	counts := map[string]int{}
	tiers := map[string]int{}
	origins := map[string]int{}
	splits := map[string]int{}
	for _, c := range b.Cases {
		counts[c.Primary]++
		tiers[c.Tier]++
		origins[c.Origin]++
		splits[c.Split]++
		if c.Review == "reviewed" {
			t.Fatal("unrun cases must not claim independent review")
		}
	}
	expected := map[string]int{"command-authoring": 46, "dynamic-expressions": 24, "discovery": 16, "executions": 14, "inputs": 12, "plan": 10, "mcp-servers": 10, "scheduling": 8, "catalogs": 6, "tags": 6, "connection": 4, "cli": 4}
	for skill, count := range expected {
		if counts["automation-pilot-"+skill] != count {
			t.Fatal(skill, counts)
		}
	}
	if tiers["local"] != 80 || tiers["fixture"] != 56 || tiers["engine"] != 24 || origins["catalog"] != 100 || origins["workflow"] != 60 || splits["development"] != 120 || splits["holdout"] != 40 {
		t.Fatal(tiers, origins, splits)
	}
	for suite, count := range map[string]int{"pilot": 20, "smoke": 40} {
		selected := selectCases(b, suite, "", "")
		if len(selected) != count {
			t.Fatal(suite, len(selected))
		}
		covered := map[string]bool{}
		for _, c := range selected {
			if c.Split != "development" {
				t.Fatal("holdout included in development subset")
			}
			covered[c.Primary] = true
		}
		if len(covered) != 12 {
			t.Fatal("missing skill in", suite)
		}
	}
	for _, id := range []string{"checksum-recovery", "continuity", "proposal-scope", "local-authoring"} {
		findCase(t, id)
	}
}
func TestUnrecognizedFieldsAndUnsafePaths(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bank.json")
	if err := os.WriteFile(path, []byte("{\"schema_version\":1,\"cases\":[],\"ignored_grader\":true}"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadBank(path); err == nil {
		t.Fatal("unknown fields accepted")
	}
	for _, path := range []string{"../oracle.json", "/private/oracle", "a/../oracle", "a\\oracle", ""} {
		if safeRelative(path) {
			t.Fatal("unsafe path", path)
		}
	}
	b := testBank(t)
	b.Cases[1].ID = b.Cases[0].ID
	if validateBank(b) == nil {
		t.Fatal("duplicate ID accepted")
	}
}
func TestAllDeterministicEvidenceChallenges(t *testing.T) {
	b := testBank(t)
	tested := 0
	for _, c := range b.Cases {
		for _, p := range deterministic(c, c.Reference) {
			if p.Status != "PASS" {
				t.Fatalf("reference %s %s: %+v", c.ID, p.ID, p)
			}
		}
		for _, m := range c.Mutants {
			semantic := false
			for _, criterion := range c.Criteria {
				if contains(m.Violates, criterion.ID) {
					semantic = true
				}
			}
			if semantic {
				continue
			}
			hit := false
			for _, p := range deterministic(c, m.Evidence) {
				if p.Status == "FAIL" && contains(m.Violates, p.ID) {
					hit = true
				}
			}
			if !hit {
				t.Fatal(c.ID, m.Name, "mutant accepted")
			}
			tested++
		}
	}
	if tested < 45 {
		t.Fatal("insufficient deterministic challenge coverage", tested)
	}
}
