package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"claude-reactor/pkg"
)

// NewRunCmd creates the run command for starting and connecting to Claude CLI containers
func NewRunCmd(app *pkg.AppContainer) *cobra.Command {
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
			return RunContainer(cmd, app)
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

// RunContainer handles the main container execution logic
func RunContainer(cmd *cobra.Command, app *pkg.AppContainer) error {
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
	persist := !noPersist // Default to true, unless --no-persist is specified
	noMounts, _ := cmd.Flags().GetBool("no-mounts")

	// Registry flags (Phase 0.1)
	devMode, _ := cmd.Flags().GetBool("dev")
	registryOff, _ := cmd.Flags().GetBool("registry-off")
	pullLatest, _ := cmd.Flags().GetBool("pull-latest")

	// Conversation control (Phase 0.3)
	noContinue, _ := cmd.Flags().GetBool("no-continue")

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

	// Session persistence: respect saved configuration
	// The user has explicitly configured session_persistence=true, so honor it
	// (Session persistence setting is already loaded from configuration above)

	// Save configuration to persist user preferences (including danger mode and session persistence)
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

	if config.SessionPersistence {
		app.Logger.Infof("📋 Configuration: image=%s, account=%s, danger=%t, shell=%t, persist=%t, session_persistence=%t",
			config.Variant, config.Account, config.DangerMode, shell, persist, config.SessionPersistence)
	} else {
		app.Logger.Infof("📋 Configuration: image=%s, account=%s, danger=%t, shell=%t, persist=%t",
			config.Variant, config.Account, config.DangerMode, shell, persist)
	}

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

	// Step 3: Build image if needed
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

	// Step 4: Create container configuration
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
		err = AddMountsToContainer(app, containerConfig, config.Account, mounts)
		if err != nil {
			return fmt.Errorf("failed to configure mounts: %w. Check that source directories exist and are accessible", err)
		}
	} else {
		app.Logger.Info("🚫 Skipping mounts due to --no-mounts flag")
		// Set empty mounts to prevent default mount creation
		containerConfig.Mounts = []pkg.Mount{}
	}

	// Step 5: Start or recover container with session persistence
	var containerID string
	if config.SessionPersistence {
		app.Logger.Info("🔄 Starting container with session persistence...")
		containerID, err = app.DockerMgr.StartOrRecoverContainer(ctx, containerConfig, config)
		if err != nil {
			return fmt.Errorf("failed to start or recover container: %w. Check Docker daemon is running and try 'docker system prune'", err)
		}

		// Update session tracking - save config after session persistence operations
		// This ensures both session ID and container ID are persisted
		config.ContainerID = containerID
		if err := app.ConfigMgr.SaveConfig(config); err != nil {
			app.Logger.Warnf("Failed to save session configuration: %v", err)
		}
	} else {
		app.Logger.Info("🏗️ Starting ephemeral container...")
		containerID, err = app.DockerMgr.StartContainer(ctx, containerConfig)
		if err != nil {
			return fmt.Errorf("failed to start container: %w. Check Docker daemon is running and try 'docker system prune'", err)
		}
	}

	app.Logger.Info("✅ Container started successfully!")

	// Step 6: Attach to container
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
		if noContinue {
			app.Logger.Info("💬 Conversation continuation disabled")
		} else {
			command = append(command, "--continue")
			app.Logger.Debug("💬 Conversation continuation enabled (default)")
		}
		if app.Debug {
			command = append(command, "-d", "--verbose")
		}

	}

	// Attach to container
	err = app.DockerMgr.AttachToContainer(ctx, containerName, command, true)
	if err != nil {
		return fmt.Errorf("failed to attach to container: %w. Try using 'docker exec -it %s %s' as fallback", err, containerName, strings.Join(command, " "))
	}

	// Step 7: Handle container persistence
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

// AddMountsToContainer adds mount points to container configuration
func AddMountsToContainer(app *pkg.AppContainer, containerConfig *pkg.ContainerConfig, account string, userMounts []string) error {
	// Add default mounts (project directory, Claude config)
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Project mount - avoid circular mount if we're already in /app
	targetPath := "/app"
	if projectDir == "/app" {
		targetPath = "/workspace" // Use different path to avoid circular mount
	}

	err = app.MountMgr.AddMountToConfig(containerConfig, projectDir, targetPath)
	if err != nil {
		return fmt.Errorf("failed to add project mount: %w", err)
	}
	app.Logger.Infof("📁 Project mount: %s -> %s", projectDir, targetPath)

	// Claude session directory mount - use account-specific session directory
	// This ensures all accounts (including default) use isolated ~/.claude-reactor/[account]/ directories
	claudeSessionDir := app.AuthMgr.GetAccountSessionDir(account)

	// Only mount if session directory exists, or create it if needed
	if err := os.MkdirAll(claudeSessionDir, 0755); err != nil {
		app.Logger.Warnf("Failed to create Claude session directory: %v", err)
	} else {
		err = app.MountMgr.AddMountToConfig(containerConfig, claudeSessionDir, "/home/claude/.claude")
		if err != nil {
			app.Logger.Warnf("Failed to add Claude session mount: %v", err)
		} else {
			app.Logger.Infof("📁 Claude session mount: %s -> /home/claude/.claude", claudeSessionDir)
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
