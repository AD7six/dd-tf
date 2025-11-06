package main

import (
	"os"

	"github.com/AD7six/dd-tf/internal/commands/dashboards"
	"github.com/AD7six/dd-tf/internal/commands/version"
	"github.com/spf13/cobra"
)

var verbose bool

func main() {
	root := &cobra.Command{
		Use:   "dd-tf",
		Short: "Datadog Terraform management CLI",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if verbose {
				os.Setenv("DEBUG", "1")
			}
		},
	}

	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose/debug output (shows curl commands)")

	root.AddCommand(dashboards.NewDashboardsCmd())
	root.AddCommand(version.NewVersionCmd())

	cobra.CheckErr(root.Execute())
}
