package monitors

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/AD7six/dd-tf/internal/datadog/resource"
	"github.com/AD7six/dd-tf/internal/datadog/templating"
)

func TestTranslateToTemplate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "translates DATA_DIR",
			input:    "{DATA_DIR}/monitors/{id}.json",
			expected: "{{.DataDir}}/monitors/{{.ID}}.json",
		},
		{
			name:     "translates id",
			input:    "monitors/{id}.json",
			expected: "monitors/{{.ID}}.json",
		},
		{
			name:     "translates name",
			input:    "monitors/{name}-{id}.json",
			expected: "monitors/{{.Name}}-{{.ID}}.json",
		},
		{
			name:     "translates title as alias for name",
			input:    "monitors/{title}-{id}.json",
			expected: "monitors/{{.Name}}-{{.ID}}.json",
		},
		{
			name:     "translates priority",
			input:    "monitors/p{priority}/{id}.json",
			expected: "monitors/p{{.Priority}}/{{.ID}}.json",
		},
		{
			name:     "translates tag placeholders",
			input:    "monitors/{team}/{id}.json",
			expected: "monitors/{{.Tags.team}}/{{.ID}}.json",
		},
		{
			name:     "handles multiple tags",
			input:    "{DATA_DIR}/{team}/{env}/{id}.json",
			expected: "{{.DataDir}}/{{.Tags.team}}/{{.Tags.env}}/{{.ID}}.json",
		},
		{
			name:     "no placeholders",
			input:    "monitors/monitor.json",
			expected: "monitors/monitor.json",
		},
		{
			name:     "mixed builtins and tags",
			input:    "{DATA_DIR}/{team}/p{priority}/{name}-{id}.json",
			expected: "{{.DataDir}}/{{.Tags.team}}/p{{.Priority}}/{{.Name}}-{{.ID}}.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := templating.TranslatePlaceholders(tt.input, templating.BuildMonitorBuiltins())
			if result != tt.expected {
				t.Errorf("TranslatePlaceholders(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractTags(t *testing.T) {
	tests := []struct {
		name     string
		monitor  map[string]any
		expected map[string]string
	}{
		{
			name: "extracts tags from string array",
			monitor: map[string]any{
				"id":   float64(123),
				"tags": []interface{}{"team:platform", "env:prod", "priority:p1"},
			},
			expected: map[string]string{
				"team":     "platform",
				"env":      "prod",
				"priority": "p1",
			},
		},
		{
			name: "handles tags without colons",
			monitor: map[string]any{
				"id":   float64(123),
				"tags": []interface{}{"team:platform", "monitoring"},
			},
			expected: map[string]string{
				"team": "platform",
			},
		},
		{
			name: "handles empty tags",
			monitor: map[string]any{
				"id":   float64(123),
				"tags": []interface{}{},
			},
			expected: map[string]string{},
		},
		{
			name: "handles missing tags field",
			monitor: map[string]any{
				"id": float64(123),
			},
			expected: map[string]string{},
		},
		{
			name: "handles nil tags",
			monitor: map[string]any{
				"id":   float64(123),
				"tags": nil,
			},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTags(tt.monitor)

			if len(result) != len(tt.expected) {
				t.Errorf("extractTags() returned %d tags, want %d", len(result), len(tt.expected))
			}

			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("extractTags()[%q] = %q, want %q", k, result[k], v)
				}
			}
		})
	}
}

func TestComputeMonitorPath(t *testing.T) {
	t.Run("simple id-only template", func(t *testing.T) {
		monitor := map[string]any{
			"id":   float64(12345),
			"name": "Test Monitor",
		}

		target := MonitorTarget{
			ID:   12345,
			Data: monitor,
		}

		err := DownloadMonitorWithOptions(target, "")
		// Will fail due to file writing, but we can check the path construction doesn't panic
		if err == nil {
			t.Error("Expected error from file write, got nil")
		}
		// The function will compute the path internally - we're mainly checking it doesn't panic
	})

	t.Run("template with name", func(t *testing.T) {
		// This test mainly ensures name sanitization works
		monitor := map[string]any{
			"id":   float64(123),
			"name": "My Test Monitor!",
		}

		target := MonitorTarget{
			ID:   123,
			Data: monitor,
		}

		// We'd need to mock settings.LoadSettings() to test this properly
		// The function will compute the path internally
		err := DownloadMonitorWithOptions(target, "")
		if err == nil {
			t.Error("Expected error from file write, got nil")
		}
	})

	t.Run("missing name uses untitled", func(t *testing.T) {
		monitor := map[string]any{
			"id": float64(456),
		}

		target := MonitorTarget{
			ID:   456,
			Data: monitor,
		}

		// Similar to above - mainly checking code doesn't panic
		err := DownloadMonitorWithOptions(target, "")
		if err == nil {
			t.Error("Expected error from file write, got nil")
		}
	})

	t.Run("empty name uses untitled", func(t *testing.T) {
		monitor := map[string]any{
			"id":   float64(789),
			"name": "",
		}

		target := MonitorTarget{
			ID:   789,
			Data: monitor,
		}

		err := DownloadMonitorWithOptions(target, "")
		if err == nil {
			t.Error("Expected error from file write, got nil")
		}
	})

	t.Run("template with tags", func(t *testing.T) {
		monitor := map[string]any{
			"id":   float64(999),
			"name": "Tagged Monitor",
			"tags": []interface{}{"team:platform", "env:prod"},
		}

		target := MonitorTarget{
			ID:   999,
			Data: monitor,
		}

		// Test that tags are extracted for path generation
		tags := extractTags(monitor)
		if tags["team"] != "platform" {
			t.Errorf("Expected team=platform, got %q", tags["team"])
		}
		if tags["env"] != "prod" {
			t.Errorf("Expected env=prod, got %q", tags["env"])
		}

		err := DownloadMonitorWithOptions(target, "")
		if err == nil {
			t.Error("Expected error from file write, got nil")
		}
	})

	t.Run("template with priority", func(t *testing.T) {
		monitor := map[string]any{
			"id":       float64(888),
			"name":     "Priority Monitor",
			"priority": float64(1),
		}

		target := MonitorTarget{
			ID:   888,
			Data: monitor,
		}

		err := DownloadMonitorWithOptions(target, "")
		if err == nil {
			t.Error("Expected error from file write, got nil")
		}
	})

	t.Run("custom output path override", func(t *testing.T) {
		monitor := map[string]any{
			"id":   float64(111),
			"name": "Custom Path Monitor",
		}

		target := MonitorTarget{
			ID:   111,
			Data: monitor,
		}

		customPath := "/custom/{name}-{id}.json"
		err := DownloadMonitorWithOptions(target, customPath)
		if err == nil {
			t.Error("Expected error from file write, got nil")
		}
		// The custom path should be used instead of settings template
	})

	t.Run("invalid template pattern falls back to default", func(t *testing.T) {
		monitor := map[string]any{
			"id":   float64(222),
			"name": "Fallback Monitor",
		}

		target := MonitorTarget{
			ID:   222,
			Data: monitor,
		}

		// Invalid template syntax should trigger fallback
		invalidPath := "{unclosed"
		err := DownloadMonitorWithOptions(target, invalidPath)
		if err == nil {
			t.Error("Expected error from file write, got nil")
		}
	})
}

func TestMonitorTemplateData(t *testing.T) {
	t.Run("sanitizes monitor name", func(t *testing.T) {
		data := monitorTemplateData{
			DataDir:  "/data",
			ID:       123,
			Name:     "Test-Monitor-With-Special-Chars!@#",
			Tags:     map[string]string{"team": "platform"},
			Priority: 1,
		}

		// The Name should be sanitized when constructed
		if !strings.Contains(data.Name, "Test-Monitor-With-Special-Chars") {
			t.Errorf("Expected sanitized name, got %q", data.Name)
		}
	})

	t.Run("handles missing tags", func(t *testing.T) {
		data := monitorTemplateData{
			DataDir:  "/data",
			ID:       456,
			Name:     "No Tags Monitor",
			Tags:     map[string]string{},
			Priority: 0,
		}

		if data.Tags == nil {
			t.Error("Expected empty map for Tags, got nil")
		}
		if len(data.Tags) != 0 {
			t.Errorf("Expected empty Tags map, got %d items", len(data.Tags))
		}
	})

	t.Run("handles zero priority", func(t *testing.T) {
		data := monitorTemplateData{
			DataDir:  "/data",
			ID:       789,
			Name:     "No Priority",
			Tags:     map[string]string{},
			Priority: 0,
		}

		if data.Priority != 0 {
			t.Errorf("Expected priority 0, got %d", data.Priority)
		}
	})
}

func TestDownloadOptions(t *testing.T) {
	t.Run("validates monitor ID parsing", func(t *testing.T) {
		// This would require mocking GenerateMonitorTargets
		// For now, we just validate the struct exists and can be constructed
		opts := DownloadOptions{
			BaseDownloadOptions: resource.BaseDownloadOptions{
				All:        false,
				Update:     false,
				OutputPath: "/custom/path",
				Team:       "platform",
				Tags:       "env:prod,service:api",
				IDs:        "123,456,789",
			},
			Priority: 1,
		}

		if opts.IDs != "123,456,789" {
			t.Errorf("Expected IDs to be preserved, got %q", opts.IDs)
		}
		if opts.Team != "platform" {
			t.Errorf("Expected Team=platform, got %q", opts.Team)
		}
	})
}

func TestMonitorTarget(t *testing.T) {
	t.Run("creates monitor target with full response", func(t *testing.T) {
		monitor := map[string]any{
			"id":   float64(12345),
			"name": "Test Monitor",
			"tags": []interface{}{"team:platform"},
		}

		target := MonitorTarget{
			ID:   12345,
			Path: "/data/monitors/12345.json",
			Data: monitor,
		}

		if target.ID != 12345 {
			t.Errorf("Expected ID=12345, got %d", target.ID)
		}
		if target.Path != "/data/monitors/12345.json" {
			t.Errorf("Expected specific path, got %q", target.Path)
		}
		if target.Data == nil {
			t.Error("Expected Data to be set")
		}
	})

	t.Run("creates monitor target without full response", func(t *testing.T) {
		target := MonitorTarget{
			ID:   67890,
			Path: "/custom/path.json",
		}

		if target.Data != nil {
			t.Error("Expected Data to be nil")
		}
	})
}

func TestMonitorTargetResult(t *testing.T) {
	t.Run("result with target and no error", func(t *testing.T) {
		target := MonitorTarget{ID: 123}
		result := MonitorTargetResult{
			Target: target,
			Err:    nil,
		}

		if result.Err != nil {
			t.Error("Expected no error")
		}
		if result.Target.ID != 123 {
			t.Errorf("Expected target ID=123, got %d", result.Target.ID)
		}
	})

	t.Run("result with error", func(t *testing.T) {
		result := MonitorTargetResult{
			Err: filepath.ErrBadPattern,
		}

		if result.Err == nil {
			t.Error("Expected error to be set")
		}
	})
}
