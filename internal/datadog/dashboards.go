package dashboards

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
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
	DownloadCmd.Flags().StringVar(&outputPath, "output", "", "Output path template (supports {DASHBOARDS_DIR}, {id}, {title}, {team} and {any-tag}")
	DownloadCmd.Flags().StringVar(&team, "team", "", "Team name (convenience for tag 'team:x')")
	DownloadCmd.Flags().StringVar(&tags, "tags", "", "Comma-separated list of tags to filter dashboards")
	DownloadCmd.Flags().StringVar(&dashboardID, "id", "", "Dashboard ID(s) to download (comma-separated)")
}

// fetchAllDashboardIDs queries the Datadog API to retrieve all dashboard IDs.
func fetchAllDashboardIDs() ([]string, error) {
	settings, err := utils.LoadSettings()
	if err != nil {
		return nil, err
	}

	client := utils.NewDatadogHTTPClient(settings.APIKey, settings.AppKey, settings.Retry429MaxAttempts)
	url := fmt.Sprintf("https://%s/api/v1/dashboard", settings.APIDomain)

	resp, err := client.GetWithRetry(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch dashboards: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s\n%s", resp.Status, string(body))
	}

	// Parse response to extract dashboard IDs
	var result struct {
		Dashboards []struct {
			ID string `json:"id"`
		} `json:"dashboards"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	ids := make([]string, 0, len(result.Dashboards))
	for _, dashboard := range result.Dashboards {
		if dashboard.ID != "" {
			ids = append(ids, dashboard.ID)
		}
	}

	return ids, nil
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

	// --all: fetch all dashboard IDs from API
	if allFlag {
		go func() {
			defer close(out)
			ids, err := fetchAllDashboardIDs()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: failed to fetch all dashboards: %v\n", err)
				return
			}
			for _, id := range ids {
				// Path will be computed in download function with actual title
				out <- dashboardTarget{ID: id, Path: ""} // empty path means use pattern
			}
		}()
		return out, nil
	}

	// --id: download specific dashboards by ID
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
	if team != "" || tags != "" {
		close(out)
		return nil, fmt.Errorf("selectors --team/--tags not implemented yet; please use --id, --all, or --update")
	}

	close(out)
	return nil, fmt.Errorf("please specify --id, --all, or --update (other selectors not implemented yet)")
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

// translateToTemplate converts simpler convenience placeholders like
//
//	{DASHBOARDS_DIR}, {id}, {title}, {team}
//
// into Go template expressions:
//
//	{{.DashboardsDir}}, {{.ID}}, {{.Title}}, {{.Tags.team}}.
//
// Any unknown {name} will be translated to {{.Tags.name}} to support dynamic
// tag-based placeholders.
func translateToTemplate(p string) string {
	// First replace known built-ins explicitly to avoid treating them as tags
	builtin := map[string]string{
		"{DASHBOARDS_DIR}": "{{.DashboardsDir}}",
		"{id}":             "{{.ID}}",
		"{title}":          "{{.Title}}",
	}
	for k, v := range builtin {
		p = strings.ReplaceAll(p, k, v)
	}

	// Then translate any remaining {word} into {{.Tags.word}}
	re := regexp.MustCompile(`\{([A-Za-z0-9_\-]+)\}`)
	p = re.ReplaceAllStringFunc(p, func(m string) string {
		sub := re.FindStringSubmatch(m)
		if len(sub) != 2 {
			return m
		}
		name := sub[1]
		// Explicit support for {team}
		if name == "team" {
			return "{{.Tags.team}}"
		}
		return fmt.Sprintf("{{.Tags.%s}}", name)
	})
	return p
}

// wrapTagReferencesWithDefaults wraps {{.Tags.xxx}} patterns to provide "none" as default
// for missing tag keys, avoiding "<no value>" in template output.
func wrapTagReferencesWithDefaults(pattern string) string {
	// Match {{.Tags.something}} and replace with {{if index .Tags "something"}}{{index .Tags "something"}}{{else}}none{{end}}
	re := regexp.MustCompile(`\{\{\.Tags\.([A-Za-z0-9_\-]+)\}\}`)
	return re.ReplaceAllStringFunc(pattern, func(m string) string {
		sub := re.FindStringSubmatch(m)
		if len(sub) != 2 {
			return m
		}
		tagKey := sub[1]
		// Use index function which safely returns zero value for missing keys, then check if it's empty
		return fmt.Sprintf(`{{if index .Tags "%s"}}{{index .Tags "%s"}}{{else}}none{{end}}`, tagKey, tagKey)
	})
}

// computeDashboardPath computes the file path from the configured pattern or --output flag using Go templates.
// Template variables:
//
//	{{.DashboardsDir}} - the dashboards directory from settings
//	{{.ID}} - dashboard ID
//	{{.Title}} - sanitized dashboard title
//	{{.Tags.team}} - value of "team" tag (empty if not found)
//	{{.Tags.x}} - value of "x" tag (empty if not found)
func computeDashboardPath(settings *utils.Settings, dashboard map[string]interface{}) string {
	// Use --output flag if provided, otherwise use setting
	pattern := outputPath
	if pattern == "" {
		pattern = settings.DashboardsPathPattern
	}

	// Translate simple placeholders like {id} to Go template variables before
	// rendering
	pattern = translateToTemplate(pattern)

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

	// Wrap tag references in the pattern to provide default "none" for missing keys
	// Replace {{.Tags.xxx}} with {{if index .Tags "xxx"}}{{index .Tags "xxx"}}{{else}}none{{end}}
	wrappedPattern := wrapTagReferencesWithDefaults(pattern)

	// Create template
	tmpl, err := template.New("path").Parse(wrappedPattern)
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
