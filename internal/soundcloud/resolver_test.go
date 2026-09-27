package soundcloud

import (
	"testing"
)

func TestResolveURL(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		wantType   PageType
		wantArtist string
		wantSlug   string
		wantErr    bool
	}{
		{
			name:       "Track URL",
			url:        "https://soundcloud.com/starsailor/cosmic-voyage",
			wantType:   PageTrack,
			wantArtist: "starsailor",
			wantSlug:   "cosmic-voyage",
			wantErr:    false,
		},
		{
			name:       "Track URL with query and fragment",
			url:        "https://soundcloud.com/starsailor/cosmic-voyage?in=starsailor/sets/stellar#t=1:00",
			wantType:   PageTrack,
			wantArtist: "starsailor",
			wantSlug:   "cosmic-voyage",
			wantErr:    false,
		},
		{
			name:       "Mobile Track URL",
			url:        "https://m.soundcloud.com/starsailor/cosmic-voyage",
			wantType:   PageTrack,
			wantArtist: "starsailor",
			wantSlug:   "cosmic-voyage",
			wantErr:    false,
		},
		{
			name:       "Playlist URL",
			url:        "https://soundcloud.com/starsailor/sets/stellar-odyssey",
			wantType:   PagePlaylist,
			wantArtist: "starsailor",
			wantSlug:   "stellar-odyssey",
			wantErr:    false,
		},
		{
			name:       "Artist Root URL",
			url:        "https://soundcloud.com/starsailor",
			wantType:   PageUser,
			wantArtist: "starsailor",
			wantSlug:   "",
			wantErr:    false,
		},
		{
			name:       "Artist Tracks URL",
			url:        "https://soundcloud.com/starsailor/tracks",
			wantType:   PageUser,
			wantArtist: "starsailor",
			wantSlug:   "",
			wantErr:    false,
		},
		{
			name:       "Artist Sets URL",
			url:        "https://soundcloud.com/starsailor/sets",
			wantType:   PageUser,
			wantArtist: "starsailor",
			wantSlug:   "",
			wantErr:    false,
		},
		{
			name:    "Reserved Path URL",
			url:     "https://soundcloud.com/discover",
			wantErr: true,
		},
		{
			name:    "Invalid Domain",
			url:     "https://example.com/artist/track",
			wantErr: true,
		},
		{
			name:    "Empty URL",
			url:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ResolveURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", got.Type, tt.wantType)
			}
			if got.Artist != tt.wantArtist {
				t.Errorf("Artist = %q, want %q", got.Artist, tt.wantArtist)
			}
			if got.Slug != tt.wantSlug {
				t.Errorf("Slug = %q, want %q", got.Slug, tt.wantSlug)
			}
		})
	}
}
