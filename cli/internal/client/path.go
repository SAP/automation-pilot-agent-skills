package client

import "strings"

// JoinPath builds an API path from segments, trimming stray slashes. Segments
// such as command IDs ("catalog:name:version") are URL-safe as-is and passed
// through verbatim to match the server's expectation (the previous curl calls
// sent raw colons).
func JoinPath(segments ...string) string {
	parts := make([]string, 0, len(segments))
	for _, s := range segments {
		s = strings.Trim(s, "/")
		if s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, "/")
}
