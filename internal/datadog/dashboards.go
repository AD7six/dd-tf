package dashboards

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	allFlag     bool
	team        string
	tags        string
	dashboardID string
)

var DownloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download Datadog dashboards by ID, team, tags, or all",
	Run: func(cmd *cobra.Command, args []string) {
		switch {
		case allFlag:
			fmt.Println("Downloading all dashboards...")
			// TODO: Implement logic to download all dashboards
		case team != "":
			tagFilter := fmt.Sprintf("team:%s", team)
			fmt.Printf("Downloading dashboards with team tag: %s\n", tagFilter)
			// TODO: Implement logic to download dashboards by team tag
		case tags != "":
			tagList := strings.Split(tags, ",")
			fmt.Printf("Downloading dashboards with tags: %v\n", tagList)
			// TODO: Implement logic to download dashboards by tags
		case dashboardID != "":
			fmt.Printf("Downloading dashboard with ID: %s\n", dashboardID)
			// TODO: Implement logic to download dashboard by ID
		default:
			fmt.Println("Please specify --all, --team, --tags, or --id")
		}
	},
}

func init() {
	DownloadCmd.Flags().BoolVar(&allFlag, "all", false, "Download all dashboards")
	DownloadCmd.Flags().StringVar(&team, "team", "", "Team name (convenience for tag 'team:x')")
	DownloadCmd.Flags().StringVar(&tags, "tags", "", "Comma-separated list of tags to filter dashboards")
	DownloadCmd.Flags().StringVar(&dashboardID, "id", "", "Dashboard ID to download")
}
