package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/AD7six/dd-tf/internal/logging"
)

const (
	// maxJSONFileSize all files should be less than 1MB so use that as a cut
	// off to avoid reading invalid, extremely large, files
	maxJSONFileSize = 1024 * 1024 // 1MB
)

var (
	// nonAlphanumericRegex matches any non-alphanumeric characters for filename sanitization
	nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9]+`)
)

// WriteJSONFile writes data as JSON to the specified path with indentation.
// Creates the parent directory if it doesn't exist.
func WriteJSONFile(path string, data any) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write JSON file
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("failed to write JSON: %w", err)
	}

	return nil
}

// SanitizeFilename replaces non-alphanumeric characters with hyphens and trims.
func SanitizeFilename(name string) string {
	return strings.Trim(nonAlphanumericRegex.ReplaceAllString(name, "-"), "-")
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
			logging.Logger.Warn("failed to access file", "path", path, "error", err)
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

		// Check file size before reading
		if info.Size() > maxJSONFileSize {
			logging.Logger.Warn("skipping file (too large)", "path", path, "size", info.Size(), "max", maxJSONFileSize)
			return nil
		}

		// Read JSON file and extract "id" field
		data, err := os.ReadFile(path)
		if err != nil {
			logging.Logger.Warn("failed to read file", "path", path, "error", err)
			return nil
		}

		var content map[string]any
		if err := json.Unmarshal(data, &content); err != nil {
			logging.Logger.Warn("failed to parse JSON", "path", path, "error", err)
			return nil
		}

		id, ok := content["id"].(string)
		if !ok || id == "" {
			logging.Logger.Warn("no valid id field", "path", path)
			return nil
		}

		// Store the first occurrence; duplicates are logged but not stored
		if existing, exists := result[id]; exists {
			logging.Logger.Warn("duplicate id", "id", id, "path", path, "existing", existing)
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

// ExtractIntIDsFromJSONFiles scans a directory recursively for JSON files and extracts integer IDs from their content.
// Returns a map of id -> absolute file path.
// Each JSON file must have an "id" field at the top level that is a number (Datadog monitors use integer IDs).
func ExtractIntIDsFromJSONFiles(dir string) (map[int]string, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory does not exist: %s", dir)
	}

	result := make(map[int]string)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logging.Logger.Warn("failed to access file", "path", path, "error", err)
			return nil // Continue walking despite errors
		}

		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".json") {
			return nil
		}
		if info.Size() > maxJSONFileSize {
			logging.Logger.Warn("skipping file (too large)", "path", path, "size", info.Size(), "max", maxJSONFileSize)
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			logging.Logger.Warn("failed to read file", "path", path, "error", err)
			return nil
		}
		var content map[string]any
		if err := json.Unmarshal(data, &content); err != nil {
			logging.Logger.Warn("failed to parse JSON", "path", path, "error", err)
			return nil
		}
		// JSON decoder uses float64 for numbers by default
		if f, ok := content["id"].(float64); ok {
			id := int(f)
			if id == 0 {
				logging.Logger.Warn("invalid id value", "path", path)
				return nil
			}
			if existing, exists := result[id]; exists {
				logging.Logger.Warn("duplicate id", "id", id, "path", path, "existing", existing)
			} else {
				result[id] = path
			}
		} else {
			logging.Logger.Warn("no numeric id field", "path", path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}
	return result, nil
}
