package playlist

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PLSWriter implements playlist.Writer for PLS format.
type PLSWriter struct{}

// Write writes the track list into a PLS playlist file.
func (w *PLSWriter) Write(path string, tracks []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory for playlist: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create PLS playlist file %q: %w", path, err)
	}
	defer f.Close()

	bw := bufio.NewWriter(f)
	defer bw.Flush()

	fmt.Fprintln(bw, "[playlist]")

	relTracks := MakeRelative(path, tracks)
	for idx, tr := range relTracks {
		entryNum := idx + 1
		title := filepath.Base(tr)
		title = strings.TrimSuffix(title, filepath.Ext(title))

		fmt.Fprintf(bw, "File%d=%s\n", entryNum, tr)
		fmt.Fprintf(bw, "Title%d=%s\n", entryNum, title)
		fmt.Fprintf(bw, "Length%d=-1\n", entryNum)
	}

	fmt.Fprintf(bw, "NumberOfEntries=%d\n", len(relTracks))
	fmt.Fprintln(bw, "Version=2")

	return nil
}
