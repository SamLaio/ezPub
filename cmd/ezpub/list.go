package main

import "strings"

func splitCommaList(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '，' || r == ';' || r == '；'
	})
	return cleanListFields(fields)
}

func cleanListFields(fields []string) []string {
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

func normalizeCommaListText(value string) string {
	return strings.Join(splitCommaList(value), ", ")
}
