package cmd

import "github.com/SAP/automation-pilot-agent-skills/cli/internal/client"

func inputsResource() *resource {
	return &resource{
		name:    "inputs",
		summary: "manage inputs (credentials and configuration)",
		verbs: []*verb{
			{"list", "list inputs in a catalog", "GET /inputs", inputsList},
			{"get", "get one input", "GET /inputs/{id}", inputsGet},
			{"create", "create an input", "POST /inputs", inputsCreate},
			{"update", "full update of an input", "PUT /inputs/{id}", inputsUpdate},
			{"patch", "partial update (values only)", "PATCH /inputs/{id}", inputsPatch},
			{"delete", "delete an input", "DELETE /inputs/{id}", inputsDelete},
			{"release", "release an input", "PUT /inputs/{id}/release", inputsRelease},
			{"deprecate", "deprecate an input", "PUT /inputs/{id}/deprecate", inputsDeprecate},
			{"restore", "restore an input", "PUT /inputs/{id}/restore", inputsRestore},
			{"bulk-create", "create many inputs", "POST /bulk/inputs", inputsBulkCreate},
			{"deploy", "upsert an input from a file", "GET+PUT|POST /inputs", inputsDeploy},
		},
	}
}

func inputsList(c *cmdContext, args []string) error {
	return catalogListVerb(c, args, "inputs list", "GET /inputs",
		"autopi inputs list --catalog my-cat --tag type=credentials", "inputs", false, true)
}

// inputsGet mirrors the MCP get_input tool: the secret-bearing values block is
// stripped by default so credentials stay out of the response. Pass --full to
// include it (the API returns it; stored secrets still cannot be read back).
func inputsGet(c *cmdContext, args []string) error {
	fs, g := newFlagSet(c, "inputs get")
	var statusOnly, full bool
	fs.BoolVar(&statusOnly, "status-only", false, "print only the HTTP status code")
	fs.BoolVar(&full, "full", false, "include the values block (omitted by default to keep secrets out of the view)")
	setUsage(fs, "GET /inputs/{id}", "autopi inputs get <id> [--full]",
		"autopi inputs get my-cat:BtpCredentials:1")
	pos, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	id, err := requireID(pos, "input id")
	if err != nil {
		return err
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}
	req := client.Request{Method: "GET", Path: client.JoinPath("inputs", id)}
	// --full / --status-only / --print-request need no projection; delegate to
	// the shared path. Only the default view strips values.
	if full || statusOnly || cfg.PrintRequest {
		return execute(c, cfg, req, statusOnly)
	}
	req.Project = func(b []byte) ([]byte, error) {
		return client.InputContract(b)
	}
	return execute(c, cfg, req, false)
}

func inputsCreate(c *cmdContext, args []string) error {
	return createVerb(c, args, "inputs create", "POST /inputs",
		"autopi inputs create -f BtpCredentials.input.json", "inputs")
}

func inputsUpdate(c *cmdContext, args []string) error {
	return writeVerbWithID(c, args, "inputs update", "PUT", "", "PUT /inputs/{id}",
		"autopi inputs update my-cat:BtpCredentials:1 -f BtpCredentials.input.json", "input id", "inputs")
}

// inputsPatch forces the merge-patch content type so the values-only partial
// update can't be sent with the wrong header.
func inputsPatch(c *cmdContext, args []string) error {
	return writeVerbWithID(c, args, "inputs patch", "PATCH", "application/merge-patch+json", "PATCH /inputs/{id}",
		`autopi inputs patch my-cat:BtpCredentials:1 -f - < protected-rotation.json`, "input id", "inputs")
}

func inputsDelete(c *cmdContext, args []string) error {
	return deleteVerb(c, args, "inputs delete", "DELETE /inputs/{id}",
		"autopi inputs delete my-cat:BtpCredentials:1", "input id", "inputs")
}

func inputsRelease(c *cmdContext, args []string) error {
	return lifecycleVerb(c, args, "inputs release", "release", "PUT /inputs/{id}/release",
		"autopi inputs release my-cat:BtpCredentials:1", "input id", "inputs")
}

func inputsDeprecate(c *cmdContext, args []string) error {
	return lifecycleVerb(c, args, "inputs deprecate", "deprecate", "PUT /inputs/{id}/deprecate",
		"autopi inputs deprecate my-cat:BtpCredentials:1", "input id", "inputs")
}

func inputsRestore(c *cmdContext, args []string) error {
	return lifecycleVerb(c, args, "inputs restore", "restore", "PUT /inputs/{id}/restore",
		"autopi inputs restore my-cat:BtpCredentials:1", "input id", "inputs")
}

func inputsBulkCreate(c *cmdContext, args []string) error {
	return createVerb(c, args, "inputs bulk-create", "POST /bulk/inputs",
		`autopi inputs bulk-create -f inputs.json   # body: {"inputs":[...]}`, "bulk/inputs")
}

func inputsDeploy(c *cmdContext, args []string) error {
	return deployVerb(c, args, "inputs deploy", "input", "inputs")
}
