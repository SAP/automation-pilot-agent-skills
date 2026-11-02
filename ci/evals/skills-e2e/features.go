package main

// Feature probes supplement suite.json. They only operate on resources created by
// this run. The durable mutation journal prevents replay after process interruption.
import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type probeFailure struct{ err error }
type probe struct{ h *harness }

func (p probe) prefix() string {
	if p.h.featureRound == "" {
		return "feature-"
	}
	return "feature-" + strings.ToLower(p.h.featureRound) + "-"
}
func (p probe) namespace() string { return p.h.state.Namespace + p.h.featureRound }
func (p probe) executionKey(label string) string {
	return "feature-execution|" + p.h.featureRound + label
}
func (p probe) remember(resource, id string) {
	p.h.state.Resources["feature-resource|"+resource+"|"+id] = id
	p.must(p.h.save())
}

type mutationResult struct {
	Response      any
	Code          int
	Error         string
	RequestSHA256 string
}

func (p probe) require(ok bool, message string) {
	if !ok {
		panic(probeFailure{errors.New(message)})
	}
}
func (p probe) must(err error) {
	if err != nil {
		panic(probeFailure{err})
	}
}
func (p probe) invoke(label string, args []string, body any, expected int) any {
	mutation := len(args) > 1 && (args[0] == "triggers" || args[1] == "create" || args[1] == "deploy" || args[1] == "update" || args[1] == "patch" || args[1] == "delete" || args[1] == "trigger" || args[1] == "action" || args[1] == "bulk-action" || args[1] == "bulk-create" || args[1] == "release" || args[1] == "deprecate" || args[1] == "restore" || args[1] == "enable" || args[1] == "disable")
	key := p.prefix() + label
	cache := filepath.Join(p.h.dir, "mutations", key+".json")
	var saved mutationResult
	requestBytes, err := json.Marshal(struct {
		Args []string
		Body any
	}{args, body})
	p.must(err)
	requestSum := sha256.Sum256(requestBytes)
	requestHash := hex.EncodeToString(requestSum[:])
	cached := false
	if mutation {
		err := read(cache, &saved)
		if err == nil {
			p.require(saved.RequestSHA256 == requestHash, "Mutation request changed since submission: "+key)
			cached = true
		} else if !os.IsNotExist(err) {
			p.must(err)
		}
		if !cached {
			p.require(p.h.state.Attempts[key] == "", "Unresolved mutation "+key+"; reconcile evidence before resuming")
			p.must(p.h.submit(key))
		}
	}
	if !cached {
		saved.RequestSHA256 = requestHash
		response, err := p.h.call(key, args, body)
		saved.Response = response
		if err != nil {
			var ce *callError
			if errors.As(err, &ce) {
				saved.Code = ce.code
				saved.Error = ce.Error()
			} else {
				p.must(err)
			}
		}
		if mutation {
			p.must(write(cache, saved))
			p.h.state.Attempts[key] = "recorded"
			p.must(p.h.save())
		}
	}
	p.require(saved.Code == expected, fmt.Sprintf("%s: expected exit %d, got %d: %s", label, expected, saved.Code, saved.Error))
	if strings.Contains(label, "stale-etag") {
		p.require(strings.Contains(saved.Error, "HTTP 412"), "Expected a stale-write 412, not another HTTP error")
	}
	return saved.Response
}
func (p probe) call(label string, args []string, body any) any { return p.invoke(label, args, body, 0) }
func (p probe) check(id string, fn func()) {
	if _, done := p.h.state.Results[p.prefix()+id]; done {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			f, ok := r.(probeFailure)
			if !ok {
				panic(r)
			}
			p.must(p.h.record(p.prefix()+id, "FAIL", f.err.Error()))
		}
	}()
	fn()
	p.must(p.h.record(p.prefix()+id, "PASS", "Live API state and applicable semantic assertions matched"))
}
func (p probe) id(name string) string {
	return p.h.state.Variables["CORE"] + ":" + p.namespace() + name + ":1"
}
func (p probe) tags() map[string]any {
	return map[string]any{"evaluation": p.h.state.Run, "feature:logs": "", "feature:autoAbort": "1"}
}
func keyDescriptor(typ string) map[string]any {
	return map[string]any{"type": typ, "description": "Synthetic feature-test value.", "sensitive": false}
}
func inputDescriptor(typ, def string) map[string]any {
	d := keyDescriptor(typ)
	d["required"] = false
	d["defaultValue"] = def
	return d
}
func step(alias, execute string, input map[string]any) map[string]any {
	return map[string]any{"alias": alias, "execute": execute, "input": input, "description": "Bounded synthetic feature probe.", "progressMessage": nil, "initialDelay": nil, "pause": nil, "when": nil, "validate": nil, "autoRetry": nil, "repeat": nil, "errorMessages": []any{}}
}
func condition(expr, op, value string) map[string]any {
	return map[string]any{"semantic": "AND", "conditions": []any{map[string]any{"semantic": "AND", "cases": []any{map[string]any{"expression": expr, "operator": op, "semantic": "OR", "values": []string{value}}}}}}
}
func (p probe) command(name string, steps []any, output map[string]any) map[string]any {
	keys := map[string]any{}
	for k := range output {
		keys[k] = keyDescriptor("string")
	}
	return map[string]any{"catalog": p.h.state.Variables["CORE"], "name": p.namespace() + name, "version": 1, "description": "Runs isolated synthetic checks without changing external systems.", "inputKeys": map[string]any{}, "outputKeys": keys, "configuration": map[string]any{"values": []any{}, "executors": steps, "output": output, "listeners": []any{}}, "tags": p.tags()}
}
func (p probe) lint(label, kind string, body any) {
	file := filepath.Join(p.h.dir, "features", label+".json")
	p.must(write(file, body))
	p.call("lint-"+label, []string{"lint", "--kind", kind, "--mode", "api", file}, nil)
}
func (p probe) create(resource, label string, body any) string {
	kind := map[string]string{"commands": "command", "inputs": "input", "scheduled-executions": "scheduled-execution", "webhooks": "webhook", "catalogs": "catalog"}[resource]
	if kind != "" {
		p.lint(label, kind, body)
	}
	if resource == "commands" || resource == "inputs" {
		suffix := map[string]string{"commands": ".command.json", "inputs": ".input.json"}[resource]
		file := filepath.Join(p.h.dir, "features", str(object(body)["name"])+suffix)
		p.must(write(file, body))
		p.call("authoring-lint-"+label, []string{"lint", file}, nil)
	}
	v := p.call("create-"+label, []string{resource, "create", "-f", "-"}, body)
	id := str(object(v)["id"])
	p.require(id != "", "Create returned no ID: "+label)
	p.remember(resource, id)
	p.must(p.h.save())
	actual := p.call("get-"+label, []string{resource, "get", id}, nil)
	expected := object(body)
	if resource == "inputs" {
		expected = cloneObject(expected)
		delete(expected, "values")
	}
	p.require(subset(expected, actual), "Read-back differs: "+label)
	if resource == "commands" {
		issues, _ := object(actual)["issues"].([]any)
		p.require(len(issues) == 0, "Command validation issues: "+label)
	}
	return id
}
func cloneObject(v map[string]any) map[string]any {
	b, _ := json.Marshal(v)
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	return out
}
func (p probe) envelope(resource, id, label string) (map[string]any, string) {
	v := object(p.call(label, []string{resource, "get", id, "--response-envelope"}, nil))
	etag := str(object(v["headers"])["ETag"])
	p.require(etag != "", "Missing ETag: "+label)
	return object(v["body"]), etag
}
func (p probe) trigger(label, id string, values map[string]any, extraTags map[string]any, references ...string) string {
	tags := p.tags()
	tags["case"] = label
	for k, v := range extraTags {
		tags[k] = v
	}
	body := map[string]any{"commandId": id, "inputValues": values, "tags": tags}
	if len(references) > 0 {
		body["inputReferences"] = references
	}
	v := p.call("trigger-"+label, []string{"executions", "trigger", "-f", "-", "--wait-seconds", "1", "--timeout", "20s", "--response-envelope"}, body)
	e := object(object(v)["body"])
	eid := str(e["id"])
	p.require(eid != "", "Trigger returned no ID: "+label)
	p.h.state.Resources[p.executionKey(label)] = eid
	p.must(p.h.save())
	return eid
}
func (p probe) wait(label, id, status string) map[string]any {
	deadline := time.Now().Add(100 * time.Second)
	for time.Now().Before(deadline) {
		v := object(p.call("state-"+label, []string{"executions", "get", id}, nil))
		s := str(v["status"])
		if s == status {
			return v
		}
		if s == "FAILED" || s == "ABORTED" || s == "SUSPENDED" || s == "FINISHED" {
			p.require(false, fmt.Sprintf("%s: expected %s, observed %s", label, status, s))
		}
		time.Sleep(2 * time.Second)
	}
	p.require(false, "State deadline: "+label)
	return nil
}
func (p probe) output(label, id string) map[string]any {
	return object(object(p.call("output-"+label, []string{"executions", "output", id}, nil))["values"])
}
func (p probe) action(label, id, typ, value string) {
	_, etag := p.envelope("executions", id, "before-"+label)
	p.call("actions-before-"+label, []string{"executions", "actions", id}, nil)
	args := []string{"executions", "action", id, "--type", typ, "--reason", "Isolated skill feature test: " + label, "--if-match", etag}
	if value != "" {
		args = append(args, "--value", value)
	}
	p.call(label, args, nil)
	time.Sleep(2 * time.Second)
	history := p.call("actions-after-"+label, []string{"executions", "actions", id}, nil)
	data, _ := json.Marshal(history)
	p.require(strings.Contains(string(data), typ) && strings.Contains(string(data), label), "Action missing from history: "+typ)
}
func (h *harness) features(only string) error {
	if e := h.connect(); e != nil {
		return e
	}
	for _, r := range h.spec.Resources {
		if h.state.Resources[r.File] == "" {
			return fmt.Errorf("deploy the acceptance suite before feature checks: %s", r.File)
		}
	}
	p := probe{h}
	groups := []struct {
		name string
		fn   func(probe)
	}{{"content", featureContent}, {"expressions", featureExpressions}, {"actions", featureActions}, {"schedules", featureSchedules}, {"events", featureEvents}, {"scripts", featureScripts}, {"orchestration", featureOrchestration}}
	selected := map[string]bool{}
	if only != "" {
		for _, name := range strings.Split(only, ",") {
			selected[strings.TrimSpace(name)] = true
		}
	}
	for name := range selected {
		known := false
		for _, g := range groups {
			if name == g.name {
				known = true
			}
		}
		if !known {
			return fmt.Errorf("unknown feature group %q", name)
		}
	}
	for _, g := range groups {
		if only == "" || selected[g.name] {
			g.fn(p)
		}
	}
	for id, v := range h.state.Results {
		if strings.HasPrefix(id, p.prefix()) && v.Status == "FAIL" {
			return fmt.Errorf("feature failures recorded; inspect individual verdicts and preserve first attempts")
		}
	}
	return nil
}

func featureContent(p probe) {
	p.check("catalog-update", func() {
		id := p.h.state.Variables["CORE"]
		current := object(p.call("catalog-before", []string{"catalogs", "get", id}, nil))
		// Reused catalogs retain their metadata; this check requires this run's ownership tag.
		p.require(str(object(current["tags"])["evaluation"]) == p.h.state.Run, "Catalog metadata test requires a catalog created for this run")
		body := map[string]any{"name": current["name"], "description": "Temporary synthetic skill tests; cleanup follows evidence collection.", "documentationUrl": current["documentationUrl"], "tags": current["tags"]}
		p.lint("catalog-update", "catalog-update", body)
		p.call("catalog-update", []string{"catalogs", "update", id, "-f", "-"}, body)
		after := p.call("catalog-after", []string{"catalogs", "get", id}, nil)
		p.require(subset(body, after), "Catalog metadata not preserved")
	})
	p.check("command-update-etag-lifecycle", func() {
		d := p.command("LifecycleProbe", []any{step("echo", "utils-sapcp:Void:1", map[string]any{"message": "before"})}, map[string]any{"message": "$(.echo.output.message)"})
		id := p.create("commands", "lifecycle-command", d)
		_, etag := p.envelope("commands", id, "command-etag")
		d["description"] = "Returns a synthetic value after a conditional update."
		p.lint("command-update", "command", d)
		p.call("command-update", []string{"commands", "update", id, "-f", "-", "--if-match", etag}, d)
		p.invoke("command-stale-etag", []string{"commands", "update", id, "-f", "-", "--if-match", etag}, d, 4)
		after := p.call("command-updated", []string{"commands", "get", id}, nil)
		p.require(subset(d, after), "Conditional update changed unrelated fields")
		for _, verb := range []string{"release", "deprecate", "restore"} {
			_, tag := p.envelope("commands", id, "command-etag-"+verb)
			p.call("command-"+verb, []string{"commands", verb, id, "--if-match", tag}, nil)
			got := object(p.call("command-after-"+verb, []string{"commands", "get", id}, nil))
			tags := object(got["tags"])
			if verb == "release" {
				_, ok := tags["autopi:released"]
				p.require(ok, "Release tag absent")
			}
			if verb == "deprecate" {
				_, ok := tags["autopi:deprecated"]
				p.require(ok, "Deprecation tag absent")
			}
			if verb == "restore" {
				_, ok := tags["autopi:deprecated"]
				p.require(!ok, "Deprecation tag retained")
			}
		}
	})
	p.check("input-merge-replace-lifecycle", func() {
		d := map[string]any{"catalog": p.h.state.Variables["CORE"], "name": p.namespace() + "TypedFixture", "version": 1, "description": "Synthetic reusable typed values.", "keys": map[string]any{"text": keyDescriptor("string"), "count": keyDescriptor("number"), "flag": keyDescriptor("boolean"), "items": keyDescriptor("array"), "config": keyDescriptor("object")}, "values": map[string]any{"text": "synthetic", "count": "42", "flag": "false", "items": "[1,2]", "config": "{\"enabled\":false}"}, "tags": map[string]any{"evaluation": p.h.state.Run}}
		id := p.create("inputs", "typed-input", d)
		before, etag := p.envelope("inputs", id, "input-etag")
		patch := map[string]any{"description": "Updated synthetic fixture.", "keys": map[string]any{"extra": keyDescriptor("string")}, "values": map[string]any{"extra": "added"}}
		p.lint("input-patch", "input-patch", patch)
		p.call("input-patch", []string{"inputs", "patch", id, "-f", "-", "--if-match", etag}, patch)
		got, etag2 := p.envelope("inputs", id, "input-patched")
		p.require(subset(before["keys"], got["keys"]), "Patch lost sibling descriptors")
		p.require(object(got["keys"])["extra"] != nil, "Additive key absent")
		p.invoke("input-stale-etag", []string{"inputs", "patch", id, "-f", "-", "--if-match", etag}, patch, 4)
		remove := map[string]any{"keys": map[string]any{"extra": nil}}
		p.lint("input-key-removal", "input-patch", remove)
		p.call("input-key-removal", []string{"inputs", "patch", id, "-f", "-", "--if-match", etag2}, remove)
		got, etag2 = p.envelope("inputs", id, "input-removed-key")
		p.require(object(got["keys"])["extra"] == nil, "Removed key retained")
		p.call("input-full-replace", []string{"inputs", "update", id, "-f", "-", "--if-match", etag2}, d)
		for _, verb := range []string{"release", "deprecate", "restore"} {
			_, tag := p.envelope("inputs", id, "input-etag-"+verb)
			p.call("input-"+verb, []string{"inputs", verb, id, "--if-match", tag}, nil)
			got = object(p.call("input-after-"+verb, []string{"inputs", "get", id}, nil))
			_, deprecated := object(got["tags"])["autopi:deprecated"]
			if verb == "deprecate" {
				p.require(deprecated, "Input deprecation absent")
			}
			if verb == "restore" {
				p.require(!deprecated, "Input deprecation retained")
			}
		}
	})
	p.check("discovery-projections-filters", func() {
		catalog := p.h.state.Variables["CORE"]
		for _, res := range []string{"commands", "inputs"} {
			list := p.call(res+"-list-tag", []string{res, "list", "--catalog", catalog, "--tag", "evaluation=" + p.h.state.Run}, nil)
			arr, ok := list.([]any)
			p.require(ok && len(arr) > 0, "Empty scoped discovery")
			for _, v := range arr {
				m := object(v)
				p.require(m["catalog"] == catalog, "List escaped catalog filter")
				p.require(m["values"] == nil && m["configuration"] == nil, "Summary contains heavy or hidden fields")
			}
		}
		id := p.h.state.Resources["FetchPost.command.json"]
		v := object(p.call("command-contract", []string{"commands", "contract", id}, nil))
		p.require(v["inputKeys"] != nil && v["configuration"] == nil && v["executors"] == nil, "Contract projection wrong")
		ids := p.call("command-ids", []string{"commands", "ids", "--catalog", catalog}, nil)
		b, _ := json.Marshal(ids)
		p.require(strings.Contains(string(b), id), "Command IDs omit known command")
	})
	p.check("input-typed-runtime", func() {
		inputID := p.id("TypedFixture")
		metadata := object(p.call("typed-fixture-metadata", []string{"inputs", "get", inputID}, nil))
		outputs := map[string]any{}
		keys := map[string]any{}
		for name, desc := range object(metadata["keys"]) {
			d := cloneObject(object(desc))
			d["required"] = true
			keys[name] = d
			outputs[name] = "$(.execution.input." + name + ")"
		}
		d := p.command("TypedConsumer", []any{step("bound", "utils-sapcp:Void:1", map[string]any{"message": "bound"})}, outputs)
		d["inputKeys"] = keys
		id := p.create("commands", "typed-consumer", d)
		eid := p.trigger("typed-values", id, map[string]any{}, nil, inputID)
		p.wait("typed-values", eid, "FINISHED")
		out := p.output("typed-values", eid)
		p.require(out["text"] == "synthetic" && out["count"] == "42" && out["flag"] == "false", "Scalar typed values changed")
		p.require(valueAt(out, "items.1") == float64(2) && valueAt(out, "config.enabled") == false, "Structured typed values changed")
	})
	p.check("bulk-and-upsert", func() {
		d := p.command("ImportProbe", []any{step("echo", "utils-sapcp:Void:1", map[string]any{"message": "import"})}, map[string]any{"message": "$(.echo.output.message)"})
		env := map[string]any{"commands": []any{d}, "skipExisting": true}
		p.lint("bulk-command", "bulk-commands", env)
		p.call("bulk-command", []string{"commands", "bulk-create", "-f", "-"}, env)
		id := p.id("ImportProbe")
		p.remember("commands", id)
		p.must(p.h.save())
		p.require(subset(d, p.call("bulk-command-read", []string{"commands", "get", id}, nil)), "Imported command differs")
		p.call("bulk-command-skip", []string{"commands", "bulk-create", "-f", "-"}, env)
		file := filepath.Join(p.h.dir, "features", "exported.command.json")
		p.call("command-export", []string{"commands", "export", id, "-o", file}, nil)
		var exported map[string]any
		p.must(read(file, &exported))
		p.require(subset(d, exported), "Export differs")
		p.call("command-deploy-existing", []string{"commands", "deploy", "-f", "-"}, d)
		fresh := cloneObject(d)
		fresh["name"] = p.namespace() + "UpsertProbe"
		p.call("command-deploy-new", []string{"commands", "deploy", "-f", "-"}, fresh)
		p.remember("commands", p.id("UpsertProbe"))
		p.must(p.h.save())
		in := map[string]any{"catalog": p.h.state.Variables["CORE"], "name": p.namespace() + "ImportFixture", "version": 1, "description": "Synthetic import fixture.", "keys": map[string]any{"message": keyDescriptor("string")}, "values": map[string]any{"message": "imported"}, "tags": map[string]any{"evaluation": p.h.state.Run}}
		inputs := map[string]any{"inputs": []any{in}, "skipExisting": true}
		p.lint("bulk-input", "bulk-inputs", inputs)
		p.call("bulk-input", []string{"inputs", "bulk-create", "-f", "-"}, inputs)
		iid := p.id("ImportFixture")
		p.remember("inputs", iid)
		p.must(p.h.save())
		p.call("bulk-input-read", []string{"inputs", "get", iid}, nil)
		p.call("input-deploy-existing", []string{"inputs", "deploy", "-f", "-"}, in)
		in["name"] = p.namespace() + "UpsertFixture"
		p.call("input-deploy-new", []string{"inputs", "deploy", "-f", "-"}, in)
		p.remember("inputs", p.id("UpsertFixture"))
		p.must(p.h.save())
	})
}

func featureExpressions(p probe) {
	tests := []struct{ name, expr, want string }{
		{"array-count", "\"[1,2]\" | toArray | length", "2"}, {"empty-array", "\"[]\" | toArray | length", "0"},
		{"serialized-object", "\"{\\\"x\\\":false}\" | toObject.x", "false"}, {"wrong-conversion", "(\"[1]\" | toObject) // \"fallback\"", "fallback"},
		{"stream-fallback", "(null, 1, null, 2) // \"fb\"", "12"}, {"all-null-fallback", "(null, false) // \"fb\"", "fb"}, {"boolean-preserve", "if false == null then true else false end", "false"},
		{"division", "7 / 2", "3.5"}, {"rounding", "(7 / 2) | floor", "3"}, {"range", "[range(3)] | toMinifiedJsonString", "[0,1,2]"},
		{"filter-map", "[{\"id\":\"a\",\"s\":0},{\"id\":\"b\",\"s\":1}] | filter(.s == 0) | map(.id) | join(\",\")", "a"},
		{"reduce", "reduce [1,2,3][] as $n (0; . + $n)", "6"}, {"max-by", "[{\"n\":2},{\"n\":5}] | maxBy(.n) | .n", "5"},
		{"group-sort", "[3,1,1,2] | unique | reverse | toMinifiedJsonString", "[3,2,1]"},
		{"string-normalize", "\"  AbC  \" | strip | toLowerCase", "abc"}, {"regex", "\"a//b///c\" | gsub(\"/+\"; \"/\")", "a/b/c"},
		{"url-encoding", "\"a b&c\" | toUrlEncoded", "a+b%26c"}, {"percent-encoding", "\"a b&c\" | toPercentEncoded", "a%20b%26c"},
		{"base64-roundtrip", "\"synthetic\" | toBase64 | fromBase64", "synthetic"},
		{"hash", "\"abc\" | toSha256", "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"},
		{"case-insensitive", "{\"Content-Type\":\"application/json\"} | getCaseInsensitive(\"content-type\")", "application/json"},
		{"version-compare", "[\"1.2.0\",\"1.10.0\"] | compareVersions", "-1"}, {"date-roundtrip", "0 | toDate(\"yyyy-MM-dd\")", "1970-01-01"},
		{"guid", "guid | isGuid", "true"}, {"clock-units", "(nowMillis / 1000 - now) as $delta | $delta >= -2 and $delta < 2", "true"},
	}
	// One independent command per expression keeps one unsupported element from
	// preventing every other case from running and preserves first-attempt evidence.
	for i, tc := range tests {
		tc := tc
		p.check("expression-"+tc.name, func() {
			name := fmt.Sprintf("Expression%02d", i)
			d := p.command(name, []any{step("evaluate", "utils-sapcp:Void:1", map[string]any{"message": "$(" + tc.expr + ")"})}, map[string]any{"message": "$(.evaluate.output.message)"})
			id := p.create("commands", "expr-"+tc.name, d)
			eid := p.trigger("expr-"+tc.name, id, map[string]any{}, nil)
			p.wait("expr-"+tc.name, eid, "FINISHED")
			got := p.output("expr-"+tc.name, eid)
			p.require(got["message"] == tc.want, fmt.Sprintf("Expected %q, observed %q", tc.want, got["message"]))
		})
	}
}

func featureActions(p probe) {
	p.check("tag-inheritance-and-overrides", func() {
		d := p.command("TagProbe", []any{step("echo", "utils-sapcp:Void:1", map[string]any{"message": "tagged"})}, map[string]any{"message": "$(.echo.output.message)"})
		tags := object(d["tags"])
		tags["origin"] = "command"
		tags["feature:priority"] = "low"
		tags["feature:retentionPolicy"] = "archive"
		tags["feature:retentionPeriod"] = "1"
		id := p.create("commands", "tag-command", d)
		eid := p.trigger("tag-override", id, map[string]any{}, map[string]any{"origin": "trigger", "feature:priority": "high"})
		state := p.wait("tag-override", eid, "FINISHED")
		actual := object(state["tags"])
		p.require(actual["origin"] == "trigger" && actual["feature:priority"] == "high" && actual["feature:retentionPolicy"] == "archive" && actual["feature:retentionPeriod"] == "1", "Tag inheritance or override differs")
		_, logs := actual["feature:logs"]
		p.require(logs, "Inherited logging tag absent")
		p.require(actual["autopi:userAgent"] != nil, "CLI origin tag absent")
	})
	p.check("pause-resume-comment", func() {
		s := step("delay", "utils-sapcp:Void:1", map[string]any{"message": "done"})
		s["initialDelay"] = map[string]any{"interval": "30s", "when": nil}
		id := p.create("commands", "delay-command", p.command("DelayProbe", []any{s}, map[string]any{"message": "$(.delay.output.message)"}))
		eid := p.trigger("pause-resume", id, map[string]any{}, nil)
		p.wait("pause-running", eid, "RUNNING")
		p.action("pause", eid, "PAUSE", "")
		p.wait("paused", eid, "PAUSED")
		p.action("comment", eid, "COMMENT", "")
		p.action("resume", eid, "RESUME", "")
		p.wait("resumed", eid, "FINISHED")
		p.require(p.output("resumed", eid)["message"] == "done", "Resume output differs")
	})
	p.check("abort-bulk-partial", func() {
		id := p.id("DelayProbe")
		a := p.trigger("bulk-a", id, map[string]any{}, nil)
		b := p.trigger("bulk-b", id, map[string]any{}, nil)
		missing := p.h.state.Tenant + "-missing-" + p.h.state.Run
		args := []string{"executions", "bulk-action", "--execution-id", a, "--execution-id", b, "--execution-id", missing, "--type", "PAUSE", "--reason", "Isolated partial-outcome test"}
		result := object(p.invoke("bulk-pause-partial", args, nil, 7))
		arr, _ := result["results"].([]any)
		p.require(len(arr) == 3, "Bulk result count differs")
		for i, id := range []string{a, b, missing} {
			m := object(arr[i])
			p.require(m["executionId"] == id && m["succeeded"] == (i < 2), "Ordered bulk result does not match target and outcome")
		}
		p.wait("bulk-a-paused", a, "PAUSED")
		p.wait("bulk-b-paused", b, "PAUSED")
		result = object(p.call("bulk-abort-paused", []string{"executions", "bulk-action", "--execution-id", a, "--execution-id", a, "--execution-id", b, "--type", "ABORT", "--reason", "Isolated test cleanup"}, nil))
		arr, _ = result["results"].([]any)
		p.require(len(arr) == 3, "Duplicate ID result count differs")
		for i, id := range []string{a, a, b} {
			m := object(arr[i])
			p.require(m["executionId"] == id && m["succeeded"] == true, "Duplicate/ordered bulk outcome differs")
		}
		p.wait("bulk-a", a, "ABORTED")
		p.wait("bulk-b", b, "ABORTED")
	})
	p.check("fail-retry-reset-skip", func() {
		guard := step("guard", "utils-sapcp:Void:1", map[string]any{"message": "expected"})
		guard["validate"] = condition("false", "EQUALS", "true")
		d := p.command("FailureProbe", []any{guard, step("after", "utils-sapcp:Void:1", map[string]any{"message": "after"})}, map[string]any{"message": "$(.after.output.message)"})
		id := p.create("commands", "failure-command", d)
		eid := p.trigger("retry", id, map[string]any{}, nil)
		state := p.wait("expected-failure", eid, "FAILED")
		p.require(state["error"] != nil, "Expected validation error absent")
		p.action("retry", eid, "RETRY", "")
		p.wait("retried-failure", eid, "FAILED")
		path := str(state["currentExecutorPath"])
		p.require(path != "", "Failed executor path absent")
		p.action("reset", eid, "RESET", path)
		p.wait("reset-failure", eid, "FAILED")
		payload, _ := json.Marshal(map[string]any{"executorPath": path, "outputValues": map[string]string{"message": "skipped"}})
		p.action("skip", eid, "SKIP", string(payload))
		p.wait("skipped", eid, "FINISHED")
		p.require(p.output("skipped", eid)["message"] == "after", "Skip did not continue")
		delayed := p.trigger("forced-fail", p.id("DelayProbe"), map[string]any{}, nil)
		p.action("pause-before-fail", delayed, "PAUSE", "")
		p.wait("paused-before-fail", delayed, "PAUSED")
		p.invoke("unsupported-fail", []string{"executions", "action", delayed, "--type", "FAIL", "--reason", "Verify unsupported operator action"}, nil, 1)
		p.wait("unchanged-after-fail-rejection", delayed, "PAUSED")
		p.action("abort-forced-fail", delayed, "ABORT", "")
		p.wait("aborted-forced-fail", delayed, "ABORTED")
	})
	for _, free := range []bool{false, true} {
		free := free
		kind := "choice"
		if free {
			kind = "free-text"
		}
		p.check("human-"+kind, func() {
			built := "utils-sapcp:GetUserChoice:1"
			in := map[string]any{"title": "Synthetic test choice; no external effects.", "options": "{\"alpha\":\"Alpha\",\"beta\":\"Beta\"}"}
			name := "ChoiceProbe"
			if free {
				built = "utils-sapcp:GetUserInput:1"
				delete(in, "options")
				name = "FreeTextProbe"
			}
			p.call("contract-"+kind, []string{"commands", "contract", built}, nil)
			d := p.command(name, []any{step("answer", built, in)}, map[string]any{"message": "$(.answer.output.selection)"})
			id := p.create("commands", kind+"-command", d)
			options := []string{"alpha", "beta"}
			if free {
				options = []string{"synthetic answer"}
			}
			for i, value := range options {
				label := fmt.Sprintf("%s-%d", kind, i)
				eid := p.trigger(label, id, map[string]any{}, nil)
				state := p.wait(label, eid, "INPUT_REQUIRED")
				p.require(state["userChoice"] != nil, "User choice metadata absent")
				p.action("choose-"+label, eid, "CHOOSE", value)
				p.wait(label, eid, "FINISHED")
				p.require(p.output(label, eid)["message"] == value, "Choice returned label instead of offered key or text")
			}
		})
	}
	p.check("history-input-output-logs", func() {
		id := p.id("DelayProbe")
		eid := p.h.state.Resources[p.executionKey("pause-resume")]
		p.require(eid != "", "Missing completed execution")
		p.call("resolved-input", []string{"executions", "input", eid}, nil)
		list := p.call("history-filter", []string{"executions", "list", "--command-id", id, "--execution-id", eid, "--status", "FINISHED", "--tag", "evaluation=" + p.h.state.Run, "--started-after", "24h", "--limit", "100"}, nil)
		arr, _ := list.([]any)
		p.require(len(arr) == 1 && str(object(arr[0])["id"]) == eid, "History filters returned wrong execution")
		p.call("history-summary", []string{"executions", "summary", "--started-after", "1h"}, nil)
		page := p.call("logs-page", []string{"executions", "logs", eid, "--page", "0", "--max-page-size", "20"}, nil)
		p.require(page != nil, "Log page absent")
		p.call("logs-executor", []string{"executions", "logs", eid, "--executor", "delay"}, nil)
		p.call("logs-empty-page", []string{"executions", "logs", eid, "--max-page-size", "0"}, nil)
	})
}

func featureSchedules(p probe) {
	recurrences := []struct {
		name  string
		value any
	}{
		{"once", map[string]any{"month": 12, "day": 31, "hour": 12, "minute": 0}},
		{"hourly", map[string]any{"minutes": []int{0, 30}}},
		{"daily", map[string]any{"hours": []int{9, 17}, "minutes": []int{0, 30}}},
		{"weekly", map[string]any{"days": []string{"MONDAY", "FRIDAY"}, "hours": []int{9}, "minutes": []int{0}}},
		{"monthly", map[string]any{"days": []int{1, 15, 31}, "hours": []int{9}, "minutes": []int{0}}},
		{"yearly", map[string]any{"months": []int{1, 4, 7, 10}, "days": []int{1}, "hours": []int{9}, "minutes": []int{0}}},
	}
	for _, r := range recurrences {
		r := r
		p.check("schedule-"+r.name+"-lifecycle", func() {
			body := map[string]any{"commandId": p.h.state.Resources["FetchPost.command.json"], "description": "Disabled synthetic schedule for binding and lifecycle verification.", "enabled": false, "inputReferences": []string{p.h.state.Resources["ApiConfiguration.input.json"], p.h.state.Resources["ScheduledArguments.input.json"]}, "inputValues": map[string]any{}, "tags": p.tags(), "schedule": map[string]any{"timeZone": "Europe/Sofia", r.name: r.value}}
			id := p.create("scheduled-executions", "schedule-"+r.name, body)
			_, etag := p.envelope("scheduled-executions", id, "schedule-etag-"+r.name)
			body["description"] = "Updated disabled synthetic schedule."
			p.call("schedule-update-"+r.name, []string{"scheduled-executions", "update", id, "-f", "-", "--if-match", etag}, body)
			after := p.call("schedule-updated-"+r.name, []string{"scheduled-executions", "get", id}, nil)
			p.require(subset(body, after), "Schedule update lost fields")
			// Only enable the future once schedule; recurring definitions remain disabled.
			if r.name == "once" {
				p.call("schedule-enable", []string{"scheduled-executions", "enable", id}, nil)
				p.require(object(p.call("schedule-enabled", []string{"scheduled-executions", "get", id}, nil))["enabled"] == true, "Enable did not persist")
				p.call("schedule-disable", []string{"scheduled-executions", "disable", id}, nil)
				p.require(object(p.call("schedule-disabled", []string{"scheduled-executions", "get", id}, nil))["enabled"] == false, "Disable did not persist")
			}
		})
	}
}

func featureEvents(p probe) {
	p.check("event-command", func() {
		d := p.command("EventProbe", []any{step("echo", "utils-sapcp:Void:1", map[string]any{"message": "$(.execution.input.event)"})}, map[string]any{"message": "$(.echo.output.message)"})
		d["inputKeys"] = map[string]any{"event": inputDescriptor("string", "")}
		p.create("commands", "event-command", d)
	})
	p.check("generic-event", func() {
		result := p.call("generic-event", []string{"triggers", "generic-event", "--command-reference", p.id("EventProbe"), "--map-event-to-input-key", "event", "--data", "synthetic event", "--content-type", "text/plain", "--tag", "evaluation=" + p.h.state.Run, "--correlation-id", p.h.state.Run + "-generic"}, nil)
		eid := str(object(result)["id"])
		p.require(eid != "", "Event returned no execution ID")
		p.h.state.Resources[p.executionKey("generic")] = eid
		p.must(p.h.save())
		p.wait("generic", eid, "FINISHED")
		p.require(p.output("generic", eid)["message"] == "synthetic event", "Generic event mapping differs")
	})
	p.check("ans-event", func() {
		event := map[string]any{"eventType": "SyntheticSkillTest", "resource": map[string]any{"resourceName": "test-resource", "resourceType": "test"}, "severity": "INFO", "category": "NOTIFICATION", "subject": "Synthetic skill test", "body": "No external notifications are sent by this test."}
		v := p.call("ans-event", []string{"triggers", "ans-event", "--command-reference", p.id("EventProbe"), "--map-event-to-input-key", "event", "--disable-resource-properties-mapping", "--disable-region-properties-mapping", "--tag", "evaluation=" + p.h.state.Run, "-f", "-"}, event)
		eid := str(object(v)["id"])
		p.require(eid != "", "ANS event returned no execution ID")
		p.h.state.Resources[p.executionKey("ans")] = eid
		p.must(p.h.save())
		p.wait("ans", eid, "FINISHED")
		var mapped map[string]any
		p.must(json.Unmarshal([]byte(str(p.output("ans", eid)["message"])), &mapped))
		p.require(subset(event, mapped), "ANS event mapping differs")
	})
	p.check("webhook-lifecycle", func() {
		body := map[string]any{"name": "SkillTest" + p.h.featureRound + p.h.state.Run, "type": "GENERIC", "commandId": p.id("EventProbe"), "description": "Synthetic webhook with no external effects.", "eventKey": "event", "inputReferences": []any{}, "inputValues": map[string]any{}, "enabled": true, "tags": p.tags()}
		file := filepath.Join(p.h.dir, "features", "probe.webhook.json")
		p.must(write(file, body))
		p.call("webhook-binding", []string{"webhooks", "validate", "-f", file}, nil)
		id := p.create("webhooks", "webhook", body)
		v := p.call("webhook-trigger", []string{"webhooks", "trigger", id, "--data", "webhook synthetic", "--content-type", "text/plain"}, nil)
		eid := str(object(v)["id"])
		p.require(eid != "", "Webhook returned no execution ID")
		p.h.state.Resources[p.executionKey("webhook")] = eid
		p.must(p.h.save())
		p.wait("webhook", eid, "FINISHED")
		p.require(p.output("webhook", eid)["message"] == "webhook synthetic", "Webhook payload differs")
		_, etag := p.envelope("webhooks", id, "webhook-etag")
		body["enabled"] = false
		p.call("webhook-disable", []string{"webhooks", "update", id, "-f", "-", "--if-match", etag}, body)
		p.require(subset(body, p.call("webhook-disabled", []string{"webhooks", "get", id}, nil)), "Webhook disable changed fields")
		p.invoke("webhook-disabled-trigger", []string{"webhooks", "trigger", id}, nil, 4)
	})
}

func featureScripts(p probe) {
	runtimes := []struct{ name, command, script, key string }{{"bash", "ExecuteScript:2", "printf 'synthetic-runtime-ok\\n'", "output"}, {"python", "ExecutePythonScript:1", "print('synthetic-runtime-ok')", "result"}, {"node", "ExecuteNodeJsScript:1", "console.log('synthetic-runtime-ok')", "result"}, {"powershell", "ExecutePowerShellScript:1", "Write-Output 'synthetic-runtime-ok'", "result"}}
	for i, r := range runtimes {
		r := r
		p.check("script-"+r.name, func() {
			built := "scripts-sapcp:" + r.command
			contract := object(p.call("contract-"+r.name, []string{"commands", "contract", built}, nil))
			p.require(object(contract["outputKeys"])[r.key] != nil, "Script output contract differs")
			// A script is required here specifically to exercise the selected runtime.
			d := p.command(fmt.Sprintf("Runtime%d", i), []any{step("run", built, map[string]any{"script": r.script, "timeout": "30"})}, map[string]any{"message": "$(.run.output." + r.key + " | toArray | join(\"\"))", "code": "$(.run.output.exitCode)"})
			id := p.create("commands", "runtime-"+r.name, d)
			eid := p.trigger("runtime-"+r.name, id, map[string]any{}, nil)
			p.wait("runtime-"+r.name, eid, "FINISHED")
			out := p.output("runtime-"+r.name, eid)
			p.require(out["code"] == "0" && out["message"] == "synthetic-runtime-ok", "Runtime output differs")
		})
	}
}

func featureOrchestration(p probe) {
	p.check("explicit-child-success", func() {
		for _, id := range []string{"utils-sapcp:TriggerExecution:1", "utils-sapcp:AwaitExecution:1", "utils-sapcp:GetExecutionOutput:1", "utils-sapcp:GetExecutionStatus:1", "utils-sapcp:GetExecutionInput:1"} {
			p.call("contract-"+strings.Split(id, ":")[1], []string{"commands", "contract", id}, nil)
		}
		child := p.create("commands", "child-command", p.command("ChildProbe", []any{step("echo", "utils-sapcp:Void:1", map[string]any{"message": "child result"})}, map[string]any{"message": "$(.echo.output.message)"}))
		trigger := step("trigger", "utils-sapcp:TriggerExecution:1", map[string]any{"command": child, "values": "{}", "tags": "{\"evaluation\":\"" + p.h.state.Run + "\"}"})
		await := step("await", "utils-sapcp:AwaitExecution:1", map[string]any{"execution": "$(.trigger.output.execution)", "statuses": "[\"FINISHED\",\"FAILED\",\"ABORTED\",\"SUSPENDED\"]", "deadline": "1"})
		await["validate"] = condition("$(.await.output.status)", "EQUALS", "FINISHED")
		d := p.command("ParentProbe", []any{trigger, await, step("read", "utils-sapcp:GetExecutionOutput:1", map[string]any{"execution": "$(.trigger.output.execution)"})}, map[string]any{"child": "$(.trigger.output.execution)", "message": "$(.read.output.output | toObject.message)"})
		id := p.create("commands", "parent-command", d)
		eid := p.trigger("parent", id, map[string]any{}, nil)
		p.wait("parent", eid, "FINISHED")
		out := p.output("parent", eid)
		p.require(out["message"] == "child result", "Nested child output conversion differs")
		cid := str(out["child"])
		p.require(cid != "", "Child ID absent")
		p.h.state.Resources[p.executionKey("child")] = cid
		p.must(p.h.save())
		state := p.wait("child", cid, "FINISHED")
		p.require(object(state["tags"])["autopi:childExecutionOf"] == eid, "Parent lineage differs")
	})
}
