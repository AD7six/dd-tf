package utils

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// DatadogHTTPClient wraps an HTTP client with Datadog API credentials, a concurrency limiter,
// and coordinated retry/pausing on 429s.
type DatadogHTTPClient struct {
	APIKey         string
	AppKey         string
	UnderlyingHTTP *http.Client

	// concurrency limiter
	sem chan struct{}

	// max retries for errors (including 5xx) and 429s
	retries int

	// If we receive a 429 all http requests wait until pauseUntil; if requests
	// aren't all subject to the same api rate limit (not all to the same api
	// endpoint) this will be a little conservative - but otherwise prevents
	// continuing to make requests we know will fail.
	pause      sync.Mutex
	pauseUntil time.Time
}

const (
	defaultMaxConcurrency = 8
	defaultRetries        = 3
	defaultHTTPTimeout    = 60 * time.Second
)

var (
	sharedOnce   sync.Once
	sharedClient *DatadogHTTPClient
)

// GetHTTPClient returns a shared client instance to ensure concurrency limiting and 429 pauses
// are coordinated across all requests in this process. If we want in the future
// to have multiple clients (for different api endpoints, with separate rate
// limits) we can add that later.
func GetHTTPClient(settings *Settings) *DatadogHTTPClient {
	sharedOnce.Do(func() {
		sharedClient = newClient(settings.APIKey, settings.AppKey, defaultMaxConcurrency, defaultRetries, settings.HTTPTimeout)
	})
	return sharedClient
}

func newClient(apiKey, appKey string, maxConcurrent, retries int, timeout time.Duration) *DatadogHTTPClient {
	if maxConcurrent <= 0 {
		maxConcurrent = defaultMaxConcurrency
	}
	if retries <= 0 {
		retries = defaultRetries
	}
	if timeout <= 0 {
		timeout = defaultHTTPTimeout
	}
	return &DatadogHTTPClient{
		APIKey:         apiKey,
		AppKey:         appKey,
		UnderlyingHTTP: &http.Client{Timeout: timeout},
		sem:            make(chan struct{}, maxConcurrent),
		retries:        retries,
	}
}

// Get performs a GET request with retry logic and context support.
// Uses context.Background() for backward compatibility.
func (c *DatadogHTTPClient) Get(url string) (*http.Response, error) {
	return c.GetWithContext(context.Background(), url)
}

// GetWithContext performs a GET request with the provided context for cancellation/timeout.
func (c *DatadogHTTPClient) GetWithContext(ctx context.Context, url string) (*http.Response, error) {
	// Acquire concurrency slot
	c.sem <- struct{}{}
	defer func() { <-c.sem }()

	// Retry loop
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		// If globally paused due to 429, wait it out
		c.waitIfPaused()

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("DD-API-KEY", c.APIKey)
		req.Header.Set("DD-APPLICATION-KEY", c.AppKey)

		resp, err := c.UnderlyingHTTP.Do(req)
		if err != nil {
			lastErr = err
			if attempt < c.retries {
				time.Sleep(backoffDuration(attempt))
				continue
			}
			return nil, lastErr
		}

		// Handle 429: set global pause, then retry after waiting
		if resp.StatusCode == http.StatusTooManyRequests {
			// Determine wait duration from Retry-After (seconds) or fall back to 1s
			wait := parseRetryAfter(resp)
			// Close body before sleeping/retrying
			if err := resp.Body.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", err)
			}

			// Set global pause
			c.setPause(wait)

			if attempt < c.retries {
				// Sleep the same period locally before retrying this request
				time.Sleep(wait)
				continue
			}
			return nil, &rateLimitedError{after: wait}
		}

		// Retry transient server errors (5xx). Do not retry other 4xx.
		if resp.StatusCode >= 500 {
			if attempt < c.retries {
				if err := resp.Body.Close(); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to close response body: %v\n", err)
				}
				time.Sleep(backoffDuration(attempt))
				continue
			}
			// return last response to caller for error handling
			return resp, nil
		}

		return resp, nil
	}

	return nil, lastErr
}

// Backoff: 500ms, 1s, 2s, capped
func backoffDuration(attempt int) time.Duration {
	d := 500 * time.Millisecond
	for i := 0; i < attempt; i++ {
		d *= 2
		if d > 5*time.Second {
			d = 5 * time.Second
			break
		}
	}
	return d
}

func (c *DatadogHTTPClient) waitIfPaused() {
	for {
		c.pause.Lock()
		now := time.Now()
		until := c.pauseUntil
		c.pause.Unlock()
		if until.IsZero() || now.After(until) || now.Equal(until) {
			return
		}
		time.Sleep(time.Until(until))
	}
}

func (c *DatadogHTTPClient) setPause(d time.Duration) {
	if d <= 0 {
		d = time.Second
	}
	c.pause.Lock()
	// If there is already a longer pause in place, keep it
	proposed := time.Now().Add(d)
	if proposed.After(c.pauseUntil) {
		c.pauseUntil = proposed
	}
	c.pause.Unlock()
}

func parseRetryAfter(resp *http.Response) time.Duration {
	if resp == nil {
		return time.Second
	}
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		if secs, err := strconv.Atoi(ra); err == nil && secs >= 0 {
			return time.Duration(secs) * time.Second
		}
		// Could be a HTTP date; ignore for simplicity
	}
	return time.Second
}

type rateLimitedError struct {
	after time.Duration
}

func (e *rateLimitedError) Error() string {
	return fmt.Sprintf("rate limited by server (retry after %v)", e.after)
}
