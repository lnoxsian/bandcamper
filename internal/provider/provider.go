package provider

import (
	"context"
	"net/url"
	"time"
)

// Provider abstracts a music source platform (Bandcamp, SoundCloud, etc.).
type Provider interface {
	Name() string
	CanHandle(u *url.URL) bool
	Resolve(ctx context.Context, u *url.URL) ([]*Release, error)
}

// Release represents a normalized album, EP, single, or playlist release across providers.
type Release struct {
	Provider    string
	URL         string
	Artist      string
	Album       string
	AlbumArtist string
	Description string
	Genre       string
	ReleaseDate time.Time
	ArtworkURL  string
	Tracks      []Track
}

// Track represents a single audio track within a release.
type Track struct {
	Number    int
	Title     string
	Artist    string
	Album     string
	Duration  time.Duration
	StreamURL string
	Lyrics    string
	Format    string
}

// MediaFormat specifies audio media container and MIME details.
type MediaFormat struct {
	Extension string
	MIME      string
}
