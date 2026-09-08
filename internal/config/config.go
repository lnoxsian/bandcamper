package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// Config holds all configuration options for the downloader.
type Config struct {
	Output       string        `toml:"output"`
	Workers      int           `toml:"workers"`
	RetryCount   int           `toml:"retry_count"`
	RetryDelay   time.Duration `toml:"retry_delay"`
	SkipExisting bool          `toml:"skip_existing"`
	Overwrite    bool          `toml:"overwrite"`

	Filename  string `toml:"filename"`
	Directory string `toml:"directory"`

	EmbedArtwork bool `toml:"embed_artwork"`
	SaveArtwork  bool `toml:"save_artwork"`
	EmbedLyrics  bool `toml:"embed_lyrics"`
	EmbedTags    bool `toml:"embed_tags"`

	Playlist string        `toml:"playlist"`
	DryRun   bool          `toml:"dry_run"`
	Verbose  bool          `toml:"verbose"`
	Quiet    bool          `toml:"quiet"`
	Color    bool          `toml:"color"`

	UserAgent string        `toml:"user_agent"`
	Timeout   time.Duration `toml:"timeout"`
}

// DefaultConfig returns a new Config populated with sensible default values.
func DefaultConfig() *Config {
	return &Config{
		Output:       "~/Music",
		Workers:      4,
		RetryCount:   3,
		RetryDelay:   time.Second,
		SkipExisting: true,
		Overwrite:    false,

		Filename:  "{tracknumber} - {title}.mp3",
		Directory: "{artist}/{album}",

		EmbedArtwork: true,
		SaveArtwork:  true,
		EmbedLyrics:  true,
		EmbedTags:    true,

		Playlist: "",
		DryRun:   false,
		Verbose:  false,
		Quiet:    false,
		Color:    true,

		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
		Timeout:   30 * time.Second,
	}
}

// DefaultConfigPath returns the OS-specific path for the configuration file.
func DefaultConfigPath() (string, error) {
	appName := "bandcamper"

	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("could not determine user home directory: %w", err)
			}
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, appName, "config.toml"), nil

	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not determine user home directory: %w", err)
		}
		return filepath.Join(home, "Library", "Application Support", appName, "config.toml"), nil

	default:
		// Linux, BSD, and other POSIX
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig != "" {
			return filepath.Join(xdgConfig, appName, "config.toml"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not determine user home directory: %w", err)
		}
		return filepath.Join(home, ".config", appName, "config.toml"), nil
	}
}

// ExpandHome expands the leading "~" in path to the user's home directory.
func ExpandHome(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
		return filepath.Join(home, path[2:])
	}
	return path
}

// Load loads configuration from a TOML file. If path is empty, the default
// location is checked. If the file does not exist, DefaultConfig is returned.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	targetPath := path
	if targetPath == "" {
		defPath, err := DefaultConfigPath()
		if err != nil {
			return cfg, nil // Fall back to default
		}
		targetPath = defPath
	} else {
		targetPath = ExpandHome(targetPath)
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		if os.IsNotExist(err) && path == "" {
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file %q: %w", targetPath, err)
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse TOML in %q: %w", targetPath, err)
	}

	return cfg, nil
}

// Save writes the configuration to the specified TOML file.
func Save(path string, cfg *Config) error {
	targetPath := ExpandHome(path)
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(targetPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
