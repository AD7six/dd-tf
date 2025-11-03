package dashboards

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/AD7six/dd-tf/internal/utils"
	"github.com/spf13/cobra"
)

var (
	allFlag     bool
	team        string
	tags        string
	dashboardID string
)

var DownloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download Datadog dashboards by ID, team, tags, or all",
	Run: func(cmd *cobra.Command, args []string) {
		idsCh, err := generateDashboardIDs()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		var wg sync.WaitGroup
		errCh := make(chan error, 8)

		for id := range idsCh {
			id := id // capture
			fmt.Printf("Downloading dashboard with ID: %s\n", id)
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := downloadDashboardByID(id); err != nil {
					errCh <- fmt.Errorf("%s: %w", id, err)
				}
			}()
		}

		// wait and close error channel
		go func() { wg.Wait(); close(errCh) }()

		// collect errors
		var hadErr bool
		for e := range errCh {
			hadErr = true
			fmt.Fprintf(os.Stderr, "Error: %v\n", e)
		}
		if hadErr {
			os.Exit(1)
		}
	},
}

func init() {
	DownloadCmd.Flags().BoolVar(&allFlag, "all", false, "Download all dashboards")
	DownloadCmd.Flags().StringVar(&team, "team", "", "Team name (convenience for tag 'team:x')")
	DownloadCmd.Flags().StringVar(&tags, "tags", "", "Comma-separated list of tags to filter dashboards")
	DownloadCmd.Flags().StringVar(&dashboardID, "id", "", "Dashboard ID(s) to download (comma-separated)")
}

// generateDashboardIDs returns a channel that yields dashboard IDs to download.
// For now, only the --id flag is supported; other selectors are not yet implemented.
func generateDashboardIDs() (<-chan string, error) {
	out := make(chan string)

	// Only implement explicit --id for now
	if dashboardID != "" {
		ids := utils.ParseCommaSeparatedIDs(dashboardID)
		go func() {
			for _, id := range ids {
				out <- id
			}
			close(out)
		}()
		return out, nil
	}

	// Placeholders for future implementations
	if allFlag || team != "" || tags != "" {
		close(out)
		return nil, fmt.Errorf("selectors --all/--team/--tags not implemented yet; please use --id")
	}

	close(out)
	return nil, fmt.Errorf("please specify --id (other selectors not implemented yet)")
}

// downloadDashboardByID fetches a dashboard by ID from the Datadog API and writes the JSON to data/dashboards/<ID>-title.json.
func downloadDashboardByID(id string) error {
	settings, err := utils.LoadSettings()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://%s/api/v1/dashboard/%s", settings.APIDomain, id)

	var resp *http.Response
	var lastErr error
	for attempt := 0; attempt <= settings.Retry429MaxAttempts; attempt++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}
		req.Header.Set("DD-API-KEY", settings.APIKey)
		req.Header.Set("DD-APPLICATION-KEY", settings.AppKey)

		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			lastErr = err
			// transient network error; retry like 429s
		} else if resp.StatusCode == http.StatusTooManyRequests { // 429
			// use Retry-After if present, else exponential backoff
			wait := retryAfterDelay(resp, attempt)
			// Drain body before retry
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if attempt < settings.Retry429MaxAttempts {
				time.Sleep(wait)
				continue
			}
			// no more retries; capture as error
			body, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("rate limited (429) after %d retries: %s", attempt, string(body))
		} else if resp.StatusCode != http.StatusOK {
			// Non-429, non-OK: return immediately
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return fmt.Errorf("API error: %s\n%s", resp.Status, string(body))
		} else {
			// OK
			break
		}

		if attempt < settings.Retry429MaxAttempts {
			// sleep a small backoff if we didn't already (network error path)
			time.Sleep(retryAfterDelay(nil, attempt))
			continue
		}
	}
	if resp == nil {
		return lastErr
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	// Get the dashboard title
	title, _ := result["title"].(string)

	// Write JSON via utils helper
	writer := utils.NewJSONWriter(settings.DashboardsDir, settings.AddTitleToFileNames)
	filename, err := writer.SavePretty(id, title, result)
	if err != nil {
		return err
	}
	fmt.Printf("Dashboard saved to %s\n", filename)
	return nil
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
	// exponential backoff
	d := time.Second << attempt
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	if d < time.Second {
		d = time.Second
	}
	return d
}
