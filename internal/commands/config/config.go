package config

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/AD7six/dd-tf/internal/config"
	internalconfig "github.com/AD7six/dd-tf/internal/config"
	"github.com/AD7six/dd-tf/internal/utils"
	"github.com/spf13/cobra"
)

// NewConfigCmd returns a cobra command that displays current configuration.
func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show runtime configuration",
		Long:  "Shows the current configuration values, with defaults applied as ENV_VAR: value pairs.",
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

func displaySettings(s *internalconfig.Settings) {
	v := reflect.ValueOf(*s)
	t := reflect.TypeOf(*s)

	// Initialize maxKeyLen to the longest DD_ key (DD_APP_KEY = 10 chars)
	maxKeyLen := 10

	// Parse defaults.env first
	defaults, err := config.GetDefaultEnv()
	if err != nil {
		defaults = make(map[string]string)
	}

	// Track which keys we've seen from Settings struct
	seen := make(map[string]struct{})

	// Collect all keys to determine maxKeyLen
	for i := 0; i < v.NumField(); i++ {
		envName := t.Field(i).Tag.Get("env")
		seen[envName] = struct{}{}
		if len(envName) > maxKeyLen {
			maxKeyLen = len(envName)
		}
	}

	// Include keys from defaults.env in maxKeyLen calculation
	for k := range defaults {
		if len(k) > maxKeyLen {
			maxKeyLen = len(k)
		}
	}

	// Print Datadog account section
	fmt.Printf("# Datadog account:\n")
	fmt.Printf("%-*s:  %s\n", maxKeyLen, "DD_API_KEY", utils.MaskSecret(s.APIKey))
	fmt.Printf("%-*s:  %s\n", maxKeyLen, "DD_APP_KEY", utils.MaskSecret(s.AppKey))
	fmt.Printf("%-*s:  %s\n", maxKeyLen, "DD_SITE", s.Site)
	fmt.Printf("\n")

	// Collect CLI options from Settings (non-DD_ keys)
	type kv struct {
		key   string
		value any
	}
	var cliOptions []kv
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		envName := field.Tag.Get("env")

		if strings.HasPrefix(envName, "DD_") {
			continue
		}

		cliOptions = append(cliOptions, kv{key: envName, value: value.Interface()})
	}

	// Add anything else from defaults.env not in Settings
	for k, v := range defaults {
		if _, already := seen[k]; !already {
			cliOptions = append(cliOptions, kv{key: k, value: v})
		}
	}

	// Sort CLI options alphabetically
	sort.Slice(cliOptions, func(i, j int) bool {
		return strings.ToLower(cliOptions[i].key) < strings.ToLower(cliOptions[j].key)
	})

	// Print CLI Options section
	fmt.Printf("# CLI Options:\n")
	for _, opt := range cliOptions {
		fmt.Printf("%-*s:  %v\n", maxKeyLen, opt.key, opt.value)
	}
}
