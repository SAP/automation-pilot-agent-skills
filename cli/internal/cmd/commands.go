package cmd

import (
	"net/url"
	"os"
	"strings"

	"github.com/SAP/automation-pilot-agent-skills/cli/internal/client"
)

func commandsResource() *resource {
	return &resource{
		name:    "commands",
		summary: "manage commands (import, export, lifecycle, bulk)",
		verbs: []*verb{
			{"list", "list commands in a catalog", "GET /commands", commandsList},
			{"ids", "list command IDs in a catalog", "GET /commands/ids", commandsIDs},
			{"get", "get one command", "GET /commands/{id}", commandsGet},
			{"create", "create a command (DRAFT)", "POST /commands", commandsCreate},
			{"update", "update a command", "PUT /commands/{id}", commandsUpdate},
			{"delete", "delete a command", "DELETE /commands/{id}", commandsDelete},
			{"release", "release a command", "PUT /commands/{id}/release", commandsRelease},
			{"deprecate", "deprecate a command", "PUT /commands/{id}/deprecate", commandsDeprecate},
			{"restore", "restore a command", "PUT /commands/{id}/restore", commandsRestore},
			{"bulk-create", "create many commands", "POST /bulk/commands", commandsBulkCreate},
			{"export", "write a command's JSON to a file", "GET /commands/{id}", commandsExport},
			{"deploy", "upsert a command from a file", "GET+PUT|POST /commands", commandsDeploy},
		},
	}
}

// listFlags are the shared filters for list-style verbs. When summarize is set
// the verb exposes a --full flag and defaults to a trimmed metadata view; the
// id-only `ids` endpoint passes summarize=false since it is already minimal.
func catalogListVerb(c *cmdContext, args []string, name, endpoint, example, collection string, includeIssues, summarize bool) error {
	fs, g := newFlagSet(c, name)
	var tag stringList
	var issues, full bool
	fs.Var(&tag, "tag", "filter by tag key:value (repeatable)")
	if includeIssues {
		fs.BoolVar(&issues, "include-issues", false, "include validation issues")
	}
	if summarize {
		fs.BoolVar(&full, "full", false, "return complete objects instead of the metadata summary")
	}
	setUsage(fs, endpoint, name+" [--catalog C] [--tag k:v]", example)
	if _, err := parseArgs(fs, args); err != nil {
		return err
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}
	// The catalog comes from the global --catalog flag or AUTOPI_DEFAULT_CATALOG.
	q := url.Values{}
	if cfg.Catalog != "" {
		q.Set("catalog", cfg.Catalog)
	}
	for _, t := range tag {
		q.Add("tag", t)
	}
	if includeIssues && issues {
		q.Set("includeIssues", "true")
	}
	req := client.Request{Method: "GET", Path: collection, Query: q, Summarize: summarize && !full}
	// /commands/ids ignores ?catalog= server-side, so scope the id list client-side.
	if strings.HasSuffix(collection, "/ids") && cfg.Catalog != "" {
		req.FilterIDPrefix = cfg.Catalog + ":"
	}
	return execute(c, cfg, req, false)
}

func commandsList(c *cmdContext, args []string) error {
	return catalogListVerb(c, args, "commands list", "GET /commands",
		"autopi commands list --catalog applm-sapcp --include-issues", "commands", true, true)
}

func commandsIDs(c *cmdContext, args []string) error {
	return catalogListVerb(c, args, "commands ids", "GET /commands/ids",
		"autopi commands ids --catalog applm-sapcp", "commands/ids", false, false)
}

func commandsGet(c *cmdContext, args []string) error {
	return getVerb(c, args, "commands get", "GET /commands/{id}",
		"autopi commands get applm-sapcp:RestartCfApp:1", "command id", "commands")
}

func commandsCreate(c *cmdContext, args []string) error {
	return createVerb(c, args, "commands create", "POST /commands",
		"autopi commands create -f MyCommand.command.json", "commands")
}

func commandsUpdate(c *cmdContext, args []string) error {
	return writeVerbWithID(c, args, "commands update", "PUT", "", "PUT /commands/{id}",
		"autopi commands update my-cat:MyCommand:1 -f MyCommand.command.json", "command id", "commands")
}

func commandsDelete(c *cmdContext, args []string) error {
	return deleteVerb(c, args, "commands delete", "DELETE /commands/{id}",
		"autopi commands delete my-cat:MyCommand:1", "command id", "commands")
}

func commandsRelease(c *cmdContext, args []string) error {
	return lifecycleVerb(c, args, "commands release", "release", "PUT /commands/{id}/release",
		"autopi commands release my-cat:MyCommand:1", "command id", "commands")
}

func commandsDeprecate(c *cmdContext, args []string) error {
	return lifecycleVerb(c, args, "commands deprecate", "deprecate", "PUT /commands/{id}/deprecate",
		"autopi commands deprecate my-cat:MyCommand:1", "command id", "commands")
}

func commandsRestore(c *cmdContext, args []string) error {
	return lifecycleVerb(c, args, "commands restore", "restore", "PUT /commands/{id}/restore",
		"autopi commands restore my-cat:MyCommand:1", "command id", "commands")
}

func commandsBulkCreate(c *cmdContext, args []string) error {
	return createVerb(c, args, "commands bulk-create", "POST /bulk/commands",
		`autopi commands bulk-create -f commands.json   # body: {"commands":[...]}`, "bulk/commands")
}

func commandsExport(c *cmdContext, args []string) error {
	return exportVerb(c, args, "commands export", "GET /commands/{id}",
		"autopi commands export my-cat:MyCommand:1 -o MyCommand.command.json", "command id", "commands")
}

func commandsDeploy(c *cmdContext, args []string) error {
	return deployVerb(c, args, "commands deploy", "command", "commands")
}

// exportVerb fetches {collection}/{id} and writes the body to -o FILE (or stdout).
func exportVerb(c *cmdContext, args []string, name, endpoint, example, idLabel, collection string) error {
	fs, g := newFlagSet(c, name)
	var out string
	fs.StringVar(&out, "o", "", "output file (default: stdout)")
	fs.StringVar(&out, "output-file", "", "output file (default: stdout)")
	setUsage(fs, endpoint, name+" <id> [-o FILE]", example)
	pos, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	id, err := requireID(pos, idLabel)
	if err != nil {
		return err
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}
	if cfg.PrintRequest {
		return execute(c, cfg, client.Request{Method: "GET", Path: client.JoinPath(collection, id)}, false)
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	cl := client.New(cfg, c.stdout)
	res, err := cl.Do(client.Request{Method: "GET", Path: client.JoinPath(collection, id)})
	if err != nil {
		return err
	}
	if out == "" {
		cl.Render(c.stdout, res)
		return nil
	}
	if err := os.WriteFile(out, res.Body, 0o644); err != nil {
		return usagef("cannot write %s: %v", out, err)
	}
	return nil
}
