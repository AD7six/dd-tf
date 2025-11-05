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
	Data map[string]any // Optional: cached dashboard data to avoid duplicate API calls
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
				if err := downloadDashboard(target); err != nil {
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
	return fetchDashboardIDsByTags(nil)
}

// fetchDashboardIDsByTags queries the Datadog API to retrieve dashboard IDs, optionally filtered by tags.
// If filterTags is nil or empty, returns all dashboards.
// Note: The list endpoint doesn't return tags, so when filtering we must fetch each dashboard individually.
func fetchDashboardIDsByTags(filterTags []string) ([]string, error) {
	settings, err := utils.LoadSettings()
	if err != nil {
		return nil, err
	}

	client := utils.GetHTTPClient(settings.APIKey, settings.AppKey)
	url := fmt.Sprintf("https://%s/api/v1/dashboard", settings.APIDomain)

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch dashboards: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s\n%s", resp.Status, string(body))
	}

	// Parse response to get dashboard IDs
	var result struct {
		Dashboards []struct {
			ID string `json:"id"`
		} `json:"dashboards"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// If no filtering, return all IDs
	if len(filterTags) == 0 {
		ids := make([]string, 0, len(result.Dashboards))
		for _, dashboard := range result.Dashboards {
			if dashboard.ID != "" {
				ids = append(ids, dashboard.ID)
			}
		}
		return ids, nil
	}

	// When filtering by tags, we need to fetch each dashboard individually
	// because the list endpoint doesn't include tags
	ids := make([]string, 0)
	for _, dashboard := range result.Dashboards {
		if dashboard.ID == "" {
			continue
		}

		// Fetch full dashboard to get tags
		dashboardURL := fmt.Sprintf("https://%s/api/v1/dashboard/%s", settings.APIDomain, dashboard.ID)
		dashResp, err := client.Get(dashboardURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to fetch dashboard %s: %v\n", dashboard.ID, err)
			continue
		}

		if dashResp.StatusCode != http.StatusOK {
			dashResp.Body.Close()
			fmt.Fprintf(os.Stderr, "Warning: failed to fetch dashboard %s: %s\n", dashboard.ID, dashResp.Status)
			continue
		}

		var dashData struct {
			Tags []string `json:"tags"`
		}

		if err := json.NewDecoder(dashResp.Body).Decode(&dashData); err != nil {
			dashResp.Body.Close()
			fmt.Fprintf(os.Stderr, "Warning: failed to decode dashboard %s: %v\n", dashboard.ID, err)
			continue
		}
		dashResp.Body.Close()

		// Check if dashboard has all required filter tags
		if hasAllTags(dashData.Tags, filterTags) {
			ids = append(ids, dashboard.ID)
		}
	}

	return ids, nil
}

// fetchDashboardsWithTagsFiltered fetches full dashboard data for dashboards matching the given tags.
// Returns a map of dashboard ID to dashboard data to avoid duplicate API calls.
func fetchDashboardsWithTagsFiltered(filterTags []string) (map[string]map[string]interface{}, error) {
	settings, err := utils.LoadSettings()
	if err != nil {
		return nil, err
	}

	client := utils.GetHTTPClient(settings.APIKey, settings.AppKey)
	url := fmt.Sprintf("https://%s/api/v1/dashboard", settings.APIDomain)

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch dashboards: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s\n%s", resp.Status, string(body))
	}

	// Parse response to get dashboard IDs
	var result struct {
		Dashboards []struct {
			ID string `json:"id"`
		} `json:"dashboards"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Fetch each dashboard individually to check tags and cache the data
	dashboards := make(map[string]map[string]interface{})
	for _, dashboard := range result.Dashboards {
		if dashboard.ID == "" {
			continue
		}

		// Fetch full dashboard to get tags
		dashboardURL := fmt.Sprintf("https://%s/api/v1/dashboard/%s", settings.APIDomain, dashboard.ID)
		dashResp, err := client.Get(dashboardURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to fetch dashboard %s: %v\n", dashboard.ID, err)
			continue
		}

		if dashResp.StatusCode != http.StatusOK {
			dashResp.Body.Close()
			fmt.Fprintf(os.Stderr, "Warning: failed to fetch dashboard %s: %s\n", dashboard.ID, dashResp.Status)
			continue
		}

		var dashData map[string]interface{}
		if err := json.NewDecoder(dashResp.Body).Decode(&dashData); err != nil {
			dashResp.Body.Close()
			fmt.Fprintf(os.Stderr, "Warning: failed to decode dashboard %s: %v\n", dashboard.ID, err)
			continue
		}
		dashResp.Body.Close()

		// Extract tags for filtering
		var tags []string
		if tagsInterface, ok := dashData["tags"]; ok {
			if tagsArray, ok := tagsInterface.([]interface{}); ok {
				for _, tag := range tagsArray {
					if tagStr, ok := tag.(string); ok {
						tags = append(tags, tagStr)
					}
				}
			}
		}

		// Check if dashboard has all required filter tags
		if hasAllTags(tags, filterTags) {
			dashboards[dashboard.ID] = dashData
		}
	}

	return dashboards, nil
}

// hasAllTags checks if dashboardTags contains all of the required filterTags.
// Tag matching is case-insensitive.
func hasAllTags(dashboardTags, filterTags []string) bool {
	if len(filterTags) == 0 {
		return true
	}

	// Normalize dashboard tags to lowercase for case-insensitive comparison
	tagSet := make(map[string]bool, len(dashboardTags))
	for _, tag := range dashboardTags {
		tagSet[strings.ToLower(tag)] = true
	}

	// Check if all filter tags are present
	for _, filterTag := range filterTags {
		if !tagSet[strings.ToLower(filterTag)] {
			return false
		}
	}

	return true
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

	// Build filter tags from --team and --tags flags
	var filterTags []string
	if team != "" {
		// --team is a convenience flag that translates to team:x tag
		filterTags = append(filterTags, fmt.Sprintf("team:%s", team))
	}
	if tags != "" {
		// Parse comma-separated tags
		parsedTags := utils.ParseCommaSeparatedIDs(tags) // Reuse the string splitting logic
		filterTags = append(filterTags, parsedTags...)
	}

	// --team or --tags: fetch dashboards filtered by tags
	if len(filterTags) > 0 {
		go func() {
			defer close(out)
			dashboards, err := fetchDashboardsWithTagsFiltered(filterTags)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: failed to fetch dashboards by tags: %v\n", err)
				return
			}
			if len(dashboards) == 0 {
				fmt.Fprintf(os.Stderr, "Warning: no dashboards found with tags: %v\n", filterTags)
			}
			for id, data := range dashboards {
				// Include cached data to avoid duplicate API call
				out <- dashboardTarget{ID: id, Path: "", Data: data}
			}
		}()
		return out, nil
	}

	close(out)
	return nil, fmt.Errorf("please specify --id, --all, --team, --tags, or --update")
}

// downloadDashboard fetches a dashboard and writes it to the specified path.
// Uses cached data from target.Data if available to avoid duplicate API calls.
// If target.Path is empty, computes the path using the configured pattern.
func downloadDashboard(target dashboardTarget) error {
	settings, err := utils.LoadSettings()
	if err != nil {
		return err
	}

	var result map[string]interface{}

	// Use cached data if available (from tag filtering)
	if target.Data != nil {
		result = target.Data
	} else {
		// Fetch from API
		client := utils.GetHTTPClient(settings.APIKey, settings.AppKey)
		url := fmt.Sprintf("https://%s/api/v1/dashboard/%s", settings.APIDomain, target.ID)

		resp, err := client.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("API error: %s\n%s", resp.Status, string(body))
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return err
		}
	}

	// Compute path if not provided (--update uses existing path)
	targetPath := target.Path
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

// downloadDashboardByID fetches a dashboard by ID from the Datadog API and writes to the specified path.
// If targetPath is empty, computes the path using the configured pattern.
// Deprecated: Use downloadDashboard with dashboardTarget instead.
func downloadDashboardByID(id, targetPath string) error {
	return downloadDashboard(dashboardTarget{ID: id, Path: targetPath, Data: nil})
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
