package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/SAP/automation-pilot-agent-skills/cli/internal/client"
)

// extractID pulls a top-level "id" string from a JSON body, if present. Used by
// deploy to derive the resource id for mcp-servers / a2a-adapters whose files
// embed their id. Commands/inputs usually need an explicit --id.
func extractID(body []byte) string {
	var m struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(body, &m) == nil {
		return m.ID
	}
	return ""
}

// deployVerb implements the upsert pattern that previously lived as a bash
// existence-check + PUT/POST block in the skills: GET {collection}/{id}; if it
// exists, PUT {collection}/{id}; otherwise POST {collection}.
func deployVerb(c *cmdContext, args []string, name, label, collection string) error {
	fs, g := newFlagSet(c, name)
	var file, data, id, ifMatch string
	fs.StringVar(&file, "f", "", "path to the JSON file to deploy")
	fs.StringVar(&file, "file", "", "path to the JSON file to deploy")
	fs.StringVar(&data, "data", "", "inline JSON body, or @file")
	fs.StringVar(&id, "id", "", "resource id (derived from the file's \"id\" when omitted)")
	fs.StringVar(&ifMatch, "if-match", "", "optimistic-locking ETag for the update path")
	setUsage(fs, "GET then PUT|POST /"+collection,
		name+" -f FILE [--id ID]",
		name+" -f my."+label+".json")
	if _, err := parseArgs(fs, args); err != nil {
		return err
	}
	body, err := resolveBody(file, data)
	if err != nil {
		return err
	}
	if id == "" {
		id = extractID(body)
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}

	postReq := client.Request{Method: "POST", Path: collection, Body: body}

	// No id to key on (e.g. mcp-servers / a2a-adapters whose identity is
	// system-generated and absent from the file on first deploy): there is
	// nothing to GET or PUT against, so create. Pass --id to force the update
	// path once the resource has a server-assigned id.
	if id == "" {
		if cfg.PrintRequest {
			cl := client.New(cfg, c.stdout)
			fmt.Fprintln(c.stdout, "# no id -> create:")
			cl.Do(postReq)
			return nil
		}
		if err := cfg.Validate(); err != nil {
			return err
		}
		fmt.Fprintf(c.stderr, "no %s id; creating\n", label)
		return execute(c, cfg, postReq, false)
	}

	putHeaders := map[string]string{}
	if ifMatch != "" {
		putHeaders["If-Match"] = ifMatch
	}
	getReq := client.Request{Method: "GET", Path: client.JoinPath(collection, id)}
	putReq := client.Request{Method: "PUT", Path: client.JoinPath(collection, id), Body: body, Headers: putHeaders}

	if cfg.PrintRequest {
		cl := client.New(cfg, c.stdout)
		fmt.Fprintln(c.stdout, "# existence check:")
		cl.Do(getReq)
		fmt.Fprintln(c.stdout, "\n# if present -> update:")
		cl.Do(putReq)
		fmt.Fprintln(c.stdout, "\n# else -> create:")
		cl.Do(postReq)
		return nil
	}

	if err := cfg.Validate(); err != nil {
		return err
	}
	cl := client.New(cfg, c.stdout)
	res, err := cl.Do(getReq)
	exists := err == nil && res != nil && res.Status >= 200 && res.Status < 300
	if exists {
		fmt.Fprintf(c.stderr, "%s %s exists; updating\n", label, id)
		return execute(c, cfg, putReq, false)
	}
	fmt.Fprintf(c.stderr, "%s %s not found; creating\n", label, id)
	return execute(c, cfg, postReq, false)
}
