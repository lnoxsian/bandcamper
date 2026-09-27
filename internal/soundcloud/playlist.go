package soundcloud

import (
	"context"
	"fmt"

	"github.com/lnoxsian/bandcamper/internal/provider"
)

// ResolvePlaylist processes a SoundCloud set / playlist page into a normalized multi-track Release.
func ResolvePlaylist(ctx context.Context, client *Client, pageHTML, playlistURL string) (*provider.Release, error) {
	playlist, err := ParsePlaylist(pageHTML)
	if err != nil {
		return nil, fmt.Errorf("parsing SoundCloud playlist: %w", err)
	}

	hydration, _ := ExtractHydration(pageHTML)
	clientID := client.GetOrDiscoverClientID(ctx, pageHTML, hydration)

	var streamURLs []string
	var validTracks []SoundCloudTrack

	for _, tr := range playlist.Tracks {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		bestTranscoding, err := SelectBestTranscoding(tr.Media)
		if err != nil {
			// Skip tracks that lack streamable representations (e.g. preview-only)
			continue
		}

		streamURL, err := client.ResolveMediaStreamURL(ctx, bestTranscoding.URL, clientID)
		if err != nil {
			continue
		}

		streamURLs = append(streamURLs, streamURL)
		validTracks = append(validTracks, tr)
	}

	if len(validTracks) == 0 {
		return nil, fmt.Errorf("%w: playlist %q contains no streamable tracks", ErrSoundCloudMediaUnavailable, playlist.Title)
	}

	playlist.Tracks = validTracks
	if playlist.PermalinkURL == "" {
		playlist.PermalinkURL = playlistURL
	}

	return playlist.ToRelease(streamURLs), nil
}
