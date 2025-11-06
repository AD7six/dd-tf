package monitors

import (
	"github.com/spf13/cobra"
)

func NewMonitorsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "monitors",
		Short: "Manage Datadog monitors",
	}
	cmd.AddCommand(NewDownloadCmd())
	return cmd
}
