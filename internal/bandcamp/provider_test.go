package bandcamp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestProviderCanHandle(t *testing.T) {
	p := NewProvider(nil)

	tests := []struct {
		urlStr string
		expect bool
	}{
		{"https://tokyoroseofficial.bandcamp.com", true},
		{"https://tokyoroseofficial.bandcamp.com/music", true},
		{"https://tokyoroseofficial.bandcamp.com/album/killwave", true},
		{"https://tokyoroseofficial.bandcamp.com/track/showdown", true},
		{"https://customdomain.com/album/special", true},
		{"https://customdomain.com/track/special", true},
		{"https://soundcloud.com/artist/track", false},
		{"https://youtube.com/watch?v=123", false},
	}

	for _, tc := range tests {
		u, err := url.Parse(tc.urlStr)
		if err != nil {
			t.Fatalf("failed parsing test url %s: %v", tc.urlStr, err)
		}
		if got := p.CanHandle(u); got != tc.expect {
			t.Errorf("CanHandle(%s) = %v, expected %v", tc.urlStr, got, tc.expect)
		}
	}
}

func TestProviderResolveArtistMusicSubpage(t *testing.T) {
	var requestedPaths []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPaths = append(requestedPaths, r.URL.Path)
		switch r.URL.Path {
		case "/music":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`
			<!DOCTYPE html>
			<html>
			<body>
				<ol id="music-grid" class="music-grid" data-client-items="[{&quot;page_url&quot;:&quot;/album/client-album&quot;,&quot;type&quot;:&quot;album&quot;}]">
					<li class="music-grid-item"><a href="/album/grid-album"><p class="title">Grid</p></a></li>
				</ol>
			</body>
			</html>
			`))
		case "/album/grid-album":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`
			<!DOCTYPE html><html><body>
			<script>
			var TralbumData = {
				"artist": "Test Artist",
				"current": {"title": "Grid Album", "type": "album"},
				"trackinfo": [{"track_num": 1, "title": "Track 1", "file": {"mp3-128": "https://stream.mp3"}}]
			};
			</script>
			</body></html>
			`))
		case "/album/client-album":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`
			<!DOCTYPE html><html><body>
			<script>
			var TralbumData = {
				"artist": "Test Artist",
				"current": {"title": "Client Album", "type": "album"},
				"trackinfo": [{"track_num": 1, "title": "Track 2", "file": {"mp3-128": "https://stream.mp3"}}]
			};
			</script>
			</body></html>
			`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(5*time.Second, "TestAgent")
	p := NewProvider(client)

	// Custom transport to redirect bandcamp host requests to local test server
	serverURL, _ := url.Parse(server.URL)
	p.Client.HTTPClient.Transport = &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return nil, nil
		},
	}
	// Re-route requests using a custom RoundTripper
	p.Client.HTTPClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		clone := req.Clone(req.Context())
		clone.URL.Scheme = serverURL.Scheme
		clone.URL.Host = serverURL.Host
		return http.DefaultTransport.RoundTrip(clone)
	})

	artistURL, _ := url.Parse("https://testartist.bandcamp.com")
	releases, err := p.Resolve(context.Background(), artistURL)
	if err != nil {
		t.Fatalf("unexpected error resolving artist: %v", err)
	}

	if len(releases) != 2 {
		t.Fatalf("expected 2 releases, got %d", len(releases))
	}

	// Verify that /music was requested first
	if len(requestedPaths) == 0 || requestedPaths[0] != "/music" {
		t.Errorf("expected first request to be /music, got paths: %v", requestedPaths)
	}

	titles := []string{releases[0].Album, releases[1].Album}
	expected := "Grid Album, Client Album"
	got := strings.Join(titles, ", ")
	if got != expected {
		t.Errorf("expected albums %q, got %q", expected, got)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
