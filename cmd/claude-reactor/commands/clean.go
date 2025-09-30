package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"claude-reactor/internal/reactor"
	"claude-reactor/pkg"
)

// NewCleanCmd creates the clean command for removing containers and images
func NewCleanCmd(app *pkg.AppContainer) *cobra.Command {
	var cleanCmd = &cobra.Command{
		Use:   "clean",
		Short: "Remove containers, sessions, and authentication data",
		Long: `Remove claude-reactor containers and data with granular cleanup levels.

Cleanup Levels:
  clean                     Containers only (default)
  clean --sessions          Containers + session data (conversation history)  
  clean --auth              Containers + session data + credentials
  clean --all               Everything including global cache

Additional Options:
  --images                  Also remove Docker images
  --cache                   Clear image validation cache`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cleanContainers(cmd, app)
		},
	}

	// Cleanup level flags (mutually exclusive)
	cleanCmd.Flags().BoolP("sessions", "s", false, "Remove containers + session data")
	cleanCmd.Flags().BoolP("auth", "", false, "Remove containers + session data + credentials")
	cleanCmd.Flags().BoolP("all", "a", false, "Remove everything including global cache")
	
	// Additional cleanup flags  
	cleanCmd.Flags().BoolP("images", "i", false, "Remove Docker images as well")
	cleanCmd.Flags().BoolP("cache", "c", false, "Clear image validation cache")
	cleanCmd.Flags().BoolP("force", "f", false, "Force removal without confirmation")

	return cleanCmd
}

// cleanContainers handles container cleanup logic with granular cleanup levels
func cleanContainers(cmd *cobra.Command, app *pkg.AppContainer) error {
	ctx := cmd.Context()

	// Parse cleanup level flags
	sessions, _ := cmd.Flags().GetBool("sessions")
	auth, _ := cmd.Flags().GetBool("auth")
	all, _ := cmd.Flags().GetBool("all")
	images, _ := cmd.Flags().GetBool("images")
	cache, _ := cmd.Flags().GetBool("cache")
	force, _ := cmd.Flags().GetBool("force")

	// Determine cleanup level (highest level wins)
	cleanupLevel := "containers"
	if sessions {
		cleanupLevel = "sessions"
	}
	if auth {
		cleanupLevel = "auth"
	}
	if all {
		cleanupLevel = "all"
	}

	// Show what will be cleaned and ask for confirmation if not forced
	if !force {
		if err := showCleanupPlan(cleanupLevel, images, cache, app); err != nil {
			return err
		}
		
		fmt.Print("Do you want to continue? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" && response != "yes" {
			app.Logger.Info("🚫 Cleanup cancelled")
			return nil
		}
	}

	// Ensure Docker components are initialized for container cleanup
	if err := reactor.EnsureDockerComponents(app); err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}

	app.Logger.Infof("🧹 Starting cleanup (level: %s)...", cleanupLevel)

	// Step 1: Clean containers (all levels include this)
	if err := cleanContainersLevel(ctx, app, all); err != nil {
		return err
	}

	// Step 2: Clean session data (sessions, auth, all levels)
	if cleanupLevel == "sessions" || cleanupLevel == "auth" || cleanupLevel == "all" {
		if err := cleanSessionsLevel(app, all); err != nil {
			return err
		}
	}

	// Step 3: Clean authentication data (auth, all levels)
	if cleanupLevel == "auth" || cleanupLevel == "all" {
		if err := cleanAuthLevel(app, all); err != nil {
			return err
		}
	}

	// Step 4: Clean images if requested
	if images {
		if err := cleanImagesLevel(ctx, app, all); err != nil {
			return err
		}
	}

	// Step 5: Clean cache (cache flag or all level)
	if cache || cleanupLevel == "all" {
		if err := cleanCacheLevel(app); err != nil {
			return err
		}
	}

	app.Logger.Info("✅ Cleanup completed successfully!")
	return nil
}

// showCleanupPlan displays what will be cleaned and asks for confirmation
func showCleanupPlan(cleanupLevel string, images, cache bool, app *pkg.AppContainer) error {
	app.Logger.Info("🧹 Cleanup Plan:")
	
	switch cleanupLevel {
	case "containers":
		app.Logger.Info("  • 🐳 Remove claude-reactor containers")
	case "sessions":
		app.Logger.Info("  • 🐳 Remove claude-reactor containers")
		app.Logger.Info("  • 📁 Remove session data (conversation history, shell snapshots)")
	case "auth":
		app.Logger.Info("  • 🐳 Remove claude-reactor containers")
		app.Logger.Info("  • 📁 Remove session data (conversation history, shell snapshots)")
		app.Logger.Info("  • 🔑 Remove authentication data (Claude configs, API keys)")
	case "all":
		app.Logger.Info("  • 🐳 Remove claude-reactor containers")
		app.Logger.Info("  • 📁 Remove session data (conversation history, shell snapshots)")
		app.Logger.Info("  • 🔑 Remove authentication data (Claude configs, API keys)")
		app.Logger.Info("  • 🗄️ Clear global cache and validation data")
	}

	if images {
		app.Logger.Info("  • 📦 Remove Docker images")
	}
	if cache && cleanupLevel != "all" {
		app.Logger.Info("  • 🗄️ Clear validation cache")
	}

	app.Logger.Info("")
	return nil
}

// cleanContainersLevel removes containers
func cleanContainersLevel(ctx interface{}, app *pkg.AppContainer, all bool) error {
	if all {
		app.Logger.Info("🗑️ Removing all claude-reactor containers...")
		err := app.DockerMgr.CleanAllContainers(ctx)
		if err != nil {
			return fmt.Errorf("failed to clean all containers: %w", err)
		}
		app.Logger.Info("✅ All containers removed")
	} else {
		// Clean only current project container
		config, err := app.ConfigMgr.LoadConfig()
		if err != nil {
			app.Logger.Info("🗑️ No current project configuration found, skipping container cleanup")
			return nil
		}

		// Normalize account using new default logic
		if config.Account == "" {
			config.Account = getDefaultAccount()
		}

		arch, err := app.ArchDetector.GetHostArchitecture()
		if err != nil {
			return fmt.Errorf("failed to detect architecture: %w", err)
		}

		// Get current working directory for project hash
		projectDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		containerName := app.DockerMgr.GenerateContainerName(projectDir, config.Variant, arch, config.Account)
		app.Logger.Infof("🗑️ Removing container: %s", containerName)

		err = app.DockerMgr.CleanContainer(ctx, containerName)
		if err != nil {
			app.Logger.Warnf("Failed to clean container %s: %v", containerName, err)
			// Don't fail completely, container might not exist
		} else {
			app.Logger.Info("✅ Project container removed")
		}
	}
	return nil
}

// cleanSessionsLevel removes session data (conversation history, etc.)
func cleanSessionsLevel(app *pkg.AppContainer, all bool) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	claudeReactorDir := filepath.Join(homeDir, ".claude-reactor")

	if all {
		app.Logger.Info("🗂️ Removing all session data...")
		
		// Remove all account session directories but preserve auth files
		accounts, err := getAccountDirectories(claudeReactorDir)
		if err != nil {
			app.Logger.Warnf("Failed to get account directories: %v", err)
			return nil
		}

		for _, account := range accounts {
			accountDir := filepath.Join(claudeReactorDir, account)
			app.Logger.Infof("🗂️ Cleaning sessions for account: %s", account)
			
			if err := os.RemoveAll(accountDir); err != nil {
				app.Logger.Warnf("Failed to remove session directory %s: %v", accountDir, err)
			}
		}
		app.Logger.Info("✅ All session data removed")
	} else {
		// Remove only current project session
		config, err := app.ConfigMgr.LoadConfig()
		if err != nil {
			app.Logger.Info("🗂️ No current project configuration found, skipping session cleanup")
			return nil
		}

		// Normalize account
		if config.Account == "" {
			config.Account = getDefaultAccount()
		}

		projectDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Calculate project session directory
		sessionDir := app.AuthMgr.GetProjectSessionDir(config.Account, projectDir)
		
		app.Logger.Infof("🗂️ Removing session data: %s", sessionDir)
		if err := os.RemoveAll(sessionDir); err != nil {
			app.Logger.Warnf("Failed to remove session directory: %v", err)
		} else {
			app.Logger.Info("✅ Project session data removed")
		}
	}
	return nil
}

// cleanAuthLevel removes authentication data (Claude configs, API keys)
func cleanAuthLevel(app *pkg.AppContainer, all bool) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	claudeReactorDir := filepath.Join(homeDir, ".claude-reactor")

	if all {
		app.Logger.Info("🔑 Removing all authentication data...")
		
		// Remove all .{account}-claude.json and .claude-reactor-{account}-env files
		entries, err := os.ReadDir(claudeReactorDir)
		if err != nil {
			app.Logger.Warnf("Failed to read claude-reactor directory: %v", err)
			return nil
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			name := entry.Name()
			if (name != ".claude.json" && 
				(name[:1] == "." && (name[len(name)-12:] == "-claude.json" || name[:15] == ".claude-reactor-"))) {
				filePath := filepath.Join(claudeReactorDir, name)
				app.Logger.Infof("🔑 Removing auth file: %s", name)
				if err := os.Remove(filePath); err != nil {
					app.Logger.Warnf("Failed to remove auth file %s: %v", filePath, err)
				}
			}
		}
		app.Logger.Info("✅ All authentication data removed")
	} else {
		// Remove only current account auth
		config, err := app.ConfigMgr.LoadConfig()
		if err != nil {
			app.Logger.Info("🔑 No current project configuration found, skipping auth cleanup")
			return nil
		}

		// Normalize account
		if config.Account == "" {
			config.Account = getDefaultAccount()
		}

		// Remove account-specific auth files
		authConfigPath := app.AuthMgr.GetAccountConfigPath(config.Account)
		apiKeyPath := app.AuthMgr.GetAPIKeyFile(config.Account)

		app.Logger.Infof("🔑 Removing auth for account: %s", config.Account)
		
		if err := os.Remove(authConfigPath); err != nil && !os.IsNotExist(err) {
			app.Logger.Warnf("Failed to remove auth config: %v", err)
		}
		
		if err := os.Remove(apiKeyPath); err != nil && !os.IsNotExist(err) {
			app.Logger.Warnf("Failed to remove API key file: %v", err)
		}
		
		app.Logger.Info("✅ Account authentication data removed")
	}
	return nil
}

// cleanImagesLevel removes Docker images
func cleanImagesLevel(ctx interface{}, app *pkg.AppContainer, all bool) error {
	app.Logger.Info("📦 Removing Docker images...")
	
	err := app.DockerMgr.CleanImages(ctx, all)
	if err != nil {
		return fmt.Errorf("failed to clean images: %w", err)
	}
	
	app.Logger.Info("✅ Docker images removed")
	return nil
}

// cleanCacheLevel clears validation cache
func cleanCacheLevel(app *pkg.AppContainer) error {
	app.Logger.Info("🗄️ Clearing validation cache...")
	
	if app.ImageValidator != nil {
		err := app.ImageValidator.ClearCache()
		if err != nil {
			return fmt.Errorf("failed to clear image validation cache: %w", err)
		}
		app.ImageValidator.ClearSessionWarnings()
	}
	
	app.Logger.Info("✅ Validation cache cleared")
	return nil
}

// getAccountDirectories returns list of account directories in claude-reactor dir
func getAccountDirectories(claudeReactorDir string) ([]string, error) {
	entries, err := os.ReadDir(claudeReactorDir)
	if err != nil {
		return nil, err
	}

	var accounts []string
	for _, entry := range entries {
		if entry.IsDir() {
			accounts = append(accounts, entry.Name())
		}
	}
	return accounts, nil
}

// getDefaultAccount returns $USER or "user" fallback per requirements
func getDefaultAccount() string {
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	return "user"
}