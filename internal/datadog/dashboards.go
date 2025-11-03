package dashboards

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

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
		switch {
		case allFlag:
			fmt.Println("Downloading all dashboards...")
			// TODO: Implement logic to download all dashboards
		case team != "":
			tagFilter := fmt.Sprintf("team:%s", team)
			fmt.Printf("Downloading dashboards with team tag: %s\n", tagFilter)
			// TODO: Implement logic to download dashboards by team tag
		case tags != "":
			tagList := strings.Split(tags, ",")
			fmt.Printf("Downloading dashboards with tags: %v\n", tagList)
			// TODO: Implement logic to download dashboards by tags
		case dashboardID != "":
			fmt.Printf("Downloading dashboard with ID: %s\n", dashboardID)
			err := downloadDashboardByID(dashboardID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error downloading dashboard: %v\n", err)
				os.Exit(1)
			}
		default:
			fmt.Println("Please specify --all, --team, --tags, or --id")
		}
	},
}

func init() {
	DownloadCmd.Flags().BoolVar(&allFlag, "all", false, "Download all dashboards")
	DownloadCmd.Flags().StringVar(&team, "team", "", "Team name (convenience for tag 'team:x')")
	DownloadCmd.Flags().StringVar(&tags, "tags", "", "Comma-separated list of tags to filter dashboards")
	DownloadCmd.Flags().StringVar(&dashboardID, "id", "", "Dashboard ID to download")
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

	// Get the dashboard title and sanitize it for filename
	title, _ := result["title"].(string)
	safeTitle := sanitizeFilename(title)

	// Ensure output directory exists (from settings)
	dashboardsDir := settings.DashboardsDir
	if err := os.MkdirAll(dashboardsDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write to file
	filename := fmt.Sprintf("%s/%s-%s.json", dashboardsDir, id, safeTitle)
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		return fmt.Errorf("failed to write JSON: %w", err)
	}

	fmt.Printf("Dashboard saved to %s\n", filename)
	return nil
}

// sanitizeFilename replaces non-alphanumeric characters with underscores.
func sanitizeFilename(name string) string {
	// Replace spaces and non-alphanumeric characters with underscores
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	return strings.Trim(re.ReplaceAllString(name, "_"), "_")
}
