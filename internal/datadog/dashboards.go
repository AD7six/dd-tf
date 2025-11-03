package dashboards

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

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
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("DD-API-KEY", settings.APIKey)
	req.Header.Set("DD-APPLICATION-KEY", settings.AppKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s\n%s", resp.Status, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	// Get the dashboard title
	title, _ := result["title"].(string)

	// Write JSON via utils helper
	writer := utils.NewJSONWriter(settings.DashboardsDir)
	filename, err := writer.SavePretty(id, title, result)
	if err != nil {
		return err
	}
	fmt.Printf("Dashboard saved to %s\n", filename)
	return nil
}
