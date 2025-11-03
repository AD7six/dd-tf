package dashboards

import (
	id "github.com/AD7six/dd-tf/internal/datadog"
	"github.com/spf13/cobra"
)

func NewDashboardsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dashboards",
		Short: "Manage Datadog dashboards",
	}

	//cmd.AddCommand(DownloadAllCmd)
	cmd.AddCommand(id.DownloadCmd)
	//cmd.AddCommand(UpdateCmd)

	return cmd
}
