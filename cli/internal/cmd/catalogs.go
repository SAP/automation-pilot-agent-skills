package cmd

import (
	"net/url"

	"github.com/SAP/automation-pilot-agent-skills/cli/internal/client"
)

func catalogsResource() *resource {
	return &resource{
		name:    "catalogs",
		summary: "list, inspect, and manage catalogs",
		verbs: []*verb{
			{"list", "list catalogs", "GET /catalogs", catalogsList},
			{"get", "get one catalog", "GET /catalogs/{id}", catalogsGet},
			{"create", "create a catalog", "POST /catalogs", catalogsCreate},
			{"update", "update a catalog", "PUT /catalogs/{id}", catalogsUpdate},
			{"delete", "delete a catalog", "DELETE /catalogs/{id}", catalogsDelete},
		},
	}
}

func catalogsList(c *cmdContext, args []string) error {
	fs, g := newFlagSet(c, "catalogs list")
	var own, provided, full bool
	var tag stringList
	fs.BoolVar(&own, "own", false, "only catalogs owned by this tenant")
	fs.BoolVar(&provided, "provided", false, "only catalogs provided by SAP")
	fs.Var(&tag, "tag", "filter by tag key=value (repeatable)")
	fs.BoolVar(&full, "full", false, "return complete objects instead of the metadata summary")
	setUsage(fs, "GET /catalogs", "autopi catalogs list [--own|--provided] [--tag k=v]", "autopi catalogs list --own")
	if err := parseNoArgs(fs, args); err != nil {
		return err
	}
	if own && provided {
		return usagef("--own and --provided are mutually exclusive")
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}
	q := url.Values{}
	if own {
		q.Set("own", "true")
	}
	if provided {
		q.Set("provided", "true")
	}
	addTagFilters(q, tag)
	return execute(c, cfg, client.Request{Method: "GET", Path: "catalogs", Query: q, Summarize: !full}, false)
}

func catalogsGet(c *cmdContext, args []string) error {
	return getVerb(c, args, "catalogs get", "GET /catalogs/{id}",
		"autopi catalogs get my-catalog", "catalog id", "catalogs")
}

func catalogsCreate(c *cmdContext, args []string) error {
	return createVerb(c, args, "catalogs create", "POST /catalogs",
		"autopi catalogs create -f catalog.json", "catalogs")
}

func catalogsUpdate(c *cmdContext, args []string) error {
	return writeVerbWithID(c, args, "catalogs update", "PUT", "", "PUT /catalogs/{id}",
		"autopi catalogs update my-catalog -f catalog.json", "catalog id", "catalogs")
}

func catalogsDelete(c *cmdContext, args []string) error {
	fs, g := newFlagSet(c, "catalogs delete")
	var force bool
	fs.BoolVar(&force, "force", false, "delete contained Commands and Inputs; return deleted counts")
	setUsage(fs, "DELETE /catalogs/{id}", "autopi catalogs delete <id> [--force]", "autopi catalogs delete my-catalog")
	pos, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	id, err := requireID(pos, "catalog id")
	if err != nil {
		return err
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}
	q := url.Values{}
	if force {
		q.Set("force", "true")
	}
	return execute(c, cfg, client.Request{Method: "DELETE", Path: client.JoinPath("catalogs", id), Query: q}, false)
}
