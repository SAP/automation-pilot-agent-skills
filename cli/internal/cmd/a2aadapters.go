package cmd

import (
	"fmt"

	"github.com/SAP/automation-pilot-agent-skills/cli/internal/client"
)

func a2aAdaptersResource() *resource {
	return &resource{
		name:    "a2a-adapters",
		summary: "manage A2A adapter (agent) definitions",
		verbs: []*verb{
			{"list", "list A2A adapters", "GET /a2a-adapters", a2aList},
			{"get", "get one adapter (use --show-etag)", "GET /a2a-adapters/{id}", a2aGet},
			{"create", "create an adapter", "POST /a2a-adapters", a2aCreate},
			{"update", "update an adapter (--if-match)", "PUT /a2a-adapters/{id}", a2aUpdate},
			{"delete", "delete an adapter", "DELETE /a2a-adapters/{id}", a2aDelete},
			{"deploy", "upsert an adapter from a file", "GET+PUT|POST /a2a-adapters", a2aDeploy},
		},
	}
}

func a2aList(c *cmdContext, args []string) error {
	return listVerb(c, args, "a2a-adapters list", "GET /a2a-adapters",
		"autopi a2a-adapters list | jq '[.[] | {name, id}]'", "a2a-adapters", true)
}

// a2aGet adds --show-etag so the If-Match update workflow is scriptable.
func a2aGet(c *cmdContext, args []string) error {
	fs, g := newFlagSet(c, "a2a-adapters get")
	var statusOnly, showETag bool
	fs.BoolVar(&statusOnly, "status-only", false, "print only the HTTP status code")
	fs.BoolVar(&showETag, "show-etag", false, "print the response ETag header instead of the body")
	setUsage(fs, "GET /a2a-adapters/{id}", "autopi a2a-adapters get <id> [--show-etag]",
		"autopi a2a-adapters get my-agent --show-etag")
	pos, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	id, err := requireID(pos, "adapter id")
	if err != nil {
		return err
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}
	if statusOnly && showETag {
		return usagef("--show-etag and --status-only are mutually exclusive")
	}
	req := client.Request{Method: "GET", Path: client.JoinPath("a2a-adapters", id)}
	if showETag && !cfg.PrintRequest {
		if err := cfg.Validate(); err != nil {
			return err
		}
		cl := client.New(cfg, c.stdout)
		res, err := fetch(c, cl, req)
		if err != nil {
			return err
		}
		etag := res.Header.Get("ETag")
		if etag == "" {
			return fmt.Errorf("adapter response has no ETag")
		}
		_, err = fmt.Fprintln(c.stdout, etag)
		return err
	}
	return execute(c, cfg, req, statusOnly)
}

func a2aCreate(c *cmdContext, args []string) error {
	return createVerb(c, args, "a2a-adapters create", "POST /a2a-adapters",
		"autopi a2a-adapters create -f agent.a2a.json", "a2a-adapters")
}

func a2aUpdate(c *cmdContext, args []string) error {
	return writeVerbWithID(c, args, "a2a-adapters update", "PUT", "", "PUT /a2a-adapters/{id}",
		"autopi a2a-adapters update my-agent -f agent.a2a.json --if-match $ETAG", "adapter id", "a2a-adapters")
}

func a2aDelete(c *cmdContext, args []string) error {
	return deleteVerb(c, args, "a2a-adapters delete", "DELETE /a2a-adapters/{id}",
		"autopi a2a-adapters delete my-agent", "adapter id", "a2a-adapters")
}

func a2aDeploy(c *cmdContext, args []string) error {
	return deployVerb(c, args, "a2a-adapters deploy", "a2a-adapter", "a2a-adapters")
}
