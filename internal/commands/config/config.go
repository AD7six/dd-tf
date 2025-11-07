package config

import (
	"fmt"
	"reflect"
	"strings"

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

// Then loop with reflection:
func displaySettings(s *internalconfig.Settings) {
	v := reflect.ValueOf(*s)
	t := reflect.TypeOf(*s)

	// Find max key length for alignment (including colon)
	maxKeyLen := 0
	ddKeys := []string{"DD_API_KEY:", "DD_APP_KEY:", "DD_SITE:"}
	for _, key := range ddKeys {
		if len(key) > maxKeyLen {
			maxKeyLen = len(key)
		}
	}

	for i := 0; i < v.NumField(); i++ {
		envName := t.Field(i).Tag.Get("env")
		if !strings.HasPrefix(envName, "DD_") {
			keyWithColon := envName + ":"
			if len(keyWithColon) > maxKeyLen {
				maxKeyLen = len(keyWithColon)
			}
		}
	}

	fmt.Printf("# Datadog account:\n")
	fmt.Printf("%-*s  %s\n", maxKeyLen, "DD_API_KEY:", maskSecret(s.APIKey))
	fmt.Printf("%-*s  %s\n", maxKeyLen, "DD_APP_KEY:", maskSecret(s.AppKey))
	fmt.Printf("%-*s  %s\n", maxKeyLen, "DD_SITE:", s.Site)
	fmt.Printf("\n")

	fmt.Printf("# CLI Options:\n")
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		envName := field.Tag.Get("env")

		if strings.HasPrefix(envName, "DD_") {
			continue
		}

		fmt.Printf("%-*s  %v\n", maxKeyLen, envName+":", value.Interface())
	}
}

// maskSecret masks all but the first 2 and last 2 characters of a secret.
func maskSecret(secret string) string {
	if len(secret) <= 4 {
		return "****"
	}
	return secret[:2] + strings.Repeat("*", len(secret)-4) + secret[len(secret)-2:]
}
