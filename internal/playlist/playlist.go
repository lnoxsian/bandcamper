package playlist

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Writer is the common interface for playlist file generators.
type Writer interface {
	Write(path string, tracks []string) error
}

// GetWriter returns the appropriate playlist Writer for the specified format.
func GetWriter(format string) (Writer, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "m3u":
		return &M3UWriter{}, nil
	case "pls":
		return &PLSWriter{}, nil
	case "wpl":
		return &WPLWriter{}, nil
	case "zpl":
		return &ZPLWriter{}, nil
	default:
		return nil, fmt.Errorf("unsupported playlist format: %q (supported: m3u, pls, wpl, zpl)", format)
	}
}

// FormatExtension returns the default file extension for a playlist format.
func FormatExtension(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "m3u":
		return ".m3u"
	case "pls":
		return ".pls"
	case "wpl":
		return ".wpl"
	case "zpl":
		return ".zpl"
	default:
		return ".m3u"
	}
}

// MakeRelative computes the relative path of each track with respect to the playlist directory.
func MakeRelative(playlistPath string, tracks []string) []string {
	playlistDir := filepath.Dir(playlistPath)
	result := make([]string, len(tracks))

	for i, tr := range tracks {
		if rel, err := filepath.Rel(playlistDir, tr); err == nil {
			// Normalize to forward slashes for cross-platform playlist compatibility
			result[i] = filepath.ToSlash(rel)
		} else {
			result[i] = tr
		}
	}

	return result
}
