package storage

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// TemplateData holds values available for path and filename templates.
type TemplateData struct {
	Artist      string
	Album       string
	AlbumArtist string
	Title       string
	TrackNumber int
	TrackTotal  int
	Year        int
	Genre       string
}

var (
	// Windows reserved file names
	windowsReservedNames = map[string]bool{
		"CON": true, "PRN": true, "AUX": true, "NUL": true,
		"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true, "COM6": true, "COM7": true, "COM8": true, "COM9": true,
		"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true, "LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
	}

	// Invalid characters on Windows, macOS, Linux
	invalidCharsRegex = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)
)

// SanitizeFilename cleans a string so that it can be safely used as a filename
// or single directory component on Windows, macOS, and Linux.
func SanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "unnamed"
	}

	// Replace invalid characters with underscore
	cleaned := invalidCharsRegex.ReplaceAllString(name, "_")

	// Replace multiple consecutive underscores with a single underscore
	underscoreRegex := regexp.MustCompile(`_+`)
	cleaned = underscoreRegex.ReplaceAllString(cleaned, "_")

	// Trim trailing dots and spaces (disallowed on Windows)
	cleaned = strings.TrimRight(cleaned, ". ")
	cleaned = strings.TrimLeft(cleaned, " ")

	if cleaned == "" {
		return "unnamed"
	}

	// Check against Windows reserved device names (e.g. CON, PRN, AUX, etc.)
	upperBase := strings.ToUpper(cleaned)
	if idx := strings.Index(upperBase, "."); idx != -1 {
		upperBase = upperBase[:idx]
	}
	if windowsReservedNames[upperBase] {
		cleaned = "_" + cleaned
	}

	// Restrict length to 255 bytes (standard maximum filesystem filename length)
	if len(cleaned) > 240 {
		runes := []rune(cleaned)
		for len(string(runes)) > 240 {
			runes = runes[:len(runes)-1]
		}
		cleaned = strings.TrimRight(string(runes), ". ")
	}

	return cleaned
}

// FormatTrackNumber formats track number with leading zero padding.
func FormatTrackNumber(num, total int) string {
	if num <= 0 {
		return "01"
	}
	if total >= 100 || num >= 100 {
		return fmt.Sprintf("%03d", num)
	}
	return fmt.Sprintf("%02d", num)
}

// RenderFilename replaces template tokens in filename template and sanitizes the result.
func RenderFilename(tmpl string, data TemplateData) string {
	if tmpl == "" {
		tmpl = "{tracknumber} - {title}.mp3"
	}

	yearStr := ""
	if data.Year > 0 {
		yearStr = fmt.Sprintf("%04d", data.Year)
	}

	trackNumStr := FormatTrackNumber(data.TrackNumber, data.TrackTotal)
	trackTotalStr := ""
	if data.TrackTotal > 0 {
		trackTotalStr = fmt.Sprintf("%d", data.TrackTotal)
	}

	albumArtist := data.AlbumArtist
	if albumArtist == "" {
		albumArtist = data.Artist
	}

	// Ensure extension is stripped from template matching before sanitizing base
	ext := ".mp3"
	lowerTmpl := strings.ToLower(tmpl)
	if strings.HasSuffix(lowerTmpl, ".mp3") {
		tmpl = tmpl[:len(tmpl)-4]
	}

	r := strings.NewReplacer(
		"{artist}", sanitizeComponent(data.Artist),
		"{album}", sanitizeComponent(data.Album),
		"{albumartist}", sanitizeComponent(albumArtist),
		"{title}", sanitizeComponent(data.Title),
		"{tracknumber}", trackNumStr,
		"{tracktotal}", trackTotalStr,
		"{year}", yearStr,
		"{genre}", sanitizeComponent(data.Genre),
	)

	result := r.Replace(tmpl)
	result = SanitizeFilename(result)
	return result + ext
}

func sanitizeComponent(val string) string {
	val = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || strings.ContainsRune(`<>:"/\|?*`, r) {
			return '_'
		}
		return r
	}, val)
	return strings.TrimSpace(val)
}
