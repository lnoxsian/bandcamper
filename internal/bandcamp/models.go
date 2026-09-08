package bandcamp

import (
	"errors"
	"time"
)

// Common errors.
var (
	ErrInvalidURL      = errors.New("invalid Bandcamp URL")
	ErrUnsupportedPage = errors.New("unsupported Bandcamp page type")
	ErrParseFailed     = errors.New("failed to parse Bandcamp page")
	ErrDownloadFailed  = errors.New("failed to download audio")
	ErrArtworkFailed   = errors.New("failed to process artwork")
	ErrMetadataFailed  = errors.New("failed to process metadata")
	ErrNoStreamURL     = errors.New("no audio stream URL available for track")
)

// Release represents an album, EP, or single release on Bandcamp.
type Release struct {
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

// Track represents an individual audio track within a release.
type Track struct {
	Number    int
	Title     string
	Artist    string
	Album     string
	Duration  time.Duration
	StreamURL string
	Lyrics    string
}
