package cmd

import (
	"encoding/json"
	"github.com/SAP/automation-pilot-agent-skills/cli/internal/domain"
)

// runPolicy exposes the same snapshot used by trigger validation and lint.
func runPolicy(c *cmdContext, args []string) error {
	if len(args) != 0 {
		return usagef("usage: autopi policy")
	}
	return json.NewEncoder(c.stdout).Encode(struct {
		Snapshot         string           `json:"snapshot"`
		Features         []domain.Feature `json:"features"`
		ControlDurations []string         `json:"controlDurations"`
	}{domain.Snapshot, domain.Features, domain.Durations})
}

func runSurface(c *cmdContext, args []string) error {
	if len(args) != 0 {
		return usagef("usage: autopi surface")
	}
	type entry struct {
		Name  string   `json:"name"`
		Verbs []string `json:"verbs"`
	}
	entries := []entry{}
	for _, resource := range registry() {
		item := entry{Name: resource.name}
		for _, verb := range resource.verbs {
			item.Verbs = append(item.Verbs, verb.name)
		}
		entries = append(entries, item)
	}
	return json.NewEncoder(c.stdout).Encode(entries)
}
