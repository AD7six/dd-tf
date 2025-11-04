package utils

import (
	"net/http"
)

// DatadogHTTPClient wraps an HTTP client with Datadog API credentials.
type DatadogHTTPClient struct {
	APIKey         string
	AppKey         string
	UnderlyingHTTP *http.Client
}

// NewDatadogHTTPClient creates a new client with the given credentials.
func NewDatadogHTTPClient(apiKey, appKey string) *DatadogHTTPClient {
	return &DatadogHTTPClient{
		APIKey:         apiKey,
		AppKey:         appKey,
		UnderlyingHTTP: http.DefaultClient,
	}
}

// Get performs a simple GET request with Datadog auth headers.
func (c *DatadogHTTPClient) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("DD-API-KEY", c.APIKey)
	req.Header.Set("DD-APPLICATION-KEY", c.AppKey)

	return c.UnderlyingHTTP.Do(req)
}
