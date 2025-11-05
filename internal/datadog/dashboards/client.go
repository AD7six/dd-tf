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

	"github.com/AD7six/dd-tf/internal/utils"
)

// DashboardTarget represents a dashboard ID and the path where it should be written.
type DashboardTarget struct {
	ID   string
	Path string

	// The response to the dashboards endpoint does not include all data
	// provided in the individual dashboard request; we need the individual
	// dashboard data. If we've already requested this dashboard's individual
	// data it is stored here to avoid re-requesting it.
	CompleteData map[string]any
}

// DashboardTargetResult wraps a DashboardTarget with a potential error from target generation.
type DashboardTargetResult struct {
	Target DashboardTarget
	Err    error
}

// DownloadOptions contains options for downloading dashboards.
type DownloadOptions struct {
	All         bool
	Update      bool
	OutputPath  string
	Team        string
	Tags        string
	DashboardID string
}

// FetchAllDashboardIDs queries the Datadog API to retrieve all dashboard IDs.
func FetchAllDashboardIDs() ([]string, error) {
	return fetchDashboardIDsByTags(nil)
}

// fetchDashboardIDsByTags queries the Datadog API to retrieve dashboard IDs,
// optionally filtered by tags. If filterTags is nil or empty, returns all
// dashboards.
// Note: The index endpoint doesn't return tags, so when filtering by tags or
// team we must (eugh) fetch each dashboard individually.
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
	// because the index endpoint doesn't include tags
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

// FetchDashboardsWithTagsFiltered fetches full dashboard data for dashboards matching the given tags.
// Returns a map of dashboard ID to dashboard data to avoid duplicate API calls.
func FetchDashboardsWithTagsFiltered(filterTags []string) (map[string]map[string]any, error) {
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
	dashboards := make(map[string]map[string]any)
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

// GenerateDashboardTargets returns a channel that yields dashboard IDs and target paths.
// For --update mode, uses existing file paths. For other modes, computes paths from pattern.
// Errors during target generation are returned as part of DashboardTargetResult.
func GenerateDashboardTargets(opts DownloadOptions) (<-chan DashboardTargetResult, error) {
	out := make(chan DashboardTargetResult)

	settings, err := utils.LoadSettings()
	if err != nil {
		close(out)
		return nil, err
	}

	// --update: scan existing dashboard files and use their paths
	if opts.Update {
		go func() {
			defer close(out)
			idToPath, err := utils.ExtractIDsFromJSONFiles(settings.DashboardsDir)
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
			ids, err := FetchAllDashboardIDs()
			if err != nil {
				out <- DashboardTargetResult{Err: fmt.Errorf("failed to fetch all dashboards: %w", err)}
				return
			}
			for _, id := range ids {
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
				// Path will be computed in download function with actual title
				out <- DashboardTargetResult{Target: DashboardTarget{ID: id, Path: ""}} // empty path means use pattern
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
			dashboards, err := FetchDashboardsWithTagsFiltered(filterTags)
			if err != nil {
				out <- DashboardTargetResult{Err: fmt.Errorf("failed to fetch dashboards by tags: %w", err)}
				return
			}
			if len(dashboards) == 0 {
				fmt.Fprintf(os.Stderr, "Warning: no dashboards found with tags: %v\n", filterTags)
			}
			for id, data := range dashboards {
				// Include cached data to avoid duplicate API call
				out <- DashboardTargetResult{Target: DashboardTarget{ID: id, Path: "", CompleteData: data}}
			}
		}()
		return out, nil
	}

	close(out)
	return nil, fmt.Errorf("please specify --id, --all, --team, --tags, or --update")
}

// DownloadDashboard fetches a dashboard and writes it to the specified path.
// Uses cached data from target.CompleteData if available to avoid duplicate API calls.
// If target.Path is empty, computes the path using the configured pattern.
func DownloadDashboard(target DashboardTarget) error {
	return DownloadDashboardWithOptions(target, "")
}

// DownloadDashboardWithOptions fetches a dashboard and writes it to the specified path.
// Uses cached data from target.CompleteData if available to avoid duplicate API calls.
// If target.Path is empty, computes the path using the configured pattern or outputPath override.
func DownloadDashboardWithOptions(target DashboardTarget, outputPath string) error {
	settings, err := utils.LoadSettings()
	if err != nil {
		return err
	}

	var result map[string]any

	// Use cached data if available (from tag filtering)
	if target.CompleteData != nil {
		result = target.CompleteData
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
		targetPath = ComputeDashboardPath(settings, result, outputPath)
	}

	// Write JSON file
	if err := utils.WriteJSONFile(targetPath, result); err != nil {
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

// ComputeDashboardPath computes the file path from the configured pattern or outputPath override using Go templates.
// Template variables:
//
//	{{.DashboardsDir}} - the dashboards directory from settings
//	{{.ID}} - dashboard ID
//	{{.Title}} - sanitized dashboard title
//	{{.Tags.team}} - value of "team" tag (empty if not found)
//	{{.Tags.x}} - value of "x" tag (empty if not found)
func ComputeDashboardPath(settings *utils.Settings, dashboard map[string]any, outputPath string) string {
	// Use outputPath override if provided, otherwise use setting
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
		Title:         utils.SanitizeFilename(title),
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
