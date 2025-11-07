package dashboards

import (
	"fmt"
	"os"
	"sync"

	"github.com/AD7six/dd-tf/internal/datadog/dashboards"
	"github.com/AD7six/dd-tf/internal/datadog/resource"
	"github.com/spf13/cobra"
)

const (
	// errorChannelBuffer defines the buffer size for the error channel.
	// This matches the default HTTP client concurrency limit to prevent blocking.
	errorChannelBuffer = 8
)

// NewDownloadCmd creates a new cobra command for downloading Datadog dashboards.
// It supports downloading dashboards by ID (--id), team (--team), tags (--tags),
// all dashboards (--all), or updating existing dashboards (--update).
func NewDownloadCmd() *cobra.Command {
	var (
		allFlag     bool
		updateFlag  bool
		outputPath  string
		team        string
		tags        string
		dashboardID string
	)

	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download Datadog dashboards by ID, team, tags, or all",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDownload(allFlag, updateFlag, outputPath, team, tags, dashboardID)
		},
	}

	cmd.Flags().BoolVar(&allFlag, "all", false, "Download all dashboards")
	cmd.Flags().BoolVar(&updateFlag, "update", false, "Update already-downloaded dashboards (scans existing files)")
	cmd.Flags().StringVar(&outputPath, "output", "", "Output path template (supports data, {id}, {title}, {team} and {any-tag}")
	cmd.Flags().StringVar(&team, "team", "", "Team name (convenience for tag 'team:x')")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated list of tags to filter dashboards")
	cmd.Flags().StringVar(&dashboardID, "id", "", "Dashboard ID(s) to download (comma-separated)")

	return cmd
}

func runDownload(allFlag, updateFlag bool, outputPath, team, tags, dashboardID string) error {
	opts := dashboards.DownloadOptions{
		BaseDownloadOptions: resource.BaseDownloadOptions{
			All:        allFlag,
			Update:     updateFlag,
			OutputPath: outputPath,
			Team:       team,
			Tags:       tags,
			IDs:        dashboardID,
		},
	}

	targetsCh, err := dashboards.GenerateDashboardTargets(opts)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	errCh := make(chan error, errorChannelBuffer)

	for result := range targetsCh {
		// Check if target generation failed
		if result.Err != nil {
			errCh <- result.Err
			continue
		}

		target := result.Target // capture
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
		return fmt.Errorf("one or more dashboards failed to download")
	}

	return nil
}
