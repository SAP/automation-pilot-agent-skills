package client

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestMCPProtocolPreservesSchemasOutputsAndSSE(t *testing.T) {
	for _, body := range []string{
		"{\"jsonrpc\":\"2.0\",\"result\":{\"tools\":[{\"inputSchema\":{\"properties\":{\"password\":{\"type\":\"string\"}}},\"outputSchema\":{\"properties\":{\"outputValues\":{\"type\":\"object\"}}}}]}}",
		"{\"jsonrpc\":\"2.0\",\"result\":{\"structuredContent\":{\"id\":\"exec-1\",\"outputValues\":{\"answer\":\"42\"}}}}",
		"event: message\ndata: {\"jsonrpc\":\"2.0\",\"result\":{\"id\":\"exec-1\"}}\n\n",
	} {
		for _, path := range []string{"mcp/server", "/api/v1/mcp/server"} {
			cfg := Config{Host: "test.invalid", Token: "token", Transport: outputTransport(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
			})}
			for _, method := range []string{"initialize", "tools/list", "tools/call"} {
				request := []byte("{\"jsonrpc\":\"2.0\",\"method\":\"" + method + "\",\"params\":{}}")
				res, err := New(cfg, io.Discard).Do(Request{Method: "POST", Path: path, Body: request})
				if err != nil || !bytes.Equal(res.Body, []byte(body)) {
					t.Fatalf("MCP protocol corrupted: %v %s", err, res.Body)
				}
			}
		}
	}
}

func TestRPCShapeCannotBypassRESTMutationProtection(t *testing.T) {
	for _, path := range []string{"inputs", "commands", "mcp-servers", "mcp/server/extra"} {
		cfg := Config{Host: "test.invalid", Token: "token", Transport: outputTransport(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("{\"password\":\"SYNTHETIC-SECRET\"}"))}, nil
		})}
		res, err := New(cfg, io.Discard).Do(Request{Method: "POST", Path: path, Body: []byte("{\"jsonrpc\":\"2.0\",\"method\":\"tools/list\"}")})
		if err != nil || bytes.Contains(res.Body, []byte("SYNTHETIC-SECRET")) {
			t.Fatalf("REST bypass: %v", err)
		}
	}
}
