package downloader

import (
	"fmt"
	"io"
	"time"
)

// ProgressStatus represents the current state of a track download job.
type ProgressStatus int

const (
	StatusQueued ProgressStatus = iota
	StatusDownloading
	StatusProcessing
	StatusCompleted
	StatusSkipped
	StatusFailed
)

func (s ProgressStatus) String() string {
	switch s {
	case StatusQueued:
		return "waiting"
	case StatusDownloading:
		return "downloading"
	case StatusProcessing:
		return "processing"
	case StatusCompleted:
		return "completed"
	case StatusSkipped:
		return "skipped"
	case StatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// TrackProgress conveys download progress and status for an individual track.
type TrackProgress struct {
	TrackNumber      int
	TrackTotal       int
	Title            string
	Artist           string
	Status           ProgressStatus
	BytesDownloaded  int64
	TotalBytes       int64
	Percent          float64
	SpeedBytesPerSec float64
	Error            error
}

// ProgressFunc defines the callback for consuming progress events.
type ProgressFunc func(p TrackProgress)

// ProgressReader wraps an io.Reader to track progress and speed.
type ProgressReader struct {
	reader     io.Reader
	totalBytes int64
	readBytes  int64
	startTime  time.Time
	lastTime   time.Time
	onUpdate   func(downloaded, total int64, speed float64, percent float64)
}

// NewProgressReader initializes a ProgressReader.
func NewProgressReader(r io.Reader, totalBytes int64, onUpdate func(downloaded, total int64, speed float64, percent float64)) *ProgressReader {
	now := time.Now()
	return &ProgressReader{
		reader:     r,
		totalBytes: totalBytes,
		startTime:  now,
		lastTime:   now,
		onUpdate:   onUpdate,
	}
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if n > 0 {
		pr.readBytes += int64(n)
		now := time.Now()
		elapsed := now.Sub(pr.startTime).Seconds()
		var speed float64
		if elapsed > 0 {
			speed = float64(pr.readBytes) / elapsed
		}

		var percent float64
		if pr.totalBytes > 0 {
			percent = (float64(pr.readBytes) / float64(pr.totalBytes)) * 100.0
			if percent > 100.0 {
				percent = 100.0
			}
		}

		// Notify callback if configured and throttled
		if pr.onUpdate != nil && (now.Sub(pr.lastTime) > 100*time.Millisecond || err != nil) {
			pr.lastTime = now
			pr.onUpdate(pr.readBytes, pr.totalBytes, speed, percent)
		}
	}
	return n, err
}

// FormatSpeed returns a human-readable speed string (e.g. "4.2 MB/s").
func FormatSpeed(bytesPerSec float64) string {
	const (
		kb = 1024.0
		mb = 1024.0 * 1024.0
	)
	switch {
	case bytesPerSec >= mb:
		return fmt.Sprintf("%.1f MB/s", bytesPerSec/mb)
	case bytesPerSec >= kb:
		return fmt.Sprintf("%.1f KB/s", bytesPerSec/kb)
	default:
		return fmt.Sprintf("%.0f B/s", bytesPerSec)
	}
}

// FormatBytes returns a human-readable size string.
func FormatBytes(b int64) string {
	const (
		kb = 1024
		mb = 1024 * 1024
	)
	switch {
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
