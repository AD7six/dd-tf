package config

import (
	"fmt"
	"os"

	internalconfig "github.com/AD7six/dd-tf/internal/config"
	"github.com/spf13/cobra"
)

// NewConfigCmd returns a cobra command that displays current configuration.
func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show effective configuration",
		Long:  "Shows the current configuration values as ENV_VAR: value pairs.",
		RunE: func(cmd *cobra.Command, args []string) error {
			settings, err := internalconfig.LoadSettings()
			if err != nil {
				return err
			}

			displaySettings(settings)
			return nil
		},
	}

	return cmd
}

// displaySettings prints each config as "ENV_VAR: value" matching names from defaults.env
func displaySettings(s *internalconfig.Settings) {
	// Required
	fmt.Printf("DD_API_KEY: %s\n", maskSecret(s.APIKey))
	fmt.Printf("DD_APP_KEY: %s\n", maskSecret(s.AppKey))

	// Optional (match defaults.env variable names and value forms)
	fmt.Printf("DD_SITE: %s\n", s.Site)
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}
	fmt.Printf("DATA_DIR: %s\n", dataDir)
	fmt.Printf("DASHBOARDS_PATH_TEMPLATE: %s\n", s.DashboardsPathTemplate)
	fmt.Printf("MONITORS_PATH_TEMPLATE: %s\n", s.MonitorsPathTemplate)
	// HTTP_TIMEOUT is defined as seconds in defaults.env, convert duration to whole seconds
	fmt.Printf("HTTP_TIMEOUT: %d\n", int(s.HTTPTimeout.Seconds()))
	fmt.Printf("HTTP_MAX_BODY_SIZE: %d\n", s.HTTPMaxBodySize)
}

// maskSecret masks all but the last 4 characters of a secret.
func maskSecret(secret string) string {
	if len(secret) <= 4 {
		return "****"
	}
	return "****" + secret[len(secret)-4:]
}
