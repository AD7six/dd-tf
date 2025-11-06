package resource

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/AD7six/dd-tf/internal/config"
)

// HTTPClient is an interface for HTTP clients that can perform GET requests.
// This allows using both *http.Client and *internalhttp.DatadogHTTPClient.
type HTTPClient interface {
	Get(url string) (*http.Response, error)
}

// FetchResourceFromAPI fetches a resource from the Datadog API.
// Returns the decoded JSON data or an error.
// This consolidates the common pattern of: HTTP GET, check status, decode JSON.
func FetchResourceFromAPI(client HTTPClient, url string, settings *config.Settings) (map[string]any, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(io.LimitReader(resp.Body, settings.HTTPMaxBodySize))
		if err != nil {
			return nil, fmt.Errorf("API error %s (failed to read response body: %w)", resp.Status, err)
		}
		return nil, fmt.Errorf("API error: %s\n%s", resp.Status, string(body))
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}
