package metadata

import (
	"html"
	"regexp"
	"strings"
)

var (
	brRegex  = regexp.MustCompile(`(?i)<br\s*/?>`)
	pRegex   = regexp.MustCompile(`(?i)</?p\s*/?>`)
	tagRegex = regexp.MustCompile(`<[^>]*>`)
)

// NormalizeLyrics strips HTML tags, normalizes line breaks, and trims whitespace.
func NormalizeLyrics(raw string) string {
	if raw == "" {
		return ""
	}
	s := html.UnescapeString(raw)
	s = brRegex.ReplaceAllString(s, "\n")
	s = pRegex.ReplaceAllString(s, "\n")
	s = tagRegex.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.TrimSpace(s)
}

// HasLyrics checks whether lyrics content is non-empty.
func HasLyrics(lyrics string) bool {
	return len(strings.TrimSpace(lyrics)) > 0
}
