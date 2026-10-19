package cmd

import (
	"encoding/json"

	"github.com/SAP/automation-pilot-agent-skills/cli/internal/client"
)

func connectionResource() *resource {
	return &resource{
		name:    "connection",
		summary: "inspect the active connection (non-secret identity & permissions)",
		verbs: []*verb{
			{"info", "connected URL, tenant, user, permissions, plan",
				"GET /server-info + /user-information + /tenant", connectionInfo},
		},
	}
}

// connectionInfo mirrors the MCP get_connection_info tool: it makes three GET
// calls and composes a single non-secret summary. The configured credential is
// never echoed; permissions come from /user-information regardless of whether
// auth is Basic (this CLI) or Bearer (the MCP server).
func connectionInfo(c *cmdContext, args []string) error {
	fs, g := newFlagSet(c, "connection info")
	setUsage(fs, "GET /server-info + /user-information + /tenant",
		"autopi connection info", "autopi connection info")
	if _, err := parseArgs(fs, args); err != nil {
		return err
	}
	cfg, err := g.config()
	if err != nil {
		return err
	}

	cl := client.New(cfg, c.stdout)

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
		res, err := cl.Do(client.Request{Method: "GET", Path: path})
		if err != nil {
			if res != nil {
				cl.Render(c.stdout, res)
			}
			return nil, err
		}
		var m map[string]any
		if len(res.Body) > 0 {
			if err := json.Unmarshal(res.Body, &m); err != nil {
				return nil, err
			}
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
	cl.Render(c.stdout, &client.Result{Status: 200, Body: out})
	return nil
}
