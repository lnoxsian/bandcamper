package downloader

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lnoxsian/bandcamper/internal/bandcamp"
	"github.com/lnoxsian/bandcamper/internal/config"
	"github.com/lnoxsian/bandcamper/internal/metadata"
	"github.com/lnoxsian/bandcamper/internal/playlist"
	"github.com/lnoxsian/bandcamper/internal/provider"
	"github.com/lnoxsian/bandcamper/internal/soundcloud"
	"github.com/lnoxsian/bandcamper/internal/storage"
)

// TrackError records an error encountered for a specific track.
type TrackError struct {
	TrackNumber int
	Title       string
	Error       error
}

// DownloadResult summarizes the outcome of downloading a Bandcamp release.
type DownloadResult struct {
	Release          *bandcamp.Release
	OutputDir        string
	DownloadedTracks []string
	SkippedTracks    []string
	FailedTracks     []TrackError
	CoverPath        string
	PlaylistPath     string
}

// Downloader orchestrates downloading of music releases, artwork, and playlist generation across providers.
type Downloader struct {
	Config         *config.Config
	Client         *bandcamp.Client
	Registry       *provider.Registry
	ArtworkCache   *metadata.ArtworkCache
	Progress       ProgressFunc
	OnReleaseStart func(rel *bandcamp.Release)
	OnReleaseDone  func(res *DownloadResult)
}

// New creates a new Downloader instance and initializes supported providers.
func New(cfg *config.Config, client *bandcamp.Client, progress ProgressFunc) *Downloader {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	if client == nil {
		client = bandcamp.NewClient(cfg.Timeout, cfg.UserAgent)
	}

	reg := provider.NewRegistry()
	if cfg.Providers.Bandcamp {
		reg.Register(bandcamp.NewProvider(client))
	}
	if cfg.Providers.Soundcloud {
		scClient := soundcloud.NewClient(cfg.Timeout, cfg.UserAgent)
		reg.Register(soundcloud.NewProvider(scClient))
	}

	return &Downloader{
		Config:       cfg,
		Client:       client,
		Registry:     reg,
		ArtworkCache: metadata.NewArtworkCache(),
		Progress:     progress,
	}
}

// NewWithRegistry creates a Downloader with an explicit custom Registry.
func NewWithRegistry(cfg *config.Config, reg *provider.Registry, client *bandcamp.Client, progress ProgressFunc) *Downloader {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	if client == nil {
		client = bandcamp.NewClient(cfg.Timeout, cfg.UserAgent)
	}
	if reg == nil {
		reg = provider.NewRegistry()
	}
	return &Downloader{
		Config:       cfg,
		Client:       client,
		Registry:     reg,
		ArtworkCache: metadata.NewArtworkCache(),
		Progress:     progress,
	}
}

// DownloadURL resolves any supported URL (Bandcamp, SoundCloud, etc.) and downloads all associated releases.
func (d *Downloader) DownloadURL(ctx context.Context, rawURL string) ([]*DownloadResult, error) {
	return d.DownloadURLWithProvider(ctx, rawURL, "")
}

// DownloadURLWithProvider downloads a URL using an explicit provider or auto-detecting if providerName is empty.
func (d *Downloader) DownloadURLWithProvider(ctx context.Context, rawURL string, providerName string) ([]*DownloadResult, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, fmt.Errorf("%w: empty URL", bandcamp.ErrInvalidURL)
	}

	targetURL := rawURL
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "https://" + targetURL
	}

	parsed, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}

	prov, err := d.Registry.ResolveProvider(parsed, providerName)
	if err != nil {
		return nil, err
	}

	releases, err := prov.Resolve(ctx, parsed)
	if err != nil {
		return nil, err
	}

	var results []*DownloadResult
	for _, rel := range releases {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		res, err := d.DownloadRelease(ctx, rel)
		if err != nil {
			return results, err
		}
		results = append(results, res)
	}

	return results, nil
}

func (d *Downloader) downloadArtist(ctx context.Context, artistURL string) ([]*DownloadResult, error) {
	htmlContent, err := d.Client.FetchString(ctx, artistURL, 5*1024*1024)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch artist page %s: %w", artistURL, err)
	}

	releaseURLs, err := bandcamp.ParseArtistDiscography(htmlContent, artistURL)
	if err != nil {
		return nil, err
	}

	var results []*DownloadResult
	var lastErr error

	for _, relURL := range releaseURLs {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		resList, err := d.DownloadURL(ctx, relURL)
		if err != nil {
			lastErr = err
			continue
		}
		results = append(results, resList...)
	}

	if len(results) == 0 && lastErr != nil {
		return nil, lastErr
	}

	return results, nil
}

func (d *Downloader) downloadAlbum(ctx context.Context, albumURL string) (*DownloadResult, error) {
	htmlContent, err := d.Client.FetchString(ctx, albumURL, 5*1024*1024)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch album page %s: %w", albumURL, err)
	}

	release, err := bandcamp.ParseAlbum(htmlContent, albumURL)
	if err != nil {
		return nil, err
	}

	return d.DownloadRelease(ctx, release)
}

func (d *Downloader) downloadTrack(ctx context.Context, trackURL string) (*DownloadResult, error) {
	htmlContent, err := d.Client.FetchString(ctx, trackURL, 5*1024*1024)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch track page %s: %w", trackURL, err)
	}

	release, err := bandcamp.ParseTrack(htmlContent, trackURL)
	if err != nil {
		return nil, err
	}

	return d.DownloadRelease(ctx, release)
}

// DownloadRelease processes all tracks in a parsed Release model.
func (d *Downloader) DownloadRelease(ctx context.Context, rel *bandcamp.Release) (*DownloadResult, error) {
	if rel == nil || len(rel.Tracks) == 0 {
		return nil, fmt.Errorf("%w: release has no tracks", bandcamp.ErrParseFailed)
	}

	if d.OnReleaseStart != nil {
		d.OnReleaseStart(rel)
	}

	year := 0
	if !rel.ReleaseDate.IsZero() {
		year = rel.ReleaseDate.Year()
	}

	tmplData := storage.TemplateData{
		Provider:    rel.Provider,
		Artist:      rel.Artist,
		Album:       rel.Album,
		AlbumArtist: rel.AlbumArtist,
		Year:        year,
		Genre:       rel.Genre,
	}

	outputDir := storage.ResolveOutputDir(d.Config.Output, d.Config.Directory, tmplData)

	result := &DownloadResult{
		Release:   rel,
		OutputDir: outputDir,
	}

	// Save cover artwork if requested
	if d.Config.SaveArtwork && rel.ArtworkURL != "" && !d.Config.DryRun {
		if art, err := metadata.FetchArtwork(ctx, d.Client, d.ArtworkCache, rel.ArtworkURL); err == nil && art != nil {
			if coverPath, err := metadata.SaveCoverFile(outputDir, art); err == nil {
				result.CoverPath = coverPath
			}
		}
	}

	// Prepare track jobs
	trackTotal := len(rel.Tracks)
	var jobs []*TrackJob

	for idx, tr := range rel.Tracks {
		trackTmplData := storage.TemplateData{
			Provider:    rel.Provider,
			Artist:      tr.Artist,
			Album:       tr.Album,
			AlbumArtist: rel.AlbumArtist,
			Title:       tr.Title,
			TrackNumber: tr.Number,
			TrackTotal:  trackTotal,
			Year:        year,
			Genre:       rel.Genre,
		}

		finalPath := filepath.Join(outputDir, storage.RenderFilename(d.Config.Filename, trackTmplData))
		partPath := storage.PartFilePath(finalPath)

		copyTrack := tr
		jobs = append(jobs, &TrackJob{
			Release:    rel,
			Track:      &copyTrack,
			TrackIndex: idx,
			TrackTotal: trackTotal,
			FinalPath:  finalPath,
			PartPath:   partPath,
		})
	}

	jobResults := ExecuteWorkerPool(ctx, jobs, d.Config, d.Client, d.ArtworkCache, d.Progress)

	// Sort results to preserve the original track index order
	sort.Slice(jobResults, func(i, j int) bool {
		return jobResults[i].Job.TrackIndex < jobResults[j].Job.TrackIndex
	})

	var validTrackFiles []string

	for _, jr := range jobResults {
		if jr.Error != nil {
			result.FailedTracks = append(result.FailedTracks, TrackError{
				TrackNumber: jr.Job.Track.Number,
				Title:       jr.Job.Track.Title,
				Error:       jr.Error,
			})
		} else if jr.Skipped {
			result.SkippedTracks = append(result.SkippedTracks, jr.Job.FinalPath)
			validTrackFiles = append(validTrackFiles, jr.Job.FinalPath)
		} else if jr.Success {
			result.DownloadedTracks = append(result.DownloadedTracks, jr.Job.FinalPath)
			validTrackFiles = append(validTrackFiles, jr.Job.FinalPath)
		}
	}

	// Generate playlist if configured and any tracks succeeded/were skipped
	if d.Config.Playlist != "" && len(validTrackFiles) > 0 && !d.Config.DryRun {
		writer, err := playlist.GetWriter(d.Config.Playlist)
		if err == nil {
			ext := playlist.FormatExtension(d.Config.Playlist)
			playlistBase := storage.SanitizeFilename(rel.Artist + " - " + rel.Album)
			playlistPath := filepath.Join(outputDir, playlistBase+ext)

			if err := writer.Write(playlistPath, validTrackFiles); err == nil {
				result.PlaylistPath = playlistPath
			}
		}
	}

	if d.OnReleaseDone != nil {
		d.OnReleaseDone(result)
	}

	return result, nil
}

// DownloadAll executes downloads for multiple URLs.
func (d *Downloader) DownloadAll(ctx context.Context, urls []string) ([]*DownloadResult, error) {
	var allResults []*DownloadResult
	var failedURLs []string

	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		resList, err := d.DownloadURL(ctx, u)
		if err != nil {
			failedURLs = append(failedURLs, fmt.Sprintf("%s: %v", u, err))
			continue
		}
		allResults = append(allResults, resList...)
	}

	if len(failedURLs) > 0 && len(allResults) == 0 {
		return nil, fmt.Errorf("failed downloading URLs:\n%s", strings.Join(failedURLs, "\n"))
	}

	return allResults, nil
}
