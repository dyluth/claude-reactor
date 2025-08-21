package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"claude-reactor/internal/reactor"
	"claude-reactor/pkg"
)

var (
	// Version information - will be set during build
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

func main() {
	if err := Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// Execute runs the root command
func Execute() error {
	ctx := context.Background()
	
	// Initialize application container
	app, err := reactor.NewAppContainer()
	if err != nil {
		return fmt.Errorf("failed to initialize application: %w", err)
	}

	rootCmd := newRootCmd(app)
	return rootCmd.ExecuteContext(ctx)
}

func newRootCmd(app *pkg.AppContainer) *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:   "claude-reactor",
		Short: "Claude CLI in Docker - Modern containerization for Claude development",
		Long: `Claude-Reactor provides a professional, modular Docker containerization system 
for Claude CLI development workflows. It transforms the basic Claude CLI into a 
comprehensive development environment with intelligent automation, multi-language 
support, and production-ready tooling.`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", Version, GitCommit, BuildDate),
		Run: func(cmd *cobra.Command, args []string) {
			// Handle deprecated flags with clear migration guidance
			
			if listVariants, _ := cmd.Flags().GetBool("list-variants"); listVariants {
				fmt.Fprintf(os.Stderr, "❌ The --list-variants flag has been removed. Use:\n")
				fmt.Fprintf(os.Stderr, "   claude-reactor debug info\n")
				os.Exit(1)
			}
			
			if variant, _ := cmd.Flags().GetString("variant"); variant != "" {
				fmt.Fprintf(os.Stderr, "❌ The --variant flag has been removed. Use:\n")
				fmt.Fprintf(os.Stderr, "   claude-reactor run --image %s\n", variant)
				os.Exit(1)
			}
			
			if showConfig, _ := cmd.Flags().GetBool("show-config"); showConfig {
				fmt.Fprintf(os.Stderr, "❌ The --show-config flag has been removed. Use:\n")
				fmt.Fprintf(os.Stderr, "   claude-reactor config show\n")
				os.Exit(1)
			}
			
			// Default action - if config exists, run; otherwise show help
			config, err := app.ConfigMgr.LoadConfig()
			if err == nil && (config.Variant != "" || config.Account != "") {
				// Configuration exists, default to run command
				app.Logger.Info("🚀 Found existing configuration, running container...")
				runCmd := newRunCmd(app)
				if runErr := runCmd.RunE(cmd, args); runErr != nil {
					cmd.PrintErrf("Run failed: %v\n", runErr)
					os.Exit(1)
				}
				return
			}
			
			// No configuration found, show help
			cmd.Help()
		},
	}

	// Global flags
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().String("log-level", "info", "Set log level (debug, info, warn, error)")
	
	// Deprecated flags (hidden, show clear migration error)
	rootCmd.Flags().Bool("list-variants", false, "Removed: use 'debug info'")
	rootCmd.Flags().Bool("show-config", false, "Removed: use 'config show'")
	rootCmd.Flags().String("variant", "", "Removed: use 'run --image'")
	rootCmd.Flags().MarkHidden("list-variants")
	rootCmd.Flags().MarkHidden("show-config")
	rootCmd.Flags().MarkHidden("variant")

	// Add subcommands
	rootCmd.AddCommand(
		newRunCmd(app),
		newBuildCmd(app),
		newConfigCmd(app),
		newCleanCmd(app),
		newDevContainerCmd(app),
		newTemplateCmd(app),
		newDependencyCmd(app),
		newHotReloadCmd(app),
		newDebugCmd(app),
		newCompletionCmd(app),
	)

	return rootCmd
}

func newRunCmd(app *pkg.AppContainer) *cobra.Command {
	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "Start and connect to Claude CLI container",
		Long: `Start and connect to a Claude CLI container with intelligent project detection.
Auto-detects project type, builds container if needed, and launches Claude CLI 
with appropriate development tools and configurations.

Built-in Images: base, go, full, cloud, k8s (optimized for development)
Custom Images: Any Docker Hub or registry image (validated for compatibility)

Examples:
  claude-reactor run                           # Auto-detect image and run
  claude-reactor run --image go                # Use Go toolchain image
  claude-reactor run --image ubuntu:22.04     # Use custom Ubuntu image
  claude-reactor run --image ghcr.io/org/dev  # Use custom registry image
  claude-reactor run --shell                  # Launch interactive shell instead
  claude-reactor run --danger                 # Enable danger mode (skip permissions)
  claude-reactor run --account work           # Use specific account configuration
  claude-reactor run --account work --apikey sk-ant-xxx  # Set API key for work account
  claude-reactor run --account new --interactive-login   # Force interactive login for new account
  claude-reactor run --no-persist             # Remove container when finished
  
  # Registry control (v2 images)
  claude-reactor run --dev                    # Force local build (disable registry)
  claude-reactor run --registry-off           # Disable registry completely
  claude-reactor run --pull-latest            # Force pull latest from registry
  claude-reactor run --no-continue            # Disable conversation continuation

Custom Image Requirements:
  • Must be Linux-based (linux/amd64 or linux/arm64)
  • Must have Claude CLI installed: 'claude --version' should work
  • Recommended tools: git, curl, make, nano (warnings shown if missing)

Troubleshooting:
  Use 'claude-reactor debug info' to check Docker connectivity
  Use 'claude-reactor debug image <name>' to test custom images
  Use '--verbose' flag for detailed validation information
  
Related Commands:
  claude-reactor build         Build container images
  claude-reactor clean         Remove containers
  claude-reactor config show   View current configuration`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runContainer(cmd, app)
		},
	}

	// Run command flags
	runCmd.Flags().StringP("image", "", "", "Container image (base, go, full, cloud, k8s, or custom Docker image)")
	runCmd.Flags().StringP("account", "", "", "Claude account to use")
	runCmd.Flags().StringP("apikey", "", "", "Set API key for this session (creates account-specific env file)")
	runCmd.Flags().BoolP("interactive-login", "", false, "Force interactive authentication for account")
	runCmd.Flags().BoolP("danger", "", false, "Enable danger mode (--dangerously-skip-permissions)")
	runCmd.Flags().BoolP("shell", "", false, "Launch shell instead of Claude CLI")
	runCmd.Flags().StringSliceP("mount", "m", []string{}, "Additional mount points (can be used multiple times)")
	runCmd.Flags().BoolP("no-persist", "", false, "Remove container when finished (default: keep running)")
	runCmd.Flags().BoolP("no-mounts", "", false, "Skip mounting directories (for testing)")
	
	// Registry flags (Phase 0.1)
	runCmd.Flags().BoolP("dev", "", false, "Force local build (disable registry pulls)")
	runCmd.Flags().BoolP("registry-off", "", false, "Disable registry completely")
	runCmd.Flags().BoolP("pull-latest", "", false, "Force pull latest from registry")
	runCmd.Flags().BoolP("no-continue", "", false, "Disable conversation continuation (default: enabled)")

	return runCmd
}

func newBuildCmd(app *pkg.AppContainer) *cobra.Command {
	var buildCmd = &cobra.Command{
		Use:   "build [image]",
		Short: "Build container images",
		Long: `Build Docker container images for the specified image variant.
Uses multi-stage Dockerfile with architecture-aware optimizations for your platform.

Use the Makefile for building multiple images efficiently:
  make build-all      # Build core images (base, go, full)
  make build-extended # Build all images

Examples:
  claude-reactor build               # Build base image
  claude-reactor build go            # Build Go toolchain image
  claude-reactor build --rebuild     # Force rebuild of base image
  claude-reactor build full --rebuild # Force rebuild of full image`,
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Logger.Info("🔨 Building container images...")
			
			image := "base"
			if len(args) > 0 {
				image = args[0]
			}
			
			rebuild, _ := cmd.Flags().GetBool("rebuild")
			app.Logger.Infof("📋 Building image: %s, force rebuild: %t", image, rebuild)
			
			if rebuild {
				app.Logger.Info("⚡ Force rebuild enabled - removing existing images first")
			}
			
			// Get Docker platform from architecture detector
			app.Logger.Info("🔧 Detecting system architecture...")
			platform, err := app.ArchDetector.GetDockerPlatform()
			if err != nil {
				return fmt.Errorf("failed to determine Docker platform: %w. Architecture may not be supported", err)
			}
			app.Logger.Infof("🐳 Building for platform: %s", platform)
			
			// Use Docker manager to build image
			ctx := cmd.Context()
			app.Logger.Info("⏳ This may take several minutes for first-time build...")
			
			if rebuild {
				err = app.DockerMgr.RebuildImage(ctx, image, platform, true)
			} else {
				err = app.DockerMgr.BuildImage(ctx, image, platform)
			}
			
			if err != nil {
				return fmt.Errorf("failed to build image: %w. Try running 'docker system prune' to free space", err)
			}
			
			app.Logger.Info("✅ Image build completed successfully!")
			app.Logger.Info("💡 Use 'claude-reactor run' to start the container")
			app.Logger.Info("💡 Use 'claude-reactor clean --images' to remove old images")
			return nil
		},
	}

	buildCmd.Flags().BoolP("rebuild", "", false, "Force rebuild of existing images")

	return buildCmd
}

func newConfigCmd(app *pkg.AppContainer) *cobra.Command {
	var configCmd = &cobra.Command{
		Use:   "config",
		Short: "Configuration management",
		Long:  "Manage claude-reactor configuration settings.",
	}

	// Config show subcommand with enhanced display
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show current configuration with registry status",
		Long: `Display comprehensive configuration information including:
- Project configuration (variant, account, danger mode)
- Registry configuration and status
- System paths and architecture details
- Raw configuration file contents

Examples:
  claude-reactor config show           # Show basic configuration
  claude-reactor config show --raw     # Include raw config file contents
  claude-reactor config show --verbose # Show detailed system information`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return showEnhancedConfig(cmd, app)
		},
	}
	
	// Add flags to show subcommand
	showCmd.Flags().Bool("raw", false, "Include raw configuration file contents")
	showCmd.Flags().Bool("verbose", false, "Show detailed system information")

	// Add config subcommands
	configCmd.AddCommand(
		showCmd,
		&cobra.Command{
			Use:   "validate",
			Short: "Validate configuration",
			RunE: func(cmd *cobra.Command, args []string) error {
				config, err := app.ConfigMgr.LoadConfig()
				if err != nil {
					return fmt.Errorf("failed to load configuration: %w", err)
				}
				
				if err := app.ConfigMgr.ValidateConfig(config); err != nil {
					return fmt.Errorf("configuration validation failed: %w", err)
				}
				
				cmd.Println("Configuration is valid ✓")
				return nil
			},
		},
	)

	return configCmd
}

func newCleanCmd(app *pkg.AppContainer) *cobra.Command {
	var cleanCmd = &cobra.Command{
		Use:   "clean",
		Short: "Clean up containers and images",
		Long: `Remove stopped containers and optionally clean up images and cache.
Use --all to remove all claude-reactor containers across all accounts.

Examples:
  claude-reactor clean                # Remove current project container
  claude-reactor clean --all          # Remove all claude-reactor containers
  claude-reactor clean --images       # Also remove project images
  claude-reactor clean --cache        # Also clear image validation cache
  claude-reactor clean --all --images --cache # Remove everything (containers + images + cache)

Related Commands:
  claude-reactor run           Start new container
  claude-reactor build         Build container images
  claude-reactor config show   View current configuration`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cleanContainers(cmd, app)
		},
	}

	cleanCmd.Flags().BoolP("all", "", false, "Clean all claude-reactor containers")
	cleanCmd.Flags().BoolP("images", "", false, "Also remove images")
	cleanCmd.Flags().BoolP("cache", "", false, "Also clear image validation cache")

	return cleanCmd
}

func newDebugCmd(app *pkg.AppContainer) *cobra.Command {
	var debugCmd = &cobra.Command{
		Use:   "debug",
		Short: "Debug information and troubleshooting",
		Long:  "Provide debug information and troubleshooting tools for claude-reactor.",
		Example: `# Show system information
claude-reactor debug info

# Test custom image compatibility
claude-reactor debug image ubuntu:22.04

# Clear validation cache
claude-reactor debug cache clear

# Show cache statistics
claude-reactor debug cache info`,
	}

	debugCmd.AddCommand(
		&cobra.Command{
			Use:   "info",
			Short: "Show system information",
			Long:  "Display comprehensive system information including Docker connectivity, architecture, and version details.",
			RunE: func(cmd *cobra.Command, args []string) error {
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
				cmd.Printf("Version: %s\n", Version)
				cmd.Printf("Git Commit: %s\n", GitCommit)
				cmd.Printf("Build Date: %s\n", BuildDate)
				
				// Docker connectivity test
				ctx := cmd.Context()
				_, dockerErr := app.DockerMgr.IsContainerRunning(ctx, "test-connection")
				if dockerErr != nil {
					cmd.Printf("Docker Connection: ❌ Failed (%v)\n", dockerErr)
				} else {
					cmd.Printf("Docker Connection: ✅ Connected\n")
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
				imageName := args[0]
				ctx := cmd.Context()
				
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
	
	debugCmd.AddCommand(cacheCmd)
	
	return debugCmd
}


// runContainer implements the core run command logic
func runContainer(cmd *cobra.Command, app *pkg.AppContainer) error {
	ctx := cmd.Context()
	
	// Parse command flags
	image, _ := cmd.Flags().GetString("image")
	account, _ := cmd.Flags().GetString("account")
	apikey, _ := cmd.Flags().GetString("apikey")
	interactiveLogin, _ := cmd.Flags().GetBool("interactive-login")
	danger, _ := cmd.Flags().GetBool("danger")
	shell, _ := cmd.Flags().GetBool("shell")
	mounts, _ := cmd.Flags().GetStringSlice("mount")
	noPersist, _ := cmd.Flags().GetBool("no-persist")
	persist := !noPersist  // Default to true, unless --no-persist is specified
	noMounts, _ := cmd.Flags().GetBool("no-mounts")
	
	// Registry flags (Phase 0.1)
	devMode, _ := cmd.Flags().GetBool("dev")
	registryOff, _ := cmd.Flags().GetBool("registry-off")
	pullLatest, _ := cmd.Flags().GetBool("pull-latest")
	
	// Conversation control (Phase 0.3)
	noContinue, _ := cmd.Flags().GetBool("no-continue")
	continueConversation := !noContinue  // Default to true, unless --no-continue is specified
	
	app.Logger.Info("🚀 Starting Claude CLI container...")
	
	// Step 1: Load or create configuration
	app.Logger.Info("📋 Loading configuration...")
	config, err := app.ConfigMgr.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w. Try running 'claude-reactor config validate' to check your setup", err)
	}
	
	// Override config with command-line flags
	if image != "" {
		config.Variant = image
	}
	if account != "" {
		config.Account = account
	}
	
	// Handle danger mode with persistence logic
	// Only override if the flag was explicitly set
	if cmd.Flags().Changed("danger") {
		config.DangerMode = danger
		if danger {
			app.Logger.Info("🔥 Danger mode enabled and will be persisted")
		} else {
			app.Logger.Info("🛡️  Danger mode disabled and will be persisted")
		}
	} else if config.DangerMode {
		app.Logger.Info("🔥 Using persistent danger mode setting")
	}
	
	// Handle authentication flags
	if apikey != "" {
		app.Logger.Infof("🔑 Setting up API key for account: %s", config.Account)
		if err := app.AuthMgr.SetupAuth(config.Account, apikey); err != nil {
			return fmt.Errorf("failed to setup API key authentication: %w", err)
		}
		app.Logger.Info("✅ API key authentication configured")
	}
	
	if interactiveLogin {
		app.Logger.Infof("🔐 Forcing interactive login for account: %s", config.Account)
		// Note: Interactive login is handled by the Claude CLI inside the container
		// This flag will be passed to the container startup
	}
	
	// Auto-detect variant if not specified
	if config.Variant == "" {
		app.Logger.Info("🔍 Auto-detecting project type...")
		detectedVariant, err := app.ConfigMgr.AutoDetectVariant("")
		if err != nil {
			app.Logger.Warnf("Failed to auto-detect image: %v", err)
			app.Logger.Info("💡 Defaulting to 'base' image. Use --image flag to specify manually")
			config.Variant = "base"
		} else {
			config.Variant = detectedVariant
			app.Logger.Infof("✅ Auto-detected image: %s", config.Variant)
		}
	}
	
	// Validate configuration
	app.Logger.Info("✅ Validating configuration...")
	if err := app.ConfigMgr.ValidateConfig(config); err != nil {
		return fmt.Errorf("invalid configuration: %w. Try using --image with one of: base, go, full, cloud, k8s, or a custom Docker image", err)
	}
	
	// Save configuration to persist user preferences
	if err := app.ConfigMgr.SaveConfig(config); err != nil {
		app.Logger.Warnf("Failed to save configuration: %v", err)
		// Don't fail the entire operation for this, just warn
	}
	
	// Step 1.5: Validate custom Docker images
	builtinVariants := []string{"base", "go", "full", "cloud", "k8s"}
	isBuiltinVariant := false
	for _, variant := range builtinVariants {
		if config.Variant == variant {
			isBuiltinVariant = true
			break
		}
	}
	
	if !isBuiltinVariant {
		app.Logger.Infof("🔍 Validating custom Docker image: %s (compatibility + package analysis)", config.Variant)
		
		// Pull image if needed and validate it
		validationResult, err := app.ImageValidator.ValidateImage(ctx, config.Variant, true)
		if err != nil {
			return fmt.Errorf("failed to validate custom image '%s': %w. Ensure the image exists and is accessible", config.Variant, err)
		}
		
		if !validationResult.Compatible {
			app.Logger.Error("❌ Custom image validation failed:")
			for _, errMsg := range validationResult.Errors {
				app.Logger.Errorf("  - %s", errMsg)
			}
			return fmt.Errorf("custom image '%s' is not compatible with claude-reactor. See errors above", config.Variant)
		}
		
		// Show warnings if any
		if len(validationResult.Warnings) > 0 {
			app.Logger.Warn("⚠️ Custom image warnings:")
			for _, warning := range validationResult.Warnings {
				app.Logger.Warnf("  - %s", warning)
			}
		}
		
		app.Logger.Infof("✅ Custom image validated successfully: %s (digest: %.12s)", 
			config.Variant, validationResult.Digest)
		
		if validationResult.HasClaude {
			app.Logger.Debug("✅ Claude CLI detected in custom image")
		}
		
		// Show package information if available
		if packages, ok := validationResult.Metadata["packages"].(map[string]interface{}); ok {
			if totalAvailable, ok := packages["total_available"].(int); ok {
				if totalChecked, ok := packages["total_checked"].(int); ok {
					app.Logger.Infof("📦 Package analysis: %d/%d recommended tools available", totalAvailable, totalChecked)
				}
			}
		}
	}
	
	// Log registry configuration if relevant
	if devMode {
		app.Logger.Info("🔨 Registry: Dev mode enabled - forcing local builds")
	} else if registryOff {
		app.Logger.Info("🔨 Registry: Registry disabled - using local builds only")
	} else if pullLatest {
		app.Logger.Info("📦 Registry: Force pulling latest images from registry")
	}
	
	app.Logger.Infof("📋 Configuration: image=%s, account=%s, danger=%t, shell=%t, persist=%t", 
		config.Variant, config.Account, config.DangerMode, shell, persist)
	
	// Show which Claude config file will be mounted
	claudeConfigPath := app.AuthMgr.GetAccountConfigPath(config.Account)
	app.Logger.Infof("🔑 Claude config: %s", claudeConfigPath)
	
	// Step 2: Generate container and image names
	app.Logger.Info("🔧 Detecting system architecture...")
	arch, err := app.ArchDetector.GetHostArchitecture()
	if err != nil {
		return fmt.Errorf("failed to detect architecture: %w. Your system may not be supported", err)
	}
	
	containerName := app.DockerMgr.GenerateContainerName("", config.Variant, arch, config.Account)
	app.Logger.Infof("🏷️ Container name: %s", containerName)
	
	// Step 3: Check existing container status
	app.Logger.Info("🔍 Checking existing containers...")
	status, err := app.DockerMgr.GetContainerStatus(ctx, containerName)
	if err != nil {
		return fmt.Errorf("failed to check container status: %w. Check if Docker daemon is running", err)
	}
	
	var containerID string
	
	if status.Exists && status.Running {
		app.Logger.Infof("✅ Container %s is already running", containerName)
		containerID = status.ID
	} else {
		// Step 4: Build image if needed
		app.Logger.Info("🐳 Preparing Docker environment...")
		platform, err := app.ArchDetector.GetDockerPlatform()
		if err != nil {
			return fmt.Errorf("failed to get Docker platform: %w. Architecture detection failed", err)
		}
		
		imageName := app.DockerMgr.GetImageName(config.Variant, arch)
		app.Logger.Infof("🔨 Building image if needed: %s", imageName)
		app.Logger.Info("⏳ This may take a few minutes for first-time setup...")
		
		// Build image with registry support (Phase 0.1)
		err = app.DockerMgr.BuildImageWithRegistry(ctx, config.Variant, platform, devMode, registryOff, pullLatest)
		if err != nil {
			return fmt.Errorf("failed to build image: %w. Try running 'docker system prune' to free space or check your Dockerfile", err)
		}
		
		// Step 5: Create container configuration
		containerConfig := &pkg.ContainerConfig{
			Image:            imageName,
			Name:             containerName,
			Variant:          config.Variant,
			Platform:         platform,
			Interactive:      true,
			TTY:              true,
			Remove:           false, // Don't auto-remove - we manage lifecycle
			RunClaudeUpgrade: true,  // Run claude upgrade after container startup
		}
		
		// Add mounts (skip if requested for testing)
		if !noMounts {
			app.Logger.Info("📁 Configuring container mounts...")
			err = addMountsToContainer(app, containerConfig, config.Account, mounts)
			if err != nil {
				return fmt.Errorf("failed to configure mounts: %w. Check that source directories exist and are accessible", err)
			}
		} else {
			app.Logger.Info("🚫 Skipping mounts due to --no-mounts flag")
			// Set empty mounts to prevent default mount creation
			containerConfig.Mounts = []pkg.Mount{}
		}
		
		// Step 6: Start container
		if status.Exists {
			app.Logger.Infof("🔄 Starting existing container: %s", containerName)
			app.Logger.Info("💡 Container already exists, reusing it...")
			// Container exists but isn't running - start it
			// TODO: Implement StartExistingContainer method or use Docker SDK directly
			containerID = status.ID
		} else {
			app.Logger.Infof("🏗️ Creating new container: %s", containerName)
			app.Logger.Info("⚡ Starting container with your configuration...")
			containerID, err = app.DockerMgr.StartContainer(ctx, containerConfig)
			if err != nil {
				return fmt.Errorf("failed to start container: %w. Check Docker daemon is running and try 'docker system prune'", err)
			}
			app.Logger.Info("✅ Container created and started successfully!")
		}
	}
	
	// Step 7: Attach to container
	var command []string
	if shell {
		command = []string{"/bin/bash"}
		app.Logger.Info("🐚 Launching interactive shell in container...")
		app.Logger.Info("💡 Type 'claude' to start Claude CLI, or 'exit' to leave the container")
	} else {
		// Build Claude CLI command with flags
		command = []string{"claude"}
		
		if config.DangerMode {
			command = append(command, "--dangerously-skip-permissions")
			app.Logger.Info("🤖 Launching Claude CLI in DANGER MODE...")
			app.Logger.Info("⚠️  Danger mode bypasses permission checks - use with caution!")
		} else {
			app.Logger.Info("🤖 Launching Claude CLI in container...")
		}
		
		// Conversation control (Phase 0.3)
		if !continueConversation {
			command = append(command, "--no-conversation-continuation")
			app.Logger.Info("💬 Conversation continuation disabled")
		} else {
			app.Logger.Debug("💬 Conversation continuation enabled (default)")
		}
	}
	
	// Attach to container
	err = app.DockerMgr.AttachToContainer(ctx, containerName, command, true)
	if err != nil {
		return fmt.Errorf("failed to attach to container: %w. Try using 'docker exec -it %s %s' as fallback", err, containerName, strings.Join(command, " "))
	}
	
	// Step 8: Handle container persistence
	if !persist {
		app.Logger.Info("🧹 Stopping container due to --persist=false...")
		if err := app.DockerMgr.StopContainer(ctx, containerID); err != nil {
			app.Logger.Warnf("Failed to stop container: %v", err)
		}
	} else {
		app.Logger.Info("💾 Container will remain running (use 'claude-reactor clean' to stop)")
	}
	
	return nil
}

// addMountsToContainer configures the container mounts
func addMountsToContainer(app *pkg.AppContainer, containerConfig *pkg.ContainerConfig, account string, userMounts []string) error {
	// Add default mounts (project directory, Claude config)
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	
	// Project mount - avoid circular mount if we're already in /app
	targetPath := "/app"
	if projectDir == "/app" {
		targetPath = "/workspace"  // Use different path to avoid circular mount
	}
	
	err = app.MountMgr.AddMountToConfig(containerConfig, projectDir, targetPath)
	if err != nil {
		return fmt.Errorf("failed to add project mount: %w", err)
	}
	app.Logger.Infof("📁 Project mount: %s -> %s", projectDir, targetPath)
	
	// Claude config mount
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	
	claudeConfigPath := filepath.Join(homeDir, ".claude")
	if account != "" && account != "default" {
		claudeConfigPath = app.AuthMgr.GetAccountConfigPath(account)
	}
	
	// Only mount if config exists
	if _, err := os.Stat(claudeConfigPath); err == nil {
		err = app.MountMgr.AddMountToConfig(containerConfig, claudeConfigPath, "/home/claude/.claude")
		if err != nil {
			app.Logger.Warnf("Failed to add Claude config mount: %v", err)
		}
	}
	
	// Add user-specified mounts
	for _, mountPath := range userMounts {
		validatedPath, err := app.MountMgr.ValidateMountPath(mountPath)
		if err != nil {
			return fmt.Errorf("invalid mount path '%s': %w", mountPath, err)
		}
		
		// Generate target path (mount to /mnt/basename)
		targetPath := "/mnt/" + filepath.Base(validatedPath)
		err = app.MountMgr.AddMountToConfig(containerConfig, validatedPath, targetPath)
		if err != nil {
			return fmt.Errorf("failed to add user mount '%s': %w", mountPath, err)
		}
		
		app.Logger.Infof("📁 Added mount: %s -> %s", validatedPath, targetPath)
	}
	
	return nil
}

// cleanContainers implements the clean command logic
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


// showEnhancedConfig displays comprehensive configuration information (Phase 0.4)
func showEnhancedConfig(cmd *cobra.Command, app *pkg.AppContainer) error {
	// Parse flags
	showRaw, _ := cmd.Flags().GetBool("raw")
	verbose, _ := cmd.Flags().GetBool("verbose")
	
	// Load current configuration
	config, err := app.ConfigMgr.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	// Basic configuration
	cmd.Printf("=== Claude-Reactor Configuration ===\n\n")
	cmd.Printf("Project Configuration:\n")
	cmd.Printf("  Variant:     %s\n", getDisplayValue(config.Variant, "auto-detect"))
	cmd.Printf("  Account:     %s\n", getDisplayValue(config.Account, "default"))
	cmd.Printf("  Danger Mode: %t\n", config.DangerMode)
	cmd.Printf("  Project Path: %s\n", getDisplayValue(config.ProjectPath, getCurrentDir()))
	
	// Registry configuration (Phase 0.4)
	cmd.Printf("\nRegistry Configuration:\n")
	registryURL := os.Getenv("CLAUDE_REACTOR_REGISTRY")
	if registryURL == "" {
		registryURL = "ghcr.io/dyluth/claude-reactor (default)"
	}
	cmd.Printf("  Registry URL: %s\n", registryURL)
	
	registryTag := os.Getenv("CLAUDE_REACTOR_TAG")
	if registryTag == "" {
		registryTag = "latest (default)"
	}
	cmd.Printf("  Tag:          %s\n", registryTag)
	
	useRegistry := os.Getenv("CLAUDE_REACTOR_USE_REGISTRY")
	registryStatus := "enabled (default)"
	if useRegistry == "false" || useRegistry == "0" {
		registryStatus = "disabled"
	}
	cmd.Printf("  Status:       %s\n", registryStatus)
	
	// System information
	if verbose {
		cmd.Printf("\nSystem Information:\n")
		
		// Architecture
		arch, err := app.ArchDetector.GetHostArchitecture()
		if err != nil {
			arch = fmt.Sprintf("error: %v", err)
		}
		cmd.Printf("  Architecture: %s\n", arch)
		
		platform, err := app.ArchDetector.GetDockerPlatform()
		if err != nil {
			platform = fmt.Sprintf("error: %v", err)
		}
		cmd.Printf("  Docker Platform: %s\n", platform)
		cmd.Printf("  Multi-arch Support: %t\n", app.ArchDetector.IsMultiArchSupported())
		
		// Container naming
		containerName := app.DockerMgr.GenerateContainerName("", config.Variant, arch, config.Account)
		cmd.Printf("  Container Name: %s\n", containerName)
		
		projectHash := app.DockerMgr.GenerateProjectHash("")
		cmd.Printf("  Project Hash: %s\n", projectHash)
		
		imageName := app.DockerMgr.GetImageName(config.Variant, arch)
		cmd.Printf("  Image Name: %s\n", imageName)
		
		// Authentication paths
		authPath := app.AuthMgr.GetAccountConfigPath(config.Account)
		cmd.Printf("  Auth Config Path: %s\n", authPath)
		
		apiKeyFile := app.AuthMgr.GetAPIKeyFile(config.Account)
		cmd.Printf("  API Key File: %s\n", apiKeyFile)
	}
	
	// Raw configuration (Phase 0.4)
	if showRaw {
		cmd.Printf("\nRaw Configuration File:\n")
		configPath := ".claude-reactor" // Default config file name
		
		if data, err := os.ReadFile(configPath); err != nil {
			cmd.Printf("  File: %s (not found or unreadable)\n", configPath)
			cmd.Printf("  Status: Using default configuration\n")
		} else {
			cmd.Printf("  File: %s\n", configPath)
			cmd.Printf("  Contents:\n")
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					cmd.Printf("    %s\n", line)
				}
			}
		}
	}
	
	return nil
}

// getDisplayValue returns the value or a default display string
func getDisplayValue(value, defaultDisplay string) string {
	if value == "" {
		return fmt.Sprintf("(using %s)", defaultDisplay)
	}
	return value
}

// getCurrentDir safely gets the current directory
func getCurrentDir() string {
	if dir, err := os.Getwd(); err == nil {
		return dir
	}
	return "unknown"
}

// newDevContainerCmd creates the devcontainer command for VS Code integration
func newDevContainerCmd(app *pkg.AppContainer) *cobra.Command {
	var devcontainerCmd = &cobra.Command{
		Use:   "devcontainer",
		Short: "VS Code Dev Container integration",
		Long:  `Generate and manage VS Code Dev Container configurations for seamless IDE integration.`,
	}
	
	// Generate subcommand
	generateCmd := &cobra.Command{
		Use:   "generate [project-path]",
		Short: "Generate .devcontainer configuration",
		Long: `Generate VS Code Dev Container configuration based on project detection.
		
This command will:
- Detect your project type (Go, Rust, Node.js, Python, Java, etc.)
- Select the appropriate claude-reactor container variant
- Install relevant VS Code extensions automatically
- Configure proper mount points and settings
- Enable one-click "Reopen in Container" workflow`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return generateDevContainer(cmd, args, app)
		},
	}
	
	// Validate subcommand
	validateCmd := &cobra.Command{
		Use:   "validate [project-path]",
		Short: "Validate existing .devcontainer configuration",
		Long:  "Check if existing .devcontainer configuration is valid and properly structured.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return validateDevContainer(cmd, args, app)
		},
	}
	
	// Update subcommand
	updateCmd := &cobra.Command{
		Use:   "update [project-path]",
		Short: "Update existing .devcontainer configuration",
		Long:  "Update existing .devcontainer configuration with latest templates and settings.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return updateDevContainer(cmd, args, app)
		},
	}
	
	// Remove subcommand
	removeCmd := &cobra.Command{
		Use:   "remove [project-path]",
		Short: "Remove .devcontainer configuration",
		Long:  "Remove .devcontainer directory and configurations from project.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return removeDevContainer(cmd, args, app)
		},
	}
	
	// Info subcommand
	infoCmd := &cobra.Command{
		Use:   "info [project-path]",
		Short: "Show project detection information",
		Long:  "Display detailed project detection results and recommended VS Code extensions.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return showDevContainerInfo(cmd, args, app)
		},
	}
	
	// Add flags
	generateCmd.Flags().String("image", "", "Force specific container image (base, go, full, cloud, k8s)")
	generateCmd.Flags().Bool("force", false, "Overwrite existing .devcontainer configuration")
	
	// Help subcommand
	helpCmd := &cobra.Command{
		Use:   "help",
		Short: "Detailed VS Code Dev Container setup guide",
		Long:  "Complete guide for setting up and using VS Code Dev Containers with claude-reactor.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showVSCodeHelp(cmd, args, app)
		},
	}
	
	// Add subcommands
	devcontainerCmd.AddCommand(
		generateCmd,
		validateCmd,
		updateCmd,
		removeCmd,
		infoCmd,
		helpCmd,
	)
	
	return devcontainerCmd
}

// generateDevContainer generates a new .devcontainer configuration
func generateDevContainer(cmd *cobra.Command, args []string, app *pkg.AppContainer) error {
	projectPath := getCurrentDir()
	if len(args) > 0 {
		projectPath = args[0]
	}
	
	// Load current config
	config, err := app.ConfigMgr.LoadConfig()
	if err != nil {
		config = app.ConfigMgr.GetDefaultConfig()
	}
	config.ProjectPath = projectPath
	
	// Override image if specified
	if image, _ := cmd.Flags().GetString("image"); image != "" {
		if err := app.ConfigMgr.ValidateConfig(&pkg.Config{Variant: image}); err != nil {
			return fmt.Errorf("invalid image '%s': %w", image, err)
		}
		config.Variant = image
	}
	
	// Check if devcontainer already exists
	force, _ := cmd.Flags().GetBool("force")
	devcontainerPath := filepath.Join(projectPath, ".devcontainer")
	if _, err := os.Stat(devcontainerPath); err == nil && !force {
		return fmt.Errorf(".devcontainer already exists at %s (use --force to overwrite)", devcontainerPath)
	}
	
	app.Logger.Infof("🔧 Generating VS Code Dev Container configuration...")
	
	// Generate devcontainer configuration
	if err := app.DevContainerMgr.GenerateDevContainer(projectPath, config); err != nil {
		return fmt.Errorf("failed to generate devcontainer: %w", err)
	}
	
	app.Logger.Infof("✅ Successfully generated .devcontainer configuration!")
	app.Logger.Infof("")
	app.Logger.Infof("📋 VS Code Setup Instructions:")
	app.Logger.Infof("   1. Install 'Dev Containers' extension: ms-vscode-remote.remote-containers")
	app.Logger.Infof("   2. Open this project in VS Code: code .")
	app.Logger.Infof("   3. VS Code will show: 'Folder contains a Dev Container configuration file'")
	app.Logger.Infof("   4. Click 'Reopen in Container' or use Command Palette:")
	app.Logger.Infof("      • Press Ctrl/Cmd+Shift+P")
	app.Logger.Infof("      • Type: 'Dev Containers: Reopen in Container'")
	app.Logger.Infof("   5. Wait for container build and VS Code connection (~30-60 seconds)")
	app.Logger.Infof("   6. Extensions will auto-install - check bottom status bar")
	app.Logger.Infof("")
	app.Logger.Infof("🔧 Troubleshooting:")
	app.Logger.Infof("   • If 'Dockerfile does not exist': Ensure you opened VS Code from the project root directory")
	app.Logger.Infof("   • If build fails: Ensure Docker is running and try 'Rebuild Container'")
	app.Logger.Infof("   • If extensions missing: Check 'Extensions' view, they install automatically")
	app.Logger.Infof("   • To rebuild: Command Palette → 'Dev Containers: Rebuild Container'")
	app.Logger.Infof("   • To exit container: Command Palette → 'Dev Containers: Reopen Folder Locally'")
	app.Logger.Infof("   • Path issues: Run 'claude-reactor devcontainer validate' to check configuration")
	app.Logger.Infof("")
	app.Logger.Infof("🚀 You're all set for containerized development!")
	
	return nil
}

// validateDevContainer validates existing .devcontainer configuration
func validateDevContainer(cmd *cobra.Command, args []string, app *pkg.AppContainer) error {
	projectPath := getCurrentDir()
	if len(args) > 0 {
		projectPath = args[0]
	}
	
	app.Logger.Infof("🔍 Validating .devcontainer configuration...")
	
	if err := app.DevContainerMgr.ValidateDevContainer(projectPath); err != nil {
		app.Logger.Errorf("❌ Validation failed: %v", err)
		return err
	}
	
	app.Logger.Infof("✅ DevContainer configuration is valid!")
	return nil
}

// updateDevContainer updates existing .devcontainer configuration
func updateDevContainer(cmd *cobra.Command, args []string, app *pkg.AppContainer) error {
	projectPath := getCurrentDir()
	if len(args) > 0 {
		projectPath = args[0]
	}
	
	// Load current config
	config, err := app.ConfigMgr.LoadConfig()
	if err != nil {
		config = app.ConfigMgr.GetDefaultConfig()
	}
	config.ProjectPath = projectPath
	
	app.Logger.Infof("🔄 Updating .devcontainer configuration...")
	
	if err := app.DevContainerMgr.UpdateDevContainer(projectPath, config); err != nil {
		return fmt.Errorf("failed to update devcontainer: %w", err)
	}
	
	app.Logger.Infof("✅ Successfully updated .devcontainer configuration!")
	return nil
}

// removeDevContainer removes .devcontainer configuration
func removeDevContainer(cmd *cobra.Command, args []string, app *pkg.AppContainer) error {
	projectPath := getCurrentDir()
	if len(args) > 0 {
		projectPath = args[0]
	}
	
	app.Logger.Infof("🗑️  Removing .devcontainer configuration...")
	
	if err := app.DevContainerMgr.RemoveDevContainer(projectPath); err != nil {
		return fmt.Errorf("failed to remove devcontainer: %w", err)
	}
	
	return nil
}

// showDevContainerInfo shows project detection information
func showDevContainerInfo(cmd *cobra.Command, args []string, app *pkg.AppContainer) error {
	projectPath := getCurrentDir()
	if len(args) > 0 {
		projectPath = args[0]
	}
	
	app.Logger.Infof("🔍 Analyzing project: %s", projectPath)
	
	// Perform project detection
	detection, err := app.DevContainerMgr.DetectProjectType(projectPath)
	if err != nil {
		return fmt.Errorf("failed to detect project type: %w", err)
	}
	
	// Display results
	fmt.Printf("\n📊 Project Detection Results:\n")
	fmt.Printf("  Project Type: %s\n", detection.ProjectType)
	fmt.Printf("  Recommended Variant: %s\n", detection.Variant)
	fmt.Printf("  Confidence: %.1f%%\n", detection.Confidence*100)
	
	if len(detection.Languages) > 0 {
		fmt.Printf("  Languages: %s\n", strings.Join(detection.Languages, ", "))
	}
	
	if len(detection.Frameworks) > 0 {
		fmt.Printf("  Frameworks: %s\n", strings.Join(detection.Frameworks, ", "))
	}
	
	if len(detection.Features) > 0 {
		fmt.Printf("  Features: %s\n", strings.Join(detection.Features, ", "))
	}
	
	if len(detection.Files) > 0 {
		fmt.Printf("  Detected Files: %s\n", strings.Join(detection.Files, ", "))
	}
	
	if len(detection.Extensions) > 0 {
		fmt.Printf("\n📦 Recommended VS Code Extensions:\n")
		for i, ext := range detection.Extensions {
			if i < 10 { // Show first 10 extensions
				fmt.Printf("  - %s\n", ext)
			}
		}
		if len(detection.Extensions) > 10 {
			fmt.Printf("  ... and %d more extensions\n", len(detection.Extensions)-10)
		}
	}
	
	fmt.Printf("\n💡 Run 'claude-reactor devcontainer generate' to create .devcontainer configuration\n")
	
	return nil
}

// showVSCodeHelp displays comprehensive VS Code Dev Container setup guide
func showVSCodeHelp(cmd *cobra.Command, args []string, app *pkg.AppContainer) error {
	helpText := `
🎯 VS Code Dev Container Integration Guide

═══════════════════════════════════════════════════════════════════════════════

📋 PREREQUISITES

1. Install Docker Desktop
   • macOS/Windows: https://www.docker.com/products/docker-desktop
   • Linux: Install Docker Engine + Docker Compose

2. Install VS Code
   • Download: https://code.visualstudio.com/

3. Install Dev Containers Extension
   • Open VS Code
   • Go to Extensions (Ctrl/Cmd+Shift+X)
   • Search: "Dev Containers" 
   • Install: ms-vscode-remote.remote-containers
   • Or install via command line: code --install-extension ms-vscode-remote.remote-containers

═══════════════════════════════════════════════════════════════════════════════

🚀 QUICK START

1. Generate devcontainer for your project:
   claude-reactor devcontainer generate

2. Open project in VS Code:
   code .

3. When prompted, click "Reopen in Container"
   • Or use Command Palette: Ctrl/Cmd+Shift+P → "Dev Containers: Reopen in Container"

4. Wait for container build (30-60 seconds first time)

5. Start coding with full IDE integration! 🎉

═══════════════════════════════════════════════════════════════════════════════

🔧 DETAILED WORKFLOW

Step 1: Project Analysis
   • Run: claude-reactor devcontainer info
   • Review detected project type, variant, and recommended extensions
   
Step 2: Generate Configuration  
   • Run: claude-reactor devcontainer generate
   • Creates .devcontainer/devcontainer.json with optimal settings
   • Force overwrite existing: claude-reactor devcontainer generate --force
   • Specify image: claude-reactor devcontainer generate --image go

Step 3: Open in VS Code
   • Launch VS Code in project directory: code .
   • VS Code detects .devcontainer/devcontainer.json automatically
   • Shows notification: "Folder contains a Dev Container configuration file"

Step 4: Container Setup
   • Click "Reopen in Container" in notification
   • Alternative: Command Palette (Ctrl/Cmd+Shift+P) → "Dev Containers: Reopen in Container"
   • VS Code builds Docker container (uses claude-reactor Dockerfile)
   • Connects to container and installs extensions
   • Terminal opens inside container environment

Step 5: Development
   • All tools available: go, rust, python, node, docker, kubectl, etc.
   • Extensions auto-installed and configured
   • IntelliSense, debugging, and Git work seamlessly
   • Files sync between host and container automatically

═══════════════════════════════════════════════════════════════════════════════

⚙️ CONTAINER VARIANTS & PROJECT TYPES

Base Variant (Python, Node.js):
   • Languages: Python 3.11, Node.js 18, pip, npm, uv
   • Extensions: Python, TypeScript, ESLint, Prettier
   • Use case: Web apps, APIs, scripts

Go Variant (Go projects):
   • Languages: Go 1.23, all Go tools (gofmt, golint, etc.)
   • Extensions: Go, go-nightly, vscode-go
   • Use case: Go applications, CLIs, microservices

Full Variant (Rust, Java):
   • Languages: Rust, Java 17, Maven, Gradle
   • Extensions: rust-analyzer, Java extension pack
   • Use case: System programming, enterprise apps

Cloud Variant (Cloud development):
   • All Full variant tools + AWS CLI, gcloud, Azure CLI
   • Extensions: AWS Toolkit, Cloud Code, Docker
   • Use case: Cloud-native applications, DevOps

K8s Variant (Kubernetes development):
   • All Full variant tools + kubectl, helm, k9s, stern
   • Extensions: Kubernetes Tools, Helm IntelliSense
   • Use case: Kubernetes operators, microservices, DevOps

═══════════════════════════════════════════════════════════════════════════════

🔍 TROUBLESHOOTING

Problem: "Reopen in Container" notification doesn't appear
Solution: 
   • Ensure .devcontainer/devcontainer.json exists
   • Reload VS Code: Ctrl/Cmd+Shift+P → "Developer: Reload Window"
   • Manually trigger: Command Palette → "Dev Containers: Reopen in Container"

Problem: Container build fails
Solutions:
   • Ensure Docker is running: docker ps
   • Check Docker resources (memory > 4GB recommended)
   • Rebuild: Command Palette → "Dev Containers: Rebuild Container"
   • View build logs: Command Palette → "Dev Containers: Show Container Log"

Problem: Extensions not installing
Solutions:
   • Check Extensions view - they install automatically
   • Restart VS Code if needed
   • Manually install missing extensions inside container
   • Check internet connection for extension downloads

Problem: Slow performance
Solutions:
   • Increase Docker memory allocation (Docker Desktop settings)
   • Use volume mounts instead of bind mounts for large projects
   • Close unnecessary extensions
   • Check host system resources

Problem: File sync issues
Solutions:
   • Files should sync automatically with bind mounts
   • If not syncing, restart container: Command Palette → "Dev Containers: Rebuild Container"
   • Check file permissions and Docker volume settings

Problem: Git integration not working
Solutions:
   • VS Code mounts ~/.gitconfig automatically
   • For SSH keys, ensure ssh-agent is running on host
   • Configure Git inside container if needed: git config --global user.name "Your Name"

═══════════════════════════════════════════════════════════════════════════════

💡 PRO TIPS

1. Keyboard Shortcuts:
   • Open Command Palette: Ctrl/Cmd+Shift+P
   • Open terminal: Ctrl/Cmd+backtick
   • Switch between local/container: Command Palette → "Dev Containers: Reopen Folder Locally"

2. Multiple Projects:
   • Each project gets its own devcontainer configuration
   • VS Code remembers container settings per project
   • Switch between projects seamlessly

3. Team Collaboration:
   • Commit .devcontainer/ to version control
   • All team members get identical development environment
   • No more "works on my machine" issues

4. Performance Optimization:
   • First container build is slow (~2-3 minutes)
   • Subsequent builds are fast (~30 seconds) due to Docker layer caching
   • Keep Docker running to avoid startup delays

5. Custom Extensions:
   • Edit .devcontainer/devcontainer.json to add custom extensions
   • Run: claude-reactor devcontainer update to refresh configuration

═══════════════════════════════════════════════════════════════════════════════

📚 USEFUL COMMANDS

claude-reactor devcontainer info              # Analyze current project
claude-reactor devcontainer generate          # Create .devcontainer config  
claude-reactor devcontainer generate --force  # Overwrite existing config
claude-reactor devcontainer validate          # Check config validity
claude-reactor devcontainer update            # Update existing config
claude-reactor devcontainer remove            # Delete .devcontainer directory

VS Code Command Palette (Ctrl/Cmd+Shift+P):
   • Dev Containers: Reopen in Container
   • Dev Containers: Reopen Folder Locally  
   • Dev Containers: Rebuild Container
   • Dev Containers: Show Container Log
   • Dev Containers: Attach to Running Container

═══════════════════════════════════════════════════════════════════════════════

🎯 SUCCESS INDICATORS

✅ Container Built Successfully:
   • VS Code status bar shows: "Dev Container: Claude Reactor [Variant]"
   • Terminal prompt shows container environment
   • Extensions appear in Extensions view

✅ Tools Working:
   • IntelliSense provides code completion
   • Go to Definition works (F12)
   • Debugging available (F5)
   • Integrated terminal has all development tools

✅ Full Integration:
   • Git operations work seamlessly
   • File changes sync between host and container
   • Port forwarding works for web applications
   • Extensions provide full functionality

Happy containerized coding! 🚀

Run 'claude-reactor devcontainer help' anytime to see this guide.

`
	
	fmt.Print(helpText)
	return nil
}

// newTemplateCmd creates the template command for project scaffolding
func newTemplateCmd(app *pkg.AppContainer) *cobra.Command {
	var templateCmd = &cobra.Command{
		Use:   "template",
		Short: "Project template management and scaffolding",
		Long: `The template command provides intelligent project scaffolding and template management.
		
Create new projects from templates, manage custom templates, and scaffold projects
with best practices and proper configuration for claude-reactor development.`,
		Example: `# List available templates
claude-reactor template list

# Create new project from template
claude-reactor template new go-api my-api

# Interactive project creation
claude-reactor template init

# Show template details
claude-reactor template show go-api`,
	}

	templateCmd.AddCommand(
		newTemplateListCmd(app),
		newTemplateShowCmd(app),
		newTemplateNewCmd(app),
		newTemplateInitCmd(app),
		newTemplateCreateCmd(app),
		newTemplateInstallCmd(app),
		newTemplateUninstallCmd(app),
		newTemplateValidateCmd(app),
	)

	return templateCmd
}

// newTemplateListCmd lists available templates
func newTemplateListCmd(app *pkg.AppContainer) *cobra.Command {
	var listCmd = &cobra.Command{
		Use:   "list [language]",
		Short: "List available project templates",
		Long: `Lists all available project templates or filters by language.
		
Templates are organized by language and framework, each providing a complete
project structure with best practices and proper claude-reactor integration.`,
		Example: `# List all templates
claude-reactor template list

# List Go templates only  
claude-reactor template list go

# List with details
claude-reactor template list --detailed`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return listTemplates(cmd, args, app)
		},
	}

	listCmd.Flags().BoolP("detailed", "d", false, "Show detailed template information")
	return listCmd
}

// newTemplateShowCmd shows template details
func newTemplateShowCmd(app *pkg.AppContainer) *cobra.Command {
	var showCmd = &cobra.Command{
		Use:   "show <template-name>",
		Short: "Show detailed information about a template",
		Long: `Displays comprehensive information about a specific template including:
- Description and metadata
- Files that will be created
- Template variables
- Post-creation commands
- Requirements and dependencies`,
		Example: `# Show Go API template details
claude-reactor template show go-api

# Show with file contents
claude-reactor template show go-api --files`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return showTemplate(cmd, args, app)
		},
	}

	showCmd.Flags().Bool("files", false, "Show file contents preview")
	return showCmd
}

// newTemplateNewCmd creates new project from template
func newTemplateNewCmd(app *pkg.AppContainer) *cobra.Command {
	var newCmd = &cobra.Command{
		Use:   "new <template-name> <project-name> [path]",
		Short: "Create new project from template",
		Long: `Creates a new project from the specified template with intelligent scaffolding.
		
The command will:
1. Create project directory structure
2. Generate files with template variables
3. Set up .claude-reactor configuration
4. Initialize git repository (optional)
5. Generate devcontainer configuration (optional)
6. Run post-creation commands`,
		Example: `# Create Go API project
claude-reactor template new go-api my-api

# Create in specific directory
claude-reactor template new go-api my-api ./projects/

# Skip git initialization
claude-reactor template new go-api my-api --no-git

# Set template variables
claude-reactor template new go-api my-api --var PORT=3000 --var AUTHOR="John Doe"`,
		Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			return createFromTemplate(cmd, args, app)
		},
	}

	newCmd.Flags().Bool("no-git", false, "Skip git repository initialization")
	newCmd.Flags().Bool("no-devcontainer", false, "Skip devcontainer generation")
	newCmd.Flags().StringSlice("var", []string{}, "Set template variables (key=value)")
	newCmd.Flags().Bool("force", false, "Overwrite existing directory")
	return newCmd
}

// newTemplateInitCmd runs interactive project creation
func newTemplateInitCmd(app *pkg.AppContainer) *cobra.Command {
	var initCmd = &cobra.Command{
		Use:   "init [path]",
		Short: "Interactive project initialization wizard",
		Long: `Runs an interactive wizard to create a new project.
		
The wizard will:
1. Show available templates
2. Help you select the best template
3. Collect project information
4. Set template variables
5. Create the project with all configurations`,
		Example: `# Start interactive wizard in current directory
claude-reactor template init

# Start wizard in specific directory
claude-reactor template init ./projects/`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return interactiveInit(cmd, args, app)
		},
	}

	return initCmd
}

// newTemplateCreateCmd creates template from existing project
func newTemplateCreateCmd(app *pkg.AppContainer) *cobra.Command {
	var createCmd = &cobra.Command{
		Use:   "create <template-name> [project-path]",
		Short: "Create template from existing project",
		Long: `Creates a new template based on an existing project structure.
		
This is useful for:
- Creating custom templates from your projects
- Sharing project structures with team members
- Building organization-specific templates`,
		Example: `# Create template from current directory
claude-reactor template create my-template

# Create template from specific project
claude-reactor template create my-template ./my-project/

# Include custom metadata
claude-reactor template create my-template --description "My custom template" --author "John Doe"`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return createTemplate(cmd, args, app)
		},
	}

	createCmd.Flags().String("description", "", "Template description")
	createCmd.Flags().String("author", "", "Template author")
	createCmd.Flags().StringSlice("tags", []string{}, "Template tags")
	return createCmd
}

// newTemplateInstallCmd installs template from file or URL
func newTemplateInstallCmd(app *pkg.AppContainer) *cobra.Command {
	var installCmd = &cobra.Command{
		Use:   "install <source>",
		Short: "Install template from file or URL",
		Long: `Installs a template from a local file or remote URL.
		
Supported sources:
- Local YAML files (template.yaml)
- Git repositories (future)
- Template registries (future)`,
		Example: `# Install from local file
claude-reactor template install ./my-template.yaml

# Install from URL (future)
claude-reactor template install https://example.com/template.yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return installTemplate(cmd, args, app)
		},
	}

	return installCmd
}

// newTemplateUninstallCmd removes a template
func newTemplateUninstallCmd(app *pkg.AppContainer) *cobra.Command {
	var uninstallCmd = &cobra.Command{
		Use:   "uninstall <template-name>",
		Short: "Remove an installed template",
		Long: `Removes a template from the local template registry.
		
This will permanently delete the template configuration and files.
Built-in templates cannot be uninstalled.`,
		Example: `# Remove custom template
claude-reactor template uninstall my-custom-template

# Force removal without confirmation
claude-reactor template uninstall my-template --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return uninstallTemplate(cmd, args, app)
		},
	}

	uninstallCmd.Flags().Bool("force", false, "Force removal without confirmation")
	return uninstallCmd
}

// newTemplateValidateCmd validates a template
func newTemplateValidateCmd(app *pkg.AppContainer) *cobra.Command {
	var validateCmd = &cobra.Command{
		Use:   "validate <template-file>",
		Short: "Validate template configuration",
		Long: `Validates a template configuration file for correctness.
		
Checks:
- YAML syntax
- Required fields
- File references
- Variable definitions
- Template syntax`,
		Example: `# Validate local template file
claude-reactor template validate ./template.yaml

# Validate installed template
claude-reactor template validate go-api`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return validateTemplate(cmd, args, app)
		},
	}

	return validateCmd
}

// listTemplates lists available project templates
func listTemplates(cmd *cobra.Command, args []string, app *pkg.AppContainer) error {
	detailed, _ := cmd.Flags().GetBool("detailed")
	
	var templates []*pkg.ProjectTemplate
	var err error
	
	if len(args) > 0 {
		// Filter by language
		language := args[0]
		templates, err = app.TemplateMgr.GetTemplatesForLanguage(language)
		if err != nil {
			return fmt.Errorf("failed to get templates for language %s: %w", language, err)
		}
		app.Logger.Infof("📋 Available %s templates:", strings.Title(language))
	} else {
		// Get all templates
		templates, err = app.TemplateMgr.ListTemplates()
		if err != nil {
			return fmt.Errorf("failed to list templates: %w", err)
		}
		app.Logger.Infof("📋 Available project templates:")
	}
	
	if len(templates) == 0 {
		fmt.Println("\n❌ No templates found")
		fmt.Println("💡 Templates will be automatically created on first use")
		return nil
	}
	
	if detailed {
		for _, template := range templates {
			fmt.Printf("\n🎯 %s (%s)\n", template.Name, template.Language)
			fmt.Printf("   Description: %s\n", template.Description)
			fmt.Printf("   Framework: %s\n", template.Framework)
			fmt.Printf("   Variant: %s\n", template.Variant)
			fmt.Printf("   Version: %s\n", template.Version)
			if len(template.Tags) > 0 {
				fmt.Printf("   Tags: %s\n", strings.Join(template.Tags, ", "))
			}
			fmt.Printf("   Files: %d\n", len(template.Files))
			if len(template.Variables) > 0 {
				fmt.Printf("   Variables: %d\n", len(template.Variables))
			}
			if template.DevContainer {
				fmt.Printf("   ✅ Includes VS Code Dev Container\n")
			}
		}
	} else {
		// Group by language
		languageGroups := make(map[string][]*pkg.ProjectTemplate)
		for _, template := range templates {
			languageGroups[template.Language] = append(languageGroups[template.Language], template)
		}
		
		for language, langTemplates := range languageGroups {
			fmt.Printf("\n📁 %s:\n", strings.Title(language))
			for _, template := range langTemplates {
				framework := ""
				if template.Framework != "" {
					framework = fmt.Sprintf(" (%s)", template.Framework)
				}
				devcontainer := ""
				if template.DevContainer {
					devcontainer = " 📦"
				}
				fmt.Printf("  • %s%s - %s%s\n", template.Name, framework, template.Description, devcontainer)
			}
		}
		
		fmt.Printf("\n💡 Use 'claude-reactor template show <template-name>' for details\n")
		fmt.Printf("💡 Use 'claude-reactor template new <template-name> <project-name>' to create project\n")
	}
	
	return nil
}

// showTemplate shows detailed information about a template
func showTemplate(cmd *cobra.Command, args []string, app *pkg.AppContainer) error {
	templateName := args[0]
	showFiles, _ := cmd.Flags().GetBool("files")
	
	template, err := app.TemplateMgr.GetTemplate(templateName)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}
	
	fmt.Printf("🎯 Template: %s\n", template.Name)
	fmt.Printf("📝 Description: %s\n", template.Description)
	fmt.Printf("🔧 Language: %s\n", template.Language)
	if template.Framework != "" {
		fmt.Printf("🏗️  Framework: %s\n", template.Framework)
	}
	fmt.Printf("📦 Variant: %s\n", template.Variant)
	fmt.Printf("🏷️  Version: %s\n", template.Version)
	if template.Author != "" {
		fmt.Printf("👤 Author: %s\n", template.Author)
	}
	if len(template.Tags) > 0 {
		fmt.Printf("🏷️  Tags: %s\n", strings.Join(template.Tags, ", "))
	}
	
	if template.DevContainer {
		fmt.Printf("✅ Includes VS Code Dev Container integration\n")
	}
	
	if len(template.Variables) > 0 {
		fmt.Printf("\n📋 Template Variables:\n")
		for _, variable := range template.Variables {
			defaultStr := ""
			if variable.Default != nil {
				defaultStr = fmt.Sprintf(" (default: %v)", variable.Default)
			}
			requiredStr := ""
			if variable.Required {
				requiredStr = " *required*"
			}
			fmt.Printf("  • %s (%s): %s%s%s\n", variable.Name, variable.Type, variable.Description, defaultStr, requiredStr)
			if variable.Type == "choice" && len(variable.Choices) > 0 {
				fmt.Printf("    Choices: %s\n", strings.Join(variable.Choices, ", "))
			}
		}
	}
	
	fmt.Printf("\n📁 Files (%d):\n", len(template.Files))
	for _, file := range template.Files {
		templateStr := ""
		if file.Template {
			templateStr = " [templated]"
		}
		execStr := ""
		if file.Executable {
			execStr = " [executable]"
		}
		fmt.Printf("  • %s%s%s\n", file.Path, templateStr, execStr)
		
		if showFiles && file.Content != "" {
			lines := strings.Split(file.Content, "\n")
			preview := lines
			if len(lines) > 10 {
				preview = lines[:10]
				fmt.Printf("    Preview (first 10 lines):\n")
			} else {
				fmt.Printf("    Content:\n")
			}
			for _, line := range preview {
				fmt.Printf("      %s\n", line)
			}
			if len(lines) > 10 {
				fmt.Printf("    ... (%d more lines)\n", len(lines)-10)
			}
			fmt.Println()
		}
	}
	
	if len(template.GitIgnore) > 0 {
		fmt.Printf("\n🚫 Git Ignore Patterns (%d):\n", len(template.GitIgnore))
		for _, pattern := range template.GitIgnore {
			if pattern != "" {
				fmt.Printf("  • %s\n", pattern)
			}
		}
	}
	
	if len(template.PostCreate) > 0 {
		fmt.Printf("\n⚡ Post-Creation Commands:\n")
		for _, cmd := range template.PostCreate {
			fmt.Printf("  • %s\n", cmd)
		}
	}
	
	if len(template.Dependencies) > 0 {
		fmt.Printf("\n📦 Dependencies:\n")
		for _, dep := range template.Dependencies {
			fmt.Printf("  • %s\n", dep)
		}
	}
	
	fmt.Printf("\n💡 Create project: claude-reactor template new %s <project-name>\n", template.Name)
	
	return nil
}

// createFromTemplate creates a new project from template
func createFromTemplate(cmd *cobra.Command, args []string, app *pkg.AppContainer) error {
	templateName := args[0]
	projectName := args[1]
	
	projectPath := "."
	if len(args) > 2 {
		projectPath = args[2]
	}
	
	// Get flags
	force, _ := cmd.Flags().GetBool("force")
	noGit, _ := cmd.Flags().GetBool("no-git")
	noDevcontainer, _ := cmd.Flags().GetBool("no-devcontainer")
	vars, _ := cmd.Flags().GetStringSlice("var")
	
	// Parse template variables
	variables := make(map[string]string)
	for _, v := range vars {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 {
			variables[parts[0]] = parts[1]
		} else {
			app.Logger.Warnf("Invalid variable format: %s (expected key=value)", v)
		}
	}
	
	// Build full project path
	fullProjectPath := filepath.Join(projectPath, projectName)
	
	// Check if directory exists
	if _, err := os.Stat(fullProjectPath); err == nil && !force {
		return fmt.Errorf("directory %s already exists (use --force to overwrite)", fullProjectPath)
	}
	
	app.Logger.Infof("🏗️ Creating project '%s' from template '%s'", projectName, templateName)
	
	// Create project from template
	result, err := app.TemplateMgr.ScaffoldProject(templateName, fullProjectPath, projectName, variables)
	if err != nil {
		return fmt.Errorf("failed to scaffold project: %w", err)
	}
	
	// Display results
	fmt.Printf("\n✅ Project '%s' created successfully!\n", projectName)
	fmt.Printf("📁 Location: %s\n", result.ProjectPath)
	fmt.Printf("🎯 Template: %s (%s)\n", result.TemplateName, result.Language)
	fmt.Printf("📦 Variant: %s\n", result.Variant)
	fmt.Printf("📄 Files Created: %d\n", len(result.FilesCreated))
	
	if result.DevContainerGen && !noDevcontainer {
		fmt.Printf("✅ VS Code Dev Container configured\n")
	}
	
	if result.GitInitialized && !noGit {
		fmt.Printf("✅ Git repository initialized\n")
	}
	
	if result.PostCreateRan {
		fmt.Printf("✅ Post-creation commands executed\n")
	}
	
	fmt.Printf("\n🚀 Next Steps:\n")
	fmt.Printf("  1. cd %s\n", projectName)
	if result.Language == "go" || result.Language == "rust" || result.Language == "node" {
		fmt.Printf("  2. Install dependencies (see README.md)\n")
	}
	if result.DevContainerGen {
		fmt.Printf("  3. Open in VS Code: code .\n")
		fmt.Printf("  4. Reopen in Dev Container when prompted\n")
	} else {
		fmt.Printf("  3. Start development with: claude-reactor run\n")
	}
	
	return nil
}

// interactiveInit runs the interactive project creation wizard
func interactiveInit(cmd *cobra.Command, args []string, app *pkg.AppContainer) error {
	projectPath := "."
	if len(args) > 0 {
		projectPath = args[0]
	}
	
	app.Logger.Infof("🧙 Starting interactive project creation wizard...")
	
	result, err := app.TemplateMgr.InteractiveScaffold(projectPath)
	if err != nil {
		return fmt.Errorf("interactive scaffolding failed: %w", err)
	}
	
	// Display results (same as createFromTemplate)
	fmt.Printf("\n✅ Project '%s' created successfully!\n", result.ProjectName)
	fmt.Printf("📁 Location: %s\n", result.ProjectPath)
	fmt.Printf("🎯 Template: %s (%s)\n", result.TemplateName, result.Language)
	fmt.Printf("📦 Variant: %s\n", result.Variant)
	fmt.Printf("📄 Files Created: %d\n", len(result.FilesCreated))
	
	fmt.Printf("\n🚀 Next Steps:\n")
	fmt.Printf("  1. cd %s\n", result.ProjectName)
	if result.DevContainerGen {
		fmt.Printf("  2. Open in VS Code: code .\n")
		fmt.Printf("  3. Reopen in Dev Container when prompted\n")
	} else {
		fmt.Printf("  2. Start development with: claude-reactor run\n")
	}
	
	return nil
}

// createTemplate creates a template from existing project
func createTemplate(cmd *cobra.Command, args []string, app *pkg.AppContainer) error {
	templateName := args[0]
	projectPath := "."
	if len(args) > 1 {
		projectPath = args[1]
	}
	
	description, _ := cmd.Flags().GetString("description")
	author, _ := cmd.Flags().GetString("author")
	tags, _ := cmd.Flags().GetStringSlice("tags")
	
	app.Logger.Infof("🏗️ Creating template '%s' from project at %s", templateName, projectPath)
	
	template, err := app.TemplateMgr.CreateTemplate(projectPath, templateName)
	if err != nil {
		return fmt.Errorf("failed to create template: %w", err)
	}
	
	// Update template metadata if provided
	if description != "" {
		template.Description = description
	}
	if author != "" {
		template.Author = author
	}
	if len(tags) > 0 {
		template.Tags = append(template.Tags, tags...)
	}
	
	fmt.Printf("\n✅ Template '%s' created successfully!\n", template.Name)
	fmt.Printf("📝 Description: %s\n", template.Description)
	fmt.Printf("🔧 Language: %s\n", template.Language)
	fmt.Printf("📦 Variant: %s\n", template.Variant)
	fmt.Printf("📄 Files: %d\n", len(template.Files))
	
	fmt.Printf("\n💡 Use template: claude-reactor template new %s <project-name>\n", template.Name)
	
	return nil
}

// installTemplate installs a template from file or URL
func installTemplate(cmd *cobra.Command, args []string, app *pkg.AppContainer) error {
	source := args[0]
	
	app.Logger.Infof("📦 Installing template from %s", source)
	
	if err := app.TemplateMgr.InstallTemplate(source); err != nil {
		return fmt.Errorf("failed to install template: %w", err)
	}
	
	fmt.Printf("✅ Template installed successfully!\n")
	fmt.Printf("💡 Use 'claude-reactor template list' to see all available templates\n")
	
	return nil
}

// uninstallTemplate removes a template
func uninstallTemplate(cmd *cobra.Command, args []string, app *pkg.AppContainer) error {
	templateName := args[0]
	force, _ := cmd.Flags().GetBool("force")
	
	if !force {
		// Check if template exists
		_, err := app.TemplateMgr.GetTemplate(templateName)
		if err != nil {
			return fmt.Errorf("template '%s' not found", templateName)
		}
		
		fmt.Printf("⚠️  This will permanently remove template '%s'\n", templateName)
		fmt.Print("Are you sure? (y/N): ")
		
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("❌ Cancelled")
			return nil
		}
	}
	
	app.Logger.Infof("🗑️ Removing template '%s'", templateName)
	
	if err := app.TemplateMgr.UninstallTemplate(templateName); err != nil {
		return fmt.Errorf("failed to uninstall template: %w", err)
	}
	
	fmt.Printf("✅ Template '%s' removed successfully!\n", templateName)
	
	return nil
}

// validateTemplate validates a template configuration
func validateTemplate(cmd *cobra.Command, args []string, app *pkg.AppContainer) error {
	templateSource := args[0]
	
	app.Logger.Infof("🔍 Validating template: %s", templateSource)
	
	// Check if it's a template name or file path
	var template *pkg.ProjectTemplate
	var err error
	
	if strings.HasSuffix(templateSource, ".yaml") || strings.HasSuffix(templateSource, ".yml") {
		// Validate file directly
		fmt.Printf("⚠️  Direct file validation not yet implemented\n")
		fmt.Printf("💡 Use: claude-reactor template install %s && claude-reactor template validate <template-name>\n", templateSource)
		return nil
	} else {
		// Get installed template
		template, err = app.TemplateMgr.GetTemplate(templateSource)
		if err != nil {
			return fmt.Errorf("failed to get template: %w", err)
		}
	}
	
	// Validate template
	if err := app.TemplateMgr.ValidateTemplate(template); err != nil {
		fmt.Printf("❌ Template validation failed:\n")
		fmt.Printf("   %v\n", err)
		return nil
	}
	
	fmt.Printf("✅ Template '%s' is valid!\n", template.Name)
	fmt.Printf("📝 Description: %s\n", template.Description)
	fmt.Printf("🔧 Language: %s\n", template.Language)
	fmt.Printf("📦 Variant: %s\n", template.Variant)
	fmt.Printf("📄 Files: %d\n", len(template.Files))
	if len(template.Variables) > 0 {
		fmt.Printf("🔧 Variables: %d\n", len(template.Variables))
	}
	
	return nil
}

// newDependencyCmd creates the dependency management command tree
func newDependencyCmd(app *pkg.AppContainer) *cobra.Command {
	var dependencyCmd = &cobra.Command{
		Use:   "dependency",
		Short: "Dependency management and package manager operations",
		Long: `Manage dependencies across all supported package managers.

Supports unified operations for:
• Go Modules (go.mod, go.sum)  
• Cargo (Cargo.toml, Cargo.lock)
• npm (package.json, package-lock.json)
• Yarn (package.json, yarn.lock)
• pnpm (package.json, pnpm-lock.yaml)
• pip (requirements.txt, setup.py, pyproject.toml)
• Poetry (pyproject.toml, poetry.lock)
• Pipenv (Pipfile, Pipfile.lock)
• Maven (pom.xml)
• Gradle (build.gradle, build.gradle.kts)

Examples:
  claude-reactor dependency detect     # Detect package managers in project
  claude-reactor dependency list      # List all dependencies
  claude-reactor dependency install   # Install dependencies for all package managers
  claude-reactor dependency update    # Update dependencies to latest versions
  claude-reactor dependency audit     # Scan for security vulnerabilities
  claude-reactor dependency outdated  # Check for outdated dependencies
  claude-reactor dependency report    # Generate comprehensive dependency report`,
		Aliases: []string{"deps", "dep"},
	}

	// Add subcommands
	dependencyCmd.AddCommand(
		newDependencyDetectCmd(app),
		newDependencyListCmd(app),
		newDependencyInstallCmd(app),
		newDependencyUpdateCmd(app),
		newDependencyAuditCmd(app),
		newDependencyOutdatedCmd(app),
		newDependencyReportCmd(app),
		newDependencyCleanCmd(app),
	)

	return dependencyCmd
}

// newDependencyDetectCmd detects package managers in the current project
func newDependencyDetectCmd(app *pkg.AppContainer) *cobra.Command {
	return &cobra.Command{
		Use:   "detect [path]",
		Short: "Detect package managers in project",
		Long: `Detect package managers and their configuration files in the current or specified project directory.

Shows available package managers on the system and detected package managers in the project.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectPath := "."
			if len(args) > 0 {
				projectPath = args[0]
			}

			abs, err := filepath.Abs(projectPath)
			if err != nil {
				return fmt.Errorf("failed to resolve project path: %w", err)
			}

			app.Logger.Infof("🔍 Detecting package managers in: %s", abs)
			fmt.Printf("📦 Detecting package managers in: %s\n\n", abs)

			// Detect project package managers
			packageManagers, dependencies, err := app.DependencyMgr.DetectProjectDependencies(projectPath)
			if err != nil {
				return fmt.Errorf("failed to detect dependencies: %w", err)
			}

			if len(packageManagers) == 0 {
				fmt.Printf("❌ No package managers detected in project\n\n")
			} else {
				fmt.Printf("✅ Found %d package manager(s):\n", len(packageManagers))
				for _, pm := range packageManagers {
					status := "❌ not available"
					if pm.Available {
						status = "✅ available"
					}
					
					version := ""
					if pm.Version != "" {
						version = fmt.Sprintf(" (v%s)", pm.Version)
					}
					
					fmt.Printf("  • %s%s - %s\n", pm.Name, version, status)
					if len(pm.ConfigFiles) > 0 {
						fmt.Printf("    Config files: %s\n", strings.Join(pm.ConfigFiles, ", "))
					}
					if len(pm.LockFiles) > 0 {
						fmt.Printf("    Lock files: %s\n", strings.Join(pm.LockFiles, ", "))
					}
				}
				fmt.Printf("\n📊 Total dependencies detected: %d\n", len(dependencies))
			}

			return nil
		},
	}
}

// newDependencyListCmd lists all dependencies in the project
func newDependencyListCmd(app *pkg.AppContainer) *cobra.Command {
	var showDetails bool
	var packageManager string
	
	cmd := &cobra.Command{
		Use:   "list [path]",
		Short: "List all project dependencies",
		Long: `List dependencies from all detected package managers in the project.

Shows dependency name, version, type (direct/indirect), and package manager.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectPath := "."
			if len(args) > 0 {
				projectPath = args[0]
			}

			app.Logger.Infof("📋 Listing dependencies in: %s", projectPath)

			packageManagers, dependencies, err := app.DependencyMgr.DetectProjectDependencies(projectPath)
			if err != nil {
				return fmt.Errorf("failed to detect dependencies: %w", err)
			}

			if len(packageManagers) == 0 {
				fmt.Printf("❌ No package managers detected\n")
				return nil
			}

			// Filter by package manager if specified
			if packageManager != "" {
				var filtered []*pkg.DependencyInfo
				for _, dep := range dependencies {
					if dep.PackageManager == packageManager {
						filtered = append(filtered, dep)
					}
				}
				dependencies = filtered
			}

			if len(dependencies) == 0 {
				fmt.Printf("❌ No dependencies found\n")
				return nil
			}

			fmt.Printf("📋 Found %d dependencies across %d package managers:\n\n", len(dependencies), len(packageManagers))

			// Group by package manager
			depsByPM := make(map[string][]*pkg.DependencyInfo)
			for _, dep := range dependencies {
				depsByPM[dep.PackageManager] = append(depsByPM[dep.PackageManager], dep)
			}

			for pmType, deps := range depsByPM {
				if len(deps) == 0 {
					continue
				}
				
				fmt.Printf("🔧 %s (%d dependencies):\n", strings.ToUpper(pmType), len(deps))
				for _, dep := range deps {
					typeIcon := "📦"
					if dep.Type == "dev" {
						typeIcon = "🛠️"
					} else if dep.Type == "indirect" {
						typeIcon = "🔗"
					}
					
					fmt.Printf("  %s %s@%s", typeIcon, dep.Name, dep.CurrentVersion)
					
					if showDetails {
						if dep.Description != "" {
							fmt.Printf(" - %s", dep.Description)
						}
						if dep.License != "" {
							fmt.Printf(" [%s]", dep.License)
						}
						if dep.IsOutdated {
							fmt.Printf(" (outdated)")
						}
						if dep.HasVulnerability {
							fmt.Printf(" ⚠️ vulnerable")
						}
					}
					fmt.Printf("\n")
				}
				fmt.Printf("\n")
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&showDetails, "details", "d", false, "Show detailed dependency information")
	cmd.Flags().StringVarP(&packageManager, "manager", "m", "", "Filter by package manager (go, npm, cargo, etc.)")

	return cmd
}

// newDependencyInstallCmd installs dependencies for all package managers
func newDependencyInstallCmd(app *pkg.AppContainer) *cobra.Command {
	return &cobra.Command{
		Use:   "install [path]",
		Short: "Install dependencies for all package managers",
		Long: `Install dependencies for all detected package managers in the project.

Runs the appropriate install command for each package manager found.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectPath := "."
			if len(args) > 0 {
				projectPath = args[0]
			}

			app.Logger.Infof("📦 Installing dependencies in: %s", projectPath)
			fmt.Printf("📦 Installing dependencies in: %s\n\n", projectPath)

			results, err := app.DependencyMgr.InstallAllDependencies(projectPath)
			if err != nil {
				return fmt.Errorf("failed to install dependencies: %w", err)
			}

			if len(results) == 0 {
				fmt.Printf("❌ No package managers found\n")
				return nil
			}

			successCount := 0
			for _, result := range results {
				if result.Success {
					fmt.Printf("✅ %s: installed successfully (%s)\n", result.PackageManager, result.Duration)
					successCount++
				} else {
					fmt.Printf("❌ %s: failed - %s\n", result.PackageManager, result.Error)
				}
			}

			fmt.Printf("\n📊 Summary: %d/%d package managers installed successfully\n", successCount, len(results))

			return nil
		},
	}
}

// newDependencyUpdateCmd updates dependencies for detected package managers
func newDependencyUpdateCmd(app *pkg.AppContainer) *cobra.Command {
	return &cobra.Command{
		Use:   "update [path]",
		Short: "Update dependencies for detected package managers",
		Long: `Update dependencies for all detected package managers in the current or specified project directory.
		
This will run the appropriate update command for each detected package manager (npm update, cargo update, etc.).`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectPath := "."
			if len(args) > 0 {
				projectPath = args[0]
			}

			abs, err := filepath.Abs(projectPath)
			if err != nil {
				return fmt.Errorf("failed to resolve project path: %w", err)
			}

			app.Logger.Infof("⬆️ Updating dependencies in: %s", abs)
			fmt.Printf("⬆️ Updating dependencies in: %s\n\n", abs)

			// Update dependencies for all detected package managers
			results, err := app.DependencyMgr.UpdateAllDependencies(projectPath)
			if err != nil {
				return fmt.Errorf("failed to update dependencies: %w", err)
			}

			if len(results) == 0 {
				fmt.Printf("❌ No package managers found to update\n")
				return nil
			}

			// Display results
			successCount := 0
			for _, result := range results {
				if result.Success {
					successCount++
					fmt.Printf("✅ %s: updated successfully (took %s)\n", result.PackageManager, result.Duration)
				} else {
					fmt.Printf("❌ %s: failed - %s\n", result.PackageManager, result.Error)
				}
			}

			fmt.Printf("\n📊 Summary: %d/%d package managers updated successfully\n", successCount, len(results))

			return nil
		},
	}
}

// newDependencyAuditCmd audits dependencies for vulnerabilities
func newDependencyAuditCmd(app *pkg.AppContainer) *cobra.Command {
	return &cobra.Command{
		Use:   "audit [path]",
		Short: "Audit dependencies for security vulnerabilities",
		Long: `Audit dependencies for security vulnerabilities using appropriate tools for each package manager.
		
This will run security audits using tools like npm audit, cargo-audit, pip-audit, etc.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectPath := "."
			if len(args) > 0 {
				projectPath = args[0]
			}

			abs, err := filepath.Abs(projectPath)
			if err != nil {
				return fmt.Errorf("failed to resolve project path: %w", err)
			}

			app.Logger.Infof("🔍 Auditing dependencies in: %s", abs)
			fmt.Printf("🔍 Auditing dependencies for vulnerabilities in: %s\n\n", abs)

			// Audit dependencies for all detected package managers
			auditResults, err := app.DependencyMgr.AuditAllDependencies(projectPath)
			if err != nil {
				return fmt.Errorf("failed to audit dependencies: %w", err)
			}

			if len(auditResults) == 0 {
				fmt.Printf("✅ No vulnerabilities found or no compatible audit tools available\n")
				return nil
			}

			// Display vulnerability results
			fmt.Printf("🚨 Found %d vulnerabilities:\n\n", len(auditResults))

			criticalCount := 0
			highCount := 0
			moderateCount := 0
			lowCount := 0

			for _, vuln := range auditResults {
				severityIcon := "⚠️"
				switch strings.ToLower(vuln.Severity) {
				case "critical":
					severityIcon = "🔴"
					criticalCount++
				case "high":
					severityIcon = "🟠"
					highCount++
				case "moderate":
					severityIcon = "🟡"
					moderateCount++
				case "low":
					severityIcon = "🟢"
					lowCount++
				}

				fmt.Printf("%s %s - %s\n", severityIcon, vuln.ID, vuln.Title)
				fmt.Printf("   Description: %s\n", vuln.Description)
				if vuln.FixedIn != "" {
					fmt.Printf("   Fixed in: %s\n", vuln.FixedIn)
				}
				if vuln.Reference != "" {
					fmt.Printf("   Reference: %s\n", vuln.Reference)
				}
				fmt.Println()
			}

			fmt.Printf("📊 Vulnerability Summary:\n")
			fmt.Printf("   🔴 Critical: %d\n", criticalCount)
			fmt.Printf("   🟠 High: %d\n", highCount)
			fmt.Printf("   🟡 Moderate: %d\n", moderateCount)
			fmt.Printf("   🟢 Low: %d\n", lowCount)

			return nil
		},
	}
}

// newDependencyOutdatedCmd shows outdated dependencies
func newDependencyOutdatedCmd(app *pkg.AppContainer) *cobra.Command {
	return &cobra.Command{
		Use:   "outdated [path]",
		Short: "Show outdated dependencies",
		Long: `Show outdated dependencies for all detected package managers in the current or specified project directory.
		
This will check for available updates using tools like npm outdated, cargo-outdated, pip list --outdated, etc.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectPath := "."
			if len(args) > 0 {
				projectPath = args[0]
			}

			abs, err := filepath.Abs(projectPath)
			if err != nil {
				return fmt.Errorf("failed to resolve project path: %w", err)
			}

			app.Logger.Infof("📊 Checking outdated dependencies in: %s", abs)
			fmt.Printf("📊 Checking outdated dependencies in: %s\n\n", abs)

			// Get outdated dependencies for all detected package managers
			outdatedDeps, err := app.DependencyMgr.GetAllOutdatedDependencies(projectPath)
			if err != nil {
				return fmt.Errorf("failed to check outdated dependencies: %w", err)
			}

			if len(outdatedDeps) == 0 {
				fmt.Printf("✅ All dependencies are up to date!\n")
				return nil
			}

			// Group by package manager
			pmGroups := make(map[string][]*pkg.DependencyInfo)
			for _, dep := range outdatedDeps {
				pmGroups[dep.PackageManager] = append(pmGroups[dep.PackageManager], dep)
			}

			fmt.Printf("📋 Found %d outdated dependencies:\n\n", len(outdatedDeps))

			for pmType, deps := range pmGroups {
				fmt.Printf("📦 %s:\n", strings.ToUpper(pmType))
				for _, dep := range deps {
					fmt.Printf("   %s: %s → %s", dep.Name, dep.CurrentVersion, dep.LatestVersion)
					if dep.RequestedVersion != "" && dep.RequestedVersion != dep.CurrentVersion {
						fmt.Printf(" (requested: %s)", dep.RequestedVersion)
					}
					fmt.Println()
				}
				fmt.Println()
			}

			return nil
		},
	}
}

// newDependencyReportCmd generates comprehensive dependency reports
func newDependencyReportCmd(app *pkg.AppContainer) *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "report [path]",
		Short: "Generate comprehensive dependency report",
		Long: `Generate a comprehensive dependency report including dependency trees, vulnerability analysis,
outdated packages, and package manager health for the current or specified project directory.

Output formats:
  - text: Human-readable text format (default)
  - json: Machine-readable JSON format`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectPath := "."
			if len(args) > 0 {
				projectPath = args[0]
			}

			abs, err := filepath.Abs(projectPath)
			if err != nil {
				return fmt.Errorf("failed to resolve project path: %w", err)
			}

			app.Logger.Infof("📊 Generating dependency report for: %s", abs)

			// Generate comprehensive report
			report, err := app.DependencyMgr.GenerateDependencyReport(projectPath)
			if err != nil {
				return fmt.Errorf("failed to generate dependency report: %w", err)
			}

			// Output in requested format
			if outputFormat == "json" {
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")
				return encoder.Encode(report)
			}

			// Text format output
			fmt.Printf("📊 Dependency Report for: %s\n", abs)
			fmt.Printf("Generated: %s\n\n", report.GeneratedAt)

			// Summary
			fmt.Printf("📋 Summary:\n")
			fmt.Printf("   Total Dependencies: %d\n", report.TotalDependencies)
			fmt.Printf("   Direct Dependencies: %d\n", report.DirectDependencies)
			fmt.Printf("   Indirect Dependencies: %d\n", report.IndirectDependencies)
			fmt.Printf("   Outdated Dependencies: %d\n", report.OutdatedDependencies)
			fmt.Printf("   Vulnerabilities: %d\n", report.Vulnerabilities)
			fmt.Printf("   Security Score: %.1f/100\n\n", report.SecurityScore)

			// Package Managers
			fmt.Printf("📦 Detected Package Managers:\n")
			for _, pm := range report.PackageManagers {
				statusIcon := "✅"
				if !pm.Available {
					statusIcon = "❌"
				}
				fmt.Printf("   %s %s (%s)\n", statusIcon, pm.Name, pm.Version)
				for _, configFile := range pm.ConfigFiles {
					fmt.Printf("     📄 %s\n", configFile)
				}
			}

			if report.Vulnerabilities > 0 {
				fmt.Printf("\n🚨 Security Vulnerabilities: %d found\n", report.Vulnerabilities)
				fmt.Printf("   See 'dependency audit' for detailed vulnerability information\n")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "format", "f", "text", "Output format (text|json)")

	return cmd
}

// newDependencyCleanCmd cleans package manager caches
func newDependencyCleanCmd(app *pkg.AppContainer) *cobra.Command {
	return &cobra.Command{
		Use:   "clean [path]",
		Short: "Clean package manager caches",
		Long: `Clean caches for all detected package managers in the current or specified project directory.
		
This will run cache cleaning commands like npm cache clean, cargo clean, pip cache purge, etc.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectPath := "."
			if len(args) > 0 {
				projectPath = args[0]
			}

			abs, err := filepath.Abs(projectPath)
			if err != nil {
				return fmt.Errorf("failed to resolve project path: %w", err)
			}

			app.Logger.Infof("🧹 Cleaning package manager caches in: %s", abs)
			fmt.Printf("🧹 Cleaning package manager caches in: %s\n\n", abs)

			// Clean caches for all detected package managers
			results, err := app.DependencyMgr.CleanAllCaches(projectPath)
			if err != nil {
				return fmt.Errorf("failed to clean caches: %w", err)
			}

			if len(results) == 0 {
				fmt.Printf("❌ No package managers found to clean\n")
				return nil
			}

			// Display results
			successCount := 0
			for _, result := range results {
				if result.Success {
					successCount++
					fmt.Printf("✅ %s: cache cleaned successfully (took %s)\n", result.PackageManager, result.Duration)
				} else {
					fmt.Printf("❌ %s: failed - %s\n", result.PackageManager, result.Error)
				}
			}

			fmt.Printf("\n📊 Summary: %d/%d package managers cleaned successfully\n", successCount, len(results))

			return nil
		},
	}
}

// newHotReloadCmd creates the hot reload command with subcommands
func newHotReloadCmd(app *pkg.AppContainer) *cobra.Command {
	var hotReloadCmd = &cobra.Command{
		Use:   "hotreload",
		Short: "Hot reload functionality for faster development cycles",
		Long: `Hot reload provides automatic file watching, building, and container synchronization
for faster development cycles. Changes to your project files are automatically detected,
built (if needed), and synchronized with the running container.

This feature supports multiple project types including Go, Node.js, Python, Rust, and Java.`,
	}

	// Command flags
	var (
		hotReloadContainerFlag string
		hotReloadWatchPatternsFlag []string
		hotReloadIgnorePatternsFlag []string
		hotReloadDebounceFlag int
		hotReloadDisableBuildFlag bool
		hotReloadDisableSyncFlag bool
		hotReloadVerboseFlag bool
	)

	// Start command
	var hotReloadStartCmd = &cobra.Command{
		Use:   "start [project-path]",
		Short: "Start hot reload for a project",
		Long: `Start hot reload monitoring for a project. This will:

1. Auto-detect the project type (Go, Node.js, Python, etc.)
2. Set up file watching with appropriate patterns
3. Configure build triggers for the detected language
4. Start container synchronization for fast file updates

If no project path is specified, the current directory is used.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			// Determine project path
			projectPath := "."
			if len(args) > 0 {
				projectPath = args[0]
			}
			
			// Get absolute project path
			if !strings.HasPrefix(projectPath, "/") {
				if projectPath == "." {
					cwd, err := os.Getwd()
					if err != nil {
						return fmt.Errorf("failed to get current directory: %w", err)
					}
					projectPath = cwd
				} else {
					cwd, err := os.Getwd()
					if err != nil {
						return fmt.Errorf("failed to get current directory: %w", err)
					}
					projectPath = cwd + "/" + strings.TrimPrefix(projectPath, "./")
				}
			}

			fmt.Printf("🔥 Starting hot reload for project: %s\n", projectPath)

			// Determine container ID
			containerID := hotReloadContainerFlag
			if containerID == "" {
				// Auto-detect running container for this project
				projectHash := app.DockerMgr.GenerateProjectHash(projectPath)
				
				// Get current configuration
				config, err := app.ConfigMgr.LoadConfig()
				if err != nil {
					// Use default account if no config
					config = &pkg.Config{Account: "default"}
				}
				
				// Generate expected container name
				variant := config.Variant
				if variant == "" {
					variant = "base"
				}
				
				architecture, _ := app.ArchDetector.GetHostArchitecture()
				containerName := fmt.Sprintf("claude-reactor-v2-%s-%s-%s-%s", variant, architecture, projectHash, config.Account)
				
				// Check if container is running
				running, err := app.DockerMgr.IsContainerRunning(ctx, containerName)
				if err != nil {
					return fmt.Errorf("failed to check container status: %w", err)
				}
				
				if !running {
					return fmt.Errorf("no running container found for project. Please start a container first or specify --container")
				}
				
				// Get container status to get ID
				status, err := app.DockerMgr.GetContainerStatus(ctx, containerName)
				if err != nil {
					return fmt.Errorf("failed to get container status: %w", err)
				}
				
				containerID = status.ID
				fmt.Printf("🔍 Auto-detected container: %s\n", containerID[:12])
			}

			// Create hot reload options
			options := &pkg.HotReloadOptions{
				AutoDetect:          true,
				EnableNotifications: true,
			}

			// Configure watch patterns if specified
			if len(hotReloadWatchPatternsFlag) > 0 || len(hotReloadIgnorePatternsFlag) > 0 || hotReloadDebounceFlag != 500 {
				options.WatchConfig = &pkg.WatchConfig{
					IncludePatterns: hotReloadWatchPatternsFlag,
					ExcludePatterns: hotReloadIgnorePatternsFlag,
					DebounceDelay:   hotReloadDebounceFlag,
					Recursive:       true,
					EnableBuild:     !hotReloadDisableBuildFlag,
					EnableHotReload: !hotReloadDisableSyncFlag,
					ContainerName:   containerID,
				}
			}

			// Start hot reload
			session, err := app.HotReloadMgr.StartHotReload(projectPath, containerID, options)
			if err != nil {
				return fmt.Errorf("failed to start hot reload: %w", err)
			}

			fmt.Printf("✅ Hot reload started successfully\n")
			fmt.Printf("📋 Session ID: %s\n", session.ID)
			if session.ProjectInfo != nil {
				fmt.Printf("📁 Project Type: %s (%s) - %.1f%% confidence\n", 
					session.ProjectInfo.Type, session.ProjectInfo.Framework, session.ProjectInfo.Confidence)
			}
			fmt.Printf("🔍 Watching: %s\n", projectPath)
			fmt.Printf("📦 Container: %s\n", containerID[:12])

			if hotReloadVerboseFlag {
				fmt.Print("\n📊 Session Details:\n")
				fmt.Printf("   Start Time: %s\n", session.StartTime)
				fmt.Printf("   Status: %s\n", session.Status)
				if session.WatchSession != nil {
					fmt.Printf("   Watch Session: %s\n", session.WatchSession.ID)
				}
				if session.SyncSession != nil {
					fmt.Printf("   Sync Session: %s\n", session.SyncSession.ID)
				}
			}

			fmt.Print("\n💡 Use 'claude-reactor hotreload status' to monitor progress\n")
			fmt.Printf("💡 Use 'claude-reactor hotreload stop %s' to stop\n", session.ID)
			
			return nil
		},
	}

	// Stop command
	var hotReloadStopCmd = &cobra.Command{
		Use:   "stop <session-id>",
		Short: "Stop an active hot reload session",
		Long: `Stop an active hot reload session by its ID. This will:

1. Stop file watching
2. Stop build triggers
3. Stop container synchronization
4. Clean up session resources

Use 'hotreload list' to see active sessions and their IDs.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := args[0]
			
			fmt.Printf("🛑 Stopping hot reload session: %s\n", sessionID)

			// Stop hot reload
			err := app.HotReloadMgr.StopHotReload(sessionID)
			if err != nil {
				return fmt.Errorf("failed to stop hot reload: %w", err)
			}

			fmt.Printf("✅ Hot reload session stopped: %s\n", sessionID)
			return nil
		},
	}

	// Status command
	var hotReloadStatusCmd = &cobra.Command{
		Use:   "status [session-id]",
		Short: "Show hot reload status",
		Long: `Show detailed status information for hot reload sessions.

If a session ID is provided, shows detailed status for that specific session.
If no session ID is provided, shows a summary of all active sessions.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				// Show detailed status for specific session
				sessionID := args[0]
				status, err := app.HotReloadMgr.GetHotReloadStatus(sessionID)
				if err != nil {
					return fmt.Errorf("failed to get hot reload status: %w", err)
				}

				fmt.Print("🔥 Hot Reload Session Status\n")
				fmt.Printf("📋 Session ID: %s\n", status.SessionID)
				fmt.Printf("📊 Status: %s\n", status.Status)
				fmt.Printf("👀 Watching: %s\n", status.WatchingStatus)
				fmt.Printf("🔨 Build: %s\n", status.BuildStatus)
				fmt.Printf("🔄 Sync: %s\n", status.SyncStatus)
				fmt.Printf("⚡ Hot Reload: %s\n", status.HotReloadStatus)

				if status.Metrics != nil {
					fmt.Print("\n📈 Metrics:\n")
					fmt.Printf("   Uptime: %s\n", status.Metrics.Uptime)
					fmt.Printf("   Total Changes: %d\n", status.Metrics.TotalChanges)
					fmt.Printf("   Build Success Rate: %.1f%%\n", status.Metrics.BuildSuccessRate)
					fmt.Printf("   Average Build Time: %s\n", status.Metrics.AverageBuildTime)
					fmt.Printf("   Average Sync Time: %s\n", status.Metrics.AverageSyncTime)
				}

				if len(status.RecentActivity) > 0 {
					fmt.Print("\n📋 Recent Activity:\n")
					for _, activity := range status.RecentActivity {
						timestamp, _ := time.Parse(time.RFC3339, activity.Timestamp)
						fmt.Printf("   %s [%s] %s\n", 
							timestamp.Format("15:04:05"), 
							strings.ToUpper(activity.Level), 
							activity.Message)
					}
				}
			} else {
				// Show summary of all sessions
				sessions, err := app.HotReloadMgr.GetHotReloadSessions()
				if err != nil {
					return fmt.Errorf("failed to get hot reload sessions: %w", err)
				}

				if len(sessions) == 0 {
					fmt.Print("📭 No active hot reload sessions\n")
					fmt.Print("💡 Use 'claude-reactor hotreload start' to begin hot reloading\n")
					return nil
				}

				fmt.Printf("🔥 Active Hot Reload Sessions (%d)\n\n", len(sessions))

				for i, session := range sessions {
					fmt.Printf("%d. Session: %s\n", i+1, session.ID)
					fmt.Printf("   📁 Project: %s\n", session.ProjectPath)
					if session.ProjectInfo != nil {
						fmt.Printf("   🏷️  Type: %s (%s)\n", session.ProjectInfo.Type, session.ProjectInfo.Framework)
					}
					fmt.Printf("   📦 Container: %s\n", session.ContainerID[:12])
					fmt.Printf("   📊 Status: %s\n", session.Status)
					
					startTime, _ := time.Parse(time.RFC3339, session.StartTime)
					uptime := time.Since(startTime)
					fmt.Printf("   ⏱️  Uptime: %s\n", formatDuration(uptime))
					
					if session.LastActivity != "" {
						lastActivity, _ := time.Parse(time.RFC3339, session.LastActivity)
						fmt.Printf("   🕐 Last Activity: %s ago\n", time.Since(lastActivity).Truncate(time.Second))
					}
					
					if i < len(sessions)-1 {
						fmt.Print("\n")
					}
				}

				fmt.Print("\n💡 Use 'hotreload status <session-id>' for detailed information\n")
			}

			return nil
		},
	}

	// List command (alias for status with no args)
	var hotReloadListCmd = &cobra.Command{
		Use:   "list",
		Short: "List all hot reload sessions",
		Long: `List all active hot reload sessions with their IDs, project paths,
container information, and current status.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return hotReloadStatusCmd.RunE(cmd, []string{}) // Reuse the status command logic
		},
	}

	// Config command
	var hotReloadConfigCmd = &cobra.Command{
		Use:   "config <session-id>",
		Short: "Update hot reload configuration",
		Long: `Update the configuration for an active hot reload session.

This allows you to modify watching patterns, build settings, and sync options
without stopping and restarting the session.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := args[0]
			
			// Check if any config flags were provided
			hasConfigChanges := len(hotReloadWatchPatternsFlag) > 0 || 
				len(hotReloadIgnorePatternsFlag) > 0 || 
				hotReloadDebounceFlag != -1 ||
				hotReloadDisableBuildFlag ||
				hotReloadDisableSyncFlag

			if !hasConfigChanges {
				return fmt.Errorf("no configuration changes specified. Use --watch, --ignore, --debounce, --no-build, or --no-sync flags")
			}

			// Create new options with the specified changes
			options := &pkg.HotReloadOptions{
				AutoDetect:          true,
				EnableNotifications: true,
			}

			// Configure watch patterns
			if len(hotReloadWatchPatternsFlag) > 0 || len(hotReloadIgnorePatternsFlag) > 0 || hotReloadDebounceFlag != -1 {
				options.WatchConfig = &pkg.WatchConfig{
					Recursive:       true,
					EnableBuild:     !hotReloadDisableBuildFlag,
					EnableHotReload: !hotReloadDisableSyncFlag,
				}
				
				if len(hotReloadWatchPatternsFlag) > 0 {
					options.WatchConfig.IncludePatterns = hotReloadWatchPatternsFlag
				}
				
				if len(hotReloadIgnorePatternsFlag) > 0 {
					options.WatchConfig.ExcludePatterns = hotReloadIgnorePatternsFlag
				}
				
				if hotReloadDebounceFlag != -1 {
					options.WatchConfig.DebounceDelay = hotReloadDebounceFlag
				}
			}

			fmt.Printf("🔧 Updating hot reload configuration for session: %s\n", sessionID)

			// Update configuration
			err := app.HotReloadMgr.UpdateHotReloadConfig(sessionID, options)
			if err != nil {
				return fmt.Errorf("failed to update hot reload configuration: %w", err)
			}

			fmt.Printf("✅ Hot reload configuration updated for session: %s\n", sessionID)

			// Show what was updated
			if len(hotReloadWatchPatternsFlag) > 0 {
				fmt.Printf("👀 Updated watch patterns: %v\n", hotReloadWatchPatternsFlag)
			}
			if len(hotReloadIgnorePatternsFlag) > 0 {
				fmt.Printf("🚫 Updated ignore patterns: %v\n", hotReloadIgnorePatternsFlag)
			}
			if hotReloadDebounceFlag != -1 {
				fmt.Printf("⏱️  Updated debounce delay: %dms\n", hotReloadDebounceFlag)
			}
			if hotReloadDisableBuildFlag {
				fmt.Print("🔨 Disabled automatic building\n")
			}
			if hotReloadDisableSyncFlag {
				fmt.Print("🔄 Disabled file synchronization\n")
			}

			return nil
		},
	}

	// Add flags to start command
	hotReloadStartCmd.Flags().StringVarP(&hotReloadContainerFlag, "container", "c", "", "Target container name or ID (auto-detected if not specified)")
	hotReloadStartCmd.Flags().StringSliceVar(&hotReloadWatchPatternsFlag, "watch", nil, "File patterns to watch (e.g., '**/*.go', '*.js')")
	hotReloadStartCmd.Flags().StringSliceVar(&hotReloadIgnorePatternsFlag, "ignore", nil, "File patterns to ignore (e.g., 'node_modules/', '*.tmp')")
	hotReloadStartCmd.Flags().IntVar(&hotReloadDebounceFlag, "debounce", 500, "Debounce delay in milliseconds")
	hotReloadStartCmd.Flags().BoolVar(&hotReloadDisableBuildFlag, "no-build", false, "Disable automatic building")
	hotReloadStartCmd.Flags().BoolVar(&hotReloadDisableSyncFlag, "no-sync", false, "Disable file synchronization")
	hotReloadStartCmd.Flags().BoolVarP(&hotReloadVerboseFlag, "verbose", "v", false, "Verbose output")

	// Add flags to config command
	hotReloadConfigCmd.Flags().StringSliceVar(&hotReloadWatchPatternsFlag, "watch", nil, "Update file patterns to watch")
	hotReloadConfigCmd.Flags().StringSliceVar(&hotReloadIgnorePatternsFlag, "ignore", nil, "Update file patterns to ignore")
	hotReloadConfigCmd.Flags().IntVar(&hotReloadDebounceFlag, "debounce", -1, "Update debounce delay in milliseconds")
	hotReloadConfigCmd.Flags().BoolVar(&hotReloadDisableBuildFlag, "no-build", false, "Disable automatic building")
	hotReloadConfigCmd.Flags().BoolVar(&hotReloadDisableSyncFlag, "no-sync", false, "Disable file synchronization")

	// Add subcommands
	hotReloadCmd.AddCommand(
		hotReloadStartCmd,
		hotReloadStopCmd,
		hotReloadStatusCmd,
		hotReloadListCmd,
		hotReloadConfigCmd,
	)

	return hotReloadCmd
}

// Helper function for formatting duration
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	} else {
		days := int(d.Hours()) / 24
		hours := int(d.Hours()) % 24
		return fmt.Sprintf("%dd %dh", days, hours)
	}
}

// newCompletionCmd creates the completion command with installation instructions
func newCompletionCmd(app *pkg.AppContainer) *cobra.Command {
	completionCmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate completion scripts for your shell",
		Long: `Generate completion scripts for claude-reactor commands.

To load completions:

Bash:
  $ source <(claude-reactor completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ claude-reactor completion bash > /etc/bash_completion.d/claude-reactor
  # macOS:
  $ claude-reactor completion bash > /usr/local/etc/bash_completion.d/claude-reactor

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ claude-reactor completion zsh > "${fpath[1]}/_claude-reactor"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ claude-reactor completion fish | source

  # To load completions for each session, execute once:
  $ claude-reactor completion fish > ~/.config/fish/completions/claude-reactor.fish

PowerShell:
  PS> claude-reactor completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> claude-reactor completion powershell > claude-reactor.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.ExactValidArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell type %q", args[0])
			}
		},
	}

	return completionCmd
}

