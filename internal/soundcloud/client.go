package soundcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

var scriptSrcRegex = regexp.MustCompile(`<script[^>]+src=["']([^"']+)["']`)

// Client handles HTTP interactions with SoundCloud and media stream resolution.
type Client struct {
	HTTPClient     *http.Client
	UserAgent      string
	cachedClientID string
	clientIDMu     sync.RWMutex
}

// NewClient creates a new SoundCloud client with standard timeouts and User-Agent.
func NewClient(timeout time.Duration, userAgent string) *Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	if userAgent == "" {
		userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	}

	return &Client{
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
		UserAgent: userAgent,
	}
}

// FetchString retrieves the content of a URL as a UTF-8 string up to maxBytes.
func (c *Client) FetchString(ctx context.Context, targetURL string, maxBytes int64) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed creating request for %s: %w", targetURL, err)
	}
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed for %s: %w", targetURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("%w: HTTP 404 for %s", ErrSoundCloudNotFound, targetURL)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected HTTP status %d for %s", resp.StatusCode, targetURL)
	}

	var reader io.Reader = resp.Body
	if maxBytes > 0 {
		reader = io.LimitReader(resp.Body, maxBytes)
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed reading response body for %s: %w", targetURL, err)
	}

	return string(body), nil
}

// GetOrDiscoverClientID returns the active client_id from cache, page hydration, or asset scripts.
func (c *Client) GetOrDiscoverClientID(ctx context.Context, pageHTML string, hydration []HydrationItem) string {
	c.clientIDMu.RLock()
	if c.cachedClientID != "" {
		id := c.cachedClientID
		c.clientIDMu.RUnlock()
		return id
	}
	c.clientIDMu.RUnlock()

	// 1. Try extracting from current page hydration
	if id := ExtractClientID(pageHTML, hydration); id != "" {
		c.clientIDMu.Lock()
		c.cachedClientID = id
		c.clientIDMu.Unlock()
		return id
	}

	// 2. Discover client ID by inspecting script assets referenced on the page
	scripts := scriptSrcRegex.FindAllStringSubmatch(pageHTML, -1)
	for i := len(scripts) - 1; i >= 0; i-- {
		if len(scripts[i]) > 1 {
			src := scripts[i][1]
			if strings.Contains(src, "sndcdn.com/assets/") {
				if scriptContent, err := c.FetchString(ctx, src, 2*1024*1024); err == nil {
					matches := clientIDRegex.FindStringSubmatch(scriptContent)
					if len(matches) >= 2 {
						id := matches[1]
						c.clientIDMu.Lock()
						c.cachedClientID = id
						c.clientIDMu.Unlock()
						return id
					}
				}
			}
		}
	}

	return ""
}

// ResolveMediaStreamURL queries a SoundCloud media transcoding endpoint with client_id to obtain the direct audio stream URL.
func (c *Client) ResolveMediaStreamURL(ctx context.Context, mediaTranscodingURL, clientID string) (string, error) {
	if mediaTranscodingURL == "" {
		return "", fmt.Errorf("%w: empty transcoding URL", ErrSoundCloudMediaUnavailable)
	}

	parsed, err := url.Parse(mediaTranscodingURL)
	if err != nil {
		return "", fmt.Errorf("invalid transcoding URL %q: %w", mediaTranscodingURL, err)
	}

	q := parsed.Query()
	if clientID != "" {
		q.Set("client_id", clientID)
	}
	parsed.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return "", fmt.Errorf("failed creating media resolve request: %w", err)
	}
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("media resolve request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: media stream resolve returned HTTP %d", ErrSoundCloudMediaUnavailable, resp.StatusCode)
	}

	var streamResp MediaStreamResponse
	if err := json.NewDecoder(resp.Body).Decode(&streamResp); err != nil {
		return "", fmt.Errorf("failed decoding media stream response: %w", err)
	}

	if streamResp.URL == "" {
		return "", fmt.Errorf("%w: empty direct stream URL returned", ErrSoundCloudMediaUnavailable)
	}

	return streamResp.URL, nil
}
