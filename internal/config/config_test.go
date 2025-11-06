package config

import (
	"os"
	"reflect"
	"testing"
	"time"
)

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		def    string
		envVal string
		setEnv bool
		want   string
	}{
		{
			name:   "returns default when not set",
			key:    "TEST_VAR_NOT_SET",
			def:    "default_value",
			setEnv: false,
			want:   "default_value",
		},
		{
			name:   "returns env value when set",
			key:    "TEST_VAR_SET",
			def:    "default_value",
			envVal: "custom_value",
			setEnv: true,
			want:   "custom_value",
		},
		{
			name:   "returns default when env is empty string",
			key:    "TEST_VAR_EMPTY",
			def:    "default_value",
			envVal: "",
			setEnv: true,
			want:   "default_value",
		},
		{
			name:   "returns value with spaces",
			key:    "TEST_VAR_SPACES",
			def:    "default",
			envVal: "  value with spaces  ",
			setEnv: true,
			want:   "  value with spaces  ",
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

			got := getEnv(tt.key, tt.def)
			if got != tt.want {
				t.Errorf("getEnv(%q, %q) = %q, want %q", tt.key, tt.def, got, tt.want)
			}
		})
	}
}

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

func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		def    bool
		envVal string
		setEnv bool
		want   bool
	}{
		// Not set cases
		{
			name:   "returns default true when not set",
			key:    "BOOL_VAR_NOT_SET_T",
			def:    true,
			setEnv: false,
			want:   true,
		},
		{
			name:   "returns default false when not set",
			key:    "BOOL_VAR_NOT_SET_F",
			def:    false,
			setEnv: false,
			want:   false,
		},
		// Empty string
		{
			name:   "returns default when empty string",
			key:    "BOOL_VAR_EMPTY",
			def:    true,
			envVal: "",
			setEnv: true,
			want:   true,
		},
		// Truthy values
		{
			name:   "recognizes '1' as true",
			key:    "BOOL_VAR_1",
			def:    false,
			envVal: "1",
			setEnv: true,
			want:   true,
		},
		{
			name:   "recognizes 'true' as true",
			key:    "BOOL_VAR_TRUE",
			def:    false,
			envVal: "true",
			setEnv: true,
			want:   true,
		},
		{
			name:   "recognizes 'TRUE' (case-insensitive) as true",
			key:    "BOOL_VAR_TRUE_UPPER",
			def:    false,
			envVal: "TRUE",
			setEnv: true,
			want:   true,
		},
		{
			name:   "recognizes 't' as true",
			key:    "BOOL_VAR_T",
			def:    false,
			envVal: "t",
			setEnv: true,
			want:   true,
		},
		{
			name:   "recognizes 'yes' as true",
			key:    "BOOL_VAR_YES",
			def:    false,
			envVal: "yes",
			setEnv: true,
			want:   true,
		},
		{
			name:   "recognizes 'y' as true",
			key:    "BOOL_VAR_Y",
			def:    false,
			envVal: "y",
			setEnv: true,
			want:   true,
		},
		{
			name:   "recognizes 'on' as true",
			key:    "BOOL_VAR_ON",
			def:    false,
			envVal: "on",
			setEnv: true,
			want:   true,
		},
		// Falsey values
		{
			name:   "recognizes '0' as false",
			key:    "BOOL_VAR_0",
			def:    true,
			envVal: "0",
			setEnv: true,
			want:   false,
		},
		{
			name:   "recognizes 'false' as false",
			key:    "BOOL_VAR_FALSE",
			def:    true,
			envVal: "false",
			setEnv: true,
			want:   false,
		},
		{
			name:   "recognizes 'FALSE' (case-insensitive) as false",
			key:    "BOOL_VAR_FALSE_UPPER",
			def:    true,
			envVal: "FALSE",
			setEnv: true,
			want:   false,
		},
		{
			name:   "recognizes 'f' as false",
			key:    "BOOL_VAR_F",
			def:    true,
			envVal: "f",
			setEnv: true,
			want:   false,
		},
		{
			name:   "recognizes 'no' as false",
			key:    "BOOL_VAR_NO",
			def:    true,
			envVal: "no",
			setEnv: true,
			want:   false,
		},
		{
			name:   "recognizes 'n' as false",
			key:    "BOOL_VAR_N",
			def:    true,
			envVal: "n",
			setEnv: true,
			want:   false,
		},
		{
			name:   "recognizes 'off' as false",
			key:    "BOOL_VAR_OFF",
			def:    true,
			envVal: "off",
			setEnv: true,
			want:   false,
		},
		// Invalid values fall back to default
		{
			name:   "returns default for invalid value (def=true)",
			key:    "BOOL_VAR_INVALID_T",
			def:    true,
			envVal: "invalid",
			setEnv: true,
			want:   true,
		},
		{
			name:   "returns default for invalid value (def=false)",
			key:    "BOOL_VAR_INVALID_F",
			def:    false,
			envVal: "nope",
			setEnv: true,
			want:   false,
		},
		// Whitespace handling
		{
			name:   "trims whitespace from 'true'",
			key:    "BOOL_VAR_SPACES_TRUE",
			def:    false,
			envVal: "  true  ",
			setEnv: true,
			want:   true,
		},
		{
			name:   "trims whitespace from 'false'",
			key:    "BOOL_VAR_SPACES_FALSE",
			def:    true,
			envVal: "  false  ",
			setEnv: true,
			want:   false,
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

			got := getEnvBool(tt.key, tt.def)
			if got != tt.want {
				t.Errorf("getEnvBool(%q, %v) with env=%q = %v, want %v",
					tt.key, tt.def, tt.envVal, got, tt.want)
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
			DataDir:                "data",
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

		if got.HTTPTimeout != 60*time.Second {
			t.Errorf("LoadSettings().HTTPTimeout = %v, want 60s (default)", got.HTTPTimeout)
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
