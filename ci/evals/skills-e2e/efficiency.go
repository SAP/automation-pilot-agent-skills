package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// This comparison measures the agent-visible representation, not inference
// billing. bytes/4 is explicitly an estimate; model counters are separate.
func (h *harness) measure() (err error) {
	if err = h.connect(); err != nil {
		return err
	}
	p := probe{h}
	defer func() {
		if r := recover(); r != nil {
			if f, ok := r.(probeFailure); ok {
				err = f.err
			} else {
				panic(r)
			}
		}
	}()
	rows := []map[string]any{}
	compare := func(label string, small, full []string) {
		a := p.call("measure-"+label+"-focused", small, nil)
		b := p.call("measure-"+label+"-full", full, nil)
		ab, _ := json.MarshalIndent(a, "", "  ")
		bb, _ := json.MarshalIndent(b, "", "  ")
		rows = append(rows, map[string]any{"comparison": label, "focusedBytes": len(ab), "fullBytes": len(bb), "focusedEstimatedTokens": (len(ab) + 3) / 4, "fullEstimatedTokens": (len(bb) + 3) / 4, "reductionPercent": 100 * (1 - float64(len(ab))/float64(len(bb)))})
	}
	for _, file := range []string{"FetchPost.command.json", "PortfolioReport.command.json", "HttpControl.command.json"} {
		id := h.state.Resources[file]
		p.require(id != "", "Deploy suite before measuring")
		compare(file, []string{"commands", "contract", id}, []string{"commands", "get", id})
	}
	for _, res := range []string{"commands", "inputs"} {
		compare(res+"-list", []string{res, "list", "--catalog", h.state.Variables["CORE"]}, []string{res, "list", "--catalog", h.state.Variables["CORE"], "--full"})
	}
	result := map[string]any{"method": "UTF-8 JSON bytes; token estimates are ceil(bytes/4), not tokenizer or billing counts. Both representations use the same resource and formatting.", "comparisons": rows, "baselineLimit": "One live candidate; this is not a no-skill model ablation. Actual model counters must use the session usage delta for the task interval."}
	files, e := os.ReadDir(filepath.Join(h.dir, "evidence"))
	p.must(e)
	var measured, stdoutBytes, elapsed int64
	for _, f := range files {
		var record struct {
			StdoutBytes         *int64
			ElapsedMilliseconds int64
		}
		p.must(read(filepath.Join(h.dir, "evidence", f.Name()), &record))
		if record.StdoutBytes != nil {
			measured++
			stdoutBytes += *record.StdoutBytes
			elapsed += record.ElapsedMilliseconds
		}
	}
	result["harnessCalls"] = map[string]any{"total": len(files), "instrumented": measured, "instrumentedStdoutBytes": stdoutBytes, "instrumentedCLIMilliseconds": elapsed, "note": "CLI responses are consumed by Go; their size is not the number of tokens sent to the model. Earlier calls without size instrumentation are excluded from byte/time totals."}
	p.must(write(filepath.Join(h.dir, "efficiency.json"), result))
	for _, row := range rows {
		fmt.Printf("MEASURE %s: %d -> %d bytes (%.1f%% smaller)\n", row["comparison"], row["fullBytes"], row["focusedBytes"], row["reductionPercent"])
	}
	return nil
}
