package book

import (
	"path/filepath"
	"regexp"
	"strings"
)

var imgSrcRE = regexp.MustCompile(`(?i)(<img\b[^>]*\bsrc\s*=\s*)(["'])([^"']+)(["'])`)

func rewriteImageRefs(html string, resolver map[string]string) (string, []string) {
	var found []string
	rewritten := imgSrcRE.ReplaceAllStringFunc(html, func(match string) string {
		parts := imgSrcRE.FindStringSubmatch(match)
		if len(parts) != 5 {
			return match
		}
		src := strings.TrimSpace(parts[3])
		if isExternalImageRef(src) {
			return match
		}
		path, ok := resolveImagePath(src, resolver)
		if !ok {
			return match
		}
		found = append(found, path)
		return parts[1] + parts[2] + "../images/" + filepath.Base(path) + parts[4]
	})
	return rewritten, found
}

func resolveImagePath(src string, resolver map[string]string) (string, bool) {
	keys := []string{
		strings.ToLower(filepath.ToSlash(src)),
		strings.ToLower(filepath.Base(src)),
	}
	for _, key := range keys {
		if path, ok := resolver[key]; ok {
			return path, true
		}
	}
	return "", false
}

func isExternalImageRef(src string) bool {
	lower := strings.ToLower(src)
	return strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "data:") ||
		strings.HasPrefix(lower, "#") ||
		strings.HasPrefix(lower, "../images/")
}

func cleanStrings(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, value)
	}
	return out
}

func splitCommaList(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '，' || r == ';' || r == '；'
	})
	return cleanStrings(fields)
}
