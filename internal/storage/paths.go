package storage

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/lnoxsian/bandcamper/internal/config"
)

// RenderDirectory replaces template tokens in directory template and sanitizes each path component.
func RenderDirectory(tmpl string, data TemplateData) string {
	if tmpl == "" {
		tmpl = "{artist}/{album}"
	}

	yearStr := ""
	if data.Year > 0 {
		yearStr = fmt.Sprintf("%04d", data.Year)
	}

	albumArtist := data.AlbumArtist
	if albumArtist == "" {
		albumArtist = data.Artist
	}

	r := strings.NewReplacer(
		"{artist}", data.Artist,
		"{album}", data.Album,
		"{albumartist}", albumArtist,
		"{year}", yearStr,
		"{genre}", data.Genre,
	)

	expanded := r.Replace(tmpl)

	// Split by '/' or '\' and sanitize each path segment separately
	rawParts := strings.FieldsFunc(expanded, func(c rune) bool {
		return c == '/' || c == '\\'
	})

	var cleanParts []string
	for _, part := range rawParts {
		sanitized := SanitizeFilename(part)
		if sanitized != "" && sanitized != "." && sanitized != ".." {
			cleanParts = append(cleanParts, sanitized)
		}
	}

	if len(cleanParts) == 0 {
		return "Unknown"
	}

	return filepath.Join(cleanParts...)
}

// ResolveOutputDir resolves the full directory path for a release.
func ResolveOutputDir(baseOutput, dirTemplate string, data TemplateData) string {
	expandedBase := config.ExpandHome(baseOutput)
	if expandedBase == "" {
		expandedBase = "."
	}
	subDir := RenderDirectory(dirTemplate, data)
	return filepath.Join(expandedBase, subDir)
}

// ResolveTrackPath returns the full filesystem path for a track file.
func ResolveTrackPath(baseOutput, dirTemplate, filenameTemplate string, data TemplateData) string {
	dir := ResolveOutputDir(baseOutput, dirTemplate, data)
	fileName := RenderFilename(filenameTemplate, data)
	return filepath.Join(dir, fileName)
}
