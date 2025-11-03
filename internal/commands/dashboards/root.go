package dashboards

import "github.com/spf13/cobra"

func NewDashboardsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dashboards",
		Short: "Manage Datadog dashboards",
	}

	//cmd.AddCommand(DownloadAllCmd)
	//cmd.AddCommand(DownloadCmd)
	//cmd.AddCommand(UpdateCmd)

	return cmd
}
