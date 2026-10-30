package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/SAP/automation-pilot-agent-skills/cli/internal/client"
)

func connectionResource() *resource {
	return &resource{
		name:    "connection",
		summary: "inspect the active connection (non-secret identity & permissions)",
		verbs: []*verb{
			{"preflight", "check local configuration without connecting or printing values",
				"local only (no API calls)", connectionPreflight},
			{"info", "connected URL, tenant, user, permissions, plan",
				"GET /server-info + /user-information + /tenant", connectionInfo},
		},
	}
}

// Preflight is useful before credentials exist. It checks syntax, not identity
// or permission; only connection info can establish a live account summary.
func connectionPreflight(c *cmdContext, args []string) error {
	fs := flag.NewFlagSet("connection preflight", flag.ContinueOnError)
	fs.SetOutput(c.stderr)
	setUsage(fs, "local only (no API calls)", "autopi connection preflight", "autopi connection preflight")
	if err := parseNoArgs(fs, args); err != nil {
		return err
	}
	cfg := client.FromEnv()
	err := cfg.Validate()
	result := map[string]any{
		"version":            c.version,
		"hostnamePresent":    strings.TrimSpace(cfg.Host) != "",
		"tokenPresent":       strings.TrimSpace(cfg.Token) != "",
		"configurationValid": err == nil,
		"connectionChecked":  false,
		"nextStep":           "connection info verifies tenant identity and permissions",
	}
	if err != nil {
		result["configurationError"] = err.Error()
	}
	if writeErr := json.NewEncoder(c.stdout).Encode(result); writeErr != nil {
		return writeErr
	}
	return err
}

// connectionInfo mirrors the MCP get_connection_info tool: it makes three GET
// calls and composes a single non-secret summary. The configured credential is
// never echoed; permissions come from /user-information using Bearer auth.
func connectionInfo(c *cmdContext, args []string) error {
	fs, g := newFlagSet(c, "connection info")
	setUsage(fs, "GET /server-info + /user-information + /tenant",
		"autopi connection info", "autopi connection info")
	if err := parseNoArgs(fs, args); err != nil {
		return err
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}

	cl := c.client(cfg)

	// Under --print-request, render the three calls that would be made and stop.
	if cfg.PrintRequest {
		for _, p := range []string{"server-info", "user-information", "tenant"} {
			if _, err := cl.Do(client.Request{Method: "GET", Path: p}); err != nil {
				return err
			}
		}
		return nil
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	// getObject fetches a path and decodes the JSON object body. The first
	// failure short-circuits, so reaching composition means all three returned
	// 2xx — which is what makes "connected": true meaningful.
	getObject := func(path string) (map[string]any, error) {
		res, err := fetch(c, cl, client.Request{Method: "GET", Path: path})
		if err != nil {
			return nil, err
		}
		var m map[string]any
		if len(res.Body) > 0 {
			if err := json.Unmarshal(res.Body, &m); err != nil {
				return nil, err
			}
		}
		if m == nil {
			return nil, fmt.Errorf("%s returned no identity object", path)
		}
		return m, nil
	}

	serverInfo, err := getObject("server-info")
	if err != nil {
		return err
	}
	userInfo, err := getObject("user-information")
	if err != nil {
		return err
	}
	tenant, err := getObject("tenant")
	if err != nil {
		return err
	}

	summary := map[string]any{
		"connected":   true,
		"url":         serverInfo["autoPiTenantUrl"],
		"tenantId":    tenant["id"],
		"username":    userInfo["username"],
		"permissions": userInfo["permissions"],
		"plan":        userInfo["appPlan"],
	}
	out, err := json.Marshal(summary)
	if err != nil {
		return err
	}
	return cl.Render(c.stdout, &client.Result{Status: 200, Body: out})
}
