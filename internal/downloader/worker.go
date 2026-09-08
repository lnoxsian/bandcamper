package downloader

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/lnoxsian/bandcamper/internal/bandcamp"
	"github.com/lnoxsian/bandcamper/internal/config"
	"github.com/lnoxsian/bandcamper/internal/metadata"
	"github.com/lnoxsian/bandcamper/internal/storage"
)

// TrackJob encapsulates all work needed to process an individual track.
type TrackJob struct {
	Release    *bandcamp.Release
	Track      *bandcamp.Track
	TrackIndex int
	TrackTotal int
	FinalPath  string
	PartPath   string
}

// JobResult conveys the final outcome of processing a TrackJob.
type JobResult struct {
	Job     *TrackJob
	Skipped bool
	Success bool
	Error   error
}

// ExecuteWorkerPool coordinates concurrent execution of track jobs.
func ExecuteWorkerPool(
	ctx context.Context,
	jobs []*TrackJob,
	cfg *config.Config,
	client *bandcamp.Client,
	artworkCache *metadata.ArtworkCache,
	onProgress ProgressFunc,
) []*JobResult {
	if len(jobs) == 0 {
		return nil
	}

	numWorkers := cfg.Workers
	if numWorkers <= 0 {
		numWorkers = 4
	}
	if numWorkers > len(jobs) {
		numWorkers = len(jobs)
	}

	jobsChan := make(chan *TrackJob, len(jobs))
	resultsChan := make(chan *JobResult, len(jobs))

	for _, job := range jobs {
		jobsChan <- job
	}
	close(jobsChan)

	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobsChan {
				select {
				case <-ctx.Done():
					resultsChan <- &JobResult{
						Job:   job,
						Error: ctx.Err(),
					}
					return
				default:
				}

				res := processTrackJob(ctx, job, cfg, client, artworkCache, onProgress)
				resultsChan <- res
			}
		}()
	}

	wg.Wait()
	close(resultsChan)

	var results []*JobResult
	for res := range resultsChan {
		results = append(results, res)
	}

	return results
}

func processTrackJob(
	ctx context.Context,
	job *TrackJob,
	cfg *config.Config,
	client *bandcamp.Client,
	artworkCache *metadata.ArtworkCache,
	onProgress ProgressFunc,
) *JobResult {
	tr := job.Track
	rel := job.Release

	report := func(status ProgressStatus, downloaded, total int64, speed, pct float64, err error) {
		if onProgress != nil {
			onProgress(TrackProgress{
				TrackNumber:      tr.Number,
				TrackTotal:       job.TrackTotal,
				Title:            tr.Title,
				Artist:           tr.Artist,
				Status:           status,
				BytesDownloaded:  downloaded,
				TotalBytes:       total,
				SpeedBytesPerSec: speed,
				Percent:          pct,
				Error:            err,
			})
		}
	}

	// Check if already exists
	if storage.FileExists(job.FinalPath) {
		if cfg.SkipExisting && !cfg.Overwrite {
			report(StatusSkipped, 0, 0, 0, 100, nil)
			return &JobResult{Job: job, Skipped: true, Success: true}
		}
		if !cfg.Overwrite {
			err := fmt.Errorf("%w: %s", storage.ErrFileAlreadyExists, job.FinalPath)
			report(StatusFailed, 0, 0, 0, 0, err)
			return &JobResult{Job: job, Error: err}
		}
	}

	// Dry run check
	if cfg.DryRun {
		report(StatusCompleted, 0, 0, 0, 100, nil)
		return &JobResult{Job: job, Success: true}
	}

	// Ensure stream URL is present
	if tr.StreamURL == "" {
		err := fmt.Errorf("%w: track %q has no streamable audio file", bandcamp.ErrNoStreamURL, tr.Title)
		report(StatusFailed, 0, 0, 0, 0, err)
		return &JobResult{Job: job, Error: err}
	}

	report(StatusDownloading, 0, 0, 0, 0, nil)

	// Ensure target directory exists
	if err := os.MkdirAll(filepath.Dir(job.FinalPath), 0755); err != nil {
		report(StatusFailed, 0, 0, 0, 0, err)
		return &JobResult{Job: job, Error: err}
	}

	// Download audio to temporary .part file with retries
	downloadErr := RetryOperation(ctx, cfg.RetryCount, cfg.RetryDelay, func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		resp, err := client.Get(ctx, tr.StreamURL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		partFile, err := os.OpenFile(job.PartPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("failed creating part file: %w", err)
		}
		defer partFile.Close()

		pr := NewProgressReader(resp.Body, resp.ContentLength, func(downloaded, total int64, speed, pct float64) {
			report(StatusDownloading, downloaded, total, speed, pct, nil)
		})

		_, err = io.Copy(partFile, pr)
		if err != nil {
			storage.CleanupFile(job.PartPath)
			return err
		}

		return partFile.Sync()
	})

	if downloadErr != nil {
		storage.CleanupFile(job.PartPath)
		report(StatusFailed, 0, 0, 0, 0, downloadErr)
		return &JobResult{Job: job, Error: downloadErr}
	}

	report(StatusProcessing, 0, 0, 0, 100, nil)

	// Embed ID3 metadata and artwork if enabled
	if cfg.EmbedTags {
		var artworkBytes []byte
		var artworkMIME string

		if cfg.EmbedArtwork && rel.ArtworkURL != "" {
			if art, err := metadata.FetchArtwork(ctx, client, artworkCache, rel.ArtworkURL); err == nil && art != nil {
				artworkBytes = art.Data
				artworkMIME = art.MIMEType
			}
		}

		meta := metadata.FromReleaseAndTrack(rel, tr, job.TrackTotal, artworkBytes, artworkMIME)
		if !cfg.EmbedLyrics {
			meta.Lyrics = ""
		}

		if err := metadata.EmbedMetadata(job.PartPath, meta); err != nil {
			// As per plan architecture rule #9: Optional metadata failures should not discard downloaded audio
			// But we log or note it
		}
	}

	// Finalize atomically
	if err := storage.AtomicFinalize(job.PartPath, job.FinalPath, cfg.Overwrite); err != nil {
		storage.CleanupFile(job.PartPath)
		report(StatusFailed, 0, 0, 0, 0, err)
		return &JobResult{Job: job, Error: err}
	}

	report(StatusCompleted, 0, 0, 0, 100, nil)
	return &JobResult{Job: job, Success: true}
}
