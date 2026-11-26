package main

import "fmt"

const defaultSuite = "smoketest"

var suiteNames = []string{"smoketest", "minimal", "full", "e2e", "pilot", "smoke", "regression", "release", "all"}

func validateSuite(suite string) error {
	if !contains(suiteNames, suite) {
		return fmt.Errorf("unknown suite %q; choose smoketest, minimal, full, e2e, pilot, smoke, regression, release, or all", suite)
	}
	return nil
}

func selectedCases(b bank, suite, skills, tiers, caseID string) []evalCase {
	cases := selectCases(b, suite, skills, tiers)
	if caseID == "" {
		return cases
	}
	for _, c := range cases {
		if c.ID == caseID {
			return []evalCase{c}
		}
	}
	return nil
}

func runGenerations(suite string, requested int, explicit bool) int {
	if suite == "release" && !explicit {
		return 3
	}
	return requested
}
