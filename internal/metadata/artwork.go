package metadata

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/lnoxsian/bandcamper/internal/bandcamp"
)

// ArtworkData holds downloaded image bytes and MIME type.
type ArtworkData struct {
	Data     []byte
	MIMEType string
}

// ArtworkCache provides synchronized caching of release artwork.
type ArtworkCache struct {
	mu    sync.RWMutex
	cache map[string]*ArtworkData
}

// NewArtworkCache creates a new synchronized artwork cache.
func NewArtworkCache() *ArtworkCache {
	return &ArtworkCache{
		cache: make(map[string]*ArtworkData),
	}
}

// Get returns cached artwork if present.
func (c *ArtworkCache) Get(artworkURL string) (*ArtworkData, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	art, ok := c.cache[artworkURL]
	return art, ok
}

// Set stores artwork data in the cache.
func (c *ArtworkCache) Set(artworkURL string, data *ArtworkData) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[artworkURL] = data
}

// DetectImageMIME identifies the image MIME type and verifies image integrity.
func DetectImageMIME(data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("empty image data")
	}

	// Validate decoding image header
	_, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		// Fallback to http.DetectContentType
		detected := http.DetectContentType(data)
		if detected == "image/jpeg" || detected == "image/png" || detected == "image/gif" || detected == "image/webp" {
			return detected, nil
		}
		return "", fmt.Errorf("unrecognized or corrupted image format: %w", err)
	}

	switch format {
	case "jpeg":
		return "image/jpeg", nil
	case "png":
		return "image/png", nil
	case "gif":
		return "image/gif", nil
	default:
		return "image/" + format, nil
	}
}

// FetchArtwork downloads and caches artwork using the client.
func FetchArtwork(ctx context.Context, client *bandcamp.Client, cache *ArtworkCache, artworkURL string) (*ArtworkData, error) {
	if artworkURL == "" {
		return nil, nil
	}

	if cache != nil {
		if cached, ok := cache.Get(artworkURL); ok {
			return cached, nil
		}
	}

	// Fetch max 20MB for artwork
	data, err := client.FetchBytes(ctx, artworkURL, 20*1024*1024)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to fetch artwork from %s: %v", bandcamp.ErrArtworkFailed, artworkURL, err)
	}

	mimeType, err := DetectImageMIME(data)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid image from %s: %v", bandcamp.ErrArtworkFailed, artworkURL, err)
	}

	art := &ArtworkData{
		Data:     data,
		MIMEType: mimeType,
	}

	if cache != nil {
		cache.Set(artworkURL, art)
	}

	return art, nil
}

// SaveCoverFile saves the artwork to the release folder as cover.jpg or cover.png.
func SaveCoverFile(outputDir string, artwork *ArtworkData) (string, error) {
	if artwork == nil || len(artwork.Data) == 0 {
		return "", nil
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory %q: %w", outputDir, err)
	}

	ext := ".jpg"
	if artwork.MIMEType == "image/png" {
		ext = ".png"
	}

	coverPath := filepath.Join(outputDir, "cover"+ext)
	if err := os.WriteFile(coverPath, artwork.Data, 0644); err != nil {
		return "", fmt.Errorf("failed to write cover image to %q: %w", coverPath, err)
	}

	return coverPath, nil
}
