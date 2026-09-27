package soundcloud

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/lnoxsian/bandcamper/internal/provider"
)

// Provider implements provider.Provider for SoundCloud.
type Provider struct {
	Client *Client
}

// NewProvider creates a new SoundCloud provider with the given client.
func NewProvider(client *Client) *Provider {
	if client == nil {
		client = NewClient(0, "")
	}
	return &Provider{
		Client: client,
	}
}

// Name returns "soundcloud".
func (p *Provider) Name() string {
	return "soundcloud"
}

// CanHandle checks if the URL points to soundcloud.com.
func (p *Provider) CanHandle(u *url.URL) bool {
	if u == nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return strings.HasSuffix(host, "soundcloud.com")
}

// Resolve resolves a SoundCloud URL (track, playlist, or user profile) into normalized Releases.
func (p *Provider) Resolve(ctx context.Context, u *url.URL) ([]*provider.Release, error) {
	resolved, err := ResolveURL(u.String())
	if err != nil {
		return nil, err
	}

	html, err := p.Client.FetchString(ctx, resolved.NormalizedURL, 5*1024*1024)
	if err != nil {
		return nil, fmt.Errorf("fetching SoundCloud URL %s: %w", resolved.NormalizedURL, err)
	}

	switch resolved.Type {
	case PageTrack:
		rel, err := ResolveTrack(ctx, p.Client, html, resolved.NormalizedURL)
		if err != nil {
			return nil, err
		}
		return []*provider.Release{rel}, nil

	case PagePlaylist:
		rel, err := ResolvePlaylist(ctx, p.Client, html, resolved.NormalizedURL)
		if err != nil {
			return nil, err
		}
		return []*provider.Release{rel}, nil

	case PageUser:
		trackURLs, err := ParseUserTracks(html, resolved.Artist)
		if err != nil {
			return nil, err
		}

		var releases []*provider.Release
		for _, trURL := range trackURLs {
			select {
			case <-ctx.Done():
				return releases, ctx.Err()
			default:
			}

			parsedTrURL, err := url.Parse(trURL)
			if err != nil {
				continue
			}

			subRels, err := p.Resolve(ctx, parsedTrURL)
			if err != nil {
				continue
			}
			releases = append(releases, subRels...)
		}

		if len(releases) == 0 {
			return nil, fmt.Errorf("%w: no accessible tracks found for artist %s", ErrSoundCloudNotFound, resolved.Artist)
		}
		return releases, nil

	default:
		return nil, fmt.Errorf("%w: unrecognized SoundCloud URL %s", ErrInvalidSoundCloudURL, u.String())
	}
}
