package bandcamp

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strings"
)

var (
	albumHrefRegex       = regexp.MustCompile(`(?i)href=["'](/album/[a-zA-Z0-9_-]+|https?://[^"']+/album/[a-zA-Z0-9_-]+)["']`)
	trackHrefRegex       = regexp.MustCompile(`(?i)href=["'](/track/[a-zA-Z0-9_-]+|https?://[^"']+/track/[a-zA-Z0-9_-]+)["']`)
	releaseHrefRegex     = regexp.MustCompile(`(?i)href=["']((?:https?://[^"']+)?/(?:album|track)/[a-zA-Z0-9_-]+(?:\?[^"']*)?)["']`)
	dataClientItemsRegex = regexp.MustCompile(`(?s)data-client-items=["']([^"']+)["']`)
	trackTableRegex      = regexp.MustCompile(`(?is)<table[^>]*id=["']track_table["'][^>]*>.*?</table>`)
)

type clientGridItem struct {
	PageURL string `json:"page_url"`
	Type    string `json:"type"`
	Title   string `json:"title"`
}

// ParseArtistDiscography extracts all release (album & track) URLs found on an artist page.
// It inspects both server-rendered HTML (inside #music-grid or the page) and client-side pagination
// data stored in the `data-client-items` attribute.
func ParseArtistDiscography(htmlContent, artistURL string) ([]string, error) {
	seen := make(map[string]bool)
	var releases []string

	addRelease := func(rawPathOrURL string) {
		fullURL := ResolveAbsoluteURL(artistURL, rawPathOrURL)
		res, err := ResolveURL(fullURL)
		if err == nil && (res.Type == PageAlbum || res.Type == PageTrack) && !seen[res.NormalizedURL] {
			seen[res.NormalizedURL] = true
			releases = append(releases, res.NormalizedURL)
		}
	}

	// 1. If #music-grid is present, extract server-rendered releases from inside the grid in DOM order.
	gridContent := extractMusicGridContent(htmlContent)
	if gridContent != "" {
		matches := releaseHrefRegex.FindAllStringSubmatch(gridContent, -1)
		for _, m := range matches {
			if len(m) > 1 {
				addRelease(m[1])
			}
		}
	} else {
		// Non-grid page: if an album tracklist table exists (e.g. on a featured album page),
		// exclude it so tracklist items are not mistaken for standalone discography singles.
		cleanedHTML := trackTableRegex.ReplaceAllString(htmlContent, "")
		matches := releaseHrefRegex.FindAllStringSubmatch(cleanedHTML, -1)
		for _, m := range matches {
			if len(m) > 1 {
				addRelease(m[1])
			}
		}
	}

	// 2. Extract releases from data-client-items JSON attribute (used by Bandcamp for pagination / infinite scroll).
	clientMatches := dataClientItemsRegex.FindAllStringSubmatch(htmlContent, -1)
	for _, m := range clientMatches {
		if len(m) > 1 {
			unescaped := html.UnescapeString(m[1])
			var items []clientGridItem
			if err := json.Unmarshal([]byte(unescaped), &items); err == nil {
				for _, item := range items {
					if item.PageURL != "" {
						addRelease(item.PageURL)
					}
				}
			}
		}
	}

	if len(releases) == 0 {
		return nil, fmt.Errorf("%w: no album or track releases found on %s", ErrParseFailed, artistURL)
	}

	return releases, nil
}

// extractMusicGridContent finds the container with id="music-grid" and returns its HTML.
func extractMusicGridContent(htmlContent string) string {
	idx := strings.Index(htmlContent, `id="music-grid"`)
	if idx == -1 {
		idx = strings.Index(htmlContent, `id='music-grid'`)
	}
	if idx == -1 {
		return ""
	}

	// Find the start of the opening tag '<' before id="music-grid"
	tagStart := strings.LastIndex(htmlContent[:idx], "<")
	if tagStart == -1 {
		return ""
	}

	// Determine the opening tag name (e.g. "ol", "ul", "div")
	tagRest := htmlContent[tagStart+1:]
	tagEnd := strings.IndexAny(tagRest, " \t\r\n>")
	if tagEnd == -1 {
		return ""
	}
	tagName := strings.ToLower(tagRest[:tagEnd])
	closingTag := "</" + tagName + ">"

	closeIdx := strings.Index(htmlContent[idx:], closingTag)
	if closeIdx == -1 {
		return htmlContent[tagStart:]
	}
	return htmlContent[tagStart : idx+closeIdx+len(closingTag)]
}
