package cmd

import "github.com/SAP/automation-pilot-agent-skills/cli/internal/client"

func webhooksResource() *resource {
	return &resource{name: "webhooks", summary: "manage and trigger webhooks", verbs: []*verb{
		{"list", "list webhooks", "GET /webhooks", webhooksList},
		{"get", "get a webhook and its ETag", "GET /webhooks/{id}", webhooksGet},
		{"create", "create a webhook", "POST /webhooks", webhooksCreate},
		{"update", "replace a webhook", "PUT /webhooks/{id}", webhooksUpdate},
		{"delete", "delete a webhook", "DELETE /webhooks/{id}", webhooksDelete},
		{"trigger", "trigger with an optional arbitrary event body", "POST /webhooks/{id}/trigger", webhooksTrigger},
		{"validate", "read-only verification of command/Input bindings", "GET /commands/{id}, /inputs/{id}", webhooksValidate},
	}}
}
func webhooksList(c *cmdContext, args []string) error {
	return listVerb(c, args, "webhooks list", "GET /webhooks", "autopi webhooks list", "webhooks", true)
}
func webhooksGet(c *cmdContext, args []string) error {
	return getVerb(c, args, "webhooks get", "GET /webhooks/{id}", "autopi webhooks get webhook-id --show-etag", "webhook id", "webhooks")
}
func webhooksCreate(c *cmdContext, args []string) error {
	return createVerb(c, args, "webhooks create", "POST /webhooks", "autopi webhooks create -f webhook.json", "webhooks")
}
func webhooksUpdate(c *cmdContext, args []string) error {
	return writeVerbWithID(c, args, "webhooks update", "PUT", "", "PUT /webhooks/{id}", "autopi webhooks update webhook-id -f webhook.json --if-match ETAG", "webhook id", "webhooks")
}
func webhooksDelete(c *cmdContext, args []string) error {
	return deleteVerb(c, args, "webhooks delete", "DELETE /webhooks/{id}", "autopi webhooks delete webhook-id", "webhook id", "webhooks")
}
func webhooksTrigger(c *cmdContext, args []string) error {
	fs, g := newFlagSet(c, "webhooks trigger")
	var file, data, contentType string
	fs.StringVar(&file, "f", "", "optional event file; - for stdin")
	fs.StringVar(&file, "file", "", "optional event file")
	fs.StringVar(&data, "data", "", "event bytes or @file")
	fs.StringVar(&contentType, "content-type", "", "media type for the event")
	setUsage(fs, "POST /webhooks/{id}/trigger", "autopi webhooks trigger <id> [-f FILE | --data EVENT]", "autopi webhooks trigger webhook-id -f event.txt --content-type text/plain")
	pos, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	id, err := requireID(pos, "webhook id")
	if err != nil {
		return err
	}
	var body []byte
	if file != "" || data != "" {
		body, err = resolveBody(c, file, data)
		if err != nil {
			return err
		}
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}
	return execute(c, cfg, client.Request{Method: "POST", Path: client.JoinPath("webhooks", id, "trigger"), Body: body, ContentType: contentType}, false)
}
