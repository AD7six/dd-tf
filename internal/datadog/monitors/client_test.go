package monitors

import (
	"testing"

	"github.com/AD7six/dd-tf/internal/datadog/templating"
)

func TestTranslateToTemplate(t *testing.T) {
	cases := []struct {
		name, input, expected string
	}{
		{"translates id", "monitors/{id}.json", "monitors/{{.ID}}.json"},
		{"translates name", "monitors/{name}-{id}.json", "monitors/{{.Name}}-{{.ID}}.json"},
		{"translates priority", "monitors/p{priority}/{id}.json", "monitors/p{{.Priority}}/{{.ID}}.json"},
		{"translates tag placeholder", "monitors/{team}/{id}.json", "monitors/{{.Tags.team}}/{{.ID}}.json"},
		{"mixed builtins and tags", "monitors/{team}/p{priority}/{name}-{id}.json", "monitors/{{.Tags.team}}/p{{.Priority}}/{{.Name}}-{{.ID}}.json"},
		{"no placeholders", "monitors/monitor.json", "monitors/monitor.json"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := templating.TranslatePlaceholders(c.input, templating.BuildMonitorBuiltins())
			if got != c.expected {
				t.Errorf("TranslatePlaceholders(%q) = %q, want %q", c.input, got, c.expected)
			}
		})
	}
}

func TestExtractTags(t *testing.T) {
	cases := []struct {
		name     string
		monitor  map[string]any
		expected map[string]string
	}{
		{"extracts tags", map[string]any{"id": float64(1), "tags": []interface{}{"team:platform", "env:prod"}}, map[string]string{"team": "platform", "env": "prod"}},
		{"filters invalid tag entries", map[string]any{"id": float64(1), "tags": []interface{}{"team:platform", "invalid", "service:api"}}, map[string]string{"team": "platform", "service": "api"}},
		{"missing tags field", map[string]any{"id": float64(1)}, map[string]string{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := extractTags(c.monitor)
			if len(got) != len(c.expected) {
				t.Errorf("extractTags length = %d, want %d", len(got), len(c.expected))
			}
			for k, v := range c.expected {
				if got[k] != v {
					t.Errorf("extractTags[%s] = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

// Removed broad DownloadMonitorWithOptions panic-guard tests; they were
// checking side-effects instead of path construction logic.

// Removed monitorTemplateData trivial field assertion tests (struct is a plain data holder).

// Removed DownloadOptions struct field echo test (no behavior validated).

// Removed MonitorTarget construction tests (no logic beyond assignment).

// Removed MonitorTargetResult trivial getter tests.
