package soundcloud

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/lnoxsian/bandcamper/internal/provider"
)

// HydrationItem represents an individual block in window.__sc_hydration.
type HydrationItem struct {
	Hydratable string          `json:"hydratable"`
	Data       json.RawMessage `json:"data"`
}

// ApiClientData holds the client_id extracted from hydration.
type ApiClientData struct {
	ID string `json:"id"`
}

// SoundCloudUser represents the artist/user on SoundCloud.
type SoundCloudUser struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	PermalinkURL string `json:"permalink_url"`
	AvatarURL    string `json:"avatar_url"`
}

// TranscodingFormat describes the stream format and protocol.
type TranscodingFormat struct {
	Protocol string `json:"protocol"`
	MimeType string `json:"mime_type"`
}

// Transcoding represents an available stream encoding.
type Transcoding struct {
	URL      string            `json:"url"`
	Preset   string            `json:"preset"`
	Duration int64             `json:"duration"`
	Format   TranscodingFormat `json:"format"`
	Quality  string            `json:"quality"`
}

// MediaInfo contains all transcoding options for a track.
type MediaInfo struct {
	Transcodings []Transcoding `json:"transcodings"`
}

// SoundCloudTrack holds metadata and media streams for an individual track.
type SoundCloudTrack struct {
	ID           int64          `json:"id"`
	Title        string         `json:"title"`
	User         SoundCloudUser `json:"user"`
	Duration     int64          `json:"duration"` // Duration in milliseconds
	ArtworkURL   string         `json:"artwork_url"`
	Genre        string         `json:"genre"`
	Description  string         `json:"description"`
	CreatedAt    string         `json:"created_at"`
	PermalinkURL string         `json:"permalink_url"`
	Media        MediaInfo      `json:"media"`
}

// SoundCloudPlaylist represents a set or playlist containing tracks.
type SoundCloudPlaylist struct {
	ID           int64             `json:"id"`
	Title        string            `json:"title"`
	User         SoundCloudUser    `json:"user"`
	Description  string            `json:"description"`
	ArtworkURL   string            `json:"artwork_url"`
	CreatedAt    string            `json:"created_at"`
	PermalinkURL string            `json:"permalink_url"`
	Tracks       []SoundCloudTrack `json:"tracks"`
}

// MediaStreamResponse represents the payload returned by SoundCloud media stream endpoints.
type MediaStreamResponse struct {
	URL string `json:"url"`
}

// HighResArtwork converts SoundCloud thumbnail URLs (e.g. -large.jpg) to high-res (-t500x500.jpg).
func HighResArtwork(url string) string {
	if url == "" {
		return ""
	}
	if strings.Contains(url, "-large.") {
		return strings.Replace(url, "-large.", "-t500x500.", 1)
	}
	return url
}

// ParseSoundCloudTime parses ISO 8601 or RFC3339 timestamps.
func ParseSoundCloudTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006/01/02 15:04:05 +0000",
		"2006-01-02",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// ToRelease converts a single SoundCloudTrack into a normalized provider.Release.
func (t *SoundCloudTrack) ToRelease(streamURL string) *provider.Release {
	artist := strings.TrimSpace(t.User.Username)
	if artist == "" {
		artist = "SoundCloud Artist"
	}
	title := strings.TrimSpace(t.Title)
	if title == "" {
		title = "Untitled"
	}

	track := provider.Track{
		Number:    1,
		Title:     title,
		Artist:    artist,
		Album:     title,
		Duration:  time.Duration(t.Duration) * time.Millisecond,
		StreamURL: streamURL,
		Format:    "mp3",
	}

	return &provider.Release{
		Provider:    "soundcloud",
		URL:         t.PermalinkURL,
		Artist:      artist,
		Album:       title,
		AlbumArtist: artist,
		Description: t.Description,
		Genre:       t.Genre,
		ReleaseDate: ParseSoundCloudTime(t.CreatedAt),
		ArtworkURL:  HighResArtwork(t.ArtworkURL),
		Tracks:      []provider.Track{track},
	}
}

// ToRelease converts a SoundCloudPlaylist into a normalized provider.Release with multiple tracks.
func (p *SoundCloudPlaylist) ToRelease(trackStreamURLs []string) *provider.Release {
	artist := strings.TrimSpace(p.User.Username)
	if artist == "" {
		artist = "SoundCloud Creator"
	}
	album := strings.TrimSpace(p.Title)
	if album == "" {
		album = "SoundCloud Playlist"
	}

	var tracks []provider.Track
	for i, tr := range p.Tracks {
		streamURL := ""
		if i < len(trackStreamURLs) {
			streamURL = trackStreamURLs[i]
		}

		trArtist := strings.TrimSpace(tr.User.Username)
		if trArtist == "" {
			trArtist = artist
		}
		trTitle := strings.TrimSpace(tr.Title)
		if trTitle == "" {
			trTitle = "Untitled"
		}

		tracks = append(tracks, provider.Track{
			Number:    i + 1,
			Title:     trTitle,
			Artist:    trArtist,
			Album:     album,
			Duration:  time.Duration(tr.Duration) * time.Millisecond,
			StreamURL: streamURL,
			Format:    "mp3",
		})
	}

	artworkURL := p.ArtworkURL
	if artworkURL == "" && len(p.Tracks) > 0 {
		artworkURL = p.Tracks[0].ArtworkURL
	}

	return &provider.Release{
		Provider:    "soundcloud",
		URL:         p.PermalinkURL,
		Artist:      artist,
		Album:       album,
		AlbumArtist: artist,
		Description: p.Description,
		ReleaseDate: ParseSoundCloudTime(p.CreatedAt),
		ArtworkURL:  HighResArtwork(artworkURL),
		Tracks:      tracks,
	}
}
