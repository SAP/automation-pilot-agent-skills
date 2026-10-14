package cmd

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
			{"time-zones", "list supported time zones", "GET /scheduled-executions/time-zones", schedTimeZones},
		},
	}
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
