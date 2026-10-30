// Package domain owns the product rules shared by request building and linting.
package domain

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// Snapshot identifies the reviewed product contract, not live tenant validation.
const Snapshot = "2026-10-22"

var Durations = []string{"5s", "10s", "15s", "30s", "1m", "2m", "3m"}

type Feature struct {
	Name   string `json:"name"`
	Values string `json:"values"`
}

var Features = []Feature{
	{"feature:priority", "high|medium|low"},
	{"feature:autoAbort", "integer hours 1..168"},
	{"feature:retentionPolicy", "archive|delete"},
	{"feature:retentionPeriod", "integer hours 1..672 for archive; 0..672 for delete"},
	{"feature:logs", "empty string"},
	{"feature:dryRun", "empty string"},
}

var secretWords = regexp.MustCompile(`(?i)(password|passphrase|secret|credential|private.?key|api.?key)`)

// SecretKey excludes endpoint/metadata names and recognizes known secret contracts.
func SecretKey(name string) bool {
	n := strings.ToLower(strings.NewReplacer("_", "", "-", "", ".", "").Replace(name))
	if strings.HasSuffix(n, "url") || strings.HasSuffix(n, "uri") || strings.HasSuffix(n, "id") || strings.HasSuffix(n, "name") || strings.HasSuffix(n, "type") || strings.HasSuffix(n, "count") {
		return false
	}
	return secretWords.MatchString(name) || strings.HasSuffix(n, "token") || n == "authorization" || n == "authorizationheader" || n == "proxyauthorization" || n == "servicekey" || n == "clientcert" || n == "certificate" || n == "cookie" || n == "setcookie" || n == "stdin" || n == "npmrc"
}

func ValidDuration(value string) bool {
	for _, allowed := range Durations {
		if value == allowed {
			return true
		}
	}
	return false
}

// ValidateTags checks authored tags. Service tags are allowed only on read-back.
func ValidateTags(tags map[string]string, readBack bool) error {
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := tags[key]
		if len(key) < 1 || len(key) > 128 || strings.TrimSpace(key) == "" || strings.Contains(key, "=") || !tagChars(key) {
			return fmt.Errorf("invalid tag key")
		}
		if len(value) > 256 || !tagChars(value) {
			return fmt.Errorf("invalid value for tag %q", key)
		}
		lower := strings.ToLower(key)
		if strings.HasPrefix(lower, "autopi:") {
			if !readBack {
				return fmt.Errorf("tag %q uses the reserved autopi: service prefix", key)
			}
			continue
		}
		if !strings.HasPrefix(lower, "feature:") {
			continue
		}
		switch lower {
		case "feature:priority":
			if value != "high" && value != "medium" && value != "low" {
				return fmt.Errorf("invalid priority tag value")
			}
		case "feature:autoabort":
			if !integerRange(value, 1, 168) {
				return fmt.Errorf("autoAbort must be integer hours 1..168")
			}
		case "feature:retentionpolicy":
			if value != "archive" && value != "delete" {
				return fmt.Errorf("retentionPolicy must be archive or delete")
			}
		case "feature:retentionperiod":
			min := 1
			for k, v := range tags {
				if strings.EqualFold(k, "feature:retentionPolicy") && v == "delete" {
					min = 0
				}
			}
			if !integerRange(value, min, 672) {
				return fmt.Errorf("retentionPeriod is outside the policy's allowed hours")
			}
		case "feature:logs", "feature:dryrun":
			if value != "" {
				return fmt.Errorf("%s requires an empty string", key)
			}
		default:
			return fmt.Errorf("feature tag %q is not in the allow-list", key)
		}
	}
	return nil
}

func integerRange(value string, min, max int) bool {
	n, err := strconv.Atoi(value)
	return err == nil && strconv.Itoa(n) == value && n >= min && n <= max
}

func tagChars(value string) bool {
	for _, r := range value {
		if r > unicode.MaxASCII || !(unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune(" _.:/=+-@", r)) {
			return false
		}
	}
	return true
}
