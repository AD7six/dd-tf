package main

import (
	"github.com/AD7six/dd-tf/internal/commands/dashboards"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "dd-tf",
		Short: "Datadog Terraform management CLI",
	}

	root.AddCommand(dashboards.NewDashboardsCmd())

	cobra.CheckErr(root.Execute())
}
