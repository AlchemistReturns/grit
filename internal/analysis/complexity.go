package analysis

import (
	"regexp"
	"strings"
	"unicode"
)

var complexityKeywords = []string{
	"if ", "else ", "else{", "for ", "for{",
	"switch ", "case ", "select {", "select{",
	"&&", "||", "catch ", "except ", "rescue ",
	"while ", "do {", "do{",
}

// Score computes a file-wide cyclomatic-complexity-like score for source content.
func Score(content string) float64 {
	score := 1.0
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimLeftFunc(line, unicode.IsSpace)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		for _, kw := range complexityKeywords {
			score += float64(strings.Count(line, kw))
		}
	}
	return score
}

// FunctionScore holds the complexity score for a single function.
type FunctionScore struct {
	Name  string
	Score float64
	Line  int
}

var funcDeclPatterns = []*regexp.Regexp{
	// Go: func Name( or func (recv) Name(
	regexp.MustCompile(`^func\s+(?:\(\w+\s+\*?\w+\)\s+)?(\w+)\s*\(`),
	// Python
	regexp.MustCompile(`^\s{0,4}(?:async\s+)?def\s+(\w+)\s*\(`),
	// JavaScript/TypeScript
	regexp.MustCompile(`^(?:export\s+)?(?:async\s+)?function\s+(\w+)\s*\(`),
	regexp.MustCompile(`^(?:export\s+)?(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?\(`),
	// Rust
	regexp.MustCompile(`^(?:pub(?:\(crate\))?\s+)?(?:async\s+)?fn\s+(\w+)\s*`),
	// Java/C#/C++ (simplified)
	regexp.MustCompile(`^\s*(?:(?:public|private|protected|static|virtual|override|async|abstract)\s+)+\w[\w<>\[\]]*\s+(\w+)\s*\(`),
}

func detectFuncName(line string) string {
	trimmed := strings.TrimLeftFunc(line, unicode.IsSpace)
	for _, pat := range funcDeclPatterns {
		if m := pat.FindStringSubmatch(trimmed); len(m) > 1 {
			return m[1]
		}
	}
	return ""
}

// ScoreByFunction parses content into function blocks and scores each independently.
// Functions are delimited heuristically by detecting declaration lines.
func ScoreByFunction(content string) []FunctionScore {
	lines := strings.Split(content, "\n")
	var results []FunctionScore

	currentName := ""
	currentLine := 0
	currentScore := 1.0

	for i, line := range lines {
		if name := detectFuncName(line); name != "" {
			if currentName != "" {
				results = append(results, FunctionScore{
					Name:  currentName,
					Score: currentScore,
					Line:  currentLine,
				})
			}
			currentName = name
			currentLine = i + 1
			currentScore = 1.0
			continue
		}

		if currentName == "" {
			continue
		}

		trimmed := strings.TrimLeftFunc(line, unicode.IsSpace)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		for _, kw := range complexityKeywords {
			currentScore += float64(strings.Count(line, kw))
		}
	}

	if currentName != "" {
		results = append(results, FunctionScore{
			Name:  currentName,
			Score: currentScore,
			Line:  currentLine,
		})
	}

	return results
}
