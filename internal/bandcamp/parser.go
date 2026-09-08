package bandcamp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TralbumData matches Bandcamp's internal album/track data structure.
type TralbumData struct {
	ArtID      json.Number    `json:"art_id"`
	Artist     string         `json:"artist"`
	Current    TralbumCurrent `json:"current"`
	Trackinfo  []TralbumTrack `json:"trackinfo"`
	ItemType   string         `json:"item_type"`
	AlbumTitle string         `json:"album_title"`
	URL        string         `json:"url"`
	IsPreorder bool           `json:"is_preorder"`
	HasAudio   bool           `json:"hasAudio"`
}

// TralbumCurrent contains metadata for the specific album or track.
type TralbumCurrent struct {
	ID          int64       `json:"id"`
	Type        string      `json:"type"`
	Title       string      `json:"title"`
	Artist      string      `json:"artist"`
	ReleaseDate string      `json:"release_date"`
	PublishDate string      `json:"publish_date"`
	About       string      `json:"about"`
	Lyrics      string      `json:"lyrics"`
	TrackNumber int         `json:"track_number"`
	ArtID       json.Number `json:"art_id"`
}

// TralbumTrack represents track info inside TralbumData.
type TralbumTrack struct {
	ID        int64             `json:"id"`
	TrackNum  int               `json:"track_num"`
	Title     string            `json:"title"`
	Artist    string            `json:"artist"`
	Duration  float64           `json:"duration"`
	File      map[string]string `json:"file"`
	Lyrics    string            `json:"lyrics"`
	TitleLink string            `json:"title_link"`
}

// SchemaLDJson represents schema.org metadata.
type SchemaLDJson struct {
	Type          string          `json:"@type"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	DatePublished string          `json:"datePublished"`
	Image         any             `json:"image"`
	Genre         any             `json:"genre"`
	ByArtist      SchemaArtist    `json:"byArtist"`
	Track         SchemaTrackList `json:"track"`
}

type SchemaArtist struct {
	Name string `json:"name"`
}

type SchemaTrackList struct {
	ItemListElement []SchemaTrackItem `json:"itemListElement"`
}

type SchemaTrackItem struct {
	Position int                `json:"position"`
	Item     SchemaMusicTrack   `json:"item"`
}

type SchemaMusicTrack struct {
	Name     string `json:"name"`
	Duration string `json:"duration"`
}

var (
	tralbumAttrRegex = regexp.MustCompile(`(?s)data-tralbum="([^"]+)"`)
	tralbumAttrAlt   = regexp.MustCompile(`(?s)data-tralbum='([^']+)'`)
	metaPropRegex    = regexp.MustCompile(`(?i)<meta\s+property=["']([^"']+)["']\s+content=["']([^"']*)["']`)
	metaNameRegex    = regexp.MustCompile(`(?i)<meta\s+name=["']([^"']+)["']\s+content=["']([^"']*)["']`)
	linkRelRegex     = regexp.MustCompile(`(?i)<link\s+rel=["']([^"']+)["']\s+href=["']([^"']*)["']`)
	ldJsonRegex      = regexp.MustCompile(`(?s)<script[^>]*type=["']application/ld\+json["'][^>]*>(.*?)</script>`)
)

// ExtractTralbumData extracts the TralbumData object from page HTML.
func ExtractTralbumData(htmlContent string) (*TralbumData, error) {
	// Strategy 1: Check data-tralbum HTML attribute
	if match := tralbumAttrRegex.FindStringSubmatch(htmlContent); len(match) > 1 {
		unescaped := html.UnescapeString(match[1])
		var data TralbumData
		if err := json.Unmarshal([]byte(unescaped), &data); err == nil {
			return &data, nil
		}
	}
	if match := tralbumAttrAlt.FindStringSubmatch(htmlContent); len(match) > 1 {
		unescaped := html.UnescapeString(match[1])
		var data TralbumData
		if err := json.Unmarshal([]byte(unescaped), &data); err == nil {
			return &data, nil
		}
	}

	// Strategy 2: Look for JavaScript `var TralbumData = { ... };` or `TralbumData = { ... };`
	idx := strings.Index(htmlContent, "TralbumData")
	for idx != -1 {
		assignIdx := strings.Index(htmlContent[idx:], "=")
		if assignIdx != -1 && assignIdx < 40 {
			startPos := idx + assignIdx + 1
			braceIdx := strings.Index(htmlContent[startPos:], "{")
			if braceIdx != -1 && braceIdx < 20 {
				jsonStart := startPos + braceIdx
				if jsonStr, ok := extractBalancedBraces(htmlContent[jsonStart:]); ok {
					var data TralbumData
					// Try unmarshaling directly
					if err := json.Unmarshal([]byte(jsonStr), &data); err == nil && (data.Artist != "" || data.Current.Title != "" || len(data.Trackinfo) > 0) {
						return &data, nil
					}
					// If that failed due to JS keys not being quoted, clean up JS object
					cleaned := cleanJSObjectToJSON(jsonStr)
					if err := json.Unmarshal([]byte(cleaned), &data); err == nil && (data.Artist != "" || data.Current.Title != "" || len(data.Trackinfo) > 0) {
						return &data, nil
					}
				}
			}
		}
		next := strings.Index(htmlContent[idx+11:], "TralbumData")
		if next == -1 {
			break
		}
		idx += 11 + next
	}

	return nil, fmt.Errorf("%w: could not find TralbumData in HTML", ErrParseFailed)
}

// ExtractLDJson extracts SchemaLDJson data if available.
func ExtractLDJson(htmlContent string) (*SchemaLDJson, error) {
	matches := ldJsonRegex.FindAllStringSubmatch(htmlContent, -1)
	for _, match := range matches {
		if len(match) > 1 {
			raw := strings.TrimSpace(match[1])
			var schema SchemaLDJson
			if err := json.Unmarshal([]byte(raw), &schema); err == nil {
				if schema.Type == "MusicAlbum" || schema.Type == "MusicRecording" || schema.Name != "" {
					return &schema, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("%w: no schema.org JSON found", ErrParseFailed)
}

// ExtractMetaTags returns all OpenGraph and standard meta tags from HTML.
func ExtractMetaTags(htmlContent string) map[string]string {
	meta := make(map[string]string)

	propMatches := metaPropRegex.FindAllStringSubmatch(htmlContent, -1)
	for _, m := range propMatches {
		if len(m) > 2 {
			meta[strings.ToLower(m[1])] = html.UnescapeString(m[2])
		}
	}

	nameMatches := metaNameRegex.FindAllStringSubmatch(htmlContent, -1)
	for _, m := range nameMatches {
		if len(m) > 2 {
			key := strings.ToLower(m[1])
			if _, exists := meta[key]; !exists {
				meta[key] = html.UnescapeString(m[2])
			}
		}
	}

	linkMatches := linkRelRegex.FindAllStringSubmatch(htmlContent, -1)
	for _, m := range linkMatches {
		if len(m) > 2 && strings.ToLower(m[1]) == "image_src" {
			meta["image_src"] = html.UnescapeString(m[2])
		}
	}

	return meta
}

// ParseReleaseDate parses Bandcamp release dates from various formats.
func ParseReleaseDate(dateStr string) time.Time {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return time.Time{}
	}

	layouts := []string{
		"02 Jan 2006 15:04:05 MST",
		"02 Jan 2006 15:04:05 -0700",
		"02 Jan 2006",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02",
		"January 2, 2006",
		"Jan 2, 2006",
		"02-01-2006",
		"20060102",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return t
		}
	}

	return time.Time{}
}

// BuildArtworkURL constructs high-resolution artwork URL from Bandcamp art ID.
func BuildArtworkURL(artIDStr string) string {
	artIDStr = strings.TrimSpace(artIDStr)
	if artIDStr == "" || artIDStr == "0" {
		return ""
	}
	id, err := strconv.ParseInt(artIDStr, 10, 64)
	if err != nil || id <= 0 {
		return ""
	}
	// _10.jpg is the standard high-resolution artwork (typically up to 1200x1200px)
	return fmt.Sprintf("https://f4.bcbits.com/img/a%d_10.jpg", id)
}

// UpgradeArtworkQuality replaces lower-resolution suffixes (_16.jpg, _2.jpg, etc.) with _10.jpg
func UpgradeArtworkQuality(imgURL string) string {
	if !strings.Contains(imgURL, "bcbits.com") {
		return imgURL
	}
	re := regexp.MustCompile(`_(\d+)\.(jpg|png)$`)
	return re.ReplaceAllString(imgURL, "_10.$2")
}

// extractBalancedBraces captures the string between the opening { and matching }.
func extractBalancedBraces(s string) (string, bool) {
	if len(s) == 0 || s[0] != '{' {
		return "", false
	}

	depth := 0
	inString := false
	var stringDelimiter byte
	isEscaped := false

	for i := 0; i < len(s); i++ {
		c := s[i]

		if inString {
			if isEscaped {
				isEscaped = false
			} else if c == '\\' {
				isEscaped = true
			} else if c == stringDelimiter {
				inString = false
			}
			continue
		}

		if c == '"' || c == '\'' {
			inString = true
			stringDelimiter = c
			continue
		}

		if c == '{' {
			depth++
		} else if c == '}' {
			depth--
			if depth == 0 {
				return s[:i+1], true
			}
		}
	}

	return "", false
}

// cleanJSObjectToJSON quotes unquoted object keys for basic JavaScript objects.
func cleanJSObjectToJSON(js string) string {
	var buf bytes.Buffer
	inString := false
	var strChar byte
	isEscaped := false
	keyCapture := false
	var keyBuf bytes.Buffer

	for i := 0; i < len(js); i++ {
		c := js[i]

		if inString {
			buf.WriteByte(c)
			if isEscaped {
				isEscaped = false
			} else if c == '\\' {
				isEscaped = true
			} else if c == strChar {
				inString = false
			}
			continue
		}

		if c == '"' || c == '\'' {
			inString = true
			strChar = c
			buf.WriteByte('"')
			continue
		}

		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || (keyCapture && (c >= '0' && c <= '9')) {
			keyCapture = true
			keyBuf.WriteByte(c)
			continue
		}

		if keyCapture {
			// Check if following non-whitespace character is colon ':'
			rest := strings.TrimLeft(js[i:], " \t\r\n")
			if strings.HasPrefix(rest, ":") {
				buf.WriteString(`"` + keyBuf.String() + `"`)
			} else {
				buf.WriteString(keyBuf.String())
			}
			keyCapture = false
			keyBuf.Reset()
		}

		buf.WriteByte(c)
	}

	return buf.String()
}

// CleanLyrics normalizes lyric text: removes HTML tags, normalizes line breaks.
func CleanLyrics(raw string) string {
	if raw == "" {
		return ""
	}
	s := html.UnescapeString(raw)
	// Replace <br> and <p> tags with newlines
	brRegex := regexp.MustCompile(`(?i)<br\s*/?>`)
	s = brRegex.ReplaceAllString(s, "\n")
	tagRegex := regexp.MustCompile(`<[^>]*>`)
	s = tagRegex.ReplaceAllString(s, "")
	// Normalize CRLF to LF
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	// Trim trailing / leading whitespace
	return strings.TrimSpace(s)
}

// CleanDescription strips excessive HTML tags and unescapes text.
func CleanDescription(raw string) string {
	if raw == "" {
		return ""
	}
	s := html.UnescapeString(raw)
	brRegex := regexp.MustCompile(`(?i)<br\s*/?>`)
	s = brRegex.ReplaceAllString(s, "\n")
	tagRegex := regexp.MustCompile(`<[^>]*>`)
	s = tagRegex.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

// ResolveAbsoluteURL combines a relative URL with a base URL.
func ResolveAbsoluteURL(baseURLStr, refURLStr string) string {
	base, err := url.Parse(baseURLStr)
	if err != nil {
		return refURLStr
	}
	ref, err := url.Parse(refURLStr)
	if err != nil {
		return refURLStr
	}
	return base.ResolveReference(ref).String()
}
