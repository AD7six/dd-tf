package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Version is the semantic version, set by the build process
	Version = "dev"
	// Commit is the git commit hash, set by the build process
	Commit = "unknown"
	// BuildDate is when the binary was built, set by the build process
	BuildDate = "unknown"
)

// NewVersionCmd creates the version command
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("dd-tf version %s\n", Version)
			fmt.Printf("  commit: %s\n", Commit)
			fmt.Printf("  built:  %s\n", BuildDate)
		},
	}
}
