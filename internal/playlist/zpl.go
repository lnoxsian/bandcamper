package playlist

import (
	"bufio"
	"fmt"
	"html"
	"os"
	"path/filepath"
)

// ZPLWriter implements playlist.Writer for Zune Playlist format (.zpl).
type ZPLWriter struct{}

// Write writes the track list into a ZPL XML playlist file.
func (w *ZPLWriter) Write(path string, tracks []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory for playlist: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create ZPL playlist file %q: %w", path, err)
	}
	defer f.Close()

	bw := bufio.NewWriter(f)
	defer bw.Flush()

	relTracks := MakeRelative(path, tracks)

	fmt.Fprintln(bw, `<?zpl version="2.0"?>`)
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
