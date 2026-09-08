package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Normal Track", "Normal Track"},
		{"AC/DC - Thunderstruck", "AC_DC - Thunderstruck"},
		{"Questionable? *What* <Now>", "Questionable_ _What_ _Now_"},
		{"aux", "_aux"},
		{"CON", "_CON"},
		{"Track with trailing dots....", "Track with trailing dots"},
		{"   ", "unnamed"},
		{"Track: Subtitle | Mix", "Track_ Subtitle _ Mix"},
	}

	for _, tc := range tests {
		got := SanitizeFilename(tc.input)
		if got != tc.expected {
			t.Errorf("SanitizeFilename(%q) = %q, expected %q", tc.input, got, tc.expected)
		}
	}
}

func TestRenderFilename(t *testing.T) {
	data := TemplateData{
		Artist:      "Radiohead",
		Album:       "OK Computer",
		Title:       "Paranoid Android",
		TrackNumber: 2,
		TrackTotal:  12,
		Year:        1997,
	}

	tmpl := "{tracknumber} - {title}.mp3"
	got := RenderFilename(tmpl, data)
	expected := "02 - Paranoid Android.mp3"
	if got != expected {
		t.Errorf("RenderFilename() = %q, expected %q", got, expected)
	}

	customTmpl := "{year} - {artist} - {tracknumber} - {title}"
	gotCustom := RenderFilename(customTmpl, data)
	expectedCustom := "1997 - Radiohead - 02 - Paranoid Android.mp3"
	if gotCustom != expectedCustom {
		t.Errorf("RenderFilename() = %q, expected %q", gotCustom, expectedCustom)
	}
}

func TestRenderDirectory(t *testing.T) {
	data := TemplateData{
		Artist: "Pink Floyd",
		Album:  "The Wall: Disc 1",
		Year:   1979,
	}

	dir := RenderDirectory("{artist}/{album}", data)
	expected := filepath.Join("Pink Floyd", "The Wall_ Disc 1")
	if dir != expected {
		t.Errorf("RenderDirectory() = %q, expected %q", dir, expected)
	}
}

func TestAtomicFinalize(t *testing.T) {
	tmpDir := t.TempDir()
	partFile := filepath.Join(tmpDir, "test.mp3.part")
	finalFile := filepath.Join(tmpDir, "test.mp3")

	err := os.WriteFile(partFile, []byte("audio data"), 0644)
	if err != nil {
		t.Fatalf("failed to write part file: %v", err)
	}

	err = AtomicFinalize(partFile, finalFile, false)
	if err != nil {
		t.Fatalf("unexpected error during AtomicFinalize: %v", err)
	}

	if FileExists(partFile) {
		t.Errorf("part file should have been moved")
	}
	if !FileExists(finalFile) {
		t.Errorf("final file should exist")
	}

	// Try finalizing again when overwrite is false
	err = os.WriteFile(partFile, []byte("new data"), 0644)
	if err != nil {
		t.Fatalf("failed to write part file: %v", err)
	}

	err = AtomicFinalize(partFile, finalFile, false)
	if err == nil {
		t.Errorf("expected error when overwrite=false and target exists, got nil")
	}

	// Now finalize with overwrite=true
	err = AtomicFinalize(partFile, finalFile, true)
	if err != nil {
		t.Errorf("unexpected error with overwrite=true: %v", err)
	}
}
