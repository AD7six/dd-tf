package dashboards

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/AD7six/dd-tf/internal/utils"
)

func TestComputeDashboardPath_MissingFields(t *testing.T) {
	settings := &utils.Settings{
		DashboardsDir:         "/test/dashboards",
		DashboardsPathPattern: "{DASHBOARDS_DIR}/{id}-{title}.json",
	}

	t.Run("missing id field", func(t *testing.T) {
		dashboard := map[string]any{
			"title": "Test Dashboard",
		}

		path := ComputeDashboardPath(settings, dashboard, "")

		// Should use placeholder "unknown-id" instead of panicking
		if !strings.Contains(path, "unknown-id") {
			t.Errorf("Expected path to contain 'unknown-id', got: %s", path)
		}
	})

	t.Run("empty id field", func(t *testing.T) {
		dashboard := map[string]any{
			"id":    "",
			"title": "Test Dashboard",
		}

		path := ComputeDashboardPath(settings, dashboard, "")

		// Should use placeholder "unknown-id" for empty id
		if !strings.Contains(path, "unknown-id") {
			t.Errorf("Expected path to contain 'unknown-id', got: %s", path)
		}
	})

	t.Run("missing title field", func(t *testing.T) {
		dashboard := map[string]any{
			"id": "abc-123",
		}

		path := ComputeDashboardPath(settings, dashboard, "")

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

		path := ComputeDashboardPath(settings, dashboard, "")

		// Should use placeholder "untitled" for empty title
		if !strings.Contains(path, "untitled") {
			t.Errorf("Expected path to contain 'untitled', got: %s", path)
		}
	})

	t.Run("both fields missing", func(t *testing.T) {
		dashboard := map[string]any{
			"some_other_field": "value",
		}

		path := ComputeDashboardPath(settings, dashboard, "")

		// Should use both placeholders
		if !strings.Contains(path, "unknown-id") {
			t.Errorf("Expected path to contain 'unknown-id', got: %s", path)
		}
		if !strings.Contains(path, "untitled") {
			t.Errorf("Expected path to contain 'untitled', got: %s", path)
		}
	})

	t.Run("wrong type for id", func(t *testing.T) {
		dashboard := map[string]any{
			"id":    123, // number instead of string
			"title": "Test",
		}

		path := ComputeDashboardPath(settings, dashboard, "")

		// Should use placeholder instead of panicking
		if !strings.Contains(path, "unknown-id") {
			t.Errorf("Expected path to contain 'unknown-id' for non-string id, got: %s", path)
		}
	})

	t.Run("wrong type for title", func(t *testing.T) {
		dashboard := map[string]any{
			"id":    "xyz-789",
			"title": []string{"not", "a", "string"}, // array instead of string
		}

		path := ComputeDashboardPath(settings, dashboard, "")

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

		path := ComputeDashboardPath(settings, dashboard, "")

		expected := filepath.Join("/test/dashboards", "valid-123-My-Dashboard.json")
		if path != expected {
			t.Errorf("Expected path %s, got: %s", expected, path)
		}
	})
}

func TestComputeDashboardPath_WithTags(t *testing.T) {
	settings := &utils.Settings{
		DashboardsDir:         "/test/dashboards",
		DashboardsPathPattern: "{DASHBOARDS_DIR}/{team}/{title}-{id}.json",
	}

	t.Run("with valid tags", func(t *testing.T) {
		dashboard := map[string]any{
			"id":    "dash-123",
			"title": "Team Dashboard",
			"tags":  []interface{}{"team:platform", "env:prod"},
		}

		path := ComputeDashboardPath(settings, dashboard, "")

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

		path := ComputeDashboardPath(settings, dashboard, "")

		// Should use "none" as default for missing team tag
		if !strings.Contains(path, "none") {
			t.Errorf("Expected path to contain 'none' for missing team tag, got: %s", path)
		}
	})
}

func TestComputeDashboardPath_WithOutputOverride(t *testing.T) {
	settings := &utils.Settings{
		DashboardsDir:         "/test/dashboards",
		DashboardsPathPattern: "{DASHBOARDS_DIR}/{id}.json", // Default pattern
	}

	dashboard := map[string]any{
		"id":    "override-123",
		"title": "Override Test",
	}

	t.Run("uses output override when provided", func(t *testing.T) {
		outputPath := "/custom/path/{title}-{id}.json"
		path := ComputeDashboardPath(settings, dashboard, outputPath)

		if !strings.Contains(path, "Override-Test") {
			t.Errorf("Expected path to use custom pattern with title, got: %s", path)
		}
		if !strings.Contains(path, "override-123") {
			t.Errorf("Expected path to contain id, got: %s", path)
		}
	})

	t.Run("uses default pattern when override is empty", func(t *testing.T) {
		path := ComputeDashboardPath(settings, dashboard, "")

		expected := filepath.Join("/test/dashboards", "override-123.json")
		if path != expected {
			t.Errorf("Expected path %s, got: %s", expected, path)
		}
	})
}
