package playlist

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
)

// M3UWriter implements playlist.Writer for M3U playlist format.
type M3UWriter struct{}

// Write writes the track list into an M3U playlist at the specified path.
func (w *M3UWriter) Write(path string, tracks []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory for playlist: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create M3U playlist file %q: %w", path, err)
	}
	defer f.Close()

	bw := bufio.NewWriter(f)
	defer bw.Flush()

	if _, err := bw.WriteString("#EXTM3U\n"); err != nil {
		return err
	}

	relTracks := MakeRelative(path, tracks)
	for _, tr := range relTracks {
		if _, err := bw.WriteString(tr + "\n"); err != nil {
			return err
		}
	}

	return nil
}
