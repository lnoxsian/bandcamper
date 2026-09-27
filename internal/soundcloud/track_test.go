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

func TestSelectBestTranscoding(t *testing.T) {
	media := MediaInfo{
		Transcodings: []Transcoding{
			{
				URL: "http://api.example.com/hls-aac",
				Format: TranscodingFormat{
					Protocol: "hls",
					MimeType: "audio/mp4; codecs=\"mp4a.40.2\"",
				},
			},
			{
				URL: "http://api.example.com/prog-mp3",
				Format: TranscodingFormat{
					Protocol: "progressive",
					MimeType: "audio/mpeg",
				},
			},
			{
				URL: "http://api.example.com/hls-mp3",
				Format: TranscodingFormat{
					Protocol: "hls",
					MimeType: "audio/mpeg",
				},
			},
		},
	}

	best, err := SelectBestTranscoding(media)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if best.URL != "http://api.example.com/prog-mp3" {
		t.Errorf("expected progressive MP3, got %s", best.URL)
	}

	// Empty transcodings
	_, err = SelectBestTranscoding(MediaInfo{})
	if err == nil {
		t.Errorf("expected error for empty transcodings")
	}

	// Only HLS available
	mediaHLS := MediaInfo{
		Transcodings: []Transcoding{
			{
				URL: "http://api.example.com/only-hls",
				Format: TranscodingFormat{
					Protocol: "hls",
					MimeType: "audio/mpeg",
				},
			},
		},
	}
	bestHLS, err := SelectBestTranscoding(mediaHLS)
	if err != nil {
		t.Fatalf("unexpected error for HLS: %v", err)
	}
	if bestHLS.URL != "http://api.example.com/only-hls" {
		t.Errorf("expected HLS stream, got %s", bestHLS.URL)
	}
}

func TestResolveTrack(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/media/stream":
			q := r.URL.Query()
			if q.Get("client_id") != "test_client_id_abc" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(MediaStreamResponse{
				URL: "http://cdn.example.com/final_audio.mp3",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	html := strings.ReplaceAll(`<!DOCTYPE html><html><body>
<script>window.__sc_hydration = [
  {"hydratable": "apiClient", "data": {"id": "test_client_id_abc"}},
  {
    "hydratable": "sound",
    "data": {
      "id": 999,
      "title": "Nebula",
      "user": {"username": "Astral"},
      "media": {
        "transcodings": [
          {
            "url": "SERVER_URL/media/stream",
            "format": {"protocol": "progressive", "mime_type": "audio/mpeg"}
          }
        ]
      }
    }
  }
];</script></body></html>`, "SERVER_URL", server.URL)

	client := NewClient(5*time.Second, "TestAgent")
	rel, err := ResolveTrack(context.Background(), client, html, "https://soundcloud.com/astral/nebula")
	if err != nil {
		t.Fatalf("ResolveTrack failed: %v", err)
	}

	if rel.Artist != "Astral" {
		t.Errorf("expected Artist Astral, got %q", rel.Artist)
	}
	if rel.Album != "Nebula" {
		t.Errorf("expected Album Nebula, got %q", rel.Album)
	}
	if len(rel.Tracks) != 1 {
		t.Fatalf("expected 1 track, got %d", len(rel.Tracks))
	}
	if rel.Tracks[0].StreamURL != "http://cdn.example.com/final_audio.mp3" {
		t.Errorf("expected stream URL http://cdn.example.com/final_audio.mp3, got %q", rel.Tracks[0].StreamURL)
	}
}
