package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Workers != 4 {
		t.Errorf("expected 4 workers, got %d", cfg.Workers)
	}
	if !cfg.SkipExisting {
		t.Errorf("expected SkipExisting to be true")
	}
	if !cfg.EmbedArtwork {
		t.Errorf("expected EmbedArtwork to be true")
	}
	if cfg.RetryCount != 3 {
		t.Errorf("expected RetryCount to be 3, got %d", cfg.RetryCount)
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.toml")

	cfg := DefaultConfig()
	cfg.Workers = 8
	cfg.Output = "/custom/music"
	cfg.RetryDelay = 2 * time.Second
	cfg.Playlist = "m3u"

	err := Save(configPath, cfg)
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.Workers != 8 {
		t.Errorf("expected 8 workers, got %d", loaded.Workers)
	}
	if loaded.Output != "/custom/music" {
		t.Errorf("expected /custom/music, got %s", loaded.Output)
	}
	if loaded.RetryDelay != 2*time.Second {
		t.Errorf("expected 2s retry delay, got %v", loaded.RetryDelay)
	}
	if loaded.Playlist != "m3u" {
		t.Errorf("expected m3u playlist, got %s", loaded.Playlist)
	}
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("skipping home dir test: no home dir found")
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"~/Music", filepath.Join(home, "Music")},
		{"~", home},
		{"/var/music", "/var/music"},
		{"music", "music"},
	}

	for _, tc := range tests {
		got := ExpandHome(tc.input)
		if got != tc.expected {
			t.Errorf("ExpandHome(%q) = %q, expected %q", tc.input, got, tc.expected)
		}
	}
}
