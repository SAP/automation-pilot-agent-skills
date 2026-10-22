package cmd

import "github.com/SAP/automation-pilot-agent-skills/cli/internal/client"

// requireID enforces exactly one non-empty positional argument.
func requireID(pos []string, label string) (string, error) {
	if len(pos) != 1 || pos[0] == "" {
		return "", usagef("expected exactly one %s", label)
	}
	return pos[0], nil
}

// createVerb implements POST {collection} with a request body (-f/--data).
func createVerb(c *cmdContext, args []string, name, endpoint, example, collection string) error {
	fs, g := newFlagSet(c, name)
	var file, data string
	fs.StringVar(&file, "f", "", "path to a JSON file to send as the body")
	fs.StringVar(&file, "file", "", "path to a JSON file to send as the body")
	fs.StringVar(&data, "data", "", "inline JSON body, or @file")
	setUsage(fs, endpoint, name+" (-f FILE | --data JSON)", example)
	if err := parseNoArgs(fs, args); err != nil {
		return err
	}
	body, err := resolveBody(c, file, data)
	if err != nil {
		return err
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}
	return execute(c, cfg, client.Request{Method: "POST", Path: collection, Body: body}, false)
}

// writeVerbWithID implements PUT/PATCH {collection}/{id} with a body. ctype lets
// the caller force a content type (e.g. application/merge-patch+json for PATCH).
func writeVerbWithID(c *cmdContext, args []string, name, method, ctype, endpoint, example, idLabel, collection string) error {
	fs, g := newFlagSet(c, name)
	var file, data, ifMatch string
	fs.StringVar(&file, "f", "", "path to a JSON file to send as the body")
	fs.StringVar(&file, "file", "", "path to a JSON file to send as the body")
	fs.StringVar(&data, "data", "", "inline JSON body, or @file")
	fs.StringVar(&ifMatch, "if-match", "", "optimistic-locking ETag (If-Match header)")
	setUsage(fs, endpoint, name+" <id> (-f FILE | --data JSON)", example)
	pos, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	id, err := requireID(pos, idLabel)
	if err != nil {
		return err
	}
	body, err := resolveBody(c, file, data)
	if err != nil {
		return err
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}
	if flagSet(fs, "if-match") && ifMatch == "" {
		return usagef("--if-match requires a non-empty ETag")
	}
	headers := map[string]string{}
	if ifMatch != "" {
		headers["If-Match"] = ifMatch
	}
	return execute(c, cfg, client.Request{
		Method:      method,
		Path:        client.JoinPath(collection, id),
		Body:        body,
		ContentType: ctype,
		Headers:     headers,
	}, false)
}

// deleteVerb implements DELETE {collection}/{id}.
func deleteVerb(c *cmdContext, args []string, name, endpoint, example, idLabel, collection string) error {
	fs, g := newFlagSet(c, name)
	var ifMatch string
	fs.StringVar(&ifMatch, "if-match", "", "optimistic-locking ETag (If-Match header)")
	setUsage(fs, endpoint, name+" <id> [--if-match ETAG]", example)
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
	if flagSet(fs, "if-match") && ifMatch == "" {
		return usagef("--if-match requires a non-empty ETag")
	}
	headers := map[string]string{}
	if ifMatch != "" {
		headers["If-Match"] = ifMatch
	}
	return execute(c, cfg, client.Request{Method: "DELETE", Path: client.JoinPath(collection, id), Headers: headers}, false)
}

// lifecycleVerb implements PUT {collection}/{id}/{action} (release/deprecate/restore).
func lifecycleVerb(c *cmdContext, args []string, name, action, endpoint, example, idLabel, collection string) error {
	fs, g := newFlagSet(c, name)
	setUsage(fs, endpoint, name+" <id>", example)
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
	return execute(c, cfg, client.Request{Method: "PUT", Path: client.JoinPath(collection, id, action)}, false)
}

// getVerb implements GET {collection}/{id} with optional --status-only.
func getVerb(c *cmdContext, args []string, name, endpoint, example, idLabel, collection string) error {
	fs, g := newFlagSet(c, name)
	var statusOnly bool
	fs.BoolVar(&statusOnly, "status-only", false, "print only the HTTP status code")
	setUsage(fs, endpoint, name+" <id>", example)
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
	return execute(c, cfg, client.Request{Method: "GET", Path: client.JoinPath(collection, id)}, statusOnly)
}
