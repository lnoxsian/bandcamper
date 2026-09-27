package soundcloud

import (
	"fmt"
	"net/url"
	"strings"
)

// PageType represents the classified SoundCloud resource type.
type PageType int

const (
	PageUnknown PageType = iota
	PageTrack
	PagePlaylist
	PageUser
)

func (p PageType) String() string {
	switch p {
	case PageTrack:
		return "track"
	case PagePlaylist:
		return "playlist"
	case PageUser:
		return "artist"
	default:
		return "unknown"
	}
}

// ResolvedURL contains classification and normalized components of a SoundCloud URL.
type ResolvedURL struct {
	OriginalURL   string
	NormalizedURL string
	Type          PageType
	Artist        string
	Slug          string
}

var reservedPaths = map[string]bool{
	"discover":     true,
	"stream":       true,
	"upload":       true,
	"search":       true,
	"popular":      true,
	"charts":       true,
	"settings":     true,
	"messages":     true,
	"terms-of-use": true,
	"pages":        true,
	"jobs":         true,
	"imprint":      true,
	"you":          true,
	"mobile":       true,
	"stations":     true,
}

// ResolveURL parses, normalizes, and classifies a SoundCloud URL.
func ResolveURL(rawURL string) (*ResolvedURL, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, fmt.Errorf("%w: empty URL", ErrInvalidSoundCloudURL)
	}

	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSoundCloudURL, err)
	}

	host := strings.ToLower(parsed.Hostname())
	if !strings.HasSuffix(host, "soundcloud.com") {
		return nil, fmt.Errorf("%w: host %q is not SoundCloud", ErrInvalidSoundCloudURL, host)
	}

	parsed.Scheme = "https"
	parsed.Host = "soundcloud.com"
	parsed.RawQuery = ""
	parsed.Fragment = ""

	cleanPath := strings.TrimRight(parsed.Path, "/")
	segments := strings.Split(strings.Trim(cleanPath, "/"), "/")

	if len(segments) == 0 || (len(segments) == 1 && segments[0] == "") {
		return nil, fmt.Errorf("%w: missing user/track path", ErrInvalidSoundCloudURL)
	}

	firstSeg := strings.ToLower(segments[0])
	if reservedPaths[firstSeg] {
		return nil, fmt.Errorf("%w: reserved path %q cannot be resolved", ErrInvalidSoundCloudURL, firstSeg)
	}

	pageType := PageUnknown
	artist := segments[0]
	slug := ""

	if len(segments) == 1 {
		// e.g. https://soundcloud.com/artist
		pageType = PageUser
	} else if len(segments) == 2 {
		if strings.ToLower(segments[1]) == "tracks" || strings.ToLower(segments[1]) == "albums" || strings.ToLower(segments[1]) == "sets" {
			pageType = PageUser
		} else {
			// e.g. https://soundcloud.com/artist/track-slug
			pageType = PageTrack
			slug = segments[1]
		}
	} else if len(segments) >= 3 && strings.ToLower(segments[1]) == "sets" {
		// e.g. https://soundcloud.com/artist/sets/playlist-slug
		pageType = PagePlaylist
		slug = segments[2]
	} else {
		// Fallback to track for deeper links
		pageType = PageTrack
		slug = segments[len(segments)-1]
	}

	parsed.Path = "/" + strings.Join(segments, "/")
	normalized := parsed.String()

	return &ResolvedURL{
		OriginalURL:   rawURL,
		NormalizedURL: normalized,
		Type:          pageType,
		Artist:        artist,
		Slug:          slug,
	}, nil
}
