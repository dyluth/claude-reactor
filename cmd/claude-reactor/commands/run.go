package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"claude-reactor/internal/reactor"
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
  claude-reactor run --host-docker            # Enable host Docker access (⚠️  SECURITY WARNING)
  claude-reactor run --host-docker --host-docker-timeout 15m  # Host Docker with custom timeout
  claude-reactor run --host-docker --host-docker-timeout 0    # Host Docker with unlimited timeout
  claude-reactor run --danger --host-docker   # Combined danger mode and host Docker
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

Host Docker Access (⚠️  SECURITY WARNING):
  --host-docker flag grants HOST-LEVEL Docker privileges:
  • Can create/manage ANY container on the host
  • Can mount/access ANY host directory
  • Can access host network and other containers
  • Equivalent to ROOT access on the host system
  • Default timeout: 5m (override with --host-docker-timeout)
  • Only enable for trusted workflows requiring Docker management

Troubleshooting:
  Use 'claude-reactor info' to check Docker connectivity
  Use 'claude-reactor info image <name>' to test custom images
  Use '--verbose' flag for detailed validation information

Related Commands:
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
	runCmd.Flags().BoolP("host-docker", "", false, "Enable host Docker socket access (⚠️  SECURITY: grants host-level Docker privileges)")
	runCmd.Flags().StringP("host-docker-timeout", "", "5m", "Timeout for Docker operations (use '0' to disable timeout)")
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
	if app == nil {
		return fmt.Errorf("application container is not initialized")
	}

	ctx := cmd.Context()

	// Parse command flags
	image, _ := cmd.Flags().GetString("image")
	account, _ := cmd.Flags().GetString("account")
	apikey, _ := cmd.Flags().GetString("apikey")
	interactiveLogin, _ := cmd.Flags().GetBool("interactive-login")
	danger, _ := cmd.Flags().GetBool("danger")
	hostDocker, _ := cmd.Flags().GetBool("host-docker")
	hostDockerTimeout, _ := cmd.Flags().GetString("host-docker-timeout")
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

	// Detect if any arguments were passed for smart container reuse logic
	// Reuse ONLY when `claude-reactor run` called with NO arguments
	// Recreate when ANY arguments passed
	hasArgs := cmd.Flags().Changed("image") || cmd.Flags().Changed("account") || 
			   cmd.Flags().Changed("danger") || cmd.Flags().Changed("shell") ||
			   cmd.Flags().Changed("no-persist") || cmd.Flags().Changed("host-docker") ||
			   cmd.Flags().Changed("host-docker-timeout") || cmd.Flags().Changed("apikey") ||
			   cmd.Flags().Changed("interactive-login") || cmd.Flags().Changed("no-mounts") ||
			   cmd.Flags().Changed("dev") || cmd.Flags().Changed("registry-off") ||
			   cmd.Flags().Changed("pull-latest") || cmd.Flags().Changed("no-continue") ||
			   len(mounts) > 0
	
	app.Logger.Debugf("Smart container reuse: hasArgs=%v (will %s)", hasArgs, 
		func() string { if hasArgs { return "recreate" } else { return "reuse" } }())

	// Ensure Docker components are initialized
	if err := reactor.EnsureDockerComponents(app); err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}

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
	
	// Normalize account to use new default account logic ($USER fallback to "user")
	if config.Account == "" {
		config.Account = app.AuthMgr.GetDefaultAccount()
	}

	// Ensure account config file exists before any container operations to prevent corruption
	if err := app.AuthMgr.CopyMainConfigToAccount(config.Account); err != nil {
		app.Logger.Warnf("Failed to ensure account config exists: %v", err)
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

	// Handle host Docker configuration with persistence logic
	if cmd.Flags().Changed("host-docker") {
		config.HostDocker = hostDocker
		if hostDocker {
			app.Logger.Info("🐳 Host Docker access enabled and will be persisted")
		} else {
			app.Logger.Info("🔒 Host Docker access disabled and will be persisted")
		}
	} else if config.HostDocker {
		app.Logger.Info("🐳 Using persistent host Docker setting")
		hostDocker = true // Use saved setting
	}

	// Handle host Docker timeout configuration
	if cmd.Flags().Changed("host-docker-timeout") {
		config.HostDockerTimeout = hostDockerTimeout
	} else if config.HostDockerTimeout != "" {
		hostDockerTimeout = config.HostDockerTimeout
	} else {
		// Set default if not configured
		config.HostDockerTimeout = "5m"
		hostDockerTimeout = "5m"
	}

	// Validate timeout format if host Docker is enabled
	if hostDocker {
		if hostDockerTimeout != "0" && hostDockerTimeout != "0s" {
			if _, err := time.ParseDuration(hostDockerTimeout); err != nil {
				return fmt.Errorf("invalid timeout format '%s': %w\n💡 Use Go duration format: 5m, 1h30m, 30s\n💡 Valid examples: 30s, 5m, 1h, 2h30m\n💡 Disable timeout: 0", hostDockerTimeout, err)
			}
		}

		// Display security warning for host Docker access
		displayHostDockerSecurityWarning(app.Logger, hostDockerTimeout)
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

	// Step 2: Get current project directory
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Step 3: Generate container and image names
	app.Logger.Info("🔧 Detecting system architecture...")
	arch, err := app.ArchDetector.GetHostArchitecture()
	if err != nil {
		return fmt.Errorf("failed to detect architecture: %w. Your system may not be supported", err)
	}

	containerName := app.DockerMgr.GenerateContainerName(projectDir, config.Variant, arch, config.Account)
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

	// Create Docker operation context with timeout if host Docker is enabled
	dockerCtx := ctx
	if hostDocker && hostDockerTimeout != "0" && hostDockerTimeout != "0s" {
		timeout, _ := time.ParseDuration(hostDockerTimeout) // Already validated above
		var cancel context.CancelFunc
		dockerCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
		app.Logger.Infof("🕒 Docker operations timeout set to: %s", hostDockerTimeout)
	}

	// Build image with registry support (Phase 0.1)
	err = app.DockerMgr.BuildImageWithRegistry(dockerCtx, config.Variant, platform, devMode, registryOff, pullLatest)
	if err != nil {
		// Check for timeout error
		if dockerCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("Docker operation timed out after %s\n💡 For complex builds, increase timeout: --host-docker-timeout 15m\n💡 For unlimited time, disable timeout: --host-docker-timeout 0\n💡 Save preference: echo \"host_docker_timeout=15m\" >> .claude-reactor", hostDockerTimeout)
		}
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
		HostDocker:       hostDocker,
		HostDockerTimeout: hostDockerTimeout,
	}

	// Add mounts (skip if requested for testing)
	if !noMounts {
		app.Logger.Info("📁 Configuring container mounts...")
		err = AddMountsToContainer(app, containerConfig, config.Account, mounts, projectDir)
		if err != nil {
			return fmt.Errorf("failed to configure mounts: %w. Check that source directories exist and are accessible", err)
		}
	} else {
		app.Logger.Info("🚫 Skipping mounts due to --no-mounts flag")
		// Set empty mounts to prevent default mount creation
		containerConfig.Mounts = []pkg.Mount{}
	}

	// Step 5: Smart container lifecycle management
	var containerID string
	
	// Check if container already exists for this project/account
	containerExists, err := app.DockerMgr.IsContainerRunning(dockerCtx, containerName)
	if err != nil {
		app.Logger.Debugf("Failed to check container status: %v", err)
		containerExists = false
	}
	
	// Apply smart reuse logic
	if containerExists && !hasArgs {
		// Reuse existing container when no arguments passed
		app.Logger.Info("♻️ Reusing existing container (no arguments passed)...")
		
		// For reuse, we need to attach to the existing container
		// First get the container status to get its ID
		status, err := app.DockerMgr.GetContainerStatus(dockerCtx, containerName)
		if err != nil {
			return fmt.Errorf("failed to get container status for reuse: %w", err)
		}
		containerID = status.ID
		
	} else if containerExists && hasArgs {
		// Recreate container when arguments passed
		app.Logger.Info("🔄 Recreating container (arguments passed, forcing recreation)...")
		
		// Remove existing container first
		err = app.DockerMgr.CleanContainer(dockerCtx, containerName)
		if err != nil {
			app.Logger.Warnf("Failed to remove existing container: %v", err)
		}
		
		// Start new container
		if config.SessionPersistence {
			containerID, err = app.DockerMgr.StartOrRecoverContainer(dockerCtx, containerConfig, config)
		} else {
			containerID, err = app.DockerMgr.StartContainer(dockerCtx, containerConfig)
		}
		if err != nil {
			if dockerCtx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("Docker operation timed out after %s\n💡 For complex builds, increase timeout: --host-docker-timeout 15m\n💡 For unlimited time, disable timeout: --host-docker-timeout 0\n💡 Save preference: echo \"host_docker_timeout=15m\" >> .claude-reactor", hostDockerTimeout)
			}
			return fmt.Errorf("failed to start container: %w. Check Docker daemon is running and try 'docker system prune'", err)
		}
		
	} else {
		// No existing container, start new one
		if config.SessionPersistence {
			app.Logger.Info("🔄 Starting container with session persistence...")
			containerID, err = app.DockerMgr.StartOrRecoverContainer(dockerCtx, containerConfig, config)
		} else {
			app.Logger.Info("🏗️ Starting ephemeral container...")
			containerID, err = app.DockerMgr.StartContainer(dockerCtx, containerConfig)
		}
		if err != nil {
			if dockerCtx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("Docker operation timed out after %s\n💡 For complex builds, increase timeout: --host-docker-timeout 15m\n💡 For unlimited time, disable timeout: --host-docker-timeout 0\n💡 Save preference: echo \"host_docker_timeout=15m\" >> .claude-reactor", hostDockerTimeout)
			}
			return fmt.Errorf("failed to start container: %w. Check Docker daemon is running and try 'docker system prune'", err)
		}
	}
	
	// Update session tracking for session persistence
	if config.SessionPersistence {
		config.ContainerID = containerID
		if err := app.ConfigMgr.SaveConfig(config); err != nil {
			app.Logger.Warnf("Failed to save session configuration: %v", err)
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
func AddMountsToContainer(app *pkg.AppContainer, containerConfig *pkg.ContainerConfig, account string, userMounts []string, projectDir string) error {
	// Add default mounts (project directory, Claude config)

	// Project mount - avoid circular mount if we're already in /app
	targetPath := "/app"
	if projectDir == "/app" {
		targetPath = "/workspace" // Use different path to avoid circular mount
	}

	err := app.MountMgr.AddMountToConfig(containerConfig, projectDir, targetPath)
	if err != nil {
		return fmt.Errorf("failed to add project mount: %w", err)
	}
	app.Logger.Infof("📁 Project mount: %s -> %s", projectDir, targetPath)

	// Claude authentication mount - use account-specific Claude config file
	// This ensures persistent authentication for each account across container restarts
	claudeConfigPath := app.AuthMgr.GetAccountConfigPath(account)
	
	// Mount account-specific Claude config file for persistent authentication
	if _, err := os.Stat(claudeConfigPath); err == nil {
		err = app.MountMgr.AddMountToConfig(containerConfig, claudeConfigPath, "/home/claude/.claude.json")
		if err != nil {
			app.Logger.Warnf("Failed to add Claude config mount: %v", err)
		} else {
			app.Logger.Infof("🔑 Claude auth mount: %s -> /home/claude/.claude.json", claudeConfigPath)
		}
	} else {
		app.Logger.Warnf("Claude config file not found for account %s: %s", account, claudeConfigPath)
	}

	// Claude session directory mount - use project-specific session directory  
	// This contains conversation history, shell snapshots, todos, etc.
	// Format: ~/.claude-reactor/{account}/{project-name}-{project-hash}/
	claudeSessionDir := app.AuthMgr.GetProjectSessionDir(account, projectDir)
	
	// Ensure session directory exists (including parent account directory)
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
	
	// Mount the main user's credentials file for OAuth tokens
	homeDir, err := os.UserHomeDir()
	if err == nil {
		mainCredentialsPath := filepath.Join(homeDir, ".claude", ".credentials.json")
		if _, err := os.Stat(mainCredentialsPath); err == nil {
			err = app.MountMgr.AddMountToConfig(containerConfig, mainCredentialsPath, "/home/claude/.claude/.credentials.json")
			if err != nil {
				app.Logger.Warnf("Failed to add credentials mount: %v", err)
			} else {
				app.Logger.Infof("🔐 Credentials mount: %s -> /home/claude/.claude/.credentials.json", mainCredentialsPath)
			}
		} else {
			app.Logger.Debugf("Main credentials file not found: %s", mainCredentialsPath)
		}
	}

	// Add Docker socket mount if host Docker access is enabled
	if containerConfig.HostDocker {
		dockerSock := "/var/run/docker.sock"
		if _, err := os.Stat(dockerSock); err == nil {
			err = app.MountMgr.AddMountToConfig(containerConfig, dockerSock, "/var/run/docker.sock")
			if err != nil {
				return fmt.Errorf("failed to add Docker socket mount: %w", err)
			}
			app.Logger.Infof("🐳 Host Docker socket mount: %s -> /var/run/docker.sock", dockerSock)
		} else {
			return fmt.Errorf("host Docker requested but socket not available at %s\n💡 Mount Docker socket: -v /var/run/docker.sock:/var/run/docker.sock\n💡 Add docker group: --group-add docker\n💡 See documentation: claude-reactor help docker-setup", dockerSock)
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

// displayHostDockerSecurityWarning shows a prominent security warning when host Docker access is enabled
func displayHostDockerSecurityWarning(logger pkg.Logger, timeout string) {
	logger.Info("")
	logger.Info("⚠️  WARNING: HOST DOCKER ACCESS ENABLED")
	logger.Info("🔒 This grants claude-reactor container HOST-LEVEL Docker privileges:")
	logger.Info("   • Can create/manage ANY container on the host")
	logger.Info("   • Can mount/access ANY host directory")
	logger.Info("   • Can access host network and other containers")
	logger.Info("   • Equivalent to ROOT access on the host system")

	if timeout == "0" || timeout == "0s" {
		logger.Info("⏰ Docker operations: UNLIMITED TIMEOUT (no timeout protection)")
	} else {
		logger.Info(fmt.Sprintf("⏰ Docker operations timeout: %s", timeout))
	}

	logger.Info("💡 Only enable for trusted workflows requiring Docker management")
	logger.Info("")
}
