package bandcamp

import (
	"errors"
	"testing"
)

func TestResolveURL(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedType  PageType
		expectedNorm  string
		expectedErrIs error
	}{
		{
			name:         "Artist root URL",
			input:        "https://artist.bandcamp.com/",
			expectedType: PageArtist,
			expectedNorm: "https://artist.bandcamp.com",
		},
		{
			name:         "Artist root URL without scheme",
			input:        "artist.bandcamp.com",
			expectedType: PageArtist,
			expectedNorm: "https://artist.bandcamp.com",
		},
		{
			name:         "Artist music page",
			input:        "https://artist.bandcamp.com/music",
			expectedType: PageArtist,
			expectedNorm: "https://artist.bandcamp.com/music",
		},
		{
			name:         "Album URL with query params",
			input:        "https://artist.bandcamp.com/album/great-album?from=search&action=buy",
			expectedType: PageAlbum,
			expectedNorm: "https://artist.bandcamp.com/album/great-album",
		},
		{
			name:         "Track URL with trailing slash and fragment",
			input:        "https://artist.bandcamp.com/track/hit-song/#lyrics",
			expectedType: PageTrack,
			expectedNorm: "https://artist.bandcamp.com/track/hit-song",
		},
		{
			name:          "Empty URL",
			input:         "   ",
			expectedErrIs: ErrInvalidURL,
		},
		{
			name:          "Invalid domain",
			input:         "https://youtube.com/watch?v=12345",
			expectedErrIs: ErrInvalidURL,
		},
		{
			name:          "Unsupported bandcamp path",
			input:         "https://artist.bandcamp.com/feed/updates",
			expectedErrIs: ErrUnsupportedPage,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := ResolveURL(tc.input)
			if tc.expectedErrIs != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tc.expectedErrIs)
				}
				if !errors.Is(err, tc.expectedErrIs) {
					t.Fatalf("expected error wrapping %v, got %v", tc.expectedErrIs, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.Type != tc.expectedType {
				t.Errorf("expected type %v, got %v", tc.expectedType, res.Type)
			}
			if res.NormalizedURL != tc.expectedNorm {
				t.Errorf("expected normalized %q, got %q", tc.expectedNorm, res.NormalizedURL)
			}
		})
	}
}
