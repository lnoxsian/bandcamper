package downloader_test

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/lnoxsian/bandcamper/internal/bandcamp"
	"github.com/lnoxsian/bandcamper/internal/config"
	"github.com/lnoxsian/bandcamper/internal/downloader"
	"github.com/lnoxsian/bandcamper/internal/metadata"
	"github.com/lnoxsian/bandcamper/internal/storage"
)

func createTestJPEG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			img.Set(x, y, color.RGBA{R: 200, G: 50, B: 50, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, nil)
	return buf.Bytes()
}

func createTestMP3() []byte {
	frameHeader := []byte{0xFF, 0xFB, 0x90, 0x64}
	frame := append(frameHeader, make([]byte, 414)...)
	var buf bytes.Buffer
	for i := 0; i < 4; i++ {
		buf.Write(frame)
	}
	return buf.Bytes()
}

func TestEndToEndPipeline(t *testing.T) {
	mp3Data := createTestMP3()
	jpegData := createTestJPEG()

	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/album/cosmic-voyage":
			html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><title>Cosmic Voyage | Star Sailor</title></head>
<body>
<script>
var TralbumData = {
    "art_id": 9999,
    "artist": "Star Sailor",
    "current": {
        "title": "Cosmic Voyage",
        "release_date": "01 Jan 2024 00:00:00 GMT",
        "about": "An epic cosmic exploration album.",
        "type": "album"
    },
    "trackinfo": [
        {
            "id": 1,
            "track_num": 1,
            "title": "Starlight",
            "artist": "Star Sailor",
            "duration": 180.0,
            "file": {
                "mp3-128": "%s/stream/track1.mp3"
            },
            "lyrics": "Shining bright\nThrough the night"
        },
        {
            "id": 2,
            "track_num": 2,
            "title": "Supernova",
            "artist": "Star Sailor feat. Luna",
            "duration": 220.0,
            "file": {
                "mp3-128": "%s/stream/track2.mp3"
            },
            "lyrics": ""
        }
    ]
};
</script>
</body>
</html>`, serverURL, serverURL)
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(html))

		case "/stream/track1.mp3", "/stream/track2.mp3":
			w.Header().Set("Content-Type", "audio/mpeg")
			w.Write(mp3Data)

		case "/img/a9999_10.jpg":
			w.Header().Set("Content-Type", "image/jpeg")
			w.Write(jpegData)

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	tmpDir := t.TempDir()

	cfg := config.DefaultConfig()
	cfg.Output = tmpDir
	cfg.Workers = 2
	cfg.Playlist = "m3u"
	cfg.SaveArtwork = true
	cfg.EmbedArtwork = true
	cfg.EmbedLyrics = true
	cfg.EmbedTags = true

	client := bandcamp.NewClient(5*time.Second, "IntegrationTest")
	dl := downloader.New(cfg, client, nil)

	// Fetch page directly through client
	html, err := client.FetchString(context.Background(), serverURL+"/album/cosmic-voyage", 1024*1024)
	if err != nil {
		t.Fatalf("failed fetching mock album: %v", err)
	}

	release, err := bandcamp.ParseAlbum(html, "https://starsailor.bandcamp.com/album/cosmic-voyage")
	if err != nil {
		t.Fatalf("failed parsing mock album: %v", err)
	}

	// Update artwork URL to point to test server
	release.ArtworkURL = serverURL + "/img/a9999_10.jpg"

	res, err := dl.DownloadRelease(context.Background(), release)
	if err != nil {
		t.Fatalf("DownloadRelease failed: %v", err)
	}

	if len(res.DownloadedTracks) != 2 {
		t.Fatalf("expected 2 downloaded tracks, got %d", len(res.DownloadedTracks))
	}
	if len(res.FailedTracks) != 0 {
		t.Fatalf("unexpected track failures: %v", res.FailedTracks)
	}

	// Verify cover file saved
	coverFile := filepath.Join(res.OutputDir, "cover.jpg")
	if !storage.FileExists(coverFile) {
		t.Errorf("cover.jpg was not saved to %s", coverFile)
	}

	// Verify playlist file saved
	if !storage.FileExists(res.PlaylistPath) {
		t.Errorf("playlist file was not created at %s", res.PlaylistPath)
	}

	// Verify ID3 tags in track 1
	track1Path := res.DownloadedTracks[0]
	tag1, err := metadata.ReadMetadata(track1Path)
	if err != nil {
		t.Fatalf("failed reading tags from %s: %v", track1Path, err)
	}

	if tag1.Title != "Starlight" {
		t.Errorf("expected Title 'Starlight', got %q", tag1.Title)
	}
	if tag1.Artist != "Star Sailor" {
		t.Errorf("expected Artist 'Star Sailor', got %q", tag1.Artist)
	}
	if tag1.Album != "Cosmic Voyage" {
		t.Errorf("expected Album 'Cosmic Voyage', got %q", tag1.Album)
	}
	if tag1.TrackNumber != 1 {
		t.Errorf("expected TrackNumber 1, got %d", tag1.TrackNumber)
	}
	if tag1.Year != 2024 {
		t.Errorf("expected Year 2024, got %d", tag1.Year)
	}
	if tag1.Lyrics != "Shining bright\nThrough the night" {
		t.Errorf("expected lyrics, got %q", tag1.Lyrics)
	}
	if len(tag1.Artwork) == 0 {
		t.Errorf("expected embedded artwork in MP3, got 0 bytes")
	}

	// Verify track 2 artist
	track2Path := res.DownloadedTracks[1]
	tag2, err := metadata.ReadMetadata(track2Path)
	if err != nil {
		t.Fatalf("failed reading tags from %s: %v", track2Path, err)
	}
	if tag2.Artist != "Star Sailor feat. Luna" {
		t.Errorf("expected Track 2 artist 'Star Sailor feat. Luna', got %q", tag2.Artist)
	}
}
