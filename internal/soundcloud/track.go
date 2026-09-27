package soundcloud

import (
	"context"
	"fmt"
	"strings"

	"github.com/lnoxsian/bandcamper/internal/provider"
)

// SelectBestTranscoding picks the highest quality direct stream, prioritizing progressive audio/mpeg (MP3).
func SelectBestTranscoding(media MediaInfo) (*Transcoding, error) {
	if len(media.Transcodings) == 0 {
		return nil, fmt.Errorf("%w: track has no media transcodings", ErrSoundCloudMediaUnavailable)
	}

	// 1. Prefer progressive MP3
	for _, t := range media.Transcodings {
		if strings.EqualFold(t.Format.Protocol, "progressive") && strings.Contains(strings.ToLower(t.Format.MimeType), "audio/mpeg") {
			return &t, nil
		}
	}

	// 2. Fall back to any progressive stream
	for _, t := range media.Transcodings {
		if strings.EqualFold(t.Format.Protocol, "progressive") {
			return &t, nil
		}
	}

	// 3. Fall back to direct HLS if progressive is unavailable
	for _, t := range media.Transcodings {
		if strings.EqualFold(t.Format.Protocol, "hls") {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("%w: no compatible media format found", ErrSoundCloudUnsupportedFormat)
}

// ResolveTrack processes a SoundCloud track page into a normalized Release.
func ResolveTrack(ctx context.Context, client *Client, pageHTML, trackURL string) (*provider.Release, error) {
	track, err := ParseTrack(pageHTML)
	if err != nil {
		return nil, fmt.Errorf("parsing SoundCloud track: %w", err)
	}

	hydration, _ := ExtractHydration(pageHTML)
	clientID := client.GetOrDiscoverClientID(ctx, pageHTML, hydration)

	bestTranscoding, err := SelectBestTranscoding(track.Media)
	if err != nil {
		return nil, fmt.Errorf("selecting media stream for track %q: %w", track.Title, err)
	}

	streamURL, err := client.ResolveMediaStreamURL(ctx, bestTranscoding.URL, clientID)
	if err != nil {
		return nil, fmt.Errorf("resolving stream URL for track %q: %w", track.Title, err)
	}

	if track.PermalinkURL == "" {
		track.PermalinkURL = trackURL
	}

	rel := track.ToRelease(streamURL)
	return rel, nil
}
