package commands

import (
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"claude-reactor/internal/reactor"
	"claude-reactor/pkg"
)

// Version information - will be populated from main.go when creating the command
var (
	debugVersion   = "dev"
	debugGitCommit = "unknown"
	debugBuildDate = "unknown"
)

// SetVersionInfo sets version information for the debug command
func SetVersionInfo(version, gitCommit, buildDate string) {
	debugVersion = version
	debugGitCommit = gitCommit
	debugBuildDate = buildDate
}

// NewInfoCmd creates the info command with troubleshooting tools
func NewInfoCmd(app *pkg.AppContainer) *cobra.Command {
	var infoCmd = &cobra.Command{
		Use:   "info",
		Short: "System information and troubleshooting",
		Long:  "Provide system information and troubleshooting tools for claude-reactor.",
		Example: `# Show system information
claude-reactor info

# Test custom image compatibility
claude-reactor info image ubuntu:22.04

# Clear validation cache
claude-reactor info cache clear

# Show cache statistics
claude-reactor info cache info`,
	}

	infoCmd.AddCommand(
		&cobra.Command{
			Use:   "status",
			Short: "Show debug status",
			Long:  "Display current debug mode and logging configuration.",
			RunE: func(cmd *cobra.Command, args []string) error {
				// Handle help case when app is nil
				if app == nil {
					return cmd.Help()
				}
				cmd.Printf("Debug Mode: %v\n", app.Debug)

				// Try to get log level through interface or fallback
				if logger, ok := app.Logger.(interface{ GetLevel() logrus.Level }); ok {
					cmd.Printf("Log Level: %s\n", logger.GetLevel().String())
				} else {
					cmd.Printf("Log Level: unable to determine\n")
				}
				return nil
			},
		},
		&cobra.Command{
			Use:   "info",
			Short: "Show system information",
			Long:  "Display comprehensive system information including Docker connectivity, architecture, and version details.",
			RunE: func(cmd *cobra.Command, args []string) error {
				// Handle help case when app is nil
				if app == nil {
					return cmd.Help()
				}
				app.Logger.Info("=== Claude-Reactor Debug Info ===")

				// Architecture information
				arch, err := app.ArchDetector.GetHostArchitecture()
				if err != nil {
					app.Logger.Errorf("Failed to detect architecture: %v", err)
				} else {
					cmd.Printf("Host Architecture: %s\n", arch)
				}

				platform, err := app.ArchDetector.GetDockerPlatform()
				if err != nil {
					app.Logger.Errorf("Failed to get Docker platform: %v", err)
				} else {
					cmd.Printf("Docker Platform: %s\n", platform)
				}

				cmd.Printf("Multi-arch Support: %t\n", app.ArchDetector.IsMultiArchSupported())

				// Version information
				cmd.Printf("Version: %s\n", debugVersion)
				cmd.Printf("Git Commit: %s\n", debugGitCommit)
				cmd.Printf("Build Date: %s\n", debugBuildDate)

				// Docker connectivity test - try to initialize Docker
				ctx := cmd.Context()
				if err := reactor.EnsureDockerComponents(app); err != nil {
					cmd.Printf("Docker Connection: ❌ Failed (%v)\n", err)
				} else {
					_, dockerErr := app.DockerMgr.IsContainerRunning(ctx, "test-connection")
					if dockerErr != nil {
						cmd.Printf("Docker Connection: ❌ Failed (%v)\n", dockerErr)
					} else {
						cmd.Printf("Docker Connection: ✅ Connected\n")
					}
				}
				
				return nil
			},
		},
		&cobra.Command{
			Use:   "image [image-name]",
			Short: "Test custom image compatibility",
			Long: `Test a custom Docker image for claude-reactor compatibility.
This will validate platform support, Claude CLI availability, and recommended packages.
Results are the same as what you'd see during normal container startup.`,
			Example: `# Test Ubuntu image
claude-reactor debug image ubuntu:22.04

# Test with detailed output
claude-reactor debug image python:3.11 --verbose

# Test registry image
claude-reactor debug image ghcr.io/user/project:latest`,
			Args: cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				// Handle help case when app is nil
				if app == nil {
					return cmd.Help()
				}
				imageName := args[0]
				ctx := cmd.Context()

				// Ensure Docker components are initialized
				if err := reactor.EnsureDockerComponents(app); err != nil {
					cmd.Printf("❌ Docker not available: %v\n", err)
					return err
				}

				app.Logger.Infof("🔍 Testing image compatibility: %s", imageName)

				// Test image validation
				result, err := app.ImageValidator.ValidateImage(ctx, imageName, true)
				if err != nil {
					cmd.Printf("❌ Validation failed: %v\n", err)
					return err
				}
				
				cmd.Printf("\n=== Image Validation Results ===\n")
				cmd.Printf("Image: %s\n", imageName)
				cmd.Printf("Digest: %s\n", result.Digest)
				cmd.Printf("Architecture: %s\n", result.Architecture)
				cmd.Printf("Platform: %s\n", result.Platform)
				cmd.Printf("Size: %.2f MB\n", float64(result.Size)/(1024*1024))
				cmd.Printf("Compatible: %t\n", result.Compatible)
				cmd.Printf("Has Claude CLI: %t\n", result.HasClaude)
				cmd.Printf("Is Linux: %t\n", result.IsLinux)
				
				if len(result.Warnings) > 0 {
					cmd.Printf("\n⚠️ Warnings:\n")
					for _, warning := range result.Warnings {
						cmd.Printf("  - %s\n", warning)
					}
				}
				
				if len(result.Errors) > 0 {
					cmd.Printf("\n❌ Errors:\n")
					for _, errMsg := range result.Errors {
						cmd.Printf("  - %s\n", errMsg)
					}
				}
				
				// Show package analysis if available
				if packages, ok := result.Metadata["packages"].(map[string]interface{}); ok {
					cmd.Printf("\n📦 Package Analysis:\n")
					if available, ok := packages["available"].([]string); ok {
						cmd.Printf("Available tools (%d): %s\n", len(available), strings.Join(available, ", "))
					}
					if missing, ok := packages["missing_high_priority"].([]string); ok && len(missing) > 0 {
						cmd.Printf("Missing high-priority tools: %s\n", strings.Join(missing, ", "))
					}
					if totalChecked, ok := packages["total_checked"].(int); ok {
						if totalAvailable, ok := packages["total_available"].(int); ok {
							cmd.Printf("Coverage: %d/%d recommended tools available\n", totalAvailable, totalChecked)
						}
					}
				}
				
				if result.Compatible {
					cmd.Printf("\n✅ Image is compatible with claude-reactor!\n")
				} else {
					cmd.Printf("\n❌ Image is not compatible. See errors above.\n")
				}
				
				return nil
			},
		},
	)
	
	// Create cache subcommand with subcommands
	cacheCmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage image validation cache",
		Long:  "View cache statistics and clear cached validation results.",
	}
	
	cacheCmd.AddCommand(
		&cobra.Command{
			Use:   "info",
			Short: "Show cache statistics",
			Long:  "Display information about cached image validation results.",
			RunE: func(cmd *cobra.Command, args []string) error {
				// Handle help case when app is nil
				if app == nil {
					return cmd.Help()
				}
				// This would require implementing cache info functionality
				// For now, just show a placeholder
				cmd.Printf("Cache directory: ~/.claude-reactor/image-cache/\n")
				cmd.Printf("Cache duration: 30+ days (based on image digest)\n")
				cmd.Printf("Use 'claude-reactor debug cache clear' to clear cache\n")
				return nil
			},
		},
		&cobra.Command{
			Use:   "clear",
			Short: "Clear validation cache",
			Long:  "Remove all cached image validation results, forcing re-validation on next use.",
			RunE: func(cmd *cobra.Command, args []string) error {
				// Handle help case when app is nil
				if app == nil {
					return cmd.Help()
				}
				// Ensure Docker components are initialized
				if err := reactor.EnsureDockerComponents(app); err != nil {
					cmd.Printf("❌ Docker not available: %v\n", err)
					return err
				}

				err := app.ImageValidator.ClearCache()
				if err != nil {
					cmd.Printf("❌ Failed to clear cache: %v\n", err)
					return err
				}
				cmd.Printf("✅ Image validation cache cleared successfully\n")
				return nil
			},
		},
	)
	
	infoCmd.AddCommand(cacheCmd)
	
	return infoCmd
}