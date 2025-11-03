package utils

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Settings struct {
	APIKey              string
	AppKey              string
	APIDomain           string // e.g., "api.datadoghq.com" - depends on which datadog site (https://docs.datadoghq.com/getting_started/site/) the account is in
	DashboardsDir       string // Where dashboard JSON files are stored
	AddTitleToFileNames bool   // Whether to append dashboard title to output filename
	Retry429MaxAttempts int    // How many times to retry on HTTP 429 responses
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
	addTitle := getEnvBool("DASHBOARDS_ADD_TITLE", true)
	retry429 := getEnvInt("HTTP_RETRY_429_ATTEMPTS", 3)

	return &Settings{
		APIKey:              apiKey,
		AppKey:              appKey,
		APIDomain:           apiDomain,
		DashboardsDir:       dashboardsDir,
		AddTitleToFileNames: addTitle,
		Retry429MaxAttempts: retry429,
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

// getEnvInt returns an int env var or a default if unset/invalid.
func getEnvInt(key string, def int) int {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return def
	}
	if i, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
		return i
	}
	return def
}
