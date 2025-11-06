package dashboards

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/AD7six/dd-tf/internal/config"
	"github.com/AD7six/dd-tf/internal/datadog/resource"
	"github.com/AD7six/dd-tf/internal/datadog/templating"
	internalhttp "github.com/AD7six/dd-tf/internal/http"
	"github.com/AD7six/dd-tf/internal/storage"
	"github.com/AD7six/dd-tf/internal/utils"
)

var (
	// dashboardIDRegex validates dashboard ID format (xxx-xxx-xxx)
	dashboardIDRegex = regexp.MustCompile(`^(?i)[a-z0-9]+-[a-z0-9]+-[a-z0-9]+$`)
)

// DashboardTarget is an alias for the generic resource.Target with string IDs.
type DashboardTarget = resource.Target[string]

// DashboardTargetResult is an alias for the generic resource.TargetResult with string IDs.
type DashboardTargetResult = resource.TargetResult[string]

// DownloadOptions contains options for downloading dashboards.
type DownloadOptions struct {
	resource.BaseDownloadOptions // Embedded common options
}

// fetchAndFilterDashboards fetches dashboards from the Datadog API, optionally filtered by tags.
// If fullData is true, returns complete dashboard data; if false, returns minimal data (just IDs).
func fetchAndFilterDashboards(filterTags []string, fullData bool) (map[string]map[string]any, error) {
	settings, err := config.LoadSettings()
	if err != nil {
		return nil, err
	}

	client := internalhttp.GetHTTPClient(settings)

	// Fetch all dashboard IDs with pagination
	// Dashboards API uses 'start' and 'count' parameters for pagination
	var allDashboardIDs []string
	start := 0
	count := settings.PageSize
	for {
		url := fmt.Sprintf("https://api.%s/api/v1/dashboard?start=%d&count=%d", settings.Site, start, count)
		resp, err := client.Get(url)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch dashboards (start=%d): %w", start, err)
		}

		if resp.StatusCode != http.StatusOK {
			body, err := io.ReadAll(io.LimitReader(resp.Body, settings.HTTPMaxBodySize))
			resp.Body.Close()
			if err != nil {
				return nil, fmt.Errorf("API error %s (start=%d) (failed to read response body: %w)", resp.Status, start, err)
			}
			return nil, fmt.Errorf("API error (start=%d): %s\n%s", start, resp.Status, string(body))
		}

		// Parse response to get dashboard IDs
		var result struct {
			Dashboards []struct {
				ID string `json:"id"`
			} `json:"dashboards"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode response (start=%d): %w", start, err)
		}
		resp.Body.Close()

		if len(result.Dashboards) == 0 {
			break
		}

		for _, dashboard := range result.Dashboards {
			if dashboard.ID != "" {
				allDashboardIDs = append(allDashboardIDs, dashboard.ID)
			}
		}

		// If we got fewer results than requested count, this is the last page
		if len(result.Dashboards) < count {
			break
		}
		start += len(result.Dashboards)
	}

	// If no filtering and we don't need full data, return early with just IDs
	if len(filterTags) == 0 && !fullData {
		dashboards := make(map[string]map[string]any, len(allDashboardIDs))
		for _, id := range allDashboardIDs {
			dashboards[id] = nil // No data needed, just ID
		}
		return dashboards, nil
	}

	// Fetch individual dashboards when filtering or when full data is needed
	dashboards := make(map[string]map[string]any)
	for _, id := range allDashboardIDs {

		// Fetch full dashboard to get tags (and potentially cache the data)
		dashboardURL := fmt.Sprintf("https://api.%s/api/v1/dashboard/%s", settings.Site, id)
		dashResp, err := client.Get(dashboardURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to fetch dashboard %s: %v\n", id, err)
			continue
		}

		if dashResp.StatusCode != http.StatusOK {
			dashResp.Body.Close()
			fmt.Fprintf(os.Stderr, "Warning: failed to fetch dashboard %s: %s\n", id, dashResp.Status)
			continue
		}

		var dashData map[string]any
		if err := json.NewDecoder(dashResp.Body).Decode(&dashData); err != nil {
			dashResp.Body.Close()
			fmt.Fprintf(os.Stderr, "Warning: failed to decode dashboard %s: %v\n", id, err)
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
		if templating.HasAllTagsSlice(tags, filterTags) {
			if fullData {
				dashboards[id] = dashData
			} else {
				dashboards[id] = nil // Just store the ID
			}
		}
	}

	return dashboards, nil
}

// normalizezDashboardID validates that the dashboard ID follows the expected
// format (xxx-xxx-xxx). Handles case if that matters.
func normalizezDashboardID(id string) (string, error) {
	if id == "" {
		return "", fmt.Errorf("dashboard ID cannot be empty")
	}
	if !dashboardIDRegex.MatchString(id) {
		return "", fmt.Errorf("invalid dashboard ID format: %s (expected format: xxx-xxx-xxx)", id)
	}

	return strings.ToLower(id), nil
}

// GenerateDashboardTargets returns a channel that yields dashboard IDs and target paths.
// For --update mode, uses existing file paths. For other modes, computes paths from pattern.
// Errors during target generation are returned as part of DashboardTargetResult.
func GenerateDashboardTargets(opts DownloadOptions) (<-chan DashboardTargetResult, error) {
	out := make(chan DashboardTargetResult)

	settings, err := config.LoadSettings()
	if err != nil {
		close(out)
		return nil, err
	}

	// --update: scan existing dashboard files and use their paths
	if opts.Update {
		go func() {
			defer close(out)
			// Extract the static directory prefix from the path template
			dashboardsDir := templating.ExtractStaticPrefix(settings.DashboardsPathTemplate)
			if dashboardsDir == "" {
				dashboardsDir = filepath.Join(settings.DataDir, "dashboards")
			}
			idToPath, err := storage.ExtractIDsFromJSONFiles(dashboardsDir)
			if err != nil {
				out <- DashboardTargetResult{Err: fmt.Errorf("failed to scan directory: %w", err)}
				return
			}
			for id, path := range idToPath {
				out <- DashboardTargetResult{Target: DashboardTarget{ID: id, Path: path}}
			}
		}()
		return out, nil
	}

	// --all: fetch all dashboard IDs from API
	if opts.All {
		go func() {
			defer close(out)
			dashboards, err := fetchAndFilterDashboards(nil, false)
			if err != nil {
				out <- DashboardTargetResult{Err: fmt.Errorf("failed to fetch all dashboards: %w", err)}
				return
			}
			for id := range dashboards {
				// Path will be computed in download function with actual title
				out <- DashboardTargetResult{Target: DashboardTarget{ID: id, Path: ""}} // empty path means use pattern
			}
		}()
		return out, nil
	}

	// --id: download specific dashboards by ID
	if opts.IDs != "" {
		idList := opts.IDs
		ids := utils.ParseCommaSeparatedIDs(idList)
		go func() {
			defer close(out)
			for _, id := range ids {
				// Validate dashboard ID format
				normalizedId, err := normalizezDashboardID(id)
				if err != nil {
					out <- DashboardTargetResult{Err: fmt.Errorf("invalid dashboard ID %q: %w", id, err)}
					continue
				}

				// Empty path means use pattern. Path will be computed in
				// download function with actual title
				out <- DashboardTargetResult{Target: DashboardTarget{ID: normalizedId, Path: ""}}
			}
		}()
		return out, nil
	}

	// Build filter tags from --team and --tags flags
	var filterTags []string
	if opts.Team != "" {
		// --team is a convenience flag that translates to team:x tag
		filterTags = append(filterTags, fmt.Sprintf("team:%s", opts.Team))
	}
	if opts.Tags != "" {
		// Parse comma-separated tags
		parsedTags := utils.ParseCommaSeparatedIDs(opts.Tags) // Reuse the string splitting logic
		filterTags = append(filterTags, parsedTags...)
	}

	// --team or --tags: fetch dashboards filtered by tags
	if len(filterTags) > 0 {
		go func() {
			defer close(out)
			dashboards, err := fetchAndFilterDashboards(filterTags, true)
			if err != nil {
				out <- DashboardTargetResult{Err: fmt.Errorf("failed to fetch dashboards by tags: %w", err)}
				return
			}
			if len(dashboards) == 0 {
				fmt.Fprintf(os.Stderr, "Warning: no dashboards found with tags: %v\n", filterTags)
			}
			for id, data := range dashboards {
				// Include cached data to avoid duplicate API call
				out <- DashboardTargetResult{Target: DashboardTarget{ID: id, Path: "", Data: data}}
			}
		}()
		return out, nil
	}

	close(out)
	return nil, fmt.Errorf("please specify --id, --all, --team, --tags, or --update")
}

// DownloadDashboardWithOptions fetches a dashboard and writes it to the specified path.
// Uses cached data from target.Data if available to avoid duplicate API calls.
// If target.Path is empty, computes the path using the configured pattern or outputPath override.
func DownloadDashboardWithOptions(target DashboardTarget, outputPath string) error {
	normalizedId, err := normalizezDashboardID(target.ID)
	if err != nil {
		return err
	}

	target.ID = normalizedId

	settings, err := config.LoadSettings()
	if err != nil {
		return err
	}

	var result map[string]any

	// Use cached data if available (from tag filtering)
	if target.Data != nil {
		result = target.Data
	} else {
		// Fetch from API
		client := internalhttp.GetHTTPClient(settings)
		url := fmt.Sprintf("https://api.%s/api/v1/dashboard/%s", settings.Site, target.ID)

		resp, err := client.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, err := io.ReadAll(io.LimitReader(resp.Body, settings.HTTPMaxBodySize))
			if err != nil {
				return fmt.Errorf("API error %s (failed to read response body: %w)", resp.Status, err)
			}
			return fmt.Errorf("API error: %s\n%s", resp.Status, string(body))
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return err
		}
	}

	// Compute path if not provided (--update uses existing path)
	targetPath := target.Path
	if targetPath == "" {
		targetPath = ComputeDashboardPath(settings, result, outputPath)
	}

	// Write JSON file
	if err := storage.WriteJSONFile(targetPath, result); err != nil {
		return err
	}

	fmt.Printf("Dashboard saved to %s\n", targetPath)
	return nil
}

// dashboardTemplateData holds the data available in path templates
type dashboardTemplateData struct {
	DataDir string
	ID      string
	Title   string
	Tags    map[string]string
}

// ComputeDashboardPath computes the file path from the configured pattern or outputPath override using Go templates.
// Template variables:
//
//	{{.DataDir}} - the data directory from settings
//	{{.ID}} - dashboard ID
//	{{.Title}} - sanitized dashboard title
//	{{.Tags.team}} - value of "team" tag (empty if not found)
//	{{.Tags.x}} - value of "x" tag (empty if not found)
func ComputeDashboardPath(settings *config.Settings, dashboard map[string]any, outputPath string) string {
	// Use outputPath override if provided, otherwise use setting
	pattern := outputPath
	if pattern == "" {
		pattern = settings.DashboardsPathTemplate
	}

	// Translate simple placeholders like {id} to Go template variables before
	// rendering
	pattern = templating.TranslatePlaceholders(pattern, templating.BuildDashboardBuiltins())

	// Extract and sanitize tags from dashboard
	tagMap := templating.ExtractTagMap(dashboard["tags"], true)

	// Extract ID - required field
	id, ok := dashboard["id"].(string)
	if !ok || id == "" {
		// Fallback: use a placeholder if ID is missing
		fmt.Fprintf(os.Stderr, "Warning: dashboard missing valid 'id' field, using placeholder\n")
		id = "unknown-id"
	}

	// Extract title - use placeholder if missing
	title, ok := dashboard["title"].(string)
	if !ok || title == "" {
		fmt.Fprintf(os.Stderr, "Warning: dashboard %s missing valid 'title' field, using placeholder\n", id)
		title = "untitled"
	}

	// Build template data
	data := dashboardTemplateData{
		DataDir: settings.DataDir,
		ID:      id,
		Title:   storage.SanitizeFilename(title),
		Tags:    tagMap,
	}

	// Compute path from template with fallback
	fallbackPath := filepath.Join(settings.DataDir, "dashboards", id+".json")
	return templating.ComputePathFromTemplate(pattern, data, fallbackPath)
}
