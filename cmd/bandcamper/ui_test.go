package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lnoxsian/bandcamper/internal/bandcamp"
	"github.com/lnoxsian/bandcamper/internal/config"
	"github.com/lnoxsian/bandcamper/internal/downloader"
	"github.com/lnoxsian/bandcamper/internal/terminal"
)

func TestProgressUINonInteractive(t *testing.T) {
	var buf bytes.Buffer
	cfg := config.DefaultConfig()
	termInfo := terminal.Info{
		IsTerminal:     false,
		ColorSupported: false,
	}
	clr := Colorizer{Enabled: false}

	ui := NewProgressUI(cfg, termInfo, clr)
	ui.out = &buf

	// Album 1 with 3 tracks
	rel1 := &bandcamp.Release{
		Album:  "album 1",
		Artist: "Artist One",
		Tracks: []bandcamp.Track{
			{Number: 1, Title: "music 1"},
			{Number: 2, Title: "music 2"},
			{Number: 3, Title: "music 3"},
		},
	}

	ui.OnReleaseStart(rel1)
	ui.OnTrackProgress(downloader.TrackProgress{
		TrackIndex:  0,
		TrackNumber: 1,
		Title:       "music 1",
		Status:      downloader.StatusCompleted,
		Percent:     100,
	})
	ui.OnTrackProgress(downloader.TrackProgress{
		TrackIndex:  1,
		TrackNumber: 2,
		Title:       "music 2",
		Status:      downloader.StatusCompleted,
		Percent:     100,
	})
	ui.OnTrackProgress(downloader.TrackProgress{
		TrackIndex:  2,
		TrackNumber: 3,
		Title:       "music 3",
		Status:      downloader.StatusCompleted,
		Percent:     100,
	})
	ui.OnReleaseDone(&downloader.DownloadResult{
		Release: rel1,
	})

	output := buf.String()
	if !strings.Contains(output, "album 1 [0/3]") {
		t.Errorf("expected start header 'album 1 [0/3]', got:\n%s", output)
	}
	if !strings.Contains(output, "music 1") || !strings.Contains(output, "music 2") || !strings.Contains(output, "music 3") {
		t.Errorf("expected track lines, got:\n%s", output)
	}
	if !strings.Contains(output, "album 1 [3/3] OK") {
		t.Errorf("expected completion header 'album 1 [3/3] OK', got:\n%s", output)
	}

	// Album 2 with 2 tracks (1 completed, 1 in progress)
	buf.Reset()
	rel2 := &bandcamp.Release{
		Album:  "album 2",
		Artist: "Artist One",
		Tracks: []bandcamp.Track{
			{Number: 1, Title: "music 1"},
			{Number: 2, Title: "music 2"},
		},
	}

	ui.OnReleaseStart(rel2)
	ui.OnTrackProgress(downloader.TrackProgress{
		TrackIndex:  0,
		TrackNumber: 1,
		Title:       "music 1",
		Status:      downloader.StatusCompleted,
		Percent:     100,
	})

	output2 := buf.String()
	if !strings.Contains(output2, "album 2 [0/2]") {
		t.Errorf("expected start header 'album 2 [0/2]', got:\n%s", output2)
	}
	if !strings.Contains(output2, "music 1") {
		t.Errorf("expected track 1 to be printed, got:\n%s", output2)
	}
}

func TestProgressUIInteractive(t *testing.T) {
	var buf bytes.Buffer
	cfg := config.DefaultConfig()
	termInfo := terminal.Info{
		IsTerminal:     true,
		ColorSupported: false,
	}
	clr := Colorizer{Enabled: false}

	ui := NewProgressUI(cfg, termInfo, clr)
	ui.out = &buf
	ui.isTTY = true
	ui.termWidth = 80
	ui.termHeight = 24

	rel := &bandcamp.Release{
		Album:  "album 1",
		Artist: "Artist One",
		Tracks: []bandcamp.Track{
			{Number: 1, Title: "music 1"},
			{Number: 2, Title: "music 2"},
			{Number: 3, Title: "music 3"},
		},
	}

	ui.OnReleaseStart(rel)
	// Check initial rendering
	if !strings.Contains(buf.String(), "album 1 [0/3]") {
		t.Fatalf("expected initial render 'album 1 [0/3]', got:\n%s", buf.String())
	}

	// Track 1 downloading at 45%
	ui.OnTrackProgress(downloader.TrackProgress{
		TrackIndex:       0,
		TrackNumber:      1,
		Title:            "music 1",
		Status:           downloader.StatusDownloading,
		Percent:          45,
		SpeedBytesPerSec: 1024 * 1024,
	})

	if !strings.Contains(buf.String(), "45%") {
		t.Errorf("expected 45%% in output, got:\n%s", buf.String())
	}

	// Complete all tracks
	ui.OnTrackProgress(downloader.TrackProgress{
		TrackIndex:  0,
		TrackNumber: 1,
		Title:       "music 1",
		Status:      downloader.StatusCompleted,
		Percent:     100,
	})
	ui.OnTrackProgress(downloader.TrackProgress{
		TrackIndex:  1,
		TrackNumber: 2,
		Title:       "music 2",
		Status:      downloader.StatusCompleted,
		Percent:     100,
	})
	ui.OnTrackProgress(downloader.TrackProgress{
		TrackIndex:  2,
		TrackNumber: 3,
		Title:       "music 3",
		Status:      downloader.StatusCompleted,
		Percent:     100,
	})

	ui.OnReleaseDone(&downloader.DownloadResult{
		Release: rel,
	})

	finalOutput := buf.String()
	if !strings.Contains(finalOutput, "album 1 [3/3] OK") {
		t.Errorf("expected final render 'album 1 [3/3] OK', got:\n%s", finalOutput)
	}
}
