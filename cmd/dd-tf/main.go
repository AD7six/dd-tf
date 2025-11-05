package main

import (
	"github.com/AD7six/dd-tf/internal/commands/dashboards"
	"github.com/AD7six/dd-tf/internal/commands/version"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "dd-tf",
		Short: "Datadog Terraform management CLI",
	}

	root.AddCommand(dashboards.NewDashboardsCmd())
	root.AddCommand(version.NewVersionCmd())

	cobra.CheckErr(root.Execute())
}
