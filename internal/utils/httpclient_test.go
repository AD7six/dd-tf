package utils

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	t.Run("creates client with default values", func(t *testing.T) {
		client := newClient("test-key", "test-app", 0, 0, 0)

		if client.APIKey != "test-key" {
			t.Errorf("APIKey = %s, want test-key", client.APIKey)
		}
		if client.AppKey != "test-app" {
			t.Errorf("AppKey = %s, want test-app", client.AppKey)
		}
		if cap(client.sem) != defaultMaxConcurrency {
			t.Errorf("sem capacity = %d, want %d", cap(client.sem), defaultMaxConcurrency)
		}
		if client.retries != defaultRetries {
			t.Errorf("retries = %d, want %d", client.retries, defaultRetries)
		}
		if client.UnderlyingHTTP.Timeout != defaultHTTPTimeout {
			t.Errorf("timeout = %v, want %v", client.UnderlyingHTTP.Timeout, defaultHTTPTimeout)
		}
	})

	t.Run("uses default values for invalid inputs", func(t *testing.T) {
		client := newClient("key", "app", -1, -1, -1*time.Second)

		if cap(client.sem) != defaultMaxConcurrency {
			t.Errorf("sem capacity = %d, want %d", cap(client.sem), defaultMaxConcurrency)
		}
		if client.retries != defaultRetries {
			t.Errorf("retries = %d, want %d", client.retries, defaultRetries)
		}
		if client.UnderlyingHTTP.Timeout != defaultHTTPTimeout {
			t.Errorf("timeout = %v, want %v", client.UnderlyingHTTP.Timeout, defaultHTTPTimeout)
		}
	})

	t.Run("accepts custom concurrency and retry values", func(t *testing.T) {
		client := newClient("key", "app", 5, 10, 30*time.Second)

		if cap(client.sem) != 5 {
			t.Errorf("sem capacity = %d, want 5", cap(client.sem))
		}
		if client.retries != 10 {
			t.Errorf("retries = %d, want 10", client.retries)
		}
		if client.UnderlyingHTTP.Timeout != 30*time.Second {
			t.Errorf("timeout = %v, want 30s", client.UnderlyingHTTP.Timeout)
		}
	})
}

func TestGetHTTPClient(t *testing.T) {
	// Reset shared client for testing
	sharedOnce = sync.Once{}

	settings := &Settings{
		APIKey:      "test-api",
		AppKey:      "test-app",
		HTTPTimeout: 30 * time.Second,
	}

	client1 := GetHTTPClient(settings)
	client2 := GetHTTPClient(settings)

	if client1 != client2 {
		t.Error("GetHTTPClient() should return same instance")
	}
}

func TestDatadogHTTPClient_Get_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers are set
		if apiKey := r.Header.Get("DD-API-KEY"); apiKey != "test-api-key" {
			t.Errorf("DD-API-KEY = %q, want %q", apiKey, "test-api-key")
		}
		if appKey := r.Header.Get("DD-APPLICATION-KEY"); appKey != "test-app-key" {
			t.Errorf("DD-APPLICATION-KEY = %q, want %q", appKey, "test-app-key")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	client := newClient("test-api-key", "test-app-key", 1, 3, 60*time.Second)
	resp, err := client.Get(server.URL)

	if err != nil {
		t.Fatalf("Get() unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "ok") {
		t.Errorf("unexpected body: %s", string(body))
	}
}

func TestDatadogHTTPClient_Get_Retries429(t *testing.T) {
	var attemptCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attemptCount, 1)
		if count <= 2 {
			// First two attempts return 429
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": "rate limited"}`))
		} else {
			// Third attempt succeeds
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok"}`))
		}
	}))
	defer server.Close()

	client := newClient("key", "key", 1, 3, 60*time.Second)
	start := time.Now()
	resp, err := client.Get(server.URL)

	if err != nil {
		t.Fatalf("Get() unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Should have retried at least twice
	if attemptCount < 3 {
		t.Errorf("attemptCount = %d, want >= 3", attemptCount)
	}

	// Should have taken at least 2 seconds (2 retries with 1s Retry-After each)
	elapsed := time.Since(start)
	if elapsed < 2*time.Second {
		t.Errorf("elapsed time = %v, expected >= 2s (due to Retry-After)", elapsed)
	}
}

func TestDatadogHTTPClient_Get_Retries5xx(t *testing.T) {
	var attemptCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attemptCount, 1)
		if count <= 1 {
			// First attempt returns 500
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "server error"}`))
		} else {
			// Second attempt succeeds
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok"}`))
		}
	}))
	defer server.Close()

	client := newClient("key", "key", 1, 3, 60*time.Second)
	resp, err := client.Get(server.URL)

	if err != nil {
		t.Fatalf("Get() unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if attemptCount < 2 {
		t.Errorf("attemptCount = %d, want >= 2", attemptCount)
	}
}

func TestDatadogHTTPClient_Get_DoesNotRetry4xx(t *testing.T) {
	var attemptCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attemptCount, 1)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "not found"}`))
	}))
	defer server.Close()

	client := newClient("key", "key", 1, 3, 60*time.Second)
	resp, err := client.Get(server.URL)

	if err != nil {
		t.Fatalf("Get() unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}

	// Should not retry 4xx errors (except 429)
	if attemptCount != 1 {
		t.Errorf("attemptCount = %d, want 1 (no retries for 404)", attemptCount)
	}
}

func TestDatadogHTTPClient_Get_MaxRetriesExceeded(t *testing.T) {
	var attemptCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attemptCount, 1)
		w.Header().Set("Retry-After", "0") // Use 0 to speed up test
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := newClient("key", "key", 1, 2, 60*time.Second) // Only 2 retries
	resp, err := client.Get(server.URL)

	if err == nil {
		if resp != nil {
			resp.Body.Close()
		}
		t.Fatal("Get() expected error for exhausted retries, got nil")
	}

	if _, ok := err.(*rateLimitedError); !ok {
		t.Errorf("error type = %T, want *rateLimitedError", err)
	}

	// Should attempt initial + 2 retries = 3 total
	if attemptCount != 3 {
		t.Errorf("attemptCount = %d, want 3", attemptCount)
	}
}

func TestDatadogHTTPClient_Get_ConcurrencyLimit(t *testing.T) {
	var concurrentRequests int32
	var maxConcurrent int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt32(&concurrentRequests, 1)
		defer atomic.AddInt32(&concurrentRequests, -1)

		// Track max concurrency
		for {
			max := atomic.LoadInt32(&maxConcurrent)
			if current <= max || atomic.CompareAndSwapInt32(&maxConcurrent, max, current) {
				break
			}
		}

		// Simulate slow request
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newClient("key", "key", 3, 0, 60*time.Second) // Limit to 3 concurrent

	// Launch 10 requests concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := client.Get(server.URL)
			if err != nil {
				t.Errorf("Get() error: %v", err)
				return
			}
			resp.Body.Close()
		}()
	}

	wg.Wait()

	max := atomic.LoadInt32(&maxConcurrent)
	if max > 3 {
		t.Errorf("maxConcurrent = %d, want <= 3", max)
	}
}

func TestDatadogHTTPClient_GlobalPause(t *testing.T) {
	var requestTimes []time.Time
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestTimes = append(requestTimes, time.Now())
		count := len(requestTimes)
		mu.Unlock()

		if count == 1 {
			// First request returns 429 with 1 second Retry-After
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := newClient("key", "key", 5, 3, 60*time.Second)

	// Launch multiple requests concurrently
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := client.Get(server.URL)
			if err != nil {
				return
			}
			resp.Body.Close()
		}()
		// Stagger launches slightly
		time.Sleep(10 * time.Millisecond)
	}

	wg.Wait()

	mu.Lock()
	defer mu.Unlock()

	// Should have multiple requests
	if len(requestTimes) < 3 {
		t.Fatalf("requestTimes count = %d, want >= 3", len(requestTimes))
	}

	// After first 429, subsequent requests should be delayed by ~1s
	// Check that later requests happened significantly after the first one
	first := requestTimes[0]
	for i := 1; i < len(requestTimes); i++ {
		gap := requestTimes[i].Sub(first)
		if gap < 900*time.Millisecond {
			t.Errorf("request %d gap = %v, expected >= 900ms (global pause)", i, gap)
		}
	}
}

func TestBackoffDuration(t *testing.T) {
	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 500 * time.Millisecond},
		{1, 1 * time.Second},
		{2, 2 * time.Second},
		{3, 4 * time.Second},
		{4, 5 * time.Second}, // Capped at 5s
		{5, 5 * time.Second}, // Still capped
	}

	for _, tt := range tests {
		t.Run(string(rune('0'+tt.attempt)), func(t *testing.T) {
			got := backoffDuration(tt.attempt)
			if got != tt.expected {
				t.Errorf("backoffDuration(%d) = %v, want %v", tt.attempt, got, tt.expected)
			}
		})
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected time.Duration
	}{
		{
			name:     "valid seconds",
			header:   "60",
			expected: 60 * time.Second,
		},
		{
			name:     "zero seconds",
			header:   "0",
			expected: 0 * time.Second,
		},
		{
			name:     "small value",
			header:   "5",
			expected: 5 * time.Second,
		},
		{
			name:     "no header",
			header:   "",
			expected: 1 * time.Second,
		},
		{
			name:     "invalid format",
			header:   "invalid",
			expected: 1 * time.Second,
		},
		{
			name:     "negative value",
			header:   "-1",
			expected: 1 * time.Second,
		},
		{
			name:     "http date format",
			header:   "Wed, 21 Oct 2015 07:28:00 GMT",
			expected: 1 * time.Second, // Falls back to default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				Header: http.Header{},
			}
			if tt.header != "" {
				resp.Header.Set("Retry-After", tt.header)
			}

			got := parseRetryAfter(resp)
			if got != tt.expected {
				t.Errorf("parseRetryAfter() = %v, want %v", got, tt.expected)
			}
		})
	}

	t.Run("nil response", func(t *testing.T) {
		got := parseRetryAfter(nil)
		if got != 1*time.Second {
			t.Errorf("parseRetryAfter(nil) = %v, want 1s", got)
		}
	})
}

func TestRateLimitedError(t *testing.T) {
	err := &rateLimitedError{after: 5 * time.Second}
	expected := "rate limited by server (retry after 5s)"

	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestDatadogHTTPClient_SetPause(t *testing.T) {
	client := newClient("key", "key", 1, 0, 60*time.Second)

	t.Run("sets pause duration", func(t *testing.T) {
		client.setPause(2 * time.Second)

		// pauseUntil should be approximately 2 seconds from now
		client.pause.Lock()
		until := client.pauseUntil
		client.pause.Unlock()

		diff := time.Until(until)
		if diff < 1900*time.Millisecond || diff > 2100*time.Millisecond {
			t.Errorf("pause duration = %v, want ~2s", diff)
		}
	})

	t.Run("uses default for zero duration", func(t *testing.T) {
		client2 := newClient("key", "key", 1, 0, 60*time.Second)
		client2.setPause(0)

		client2.pause.Lock()
		until := client2.pauseUntil
		client2.pause.Unlock()

		diff := time.Until(until)
		if diff < 900*time.Millisecond || diff > 1100*time.Millisecond {
			t.Errorf("pause duration = %v, want ~1s (default)", diff)
		}
	})

	t.Run("keeps longer pause", func(t *testing.T) {
		client3 := newClient("key", "key", 1, 0, 60*time.Second)

		// Set a 3 second pause
		client3.setPause(3 * time.Second)
		client3.pause.Lock()
		first := client3.pauseUntil
		client3.pause.Unlock()

		// Try to set a shorter 1 second pause
		time.Sleep(100 * time.Millisecond)
		client3.setPause(1 * time.Second)
		client3.pause.Lock()
		second := client3.pauseUntil
		client3.pause.Unlock()

		// Should keep the longer (first) pause
		if !second.Equal(first) && second.Before(first) {
			t.Error("setPause() should keep longer existing pause")
		}
	})
}

func TestDatadogHTTPClient_WaitIfPaused(t *testing.T) {
	client := newClient("key", "key", 1, 0, 60*time.Second)

	t.Run("returns immediately when not paused", func(t *testing.T) {
		start := time.Now()
		client.waitIfPaused()
		elapsed := time.Since(start)

		if elapsed > 50*time.Millisecond {
			t.Errorf("waitIfPaused() took %v, expected immediate return", elapsed)
		}
	})

	t.Run("waits until pause expires", func(t *testing.T) {
		client.setPause(500 * time.Millisecond)

		start := time.Now()
		client.waitIfPaused()
		elapsed := time.Since(start)

		if elapsed < 450*time.Millisecond {
			t.Errorf("waitIfPaused() took %v, expected ~500ms", elapsed)
		}
	})
}
