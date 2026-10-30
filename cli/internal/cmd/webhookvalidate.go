package cmd

import (
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/SAP/automation-pilot-agent-skills/cli/internal/contract"
	"github.com/SAP/automation-pilot-agent-skills/cli/internal/lint"
)

func webhooksValidate(c *cmdContext, args []string) error {
	fs, g := newFlagSet(c, "webhooks validate")
	var file, metadataFile string
	fs.StringVar(&file, "f", "", "webhook definition; - for stdin")
	fs.StringVar(&file, "file", "", "webhook definition")
	fs.StringVar(&metadataFile, "metadata", "", "offline commands/inputs contract maps; otherwise read live metadata")
	setUsage(fs, "GET /commands/{id}, /inputs/{id}; no mutations", "autopi webhooks validate -f FILE [--metadata FILE]", "autopi webhooks validate -f webhook.json --metadata contracts.json")
	if err := parseNoArgs(fs, args); err != nil {
		return err
	}
	if file == "" {
		return usagef("webhook definition requires -f FILE")
	}
	if flagSet(fs, "metadata") && metadataFile == "" {
		return usagef("--metadata needs a contract file")
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}
	if cfg.PrintRequest {
		return usagef("use --metadata for offline binding validation")
	}
	body, err := resolveBody(c, file, "")
	if err != nil {
		return err
	}
	findings := []lint.Finding{}
	add := func(path, message string) {
		findings = append(findings, lint.Finding{Severity: lint.High, File: file, Path: path, Message: message})
	}
	for _, f := range contract.ValidateKind("webhook", body, false) {
		add(f.Path, f.Message)
	}
	var webhook struct {
		CommandID       string
		InputReferences []string
		InputValues     map[string]string
		EventKey        string
	}
	if len(findings) == 0 {
		if json.Unmarshal(body, &webhook) != nil {
			return usagef("invalid webhook definition")
		}
		if err := validReferences(webhook.InputReferences); err != nil {
			return err
		}
		var metadata lint.BindingMetadata
		if metadataFile != "" {
			raw, err := readFile(metadataFile)
			if err != nil {
				return err
			}
			if json.Unmarshal(raw, &metadata) != nil || metadata.Commands == nil || metadata.Inputs == nil {
				return usagef("metadata requires commands and inputs maps")
			}
		} else {
			metadata, err = fetchBindingMetadata(c.client(cfg), lint.MCPServer{MCPTools: []lint.MCPTool{{CommandID: webhook.CommandID, InputReferences: webhook.InputReferences}}})
			if err != nil {
				return err
			}
		}
		command, ok := metadata.Commands[webhook.CommandID]
		if !ok || command.InputKeys == nil {
			add("commandId", "missing command contract")
		} else {
			bound := map[string]bool{}
			for _, ref := range webhook.InputReferences {
				input, ok := metadata.Inputs[ref]
				if !ok || input.Keys == nil {
					add("inputReferences", "missing Input contract for "+ref)
					continue
				}
				names := []string{}
				for name := range input.Keys {
					names = append(names, name)
				}
				sort.Strings(names)
				for _, name := range names {
					schema := input.Keys[name]
					target, ok := command.InputKeys[name]
					if !ok {
						add("inputReferences."+name, "Input key is not in the command")
						continue
					}
					if bound[name] {
						add("inputReferences."+name, "overlapping bindings")
					}
					if schema.Type != target.Type {
						add("inputReferences."+name, "Input type differs from command")
					}
					if sensitive(schema) != sensitive(target) {
						add("inputReferences."+name, "Input sensitivity differs from command")
					}
					bound[name] = true
				}
			}
			names := []string{}
			for name := range webhook.InputValues {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				schema, ok := command.InputKeys[name]
				if !ok {
					add("inputValues."+name, "unknown command input")
					continue
				}
				if sensitive(schema) {
					add("inputValues."+name, "sensitive input must use a reusable Input")
				}
				if !validWebhookValue(webhook.InputValues[name], schema.Type) {
					add("inputValues."+name, "inline value does not match the command input type")
				}
				if bound[name] {
					add("inputValues."+name, "overlapping bindings")
				}
				bound[name] = true
			}
			if webhook.EventKey != "" {
				schema, ok := command.InputKeys[webhook.EventKey]
				if !ok {
					add("eventKey", "event input is not in the command")
				} else {
					if sensitive(schema) {
						add("eventKey", "event cannot target a sensitive input")
					}
					if bound[webhook.EventKey] {
						add("eventKey", "overlapping event binding")
					}
					bound[webhook.EventKey] = true
				}
			}
			names = nil
			for name := range command.InputKeys {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				schema := command.InputKeys[name]
				if schema.Required && !bound[name] && schema.DefaultValue == nil && schema.DefaultValueFromInput == nil {
					add("inputValues."+name, "required command input is unbound")
				}
			}
		}
	}
	enc := json.NewEncoder(c.stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(map[string]any{"valid": len(findings) == 0, "findings": findings}); err != nil {
		return err
	}
	if len(findings) > 0 {
		return &lintFindings{len(findings)}
	}
	return nil
}
func sensitive(key lint.InputKey) bool { return key.Sensitive != nil && *key.Sensitive }

// DataUtils.deserialize uses BigDecimal for numbers, strict true/false booleans,
// and JSON collections/maps. Validate the wire string without rounding or echoing it.
var decimalValue = regexp.MustCompile(`^[+-]?(?:[0-9]+(?:[.][0-9]*)?|[.][0-9]+)(?:[eE][+-]?[0-9]+)?$`)

func validWebhookValue(value, kind string) bool {
	switch kind {
	case "string", "secret":
		return true
	case "boolean":
		return strings.EqualFold(value, "true") || strings.EqualFold(value, "false")
	case "number":
		if !decimalValue.MatchString(value) {
			return false
		}
		exponent := int64(0)
		if index := strings.IndexAny(value, "eE"); index >= 0 {
			var err error
			exponent, err = strconv.ParseInt(value[index+1:], 10, 32)
			if err != nil {
				return false
			}
			value = value[:index]
		}
		scale := -exponent
		if index := strings.IndexByte(value, '.'); index >= 0 {
			scale += int64(len(value) - index - 1)
		}
		return scale >= -1<<31 && scale <= 1<<31-1
	case "array":
		var items []json.RawMessage
		return json.Unmarshal([]byte(value), &items) == nil && items != nil
	case "object":
		var fields map[string]json.RawMessage
		return json.Unmarshal([]byte(value), &fields) == nil && fields != nil
	}
	return false
}
