package epub

import "strings"

func splitCommaList(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '，' || r == ';' || r == '；'
	})
	seen := map[string]bool{}
	var out []string
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		key := strings.ToLower(field)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, field)
	}
	return out
}
