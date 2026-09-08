package downloader

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// IsTransientError checks whether an error is temporary and can be retried.
func IsTransientError(err error) bool {
	if err == nil {
		return false
	}

	// Context cancellation or deadline exceeded should NOT be retried
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// EOF during read is usually a disconnected connection
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	// Net error checks
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return true
		}
	}

	errStr := strings.ToLower(err.Error())
	if strings.Contains(errStr, "connection reset by peer") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "tls handshake timeout") ||
		strings.Contains(errStr, "temporary failure") {
		return true
	}

	// Check for transient HTTP status codes in error message (e.g. "HTTP 429", "HTTP 503")
	for _, code := range []int{
		http.StatusRequestTimeout,     // 408
		http.StatusTooManyRequests,    // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable, // 503
		http.StatusGatewayTimeout,     // 504
	} {
		if strings.Contains(errStr, strconv.Itoa(code)) {
			return true
		}
	}

	return false
}

// RetryOperation executes an operation with exponential backoff for transient failures.
func RetryOperation(ctx context.Context, maxRetries int, baseDelay time.Duration, op func() error) error {
	if maxRetries <= 0 {
		return op()
	}
	if baseDelay <= 0 {
		baseDelay = 500 * time.Millisecond
	}

	var lastErr error
	delay := baseDelay

	for attempt := 0; attempt <= maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := op()
		if err == nil {
			return nil
		}

		lastErr = err

		if !IsTransientError(err) || attempt == maxRetries {
			return err
		}

		// Wait before retrying with exponential backoff
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		delay *= 2
		// Cap max retry backoff at 15 seconds
		if delay > 15*time.Second {
			delay = 15 * time.Second
		}
	}

	return lastErr
}
