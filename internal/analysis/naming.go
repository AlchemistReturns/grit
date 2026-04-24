package analysis

import (
	"regexp"
	"strings"
)

var (
	funcPattern  = regexp.MustCompile(`(?i)\bfunc\s+(handle|process|do|get|set|manage|run)\w*\s*\(`)
	varPattern   = regexp.MustCompile(`(?i)\b(var|let|const)\s+(result|temp|tmp|data|stuff|thing|value|item|obj|res|resp|req)\b`)
	shortDeclPat = regexp.MustCompile(`(?i)\b(result|temp|tmp|data|stuff|thing|value|item|obj|res|resp|req)\s*:=`)
)

// FindWeakName checks new lines for vague identifiers using built-in lists.
// Returns the first matched name, or empty string if none found.
func FindWeakName(newLines []string) string {
	for _, line := range newLines {
		if m := funcPattern.FindString(line); m != "" {
			return extractIdentifier(m)
		}
		if m := varPattern.FindStringSubmatch(line); m != nil {
			return m[2]
		}
		if m := shortDeclPat.FindStringSubmatch(line); m != nil {
			return m[1]
		}
	}
	return ""
}

// FindWeakNameWithExtra checks new lines for weak identifiers, including language-specific extras.
func FindWeakNameWithExtra(newLines []string, extraNames []string) string {
	if name := FindWeakName(newLines); name != "" {
		return name
	}
	if len(extraNames) == 0 {
		return ""
	}
	quoted := make([]string, len(extraNames))
	for i, n := range extraNames {
		quoted[i] = regexp.QuoteMeta(n)
	}
	extraPat := regexp.MustCompile(`(?i)\b(` + strings.Join(quoted, "|") + `)\s*:=`)
	for _, line := range newLines {
		if m := extraPat.FindStringSubmatch(line); m != nil {
			return m[1]
		}
	}
	return ""
}

func extractIdentifier(s string) string {
	parts := strings.Fields(s)
	for i, p := range parts {
		if strings.EqualFold(p, "func") && i+1 < len(parts) {
			name := parts[i+1]
			if idx := strings.Index(name, "("); idx >= 0 {
				name = name[:idx]
			}
			return name
		}
	}
	return s
}

// DiffLines returns lines in current that are not in previous (set subtraction by content).
func DiffLines(previous, current string) []string {
	prevSet := make(map[string]struct{})
	for _, l := range strings.Split(previous, "\n") {
		prevSet[strings.TrimSpace(l)] = struct{}{}
	}
	var added []string
	for _, l := range strings.Split(current, "\n") {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" {
			continue
		}
		if _, exists := prevSet[trimmed]; !exists {
			added = append(added, l)
		}
	}
	return added
}
