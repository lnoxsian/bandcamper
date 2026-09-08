package downloader

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lnoxsian/bandcamper/internal/bandcamp"
	"github.com/lnoxsian/bandcamper/internal/config"
	"github.com/lnoxsian/bandcamper/internal/storage"
)

// createDummyMP3Bytes returns minimal valid MP3 frame bytes
func createDummyMP3Bytes() []byte {
	frameHeader := []byte{0xFF, 0xFB, 0x90, 0x64}
	frame := append(frameHeader, make([]byte, 414)...)
	var buf bytes.Buffer
	for i := 0; i < 3; i++ {
		buf.Write(frame)
	}
	return buf.Bytes()
}

func TestDownloaderSuccessAndPlaylist(t *testing.T) {
	mp3Bytes := createDummyMP3Bytes()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mpeg")
		w.WriteHeader(http.StatusOK)
		w.Write(mp3Bytes)
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	cfg := config.DefaultConfig()
	cfg.Output = tmpDir
	cfg.Directory = "{artist}/{album}"
	cfg.Filename = "{tracknumber} - {title}.mp3"
	cfg.Playlist = "m3u"
	cfg.EmbedArtwork = false
	cfg.SaveArtwork = false

	client := bandcamp.NewClient(5*time.Second, "TestAgent")
	dl := New(cfg, client, nil)

	release := &bandcamp.Release{
		Artist: "Test Band",
		Album:  "First Record",
		Tracks: []bandcamp.Track{
			{
				Number:    1,
				Title:     "Track One",
				Artist:    "Test Band",
				Album:     "First Record",
				StreamURL: server.URL + "/track1.mp3",
			},
			{
				Number:    2,
				Title:     "Track Two",
				Artist:    "Test Band",
				Album:     "First Record",
				StreamURL: server.URL + "/track2.mp3",
			},
		},
	}

	res, err := dl.DownloadRelease(context.Background(), release)
	if err != nil {
		t.Fatalf("DownloadRelease failed: %v", err)
	}

	if len(res.DownloadedTracks) != 2 {
		t.Fatalf("expected 2 downloaded tracks, got %d", len(res.DownloadedTracks))
	}

	for _, p := range res.DownloadedTracks {
		if !storage.FileExists(p) {
			t.Errorf("expected track file to exist: %s", p)
		}
	}

	if res.PlaylistPath == "" || !storage.FileExists(res.PlaylistPath) {
		t.Errorf("expected playlist file to exist, got %s", res.PlaylistPath)
	}
}

func TestDownloaderSkipExisting(t *testing.T) {
	mp3Bytes := createDummyMP3Bytes()

	var reqCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&reqCount, 1)
		w.WriteHeader(http.StatusOK)
		w.Write(mp3Bytes)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Output = tmpDir
	cfg.SkipExisting = true
	cfg.Overwrite = false
	cfg.EmbedArtwork = false
	cfg.SaveArtwork = false

	client := bandcamp.NewClient(5*time.Second, "TestAgent")
	dl := New(cfg, client, nil)

	release := &bandcamp.Release{
		Artist: "Artist",
		Album:  "Album",
		Tracks: []bandcamp.Track{
			{
				Number:    1,
				Title:     "Song",
				StreamURL: server.URL + "/song.mp3",
			},
		},
	}

	// First download
	res1, err := dl.DownloadRelease(context.Background(), release)
	if err != nil {
		t.Fatalf("first download failed: %v", err)
	}
	if len(res1.DownloadedTracks) != 1 {
		t.Fatalf("expected 1 downloaded track, got %d", len(res1.DownloadedTracks))
	}

	// Second download should skip
	res2, err := dl.DownloadRelease(context.Background(), release)
	if err != nil {
		t.Fatalf("second download failed: %v", err)
	}
	if len(res2.SkippedTracks) != 1 {
		t.Fatalf("expected 1 skipped track, got %d", len(res2.SkippedTracks))
	}
	if atomic.LoadInt32(&reqCount) != 1 {
		t.Errorf("expected 1 HTTP request made, got %d", atomic.LoadInt32(&reqCount))
	}
}

func TestDownloaderRetries(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		att := atomic.AddInt32(&attempts, 1)
		if att < 3 {
			// Fail first two attempts with 503 Service Unavailable
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("busy"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(createDummyMP3Bytes())
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Output = tmpDir
	cfg.RetryCount = 3
	cfg.RetryDelay = 50 * time.Millisecond
	cfg.EmbedArtwork = false
	cfg.SaveArtwork = false

	client := bandcamp.NewClient(5*time.Second, "TestAgent")
	dl := New(cfg, client, nil)

	release := &bandcamp.Release{
		Artist: "Artist",
		Album:  "Album",
		Tracks: []bandcamp.Track{
			{
				Number:    1,
				Title:     "RetrySong",
				StreamURL: server.URL + "/stream.mp3",
			},
		},
	}

	res, err := dl.DownloadRelease(context.Background(), release)
	if err != nil {
		t.Fatalf("expected success after retries: %v", err)
	}
	if len(res.DownloadedTracks) != 1 {
		t.Errorf("expected 1 downloaded track, got %d", len(res.DownloadedTracks))
	}
	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected 3 attempts, got %d", atomic.LoadInt32(&attempts))
	}
}

func TestDownloaderCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Output = tmpDir
	cfg.EmbedArtwork = false
	cfg.SaveArtwork = false

	client := bandcamp.NewClient(5*time.Second, "TestAgent")
	dl := New(cfg, client, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	release := &bandcamp.Release{
		Artist: "Artist",
		Album:  "Album",
		Tracks: []bandcamp.Track{
			{
				Number:    1,
				Title:     "SlowSong",
				StreamURL: server.URL + "/slow.mp3",
			},
		},
	}

	res, err := dl.DownloadRelease(ctx, release)
	if err == nil && len(res.FailedTracks) == 0 {
		t.Fatalf("expected cancellation error, got success")
	}

	// Verify no stray partial files left
	partFile := filepath.Join(tmpDir, "Artist", "Album", "01 - SlowSong.mp3.part")
	if storage.FileExists(partFile) {
		t.Errorf("part file should have been cleaned up after cancellation")
	}
}
