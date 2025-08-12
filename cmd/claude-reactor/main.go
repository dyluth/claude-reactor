package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"claude-reactor/internal"
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
	app, err := internal.NewAppContainer()
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
			// Handle installation flags (Phase 0.2)
			if install, _ := cmd.Flags().GetBool("install"); install {
				if err := handleInstallation(cmd, app, true); err != nil {
					cmd.PrintErrf("Installation failed: %v\n", err)
					os.Exit(1)
				}
				return
			}
			
			if uninstall, _ := cmd.Flags().GetBool("uninstall"); uninstall {
				if err := handleInstallation(cmd, app, false); err != nil {
					cmd.PrintErrf("Uninstallation failed: %v\n", err)
					os.Exit(1)
				}
				return
			}
			
			// Handle legacy flags for backward compatibility
			if listVariants, _ := cmd.Flags().GetBool("list-variants"); listVariants {
				handleLegacyListVariants(cmd, app)
				return
			}
			
			// Check for variant flag and validate it
			variant, _ := cmd.Flags().GetString("variant")
			if variant != "" {
				// Validate the variant
				variants := []string{"base", "go", "full", "cloud", "k8s"}
				validVariant := false
				for _, v := range variants {
					if v == variant {
						validVariant = true
						break
					}
				}
				if !validVariant {
					cmd.PrintErrf("Error: invalid variant '%s'. Available variants: %s\n", 
						variant, strings.Join(variants, ", "))
					os.Exit(1)
				}
				
				// Save the variant to configuration (for backward compatibility)
				config := &pkg.Config{
					Variant:     variant,
					Account:     "",
					DangerMode:  false,
					ProjectPath: "",
					Metadata:    make(map[string]string),
				}
				_ = app.ConfigMgr.SaveConfig(config)
			}
			
			if showConfig, _ := cmd.Flags().GetBool("show-config"); showConfig {
				handleLegacyShowConfig(cmd, app)
				return
			}
			
			// Default action - show help or run with default config
			cmd.Help()
		},
	}

	// Global flags
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().String("log-level", "info", "Set log level (debug, info, warn, error)")
	
	// Legacy compatibility flags (hidden from help)
	rootCmd.Flags().Bool("list-variants", false, "List available container variants")
	rootCmd.Flags().Bool("show-config", false, "Show current configuration")
	rootCmd.Flags().String("variant", "", "Container variant (for compatibility)")
	rootCmd.Flags().MarkHidden("list-variants")
	rootCmd.Flags().MarkHidden("show-config")
	rootCmd.Flags().MarkHidden("variant")
	
	// Installation flags (Phase 0.2)
	rootCmd.Flags().Bool("install", false, "Install claude-reactor to system PATH (/usr/local/bin)")
	rootCmd.Flags().Bool("uninstall", false, "Remove claude-reactor from system PATH")

	// Add subcommands
	rootCmd.AddCommand(
		newRunCmd(app),
		newBuildCmd(app),
		newConfigCmd(app),
		newCleanCmd(app),
		newDevContainerCmd(app),
		newDebugCmd(app),
	)

	return rootCmd
}

func newRunCmd(app *pkg.AppContainer) *cobra.Command {
	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "Start and connect to a Claude CLI container",
		Long: `Start and connect to a Claude CLI container with the specified variant.
This will auto-detect project type, build the container if needed, and connect you
to the Claude CLI running inside the container.

Examples:
  claude-reactor run                    # Auto-detect variant and run
  claude-reactor run --variant go       # Use Go toolchain variant
  claude-reactor run --shell            # Launch interactive shell instead
  claude-reactor run --danger           # Enable danger mode (skip permissions)
  claude-reactor run --account work     # Use specific account configuration
  claude-reactor run --persist=false    # Remove container when finished
  
  # Registry control (v2 images)
  claude-reactor run --dev              # Force local build (disable registry)
  claude-reactor run --registry-off     # Disable registry completely
  claude-reactor run --pull-latest      # Force pull latest from registry
  claude-reactor run --continue=false   # Disable conversation continuation`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runContainer(cmd, app)
		},
	}

	// Run command flags
	runCmd.Flags().StringP("variant", "", "", "Container variant (base, go, full, cloud, k8s)")
	runCmd.Flags().StringP("account", "", "", "Claude account to use")
	runCmd.Flags().BoolP("danger", "", false, "Enable danger mode (--dangerously-skip-permissions)")
	runCmd.Flags().BoolP("shell", "", false, "Launch shell instead of Claude CLI")
	runCmd.Flags().StringSliceP("mount", "m", []string{}, "Additional mount points (can be used multiple times)")
	runCmd.Flags().BoolP("persist", "", true, "Keep container running after exit (default: true)")
	runCmd.Flags().BoolP("no-mounts", "", false, "Skip mounting directories (for testing)")
	
	// Registry flags (Phase 0.1)
	runCmd.Flags().BoolP("dev", "", false, "Force local build (disable registry pulls)")
	runCmd.Flags().BoolP("registry-off", "", false, "Disable registry completely")
	runCmd.Flags().BoolP("pull-latest", "", false, "Force pull latest from registry")
	runCmd.Flags().BoolP("continue", "", true, "Enable conversation continuation (default: true)")

	return runCmd
}

func newBuildCmd(app *pkg.AppContainer) *cobra.Command {
	var buildCmd = &cobra.Command{
		Use:   "build [variant]",
		Short: "Build container images",
		Long: `Build Docker container images for the specified variant or all variants.
This will build the multi-stage Dockerfile with architecture-aware optimizations.

Examples:
  claude-reactor build               # Build base variant
  claude-reactor build go            # Build Go toolchain variant
  claude-reactor build --rebuild     # Force rebuild of base variant
  claude-reactor build full --rebuild # Force rebuild of full variant`,
		RunE: func(cmd *cobra.Command, args []string) error {
			app.Logger.Info("🔨 Building container images...")
			
			variant := "base"
			if len(args) > 0 {
				variant = args[0]
			}
			
			rebuild, _ := cmd.Flags().GetBool("rebuild")
			app.Logger.Infof("📋 Building variant: %s, force rebuild: %t", variant, rebuild)
			
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
				err = app.DockerMgr.RebuildImage(ctx, variant, platform, true)
			} else {
				err = app.DockerMgr.BuildImage(ctx, variant, platform)
			}
			
			if err != nil {
				return fmt.Errorf("failed to build image: %w. Try running 'docker system prune' to free space", err)
			}
			
			app.Logger.Info("✅ Image build completed successfully!")
			app.Logger.Info("💡 You can now run 'claude-reactor run' to start the container")
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
		Long: `Remove stopped containers and optionally clean up images.
Use --all to remove all claude-reactor containers across all accounts.

Examples:
  claude-reactor clean                # Remove current project container
  claude-reactor clean --all          # Remove all claude-reactor containers
  claude-reactor clean --images       # Also remove project images
  claude-reactor clean --all --images # Remove everything (containers + images)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cleanContainers(cmd, app)
		},
	}

	cleanCmd.Flags().BoolP("all", "", false, "Clean all claude-reactor containers")
	cleanCmd.Flags().BoolP("images", "", false, "Also remove images")

	return cleanCmd
}

func newDebugCmd(app *pkg.AppContainer) *cobra.Command {
	var debugCmd = &cobra.Command{
		Use:   "debug",
		Short: "Debug information and troubleshooting",
		Long:  "Provide debug information and troubleshooting tools.",
	}

	debugCmd.AddCommand(
		&cobra.Command{
			Use:   "info",
			Short: "Show system information",
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
	)

	return debugCmd
}

// handleLegacyListVariants handles the --list-variants flag for backward compatibility
func handleLegacyListVariants(cmd *cobra.Command, app *pkg.AppContainer) {
	cmd.Println("Available container variants:")
	variants := []string{"base", "go", "full", "cloud", "k8s"}
	for _, variant := range variants {
		cmd.Printf("  %s\n", variant)
	}
}

// handleLegacyShowConfig handles the --show-config flag for backward compatibility
func handleLegacyShowConfig(cmd *cobra.Command, app *pkg.AppContainer) {
	config, err := app.ConfigMgr.LoadConfig()
	if err != nil {
		cmd.Printf("Error loading configuration: %v\n", err)
		return
	}
	
	cmd.Printf("Current Configuration:\n")
	cmd.Printf("  Variant: %s\n", config.Variant)
	cmd.Printf("  Account: %s\n", config.Account)
	cmd.Printf("  Danger Mode: %t\n", config.DangerMode)
	cmd.Printf("  Project Path: %s\n", config.ProjectPath)
}

// runContainer implements the core run command logic
func runContainer(cmd *cobra.Command, app *pkg.AppContainer) error {
	ctx := cmd.Context()
	
	// Parse command flags
	variant, _ := cmd.Flags().GetString("variant")
	account, _ := cmd.Flags().GetString("account")
	danger, _ := cmd.Flags().GetBool("danger")
	shell, _ := cmd.Flags().GetBool("shell")
	mounts, _ := cmd.Flags().GetStringSlice("mount")
	persist, _ := cmd.Flags().GetBool("persist")
	noMounts, _ := cmd.Flags().GetBool("no-mounts")
	
	// Registry flags (Phase 0.1)
	devMode, _ := cmd.Flags().GetBool("dev")
	registryOff, _ := cmd.Flags().GetBool("registry-off")
	pullLatest, _ := cmd.Flags().GetBool("pull-latest")
	
	// Conversation control (Phase 0.3)
	continueConversation, _ := cmd.Flags().GetBool("continue")
	
	app.Logger.Info("🚀 Starting Claude CLI container...")
	
	// Step 1: Load or create configuration
	app.Logger.Info("📋 Loading configuration...")
	config, err := app.ConfigMgr.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w. Try running 'claude-reactor config validate' to check your setup", err)
	}
	
	// Override config with command-line flags
	if variant != "" {
		config.Variant = variant
	}
	if account != "" {
		config.Account = account
	}
	config.DangerMode = danger
	
	// Auto-detect variant if not specified
	if config.Variant == "" {
		app.Logger.Info("🔍 Auto-detecting project type...")
		detectedVariant, err := app.ConfigMgr.AutoDetectVariant("")
		if err != nil {
			app.Logger.Warnf("Failed to auto-detect variant: %v", err)
			app.Logger.Info("💡 Defaulting to 'base' variant. Use --variant flag to specify manually")
			config.Variant = "base"
		} else {
			config.Variant = detectedVariant
			app.Logger.Infof("✅ Auto-detected variant: %s", config.Variant)
		}
	}
	
	// Validate configuration
	app.Logger.Info("✅ Validating configuration...")
	if err := app.ConfigMgr.ValidateConfig(config); err != nil {
		return fmt.Errorf("invalid configuration: %w. Try using --variant with one of: base, go, full, cloud, k8s", err)
	}
	
	// Log registry configuration if relevant
	if devMode {
		app.Logger.Info("🔨 Registry: Dev mode enabled - forcing local builds")
	} else if registryOff {
		app.Logger.Info("🔨 Registry: Registry disabled - using local builds only")
	} else if pullLatest {
		app.Logger.Info("📦 Registry: Force pulling latest images from registry")
	}
	
	app.Logger.Infof("📋 Configuration: variant=%s, account=%s, danger=%t, shell=%t, persist=%t", 
		config.Variant, config.Account, config.DangerMode, shell, persist)
	
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
			Image:       imageName,
			Name:        containerName,
			Variant:     config.Variant,
			Platform:    platform,
			Interactive: true,
			TTY:         true,
			Remove:      false, // Don't auto-remove - we manage lifecycle
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
	
	return nil
}

// handleInstallation manages system installation and uninstallation (Phase 0.2)
func handleInstallation(cmd *cobra.Command, app *pkg.AppContainer, install bool) error {
	const installPath = "/usr/local/bin/claude-reactor"
	
	if install {
		// Installation process
		app.Logger.Info("🔧 Installing claude-reactor to system PATH...")
		
		// Get current executable path
		execPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to determine executable path: %w", err)
		}
		
		app.Logger.Infof("📋 Source: %s", execPath)
		app.Logger.Infof("🎯 Target: %s", installPath)
		
		// Check if we need sudo
		if err := checkWritePermissions("/usr/local/bin"); err != nil {
			app.Logger.Warn("⚠️  Installation requires sudo permissions for /usr/local/bin")
			app.Logger.Info("💡 You may be prompted for your password...")
			
			// Use sudo to copy the binary
			err := runWithSudo("cp", execPath, installPath)
			if err != nil {
				return fmt.Errorf("failed to install with sudo: %w", err)
			}
			
			// Make executable
			err = runWithSudo("chmod", "+x", installPath)
			if err != nil {
				return fmt.Errorf("failed to make executable with sudo: %w", err)
			}
		} else {
			// Direct copy (no sudo needed)
			err := copyFile(execPath, installPath)
			if err != nil {
				return fmt.Errorf("failed to copy binary: %w", err)
			}
			
			// Make executable
			err = os.Chmod(installPath, 0755)
			if err != nil {
				return fmt.Errorf("failed to make executable: %w", err)
			}
		}
		
		app.Logger.Info("✅ claude-reactor installed successfully!")
		app.Logger.Info("💡 You can now use 'claude-reactor' from anywhere in your terminal")
		app.Logger.Info("🧪 Test with: claude-reactor --version")
		
	} else {
		// Uninstallation process
		app.Logger.Info("🗑️ Removing claude-reactor from system PATH...")
		
		// Check if file exists
		if _, err := os.Stat(installPath); os.IsNotExist(err) {
			app.Logger.Info("✅ claude-reactor is not installed in system PATH")
			return nil
		}
		
		// Check if we need sudo
		if err := checkWritePermissions("/usr/local/bin"); err != nil {
			app.Logger.Warn("⚠️  Uninstallation requires sudo permissions for /usr/local/bin")
			app.Logger.Info("💡 You may be prompted for your password...")
			
			err := runWithSudo("rm", "-f", installPath)
			if err != nil {
				return fmt.Errorf("failed to uninstall with sudo: %w", err)
			}
		} else {
			// Direct removal (no sudo needed)
			err := os.Remove(installPath)
			if err != nil {
				return fmt.Errorf("failed to remove binary: %w", err)
			}
		}
		
		app.Logger.Info("✅ claude-reactor removed from system PATH")
		app.Logger.Info("💡 Local binary is still available at the original location")
	}
	
	return nil
}

// checkWritePermissions checks if we can write to the target directory
func checkWritePermissions(dir string) error {
	// Try to create a temporary file to test permissions
	testFile := filepath.Join(dir, ".claude-reactor-test")
	file, err := os.Create(testFile)
	if err != nil {
		return err
	}
	file.Close()
	os.Remove(testFile)
	return nil
}

// runWithSudo executes a command with sudo
func runWithSudo(command string, args ...string) error {
	allArgs := append([]string{command}, args...)
	cmd := exec.Command("sudo", allArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	
	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()
	
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}
	
	return destFile.Sync()
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
	generateCmd.Flags().String("variant", "", "Force specific container variant (base, go, full, cloud, k8s)")
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
	
	// Override variant if specified
	if variant, _ := cmd.Flags().GetString("variant"); variant != "" {
		if err := app.ConfigMgr.ValidateConfig(&pkg.Config{Variant: variant}); err != nil {
			return fmt.Errorf("invalid variant '%s': %w", variant, err)
		}
		config.Variant = variant
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
   • Specify variant: claude-reactor devcontainer generate --variant go

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