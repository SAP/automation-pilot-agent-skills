package cmd

import (
	"io"
	"net/http"
	"testing"
)

func TestLiveQuotedUUIDETagIsPreservedForConditionalWrites(t *testing.T) {
	const etag = "\"7aa8cf0c-1671-4ef2-a6af-303096a1c051\""
	for _, args := range [][]string{
		{"commands", "update", "c:Hello:1", "--data", validCommand, "--if-match", etag},
		{"inputs", "patch", "c:Fixture:1", "--data", "{\"description\":\"updated\"}", "--if-match", etag},
		{"commands", "release", "c:Hello:1", "--if-match", etag},
		{"inputs", "delete", "c:Fixture:1", "--if-match", etag},
	} {
		t.Run(args[0]+"-"+args[1], func(t *testing.T) {
			calls := 0
			code, _, stderr := memoryArgs(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Header.Get("If-Match") != etag {
					t.Errorf("ETag changed: %q", r.Header.Get("If-Match"))
				}
				w.WriteHeader(http.StatusNoContent)
			}, "", args...)
			if code != exitOK || calls != 1 {
				t.Fatalf("code=%d calls=%d stderr=%s", code, calls, stderr)
			}
		})
	}
	for _, status := range []int{http.StatusNoContent, http.StatusPreconditionFailed} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			calls := 0
			code, _, stderr := memoryArgs(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method == http.MethodGet {
					w.Header().Set("ETag", etag)
					io.WriteString(w, validCommand)
					return
				}
				if r.Method != http.MethodPut || r.Header.Get("If-Match") != etag {
					t.Fatalf("unexpected write: %s %q", r.Method, r.Header.Get("If-Match"))
				}
				w.WriteHeader(status)
			}, validCommand, "commands", "deploy", "-f", "-")
			want := exitOK
			if status == http.StatusPreconditionFailed {
				want = 4
			}
			if code != want || calls != 2 {
				t.Fatalf("code=%d calls=%d stderr=%s", code, calls, stderr)
			}
		})
	}
}

func TestUnavailableFailActionStopsBeforeTransport(t *testing.T) {
	for _, args := range [][]string{
		{"executions", "action", "id", "--type", "FAIL", "--reason", "test"},
		{"executions", "action", "id", "--data", "{\"type\":\"FAIL\",\"reason\":\"test\"}"},
		{"executions", "action", "id", "-f", "-"},
	} {
		code, _, stderr := memoryArgs(t, func(http.ResponseWriter, *http.Request) { t.Fatal("unavailable action reached transport") }, "{\"type\":\"FAIL\",\"reason\":\"test\"}", args...)
		if code != exitUsage {
			t.Fatalf("code=%d stderr=%s", code, stderr)
		}
	}
}
