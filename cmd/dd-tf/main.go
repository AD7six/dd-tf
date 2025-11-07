package main

import (
	"github.com/AD7six/dd-tf/internal/commands/config"
	"github.com/AD7six/dd-tf/internal/commands/dashboards"
	"github.com/AD7six/dd-tf/internal/commands/monitors"
	"github.com/AD7six/dd-tf/internal/commands/version"
	"github.com/AD7six/dd-tf/internal/logging"
	"github.com/spf13/cobra"
)

var verbose bool

func main() {
	root := &cobra.Command{
		Use:   "dd-tf",
		Short: "Datadog Terraform management CLI",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if verbose {
				logging.InitLogger("debug")
			}
			logging.Logger.Debug("Verbose logging enabled")
		},
	}

	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose/debug output (shows curl commands)")

	root.AddCommand(config.NewConfigCmd())
	root.AddCommand(dashboards.NewDashboardsCmd())
	root.AddCommand(monitors.NewMonitorsCmd())
	root.AddCommand(version.NewVersionCmd())

	cobra.CheckErr(root.Execute())
}
