package soundcloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestResolvePlaylist(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/media/stream1":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(MediaStreamResponse{
				URL: "http://cdn.example.com/track1.mp3",
			})
		case "/media/stream2":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(MediaStreamResponse{
				URL: "http://cdn.example.com/track2.mp3",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	html := strings.ReplaceAll(`<!DOCTYPE html><html><body>
<script>window.__sc_hydration = [
  {"hydratable": "apiClient", "data": {"id": "mock_id"}},
  {
    "hydratable": "playlist",
    "data": {
      "id": 12345,
      "title": "Cosmic EP",
      "user": {"username": "Astral"},
      "tracks": [
        {
          "id": 1,
          "title": "Intro",
          "user": {"username": "Astral"},
          "media": {
            "transcodings": [
              {"url": "SERVER_URL/media/stream1", "format": {"protocol": "progressive", "mime_type": "audio/mpeg"}}
            ]
          }
        },
        {
          "id": 2,
          "title": "Outro",
          "user": {"username": "Astral"},
          "media": {
            "transcodings": [
              {"url": "SERVER_URL/media/stream2", "format": {"protocol": "progressive", "mime_type": "audio/mpeg"}}
            ]
          }
        }
      ]
    }
  }
];</script></body></html>`, "SERVER_URL", server.URL)

	client := NewClient(5*time.Second, "TestAgent")
	rel, err := ResolvePlaylist(context.Background(), client, html, "https://soundcloud.com/astral/sets/cosmic-ep")
	if err != nil {
		t.Fatalf("unexpected error resolving playlist: %v", err)
	}

	if rel.Album != "Cosmic EP" {
		t.Errorf("expected Album Cosmic EP, got %q", rel.Album)
	}
	if len(rel.Tracks) != 2 {
		t.Fatalf("expected 2 tracks, got %d", len(rel.Tracks))
	}
	if rel.Tracks[0].StreamURL != "http://cdn.example.com/track1.mp3" {
		t.Errorf("expected stream URL 1, got %q", rel.Tracks[0].StreamURL)
	}
	if rel.Tracks[1].StreamURL != "http://cdn.example.com/track2.mp3" {
		t.Errorf("expected stream URL 2, got %q", rel.Tracks[1].StreamURL)
	}
}
