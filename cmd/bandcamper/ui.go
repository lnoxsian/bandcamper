package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/lnoxsian/bandcamper/internal/bandcamp"
	"github.com/lnoxsian/bandcamper/internal/config"
	"github.com/lnoxsian/bandcamper/internal/downloader"
	"github.com/lnoxsian/bandcamper/internal/terminal"
)

type trackState struct {
	index     int
	number    int
	title     string
	status    downloader.ProgressStatus
	percent   float64
	speed     float64
	err       error
	completed bool
}

type albumState struct {
	title       string
	artist      string
	totalTracks int
	tracks      []*trackState
	completed   int
	failed      int
	skipped     int
	done        bool
}

// ProgressUI coordinates hierarchical terminal progress rendering for albums and tracks.
type ProgressUI struct {
	mu            sync.Mutex
	cfg           *config.Config
	termInfo      terminal.Info
	clr           Colorizer
	out           io.Writer
	current       *albumState
	renderedLines int
	lastRender    time.Time
	isTTY         bool
	termWidth     int
	termHeight    int
}

// NewProgressUI initializes a new ProgressUI.
func NewProgressUI(cfg *config.Config, termInfo terminal.Info, clr Colorizer) *ProgressUI {
	width, height := terminal.GetSize()
	isTTY := termInfo.IsTerminal && (termInfo.ColorSupported || termInfo.VTEnabled || !termInfo.IsWindows)

	return &ProgressUI{
		cfg:        cfg,
		termInfo:   termInfo,
		clr:        clr,
		out:        os.Stdout,
		isTTY:      isTTY,
		termWidth:  width,
		termHeight: height,
	}
}

// OnReleaseStart is called when an album or single release starts downloading.
func (ui *ProgressUI) OnReleaseStart(rel *bandcamp.Release) {
	if ui.cfg.Quiet || ui.cfg.DryRun || rel == nil {
		return
	}

	ui.mu.Lock()
	defer ui.mu.Unlock()

	albumTitle := rel.Album
	if albumTitle == "" {
		albumTitle = "Release"
	}

	album := &albumState{
		title:       albumTitle,
		artist:      rel.Artist,
		totalTracks: len(rel.Tracks),
		tracks:      make([]*trackState, len(rel.Tracks)),
	}

	for i, tr := range rel.Tracks {
		num := tr.Number
		if num <= 0 {
			num = i + 1
		}
		album.tracks[i] = &trackState{
			index:  i,
			number: num,
			title:  tr.Title,
			status: downloader.StatusQueued,
		}
	}

	ui.current = album
	ui.renderedLines = 0
	ui.lastRender = time.Time{}

	if ui.isTTY {
		w, h := terminal.GetSize()
		if w > 0 {
			ui.termWidth = w
		}
		if h > 0 {
			ui.termHeight = h
		}
		ui.renderTTYLocked(false)
	} else {
		fmt.Fprintf(ui.out, "\n%s [0/%d]\n", album.title, album.totalTracks)
	}
}

// OnTrackProgress is called whenever a track makes progress or changes state.
func (ui *ProgressUI) OnTrackProgress(p downloader.TrackProgress) {
	if ui.cfg.Quiet || ui.cfg.DryRun {
		return
	}

	ui.mu.Lock()
	defer ui.mu.Unlock()

	album := ui.current
	if album == nil {
		return
	}

	var tr *trackState
	if p.TrackIndex >= 0 && p.TrackIndex < len(album.tracks) {
		tr = album.tracks[p.TrackIndex]
	} else {
		for _, t := range album.tracks {
			if t.number == p.TrackNumber || t.title == p.Title {
				tr = t
				break
			}
		}
	}

	if tr == nil {
		return
	}

	oldStatus := tr.status
	tr.status = p.Status
	tr.percent = p.Percent
	tr.speed = p.SpeedBytesPerSec
	tr.err = p.Error

	statusChanged := oldStatus != p.Status

	if statusChanged && !tr.completed {
		switch p.Status {
		case downloader.StatusCompleted:
			tr.completed = true
			album.completed++
		case downloader.StatusSkipped:
			tr.completed = true
			album.skipped++
			album.completed++
		case downloader.StatusFailed:
			tr.completed = true
			album.failed++
		}
	}

	if ui.isTTY {
		now := time.Now()
		if !statusChanged && now.Sub(ui.lastRender) < 60*time.Millisecond {
			return
		}
		ui.lastRender = now
		ui.renderTTYLocked(false)
	} else {
		if statusChanged && (p.Status == downloader.StatusCompleted || p.Status == downloader.StatusSkipped || p.Status == downloader.StatusFailed) {
			statusText := ui.formatStatus(tr)
			fmt.Fprintf(ui.out, "  %-32s %s\n", truncateString(tr.title, 32), statusText)
		}
	}
}

// OnReleaseDone is called when all tracks in a release have finished.
func (ui *ProgressUI) OnReleaseDone(res *downloader.DownloadResult) {
	if ui.cfg.Quiet || ui.cfg.DryRun {
		return
	}

	ui.mu.Lock()
	defer ui.mu.Unlock()

	album := ui.current
	if album == nil {
		return
	}

	album.done = true

	if ui.isTTY {
		ui.renderTTYLocked(true)
		ui.renderedLines = 0
		ui.current = nil

		if res != nil {
			if res.OutputDir != "" {
				fmt.Fprintf(ui.out, "  %s %s\n", ui.clr.Dim("Output:"), res.OutputDir)
			}
			if res.CoverPath != "" {
				fmt.Fprintf(ui.out, "  %s %s\n", ui.clr.Dim("Cover:"), res.CoverPath)
			}
			if res.PlaylistPath != "" {
				fmt.Fprintf(ui.out, "  %s %s\n", ui.clr.Dim("Playlist:"), res.PlaylistPath)
			}
		}
		fmt.Fprintln(ui.out)
	} else {
		statusTag := "OK"
		if album.failed > 0 {
			statusTag = fmt.Sprintf("[%d FAILED]", album.failed)
		}
		fmt.Fprintf(ui.out, "%s [%d/%d] %s\n", album.title, album.completed, album.totalTracks, statusTag)
		if res != nil && res.OutputDir != "" {
			fmt.Fprintf(ui.out, "  Output: %s\n", res.OutputDir)
		}
		fmt.Fprintln(ui.out)
		ui.current = nil
	}
}

func (ui *ProgressUI) renderTTYLocked(isFinal bool) {
	album := ui.current
	if album == nil {
		return
	}

	if ui.renderedLines > 0 {
		fmt.Fprintf(ui.out, "\033[%dA\r", ui.renderedLines)
	}

	linesToPrint := 0

	// 1. Album header: album 1 [3/3] OK
	header := fmt.Sprintf("%s [%d/%d]", album.title, album.completed, album.totalTracks)
	if isFinal || album.completed == album.totalTracks {
		if album.failed == 0 {
			header += " " + ui.clr.BoldGreen("OK")
		} else {
			header += " " + ui.clr.Red(fmt.Sprintf("[%d FAILED]", album.failed))
		}
	}
	fmt.Fprintf(ui.out, "\033[K%s\n", header)
	linesToPrint++

	// 2. Track list
	maxVisible := ui.termHeight - 4
	if maxVisible < 6 {
		maxVisible = 6
	}

	tracks := album.tracks
	displayTracks := tracks
	omitted := 0

	if len(tracks) > maxVisible && !isFinal {
		displayTracks, omitted = selectVisibleTracks(tracks, maxVisible)
	}

	titleWidth := ui.termWidth - 28
	if titleWidth < 20 {
		titleWidth = 20
	}
	if titleWidth > 45 {
		titleWidth = 45
	}

	for _, tr := range displayTracks {
		statusStr := ui.formatStatus(tr)
		title := truncateString(tr.title, titleWidth)
		fmt.Fprintf(ui.out, "\033[K  %-*s %s\n", titleWidth, title, statusStr)
		linesToPrint++
	}

	if omitted > 0 {
		fmt.Fprintf(ui.out, "\033[K  %s\n", ui.clr.Dim(fmt.Sprintf("... (%d more tracks)", omitted)))
		linesToPrint++
	}

	ui.renderedLines = linesToPrint
}

func (ui *ProgressUI) formatStatus(tr *trackState) string {
	switch tr.status {
	case downloader.StatusCompleted:
		return ui.clr.BoldGreen("100%")
	case downloader.StatusSkipped:
		return ui.clr.Yellow("[Skipped]")
	case downloader.StatusFailed:
		if tr.err != nil {
			return ui.clr.Red(fmt.Sprintf("[Failed: %v]", tr.err))
		}
		return ui.clr.Red("[Failed]")
	case downloader.StatusDownloading:
		pct := tr.percent
		if pct < 0 {
			pct = 0
		}
		if pct > 100 {
			pct = 100
		}
		speed := downloader.FormatSpeed(tr.speed)
		return fmt.Sprintf("%s (%s)", ui.clr.BoldCyan(fmt.Sprintf("%3.0f%%", pct)), speed)
	case downloader.StatusProcessing:
		return ui.clr.Cyan("processing...")
	case downloader.StatusQueued:
		return ui.clr.Dim("waiting")
	default:
		return ""
	}
}

func selectVisibleTracks(tracks []*trackState, max int) ([]*trackState, int) {
	if len(tracks) <= max {
		return tracks, 0
	}

	var active []*trackState
	for _, tr := range tracks {
		if tr.status == downloader.StatusDownloading || tr.status == downloader.StatusProcessing {
			active = append(active, tr)
		}
	}

	if len(active) >= max {
		return active[:max], len(tracks) - max
	}

	selected := make(map[int]bool)
	var result []*trackState
	for _, tr := range active {
		selected[tr.index] = true
		result = append(result, tr)
	}

	for _, tr := range tracks {
		if len(result) >= max {
			break
		}
		if !selected[tr.index] {
			selected[tr.index] = true
			result = append(result, tr)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].index < result[j].index
	})

	return result, len(tracks) - len(result)
}
