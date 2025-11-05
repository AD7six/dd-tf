package storage

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple alphanumeric",
			input: "MyDashboard123",
			want:  "MyDashboard123",
		},
		{
			name:  "spaces to hyphens",
			input: "My Dashboard Name",
			want:  "My-Dashboard-Name",
		},
		{
			name:  "special characters",
			input: "Dashboard: with/special\\chars!",
			want:  "Dashboard-with-special-chars",
		},
		{
			name:  "multiple consecutive special chars",
			input: "Test___---___Dashboard",
			want:  "Test-Dashboard",
		},
		{
			name:  "trims leading and trailing hyphens",
			input: "---Dashboard---",
			want:  "Dashboard",
		},
		{
			name:  "unicode characters",
			input: "Dashboard™®©",
			want:  "Dashboard",
		},
		{
			name:  "mixed case preserved",
			input: "CamelCaseDashboard",
			want:  "CamelCaseDashboard",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only special characters",
			input: "!@#$%^&*()",
			want:  "",
		},
		{
			name:  "parentheses and brackets",
			input: "Dashboard (prod) [v2]",
			want:  "Dashboard-prod-v2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeFilename(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractIDsFromJSONFiles(t *testing.T) {
	t.Run("returns error when directory does not exist", func(t *testing.T) {
		_, err := ExtractIDsFromJSONFiles("/nonexistent/path/that/does/not/exist")
		if err == nil {
			t.Error("ExtractIDsFromJSONFiles() expected error for nonexistent directory, got nil")
		}
	})

	t.Run("extracts IDs from valid JSON files", func(t *testing.T) {
		// Create temp directory
		tmpDir := t.TempDir()

		// Create test JSON files
		files := map[string]string{
			"dashboard1.json": `{"id": "abc-123", "title": "Dashboard 1"}`,
			"dashboard2.json": `{"id": "def-456", "title": "Dashboard 2"}`,
		}

		for filename, content := range files {
			path := filepath.Join(tmpDir, filename)
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to create test file %s: %v", filename, err)
			}
		}

		got, err := ExtractIDsFromJSONFiles(tmpDir)
		if err != nil {
			t.Fatalf("ExtractIDsFromJSONFiles() unexpected error: %v", err)
		}

		if len(got) != 2 {
			t.Errorf("ExtractIDsFromJSONFiles() returned %d items, want 2", len(got))
		}

		if _, ok := got["abc-123"]; !ok {
			t.Error("ExtractIDsFromJSONFiles() missing id 'abc-123'")
		}
		if _, ok := got["def-456"]; !ok {
			t.Error("ExtractIDsFromJSONFiles() missing id 'def-456'")
		}
	})

	t.Run("handles nested directories", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create nested structure
		subDir := filepath.Join(tmpDir, "subdir")
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("Failed to create subdirectory: %v", err)
		}

		// Create files in both directories
		rootFile := filepath.Join(tmpDir, "root.json")
		if err := os.WriteFile(rootFile, []byte(`{"id": "root-id"}`), 0644); err != nil {
			t.Fatalf("Failed to create root file: %v", err)
		}

		subFile := filepath.Join(subDir, "sub.json")
		if err := os.WriteFile(subFile, []byte(`{"id": "sub-id"}`), 0644); err != nil {
			t.Fatalf("Failed to create sub file: %v", err)
		}

		got, err := ExtractIDsFromJSONFiles(tmpDir)
		if err != nil {
			t.Fatalf("ExtractIDsFromJSONFiles() unexpected error: %v", err)
		}

		if len(got) != 2 {
			t.Errorf("ExtractIDsFromJSONFiles() returned %d items, want 2", len(got))
		}

		if _, ok := got["root-id"]; !ok {
			t.Error("ExtractIDsFromJSONFiles() missing id 'root-id'")
		}
		if _, ok := got["sub-id"]; !ok {
			t.Error("ExtractIDsFromJSONFiles() missing id 'sub-id'")
		}
	})

	t.Run("ignores non-JSON files", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create JSON and non-JSON files
		jsonFile := filepath.Join(tmpDir, "valid.json")
		if err := os.WriteFile(jsonFile, []byte(`{"id": "valid-id"}`), 0644); err != nil {
			t.Fatalf("Failed to create JSON file: %v", err)
		}

		txtFile := filepath.Join(tmpDir, "readme.txt")
		if err := os.WriteFile(txtFile, []byte("some text"), 0644); err != nil {
			t.Fatalf("Failed to create text file: %v", err)
		}

		got, err := ExtractIDsFromJSONFiles(tmpDir)
		if err != nil {
			t.Fatalf("ExtractIDsFromJSONFiles() unexpected error: %v", err)
		}

		if len(got) != 1 {
			t.Errorf("ExtractIDsFromJSONFiles() returned %d items, want 1", len(got))
		}

		if _, ok := got["valid-id"]; !ok {
			t.Error("ExtractIDsFromJSONFiles() missing id 'valid-id'")
		}
	})

	t.Run("skips files with invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()

		validFile := filepath.Join(tmpDir, "valid.json")
		if err := os.WriteFile(validFile, []byte(`{"id": "valid-id"}`), 0644); err != nil {
			t.Fatalf("Failed to create valid file: %v", err)
		}

		invalidFile := filepath.Join(tmpDir, "invalid.json")
		if err := os.WriteFile(invalidFile, []byte(`{invalid json`), 0644); err != nil {
			t.Fatalf("Failed to create invalid file: %v", err)
		}

		got, err := ExtractIDsFromJSONFiles(tmpDir)
		if err != nil {
			t.Fatalf("ExtractIDsFromJSONFiles() unexpected error: %v", err)
		}

		// Should only get the valid one
		if len(got) != 1 {
			t.Errorf("ExtractIDsFromJSONFiles() returned %d items, want 1", len(got))
		}

		if _, ok := got["valid-id"]; !ok {
			t.Error("ExtractIDsFromJSONFiles() missing id 'valid-id'")
		}
	})

	t.Run("skips files without id field", func(t *testing.T) {
		tmpDir := t.TempDir()

		withID := filepath.Join(tmpDir, "with-id.json")
		if err := os.WriteFile(withID, []byte(`{"id": "has-id"}`), 0644); err != nil {
			t.Fatalf("Failed to create file with ID: %v", err)
		}

		withoutID := filepath.Join(tmpDir, "without-id.json")
		if err := os.WriteFile(withoutID, []byte(`{"name": "no-id-field"}`), 0644); err != nil {
			t.Fatalf("Failed to create file without ID: %v", err)
		}

		got, err := ExtractIDsFromJSONFiles(tmpDir)
		if err != nil {
			t.Fatalf("ExtractIDsFromJSONFiles() unexpected error: %v", err)
		}

		if len(got) != 1 {
			t.Errorf("ExtractIDsFromJSONFiles() returned %d items, want 1", len(got))
		}

		if _, ok := got["has-id"]; !ok {
			t.Error("ExtractIDsFromJSONFiles() missing id 'has-id'")
		}
	})

	t.Run("skips files with empty id", func(t *testing.T) {
		tmpDir := t.TempDir()

		validFile := filepath.Join(tmpDir, "valid.json")
		if err := os.WriteFile(validFile, []byte(`{"id": "valid-id"}`), 0644); err != nil {
			t.Fatalf("Failed to create valid file: %v", err)
		}

		emptyIDFile := filepath.Join(tmpDir, "empty-id.json")
		if err := os.WriteFile(emptyIDFile, []byte(`{"id": ""}`), 0644); err != nil {
			t.Fatalf("Failed to create empty ID file: %v", err)
		}

		got, err := ExtractIDsFromJSONFiles(tmpDir)
		if err != nil {
			t.Fatalf("ExtractIDsFromJSONFiles() unexpected error: %v", err)
		}

		if len(got) != 1 {
			t.Errorf("ExtractIDsFromJSONFiles() returned %d items, want 1", len(got))
		}

		if _, ok := got["valid-id"]; !ok {
			t.Error("ExtractIDsFromJSONFiles() missing id 'valid-id'")
		}
	})

	t.Run("returns first occurrence for duplicate IDs", func(t *testing.T) {
		tmpDir := t.TempDir()

		firstFile := filepath.Join(tmpDir, "first.json")
		if err := os.WriteFile(firstFile, []byte(`{"id": "duplicate-id"}`), 0644); err != nil {
			t.Fatalf("Failed to create first file: %v", err)
		}

		secondFile := filepath.Join(tmpDir, "second.json")
		if err := os.WriteFile(secondFile, []byte(`{"id": "duplicate-id"}`), 0644); err != nil {
			t.Fatalf("Failed to create second file: %v", err)
		}

		got, err := ExtractIDsFromJSONFiles(tmpDir)
		if err != nil {
			t.Fatalf("ExtractIDsFromJSONFiles() unexpected error: %v", err)
		}

		if len(got) != 1 {
			t.Errorf("ExtractIDsFromJSONFiles() returned %d items, want 1 (duplicates should be deduplicated)", len(got))
		}

		path, ok := got["duplicate-id"]
		if !ok {
			t.Fatal("ExtractIDsFromJSONFiles() missing id 'duplicate-id'")
		}

		// Should be one of the two files (order may vary with filesystem)
		if path != firstFile && path != secondFile {
			t.Errorf("ExtractIDsFromJSONFiles() path = %s, want either %s or %s", path, firstFile, secondFile)
		}
	})

	t.Run("returns empty map for empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		got, err := ExtractIDsFromJSONFiles(tmpDir)
		if err != nil {
			t.Fatalf("ExtractIDsFromJSONFiles() unexpected error: %v", err)
		}

		want := map[string]string{}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("ExtractIDsFromJSONFiles() = %v, want %v", got, want)
		}
	})

	t.Run("skips files exceeding size limit", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a valid small file
		smallFile := filepath.Join(tmpDir, "small.json")
		if err := os.WriteFile(smallFile, []byte(`{"id": "small-id"}`), 0644); err != nil {
			t.Fatalf("Failed to create small file: %v", err)
		}

		// Create a file larger than 1MB (maxJSONFileSize)
		largeFile := filepath.Join(tmpDir, "large.json")
		prefix := []byte(`{"id": "large-id", "data": "`)
		suffix := []byte(`"}`)
		paddingSize := maxJSONFileSize + 1000 - len(prefix) - len(suffix)
		padding := bytes.Repeat([]byte("x"), paddingSize)

		largeContent := append(append(prefix, padding...), suffix...)

		if err := os.WriteFile(largeFile, largeContent, 0644); err != nil {
			t.Fatalf("Failed to create large file: %v", err)
		}

		got, err := ExtractIDsFromJSONFiles(tmpDir)
		if err != nil {
			t.Fatalf("ExtractIDsFromJSONFiles() unexpected error: %v", err)
		}

		// Should only have the small file's ID
		if len(got) != 1 {
			t.Errorf("ExtractIDsFromJSONFiles() returned %d items, want 1 (large file should be skipped)", len(got))
		}

		if _, ok := got["small-id"]; !ok {
			t.Error("ExtractIDsFromJSONFiles() missing id 'small-id'")
		}

		// Should NOT have the large file's ID
		if _, ok := got["large-id"]; ok {
			t.Error("ExtractIDsFromJSONFiles() should have skipped 'large-id' from oversized file")
		}
	})
}

func TestWriteJSONFile(t *testing.T) {
	t.Run("writes valid JSON file", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "test.json")

		data := map[string]any{
			"id":    "test-123",
			"title": "Test Dashboard",
			"value": 42,
		}

		err := WriteJSONFile(path, data)
		if err != nil {
			t.Fatalf("WriteJSONFile() unexpected error: %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Error("WriteJSONFile() did not create file")
		}

		// Verify content
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Failed to read written file: %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(content, &result); err != nil {
			t.Fatalf("Written file is not valid JSON: %v", err)
		}

		// JSON numbers are unmarshaled as float64, so compare accordingly
		expected := map[string]any{
			"id":    "test-123",
			"title": "Test Dashboard",
			"value": float64(42),
		}

		if !reflect.DeepEqual(result, expected) {
			t.Errorf("WriteJSONFile() wrote %v, want %v", result, expected)
		}
	})

	t.Run("creates nested directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "level1", "level2", "level3", "test.json")

		data := map[string]any{"id": "nested"}

		err := WriteJSONFile(path, data)
		if err != nil {
			t.Fatalf("WriteJSONFile() unexpected error: %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Error("WriteJSONFile() did not create file in nested directories")
		}

		// Verify all directories were created
		dir := filepath.Dir(path)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Error("WriteJSONFile() did not create parent directories")
		}
	})

	t.Run("formats JSON with indentation", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "formatted.json")

		data := map[string]any{
			"id":   "test",
			"name": "Test",
		}

		err := WriteJSONFile(path, data)
		if err != nil {
			t.Fatalf("WriteJSONFile() unexpected error: %v", err)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		// Check for indentation (2 spaces)
		contentStr := string(content)
		if !strings.Contains(contentStr, "  ") {
			t.Error("WriteJSONFile() did not format JSON with indentation")
		}

		// Check for newlines (formatted JSON has multiple lines)
		if strings.Count(contentStr, "\n") < 2 {
			t.Error("WriteJSONFile() did not format JSON with newlines")
		}
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "overwrite.json")

		// Write initial data
		initial := map[string]any{"version": 1}
		if err := WriteJSONFile(path, initial); err != nil {
			t.Fatalf("Initial write failed: %v", err)
		}

		// Overwrite with new data
		updated := map[string]any{"version": 2}
		if err := WriteJSONFile(path, updated); err != nil {
			t.Fatalf("WriteJSONFile() unexpected error: %v", err)
		}

		// Verify updated content
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(content, &result); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		if result["version"] != float64(2) {
			t.Errorf("WriteJSONFile() did not overwrite file, got version %v, want 2", result["version"])
		}
	})

	t.Run("handles complex nested structures", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "complex.json")

		data := map[string]any{
			"id": "complex",
			"nested": map[string]any{
				"array": []any{1, 2, 3},
				"object": map[string]any{
					"key": "value",
				},
			},
		}

		err := WriteJSONFile(path, data)
		if err != nil {
			t.Fatalf("WriteJSONFile() unexpected error: %v", err)
		}

		// Read back and verify
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(content, &result); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		// Deep comparison is tricky with interface{}, so just verify structure exists
		if _, ok := result["nested"]; !ok {
			t.Error("WriteJSONFile() did not preserve nested structure")
		}
	})

	t.Run("returns error for invalid path", func(t *testing.T) {
		// Try to write to a path that can't be created (parent is a file, not a directory)
		tmpDir := t.TempDir()
		parentFile := filepath.Join(tmpDir, "file.txt")

		// Create a regular file
		if err := os.WriteFile(parentFile, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create parent file: %v", err)
		}

		// Try to create a file "inside" the regular file (impossible)
		invalidPath := filepath.Join(parentFile, "child.json")

		err := WriteJSONFile(invalidPath, map[string]any{"id": "test"})
		if err == nil {
			t.Error("WriteJSONFile() expected error for invalid path, got nil")
		}
	})
}
