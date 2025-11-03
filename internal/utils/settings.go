package utils

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Settings struct {
	APIKey        string
	AppKey        string
	APIDomain     string
	DashboardsDir string
}

func LoadSettings() (*Settings, error) {
	// Try to load .env file if present
	_ = godotenv.Load()

	apiKey := os.Getenv("DD_API_KEY")
	appKey := os.Getenv("DD_APP_KEY")
	apiDomain := os.Getenv("DD_API_DOMAIN")
	dashboardsDir := os.Getenv("DASHBOARDS_DIR")

	// Default to US site if not set
	if apiDomain == "" {
		apiDomain = "api.datadoghq.com"
	}
	// Default dashboards output directory
	if dashboardsDir == "" {
		dashboardsDir = "data/dashboards"
	}

	if apiKey == "" {
		return nil, fmt.Errorf("DD_API_KEY environment variable must be set")
	}
	if appKey == "" {
		return nil, fmt.Errorf("DD_APP_KEY environment variable must be set")
	}

	return &Settings{
		APIKey:        apiKey,
		AppKey:        appKey,
		APIDomain:     apiDomain,
		DashboardsDir: dashboardsDir,
	}, nil
}
