package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakeEngineEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("AUTOMATION_PILOT_HOSTNAME", "fixture.invalid")
	t.Setenv("AUTOMATION_PILOT_TOKEN", "synthetic-only-token")
}

func TestEngineResolvesEnvironmentTenantAndTreatsFlagAsAssertion(t *testing.T) {
	fakeEngineEnvironment(t)
	for _, expected := range []string{"", "configured", "other"} {
		v := engineVerifier{}
		calls := 0
		v.callOverride = func(ctx context.Context, args []string, body any) (any, error) {
			calls++
			if strings.Join(args, " ") != "connection info" || body != nil {
				t.Fatal("resolution attempted a mutation", args)
			}
			return map[string]any{"connected": true, "tenantId": "configured"}, nil
		}
		err := v.resolveTenant(context.Background(), expected)
		if expected == "other" {
			if err == nil || v.tenant != "" {
				t.Fatal("identity flag selected another tenant")
			}
		} else if err != nil || v.tenant != "configured" {
			t.Fatal(v.tenant, err)
		}
		if calls != 1 {
			t.Fatal(calls)
		}
	}
}

func TestMissingOrUnverifiedEngineConnectionNeverMutates(t *testing.T) {
	fakeEngineEnvironment(t)
	for _, identity := range []any{nil, map[string]any{"tenantId": "configured"}, map[string]any{"connected": false, "tenantId": "configured"}, map[string]any{"connected": true}, map[string]any{"connected": true, "tenantId": 7}} {
		v := engineVerifier{runDir: t.TempDir()}
		v.callOverride = func(ctx context.Context, args []string, body any) (any, error) {
			if strings.Join(args, " ") != "connection info" {
				t.Fatal("unverified identity mutated tenant", args)
			}
			return identity, nil
		}
		if v.initialize(context.Background()) == nil {
			t.Fatal("unverified connection accepted")
		}
	}
	t.Setenv("AUTOMATION_PILOT_TOKEN", "")
	v := engineVerifier{}
	v.callOverride = func(context.Context, []string, any) (any, error) {
		t.Fatal("missing credentials caused an API request")
		return nil, nil
	}
	if v.resolveTenant(context.Background(), "") == nil {
		t.Fatal("missing token accepted")
	}
}

func TestEngineInitializationWithoutTenantFlagScopesOwnedCatalog(t *testing.T) {
	fakeEngineEnvironment(t)
	v := engineVerifier{runDir: t.TempDir()}
	v.callOverride = func(ctx context.Context, args []string, body any) (any, error) {
		switch strings.Join(args[:2], " ") {
		case "connection info":
			return map[string]any{"connected": true, "tenantId": "configured"}, nil
		case "commands contract":
			return map[string]any{}, nil
		case "catalogs create":
			if !strings.HasPrefix(body.(map[string]any)["technicalName"].(string), "ev") {
				t.Fatal("catalog is not run-owned")
			}
			return map[string]any{"id": "owned"}, nil
		}
		t.Fatal(args)
		return nil, nil
	}
	if err := v.initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	if v.tenant != "configured" || v.journal.Tenant != "configured" || v.journal.Catalog != "owned" {
		t.Fatal(v.journal)
	}
}

func TestEnginePreflightStopsBeforeModelSessionsOrRunDirectory(t *testing.T) {
	t.Setenv("AUTOMATION_PILOT_HOSTNAME", "")
	t.Setenv("AUTOMATION_PILOT_TOKEN", "")
	dir := filepath.Join(t.TempDir(), "not-started")
	h := hostConfig{Kind: "claude", Program: "must-never-be-started", Isolation: "native"}
	_, err := runBank(context.Background(), testBank(t), runOptions{Repository: "unused", Directory: dir, Suite: "smoketest", AgentHost: h, GraderHost: h, Ablation: "with-skills", Runs: 1, PerCase: 1, PerGrade: .25})
	if err == nil || !strings.Contains(err.Error(), "engine preflight failed before agent sessions") {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("preflight failure created a run directory")
	}
}

func TestCleanupVerifiesEnvironmentAgainstJournalBeforeDeletion(t *testing.T) {
	fakeEngineEnvironment(t)
	v := engineVerifier{journal: engineJournal{Tenant: "old-tenant", Catalog: "owned"}}
	v.callOverride = func(ctx context.Context, args []string, body any) (any, error) {
		if strings.Join(args, " ") != "connection info" {
			t.Fatal("cleanup deleted from the wrong tenant", args)
		}
		return map[string]any{"connected": true, "tenantId": "configured"}, nil
	}
	if v.cleanup(context.Background()) == nil {
		t.Fatal("journal tenant mismatch ignored")
	}
	v.journal = engineJournal{Tenant: "configured", Pending: "create catalog owned"}
	if err := v.cleanup(context.Background()); err == nil || !strings.Contains(err.Error(), "reconciliation") {
		t.Fatal("pending mutation disappeared", err)
	}
}
