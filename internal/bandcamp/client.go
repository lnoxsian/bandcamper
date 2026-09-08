package bandcamp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client wraps an http.Client with Bandcamp-specific headers, timeout, and context handling.
type Client struct {
	HTTPClient *http.Client
	UserAgent  string
}

// NewClient creates a new configured Bandcamp HTTP Client.
func NewClient(timeout time.Duration, userAgent string) *Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	if userAgent == "" {
		userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
	}

	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
	}

	return &Client{
		HTTPClient: &http.Client{
			Transport: transport,
			Timeout:   timeout,
		},
		UserAgent: userAgent,
	}
}

// Get performs an HTTP GET request with default headers and context cancellation support.
func (c *Client) Get(ctx context.Context, targetURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %q failed: %w", targetURL, err)
	}

	if resp.StatusCode >= 400 {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP %d (%s) for %s", resp.StatusCode, http.StatusText(resp.StatusCode), targetURL)
	}

	return resp, nil
}

// FetchBytes reads the entire response body into memory (with a max limit).
func (c *Client) FetchBytes(ctx context.Context, targetURL string, maxBytes int64) ([]byte, error) {
	resp, err := c.Get(ctx, targetURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var reader io.Reader = resp.Body
	if maxBytes > 0 {
		reader = io.LimitReader(resp.Body, maxBytes)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed reading response body from %q: %w", targetURL, err)
	}

	return data, nil
}

// FetchString reads the response body as a string.
func (c *Client) FetchString(ctx context.Context, targetURL string, maxBytes int64) (string, error) {
	bytes, err := c.FetchBytes(ctx, targetURL, maxBytes)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
