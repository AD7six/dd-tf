package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

//go:embed defaults.env
var embeddedDefaults string

// Settings contains configuration for the Datadog API client and dashboard management.
type Settings struct {
	APIKey                 string        `env:"DD_API_KEY"`               // Required, Datadog API key
	AppKey                 string        `env:"DD_APP_KEY"`               // Required, Datadog application key
	Site                   string        `env:"DD_SITE"`                  // Datadog site (e.g., datadoghq.com). Used to build https://api.{Site}
	DashboardsPathTemplate string        `env:"DASHBOARDS_PATH_TEMPLATE"` // Path template for dashboard full path, defaults to "{DATA_DIR}/dashboards/{id}.json"
	MonitorsPathTemplate   string        `env:"MONITORS_PATH_TEMPLATE"`   // Path template for monitor full path, defaults to "{DATA_DIR}/monitors/{id}.json"
	HTTPTimeout            time.Duration `env:"HTTP_TIMEOUT"`             // HTTP client timeout, defaults to 60 seconds
	HTTPMaxBodySize        int64         `env:"HTTP_MAX_BODY_SIZE"`       // Maximum allowed API response body size in bytes, defaults to 10MB
	PageSize               int           `env:"PAGE_SIZE"`                // Number of results per page for index endpoints, defaults to 1000
}

// LoadSettings loads configuration from environment variables and optional .env files.
// Embedded defaults are loaded first, then .env file (if present) overrides them.
// Required environment variables: DD_API_KEY, DD_APP_KEY.
// Optional variables: DD_SITE, DATA_DIR, DASHBOARDS_PATH_TEMPLATE, MONITORS_PATH_TEMPLATE, HTTP_TIMEOUT, HTTP_MAX_BODY_SIZE, PAGE_SIZE.
func LoadSettings() (*Settings, error) {
	// Load embedded defaults first
	envMap, err := godotenv.Unmarshal(embeddedDefaults)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: error parsing embedded defaults: %v\n", err)
	} else {
		// Set defaults only if not already set
		for k, v := range envMap {
			if os.Getenv(k) == "" {
				os.Setenv(k, v)
			}
		}
	}

	// Then load .env (if it exists) to override defaults
	if _, err := os.Stat(".env"); err == nil {
		err := godotenv.Overload(".env")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: error loading .env file: %v\n", err)
		}
	}

	apiKey, err := getEnvRequired("DD_API_KEY")
	if err != nil {
		return nil, err
	}
	appKey, err := getEnvRequired("DD_APP_KEY")
	if err != nil {
		return nil, err
	}

	site := getEnv("DD_SITE", "datadoghq.com")
	site = strings.TrimSpace(strings.ToLower(site))
	if strings.HasPrefix(site, "api.") {
		fmt.Fprintf(os.Stderr, "Warning: DD_SITE value \"%s\" should not have prefix 'api.', removing\n", site)
		site = strings.TrimPrefix(site, "api.")
	}

	dataDir := getEnv("DATA_DIR", "data")
	dashboardsPathTemplate := getEnv("DASHBOARDS_PATH_TEMPLATE", filepath.Join(dataDir, "dashboards", "{id}.json"))
	monitorsPathTemplate := getEnv("MONITORS_PATH_TEMPLATE", filepath.Join(dataDir, "monitors", "{id}.json"))

	httpTimeout := time.Duration(getEnvInt("HTTP_TIMEOUT", 60)) * time.Second
	HTTPMaxBodySize := int64(getEnvInt("HTTP_MAX_BODY_SIZE", 10*1024*1024)) // 10MB default
	pageSize := getEnvInt("PAGE_SIZE", 1000)

	return &Settings{
		APIKey:                 apiKey,
		AppKey:                 appKey,
		Site:                   site,
		DashboardsPathTemplate: dashboardsPathTemplate,
		MonitorsPathTemplate:   monitorsPathTemplate,
		HTTPTimeout:            httpTimeout,
		HTTPMaxBodySize:        HTTPMaxBodySize,
		PageSize:               pageSize,
	}, nil
}

// get the env variable with a default
func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

// get the env variable or raise an error
func getEnvRequired(key string) (string, error) {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v, nil
	}
	return "", fmt.Errorf("%s environment variable must be set", key)
}

// getEnvBool returns a boolean env var with support for common truthy/falsey strings, defaulting when unset/empty.
func getEnvBool(key string, def bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "t", "yes", "y", "on":
		return true
	case "0", "false", "f", "no", "n", "off":
		return false
	default:
		return def
	}
}

// getEnvInt returns an integer env var, defaulting when unset/empty or invalid.
func getEnvInt(key string, def int) int {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def
	}
	if i, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
		return i
	}
	return def
}
