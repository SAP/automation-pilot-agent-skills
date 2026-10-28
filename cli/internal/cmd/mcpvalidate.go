package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/SAP/automation-pilot-agent-skills/cli/internal/client"
	"github.com/SAP/automation-pilot-agent-skills/cli/internal/lint"
)

// mcpValidate owns both offline and read-only live binding verification.
func mcpValidate(c *cmdContext, args []string) error {
	fs, g := newFlagSet(c, "mcp-servers validate")
	var file, metadataFile, clientFile string
	fs.StringVar(&file, "f", "", "MCP definition file (or - for stdin)")
	fs.StringVar(&file, "file", "", "MCP definition file (or - for stdin)")
	fs.StringVar(&metadataFile, "metadata", "", "offline command/Input contract map; otherwise read live contracts")
	fs.StringVar(&clientFile, "client-schema", "", "optional captured tools/list response")
	setUsage(fs, "GET /commands/{id}, /inputs/{id}; no API calls with --metadata",
		"autopi mcp-servers validate -f FILE [--metadata FILE] [--client-schema FILE]",
		"autopi mcp-servers validate -f server.mcp.json --metadata contracts.json --client-schema tools-list.json")
	if err := parseNoArgs(fs, args); err != nil {
		return err
	}
	if file == "" {
		return usagef("MCP definition is required: pass -f FILE")
	}
	if flagSet(fs, "metadata") && metadataFile == "" {
		return usagef("--metadata needs a contract file")
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}
	if cfg.PrintRequest {
		return usagef("validation does not send mutations; use --metadata for offline validation instead of --print-request")
	}
	body, err := resolveBody(c, file, "")
	if err != nil {
		return err
	}
	findings := lint.CheckMCPDefinition(file, body)
	if len(findings) != 0 {
		return writeMCPReport(c, lint.MCPReport{Findings: findings, Tools: []lint.ToolContract{}})
	}
	var clientSchema []byte
	if clientFile != "" {
		clientSchema, err = readFile(clientFile)
		if err != nil {
			return err
		}
	}
	var metadata lint.BindingMetadata
	if metadataFile != "" {
		data, err := readFile(metadataFile)
		if err != nil {
			return err
		}
		if json.Unmarshal(data, &metadata) != nil || metadata.Commands == nil || metadata.Inputs == nil {
			return usagef("metadata must contain commands and inputs contract maps")
		}
	} else {
		var server lint.MCPServer
		if err := json.Unmarshal(body, &server); err != nil {
			return usagef("invalid MCP definition")
		}
		metadata, err = fetchBindingMetadata(c.client(cfg), server)
		if err != nil {
			return err
		}
	}
	return writeMCPReport(c, lint.CheckMCPBindings(file, body, metadata, clientSchema))
}

func writeMCPReport(c *cmdContext, report lint.MCPReport) error {
	encoder := json.NewEncoder(c.stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return err
	}
	if len(report.Findings) != 0 {
		return &lintFindings{n: len(report.Findings)}
	}
	return nil
}

func fetchBindingMetadata(cl *client.Client, server lint.MCPServer) (lint.BindingMetadata, error) {
	metadata := lint.BindingMetadata{Commands: map[string]lint.BindingContract{}, Inputs: map[string]lint.BindingContract{}}
	read := func(resource, identity string) (lint.BindingContract, error) {
		var contract lint.BindingContract
		res, err := cl.Do(client.Request{Method: "GET", Path: client.JoinPath(resource, identity)})
		if err != nil {
			// A command error can echo data; preserve HTTP classification without the body.
			var httpError *client.HTTPError
			if asHTTP(err, &httpError) {
				return contract, &client.HTTPError{Status: httpError.Status}
			}
			return contract, err
		}
		if res == nil || json.Unmarshal(res.Body, &contract) != nil {
			return contract, fmt.Errorf("invalid %s contract response", resource)
		}
		return contract, nil
	}
	for _, tool := range server.MCPTools {
		if _, present := metadata.Commands[tool.CommandID]; !present {
			contract, err := read("commands", tool.CommandID)
			if err != nil {
				return metadata, err
			}
			metadata.Commands[tool.CommandID] = contract
		}
		for _, identity := range tool.InputReferences {
			if _, present := metadata.Inputs[identity]; present {
				continue
			}
			contract, err := read("inputs", identity)
			if err != nil {
				return metadata, err
			}
			metadata.Inputs[identity] = contract
		}
	}
	return metadata, nil
}
