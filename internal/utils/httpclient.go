package utils

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// DatadogHTTPClient wraps an HTTP client with Datadog API credentials and retry logic for rate limiting.
type DatadogHTTPClient struct {
	APIKey         string
	AppKey         string
	MaxRetries     int
	UnderlyingHTTP *http.Client
}

// NewDatadogHTTPClient creates a new client with the given credentials and retry settings.
func NewDatadogHTTPClient(apiKey, appKey string, maxRetries int) *DatadogHTTPClient {
	return &DatadogHTTPClient{
		APIKey:         apiKey,
		AppKey:         appKey,
		MaxRetries:     maxRetries,
		UnderlyingHTTP: http.DefaultClient,
	}
}

// GetWithRetry performs a GET request with automatic retries on 429 responses.
// It uses the Retry-After header if present, otherwise exponential backoff.
func (c *DatadogHTTPClient) GetWithRetry(url string) (*http.Response, error) {
	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("DD-API-KEY", c.APIKey)
		req.Header.Set("DD-APPLICATION-KEY", c.AppKey)

		resp, err = c.UnderlyingHTTP.Do(req)
		if err != nil {
			lastErr = err
			// Transient network error; retry like 429s
			if attempt < c.MaxRetries {
				time.Sleep(retryAfterDelay(nil, attempt))
				continue
			}
			return nil, lastErr
		}

		if resp.StatusCode == http.StatusTooManyRequests { // 429
			// Use Retry-After if present, else exponential backoff
			wait := retryAfterDelay(resp, attempt)
			// Drain body before retry
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()

			if attempt < c.MaxRetries {
				time.Sleep(wait)
				continue
			}
			// No more retries; return error
			return nil, fmt.Errorf("rate limited (429) after %d retries", attempt)
		}

		// Non-429 response (success or other error); return as-is
		return resp, nil
	}

	return nil, lastErr
}

// retryAfterDelay returns a delay to wait before the next retry.
// If resp is non-nil and contains a valid Retry-After header (seconds), it is used.
// Otherwise it falls back to an exponential backoff: 1s, 2s, 4s, ... capped at 30s.
func retryAfterDelay(resp *http.Response, attempt int) time.Duration {
	if resp != nil {
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, err := strconv.Atoi(ra); err == nil && secs >= 0 {
				return time.Duration(secs) * time.Second
			}
		}
	}
	// Exponential backoff
	d := time.Second << attempt
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	if d < time.Second {
		d = time.Second
	}
	return d
}
