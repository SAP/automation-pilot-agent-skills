package cmd

import (
	"github.com/SAP/automation-pilot-agent-skills/cli/internal/client"
)

func mcpServersResource() *resource {
	return &resource{
		name:    "mcp-servers",
		summary: "manage MCP server definitions",
		verbs: []*verb{
			{"list", "list MCP servers", "GET /mcp-servers", mcpList},
			{"get", "get one MCP server", "GET /mcp-servers/{id}", mcpGet},
			{"create", "create an MCP server", "POST /mcp-servers", mcpCreate},
			{"update", "update an MCP server", "PUT /mcp-servers/{id}", mcpUpdate},
			{"delete", "delete an MCP server", "DELETE /mcp-servers/{id}", mcpDelete},
			{"deploy", "upsert an MCP server from a file", "GET+PUT|POST /mcp-servers", mcpDeploy},
		},
	}
}

func mcpList(c *cmdContext, args []string) error {
	return listVerb(c, args, "mcp-servers list", "GET /mcp-servers",
		"autopi mcp-servers list", "mcp-servers", true)
}

func mcpGet(c *cmdContext, args []string) error {
	return getVerb(c, args, "mcp-servers get", "GET /mcp-servers/{id}",
		"autopi mcp-servers get my-server", "MCP server id", "mcp-servers")
}

func mcpCreate(c *cmdContext, args []string) error {
	return createVerb(c, args, "mcp-servers create", "POST /mcp-servers",
		"autopi mcp-servers create -f server.mcp.json", "mcp-servers")
}

func mcpUpdate(c *cmdContext, args []string) error {
	return writeVerbWithID(c, args, "mcp-servers update", "PUT", "", "PUT /mcp-servers/{id}",
		"autopi mcp-servers update my-server -f server.mcp.json", "MCP server id", "mcp-servers")
}

func mcpDelete(c *cmdContext, args []string) error {
	return deleteVerb(c, args, "mcp-servers delete", "DELETE /mcp-servers/{id}",
		"autopi mcp-servers delete my-server", "MCP server id", "mcp-servers")
}

func mcpDeploy(c *cmdContext, args []string) error {
	return deployVerb(c, args, "mcp-servers deploy", "mcp-server", "mcp-servers")
}

// listVerb is a no-filter GET on a collection (for resources without catalog/tag
// query support). When summarize is set it exposes a --full flag and defaults to
// the trimmed metadata view; callers over already-light endpoints (counts,
// time-zone lists) pass summarize=false.
func listVerb(c *cmdContext, args []string, name, endpoint, example, collection string, summarize bool) error {
	fs, g := newFlagSet(c, name)
	var full bool
	if summarize {
		fs.BoolVar(&full, "full", false, "return complete objects instead of the metadata summary")
	}
	setUsage(fs, endpoint, name, example)
	if err := parseNoArgs(fs, args); err != nil {
		return err
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}
	return execute(c, cfg, client.Request{Method: "GET", Path: collection, Summarize: summarize && !full}, false)
}
