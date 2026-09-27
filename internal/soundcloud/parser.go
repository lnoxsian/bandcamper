package soundcloud

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var (
	hydrationRegex = regexp.MustCompile(`(?s)window\.__sc_hydration\s*=\s*(\[.*?\])\s*;\s*</script>`)
	clientIDRegex  = regexp.MustCompile(`client_id[:=]["']([0-9a-zA-Z]{32})["']`)
)

// ExtractHydration parses the embedded window.__sc_hydration JSON payload from SoundCloud HTML.
func ExtractHydration(html string) ([]HydrationItem, error) {
	matches := hydrationRegex.FindStringSubmatch(html)
	if len(matches) < 2 {
		return nil, fmt.Errorf("%w: window.__sc_hydration not found in HTML", ErrSoundCloudParse)
	}

	var items []HydrationItem
	if err := json.Unmarshal([]byte(matches[1]), &items); err != nil {
		return nil, fmt.Errorf("%w: unmarshaling hydration data: %v", ErrSoundCloudParse, err)
	}

	return items, nil
}

// ExtractClientID attempts to locate the public client_id from hydration data or HTML.
func ExtractClientID(html string, hydration []HydrationItem) string {
	// 1. Try finding apiClient in hydration items
	for _, item := range hydration {
		if item.Hydratable == "apiClient" {
			var clientData ApiClientData
			if err := json.Unmarshal(item.Data, &clientData); err == nil && clientData.ID != "" {
				return clientData.ID
			}
		}
	}

	// 2. Try regex in HTML
	matches := clientIDRegex.FindStringSubmatch(html)
	if len(matches) >= 2 {
		return matches[1]
	}

	return ""
}

// ParseTrack extracts an individual track model from SoundCloud HTML.
func ParseTrack(html string) (*SoundCloudTrack, error) {
	hydration, err := ExtractHydration(html)
	if err != nil {
		return nil, err
	}

	for _, item := range hydration {
		if item.Hydratable == "sound" {
			var track SoundCloudTrack
			if err := json.Unmarshal(item.Data, &track); err != nil {
				return nil, fmt.Errorf("%w: parsing track data: %v", ErrSoundCloudParse, err)
			}
			if track.Title == "" && track.ID == 0 {
				return nil, fmt.Errorf("%w: track has empty metadata", ErrSoundCloudParse)
			}
			return &track, nil
		}
	}

	return nil, fmt.Errorf("%w: no sound hydratable found", ErrSoundCloudNotFound)
}

// ParsePlaylist extracts a set / playlist model from SoundCloud HTML.
func ParsePlaylist(html string) (*SoundCloudPlaylist, error) {
	hydration, err := ExtractHydration(html)
	if err != nil {
		return nil, err
	}

	for _, item := range hydration {
		if item.Hydratable == "playlist" {
			var playlist SoundCloudPlaylist
			if err := json.Unmarshal(item.Data, &playlist); err != nil {
				return nil, fmt.Errorf("%w: parsing playlist data: %v", ErrSoundCloudParse, err)
			}
			return &playlist, nil
		}
	}

	return nil, fmt.Errorf("%w: no playlist hydratable found", ErrSoundCloudNotFound)
}

// ParseUserTracks extracts individual track links from an artist profile page.
func ParseUserTracks(html string, artistSlug string) ([]string, error) {
	cleanSlug := strings.ToLower(strings.Trim(artistSlug, "/"))
	pattern := fmt.Sprintf(`href=["']/(%s/[a-zA-Z0-9_-]+)["']`, regexp.QuoteMeta(cleanSlug))
	re := regexp.MustCompile(pattern)

	matches := re.FindAllStringSubmatch(html, -1)
	seen := make(map[string]bool)
	var trackURLs []string

	ignoredSuffixes := []string{"/sets", "/likes", "/tracks", "/albums", "/reposts", "/followers", "/following"}

	for _, m := range matches {
		if len(m) > 1 {
			p := "/" + strings.Trim(m[1], "/")
			lower := strings.ToLower(p)

			ignore := false
			for _, ig := range ignoredSuffixes {
				if strings.HasSuffix(lower, ig) {
					ignore = true
					break
				}
			}

			fullURL := "https://soundcloud.com" + p
			if !ignore && !seen[fullURL] {
				seen[fullURL] = true
				trackURLs = append(trackURLs, fullURL)
			}
		}
	}

	if len(trackURLs) == 0 {
		return nil, fmt.Errorf("%w: no track links found on artist profile %q", ErrSoundCloudNotFound, artistSlug)
	}

	return trackURLs, nil
}
