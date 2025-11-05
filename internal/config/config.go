package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Settings contains configuration for the Datadog API client and dashboard management.
type Settings struct {
	APIKey                 string        // Required, Datadog API key
	AppKey                 string        // Required, Datadog application key
	Site                   string        // Datadog site (e.g., datadoghq.com). Used to build https://api.{Site}
	DashboardsDir          string        // Where dashboard JSON files are stored
	DashboardsPathTemplate string        // Path template for dashboard full path, defaults to "{DASHBOARDS_DIR}/{id}.json"
	HTTPTimeout            time.Duration // HTTP client timeout, defaults to 60 seconds
}

// LoadSettings loads configuration from environment variables and optional .env file.
// Required environment variables: DD_API_KEY, DD_APP_KEY.
// Optional variables: DD_API_DOMAIN, DASHBOARDS_DIR, DASHBOARDS_FILENAME_PATTERN, DASHBOARDS_PATH_PATTERN.
func LoadSettings() (*Settings, error) {
	// If .env exists, try to load it
	if _, err := os.Stat(".env"); err == nil {
		err := godotenv.Load()
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

	// Determine site from DD_SITE (e.g., datadoghq.com).
	site := getEnv("DD_SITE", "datadoghq.com")

	dashboardsDir := getEnv("DASHBOARDS_DIR", "data/dashboards")
	dashboardsPathTemplate := getEnv("DASHBOARDS_PATH_TEMPLATE", filepath.Join(dashboardsDir, "{id}.json"))

	// Parse HTTP timeout from environment (in seconds), default to 60
	httpTimeout := time.Duration(getEnvInt("DD_HTTP_TIMEOUT", 60)) * time.Second

	return &Settings{
		APIKey:                 apiKey,
		AppKey:                 appKey,
		Site:                   site,
		DashboardsDir:          dashboardsDir,
		DashboardsPathTemplate: dashboardsPathTemplate,
		HTTPTimeout:            httpTimeout,
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
