package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
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
			{"contract", "get a command's contract (strips configuration)", "GET /commands/{id}", commandsContract},
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
	fs.StringVar(&g.catalog, "catalog", "", "filter by catalog")
	var tag stringList
	var issues, full bool
	fs.Var(&tag, "tag", "filter by tag key=value (repeatable)")
	if includeIssues {
		fs.BoolVar(&issues, "include-issues", false, "include validation issues")
	}
	if summarize {
		fs.BoolVar(&full, "full", false, "return complete objects instead of the metadata summary")
	}
	setUsage(fs, endpoint, name+" [--catalog C] [--tag k=v]", example)
	if err := parseNoArgs(fs, args); err != nil {
		return err
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}
	// The catalog filter comes from the global --catalog flag.
	q := url.Values{}
	if cfg.Catalog != "" {
		q.Set("catalog", cfg.Catalog)
	}
	addTagFilters(q, tag)
	if includeIssues && issues {
		q.Set("includeIssues", "true")
	}
	req := client.Request{Method: "GET", Path: collection, Query: q, Summarize: summarize && !full}
	// /commands/ids ignores ?catalog= server-side, so scope the id list client-side.
	if strings.HasSuffix(collection, "/ids") && cfg.Catalog != "" {
		req.FilterIDPrefix = cfg.Catalog + ":"
	}
	if strings.HasSuffix(collection, "/ids") {
		if len(tag) == 0 {
			req.Query = nil
		} else {
			req.Path = "commands"
			req.Project = func(body []byte) ([]byte, error) {
				var commands []map[string]any
				if err := json.Unmarshal(body, &commands); err != nil || commands == nil {
					return nil, fmt.Errorf("expected command list for ID filtering")
				}
				ids := []string{}
				for _, command := range commands {
					id, ok := command["id"].(string)
					if !ok {
						return nil, fmt.Errorf("command list item has no ID")
					}
					ids = append(ids, id)
				}
				return json.Marshal(ids)
			}
		}
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

// commandsContract mirrors the MCP get_command_contract tool: GET the command,
// then project it to its caller-facing contract (strips configuration while
// keeping inputKeys/outputKeys). The list summarize path is unusable here
// because it also strips inputKeys/outputKeys, so projection runs on the body.
func commandsContract(c *cmdContext, args []string) error {
	fs, g := newFlagSet(c, "commands contract")
	setUsage(fs, "GET /commands/{id}", "autopi commands contract <id>",
		"autopi commands contract applm-sapcp:RestartCfApp:1")
	pos, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	id, err := requireID(pos, "command id")
	if err != nil {
		return err
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}
	req := client.Request{Method: "GET", Path: client.JoinPath("commands", id)}
	req.Project = client.CommandContract
	return execute(c, cfg, req, false)
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
	cl := c.client(cfg)
	res, err := fetch(c, cl, client.Request{Method: "GET", Path: client.JoinPath(collection, id)})
	if err != nil {
		return err
	}
	if out == "" {
		return cl.Render(c.stdout, res)
	}
	var object map[string]any
	if json.Unmarshal(res.Body, &object) != nil || object == nil || object["name"] == nil {
		return fmt.Errorf("export returned an invalid command object; existing file preserved")
	}
	return atomicWrite(out, res.Body)
}

func atomicWrite(path string, body []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".autopi-export-*")
	if err != nil {
		return fmt.Errorf("cannot create export file: %w", err)
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(body); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return fmt.Errorf("cannot replace export file: %w", err)
	}
	return nil
}
