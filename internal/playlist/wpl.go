package playlist

import (
	"bufio"
	"fmt"
	"html"
	"os"
	"path/filepath"
)

// WPLWriter implements playlist.Writer for Windows Media Player format (.wpl).
type WPLWriter struct{}

// Write writes the track list into a WPL XML playlist file.
func (w *WPLWriter) Write(path string, tracks []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory for playlist: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create WPL playlist file %q: %w", path, err)
	}
	defer f.Close()

	bw := bufio.NewWriter(f)
	defer bw.Flush()

	relTracks := MakeRelative(path, tracks)

	fmt.Fprintln(bw, `<?wpl version="1.0"?>`)
	fmt.Fprintln(bw, `<smil>`)
	fmt.Fprintln(bw, `    <head>`)
	fmt.Fprintln(bw, `        <meta name="Generator" content="Bandcamper"/>`)
	fmt.Fprintf(bw, `        <meta name="ItemCount" content="%d"/>`+"\n", len(relTracks))
	title := filepath.Base(path)
	title = title[:len(title)-len(filepath.Ext(title))]
	fmt.Fprintf(bw, `        <title>%s</title>`+"\n", html.EscapeString(title))
	fmt.Fprintln(bw, `    </head>`)
	fmt.Fprintln(bw, `    <body>`)
	fmt.Fprintln(bw, `        <seq>`)

	for _, tr := range relTracks {
		fmt.Fprintf(bw, `            <media src="%s"/>`+"\n", html.EscapeString(tr))
	}

	fmt.Fprintln(bw, `        </seq>`)
	fmt.Fprintln(bw, `    </body>`)
	fmt.Fprintln(bw, `</smil>`)

	return nil
}
