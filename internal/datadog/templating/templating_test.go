package templating

import (
	"os"
	"testing"
)

func TestTranslatePlaceholders(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		builtins map[string]string
		envVars  map[string]string // environment variables to set for the test
		want     string
	}{
		{
			name:     "no placeholders",
			pattern:  "static/path/file.json",
			builtins: map[string]string{},
			want:     "static/path/file.json",
		},
		{
			name:    "builtin placeholder",
			pattern: "data/dashboards/{id}.json",
			builtins: map[string]string{
				"{id}": "{{.ID}}",
			},
			want: "data/dashboards/{{.ID}}.json",
		},
		{
			name:    "tag placeholder",
			pattern: "data/dashboards/{team}/{id}.json",
			builtins: map[string]string{
				"{id}": "{{.ID}}",
			},
			want: "data/dashboards/{{.Tags.team}}/{{.ID}}.json",
		},
		{
			name:    "environment variable placeholder",
			pattern: "{MY_CUSTOM_PATH}/dashboards/{id}.json",
			builtins: map[string]string{
				"{id}": "{{.ID}}",
			},
			envVars: map[string]string{
				"MY_CUSTOM_PATH": "/var/data",
			},
			want: "/var/data/dashboards/{{.ID}}.json",
		},
		{
			name:    "environment variable not set falls back to tag",
			pattern: "{MISSING_VAR}/dashboards/{id}.json",
			builtins: map[string]string{
				"{id}": "{{.ID}}",
			},
			want: "{{.Tags.MISSING_VAR}}/dashboards/{{.ID}}.json",
		},
		{
			name:    "multiple env vars",
			pattern: "{BASE_DIR}/{ENVIRONMENT}/{id}.json",
			builtins: map[string]string{
				"{id}": "{{.ID}}",
			},
			envVars: map[string]string{
				"BASE_DIR":    "/opt/data",
				"ENVIRONMENT": "production",
			},
			want: "/opt/data/production/{{.ID}}.json",
		},
		{
			name:    "lowercase not treated as env var",
			pattern: "{my_var}/{id}.json",
			builtins: map[string]string{
				"{id}": "{{.ID}}",
			},
			envVars: map[string]string{
				"my_var": "/should/not/be/used",
			},
			want: "{{.Tags.my_var}}/{{.ID}}.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables for this test
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			got := TranslatePlaceholders(tt.pattern, tt.builtins)
			if got != tt.want {
				t.Errorf("TranslatePlaceholders(%q, %v) = %q, want %q", tt.pattern, tt.builtins, got, tt.want)
			}
		})
	}
}

func TestExtractStaticPrefix(t *testing.T) {
	tests := []struct {
		name         string
		pathTemplate string
		envVars      map[string]string // environment variables to set for the test
		want         string
	}{
		{
			name:         "simple template with placeholder",
			pathTemplate: "data/dashboards/{id}.json",
			want:         "data/dashboards",
		},
		{
			name:         "template with SOME_ENV_DIR placeholder first - with env var",
			pathTemplate: "{SOME_ENV_DIR}/dashboards/{id}.json",
			envVars: map[string]string{
				"SOME_ENV_DIR": "/opt/data",
			},
			want: "/opt/data/dashboards",
		},
		{
			name:         "template with multiple placeholders",
			pathTemplate: "data/dashboards/{team}/{id}.json",
			want:         "data/dashboards",
		},
		{
			name:         "no placeholders - file path",
			pathTemplate: "data/dashboards/static.json",
			want:         "data/dashboards",
		},
		{
			name:         "absolute path with placeholder",
			pathTemplate: "/var/data/dashboards/{id}.json",
			want:         "/var/data/dashboards",
		},
		{
			name:         "placeholder at start",
			pathTemplate: "{id}/dashboard.json",
			want:         "",
		},
		{
			name:         "complex nested path",
			pathTemplate: "/home/user/data/dashboards/{team}/{env}/{id}.json",
			want:         "/home/user/data/dashboards",
		},
		{
			name:         "env var at start expands to static prefix",
			pathTemplate: "{MY_BASE_DIR}/dashboards/{id}.json",
			envVars: map[string]string{
				"MY_BASE_DIR": "/opt/data",
			},
			want: "/opt/data/dashboards",
		},
		{
			name:         "env var in middle expands",
			pathTemplate: "/var/{MY_SUBDIR}/dashboards/{id}.json",
			envVars: map[string]string{
				"MY_SUBDIR": "custom",
			},
			want: "/var/custom/dashboards",
		},
		{
			name:         "multiple env vars expand",
			pathTemplate: "{BASE_DIR}/{ENVIRONMENT}/dashboards/{id}.json",
			envVars: map[string]string{
				"BASE_DIR":    "/opt/app",
				"ENVIRONMENT": "production",
			},
			want: "/opt/app/production/dashboards",
		},
		{
			name:         "unset env var not expanded - no static prefix",
			pathTemplate: "{MISSING_VAR}/dashboards/{id}.json",
			want:         "",
		},
		{
			name:         "unset env var in middle - stops at first placeholder",
			pathTemplate: "/var/{MISSING_VAR}/dashboards/{id}.json",
			want:         "/var",
		},
		{
			name:         "env var fully expands template with no other placeholders",
			pathTemplate: "{FULL_PATH}",
			envVars: map[string]string{
				"FULL_PATH": "/complete/path/to/file.json",
			},
			want: "/complete/path/to",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables for this test
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			got := ExtractStaticPrefix(tt.pathTemplate)
			if got != tt.want {
				t.Errorf("ExtractStaticPrefix(%q) = %q, want %q", tt.pathTemplate, got, tt.want)
			}
		})
	}
}
