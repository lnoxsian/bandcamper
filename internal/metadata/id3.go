package metadata

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bogem/id3v2/v2"
	"github.com/lnoxsian/bandcamper/internal/bandcamp"
)

// EmbedMetadata writes ID3v2 tags into the target MP3 file.
func EmbedMetadata(mp3Path string, meta *Metadata) error {
	if meta == nil {
		return nil
	}

	tag, err := id3v2.Open(mp3Path, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("%w: failed to open mp3 file %q: %v", bandcamp.ErrMetadataFailed, mp3Path, err)
	}
	defer tag.Close()

	tag.SetDefaultEncoding(id3v2.EncodingUTF8)

	if meta.Title != "" {
		tag.SetTitle(meta.Title)
	}
	if meta.Artist != "" {
		tag.SetArtist(meta.Artist)
	}
	if meta.Album != "" {
		tag.SetAlbum(meta.Album)
	}
	if meta.Genre != "" {
		tag.SetGenre(meta.Genre)
	}
	if meta.Year > 0 {
		tag.SetYear(fmt.Sprintf("%d", meta.Year))
	}

	// Album Artist (TPE2)
	if meta.AlbumArtist != "" {
		tag.AddTextFrame("TPE2", id3v2.EncodingUTF8, meta.AlbumArtist)
	}

	// Track Number and Total (TRCK)
	if meta.TrackNumber > 0 {
		trckStr := fmt.Sprintf("%d", meta.TrackNumber)
		if meta.TrackTotal > 0 {
			trckStr = fmt.Sprintf("%d/%d", meta.TrackNumber, meta.TrackTotal)
		}
		tag.AddTextFrame("TRCK", id3v2.EncodingUTF8, trckStr)
	}

	// Lyrics (USLT)
	if meta.Lyrics != "" {
		tag.AddUnsynchronisedLyricsFrame(id3v2.UnsynchronisedLyricsFrame{
			Encoding:          id3v2.EncodingUTF8,
			Language:          "eng",
			ContentDescriptor: "",
			Lyrics:            meta.Lyrics,
		})
	}

	// Comment (COMM)
	if meta.Comment != "" {
		tag.AddCommentFrame(id3v2.CommentFrame{
			Encoding:    id3v2.EncodingUTF8,
			Language:    "eng",
			Description: "",
			Text:        meta.Comment,
		})
	}

	// Front Cover Artwork (APIC)
	if len(meta.Artwork) > 0 {
		mime := meta.ArtworkMIME
		if mime == "" {
			mime = "image/jpeg"
		}
		tag.AddAttachedPicture(id3v2.PictureFrame{
			Encoding:    id3v2.EncodingUTF8,
			MimeType:    mime,
			PictureType: id3v2.PTFrontCover,
			Description: "Front Cover",
			Picture:     meta.Artwork,
		})
	}

	if err := tag.Save(); err != nil {
		return fmt.Errorf("%w: failed to save id3 tags for %q: %v", bandcamp.ErrMetadataFailed, mp3Path, err)
	}

	return nil
}

// ReadMetadata reads ID3v2 tags from an MP3 file into a Metadata struct.
func ReadMetadata(mp3Path string) (*Metadata, error) {
	tag, err := id3v2.Open(mp3Path, id3v2.Options{Parse: true})
	if err != nil {
		return nil, fmt.Errorf("failed to open mp3 for reading tags: %w", err)
	}
	defer tag.Close()

	meta := &Metadata{
		Title:   tag.Title(),
		Artist:  tag.Artist(),
		Album:   tag.Album(),
		Genre:   tag.Genre(),
		Comment: "",
	}

	if yearStr := tag.Year(); yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil {
			meta.Year = y
		}
	}

	// Read TPE2
	if frames := tag.GetFrames("TPE2"); len(frames) > 0 {
		if tf, ok := frames[0].(id3v2.TextFrame); ok {
			meta.AlbumArtist = tf.Text
		}
	}

	// Read TRCK
	if frames := tag.GetFrames("TRCK"); len(frames) > 0 {
		if tf, ok := frames[0].(id3v2.TextFrame); ok {
			parts := strings.Split(tf.Text, "/")
			if len(parts) > 0 {
				meta.TrackNumber, _ = strconv.Atoi(parts[0])
			}
			if len(parts) > 1 {
				meta.TrackTotal, _ = strconv.Atoi(parts[1])
			}
		}
	}

	// Read USLT
	if frames := tag.GetFrames("USLT"); len(frames) > 0 {
		if uf, ok := frames[0].(id3v2.UnsynchronisedLyricsFrame); ok {
			meta.Lyrics = uf.Lyrics
		}
	}

	// Read APIC
	if frames := tag.GetFrames("APIC"); len(frames) > 0 {
		if pic, ok := frames[0].(id3v2.PictureFrame); ok {
			meta.Artwork = pic.Picture
			meta.ArtworkMIME = pic.MimeType
		}
	}

	return meta, nil
}
