package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// JSONWriter is a small helper to write pretty JSON files under a base directory.
// It is constructed with a target directory (e.g., settings.DashboardsDir).
// Methods return the written file path.
type JSONWriter struct {
	Dir      string
	AddTitle bool
}

func NewJSONWriter(dir string, addTitle bool) *JSONWriter {
	return &JSONWriter{Dir: dir, AddTitle: addTitle}
}

// SavePretty writes data as pretty-printed JSON to <Dir>/<id>-<sanitized title>.json
func (w *JSONWriter) SavePretty(id, title string, data any) (string, error) {
	if w.Dir == "" {
		return "", fmt.Errorf("JSONWriter.Dir must not be empty")
	}

	// Ensure output directory exists
	if err := os.MkdirAll(w.Dir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Remove any existing files for the same ID (e.g., old titles)
	patterns := []string{
		filepath.Join(w.Dir, fmt.Sprintf("%s-*.json", id)), // titled files
		filepath.Join(w.Dir, fmt.Sprintf("%s.json", id)),   // non-titled file
	}
	var matches []string
	for _, p := range patterns {
		ms, _ := filepath.Glob(p)
		matches = append(matches, ms...)
	}
	for _, m := range matches {
		if err := os.Remove(m); err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to remove previous file '%s': %w", m, err)
		}
	}

	filename := filepath.Join(w.Dir, fmt.Sprintf("%s.json", id))
	if w.AddTitle {
		safeTitle := SanitizeFilename(title)
		filename = filepath.Join(w.Dir, fmt.Sprintf("%s-%s.json", id, safeTitle))
	}

	f, err := os.Create(filename)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return "", fmt.Errorf("failed to write JSON: %w", err)
	}

	return filename, nil
}

// SanitizeFilename replaces non-alphanumeric characters with underscores and trims.
func SanitizeFilename(name string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	return strings.Trim(re.ReplaceAllString(name, "-"), "-")
}

// ExtractIDsFromJSONFiles scans a directory for JSON files and extracts IDs from their content.
// Returns a map of id -> absolute file path.
// Each JSON file must have an "id" field at the top level.
func ExtractIDsFromJSONFiles(dir string) (map[string]string, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory does not exist: %s", dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	result := make(map[string]string)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filename := entry.Name()
		if !strings.HasSuffix(filename, ".json") {
			continue
		}

		// Read JSON file and extract "id" field
		absPath := filepath.Join(dir, filename)
		data, err := os.ReadFile(absPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to read %s: %v\n", filename, err)
			continue
		}

		var content map[string]interface{}
		if err := json.Unmarshal(data, &content); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", filename, err)
			continue
		}

		id, ok := content["id"].(string)
		if !ok || id == "" {
			fmt.Fprintf(os.Stderr, "Warning: no valid 'id' field in %s\n", filename)
			continue
		}

		// Store the first occurrence; duplicates are logged but not stored
		if existing, exists := result[id]; exists {
			fmt.Fprintf(os.Stderr, "Warning: duplicate id '%s' in %s (already found in %s)\n", id, filename, filepath.Base(existing))
		} else {
			result[id] = absPath
		}
	}

	return result, nil
}
