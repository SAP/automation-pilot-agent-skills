package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// refreshFixtureReferences records simulated reference actions against the real
// fixture handler. These are grader challenges, never fresh-agent run evidence.
func refreshFixtureReferences(b bank, out string) error {
	if out == "" {
		return fmt.Errorf("refresh-fixtures requires --out; review the new bank before replacing it")
	}
	for i := range b.Cases {
		c := &b.Cases[i]
		if c.Fixture == nil {
			continue
		}
		f := startFixture(*c.Fixture, "reference-synthetic-token")
		planned := append([]requestRecord{}, c.Reference.Requests...)
		if c.ID == "block-mcp-incompatible-binding" {
			for _, path := range sortedKeys(c.Fixture.Resources) {
				planned = append(planned, requestRecord{Method: "GET", Path: path})
			}
		}
		if c.ID == "catalog-delete-owned-empty" {
			planned = append(planned[:1], append([]requestRecord{
				{Method: "GET", Path: "commands", Query: map[string][]string{"catalog": {"eval-fixture"}}},
				{Method: "GET", Path: "inputs", Query: map[string][]string{"catalog": {"eval-fixture"}}},
			}, planned[1:]...)...)
		}
		for _, r := range planned {
			u := f.server.URL + "/api/v1/" + r.Path
			if len(r.Query) > 0 {
				req, _ := http.NewRequest(r.Method, u, nil)
				q := req.URL.Query()
				for k, vs := range r.Query {
					for _, v := range vs {
						q.Add(k, v)
					}
				}
				u += "?" + q.Encode()
			}
			var body io.Reader
			if r.Body != nil {
				body = bytes.NewReader(mustJSON(r.Body))
			}
			req, err := http.NewRequest(r.Method, u, body)
			if err != nil {
				f.close()
				return err
			}
			req.Header.Set("Authorization", "Bearer reference-synthetic-token")
			for k, v := range r.Headers {
				req.Header.Set(k, v)
			}
			res, err := f.server.Client().Do(req)
			if err != nil {
				f.close()
				return err
			}
			_, err = io.Copy(io.Discard, res.Body)
			res.Body.Close()
			if err != nil {
				f.close()
				return err
			}
			if r.Status > 0 && res.StatusCode != r.Status {
				f.close()
				return fmt.Errorf("%s reference %s %s: HTTP %d instead of %d", c.ID, r.Method, r.Path, res.StatusCode, r.Status)
			}
		}
		requests, state := f.snapshot()
		f.close()
		// Remove random endpoint identities from the versioned reference.
		raw := strings.ReplaceAll(string(mustJSON(requests)), f.server.URL, "https://fixture.example.test")
		if err := json.Unmarshal([]byte(raw), &requests); err != nil {
			return err
		}
		c.Reference.Requests = requests
		c.Reference.State = state
		c.Version++
	}
	return writeJSON(out, b)
}
