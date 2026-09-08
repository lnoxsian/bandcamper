package metadata

import (
	"github.com/lnoxsian/bandcamper/internal/bandcamp"
)

// Metadata represents the standardized audio metadata to embed into files.
type Metadata struct {
	Title       string
	Artist      string
	Album       string
	AlbumArtist string
	TrackNumber int
	TrackTotal  int
	Year        int
	Genre       string
	Lyrics      string
	Comment     string
	Artwork     []byte
	ArtworkMIME string
}

// FromReleaseAndTrack creates a Metadata instance from Release and Track models.
func FromReleaseAndTrack(rel *bandcamp.Release, tr *bandcamp.Track, trackTotal int, artwork []byte, artworkMIME string) *Metadata {
	year := 0
	if !rel.ReleaseDate.IsZero() {
		year = rel.ReleaseDate.Year()
	}

	artist := tr.Artist
	if artist == "" {
		artist = rel.Artist
	}

	albumArtist := rel.AlbumArtist
	if albumArtist == "" {
		albumArtist = rel.Artist
	}

	comment := ""
	if rel.URL != "" {
		comment = "Downloaded from " + rel.URL
	}

	return &Metadata{
		Title:       tr.Title,
		Artist:      artist,
		Album:       tr.Album,
		AlbumArtist: albumArtist,
		TrackNumber: tr.Number,
		TrackTotal:  trackTotal,
		Year:        year,
		Genre:       rel.Genre,
		Lyrics:      tr.Lyrics,
		Comment:     comment,
		Artwork:     artwork,
		ArtworkMIME: artworkMIME,
	}
}
