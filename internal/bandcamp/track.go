package bandcamp

import (
	"fmt"
	"strings"
	"time"
)

// ParseTrack parses an individual track page HTML and returns a Release representing that track.
func ParseTrack(htmlContent, pageURL string) (*Release, error) {
	tralbum, _ := ExtractTralbumData(htmlContent)
	meta := ExtractMetaTags(htmlContent)
	schema, _ := ExtractLDJson(htmlContent)

	release := &Release{
		URL: pageURL,
	}

	var track Track

	if tralbum != nil {
		release.Artist = strings.TrimSpace(tralbum.Artist)
		if release.Artist == "" {
			release.Artist = strings.TrimSpace(tralbum.Current.Artist)
		}

		release.Album = strings.TrimSpace(tralbum.AlbumTitle)
		if release.Album == "" {
			// If not part of an album, use the track title or release title as single album name
			release.Album = strings.TrimSpace(tralbum.Current.Title)
		}

		release.AlbumArtist = release.Artist
		release.Description = CleanDescription(tralbum.Current.About)
		release.ReleaseDate = ParseReleaseDate(tralbum.Current.ReleaseDate)
		if release.ReleaseDate.IsZero() {
			release.ReleaseDate = ParseReleaseDate(tralbum.Current.PublishDate)
		}

		artID := tralbum.ArtID.String()
		if artID == "" || artID == "0" {
			artID = tralbum.Current.ArtID.String()
		}
		if artID != "" && artID != "0" {
			release.ArtworkURL = BuildArtworkURL(artID)
		}

		// Find the track in trackinfo
		if len(tralbum.Trackinfo) > 0 {
			t := tralbum.Trackinfo[0]
			streamURL := ""
			if t.File != nil {
				if url, ok := t.File["mp3-128"]; ok {
					streamURL = url
				} else {
					for _, url := range t.File {
						streamURL = url
						break
					}
				}
			}

			lyrics := CleanLyrics(t.Lyrics)
			if lyrics == "" {
				lyrics = CleanLyrics(tralbum.Current.Lyrics)
			}

			trackArtist := strings.TrimSpace(t.Artist)
			if trackArtist == "" {
				trackArtist = release.Artist
			}

			trackNum := t.TrackNum
			if trackNum <= 0 {
				trackNum = tralbum.Current.TrackNumber
			}
			if trackNum <= 0 {
				trackNum = 1
			}

			track = Track{
				Number:    trackNum,
				Title:     strings.TrimSpace(t.Title),
				Artist:    trackArtist,
				Album:     release.Album,
				Duration:  time.Duration(t.Duration * float64(time.Second)),
				StreamURL: streamURL,
				Lyrics:    lyrics,
			}
		} else {
			// Fallback from TralbumCurrent
			trackNum := tralbum.Current.TrackNumber
			if trackNum <= 0 {
				trackNum = 1
			}
			track = Track{
				Number: trackNum,
				Title:  strings.TrimSpace(tralbum.Current.Title),
				Artist: release.Artist,
				Album:  release.Album,
				Lyrics: CleanLyrics(tralbum.Current.Lyrics),
			}
		}
	}

	// Schema fallback
	if schema != nil {
		if release.Artist == "" && schema.ByArtist.Name != "" {
			release.Artist = schema.ByArtist.Name
			release.AlbumArtist = release.Artist
		}
		if track.Title == "" && schema.Name != "" {
			track.Title = schema.Name
		}
		if release.Album == "" {
			release.Album = track.Title
		}
		if release.ReleaseDate.IsZero() && schema.DatePublished != "" {
			release.ReleaseDate = ParseReleaseDate(schema.DatePublished)
		}
		if release.ArtworkURL == "" {
			if imgStr, ok := schema.Image.(string); ok && imgStr != "" {
				release.ArtworkURL = UpgradeArtworkQuality(imgStr)
			}
		}
	}

	// Meta tag fallback
	if release.Artist == "" {
		if siteName, ok := meta["og:site_name"]; ok {
			release.Artist = siteName
			release.AlbumArtist = siteName
		}
	}
	if track.Title == "" {
		if title, ok := meta["og:title"]; ok {
			track.Title = title
		}
	}
	if release.Album == "" {
		release.Album = track.Title
	}
	if release.ArtworkURL == "" {
		if img, ok := meta["og:image"]; ok && img != "" {
			release.ArtworkURL = UpgradeArtworkQuality(img)
		} else if img, ok := meta["image_src"]; ok && img != "" {
			release.ArtworkURL = UpgradeArtworkQuality(img)
		}
	}

	if track.Artist == "" {
		track.Artist = release.Artist
	}
	if track.Album == "" {
		track.Album = release.Album
	}
	if track.Number <= 0 {
		track.Number = 1
	}

	if release.Artist == "" && track.Title == "" {
		return nil, fmt.Errorf("%w: failed to extract track details from %s", ErrParseFailed, pageURL)
	}

	release.Tracks = []Track{track}
	return release, nil
}
