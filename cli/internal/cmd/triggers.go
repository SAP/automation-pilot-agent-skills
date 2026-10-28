package cmd

import (
	"fmt"
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
	var inputRefs, tags stringList
	var correlationID, eventKey, contentType, subaccount, resourceGroup, resourceName, resourceInstance string
	var disableResource, disableRegion bool
	fs.StringVar(&file, "f", "", "path to a JSON file to send as the event body")
	fs.StringVar(&file, "file", "", "path to a JSON file to send as the event body")
	fs.StringVar(&data, "data", "", "inline JSON event body, or @file")
	fs.StringVar(&commandRef, "command-reference", "", "command id to trigger (required)")
	fs.Var(&inputRefs, "input-reference", "reusable Input ID (repeatable)")
	fs.Var(&tags, "tag", "execution tag key=value (repeatable)")
	fs.StringVar(&correlationID, "correlation-id", "", "execution correlation ID")
	fs.StringVar(&eventKey, "map-event-to-input-key", "", "forward the event to this command input key")
	ans := collection == "triggers/ans-event"
	if ans {
		fs.StringVar(&subaccount, "sub-account", "", "affected subaccount override")
		fs.StringVar(&resourceGroup, "resource-group", "", "resource group override")
		fs.StringVar(&resourceName, "resource-name", "", "resource name override")
		fs.StringVar(&resourceInstance, "resource-instance", "", "resource instance override")
		fs.BoolVar(&disableResource, "disable-resource-properties-mapping", false, "disable event-derived resource properties")
		fs.BoolVar(&disableRegion, "disable-region-properties-mapping", false, "disable event-derived region properties")
	} else {
		fs.StringVar(&contentType, "content-type", "", "media type for an arbitrary generic-event body")
	}
	setUsage(fs, endpoint, name+" --command-reference ID [-f FILE | --data PAYLOAD]", example)
	if err := parseNoArgs(fs, args); err != nil {
		return err
	}
	if commandRef == "" {
		return usagef("missing --command-reference")
	}
	var body []byte
	var err error
	if file != "" || data != "" || ans {
		body, err = resolveBody(c, file, data)
		if err != nil {
			return err
		}
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}
	q := url.Values{}
	q.Set("commandReference", commandRef)
	for _, ref := range inputRefs {
		q.Add("inputReference", ref)
	}
	for _, tag := range tags {
		q.Add("tag", tag)
	}
	for key, value := range map[string]string{"correlationId": correlationID, "mapEventToInputKey": eventKey, "subAccount": subaccount, "resourceGroup": resourceGroup, "resourceName": resourceName, "resourceInstance": resourceInstance} {
		if value != "" {
			q.Set(key, value)
		}
	}
	if flagSet(fs, "disable-resource-properties-mapping") {
		q.Set("disableResourcePropertiesMapping", fmt.Sprint(disableResource))
	}
	if flagSet(fs, "disable-region-properties-mapping") {
		q.Set("disableRegionPropertiesMapping", fmt.Sprint(disableRegion))
	}
	return execute(c, cfg, client.Request{Method: "POST", Path: collection, Query: q, Body: body, ContentType: contentType}, false)
}
