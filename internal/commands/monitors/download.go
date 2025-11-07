package monitors

import (
	"fmt"
	"os"
	"sync"

	"github.com/AD7six/dd-tf/internal/datadog/monitors"
	"github.com/AD7six/dd-tf/internal/datadog/resource"
	"github.com/spf13/cobra"
)

const (
	// errorChannelBuffer defines the buffer size for the error channel.
	// This matches the default HTTP client concurrency limit to prevent blocking.
	errorChannelBuffer = 8
)

// NewDownloadCmd creates a new cobra command for downloading Datadog monitors.
// It supports downloading monitors by ID (--id), team (--team), tags (--tags),
// priority (--priority), all monitors (--all), or updating existing monitors (--update).
func NewDownloadCmd() *cobra.Command {
	var (
		allFlag    bool
		updateFlag bool
		outputPath string
		team       string
		tags       string
		monitorID  string
		priority   int
	)

	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download Datadog monitors by ID, team, tags, priority, or all",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDownload(allFlag, updateFlag, outputPath, team, tags, monitorID, priority)
		},
	}

	cmd.Flags().BoolVar(&allFlag, "all", false, "Download all monitors")
	cmd.Flags().BoolVar(&updateFlag, "update", false, "Update already-downloaded monitors (scans existing files)")
	cmd.Flags().StringVar(&outputPath, "output", "", "Output path template (supports {id}, {name}, {team}, {priority}, {any-tag} and {ANY_ENV_VAR})")
	cmd.Flags().StringVar(&team, "team", "", "Team name (convenience for tag 'team:x')")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated list of tags to filter monitors")
	cmd.Flags().StringVar(&monitorID, "id", "", "Monitor ID(s) to download (comma-separated)")
	cmd.Flags().IntVar(&priority, "priority", 0, "Filter by monitor priority (integer)")

	return cmd
}

func runDownload(allFlag, updateFlag bool, outputPath, team, tags, monitorID string, priority int) error {
	opts := monitors.DownloadOptions{
		BaseDownloadOptions: resource.BaseDownloadOptions{
			All:        allFlag,
			Update:     updateFlag,
			OutputPath: outputPath,
			Team:       team,
			Tags:       tags,
			IDs:        monitorID,
		},
		Priority: priority,
	}

	targetsCh, err := monitors.GenerateMonitorTargets(opts)
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
		fmt.Printf("Downloading monitor with ID: %d\n", target.ID)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := monitors.DownloadMonitorWithOptions(target, outputPath); err != nil {
				errCh <- fmt.Errorf("%d: %w", target.ID, err)
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
		return fmt.Errorf("one or more monitors failed to download")
	}

	return nil
}
