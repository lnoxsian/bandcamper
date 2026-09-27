package bandcamp

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/lnoxsian/bandcamper/internal/provider"
)

// BandcampProvider implements provider.Provider for Bandcamp.
type BandcampProvider struct {
	Client *Client
}

// NewProvider creates a new BandcampProvider.
func NewProvider(client *Client) *BandcampProvider {
	if client == nil {
		client = NewClient(0, "")
	}
	return &BandcampProvider{
		Client: client,
	}
}

// Name returns "bandcamp".
func (p *BandcampProvider) Name() string {
	return "bandcamp"
}

// CanHandle determines if the URL belongs to Bandcamp.
func (p *BandcampProvider) CanHandle(u *url.URL) bool {
	if u == nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if strings.HasSuffix(host, ".bandcamp.com") || host == "bandcamp.com" {
		return true
	}

	cleanP := strings.Trim(strings.TrimRight(u.Path, "/"), "/")
	segs := strings.Split(cleanP, "/")
	if len(segs) >= 2 && (segs[0] == "album" || segs[0] == "track") {
		return true
	}
	return false
}

// Resolve fetches and parses Bandcamp artist, album, or track pages into normalized Releases.
func (p *BandcampProvider) Resolve(ctx context.Context, u *url.URL) ([]*provider.Release, error) {
	resolved, err := ResolveURL(u.String())
	if err != nil {
		return nil, err
	}

	switch resolved.Type {
	case PageArtist:
		htmlContent, err := p.Client.FetchString(ctx, resolved.NormalizedURL, 5*1024*1024)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch artist page %s: %w", resolved.NormalizedURL, err)
		}

		releaseURLs, err := ParseArtistDiscography(htmlContent, resolved.NormalizedURL)
		if err != nil {
			return nil, err
		}

		var releases []*provider.Release
		for _, relURL := range releaseURLs {
			subU, err := url.Parse(relURL)
			if err != nil {
				continue
			}
			subRels, err := p.Resolve(ctx, subU)
			if err != nil {
				continue
			}
			releases = append(releases, subRels...)
		}
		if len(releases) == 0 {
			return nil, fmt.Errorf("%w: no releases found for %s", ErrParseFailed, resolved.NormalizedURL)
		}
		return releases, nil

	case PageAlbum:
		htmlContent, err := p.Client.FetchString(ctx, resolved.NormalizedURL, 5*1024*1024)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch album page %s: %w", resolved.NormalizedURL, err)
		}
		release, err := ParseAlbum(htmlContent, resolved.NormalizedURL)
		if err != nil {
			return nil, err
		}
		return []*provider.Release{release}, nil

	case PageTrack:
		htmlContent, err := p.Client.FetchString(ctx, resolved.NormalizedURL, 5*1024*1024)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch track page %s: %w", resolved.NormalizedURL, err)
		}
		release, err := ParseTrack(htmlContent, resolved.NormalizedURL)
		if err != nil {
			return nil, err
		}
		return []*provider.Release{release}, nil

	default:
		return nil, fmt.Errorf("%w: unrecognized page type for %s", ErrUnsupportedPage, u.String())
	}
}
