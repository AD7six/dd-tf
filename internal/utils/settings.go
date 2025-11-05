package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

type Settings struct {
	APIKey                    string // Required, Datadog API key
	AppKey                    string // Required, Datadog application key
	APIDomain                 string // Which datadog site (https://docs.datadoghq.com/getting_started/site/) the account is in, defaults to "api.datadoghq.com"
	DashboardsDir             string // Where dashboard JSON files are stored
	DashboardsFilenamePattern string // Path pattern for dashboard files, defaults to "{id}.json"
	DashboardsPathPattern     string // Path pattern for dashboard full path, defaults to "{DASHBOARDS_DIR}/{id}.json"
	AddTitleToFileNames       bool   // Whether to append dashboard title to output filename
}

func LoadSettings() (*Settings, error) {
	// Try to load .env file if present
	_ = godotenv.Load()

	apiKey, err := getEnvRequired("DD_API_KEY")
	if err != nil {
		return nil, err
	}
	appKey, err := getEnvRequired("DD_APP_KEY")
	if err != nil {
		return nil, err
	}

	apiDomain := getEnv("DD_API_DOMAIN", "api.datadoghq.com")
	dashboardsDir := getEnv("DASHBOARDS_DIR", "data/dashboards")
	DashboardsFilenamePattern := getEnv("DASHBOARDS_FILENAME_PATTERN", "{id}.json")
	DashboardsPathPattern := getEnv("DASHBOARDS_PATH_PATTERN", filepath.Join(dashboardsDir, DashboardsFilenamePattern))
	addTitle := getEnvBool("DASHBOARDS_ADD_TITLE", true)

	return &Settings{
		APIKey:                apiKey,
		AppKey:                appKey,
		APIDomain:             apiDomain,
		DashboardsDir:         dashboardsDir,
		DashboardsPathPattern: DashboardsPathPattern,
		AddTitleToFileNames:   addTitle,
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
