package soundcloud

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseTrackFixture(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "tests", "fixtures", "soundcloud", "track.html")
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed reading track fixture: %v", err)
	}

	track, err := ParseTrack(string(content))
	if err != nil {
		t.Fatalf("unexpected error parsing track fixture: %v", err)
	}

	if track.Title != "Cosmic Voyage" {
		t.Errorf("expected Title 'Cosmic Voyage', got %q", track.Title)
	}
	if track.User.Username != "Star Sailor" {
		t.Errorf("expected Artist 'Star Sailor', got %q", track.User.Username)
	}
	if track.Genre != "Electronic" {
		t.Errorf("expected Genre 'Electronic', got %q", track.Genre)
	}
	if track.Duration != 210000 {
		t.Errorf("expected Duration 210000, got %d", track.Duration)
	}
	if len(track.Media.Transcodings) != 2 {
		t.Errorf("expected 2 transcodings, got %d", len(track.Media.Transcodings))
	}

	hydration, _ := ExtractHydration(string(content))
	clientID := ExtractClientID(string(content), hydration)
	if clientID != "mock_client_id_1234567890abcdef" {
		t.Errorf("expected client_id 'mock_client_id_1234567890abcdef', got %q", clientID)
	}

	rel := track.ToRelease("http://example.com/audio.mp3")
	if rel.Artist != "Star Sailor" {
		t.Errorf("expected Release Artist 'Star Sailor', got %q", rel.Artist)
	}
	if rel.Album != "Cosmic Voyage" {
		t.Errorf("expected Release Album 'Cosmic Voyage', got %q", rel.Album)
	}
	if len(rel.Tracks) != 1 {
		t.Fatalf("expected 1 track in Release, got %d", len(rel.Tracks))
	}
	if rel.Tracks[0].Title != "Cosmic Voyage" {
		t.Errorf("expected Track Title 'Cosmic Voyage', got %q", rel.Tracks[0].Title)
	}
	if !strings.Contains(rel.ArtworkURL, "-t500x500.jpg") {
		t.Errorf("expected high-res artwork URL, got %q", rel.ArtworkURL)
	}
}

func TestParsePlaylistFixture(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "tests", "fixtures", "soundcloud", "playlist.html")
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed reading playlist fixture: %v", err)
	}

	playlist, err := ParsePlaylist(string(content))
	if err != nil {
		t.Fatalf("unexpected error parsing playlist fixture: %v", err)
	}

	if playlist.Title != "Stellar Odyssey" {
		t.Errorf("expected Playlist Title 'Stellar Odyssey', got %q", playlist.Title)
	}
	if playlist.User.Username != "Star Sailor" {
		t.Errorf("expected User 'Star Sailor', got %q", playlist.User.Username)
	}
	if len(playlist.Tracks) != 2 {
		t.Fatalf("expected 2 tracks, got %d", len(playlist.Tracks))
	}

	streamURLs := []string{"http://example.com/t1.mp3", "http://example.com/t2.mp3"}
	rel := playlist.ToRelease(streamURLs)
	if rel.Album != "Stellar Odyssey" {
		t.Errorf("expected Album 'Stellar Odyssey', got %q", rel.Album)
	}
	if len(rel.Tracks) != 2 {
		t.Fatalf("expected 2 tracks in Release, got %d", len(rel.Tracks))
	}
	if rel.Tracks[0].Title != "Supernova" || rel.Tracks[0].Number != 1 {
		t.Errorf("track 1 mismatch: %+v", rel.Tracks[0])
	}
	if rel.Tracks[1].Title != "Nebula Dreams" || rel.Tracks[1].Number != 2 {
		t.Errorf("track 2 mismatch: %+v", rel.Tracks[1])
	}
}

func TestParseUnicodeAndMissingFields(t *testing.T) {
	html := `<!DOCTYPE html><html><body>
<script>window.__sc_hydration = [
  {
    "hydratable": "sound",
    "data": {
      "id": 123,
      "title": "東京フラッシュ 🎵",
      "user": {
        "username": "ヴァウンディ Vaundy"
      },
      "media": {
        "transcodings": [
          {
            "url": "http://api.example.com/stream",
            "format": {"protocol": "progressive", "mime_type": "audio/mpeg"}
          }
        ]
      }
    }
  }
];</script></body></html>`

	track, err := ParseTrack(html)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if track.Title != "東京フラッシュ 🎵" {
		t.Errorf("expected Unicode title, got %q", track.Title)
	}
	if track.User.Username != "ヴァウンディ Vaundy" {
		t.Errorf("expected Unicode username, got %q", track.User.Username)
	}

	rel := track.ToRelease("http://stream.example.com/audio.mp3")
	if rel.Genre != "" {
		t.Errorf("expected empty genre, got %q", rel.Genre)
	}
	if rel.ArtworkURL != "" {
		t.Errorf("expected empty artwork URL, got %q", rel.ArtworkURL)
	}
	if rel.Description != "" {
		t.Errorf("expected empty description, got %q", rel.Description)
	}
}

func TestParseUserTracks(t *testing.T) {
	html := `<!DOCTYPE html><html><body>
<a href="/starsailor/cosmic-voyage">Cosmic</a>
<a href="/starsailor/sets/stellar-odyssey">Set</a>
<a href="/starsailor/supernova">Supernova</a>
<a href="/starsailor/likes">Likes</a>
</body></html>`

	urls, err := ParseUserTracks(html, "starsailor")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(urls) != 2 {
		t.Fatalf("expected 2 track URLs, got %d: %v", len(urls), urls)
	}
	if urls[0] != "https://soundcloud.com/starsailor/cosmic-voyage" {
		t.Errorf("unexpected URL 0: %q", urls[0])
	}
	if urls[1] != "https://soundcloud.com/starsailor/supernova" {
		t.Errorf("unexpected URL 1: %q", urls[1])
	}
}
