package main

import (
	"context"
	"fmt"
	"testing"
)

func TestNativeAcceptanceRejectsExternalOperationsAndPreservesCandidate(t *testing.T) {
	c := findCase(t, "native-iterator-wrapper")
	if err := validateNativeCandidate(c.Reference.Artifact); err != nil {
		t.Fatal(err)
	}
	artifact := jsonValue(c.Reference.Artifact).(map[string]any)
	config := artifact["configuration"].(map[string]any)
	steps := config["executors"].([]any)
	steps[0].(map[string]any)["execute"] = "scripts-sapcp:ExecuteScript:2"
	if validateNativeCandidate(artifact) == nil {
		t.Fatal("script admitted to controlled acceptance")
	}
}
func TestEngineOutputEnvelopeAndAmbiguousTriggerJournal(t *testing.T) {
	c := findCase(t, "preserve-policy-false")
	v := engineVerifier{runDir: t.TempDir(), journal: engineJournal{Tenant: "fixture", Catalog: "eval-fixture"}}
	index := -1
	calls := 0
	v.callOverride = func(ctx context.Context, args []string, body any) (any, error) {
		switch args[0] + " " + args[1] {
		case "lint --kind":
			return []any{}, nil
		case "commands create":
			return map[string]any{"id": body.(map[string]any)["id"]}, nil
		case "executions trigger":
			index++
			return map[string]any{"id": fmt.Sprintf("exec-%d", index)}, nil
		case "executions get":
			return map[string]any{"status": "FINISHED"}, nil
		case "executions output":
			value := c.Engine.Partitions[index].Expected
			return map[string]any{"values": map[string]any{"result": fmt.Sprint(value)}}, nil
		default:
			t.Fatal(args)
			return nil, nil
		}
	}
	original := digest(c.Reference.Artifact)
	out, err := v.evaluate(context.Background(), c, c.Reference.Artifact)
	if err != nil {
		t.Fatal(err)
	}
	if out["partition-3"] != false || out["partition-1"] != true {
		t.Fatal("wire boolean or values envelope lost", out)
	}
	if digest(c.Reference.Artifact) != original {
		t.Fatal("submitted artifact modified")
	}
	v.journal.Pending = ""
	v.callOverride = func(ctx context.Context, args []string, body any) (any, error) {
		if args[0] == "executions" && args[1] == "trigger" {
			calls++
			return nil, fmt.Errorf("connection lost after submission")
		}
		return map[string]any{"id": "created"}, nil
	}
	if _, err = v.evaluate(context.Background(), c, c.Reference.Artifact); err == nil {
		t.Fatal("ambiguous trigger accepted")
	}
	if v.journal.Pending == "" {
		t.Fatal("ambiguous mutation not retained")
	}
	if _, err = v.evaluate(context.Background(), c, c.Reference.Artifact); err == nil || calls != 1 {
		t.Fatal("trigger replayed", calls)
	}
}

func TestEngineRejectsInvalidPersistedWireAsCandidateFailure(t *testing.T) {
	c := findCase(t, "binding-count-zero")
	v := engineVerifier{runDir: t.TempDir(), journal: engineJournal{Tenant: "fixture", Catalog: "owned"}}
	v.callOverride = func(ctx context.Context, args []string, body any) (any, error) {
		switch args[0] + " " + args[1] {
		case "lint --kind":
			return []any{}, nil
		case "commands create":
			return map[string]any{"id": "created"}, nil
		case "executions trigger":
			return map[string]any{"id": "exec"}, nil
		case "executions get":
			return map[string]any{"status": "FINISHED"}, nil
		case "executions output":
			return map[string]any{"values": map[string]any{"result": nil}}, nil
		}
		t.Fatal(args)
		return nil, nil
	}
	out, err := v.evaluate(context.Background(), c, map[string]any{"expression": `$("incorrect")`})
	if err != nil {
		t.Fatal(err)
	}
	scores := deterministic(c, evidence{EngineResults: out})
	if aggregate(scores) != "FAIL" {
		t.Fatal("invalid candidate wire output was not rejected", out)
	}
}

func TestRejectedCreateClearsJournalOnlyAfterVerifiedAbsence(t *testing.T) {
	for _, status := range []float64{404, 403} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			c := findCase(t, "binding-count-zero")
			v := engineVerifier{runDir: t.TempDir(), journal: engineJournal{Tenant: "fixture", Catalog: "owned"}}
			v.callOverride = func(ctx context.Context, args []string, body any) (any, error) {
				switch args[0] + " " + args[1] {
				case "lint --kind":
					return []any{}, nil
				case "commands create":
					return nil, &engineHTTPError{Status: 400, Err: fmt.Errorf("rejected")}
				case "commands get":
					return status, nil
				}
				t.Fatal(args)
				return nil, nil
			}
			out, err := v.evaluate(context.Background(), c, map[string]any{"expression": `$("incorrect")`})
			if err == nil {
				t.Fatal("rejection disappeared")
			}
			if status == 404 && (v.journal.Pending != "" || len(out) == 0) {
				t.Fatal("verified absent rejection remained ambiguous", v.journal, out)
			}
			if status == 403 && (v.journal.Pending == "" || out != nil) {
				t.Fatal("authorization error treated as absence")
			}
		})
	}
}
