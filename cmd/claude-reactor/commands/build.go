package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"claude-reactor/pkg"
)

// NewBuildCmd creates the build command for building container images
func NewBuildCmd(app *pkg.AppContainer) *cobra.Command {
	var buildCmd = &cobra.Command{
		Use:   "build [variant]",
		Short: "Build container images",
		Long: `Build Docker container images for development.
Supports building specific variants or all variants with intelligent caching.

Available variants: base, go, full, cloud, k8s

Examples:
  claude-reactor build              # Build all core variants
  claude-reactor build go           # Build only go variant
  claude-reactor build --rebuild    # Force rebuild without cache`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return buildImages(cmd, args, app)
		},
	}

	buildCmd.Flags().BoolP("rebuild", "r", false, "Force rebuild without cache")
	buildCmd.Flags().BoolP("all", "a", false, "Build all variants including cloud and k8s")
	buildCmd.Flags().StringP("platform", "p", "", "Target platform (linux/amd64, linux/arm64)")

	return buildCmd
}

// buildImages handles the image building logic
func buildImages(cmd *cobra.Command, args []string, app *pkg.AppContainer) error {
	// Implementation will be moved from main.go
	return fmt.Errorf("buildImages implementation needs to be moved from main.go")
}