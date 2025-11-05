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
	"text/template"

	"github.com/AD7six/dd-tf/internal/config"
	internalhttp "github.com/AD7six/dd-tf/internal/http"
	"github.com/AD7six/dd-tf/internal/storage"
	"github.com/AD7six/dd-tf/internal/utils"
)

const (
	// maxResponseBodySize is the maximum size allowed for API response bodies to prevent DoS
	maxResponseBodySize = 10 * 1024 * 1024 // 10MB
)

var (
	// placeholderRegex matches placeholder patterns like {word} for template conversion
	placeholderRegex = regexp.MustCompile(`\{([A-Za-z0-9_\-]+)\}`)
	// dashboardIDRegex validates dashboard ID format (xxx-xxx-xxx)
	dashboardIDRegex = regexp.MustCompile(`^(?i)[a-z0-9]+-[a-z0-9]+-[a-z0-9]+$`)
)

// DashboardTarget represents a dashboard ID and the path where it should be written.
type DashboardTarget struct {
	ID   string
	Path string

	// The response to the dashboards endpoint does not include all data
	// provided in the individual dashboard request; we need the individual
	// dashboard data. If we've already requested this dashboard's individual
	// data it is stored here to avoid re-requesting it.
	FullDashboardResponse map[string]any
}

// DashboardTargetResult wraps a DashboardTarget with a potential error from target generation.
type DashboardTargetResult struct {
	Target DashboardTarget // The dashboard target containing ID and path information
	Err    error           // Error encountered during target generation, if any
}

// DownloadOptions contains options for downloading dashboards.
type DownloadOptions struct {
	All         bool   // Download all dashboards
	Update      bool   // Update existing dashboards from local files
	OutputPath  string // Custom output path pattern (overrides settings)
	Team        string // Filter by team tag (convenience flag for team:x)
	Tags        string // Comma-separated list of tags to filter by
	DashboardID string // Comma-separated list of dashboard IDs to download
}

// fetchAndFilterDashboards fetches dashboards from the Datadog API, optionally filtered by tags.
// If fullData is true, returns complete dashboard data; if false, returns minimal data (just IDs).
func fetchAndFilterDashboards(filterTags []string, fullData bool) (map[string]map[string]any, error) {
	settings, err := config.LoadSettings()
	if err != nil {
		return nil, err
	}

	client := internalhttp.GetHTTPClient(settings)
	url := fmt.Sprintf("https://api.%s/api/v1/dashboard", settings.Site)

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch dashboards: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))
		if err != nil {
			return nil, fmt.Errorf("API error %s (failed to read response body: %w)", resp.Status, err)
		}
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

	// If no filtering and we don't need full data, return early with just IDs
	if len(filterTags) == 0 && !fullData {
		dashboards := make(map[string]map[string]any, len(result.Dashboards))
		for _, dashboard := range result.Dashboards {
			if dashboard.ID != "" {
				dashboards[dashboard.ID] = nil // No data needed, just ID
			}
		}
		return dashboards, nil
	}

	// Fetch individual dashboards when filtering or when full data is needed
	dashboards := make(map[string]map[string]any)
	for _, dashboard := range result.Dashboards {
		if dashboard.ID == "" {
			continue
		}

		// Fetch full dashboard to get tags (and potentially cache the data)
		dashboardURL := fmt.Sprintf("https://api.%s/api/v1/dashboard/%s", settings.Site, dashboard.ID)
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

		var dashData map[string]any
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
			if fullData {
				dashboards[dashboard.ID] = dashData
			} else {
				dashboards[dashboard.ID] = nil // Just store the ID
			}
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

// normalizezDashboardID validates that the dashboard ID follows the expected
// format (xxx-xxx-xxx). Handles case if that matters.
func normalizezDashboardID(id string) (string, error) {
	if id == "" {
		return "", fmt.Errorf("dashboard ID cannot be empty")
	}
	if len(id) > 100 {
		return "", fmt.Errorf("dashboard ID too long (max 100 characters)")
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
			idToPath, err := storage.ExtractIDsFromJSONFiles(settings.DashboardsDir)
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
	if opts.DashboardID != "" {
		ids := utils.ParseCommaSeparatedIDs(opts.DashboardID)
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
				out <- DashboardTargetResult{Target: DashboardTarget{ID: id, Path: "", FullDashboardResponse: data}}
			}
		}()
		return out, nil
	}

	close(out)
	return nil, fmt.Errorf("please specify --id, --all, --team, --tags, or --update")
}

// DownloadDashboardWithOptions fetches a dashboard and writes it to the specified path.
// Uses cached data from target.FullDashboardResponse if available to avoid duplicate API calls.
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
	if target.FullDashboardResponse != nil {
		result = target.FullDashboardResponse
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
			body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))
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
	p = placeholderRegex.ReplaceAllStringFunc(p, func(m string) string {
		sub := placeholderRegex.FindStringSubmatch(m)
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

// ComputeDashboardPath computes the file path from the configured pattern or outputPath override using Go templates.
// Template variables:
//
//	{{.DashboardsDir}} - the dashboards directory from settings
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
						tagMap[key] = storage.SanitizeFilename(value)
					}
				}
			}
		}
	}

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
		DashboardsDir: settings.DashboardsDir,
		ID:            id,
		Title:         storage.SanitizeFilename(title),
		Tags:          tagMap,
	}

	// Create template
	tmpl, err := template.New("path").Parse(pattern)
	if err != nil {
		// If template parsing fails, fall back to literal pattern
		fmt.Fprintf(os.Stderr, "Warning: failed to parse path template: %v\n", err)
		return filepath.Join(settings.DashboardsDir, id+".json")
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		// If execution fails, fall back to literal pattern
		fmt.Fprintf(os.Stderr, "Warning: failed to execute path template: %v\n", err)
		return filepath.Join(settings.DashboardsDir, id+".json")
	}

	// Replace "<no value>" (from missing tags) with "none"
	result := strings.ReplaceAll(buf.String(), "<no value>", "none")
	return result
}
