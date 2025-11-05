package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SanitizeFilename replaces non-alphanumeric characters with hyphens and trims.
func SanitizeFilename(name string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	return strings.Trim(re.ReplaceAllString(name, "-"), "-")
}

// ExtractIDsFromJSONFiles scans a directory recursively for JSON files and extracts IDs from their content.
// Returns a map of id -> absolute file path.
// Each JSON file must have an "id" field at the top level.
func ExtractIDsFromJSONFiles(dir string) (map[string]string, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory does not exist: %s", dir)
	}

	result := make(map[string]string)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to access %s: %v\n", path, err)
			return nil // Continue walking despite errors
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process .json files
		if !strings.HasSuffix(info.Name(), ".json") {
			return nil
		}

		// Read JSON file and extract "id" field
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to read %s: %v\n", path, err)
			return nil
		}

		var content map[string]any
		if err := json.Unmarshal(data, &content); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", path, err)
			return nil
		}

		id, ok := content["id"].(string)
		if !ok || id == "" {
			fmt.Fprintf(os.Stderr, "Warning: no valid 'id' field in %s\n", path)
			return nil
		}

		// Store the first occurrence; duplicates are logged but not stored
		if existing, exists := result[id]; exists {
			fmt.Fprintf(os.Stderr, "Warning: duplicate id '%s' in %s (already found in %s)\n", id, path, existing)
		} else {
			result[id] = path
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return result, nil
}
