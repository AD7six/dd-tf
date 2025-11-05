package dashboards

import (
	"fmt"
	"os"
	"sync"

	"github.com/AD7six/dd-tf/internal/datadog/dashboards"
	"github.com/spf13/cobra"
)

var (
	allFlag     bool
	updateFlag  bool
	outputPath  string
	team        string
	tags        string
	dashboardID string
)

func NewDownloadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download Datadog dashboards by ID, team, tags, or all",
		Run:   runDownload,
	}

	cmd.Flags().BoolVar(&allFlag, "all", false, "Download all dashboards")
	cmd.Flags().BoolVar(&updateFlag, "update", false, "Update already-downloaded dashboards (scans existing files)")
	cmd.Flags().StringVar(&outputPath, "output", "", "Output path template (supports {DASHBOARDS_DIR}, {id}, {title}, {team} and {any-tag}")
	cmd.Flags().StringVar(&team, "team", "", "Team name (convenience for tag 'team:x')")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated list of tags to filter dashboards")
	cmd.Flags().StringVar(&dashboardID, "id", "", "Dashboard ID(s) to download (comma-separated)")

	return cmd
}

func runDownload(cmd *cobra.Command, args []string) {
	opts := dashboards.DownloadOptions{
		All:         allFlag,
		Update:      updateFlag,
		OutputPath:  outputPath,
		Team:        team,
		Tags:        tags,
		DashboardID: dashboardID,
	}

	targetsCh, err := dashboards.GenerateDashboardTargets(opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 8)

	for target := range targetsCh {
		target := target // capture
		fmt.Printf("Downloading dashboard with ID: %s\n", target.ID)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := dashboards.DownloadDashboardWithOptions(target, outputPath); err != nil {
				errCh <- fmt.Errorf("%s: %w", target.ID, err)
			}
		}()
	}

	// wait and close error channel
	go func() { wg.Wait(); close(errCh) }()

	// collect errors
	var hadErr bool
	for e := range errCh {
		hadErr = true
		fmt.Fprintf(os.Stderr, "Error: %v\n", e)
	}
	if hadErr {
		os.Exit(1)
	}
}
