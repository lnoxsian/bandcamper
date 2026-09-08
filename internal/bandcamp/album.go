package bandcamp

import (
	"fmt"
	"strings"
	"time"
)

// ParseAlbum parses an album page HTML and extracts a full Release model.
func ParseAlbum(htmlContent, pageURL string) (*Release, error) {
	tralbum, _ := ExtractTralbumData(htmlContent)
	meta := ExtractMetaTags(htmlContent)
	schema, _ := ExtractLDJson(htmlContent)

	release := &Release{
		URL: pageURL,
	}

	if tralbum != nil {
		release.Artist = strings.TrimSpace(tralbum.Artist)
		if release.Artist == "" {
			release.Artist = strings.TrimSpace(tralbum.Current.Artist)
		}

		release.Album = strings.TrimSpace(tralbum.Current.Title)
		if release.Album == "" {
			release.Album = strings.TrimSpace(tralbum.AlbumTitle)
		}

		release.AlbumArtist = release.Artist
		release.Description = CleanDescription(tralbum.Current.About)
		release.ReleaseDate = ParseReleaseDate(tralbum.Current.ReleaseDate)
		if release.ReleaseDate.IsZero() {
			release.ReleaseDate = ParseReleaseDate(tralbum.Current.PublishDate)
		}

		// Artwork
		artID := tralbum.ArtID.String()
		if artID == "" || artID == "0" {
			artID = tralbum.Current.ArtID.String()
		}
		if artID != "" && artID != "0" {
			release.ArtworkURL = BuildArtworkURL(artID)
		}

		// Parse tracks
		for idx, t := range tralbum.Trackinfo {
			trackNum := t.TrackNum
			if trackNum <= 0 {
				trackNum = idx + 1
			}

			trackArtist := strings.TrimSpace(t.Artist)
			if trackArtist == "" {
				trackArtist = release.Artist
			}

			streamURL := ""
			if t.File != nil {
				// Prefer mp3-128 stream
				if url, ok := t.File["mp3-128"]; ok {
					streamURL = url
				} else {
					for _, url := range t.File {
						streamURL = url
						break
					}
				}
			}

			track := Track{
				Number:    trackNum,
				Title:     strings.TrimSpace(t.Title),
				Artist:    trackArtist,
				Album:     release.Album,
				Duration:  time.Duration(t.Duration * float64(time.Second)),
				StreamURL: streamURL,
				Lyrics:    CleanLyrics(t.Lyrics),
			}

			release.Tracks = append(release.Tracks, track)
		}
	}

	// Schema.org fallback / enrichment
	if schema != nil {
		if release.Artist == "" && schema.ByArtist.Name != "" {
			release.Artist = schema.ByArtist.Name
			release.AlbumArtist = release.Artist
		}
		if release.Album == "" && schema.Name != "" {
			release.Album = schema.Name
		}
		if release.ReleaseDate.IsZero() && schema.DatePublished != "" {
			release.ReleaseDate = ParseReleaseDate(schema.DatePublished)
		}
		if release.ArtworkURL == "" {
			if imgStr, ok := schema.Image.(string); ok && imgStr != "" {
				release.ArtworkURL = UpgradeArtworkQuality(imgStr)
			}
		}
		if release.Description == "" && schema.Description != "" {
			release.Description = CleanDescription(schema.Description)
		}
		if genreStr, ok := schema.Genre.(string); ok && genreStr != "" {
			release.Genre = genreStr
		}

		// If no tracks were found in TralbumData, populate from Schema.org
		if len(release.Tracks) == 0 && len(schema.Track.ItemListElement) > 0 {
			for idx, item := range schema.Track.ItemListElement {
				trackNum := item.Position
				if trackNum <= 0 {
					trackNum = idx + 1
				}
				release.Tracks = append(release.Tracks, Track{
					Number: trackNum,
					Title:  strings.TrimSpace(item.Item.Name),
					Artist: release.Artist,
					Album:  release.Album,
				})
			}
		}
	}

	// OpenGraph / Meta tag fallback
	if release.Artist == "" {
		if siteName, ok := meta["og:site_name"]; ok {
			release.Artist = siteName
			release.AlbumArtist = siteName
		}
	}
	if release.Album == "" {
		if title, ok := meta["og:title"]; ok {
			release.Album = title
		}
	}
	if release.ArtworkURL == "" {
		if img, ok := meta["og:image"]; ok && img != "" {
			release.ArtworkURL = UpgradeArtworkQuality(img)
		} else if img, ok := meta["image_src"]; ok && img != "" {
			release.ArtworkURL = UpgradeArtworkQuality(img)
		}
	}
	if release.Description == "" {
		if desc, ok := meta["og:description"]; ok {
			release.Description = CleanDescription(desc)
		}
	}

	if release.Artist == "" && release.Album == "" && len(release.Tracks) == 0 {
		return nil, fmt.Errorf("%w: failed to extract album details from %s", ErrParseFailed, pageURL)
	}

	return release, nil
}
