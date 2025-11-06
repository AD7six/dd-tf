package monitors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/AD7six/dd-tf/internal/config"
	"github.com/AD7six/dd-tf/internal/datadog/resource"
	"github.com/AD7six/dd-tf/internal/datadog/templating"
	internalhttp "github.com/AD7six/dd-tf/internal/http"
	"github.com/AD7six/dd-tf/internal/storage"
)

// MonitorTarget is an alias for the generic resource.Target with int IDs.
type MonitorTarget = resource.Target[int]

// MonitorTargetResult is an alias for the generic resource.TargetResult with int IDs.
type MonitorTargetResult = resource.TargetResult[int]

// DownloadOptions contains options for downloading monitors.
type DownloadOptions struct {
	resource.BaseDownloadOptions     // Embedded common options
	Priority                     int // Filter by monitor priority
}

// monitorTemplateData holds the data available in path templates for monitors
type monitorTemplateData struct {
	DataDir  string
	ID       int
	Name     string
	Tags     map[string]string
	Priority int
}

// GenerateMonitorTargets returns a channel that yields monitor IDs and target paths.
// If filterTags or team is set, fetches all monitors and filters by tags/team/priority.
func GenerateMonitorTargets(opts DownloadOptions) (<-chan MonitorTargetResult, error) {
	out := make(chan MonitorTargetResult)
	settings, err := config.LoadSettings()
	if err != nil {
		close(out)
		return nil, err
	}

	// Parse monitor IDs from comma-separated string
	var ids []int
	if opts.IDs != "" {
		idList := opts.IDs
		idStrs := strings.Split(idList, ",")
		for _, s := range idStrs {
			var id int
			if _, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &id); err != nil {
				close(out)
				return nil, fmt.Errorf("invalid monitor ID: %s", s)
			}
			ids = append(ids, id)
		}
	}

	// Parse filter tags from comma-separated string
	var filterTags []string
	if opts.Tags != "" {
		tagStrs := strings.Split(opts.Tags, ",")
		for _, t := range tagStrs {
			filterTags = append(filterTags, strings.TrimSpace(t))
		}
	}

	go func() {
		defer close(out)
		client := internalhttp.GetHTTPClient(settings)
		// --update: scan existing monitor files and use their paths
		if opts.Update {
			monitorsDir := filepath.Join(settings.DataDir, "monitors")
			idToPath, err := storage.ExtractIntIDsFromJSONFiles(monitorsDir)
			if err != nil {
				out <- MonitorTargetResult{Err: fmt.Errorf("failed to scan directory: %w", err)}
				return
			}
			for id, path := range idToPath {
				out <- MonitorTargetResult{Target: MonitorTarget{ID: id, Path: path}}
			}
			return
		}
		// Always fetch from the list endpoint - it contains all the data we need
		// (including matching_downtimes which is not in the individual monitor endpoint)
		// Use pagination to handle large numbers of monitors
		var allMonitors []map[string]any
		page := 0
		pageSize := settings.PageSize
		for {
			url := fmt.Sprintf("https://api.%s/api/v1/monitor?page=%d&page_size=%d", settings.Site, page, pageSize)
			resp, err := client.Get(url)
			if err != nil {
				out <- MonitorTargetResult{Err: fmt.Errorf("failed to fetch monitors page %d: %w", page, err)}
				return
			}
			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(io.LimitReader(resp.Body, settings.HTTPMaxBodySize))
				resp.Body.Close()
				out <- MonitorTargetResult{Err: fmt.Errorf("API error on page %d: %s\n%s", page, resp.Status, string(body))}
				return
			}
			var monitorsList []map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&monitorsList); err != nil {
				resp.Body.Close()
				out <- MonitorTargetResult{Err: fmt.Errorf("failed to decode monitors page %d: %w", page, err)}
				return
			}
			resp.Body.Close()

			if len(monitorsList) == 0 {
				break
			}
			allMonitors = append(allMonitors, monitorsList...)

			// If we got fewer results than page size, this is the last page
			if len(monitorsList) < pageSize {
				break
			}
			page++
		}

		for _, mon := range allMonitors {
			// Filter by ID if specified and not --all
			if len(ids) > 0 {
				idVal, ok := mon["id"].(float64)
				if !ok {
					continue
				}
				idInt := int(idVal)
				found := false
				for _, want := range ids {
					if want == idInt {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}
			// Filter by tags/team
			tags := extractTags(mon)
			if opts.Team != "" && tags["team"] != opts.Team {
				continue
			}
			if len(filterTags) > 0 && !templating.HasAllTagsMap(tags, filterTags) {
				continue
			}
			// Filter by priority
			if opts.Priority > 0 {
				if p, ok := mon["priority"].(float64); !ok || int(p) != opts.Priority {
					continue
				}
			}
			// Yield monitor
			idVal, ok := mon["id"].(float64)
			if !ok {
				continue
			}
			idInt := int(idVal)
			out <- MonitorTargetResult{Target: MonitorTarget{ID: idInt, Path: "", Data: mon}}
		}
	}()
	return out, nil
}

// extractTags extracts tags from a monitor JSON object as a map[string]string
func extractTags(mon map[string]any) map[string]string {
	if raw, ok := mon["tags"]; ok {
		return templating.ExtractTagMap(raw, false)
	}
	return map[string]string{}
}

// DownloadMonitorWithOptions fetches a monitor and writes it to the specified path.
func DownloadMonitorWithOptions(target MonitorTarget, outputPath string) error {
	settings, err := config.LoadSettings()
	if err != nil {
		return err
	}
	var result map[string]any
	if target.Data != nil {
		result = target.Data
	} else {
		client := internalhttp.GetHTTPClient(settings)
		url := fmt.Sprintf("https://api.%s/api/v1/monitor/%d", settings.Site, target.ID)
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

	// Remove runtime state fields that cause unnecessary churn
	delete(result, "matching_downtimes")

	// Compute path if not provided
	targetPath := target.Path
	if targetPath == "" {
		// Build template pattern (output override or settings default)
		pattern := outputPath
		if pattern == "" {
			pattern = settings.MonitorsPathTemplate
		}
		pattern = templating.TranslatePlaceholders(pattern, templating.BuildMonitorBuiltins())

		// Extract and sanitize data for templating
		name := "untitled"
		if v, ok := result["name"].(string); ok && v != "" {
			name = storage.SanitizeFilename(v)
		}

		tagMap := templating.ExtractTagMap(result["tags"], true)
		var prio int
		if p, ok := result["priority"].(float64); ok {
			prio = int(p)
		}

		data := monitorTemplateData{
			DataDir:  settings.DataDir,
			ID:       target.ID,
			Name:     name,
			Tags:     tagMap,
			Priority: prio,
		}
		tmpl, err := template.New("path").Parse(pattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse path template: %v\n", err)
			targetPath = filepath.Join(settings.DataDir, "monitors", fmt.Sprintf("%d.json", target.ID))
		} else {
			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, data); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to execute path template: %v\n", err)
				targetPath = filepath.Join(settings.DataDir, "monitors", fmt.Sprintf("%d.json", target.ID))
			} else {
				targetPath = strings.ReplaceAll(buf.String(), "<no value>", "none")
			}
		}
	}
	if err := storage.WriteJSONFile(targetPath, result); err != nil {
		return err
	}
	fmt.Printf("Monitor saved to %s\n", targetPath)
	return nil
}
