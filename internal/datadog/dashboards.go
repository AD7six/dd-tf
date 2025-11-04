package dashboards

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/AD7six/dd-tf/internal/utils"
	"github.com/spf13/cobra"
)

// dashboardTarget represents a dashboard ID and the path where it should be written.
type dashboardTarget struct {
	ID   string
	Path string
}

var (
	allFlag     bool
	updateFlag  bool
	outputPath  string
	team        string
	tags        string
	dashboardID string
)

var DownloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download Datadog dashboards by ID, team, tags, or all",
	Run: func(cmd *cobra.Command, args []string) {
		targetsCh, err := generateDashboardTargets()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		var wg sync.WaitGroup
		errCh := make(chan error, 8)

		for target := range targetsCh {
			target := target // capture
			fmt.Printf("Downloading dashboard with ID: %s\n", target.ID)
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := downloadDashboardByID(target.ID, target.Path); err != nil {
					errCh <- fmt.Errorf("%s: %w", target.ID, err)
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
	DownloadCmd.Flags().BoolVar(&updateFlag, "update", false, "Update already-downloaded dashboards (scans existing files)")
	DownloadCmd.Flags().StringVar(&outputPath, "output", "", "Output path template (supports {{.DashboardsDir}}, {{.ID}}, {{.Title}}, {{.Tags.team}}, etc.)")
	DownloadCmd.Flags().StringVar(&team, "team", "", "Team name (convenience for tag 'team:x')")
	DownloadCmd.Flags().StringVar(&tags, "tags", "", "Comma-separated list of tags to filter dashboards")
	DownloadCmd.Flags().StringVar(&dashboardID, "id", "", "Dashboard ID(s) to download (comma-separated)")
}

// generateDashboardTargets returns a channel that yields dashboard IDs and target paths.
// For --update mode, uses existing file paths. For other modes, computes paths from pattern.
func generateDashboardTargets() (<-chan dashboardTarget, error) {
	out := make(chan dashboardTarget)

	settings, err := utils.LoadSettings()
	if err != nil {
		close(out)
		return nil, err
	}

	// --update: scan existing dashboard files and use their paths
	if updateFlag {
		go func() {
			defer close(out)
			idToPath, err := utils.ExtractIDsFromJSONFiles(settings.DashboardsDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to scan directory: %v\n", err)
				return
			}
			for id, path := range idToPath {
				out <- dashboardTarget{ID: id, Path: path}
			}
		}()
		return out, nil
	}

	// Other modes: compute path from pattern (pattern will be resolved with title later)
	if dashboardID != "" {
		ids := utils.ParseCommaSeparatedIDs(dashboardID)
		go func() {
			defer close(out)
			for _, id := range ids {
				// Path will be computed in download function with actual title
				out <- dashboardTarget{ID: id, Path: ""} // empty path means use pattern
			}
		}()
		return out, nil
	}

	// Placeholders for future implementations
	if allFlag || team != "" || tags != "" {
		close(out)
		return nil, fmt.Errorf("selectors --all/--team/--tags not implemented yet; please use --id or --update")
	}

	close(out)
	return nil, fmt.Errorf("please specify --id or --update (other selectors not implemented yet)")
}

// downloadDashboardByID fetches a dashboard by ID from the Datadog API and writes to the specified path.
// If targetPath is empty, computes the path using the configured pattern.
func downloadDashboardByID(id, targetPath string) error {
	settings, err := utils.LoadSettings()
	if err != nil {
		return err
	}

	// Create HTTP client with retry logic
	client := utils.NewDatadogHTTPClient(settings.APIKey, settings.AppKey, settings.Retry429MaxAttempts)
	url := fmt.Sprintf("https://%s/api/v1/dashboard/%s", settings.APIDomain, id)

	resp, err := client.GetWithRetry(url)
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

	// Compute path if not provided (--update uses existing path)
	if targetPath == "" {
		targetPath = computeDashboardPath(settings, result)
	}

	// Ensure directory exists
	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write JSON file
	f, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		return fmt.Errorf("failed to write JSON: %w", err)
	}

	fmt.Printf("Dashboard saved to %s\n", targetPath)
	return nil
}

// dashboardTemplateData holds the data available in path templates
type dashboardTemplateData struct {
	DashboardsDir string
	ID            string
	Title         string
	Tags          map[string]string
}

// computeDashboardPath computes the file path from the configured pattern or --output flag using Go templates.
// Template variables:
//
//	{{.DashboardsDir}} - the dashboards directory from settings
//	{{.ID}} - dashboard ID
//	{{.Title}} - sanitized dashboard title
//	{{.Tags.team}} - value of "team" tag (or "none" if not found)
//	{{.Tags.env}} - value of "env" tag (or "none" if not found)
//	... any other tag name
//
// Custom functions:
//
//	{{tag "team"}} - returns tag value or "none" if not found
func computeDashboardPath(settings *utils.Settings, dashboard map[string]interface{}) string {
	// Use --output flag if provided, otherwise use setting
	pattern := outputPath
	if pattern == "" {
		pattern = settings.DashboardsPathPattern
	}

	// Extract tags from dashboard and build tag map
	tagMap := make(map[string]string)
	if tagsInterface, ok := dashboard["tags"]; ok {
		if tags, ok := tagsInterface.([]interface{}); ok {
			for _, tagInterface := range tags {
				if tag, ok := tagInterface.(string); ok {
					// Tags are in format "key:value"
					parts := strings.SplitN(tag, ":", 2)
					if len(parts) == 2 {
						key := strings.TrimSpace(parts[0])
						value := strings.TrimSpace(parts[1])
						tagMap[key] = utils.SanitizeFilename(value)
					}
				}
			}
		}
	}

	// Build template data
	data := dashboardTemplateData{
		DashboardsDir: settings.DashboardsDir,
		ID:            dashboard["id"].(string),
		Title:         utils.SanitizeFilename(dashboard["title"].(string)),
		Tags:          tagMap,
	}

	// Create template with custom function to handle missing tags
	tmpl, err := template.New("path").Funcs(template.FuncMap{
		"tag": func(key string) string {
			if value, ok := tagMap[key]; ok {
				return value
			}
			return "none"
		},
	}).Parse(pattern)
	if err != nil {
		// If template parsing fails, fall back to literal pattern
		fmt.Fprintf(os.Stderr, "Warning: failed to parse path template: %v\n", err)
		return filepath.Join(settings.DashboardsDir, dashboard["id"].(string)+".json")
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		// If execution fails, fall back to literal pattern
		fmt.Fprintf(os.Stderr, "Warning: failed to execute path template: %v\n", err)
		return filepath.Join(settings.DashboardsDir, dashboard["id"].(string)+".json")
	}

	return buf.String()
}
