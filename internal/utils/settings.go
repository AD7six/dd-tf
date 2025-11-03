package utils

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Settings struct {
	APIKey        string
	AppKey        string
	APIDomain     string // e.g., "api.datadoghq.com" - depends on which datadog site (https://docs.datadoghq.com/getting_started/site/) the account is in
	DashboardsDir string // Where dashboard JSON files are stored
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

	return &Settings{
		APIKey:        apiKey,
		AppKey:        appKey,
		APIDomain:     apiDomain,
		DashboardsDir: dashboardsDir,
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
