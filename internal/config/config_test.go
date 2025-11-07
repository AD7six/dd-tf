package config

import (
	"os"
	"reflect"
	"testing"
	"time"
)

func TestGetEnvRequired(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		envVal    string
		setEnv    bool
		want      string
		wantError bool
	}{
		{
			name:      "returns error when not set",
			key:       "REQUIRED_VAR_NOT_SET",
			setEnv:    false,
			wantError: true,
		},
		{
			name:      "returns error when empty string",
			key:       "REQUIRED_VAR_EMPTY",
			envVal:    "",
			setEnv:    true,
			wantError: true,
		},
		{
			name:      "returns value when set",
			key:       "REQUIRED_VAR_SET",
			envVal:    "required_value",
			setEnv:    true,
			want:      "required_value",
			wantError: false,
		},
		{
			name:      "returns value with whitespace preserved",
			key:       "REQUIRED_VAR_SPACES",
			envVal:    "  spaced value  ",
			setEnv:    true,
			want:      "  spaced value  ",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before and after
			os.Unsetenv(tt.key)
			defer os.Unsetenv(tt.key)

			if tt.setEnv {
				os.Setenv(tt.key, tt.envVal)
			}

			got, err := getEnvRequired(tt.key)
			if tt.wantError {
				if err == nil {
					t.Errorf("getEnvRequired(%q) expected error, got nil", tt.key)
				}
			} else {
				if err != nil {
					t.Errorf("getEnvRequired(%q) unexpected error: %v", tt.key, err)
				}
				if got != tt.want {
					t.Errorf("getEnvRequired(%q) = %q, want %q", tt.key, got, tt.want)
				}
			}
		})
	}
}

func TestLoadSettings(t *testing.T) {
	// Clean up any existing env vars
	cleanup := func() {
		os.Unsetenv("DD_API_KEY")
		os.Unsetenv("DD_APP_KEY")
		os.Unsetenv("DD_SITE")
		os.Unsetenv("DATA_DIR")
		os.Unsetenv("DASHBOARDS_PATH_TEMPLATE")
		os.Unsetenv("HTTP_TIMEOUT")
	}
	cleanup()
	defer cleanup()

	t.Run("returns error when DD_API_KEY missing", func(t *testing.T) {
		os.Setenv("DD_APP_KEY", "test_app_key")
		defer cleanup()

		_, err := LoadSettings()
		if err == nil {
			t.Error("LoadSettings() expected error when DD_API_KEY missing, got nil")
		}
	})

	t.Run("returns error when DD_APP_KEY missing", func(t *testing.T) {
		os.Setenv("DD_API_KEY", "test_api_key")
		defer cleanup()

		_, err := LoadSettings()
		if err == nil {
			t.Error("LoadSettings() expected error when DD_APP_KEY missing, got nil")
		}
	})

	t.Run("uses defaults when only required vars set", func(t *testing.T) {
		os.Setenv("DD_API_KEY", "test_api_key")
		os.Setenv("DD_APP_KEY", "test_app_key")
		defer cleanup()

		got, err := LoadSettings()
		if err != nil {
			t.Fatalf("LoadSettings() unexpected error: %v", err)
		}

		want := &Settings{
			APIKey:                 "test_api_key",
			AppKey:                 "test_app_key",
			Site:                   "datadoghq.com",
			DashboardsPathTemplate: "data/dashboards/{id}.json",
			MonitorsPathTemplate:   "data/monitors/{id}.json",
			HTTPTimeout:            60 * time.Second,
			HTTPMaxBodySize:        10 * 1024 * 1024, // 10MB
			PageSize:               1000,
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("LoadSettings() = %+v, want %+v", got, want)
		}
	})

	t.Run("parses custom HTTP timeout", func(t *testing.T) {
		os.Setenv("DD_API_KEY", "test_api_key")
		os.Setenv("DD_APP_KEY", "test_app_key")
		os.Setenv("HTTP_TIMEOUT", "30")
		defer cleanup()

		got, err := LoadSettings()
		if err != nil {
			t.Fatalf("LoadSettings() unexpected error: %v", err)
		}

		if got.HTTPTimeout != 30*time.Second {
			t.Errorf("LoadSettings().HTTPTimeout = %v, want 30s", got.HTTPTimeout)
		}
	})

	t.Run("uses default timeout for invalid HTTP_TIMEOUT", func(t *testing.T) {
		os.Setenv("DD_API_KEY", "test_api_key")
		os.Setenv("DD_APP_KEY", "test_app_key")
		os.Setenv("HTTP_TIMEOUT", "invalid")
		defer cleanup()

		got, err := LoadSettings()
		if err != nil {
			t.Fatalf("LoadSettings() unexpected error: %v", err)
		}

		// Invalid value falls back to 0 since getEnvInt can't parse it
		if got.HTTPTimeout != 0 {
			t.Errorf("LoadSettings().HTTPTimeout = %v, want 0s (fallback for invalid)", got.HTTPTimeout)
		}
	})

	t.Run("accepts zero HTTP timeout", func(t *testing.T) {
		os.Setenv("DD_API_KEY", "test_api_key")
		os.Setenv("DD_APP_KEY", "test_app_key")
		os.Setenv("HTTP_TIMEOUT", "0")
		defer cleanup()

		got, err := LoadSettings()
		if err != nil {
			t.Fatalf("LoadSettings() unexpected error: %v", err)
		}

		if got.HTTPTimeout != 0 {
			t.Errorf("LoadSettings().HTTPTimeout = %v, want 0s", got.HTTPTimeout)
		}
	})
}
