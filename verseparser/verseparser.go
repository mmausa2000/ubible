package verseparser

import (
	"regexp"
	"strings"
)

var numPrefix = regexp.MustCompile(`^\d+\.`)

func normalize(line string) string {
	line = strings.TrimSpace(line)
	line = strings.ReplaceAll(line, "\u202F", " ")
	line = strings.ReplaceAll(line, "\u00A0", " ")
	line = strings.ReplaceAll(line, "–", " ")
	line = strings.ReplaceAll(line, "—", " ")
	line = strings.ReplaceAll(line, "=>", " ")
	line = strings.ReplaceAll(line, "->", " ")
	return line
}

func ParseVerseSmart(line string) (string, string) {
	line = normalize(line)
	tokens := strings.Fields(line)
	
	if len(tokens) < 5 {
		return "", ""
	}
	
	// Find colon token
	colonIdx := -1
	for i, t := range tokens {
		if strings.Contains(t, ":") {
			colonIdx = i
			break
		}
	}
	
	if colonIdx == -1 {
		return "", ""
	}
	
	// Book name: up to 3 words before colon
	bookStart := 0
	if numPrefix.MatchString(tokens[0]) {
		bookStart = 1
	}
	
	if colonIdx - bookStart > 3 {
		bookStart = colonIdx - 3
	}
	
	// Build reference: book + chapter:verse
	refTokens := tokens[bookStart : colonIdx+1]
	
	// Check for verse range: "5:22 — 23" or "5:22 23"
	textStart := colonIdx + 1
	if textStart < len(tokens) {
		next := tokens[textStart]
		if regexp.MustCompile(`^\d+$`).MatchString(next) {
			// It's a verse range
			refTokens = append(refTokens, next)
			textStart++
		}
	}
	
	// Text must have at least 2 words
	if len(tokens) - textStart < 2 {
		return "", ""
	}
	
	// Build reference string
	ref := strings.Join(refTokens, " ")
	ref = numPrefix.ReplaceAllString(ref, "")
	ref = strings.TrimSpace(ref)
	
	// Normalize verse range format
	if regexp.MustCompile(`\d+:\d+\s+\d+$`).MatchString(ref) {
		ref = regexp.MustCompile(`(\d+:\d+)\s+(\d+)`).ReplaceAllString(ref, "$1-$2")
	}
	
	// Build text string
	text := strings.Join(tokens[textStart:], " ")
	text = strings.TrimSpace(text)
	
	return ref, text
}
