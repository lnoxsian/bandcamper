package playlist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlaylists(t *testing.T) {
	tmpDir := t.TempDir()
	tracks := []string{
		filepath.Join(tmpDir, "01 - Track One.mp3"),
		filepath.Join(tmpDir, "02 - Track Two.mp3"),
	}

	formats := []string{"m3u", "pls", "wpl", "zpl"}
	for _, fmtName := range formats {
		t.Run(fmtName, func(t *testing.T) {
			writer, err := GetWriter(fmtName)
			if err != nil {
				t.Fatalf("unexpected error getting writer for %s: %v", fmtName, err)
			}

			ext := FormatExtension(fmtName)
			outPath := filepath.Join(tmpDir, "playlist"+ext)

			err = writer.Write(outPath, tracks)
			if err != nil {
				t.Fatalf("failed to write %s playlist: %v", fmtName, err)
			}

			data, err := os.ReadFile(outPath)
			if err != nil {
				t.Fatalf("failed reading written playlist: %v", err)
			}

			content := string(data)
			if !strings.Contains(content, "01 - Track One.mp3") {
				t.Errorf("%s missing track 1: %s", fmtName, content)
			}
			if !strings.Contains(content, "02 - Track Two.mp3") {
				t.Errorf("%s missing track 2: %s", fmtName, content)
			}
		})
	}
}

func TestMakeRelative(t *testing.T) {
	playlistPath := "/music/artist/album/playlist.m3u"
	tracks := []string{
		"/music/artist/album/01 - Song.mp3",
		"/music/artist/album/02 - Song.mp3",
	}

	rel := MakeRelative(playlistPath, tracks)
	if len(rel) != 2 {
		t.Fatalf("expected 2 tracks, got %d", len(rel))
	}
	if rel[0] != "01 - Song.mp3" {
		t.Errorf("expected relative path '01 - Song.mp3', got %q", rel[0])
	}
}
