package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"claude-reactor/pkg"
)

// NewCleanCmd creates the clean command for removing containers and images
func NewCleanCmd(app *pkg.AppContainer) *cobra.Command {
	var cleanCmd = &cobra.Command{
		Use:   "clean",
		Short: "Remove containers and images",
		Long: `Remove claude-reactor containers and images.
Clean up development containers, cached images, and temporary resources.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cleanContainers(cmd, app)
		},
	}

	cleanCmd.Flags().BoolP("all", "a", false, "Remove all containers and images")
	cleanCmd.Flags().BoolP("images", "i", false, "Remove images as well")
	cleanCmd.Flags().BoolP("cache", "c", false, "Clear image validation cache")
	cleanCmd.Flags().BoolP("force", "f", false, "Force removal without confirmation")

	return cleanCmd
}

// cleanContainers handles container cleanup logic
func cleanContainers(cmd *cobra.Command, app *pkg.AppContainer) error {
	ctx := cmd.Context()
	
	all, _ := cmd.Flags().GetBool("all")
	images, _ := cmd.Flags().GetBool("images")
	cache, _ := cmd.Flags().GetBool("cache")
	
	app.Logger.Info("🧹 Cleaning up containers...")
	
	if all {
		// Clean all claude-reactor containers across all accounts
		app.Logger.Info("🗑️ Removing all claude-reactor containers across all accounts...")
		app.Logger.Info("⏳ This will stop and remove all running claude-reactor containers...")
		err := app.DockerMgr.CleanAllContainers(ctx)
		if err != nil {
			return fmt.Errorf("failed to clean all containers: %w. Try running 'docker container prune' manually", err)
		}
		app.Logger.Info("✅ All claude-reactor containers removed successfully")
	} else {
		// Clean only current project/account container
		app.Logger.Info("🔍 Loading current configuration to identify container...")
		config, err := app.ConfigMgr.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w. Try running 'claude-reactor config validate' to check setup", err)
		}
		
		// Auto-detect variant if needed
		if config.Variant == "" {
			app.Logger.Info("🔍 Auto-detecting project type...")
			detectedVariant, err := app.ConfigMgr.AutoDetectVariant("")
			if err != nil {
				app.Logger.Warnf("Failed to auto-detect variant: %v", err)
				app.Logger.Info("💡 Defaulting to 'base' variant")
				config.Variant = "base"
			} else {
				config.Variant = detectedVariant
			}
		}
		
		arch, err := app.ArchDetector.GetHostArchitecture()
		if err != nil {
			return fmt.Errorf("failed to detect architecture: %w. System may not be supported", err)
		}
		
		containerName := app.DockerMgr.GenerateContainerName("", config.Variant, arch, config.Account)
		app.Logger.Infof("🗑️ Removing container for current project: %s", containerName)
		
		err = app.DockerMgr.CleanContainer(ctx, containerName)
		if err != nil {
			return fmt.Errorf("failed to clean container %s: %w. Try 'docker ps' to check container status", containerName, err)
		}
		app.Logger.Info("✅ Project container removed successfully")
		
		// Clear container ID from config when cleaning current project container
		config.ContainerID = ""
		config.LastSessionID = ""
		if err := app.ConfigMgr.SaveConfig(config); err != nil {
			app.Logger.Warnf("Failed to clear session data from config: %v", err)
		}
	}
	
	// Clean images if requested
	if images {
		if all {
			app.Logger.Info("🗑️ Cleaning all claude-reactor images...")
			app.Logger.Info("⏳ This will remove all cached claude-reactor images...")
		} else {
			app.Logger.Info("🗑️ Cleaning current project claude-reactor images...")
		}
		
		err := app.DockerMgr.CleanImages(ctx, all)
		if err != nil {
			return fmt.Errorf("failed to clean images: %w. Try running 'docker image prune' manually", err)
		}
		
		app.Logger.Info("✅ Image cleanup completed successfully")
		app.Logger.Info("💡 Images will be rebuilt automatically on next run")
	}
	
	if cache {
		app.Logger.Info("🗄️ Clearing image validation cache...")
		
		err := app.ImageValidator.ClearCache()
		if err != nil {
			return fmt.Errorf("failed to clear image validation cache: %w. You can manually delete ~/.claude-reactor/image-cache/", err)
		}
		
		// Also clear session warnings so warnings will show again
		app.ImageValidator.ClearSessionWarnings()
		
		app.Logger.Info("✅ Image validation cache cleared successfully")
		app.Logger.Info("💡 Custom images will be re-validated and warnings reshown on next use")
	}
	
	return nil
}