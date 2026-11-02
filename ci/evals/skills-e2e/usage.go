package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Reads only usage-counter fields. Conversation text, tool arguments, credentials,
// and the log path are never copied into the measurement artifact.
func (h *harness) measureModelUsage(path, since string) error {
	start, err := time.Parse(time.RFC3339Nano, since)
	if err != nil {
		return fmt.Errorf("--since must be RFC3339: %w", err)
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	decoder := json.NewDecoder(f)
	var before, after map[string]int64
	var from, to string
	for {
		var entry struct {
			Timestamp string
			Type      string
			Payload   struct {
				Type string
				Info struct {
					Total map[string]int64 `json:"total_token_usage"`
				}
			}
		}
		err = decoder.Decode(&entry)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return err
		}
		if entry.Type != "event_msg" || entry.Payload.Type != "token_count" || entry.Payload.Info.Total == nil {
			continue
		}
		stamp, e := time.Parse(time.RFC3339Nano, entry.Timestamp)
		if e != nil {
			continue
		}
		if stamp.Before(start) {
			before = entry.Payload.Info.Total
			from = entry.Timestamp
		} else {
			after = entry.Payload.Info.Total
			to = entry.Timestamp
		}
	}
	if before == nil || after == nil {
		return fmt.Errorf("no usage counters bracket the requested interval")
	}
	delta := map[string]int64{}
	for k, v := range after {
		delta[k] = v - before[k]
		if delta[k] < 0 {
			return fmt.Errorf("usage counter %s reset during interval", k)
		}
	}
	delta["uncached_input_tokens"] = delta["input_tokens"] - delta["cached_input_tokens"]
	result := map[string]any{"requestedStart": since, "baselineCounterAt": from, "latestCounterAt": to, "usageDelta": delta, "scope": "Actual host-reported model counters for the supplied session interval. Includes all model work between the recorded counter timestamps, plus cache effects. Not a controlled no-skill baseline or per-skill causal comparison. Work after the latest sampled counter is excluded."}
	if err = write(filepath.Join(h.dir, "model-usage.json"), result); err != nil {
		return err
	}
	fmt.Printf("MODEL USAGE: %d input (%d cached, %d uncached), %d output tokens through %s\n", delta["input_tokens"], delta["cached_input_tokens"], delta["uncached_input_tokens"], delta["output_tokens"], to)
	return nil
}
