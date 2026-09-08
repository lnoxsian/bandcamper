package bandcamp

import (
	"fmt"
	"net/url"
	"strings"
)

// PageType represents the classified Bandcamp page type.
type PageType int

const (
	PageUnknown PageType = iota
	PageArtist
	PageAlbum
	PageTrack
)

func (p PageType) String() string {
	switch p {
	case PageArtist:
		return "artist"
	case PageAlbum:
		return "album"
	case PageTrack:
		return "track"
	default:
		return "unknown"
	}
}

// ResolvedURL contains the result of resolving and normalizing a Bandcamp URL.
type ResolvedURL struct {
	OriginalURL   string
	NormalizedURL string
	Type          PageType
	Host          string
	Artist        string
	Slug          string
}

// ResolveURL validates, classifies, and normalizes a Bandcamp URL.
func ResolveURL(rawURL string) (*ResolvedURL, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, fmt.Errorf("%w: empty URL", ErrInvalidURL)
	}

	// Prepend https:// if no scheme is specified
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}

	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return nil, fmt.Errorf("%w: missing host", ErrInvalidURL)
	}

	// Check if this looks like a Bandcamp domain or artist subdomain
	isBandcamp := strings.HasSuffix(host, ".bandcamp.com") || host == "bandcamp.com"

	// If not a bandcamp.com host, require album or track path for custom domain support
	if !isBandcamp && !strings.Contains(host, "bandcamp") {
		cleanP := strings.Trim(strings.TrimRight(parsed.Path, "/"), "/")
		segs := strings.Split(cleanP, "/")
		if len(segs) < 2 || (segs[0] != "album" && segs[0] != "track") {
			return nil, fmt.Errorf("%w: host %q is not a Bandcamp domain", ErrInvalidURL, host)
		}
	}

	// Normalize scheme to https
	parsed.Scheme = "https"
	// Clear query and fragment for canonical release/track/artist URL
	parsed.RawQuery = ""
	parsed.Fragment = ""

	cleanPath := strings.TrimRight(parsed.Path, "/")
	segments := strings.Split(strings.Trim(cleanPath, "/"), "/")

	pageType := PageUnknown
	slug := ""

	if len(segments) == 0 || (len(segments) == 1 && segments[0] == "") {
		// Root artist page e.g. https://artist.bandcamp.com/
		if !isBandcamp {
			return nil, fmt.Errorf("%w: host %q is not a recognized Bandcamp domain", ErrInvalidURL, host)
		}
		pageType = PageArtist
	} else if len(segments) == 1 && (segments[0] == "music" || segments[0] == "releases") {
		if !isBandcamp {
			return nil, fmt.Errorf("%w: host %q is not a recognized Bandcamp domain", ErrInvalidURL, host)
		}
		pageType = PageArtist
		cleanPath = "/" + segments[0]
	} else if len(segments) >= 2 && segments[0] == "album" {
		pageType = PageAlbum
		slug = segments[1]
		cleanPath = "/album/" + slug
	} else if len(segments) >= 2 && segments[0] == "track" {
		pageType = PageTrack
		slug = segments[1]
		cleanPath = "/track/" + slug
	} else {
		return nil, fmt.Errorf("%w: unsupported path %q", ErrUnsupportedPage, parsed.Path)
	}

	artistSubdomain := ""
	if strings.HasSuffix(host, ".bandcamp.com") {
		artistSubdomain = strings.TrimSuffix(host, ".bandcamp.com")
	}

	parsed.Path = cleanPath
	normalized := parsed.String()

	return &ResolvedURL{
		OriginalURL:   rawURL,
		NormalizedURL: normalized,
		Type:          pageType,
		Host:          host,
		Artist:        artistSubdomain,
		Slug:          slug,
	}, nil
}
