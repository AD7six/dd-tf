package templating

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

var (
	// PlaceholderRegex matches placeholder patterns like {word}
	PlaceholderRegex = regexp.MustCompile(`\{([A-Za-z0-9_\-]+)\}`)

	// EnvVarRegex matches environment variable naming pattern (uppercase letters, numbers, underscores)
	EnvVarRegex = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
)

// replaceEnvVars replaces environment variable placeholders in a string.
// Placeholders matching the pattern {VAR_NAME} where VAR_NAME is all uppercase
// with underscores are replaced with the value of the environment variable.
// If the environment variable is not set or empty, the placeholder is left as-is.
func replaceEnvVars(pattern string) string {
	return PlaceholderRegex.ReplaceAllStringFunc(pattern, func(m string) string {
		sub := PlaceholderRegex.FindStringSubmatch(m)
		if len(sub) != 2 {
			return m
		}
		name := sub[1]

		// Check if it looks like an environment variable (uppercase with underscores)
		if EnvVarRegex.MatchString(name) {
			// Get the environment variable value
			if val := os.Getenv(name); val != "" {
				return val
			}
		}

		// Leave other placeholders as-is
		return m
	})
}

// TranslatePlaceholders converts placeholders like {id} into Go template expressions.
// Builtins should map placeholders (e.g. "{id}") to template expressions (e.g. "{{.ID}}").
// Environment variable placeholders (e.g. {MY_VAR}) are replaced with their env var values first.
// Any remaining {word} will be mapped to {{.Tags.word}}.
func TranslatePlaceholders(pattern string, builtins map[string]string) string {
	// First, replace environment variables
	p := replaceEnvVars(pattern)

	// Then, replace all builtins
	for k, v := range builtins {
		p = strings.ReplaceAll(p, k, v)
	}

	// Finally, handle remaining placeholders as tags
	p = PlaceholderRegex.ReplaceAllStringFunc(p, func(m string) string {
		sub := PlaceholderRegex.FindStringSubmatch(m)
		if len(sub) != 2 {
			return m
		}
		name := sub[1]
		return fmt.Sprintf("{{.Tags.%s}}", name)
	})
	return p
}

// BuildDashboardBuiltins returns the builtins map for dashboard path templates.
func BuildDashboardBuiltins() map[string]string {
	return map[string]string{
		"{DATA_DIR}": "{{.DataDir}}",
		"{id}":       "{{.ID}}",
		"{title}":    "{{.Title}}",
		"{name}":     "{{.Title}}", // Alias for consistency with monitors
	}
}

// BuildMonitorBuiltins returns the builtins map for monitor path templates.
func BuildMonitorBuiltins() map[string]string {
	return map[string]string{
		"{DATA_DIR}": "{{.DataDir}}",
		"{id}":       "{{.ID}}",
		"{name}":     "{{.Name}}",
		"{title}":    "{{.Name}}", // Alias for consistency with dashboards
		"{priority}": "{{.Priority}}",
	}
}

// ExtractStaticPrefix returns the longest static prefix from a path template.
// For example, "data/dashboards/{id}.json" returns "data/dashboards".
// Environment variable placeholders (e.g., {MY_VAR}) and {DATA_DIR} are expanded before extraction.
// For example, if MY_BASE=/opt/data, then "{MY_BASE}/dashboards/{id}.json" returns "/opt/data/dashboards".
// This is used to determine the base directory to scan when updating existing files.
func ExtractStaticPrefix(pathTemplate string) string {
	if pathTemplate == "" {
		return ""
	}

	// First, expand environment variable placeholders
	expanded := replaceEnvVars(pathTemplate)

	// Also handle {DATA_DIR} placeholder by replacing with DATA_DIR env var if set
	if dataDir := os.Getenv("DATA_DIR"); dataDir != "" {
		expanded = strings.ReplaceAll(expanded, "{DATA_DIR}", dataDir)
	}

	// Find the first remaining placeholder
	idx := strings.Index(expanded, "{")
	if idx == -1 {
		// No placeholders, return the directory portion
		dir := filepath.Dir(expanded)
		if dir == "." {
			return ""
		}
		return dir
	}

	if idx == 0 {
		// Placeholder at the start, no static prefix
		return ""
	}

	// Get everything before the first placeholder
	prefix := expanded[:idx]

	// Remove trailing path separator
	prefix = strings.TrimRight(prefix, string(filepath.Separator))

	// Handle edge case where we get "." or empty string
	if prefix == "." || prefix == "" {
		return ""
	}

	return prefix
}

// ComputePathFromTemplate executes a Go template to compute a file path.
// It handles template parsing, execution, and error fallback.
// The pattern should already be translated (using TranslatePlaceholders).
// Returns the computed path, replacing "<no value>" with "none".
func ComputePathFromTemplate(pattern string, data any, fallbackPath string) string {
	// Parse template
	tmpl, err := template.New("path").Parse(pattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to parse path template: %v\n", err)
		return fallbackPath
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to execute path template: %v\n", err)
		return fallbackPath
	}

	// Replace "<no value>" (from missing template fields) with "none"
	result := strings.ReplaceAll(buf.String(), "<no value>", "none")
	return result
}
