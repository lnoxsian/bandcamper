package bandcamp

import (
	"errors"

	"github.com/lnoxsian/bandcamper/internal/provider"
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

// Release aliases the normalized provider.Release model.
type Release = provider.Release

// Track aliases the normalized provider.Track model.
type Track = provider.Track
