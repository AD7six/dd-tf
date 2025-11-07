package dashboards

import (
	"strings"
	"testing"

	"github.com/AD7six/dd-tf/internal/config"
)

func TestComputeDashboardPath_MissingFields(t *testing.T) {
	settings := &config.Settings{
		DashboardsPathTemplate: "data/dashboards/{id}-{title}.json",
	}

	t.Run("missing id field", func(t *testing.T) {
		dashboard := map[string]any{
			"title": "Test Dashboard",
		}

		_, err := ComputeDashboardPath(settings, dashboard, "")

		// Should return an error for missing ID
		if err == nil {
			t.Error("Expected error for missing 'id' field, got nil")
		}
	})

	t.Run("empty id field", func(t *testing.T) {
		dashboard := map[string]any{
			"id":    "",
			"title": "Test Dashboard",
		}

		_, err := ComputeDashboardPath(settings, dashboard, "")

		// Should return an error for empty ID
		if err == nil {
			t.Error("Expected error for empty 'id' field, got nil")
		}
	})

	t.Run("missing title field", func(t *testing.T) {
		dashboard := map[string]any{
			"id": "abc-123",
		}

		path, err := ComputeDashboardPath(settings, dashboard, "")
		if err != nil {
			t.Fatalf("ComputeDashboardPath() error = %v", err)
		}
		// Should use placeholder "untitled" instead of panicking
		if !strings.Contains(path, "untitled") {
			t.Errorf("Expected path to contain 'untitled', got: %s", path)
		}
		if !strings.Contains(path, "abc-123") {
			t.Errorf("Expected path to contain id 'abc-123', got: %s", path)
		}
	})

	t.Run("empty title field", func(t *testing.T) {
		dashboard := map[string]any{
			"id":    "def-456",
			"title": "",
		}

		path, err := ComputeDashboardPath(settings, dashboard, "")
		if err != nil {
			t.Fatalf("ComputeDashboardPath() error = %v", err)
		}

		// Should use placeholder "untitled" for empty title
		if !strings.Contains(path, "untitled") {
			t.Errorf("Expected path to contain 'untitled', got: %s", path)
		}
	})

	t.Run("both fields missing", func(t *testing.T) {
		dashboard := map[string]any{
			"some_other_field": "value",
		}

		_, err := ComputeDashboardPath(settings, dashboard, "")

		// Should return an error for missing ID
		if err == nil {
			t.Error("Expected error for missing 'id' field, got nil")
		}
	})

	t.Run("wrong type for id", func(t *testing.T) {
		dashboard := map[string]any{
			"id":    123, // number instead of string
			"title": "Test",
		}

		_, err := ComputeDashboardPath(settings, dashboard, "")

		// Should return an error for wrong type ID
		if err == nil {
			t.Error("Expected error for non-string 'id' field, got nil")
		}
	})

	t.Run("wrong type for title", func(t *testing.T) {
		dashboard := map[string]any{
			"id":    "xyz-789",
			"title": []string{"not", "a", "string"}, // array instead of string
		}

		path, err := ComputeDashboardPath(settings, dashboard, "")
		if err != nil {
			t.Fatalf("ComputeDashboardPath() error = %v", err)
		}

		// Should use placeholder instead of panicking
		if !strings.Contains(path, "untitled") {
			t.Errorf("Expected path to contain 'untitled' for non-string title, got: %s", path)
		}
		if !strings.Contains(path, "xyz-789") {
			t.Errorf("Expected path to contain id 'xyz-789', got: %s", path)
		}
	})

	t.Run("valid fields work normally", func(t *testing.T) {
		dashboard := map[string]any{
			"id":    "valid-123",
			"title": "My Dashboard",
		}

		path, err := ComputeDashboardPath(settings, dashboard, "")
		if err != nil {
			t.Fatalf("ComputeDashboardPath() error = %v", err)
		}

		expected := "data/dashboards/valid-123-My-Dashboard.json"
		if path != expected {
			t.Errorf("Expected path %s, got: %s", expected, path)
		}
	})
}

func TestComputeDashboardPath_WithTags(t *testing.T) {
	settings := &config.Settings{
		DashboardsPathTemplate: "data/dashboards/{team}/{title}-{id}.json",
	}

	t.Run("with valid tags", func(t *testing.T) {
		dashboard := map[string]any{
			"id":    "dash-123",
			"title": "Team Dashboard",
			"tags":  []interface{}{"team:platform", "env:prod"},
		}

		path, err := ComputeDashboardPath(settings, dashboard, "")
		if err != nil {
			t.Fatalf("ComputeDashboardPath() error = %v", err)
		}

		if !strings.Contains(path, "platform") {
			t.Errorf("Expected path to contain 'platform' team, got: %s", path)
		}
		if !strings.Contains(path, "Team-Dashboard") {
			t.Errorf("Expected path to contain sanitized title, got: %s", path)
		}
	})

	t.Run("missing tag uses default", func(t *testing.T) {
		dashboard := map[string]any{
			"id":    "dash-456",
			"title": "No Team Dashboard",
			"tags":  []interface{}{"env:prod"},
		}

		path, err := ComputeDashboardPath(settings, dashboard, "")
		if err != nil {
			t.Fatalf("ComputeDashboardPath() error = %v", err)
		}

		// Should use "none" as default for missing team tag
		if !strings.Contains(path, "none") {
			t.Errorf("Expected path to contain 'none' for missing team tag, got: %s", path)
		}
	})
}

func TestComputeDashboardPath_WithOutputOverride(t *testing.T) {
	settings := &config.Settings{
		DashboardsPathTemplate: "data/dashboards/{id}.json", // Default pattern
	}

	dashboard := map[string]any{
		"id":    "override-123",
		"title": "Override Test",
	}

	t.Run("uses output override when provided", func(t *testing.T) {
		outputPath := "/custom/path/{title}-{id}.json"
		path, err := ComputeDashboardPath(settings, dashboard, outputPath)
		if err != nil {
			t.Fatalf("ComputeDashboardPath() error = %v", err)
		}

		if !strings.Contains(path, "Override-Test") {
			t.Errorf("Expected path to use custom pattern with title, got: %s", path)
		}
		if !strings.Contains(path, "override-123") {
			t.Errorf("Expected path to contain id, got: %s", path)
		}
	})

	t.Run("uses default pattern when override is empty", func(t *testing.T) {
		path, err := ComputeDashboardPath(settings, dashboard, "")
		if err != nil {
			t.Fatalf("ComputeDashboardPath() error = %v", err)
		}

		expected := "data/dashboards/override-123.json"
		if path != expected {
			t.Errorf("Expected path %s, got: %s", expected, path)
		}
	})
}

func TestNormalizezDashboardID(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		normalized string
		wantErr    bool
	}{
		{
			name:    "valid dashboard ID",
			id:      "abc-123-def",
			wantErr: false,
		},
		{
			name:       "valid dashboard ID, needs normalization",
			id:         "ABC-123-DEF",
			normalized: "abc-123-def",
			wantErr:    false,
		},
		{
			name:    "empty dashboard ID",
			id:      "",
			wantErr: true,
		},
		{
			name:    "invalid format - missing dashes",
			id:      "abcdefghi",
			wantErr: true,
		},
		{
			name:    "invalid format - too short",
			id:      "abc-def",
			wantErr: true,
		},
		{
			name:    "invalid format - too long",
			id:      "abc-def-ghi-jkl",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized, err := normalizezDashboardID(tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("normalizezDashboardID(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
				return
			}

			// If we aren't expecting an error and there's no explicit
			// normalized value - we expect the input back
			if tt.normalized == "" && tt.wantErr == false {
				tt.normalized = tt.id
			}

			if normalized != tt.normalized {
				t.Errorf("normalizezDashboardID(%q) normalized = %q, want %q", tt.id, normalized, tt.normalized)
			}
		})
	}
}
