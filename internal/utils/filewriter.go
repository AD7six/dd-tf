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
