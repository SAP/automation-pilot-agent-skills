package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/SAP/automation-pilot-agent-skills/cli/internal/client"
)

func scheduledResource() *resource {
	return &resource{
		name:    "scheduled-executions",
		summary: "manage recurring (scheduled) executions",
		verbs: []*verb{
			{"list", "list scheduled executions", "GET /scheduled-executions", schedList},
			{"get", "get one schedule", "GET /scheduled-executions/{id}", schedGet},
			{"create", "create a schedule", "POST /scheduled-executions", schedCreate},
			{"update", "update a schedule", "PUT /scheduled-executions/{id}", schedUpdate},
			{"delete", "delete a schedule", "DELETE /scheduled-executions/{id}", schedDelete},
			{"enable", "enable using read-modify-conditional PUT", "GET then PUT /scheduled-executions/{id}", schedEnable},
			{"disable", "disable using read-modify-conditional PUT", "GET then PUT /scheduled-executions/{id}", schedDisable},
			{"time-zones", "list supported time zones", "GET /scheduled-executions/time-zones", schedTimeZones},
		},
	}
}
func schedEnable(c *cmdContext, args []string) error  { return schedSetEnabled(c, args, true) }
func schedDisable(c *cmdContext, args []string) error { return schedSetEnabled(c, args, false) }
func schedSetEnabled(c *cmdContext, args []string, enabled bool) error {
	verb := "disable"
	if enabled {
		verb = "enable"
	}
	fs, g := newFlagSet(c, "scheduled-executions "+verb)
	setUsage(fs, "GET then PUT /scheduled-executions/{id}", "autopi scheduled-executions "+verb+" <id>", "autopi scheduled-executions "+verb+" sched-id")
	pos, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	id, err := requireID(pos, "schedule id")
	if err != nil {
		return err
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}
	cl := c.client(cfg)
	path := client.JoinPath("scheduled-executions", id)
	res, err := fetch(c, cl, client.Request{Method: "GET", Path: path})
	if err != nil {
		return err
	}
	if cfg.PrintRequest {
		_, err := fmt.Fprintln(c.stdout, "# Read the schedule, set enabled, and PUT the complete writable object with its fetched If-Match ETag.")
		return err
	}
	etag := res.Header.Get("ETag")
	if etag == "" {
		return fmt.Errorf("schedule response has no ETag; refusing an unlocked update")
	}
	var object map[string]any
	if err := json.Unmarshal(res.Body, &object); err != nil || object == nil {
		return fmt.Errorf("invalid schedule response")
	}
	delete(object, "state")
	delete(object, "id")
	object["enabled"] = enabled
	body, err := json.Marshal(object)
	if err != nil {
		return err
	}
	return execute(c, cfg, client.Request{Method: "PUT", Path: path, Body: body, Headers: map[string]string{"If-Match": etag}}, false)
}

func schedList(c *cmdContext, args []string) error {
	return listVerb(c, args, "scheduled-executions list", "GET /scheduled-executions",
		"autopi scheduled-executions list", "scheduled-executions", true)
}

func schedGet(c *cmdContext, args []string) error {
	return getVerb(c, args, "scheduled-executions get", "GET /scheduled-executions/{id}",
		"autopi scheduled-executions get sched123", "schedule id", "scheduled-executions")
}

func schedCreate(c *cmdContext, args []string) error {
	return createVerb(c, args, "scheduled-executions create", "POST /scheduled-executions",
		"autopi scheduled-executions create -f schedule.json", "scheduled-executions")
}

func schedUpdate(c *cmdContext, args []string) error {
	return writeVerbWithID(c, args, "scheduled-executions update", "PUT", "", "PUT /scheduled-executions/{id}",
		"autopi scheduled-executions update sched123 -f schedule.json", "schedule id", "scheduled-executions")
}

func schedDelete(c *cmdContext, args []string) error {
	return deleteVerb(c, args, "scheduled-executions delete", "DELETE /scheduled-executions/{id}",
		"autopi scheduled-executions delete sched123", "schedule id", "scheduled-executions")
}

func schedTimeZones(c *cmdContext, args []string) error {
	return listVerb(c, args, "scheduled-executions time-zones", "GET /scheduled-executions/time-zones",
		"autopi scheduled-executions time-zones", "scheduled-executions/time-zones", false)
}
