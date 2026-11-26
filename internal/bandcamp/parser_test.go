package bandcamp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseAlbum(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "tests", "fixtures", "album_sample.html"))
	if err != nil {
		t.Fatalf("failed reading fixture: %v", err)
	}

	release, err := ParseAlbum(string(content), "https://starsailor.bandcamp.com/album/cosmic-journey")
	if err != nil {
		t.Fatalf("unexpected error parsing album: %v", err)
	}

	if release.Artist != "Star Sailor" {
		t.Errorf("expected artist 'Star Sailor', got %q", release.Artist)
	}
	if release.Album != "Cosmic Journey" {
		t.Errorf("expected album 'Cosmic Journey', got %q", release.Album)
	}
	if len(release.Tracks) != 2 {
		t.Fatalf("expected 2 tracks, got %d", len(release.Tracks))
	}

	// Track 1
	t1 := release.Tracks[0]
	if t1.Number != 1 {
		t.Errorf("expected track 1, got %d", t1.Number)
	}
	if t1.Title != "Starlight Voyage" {
		t.Errorf("expected title 'Starlight Voyage', got %q", t1.Title)
	}
	if t1.Artist != "Star Sailor" {
		t.Errorf("expected inherited artist 'Star Sailor', got %q", t1.Artist)
	}
	if t1.StreamURL != "https://t4.bcbits.com/stream/track1.mp3" {
		t.Errorf("unexpected stream URL: %q", t1.StreamURL)
	}
	if !strings.Contains(t1.Lyrics, "Floating in the starlight") {
		t.Errorf("expected lyrics, got %q", t1.Lyrics)
	}

	// Track 2
	t2 := release.Tracks[1]
	if t2.Number != 2 {
		t.Errorf("expected track 2, got %d", t2.Number)
	}
	if t2.Artist != "Star Sailor feat. Luna" {
		t.Errorf("expected track artist 'Star Sailor feat. Luna', got %q", t2.Artist)
	}

	// Artwork
	if release.ArtworkURL != "https://f4.bcbits.com/img/a1234567890_10.jpg" {
		t.Errorf("unexpected artwork URL: %q", release.ArtworkURL)
	}

	// Date
	if release.ReleaseDate.Year() != 2024 || release.ReleaseDate.Month() != time.March || release.ReleaseDate.Day() != 15 {
		t.Errorf("unexpected release date: %v", release.ReleaseDate)
	}
}

func TestParseTrack(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "tests", "fixtures", "track_sample.html"))
	if err != nil {
		t.Fatalf("failed reading fixture: %v", err)
	}

	release, err := ParseTrack(string(content), "https://ambientwave.bandcamp.com/track/solo-echo")
	if err != nil {
		t.Fatalf("unexpected error parsing track: %v", err)
	}

	if release.Artist != "Ambient Wave" {
		t.Errorf("expected artist 'Ambient Wave', got %q", release.Artist)
	}
	if len(release.Tracks) != 1 {
		t.Fatalf("expected 1 track, got %d", len(release.Tracks))
	}

	tr := release.Tracks[0]
	if tr.Title != "Solo Echo" {
		t.Errorf("expected title 'Solo Echo', got %q", tr.Title)
	}
	if tr.StreamURL != "https://t4.bcbits.com/stream/single.mp3" {
		t.Errorf("unexpected stream URL: %q", tr.StreamURL)
	}
	if !strings.Contains(tr.Lyrics, "Echoes in the quiet room") {
		t.Errorf("expected lyrics in track, got %q", tr.Lyrics)
	}
}

func TestParseArtistDiscography(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "tests", "fixtures", "artist_sample.html"))
	if err != nil {
		t.Fatalf("failed reading fixture: %v", err)
	}

	urls, err := ParseArtistDiscography(string(content), "https://starsailor.bandcamp.com")
	if err != nil {
		t.Fatalf("unexpected error parsing artist discography: %v", err)
	}

	if len(urls) != 3 {
		t.Fatalf("expected 3 release URLs, got %d: %v", len(urls), urls)
	}

	expected := []string{
		"https://starsailor.bandcamp.com/album/cosmic-journey",
		"https://starsailor.bandcamp.com/album/solar-flares",
		"https://starsailor.bandcamp.com/track/deep-space-single",
	}

	for i, exp := range expected {
		if urls[i] != exp {
			t.Errorf("index %d: expected %q, got %q", i, exp, urls[i])
		}
	}
}

func TestParseArtistDiscographyWithDataClientItems(t *testing.T) {
	htmlContent := `
	<!DOCTYPE html>
	<html>
	<body>
		<ol id="music-grid" class="music-grid" data-client-items="[{&quot;art_id&quot;:100,&quot;page_url&quot;:&quot;/album/client-album&quot;,&quot;title&quot;:&quot;Client Album&quot;,&quot;type&quot;:&quot;album&quot;},{&quot;art_id&quot;:101,&quot;page_url&quot;:&quot;/track/client-track&quot;,&quot;title&quot;:&quot;Client Track&quot;,&quot;type&quot;:&quot;track&quot;}]">
			<li class="music-grid-item" data-item-id="album-1">
				<a href="/album/grid-album">
					<p class="title">Grid Album</p>
				</a>
			</li>
			<li class="music-grid-item" data-item-id="track-1">
				<a href="/track/grid-track">
					<p class="title">Grid Track</p>
				</a>
			</li>
		</ol>
	</body>
	</html>
	`

	urls, err := ParseArtistDiscography(htmlContent, "https://tokyoroseofficial.bandcamp.com/music")
	if err != nil {
		t.Fatalf("unexpected error parsing discography: %v", err)
	}

	if len(urls) != 4 {
		t.Fatalf("expected 4 releases (2 grid + 2 client items), got %d: %v", len(urls), urls)
	}

	expected := []string{
		"https://tokyoroseofficial.bandcamp.com/album/grid-album",
		"https://tokyoroseofficial.bandcamp.com/track/grid-track",
		"https://tokyoroseofficial.bandcamp.com/album/client-album",
		"https://tokyoroseofficial.bandcamp.com/track/client-track",
	}

	for i, exp := range expected {
		if urls[i] != exp {
			t.Errorf("index %d: expected %q, got %q", i, exp, urls[i])
		}
	}
}

func TestParseArtistDiscographyFeaturedAlbumExcludesTrackTable(t *testing.T) {
	htmlContent := `
	<!DOCTYPE html>
	<html>
	<body>
		<div id="name-section">
			<h2 class="trackTitle"><a href="/album/featured-album">Featured Album</a></h2>
		</div>
		<table id="track_table" class="track_list">
			<tr class="track_row_view">
				<td><a href="/track/track-one">Track One</a></td>
			</tr>
			<tr class="track_row_view">
				<td><a href="/track/track-two">Track Two</a></td>
			</tr>
		</table>
		<div id="discography">
			<ul class="discography">
				<li><a href="/album/sidebar-album">Sidebar Album</a></li>
			</ul>
		</div>
	</body>
	</html>
	`

	urls, err := ParseArtistDiscography(htmlContent, "https://tokyoroseofficial.bandcamp.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should include /album/featured-album and /album/sidebar-album, but NOT /track/track-one or /track/track-two
	if len(urls) != 2 {
		t.Fatalf("expected 2 releases (featured album + sidebar album, excluding tracklist), got %d: %v", len(urls), urls)
	}

	expected := []string{
		"https://tokyoroseofficial.bandcamp.com/album/featured-album",
		"https://tokyoroseofficial.bandcamp.com/album/sidebar-album",
	}

	for i, exp := range expected {
		if urls[i] != exp {
			t.Errorf("index %d: expected %q, got %q", i, exp, urls[i])
		}
	}
}

func TestParseTokyoRoseDiscographyFixture(t *testing.T) {
	content, err := os.ReadFile("/tmp/tokyorose.html")
	if err != nil {
		t.Skip("skipping /tmp/tokyorose.html test: fixture not present")
	}

	urls, err := ParseArtistDiscography(string(content), "https://tokyoroseofficial.bandcamp.com/music")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(urls) != 29 {
		t.Fatalf("expected 29 releases for Tokyo Rose, got %d: %v", len(urls), urls)
	}
}

func TestDataTralbumAttributeParsing(t *testing.T) {
	htmlWithAttr := `
	<!DOCTYPE html>
	<html>
	<body>
	<div id="pagedata" data-tralbum="{&quot;artist&quot;:&quot;Unicode Ärtist 🎵&quot;,&quot;current&quot;:{&quot;title&quot;:&quot;Special EP&quot;,&quot;release_date&quot;:&quot;2024-05-01&quot;},&quot;trackinfo&quot;:[{&quot;track_num&quot;:1,&quot;title&quot;:&quot;T1&quot;,&quot;file&quot;:{&quot;mp3-128&quot;:&quot;https://stream.mp3&quot;}}]}"></div>
	</body>
	</html>
	`
	release, err := ParseAlbum(htmlWithAttr, "https://example.bandcamp.com/album/special-ep")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if release.Artist != "Unicode Ärtist 🎵" {
		t.Errorf("expected unicode artist, got %q", release.Artist)
	}
	if release.Album != "Special EP" {
		t.Errorf("expected 'Special EP', got %q", release.Album)
	}
}
