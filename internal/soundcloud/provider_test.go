package soundcloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestProviderCanHandle(t *testing.T) {
	p := NewProvider(nil)

	if p.Name() != "soundcloud" {
		t.Errorf("expected name soundcloud, got %s", p.Name())
	}

	u1, _ := url.Parse("https://soundcloud.com/artist/track")
	if !p.CanHandle(u1) {
		t.Errorf("expected true for soundcloud.com")
	}

	u2, _ := url.Parse("https://m.soundcloud.com/artist/track")
	if !p.CanHandle(u2) {
		t.Errorf("expected true for m.soundcloud.com")
	}

	u3, _ := url.Parse("https://artist.bandcamp.com/album/example")
	if p.CanHandle(u3) {
		t.Errorf("expected false for bandcamp.com")
	}

	if p.CanHandle(nil) {
		t.Errorf("expected false for nil URL")
	}
}

func TestProviderResolve(t *testing.T) {
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/artist/sample-track":
			html := strings.ReplaceAll(`<!DOCTYPE html><html><body>
<script>window.__sc_hydration = [
  {"hydratable": "apiClient", "data": {"id": "mock_id"}},
  {
    "hydratable": "sound",
    "data": {
      "id": 1234,
      "title": "Sample Track",
      "user": {"username": "Artist"},
      "media": {
        "transcodings": [
          {"url": "SERVER_URL/media/stream", "format": {"protocol": "progressive", "mime_type": "audio/mpeg"}}
        ]
      }
    }
  }
];</script></body></html>`, "SERVER_URL", serverURL)
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(html))

		case "/media/stream":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(MediaStreamResponse{
				URL: serverURL + "/audio.mp3",
			})

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	client := NewClient(5*time.Second, "TestAgent")
	p := NewProvider(client)

	// Custom URL pointing to test server path
	u, _ := url.Parse("https://soundcloud.com/artist/sample-track")

	// Inject custom HTTP client transport so requests to soundcloud.com go to test server
	p.Client.HTTPClient.Transport = &httpMockTransport{
		serverURL: serverURL,
	}

	releases, err := p.Resolve(context.Background(), u)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(releases) != 1 {
		t.Fatalf("expected 1 release, got %d", len(releases))
	}
	if releases[0].Artist != "Artist" {
		t.Errorf("expected Artist, got %q", releases[0].Artist)
	}
	if len(releases[0].Tracks) != 1 {
		t.Fatalf("expected 1 track, got %d", len(releases[0].Tracks))
	}
	if releases[0].Tracks[0].StreamURL != serverURL+"/audio.mp3" {
		t.Errorf("expected stream URL %s, got %s", serverURL+"/audio.mp3", releases[0].Tracks[0].StreamURL)
	}
}

type httpMockTransport struct {
	serverURL string
}

func (t *httpMockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Redirect soundcloud.com request to mock server
	mockURL, _ := url.Parse(t.serverURL)
	req.URL.Scheme = mockURL.Scheme
	req.URL.Host = mockURL.Host
	return http.DefaultTransport.RoundTrip(req)
}
