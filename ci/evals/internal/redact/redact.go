// Package redact protects persisted evaluation evidence, not candidate correctness.
package redact

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
)

func EnvironmentSecrets() []string {
	var values []string
	for _, entry := range os.Environ() {
		key, value, _ := strings.Cut(entry, "=")
		key = strings.ToUpper(key)
		sensitive := strings.Contains(key, "API_KEY") || strings.HasSuffix(key, "_TOKEN") || strings.HasSuffix(key, "_SECRET") || strings.HasSuffix(key, "_PASSWORD") || strings.HasSuffix(key, "PRIVATE_KEY")
		if sensitive && value != "" {
			values = append(values, value)
		}
	}
	return values
}
func variants(secrets []string) []string {
	unique := map[string]bool{}
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		unique[secret] = true
		encoded, _ := json.Marshal(secret)
		unique[string(encoded[1:len(encoded)-1])] = true
	}
	out := make([]string, 0, len(unique))
	for value := range unique {
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return len(out[i]) > len(out[j]) })
	return out
}
func Contains(text string, secrets []string) bool {
	for _, secret := range variants(secrets) {
		if strings.Contains(text, secret) {
			return true
		}
	}
	return false
}
func Text(text string, secrets []string) string {
	for _, secret := range variants(secrets) {
		text = strings.ReplaceAll(text, secret, "[REDACTED]")
	}
	return text
}
func Value(value any, secrets []string) (any, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var safe any
	err = json.Unmarshal([]byte(Text(string(raw), secrets)), &safe)
	return safe, err
}
