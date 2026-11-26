package main

import (
	"context"
	"path/filepath"
	"testing"
)

func TestActualCLIValidatesEveryReferenceArtifact(t *testing.T) {
	repository, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	if err = checkReferenceArtifacts(context.Background(), testBank(t), repository); err != nil {
		t.Fatal(err)
	}
}
func TestCleanupIsResumableAndDoesNotTreatAuthorizationAsAbsence(t *testing.T) {
	fakeEngineEnvironment(t)
	v := engineVerifier{runDir: t.TempDir(), journal: engineJournal{Tenant: "tenant", Catalog: "owned", Executions: []string{"already-absent", "blocked"}}}
	calls := []string{}
	v.callOverride = func(ctx context.Context, args []string, body any) (any, error) {
		call := args[0] + " " + args[1]
		calls = append(calls, call)
		if call == "connection info" {
			return map[string]any{"connected": true, "tenantId": "tenant"}, nil
		}
		if call == "executions get" {
			if args[2] == "already-absent" {
				return float64(404), nil
			}
			return float64(403), nil
		}
		t.Fatal("unscoped cleanup", args)
		return nil, nil
	}
	if err := v.cleanup(context.Background()); err == nil {
		t.Fatal("permission failure ignored")
	}
	if len(v.journal.Executions) != 1 || v.journal.Executions[0] != "blocked" {
		t.Fatal("confirmed absence not removed", v.journal.Executions)
	}
	for _, call := range calls {
		if call == "catalogs delete" {
			t.Fatal("parent deleted after incomplete cleanup")
		}
	}
}
