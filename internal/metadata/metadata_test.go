package metadata

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lnoxsian/bandcamper/internal/bandcamp"
)

// createDummyMP3 writes a minimal valid MP3 stream (several silent MPEG-1 Layer 3 frames).
func createDummyMP3(t *testing.T, path string) {
	t.Helper()
	// Minimal MPEG 1 Layer 3 128kbps 44.1kHz stereo frame is 417/418 bytes.
	// Header: 0xFF, 0xFB, 0x90, 0x64 (or 0x00)
	frameHeader := []byte{0xFF, 0xFB, 0x90, 0x64}
	frame := append(frameHeader, make([]byte, 414)...)

	var buf bytes.Buffer
	for i := 0; i < 5; i++ {
		buf.Write(frame)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatalf("failed to write dummy mp3: %v", err)
	}
}

// createDummyJPEG creates a small in-memory JPEG image.
func createDummyJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for x := 0; x < 10; x++ {
		for y := 0; y < 10; y++ {
			img.Set(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("failed to encode dummy jpeg: %v", err)
	}
	return buf.Bytes()
}

func TestEmbedAndReadMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	mp3File := filepath.Join(tmpDir, "test.mp3")
	createDummyMP3(t, mp3File)

	jpegBytes := createDummyJPEG(t)

	meta := &Metadata{
		Title:       "Test Track",
		Artist:      "Test Artist",
		Album:       "Test Album",
		AlbumArtist: "Test Album Artist",
		TrackNumber: 3,
		TrackTotal:  10,
		Year:        2024,
		Genre:       "Electronic",
		Lyrics:      "Here are the test lyrics.\nLine two.",
		Comment:     "Test comment",
		Artwork:     jpegBytes,
		ArtworkMIME: "image/jpeg",
	}

	err := EmbedMetadata(mp3File, meta)
	if err != nil {
		t.Fatalf("EmbedMetadata failed: %v", err)
	}

	read, err := ReadMetadata(mp3File)
	if err != nil {
		t.Fatalf("ReadMetadata failed: %v", err)
	}

	if read.Title != meta.Title {
		t.Errorf("expected Title %q, got %q", meta.Title, read.Title)
	}
	if read.Artist != meta.Artist {
		t.Errorf("expected Artist %q, got %q", meta.Artist, read.Artist)
	}
	if read.Album != meta.Album {
		t.Errorf("expected Album %q, got %q", meta.Album, read.Album)
	}
	if read.AlbumArtist != meta.AlbumArtist {
		t.Errorf("expected AlbumArtist %q, got %q", meta.AlbumArtist, read.AlbumArtist)
	}
	if read.TrackNumber != meta.TrackNumber {
		t.Errorf("expected TrackNumber %d, got %d", meta.TrackNumber, read.TrackNumber)
	}
	if read.TrackTotal != meta.TrackTotal {
		t.Errorf("expected TrackTotal %d, got %d", meta.TrackTotal, read.TrackTotal)
	}
	if read.Year != meta.Year {
		t.Errorf("expected Year %d, got %d", meta.Year, read.Year)
	}
	if read.Genre != meta.Genre {
		t.Errorf("expected Genre %q, got %q", meta.Genre, read.Genre)
	}
	if read.Lyrics != meta.Lyrics {
		t.Errorf("expected Lyrics %q, got %q", meta.Lyrics, read.Lyrics)
	}
	if len(read.Artwork) == 0 {
		t.Errorf("expected artwork to be read back, got empty")
	}
}

func TestArtworkCache(t *testing.T) {
	cache := NewArtworkCache()
	url := "https://f4.bcbits.com/img/a123_10.jpg"

	_, ok := cache.Get(url)
	if ok {
		t.Fatalf("expected empty cache")
	}

	data := &ArtworkData{
		Data:     []byte{0x01, 0x02},
		MIMEType: "image/jpeg",
	}
	cache.Set(url, data)

	retrieved, ok := cache.Get(url)
	if !ok || retrieved == nil {
		t.Fatalf("expected cached data")
	}
	if len(retrieved.Data) != 2 {
		t.Errorf("expected 2 bytes, got %d", len(retrieved.Data))
	}
}

func TestNormalizeLyrics(t *testing.T) {
	raw := "Hello world<br>Second line<br />Third line<p>New paragraph</p>&amp; more"
	got := NormalizeLyrics(raw)
	expected := "Hello world\nSecond line\nThird line\nNew paragraph\n& more"
	if got != expected {
		t.Errorf("NormalizeLyrics() = %q, expected %q", got, expected)
	}
}

func TestFromReleaseAndTrack(t *testing.T) {
	rel := &bandcamp.Release{
		Artist:      "Band Name",
		Album:       "Debut Album",
		ReleaseDate: time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC),
		Genre:       "Rock",
	}
	tr := &bandcamp.Track{
		Number: 1,
		Title:  "Opening Song",
		Artist: "Band Name",
		Album:  "Debut Album",
		Lyrics: "Verse 1",
	}

	meta := FromReleaseAndTrack(rel, tr, 8, nil, "")
	if meta.TrackNumber != 1 || meta.TrackTotal != 8 {
		t.Errorf("expected track 1/8, got %d/%d", meta.TrackNumber, meta.TrackTotal)
	}
	if meta.Year != 2023 {
		t.Errorf("expected year 2023, got %d", meta.Year)
	}
}
