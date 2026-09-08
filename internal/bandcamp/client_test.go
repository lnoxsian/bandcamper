package bandcamp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientGet(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "CustomAgent/1.0" {
			t.Errorf("unexpected User-Agent: %s", r.Header.Get("User-Agent"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello bandcamp"))
	}))
	defer ts.Close()

	client := NewClient(5*time.Second, "CustomAgent/1.0")
	body, err := client.FetchString(context.Background(), ts.URL, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body != "hello bandcamp" {
		t.Errorf("expected 'hello bandcamp', got %q", body)
	}
}

func TestClientContextCancellation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := NewClient(5*time.Second, "TestAgent")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.Get(ctx, ts.URL)
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
}

func TestClientErrorStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	client := NewClient(5*time.Second, "TestAgent")
	_, err := client.Get(context.Background(), ts.URL)
	if err == nil {
		t.Fatalf("expected 404 error, got nil")
	}
}
