package bandcamp

import (
	"fmt"
	"regexp"
)

var (
	albumHrefRegex = regexp.MustCompile(`(?i)href=["'](/album/[a-zA-Z0-9_-]+|https?://[^"']+/album/[a-zA-Z0-9_-]+)["']`)
	trackHrefRegex = regexp.MustCompile(`(?i)href=["'](/track/[a-zA-Z0-9_-]+|https?://[^"']+/track/[a-zA-Z0-9_-]+)["']`)
)

// ParseArtistDiscography extracts all release (album & track) URLs found on an artist page.
func ParseArtistDiscography(htmlContent, artistURL string) ([]string, error) {
	seen := make(map[string]bool)
	var releases []string

	// Find album links
	albumMatches := albumHrefRegex.FindAllStringSubmatch(htmlContent, -1)
	for _, m := range albumMatches {
		if len(m) > 1 {
			fullURL := ResolveAbsoluteURL(artistURL, m[1])
			res, err := ResolveURL(fullURL)
			if err == nil && !seen[res.NormalizedURL] {
				seen[res.NormalizedURL] = true
				releases = append(releases, res.NormalizedURL)
			}
		}
	}

	// Find standalone track links
	trackMatches := trackHrefRegex.FindAllStringSubmatch(htmlContent, -1)
	for _, m := range trackMatches {
		if len(m) > 1 {
			fullURL := ResolveAbsoluteURL(artistURL, m[1])
			res, err := ResolveURL(fullURL)
			if err == nil && !seen[res.NormalizedURL] {
				seen[res.NormalizedURL] = true
				releases = append(releases, res.NormalizedURL)
			}
		}
	}

	if len(releases) == 0 {
		return nil, fmt.Errorf("%w: no album or track releases found on %s", ErrParseFailed, artistURL)
	}

	return releases, nil
}
