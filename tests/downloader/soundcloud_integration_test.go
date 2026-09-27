package downloader_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/lnoxsian/bandcamper/internal/bandcamp"
	"github.com/lnoxsian/bandcamper/internal/config"
	"github.com/lnoxsian/bandcamper/internal/downloader"
	"github.com/lnoxsian/bandcamper/internal/metadata"
	"github.com/lnoxsian/bandcamper/internal/provider"
	"github.com/lnoxsian/bandcamper/internal/soundcloud"
	"github.com/lnoxsian/bandcamper/internal/storage"
)

type mockRoundTripper struct {
	targetHost string
	transport  http.RoundTripper
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.URL.Scheme = "http"
	req2.URL.Host = m.targetHost
	return m.transport.RoundTrip(req2)
}

func TestSoundCloudEndToEndPipeline(t *testing.T) {
	mp3Data := createTestMP3()
	jpegData := createTestJPEG()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/starsailor/starsound":
			html := `<!DOCTYPE html>
<html>
<head><title>Star Sound by Star Sailor | Listen on SoundCloud</title></head>
<body>
<script>window.__sc_hydration = [
  {
    "hydratable": "apiClient",
    "data": { "id": "test_client_id_123456" }
  },
  {
    "hydratable": "sound",
    "data": {
      "id": 1001,
      "title": "Star Sound",
      "duration": 180000,
      "genre": "Ambient",
      "description": "A soothing ambient journey.",
      "created_at": "2024-04-10T00:00:00Z",
      "permalink_url": "https://soundcloud.com/starsailor/starsound",
      "artwork_url": "https://soundcloud.com/img/artwork.jpg",
      "user": {
        "id": 555,
        "username": "Star Sailor",
        "permalink_url": "https://soundcloud.com/starsailor"
      },
      "media": {
        "transcodings": [
          {
            "url": "https://soundcloud.com/transcoding/1001",
            "preset": "mp3_128",
            "duration": 180000,
            "format": {
              "protocol": "progressive",
              "mime_type": "audio/mpeg"
            },
            "quality": "sq"
          }
        ]
      }
    }
  }
];</script>
</body>
</html>`
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(html))

		case "/transcoding/1001":
			// API response containing the stream URL
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"url": "https://soundcloud.com/stream/1001.mp3"}`))

		case "/stream/1001.mp3":
			w.Header().Set("Content-Type", "audio/mpeg")
			w.Write(mp3Data)

		case "/img/artwork.jpg":
			w.Header().Set("Content-Type", "image/jpeg")
			w.Write(jpegData)

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	cfg := config.DefaultConfig()
	cfg.Output = tmpDir
	cfg.Directory = "{provider}/{artist}/{album}"
	cfg.Filename = "{tracknumber} - {title}"
	cfg.SaveArtwork = true
	cfg.EmbedArtwork = true
	cfg.EmbedTags = true

	targetHost := server.Listener.Addr().String()
	transport := &mockRoundTripper{
		targetHost: targetHost,
		transport:  http.DefaultTransport,
	}

	scClient := soundcloud.NewClient(5*time.Second, "TestBot")
	scClient.HTTPClient.Transport = transport
	scProvider := soundcloud.NewProvider(scClient)

	bcClient := bandcamp.NewClient(5*time.Second, "TestBot")
	bcClient.HTTPClient.Transport = transport

	// Resolve the track
	parsedURL, err := url.Parse("https://soundcloud.com/starsailor/starsound")
	if err != nil {
		t.Fatalf("failed to parse test URL: %v", err)
	}

	releases, err := scProvider.Resolve(context.Background(), parsedURL)
	if err != nil {
		t.Fatalf("scProvider.Resolve failed: %v", err)
	}
	if len(releases) == 0 {
		t.Fatalf("expected at least 1 release, got 0")
	}
	release := releases[0]

	if release.Provider != "soundcloud" {
		t.Errorf("expected Provider 'soundcloud', got %q", release.Provider)
	}
	if release.Artist != "Star Sailor" {
		t.Errorf("expected Artist 'Star Sailor', got %q", release.Artist)
	}
	if len(release.Tracks) != 1 {
		t.Fatalf("expected 1 track, got %d", len(release.Tracks))
	}

	// Verify registry integration
	registry := provider.NewRegistry()
	registry.Register(scProvider)

	dl := downloader.NewWithRegistry(cfg, registry, bcClient, nil)
	res, err := dl.DownloadRelease(context.Background(), release)
	if err != nil {
		t.Fatalf("DownloadRelease failed: %v", err)
	}

	if len(res.DownloadedTracks) != 1 {
		t.Fatalf("expected 1 downloaded track, got %d", len(res.DownloadedTracks))
	}
	if len(res.FailedTracks) != 0 {
		t.Fatalf("unexpected track failures: %v", res.FailedTracks)
	}

	// Verify folder format with {provider} placeholder
	expectedRelDir := filepath.Join("SoundCloud", "Star Sailor", "Star Sound")
	if !filepath.IsAbs(res.OutputDir) {
		t.Errorf("expected output dir to be absolute: %s", res.OutputDir)
	}
	if filepath.Base(filepath.Dir(res.OutputDir)) != "Star Sailor" {
		t.Errorf("output dir structure mismatch: %s", res.OutputDir)
	}

	// Verify cover file saved
	coverFile := filepath.Join(res.OutputDir, "cover.jpg")
	if !storage.FileExists(coverFile) {
		t.Errorf("cover.jpg was not saved to %s", coverFile)
	}

	// Verify ID3 tags
	trackPath := res.DownloadedTracks[0]
	tags, err := metadata.ReadMetadata(trackPath)
	if err != nil {
		t.Fatalf("failed reading tags from %s: %v", trackPath, err)
	}

	if tags.Title != "Star Sound" {
		t.Errorf("expected Title 'Star Sound', got %q", tags.Title)
	}
	if tags.Artist != "Star Sailor" {
		t.Errorf("expected Artist 'Star Sailor', got %q", tags.Artist)
	}
	if tags.Genre != "Ambient" {
		t.Errorf("expected Genre 'Ambient', got %q", tags.Genre)
	}
	if tags.Year != 2024 {
		t.Errorf("expected Year 2024, got %d", tags.Year)
	}
	if len(tags.Artwork) == 0 {
		t.Errorf("expected embedded artwork in MP3, got 0 bytes")
	}
	_ = expectedRelDir
}

func TestSoundCloudPlaylistPipeline(t *testing.T) {
	mp3Data := createTestMP3()
	jpegData := createTestJPEG()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/starsailor/sets/ambient-tales":
			html := `<!DOCTYPE html>
<html>
<head><title>Ambient Tales by Star Sailor | SoundCloud</title></head>
<body>
<script>window.__sc_hydration = [
  {
    "hydratable": "apiClient",
    "data": { "id": "test_client_id_playlist" }
  },
  {
    "hydratable": "playlist",
    "data": {
      "id": 8888,
      "title": "Ambient Tales",
      "description": "Collection of chill tracks",
      "artwork_url": "https://soundcloud.com/img/playlist_art.jpg",
      "user": {
        "id": 555,
        "username": "Star Sailor",
        "permalink_url": "https://soundcloud.com/starsailor"
      },
      "tracks": [
        {
          "id": 8801,
          "title": "Morning Mist",
          "duration": 120000,
          "genre": "Ambient",
          "user": { "username": "Star Sailor" },
          "media": {
            "transcodings": [
              {
                "url": "https://soundcloud.com/transcoding/8801",
                "format": { "protocol": "progressive", "mime_type": "audio/mpeg" }
              }
            ]
          }
        },
        {
          "id": 8802,
          "title": "Evening Calm",
          "duration": 140000,
          "genre": "Ambient",
          "user": { "username": "Star Sailor" },
          "media": {
            "transcodings": [
              {
                "url": "https://soundcloud.com/transcoding/8802",
                "format": { "protocol": "progressive", "mime_type": "audio/mpeg" }
              }
            ]
          }
        }
      ]
    }
  }
];</script>
</body>
</html>`
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(html))

		case "/transcoding/8801":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"url": "https://soundcloud.com/stream/8801.mp3"}`))

		case "/transcoding/8802":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"url": "https://soundcloud.com/stream/8802.mp3"}`))

		case "/stream/8801.mp3", "/stream/8802.mp3":
			w.Header().Set("Content-Type", "audio/mpeg")
			w.Write(mp3Data)

		case "/img/playlist_art.jpg":
			w.Header().Set("Content-Type", "image/jpeg")
			w.Write(jpegData)

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	cfg := config.DefaultConfig()
	cfg.Output = tmpDir
	cfg.Playlist = "m3u"
	cfg.SaveArtwork = true
	cfg.EmbedArtwork = true
	cfg.EmbedTags = true

	targetHost := server.Listener.Addr().String()
	transport := &mockRoundTripper{
		targetHost: targetHost,
		transport:  http.DefaultTransport,
	}

	scClient := soundcloud.NewClient(5*time.Second, "TestBot")
	scClient.HTTPClient.Transport = transport
	scProvider := soundcloud.NewProvider(scClient)

	bcClient := bandcamp.NewClient(5*time.Second, "TestBot")
	bcClient.HTTPClient.Transport = transport

	registry := provider.NewRegistry()
	registry.Register(scProvider)

	dl := downloader.NewWithRegistry(cfg, registry, bcClient, nil)

	results, err := dl.DownloadURL(context.Background(), "https://soundcloud.com/starsailor/sets/ambient-tales")
	if err != nil {
		t.Fatalf("DownloadURL failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	res := results[0]
	if len(res.DownloadedTracks) != 2 {
		t.Fatalf("expected 2 downloaded tracks, got %d", len(res.DownloadedTracks))
	}
	if len(res.FailedTracks) != 0 {
		t.Fatalf("unexpected track failures: %v", res.FailedTracks)
	}

	// Verify playlist file
	if !storage.FileExists(res.PlaylistPath) {
		t.Errorf("expected playlist file at %s", res.PlaylistPath)
	}

	// Verify ID3 tags
	tag1, err := metadata.ReadMetadata(res.DownloadedTracks[0])
	if err != nil {
		t.Fatalf("failed to read metadata from track 1: %v", err)
	}
	if tag1.Title != "Morning Mist" {
		t.Errorf("expected Title 'Morning Mist', got %q", tag1.Title)
	}
	if tag1.Album != "Ambient Tales" {
		t.Errorf("expected Album 'Ambient Tales', got %q", tag1.Album)
	}
	if tag1.TrackNumber != 1 {
		t.Errorf("expected TrackNumber 1, got %d", tag1.TrackNumber)
	}

	tag2, err := metadata.ReadMetadata(res.DownloadedTracks[1])
	if err != nil {
		t.Fatalf("failed to read metadata from track 2: %v", err)
	}
	if tag2.Title != "Evening Calm" {
		t.Errorf("expected Title 'Evening Calm', got %q", tag2.Title)
	}
	if tag2.TrackNumber != 2 {
		t.Errorf("expected TrackNumber 2, got %d", tag2.TrackNumber)
	}
}

