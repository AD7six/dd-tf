package templating

import (
	"strings"

	"github.com/AD7six/dd-tf/internal/storage"
)

// ExtractTagMap converts a raw tags value (typically []any or []interface{}) into a map[key]value.
// If sanitize is true, values are sanitized via storage.SanitizeFilename.
func ExtractTagMap(raw any, sanitize bool) map[string]string {
	tagMap := make(map[string]string)
	switch v := raw.(type) {
	case []interface{}:
		for _, t := range v {
			if s, ok := t.(string); ok {
				parts := strings.SplitN(s, ":", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					val := strings.TrimSpace(parts[1])
					if sanitize {
						val = storage.SanitizeFilename(val)
					}
					tagMap[key] = val
				}
			}
		}
	}
	return tagMap
}

// HasAllTagsMap checks if tags contain all required filterTags (case-insensitive),
// where filterTags are in the form key:value.
func HasAllTagsMap(tags map[string]string, filterTags []string) bool {
	if len(filterTags) == 0 {
		return true
	}
	for _, want := range filterTags {
		wantLower := strings.ToLower(want)
		found := false
		for k, v := range tags {
			if strings.ToLower(k+":"+v) == wantLower {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// HasAllTagsSlice checks if all filterTags are present in dashboardTags (both lowercase for comparison).
func HasAllTagsSlice(dashboardTags []string, filterTags []string) bool {
	if len(filterTags) == 0 {
		return true
	}
	set := make(map[string]struct{}, len(dashboardTags))
	for _, t := range dashboardTags {
		set[strings.ToLower(t)] = struct{}{}
	}
	for _, want := range filterTags {
		if _, ok := set[strings.ToLower(want)]; !ok {
			return false
		}
	}
	return true
}
