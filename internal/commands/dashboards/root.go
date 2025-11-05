package dashboards

import (
	"github.com/spf13/cobra"
)

// NewDashboardsCmd creates a new cobra command for managing Datadog dashboards.
// It serves as a parent command for dashboard-related subcommands.
func NewDashboardsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dashboards",
		Short: "Manage Datadog dashboards",
	}

	cmd.AddCommand(NewDownloadCmd())

	return cmd
}
