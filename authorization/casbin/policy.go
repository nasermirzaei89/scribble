package casbin

import (
	"strings"
)

// parsePolicyCSV converts a multi-line casbin policy CSV string into a 2-D
// slice of strings suitable for LoadPolicyArray.
func parsePolicyCSV(content string) [][]string {
	var rules [][]string

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, ",")

		rule := make([]string, 0, len(parts))
		for _, p := range parts {
			rule = append(rule, strings.TrimSpace(p))
		}

		if len(rule) > 0 {
			rules = append(rules, rule)
		}
	}

	return rules
}
