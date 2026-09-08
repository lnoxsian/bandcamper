package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/lnoxsian/bandcamper/internal/bandcamp"
	"github.com/lnoxsian/bandcamper/internal/config"
	"github.com/lnoxsian/bandcamper/internal/downloader"
	"github.com/lnoxsian/bandcamper/internal/storage"
)

var Version = "1.0.0"

// Exit codes conforming to Section 26 of plan.md
const (
	ExitSuccess          = 0
	ExitGeneralError     = 1
	ExitInvalidArguments = 2
	ExitInvalidURL       = 3
	ExitParseError       = 4
	ExitDownloadError    = 5
	ExitMetadataError    = 6
)

func main() {
	exitCode := run(os.Args[1:])
	os.Exit(exitCode)
}

func run(args []string) int {
	fs := flag.NewFlagSet("bandcamper", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var (
		configFile    string
		urlFile       string
		outputDir     string
		jobs          int
		retryCount    int
		filenameTmpl  string
		dirTmpl       string
		playlistFmt   string
		skipExisting  bool
		overwrite     bool
		noTags        bool
		noArtwork     bool
		noLyrics      bool
		noSaveArtwork bool
		dryRun        bool
		verbose       bool
		quiet         bool
		showVersion   bool
		showHelp      bool
	)

	fs.StringVar(&configFile, "config", "", "Path to configuration file")
	fs.StringVar(&urlFile, "file", "", "File containing Bandcamp URLs (one per line)")
	fs.StringVar(&urlFile, "f", "", "File containing Bandcamp URLs (shorthand)")

	fs.StringVar(&outputDir, "output", "", "Output directory for downloads (default: ~/Music)")
	fs.StringVar(&outputDir, "o", "", "Output directory (shorthand)")

	fs.IntVar(&jobs, "jobs", 0, "Number of concurrent download workers (default: 4)")
	fs.IntVar(&jobs, "j", 0, "Number of concurrent download workers (shorthand)")

	fs.IntVar(&retryCount, "retry", -1, "Number of retries for failed downloads (default: 3)")

	fs.StringVar(&filenameTmpl, "filename", "", "Filename format template (default: {tracknumber} - {title}.mp3)")
	fs.StringVar(&dirTmpl, "directory", "", "Directory format template (default: {artist}/{album})")
	fs.StringVar(&playlistFmt, "playlist", "", "Playlist format to generate (m3u, pls, wpl, zpl)")

	fs.BoolVar(&skipExisting, "skip-existing", false, "Skip downloading existing files (default: true)")
	fs.BoolVar(&overwrite, "overwrite", false, "Overwrite existing files (default: false)")

	fs.BoolVar(&noTags, "no-tags", false, "Disable ID3 metadata embedding")
	fs.BoolVar(&noArtwork, "no-artwork", false, "Disable embedding artwork into audio files")
	fs.BoolVar(&noLyrics, "no-lyrics", false, "Disable embedding lyrics into audio files")
	fs.BoolVar(&noSaveArtwork, "no-save-artwork", false, "Do not save cover artwork file alongside music")

	fs.BoolVar(&dryRun, "dry-run", false, "Simulate operations without downloading any files")
	fs.BoolVar(&verbose, "verbose", false, "Enable verbose debug output")
	fs.BoolVar(&verbose, "v", false, "Enable verbose output (shorthand)")
	fs.BoolVar(&quiet, "quiet", false, "Suppress normal output except errors")
	fs.BoolVar(&quiet, "q", false, "Suppress output (shorthand)")

	fs.BoolVar(&showVersion, "version", false, "Show version information and exit")
	fs.BoolVar(&showHelp, "help", false, "Show help message and exit")
	fs.BoolVar(&showHelp, "h", false, "Show help message (shorthand)")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Bandcamper %s — Fast, robust Bandcamp downloader\n\n", Version)
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  bandcamper [options] <URL>...\n")
		fmt.Fprintf(os.Stderr, "  bandcamper [options] --file <urls.txt>\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  bandcamper https://artist.bandcamp.com/album/example\n")
		fmt.Fprintf(os.Stderr, "  bandcamper --output ~/Music --jobs 4 --playlist m3u https://artist.bandcamp.com\n")
		fmt.Fprintf(os.Stderr, "  bandcamper --dry-run https://artist.bandcamp.com/track/example-track\n")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitSuccess
		}
		return ExitInvalidArguments
	}

	if showHelp {
		fs.Usage()
		return ExitSuccess
	}

	if showVersion {
		fmt.Printf("bandcamper version %s\n", Version)
		return ExitSuccess
	}

	// Load configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		return ExitGeneralError
	}

	// CLI flags override configuration
	if outputDir != "" {
		cfg.Output = outputDir
	}
	if jobs > 0 {
		cfg.Workers = jobs
	}
	if retryCount >= 0 {
		cfg.RetryCount = retryCount
	}
	if filenameTmpl != "" {
		cfg.Filename = filenameTmpl
	}
	if dirTmpl != "" {
		cfg.Directory = dirTmpl
	}
	if playlistFmt != "" {
		cfg.Playlist = playlistFmt
	}
	if fs.Lookup("skip-existing") != nil && skipExisting {
		cfg.SkipExisting = true
	}
	if fs.Lookup("overwrite") != nil && overwrite {
		cfg.Overwrite = true
	}
	if noTags {
		cfg.EmbedTags = false
	}
	if noArtwork {
		cfg.EmbedArtwork = false
	}
	if noLyrics {
		cfg.EmbedLyrics = false
	}
	if noSaveArtwork {
		cfg.SaveArtwork = false
	}
	if dryRun {
		cfg.DryRun = true
	}
	if verbose {
		cfg.Verbose = true
	}
	if quiet {
		cfg.Quiet = true
	}

	// Collect target URLs from positional arguments and --file
	var targetURLs []string
	if urlFile != "" {
		fileURLs, err := readLines(urlFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading URL file %q: %v\n", urlFile, err)
			return ExitInvalidArguments
		}
		targetURLs = append(targetURLs, fileURLs...)
	}
	targetURLs = append(targetURLs, fs.Args()...)

	if len(targetURLs) == 0 {
		fmt.Fprintf(os.Stderr, "Error: No Bandcamp URLs provided.\n\n")
		fs.Usage()
		return ExitInvalidArguments
	}

	// Setup context with signal cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		if !cfg.Quiet {
			fmt.Fprintf(os.Stderr, "\n[Interrupt] Cancelling active downloads...\n")
		}
		cancel()
	}()

	client := bandcamp.NewClient(cfg.Timeout, cfg.UserAgent)

	var lastLineMu sync.Mutex
	var lastPrintedTrack string

	progressFn := func(p downloader.TrackProgress) {
		if cfg.Quiet {
			return
		}

		lastLineMu.Lock()
		defer lastLineMu.Unlock()

		switch p.Status {
		case downloader.StatusCompleted:
			fmt.Printf("  [%d/%d] %-30s [Done]\n", p.TrackNumber, p.TrackTotal, truncateString(p.Title, 30))
		case downloader.StatusSkipped:
			fmt.Printf("  [%d/%d] %-30s [Skipped - already exists]\n", p.TrackNumber, p.TrackTotal, truncateString(p.Title, 30))
		case downloader.StatusFailed:
			fmt.Printf("  [%d/%d] %-30s [Failed: %v]\n", p.TrackNumber, p.TrackTotal, truncateString(p.Title, 30), p.Error)
		case downloader.StatusDownloading:
			trackKey := fmt.Sprintf("%d-%s", p.TrackNumber, p.Title)
			if trackKey != lastPrintedTrack {
				lastPrintedTrack = trackKey
				if cfg.Verbose {
					fmt.Printf("  [%d/%d] Downloading %q...\n", p.TrackNumber, p.TrackTotal, p.Title)
				}
			}
		}
	}

	dl := downloader.New(cfg, client, progressFn)

	var (
		hasDownloadError bool
		hasParseError    bool
		hasInvalidURL    bool
	)

	for _, rawURL := range targetURLs {
		rawURL = strings.TrimSpace(rawURL)
		if rawURL == "" {
			continue
		}

		resolved, err := bandcamp.ResolveURL(rawURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid Bandcamp URL %q: %v\n", rawURL, err)
			hasInvalidURL = true
			continue
		}

		if !cfg.Quiet {
			fmt.Printf("\nResolving %s (%s)...\n", resolved.NormalizedURL, resolved.Type)
		}

		results, err := dl.DownloadURL(ctx, resolved.NormalizedURL)
		if err != nil {
			if errors.Is(err, bandcamp.ErrParseFailed) {
				hasParseError = true
			} else {
				hasDownloadError = true
			}
			fmt.Fprintf(os.Stderr, "Failed processing %s: %v\n", resolved.NormalizedURL, err)
			continue
		}

		for _, res := range results {
			if cfg.DryRun {
				printDryRun(res, cfg)
				continue
			}

			if !cfg.Quiet {
				fmt.Printf("\nFinished: %s - %s\n", res.Release.Artist, res.Release.Album)
				fmt.Printf("Output directory: %s\n", res.OutputDir)
				fmt.Printf("Downloaded: %d, Skipped: %d, Failed: %d\n",
					len(res.DownloadedTracks), len(res.SkippedTracks), len(res.FailedTracks))
				if res.CoverPath != "" {
					fmt.Printf("Cover artwork saved: %s\n", res.CoverPath)
				}
				if res.PlaylistPath != "" {
					fmt.Printf("Playlist generated: %s\n", res.PlaylistPath)
				}
			}

			if len(res.FailedTracks) > 0 {
				hasDownloadError = true
			}
		}
	}

	if hasInvalidURL {
		return ExitInvalidURL
	}
	if hasParseError {
		return ExitParseError
	}
	if hasDownloadError {
		return ExitDownloadError
	}

	return ExitSuccess
}

func printDryRun(res *downloader.DownloadResult, cfg *config.Config) {
	rel := res.Release
	fmt.Printf("\n[DRY RUN]\n")
	fmt.Printf("Artist:  %s\n", rel.Artist)
	fmt.Printf("Album:   %s\n", rel.Album)
	fmt.Printf("Tracks:  %d\n", len(rel.Tracks))
	fmt.Printf("Output:  %s\n\n", res.OutputDir)

	year := 0
	if !rel.ReleaseDate.IsZero() {
		year = rel.ReleaseDate.Year()
	}

	for _, tr := range rel.Tracks {
		tmplData := storage.TemplateData{
			Artist:      tr.Artist,
			Album:       tr.Album,
			AlbumArtist: rel.AlbumArtist,
			Title:       tr.Title,
			TrackNumber: tr.Number,
			TrackTotal:  len(rel.Tracks),
			Year:        year,
			Genre:       rel.Genre,
		}
		fname := storage.RenderFilename(cfg.Filename, tmplData)
		fmt.Printf("  %s\n", fname)
	}
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}

	return lines, scanner.Err()
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}
