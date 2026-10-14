package cmd

import (
	"net/url"

	"github.com/SAP/automation-pilot-agent-skills/cli/internal/client"
)

func triggersResource() *resource {
	return &resource{
		name:    "triggers",
		summary: "post event triggers",
		verbs: []*verb{
			{"generic-event", "post a generic event", "POST /triggers/generic-event", triggersGeneric},
			{"ans-event", "post an ANS event", "POST /triggers/ans-event", triggersANS},
		},
	}
}

func triggersGeneric(c *cmdContext, args []string) error {
	return triggerVerb(c, args, "triggers generic-event", "POST /triggers/generic-event",
		`autopi triggers generic-event --command-reference my-cat:MyCommand:1 --data '{"type":"alert"}'`,
		"triggers/generic-event")
}

func triggersANS(c *cmdContext, args []string) error {
	return triggerVerb(c, args, "triggers ans-event", "POST /triggers/ans-event",
		`autopi triggers ans-event --command-reference my-cat:MyCommand:1 -f ans-event.json`,
		"triggers/ans-event")
}

// triggerVerb implements POST {collection}?commandReference=<id> with the event
// payload as the body. The API requires commandReference as a query parameter —
// sending it inside the body returns 400 "Required request parameter
// 'commandReference' ... not present".
func triggerVerb(c *cmdContext, args []string, name, endpoint, example, collection string) error {
	fs, g := newFlagSet(c, name)
	var file, data, commandRef string
	fs.StringVar(&file, "f", "", "path to a JSON file to send as the event body")
	fs.StringVar(&file, "file", "", "path to a JSON file to send as the event body")
	fs.StringVar(&data, "data", "", "inline JSON event body, or @file")
	fs.StringVar(&commandRef, "command-reference", "", "command id to trigger (required)")
	setUsage(fs, endpoint, name+" --command-reference ID (-f FILE | --data JSON)", example)
	if _, err := parseArgs(fs, args); err != nil {
		return err
	}
	if commandRef == "" {
		return usagef("missing --command-reference")
	}
	body, err := resolveBody(file, data)
	if err != nil {
		return err
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}
	q := url.Values{}
	q.Set("commandReference", commandRef)
	return execute(c, cfg, client.Request{Method: "POST", Path: collection, Query: q, Body: body}, false)
}
