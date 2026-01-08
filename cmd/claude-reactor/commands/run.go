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
  claude-reactor run --ssh-agent              # Enable SSH agent forwarding (auto-detect)
  claude-reactor run --ssh-agent=/tmp/ssh.sock # SSH agent with explicit socket path
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
			// Handle help case when app is nil
			if app == nil {
				return cmd.Help()
			}
			return RunContainer(cmd, app)
		},
	}

	// Run command flags
	runCmd.Flags().StringP("image", "", "", "Container image (base, go, full, cloud, k8s, or custom Docker image)")
	runCmd.Flags().StringP("account", "", "", "Claude account to use")
	runCmd.Flags().StringP("apikey", "", "", "Set API key for this session (creates account-specific env file)")
	runCmd.Flags().BoolP("interactive-login", "", false, "Force interactive authentication for account")
	runCmd.Flags().BoolP("shell", "", false, "Launch shell instead of Claude CLI")
	runCmd.Flags().StringSliceP("mount", "m", []string{}, "Additional mount points (can be used multiple times)")
	runCmd.Flags().BoolP("no-persist", "", false, "Remove container when finished (default: keep running)")

	// Advanced / Deprecated flags (use config instead)
	runCmd.Flags().BoolP("danger", "", false, "Enable danger mode")
	runCmd.Flags().MarkHidden("danger")

	runCmd.Flags().BoolP("host-docker", "", false, "Enable host Docker socket access")
	runCmd.Flags().MarkHidden("host-docker")

	runCmd.Flags().StringP("host-docker-timeout", "", "5m", "Timeout for Docker operations")
	runCmd.Flags().MarkHidden("host-docker-timeout")

	runCmd.Flags().StringP("ssh-agent", "", "", "Enable SSH agent forwarding")
	runCmd.Flags().MarkHidden("ssh-agent")
	runCmd.Flags().Lookup("ssh-agent").NoOptDefVal = "auto"

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
	sshAgent, _ := cmd.Flags().GetString("ssh-agent")
	shell, _ := cmd.Flags().GetBool("shell")
	mounts, _ := cmd.Flags().GetStringSlice("mount")
	noPersist, _ := cmd.Flags().GetBool("no-persist")
	persist := !noPersist // Default to true, unless --no-persist is specified

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

	// Handle SSH agent configuration with persistence logic
	var sshAgentEnabled bool
	var sshAgentSocket string

	if cmd.Flags().Changed("ssh-agent") {
		// SSH agent flag was provided (with or without value)
		sshAgentEnabled = true

		if sshAgent == "" || sshAgent == "auto" {
			// Auto-detect SSH agent socket
			detectedSocket, err := app.ConfigMgr.DetectSSHAgent()
			if err != nil {
				return fmt.Errorf("failed to detect SSH agent: %w", err)
			}
			sshAgentSocket = detectedSocket
			config.SSHAgentSocket = "auto"
			app.Logger.Info("🔑 SSH agent auto-detected and will be persisted")
		} else {
			// Explicit socket path provided
			sshAgentSocket = sshAgent
			config.SSHAgentSocket = sshAgent
			app.Logger.Infof("🔑 SSH agent socket specified: %s (will be persisted)", sshAgent)
		}

		// Validate SSH agent connectivity
		if err := app.ConfigMgr.ValidateSSHAgent(sshAgentSocket); err != nil {
			return fmt.Errorf("SSH agent validation failed: %w", err)
		}

		config.SSHAgent = true
		app.Logger.Info("✅ SSH agent validation passed")
	} else if config.SSHAgent {
		// Use persistent SSH agent setting
		sshAgentEnabled = true
		if config.SSHAgentSocket == "auto" || config.SSHAgentSocket == "" {
			// Re-detect for auto mode
			detectedSocket, err := app.ConfigMgr.DetectSSHAgent()
			if err != nil {
				app.Logger.Warnf("Failed to re-detect SSH agent, disabling: %v", err)
				config.SSHAgent = false
				sshAgentEnabled = false
			} else {
				sshAgentSocket = detectedSocket
				app.Logger.Info("🔑 Using persistent SSH agent setting (auto-detected)")
			}
		} else {
			// Use explicit socket from config
			sshAgentSocket = config.SSHAgentSocket
			app.Logger.Infof("🔑 Using persistent SSH agent setting: %s", sshAgentSocket)

			// Re-validate socket
			if err := app.ConfigMgr.ValidateSSHAgent(sshAgentSocket); err != nil {
				app.Logger.Warnf("Persistent SSH agent socket invalid, disabling: %v", err)
				config.SSHAgent = false
				sshAgentEnabled = false
			}
		}
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
	config.ProjectPath = projectDir

	// Step 3: Generate container and image names
	app.Logger.Info("🔧 Detecting system architecture...")
	arch, err := app.ArchDetector.GetHostArchitecture()
	if err != nil {
		return fmt.Errorf("failed to detect architecture: %w. Your system may not be supported", err)
	}

	containerName := app.DockerMgr.GenerateContainerName(projectDir, config.Variant, arch, config.Account)
	app.Logger.Infof("🏷️ Container name: %s", containerName)

	// Step 4: Resolve and Ensure Image
	imageName := app.DockerMgr.GetImageName(config.Variant, arch)

	if isBuiltinVariant {
		// Check if local image exists by checking its validity without pull
		if _, err := app.ImageValidator.ValidateImage(ctx, imageName, false); err != nil {
			// Local image missing, switch to registry
			registryImage := fmt.Sprintf("ghcr.io/dyluth/claude-reactor-%s:latest", config.Variant)
			app.Logger.Infof("📦 Local image '%s' not found, using registry: %s", imageName, registryImage)
			imageName = registryImage
		} else {
			app.Logger.Infof("✅ Found local image: %s", imageName)
		}
	}

	app.Logger.Info("🐳 Preparing Docker environment...")
	platform, err := app.ArchDetector.GetDockerPlatform()
	if err != nil {
		return fmt.Errorf("failed to get Docker platform: %w. Architecture detection failed", err)
	}

	// Create Docker operation context with timeout if host Docker is enabled
	dockerCtx := ctx
	if hostDocker && hostDockerTimeout != "0" && hostDockerTimeout != "0s" {
		timeout, _ := time.ParseDuration(hostDockerTimeout) // Already validated above
		var cancel context.CancelFunc
		dockerCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
		app.Logger.Infof("🕒 Docker operations timeout set to: %s", hostDockerTimeout)
	}

	// Step 5: Create container configuration
	containerConfig := &pkg.ContainerConfig{
		Image:             imageName,
		Name:              containerName,
		Variant:           config.Variant,
		Platform:          platform,
		Interactive:       true,
		TTY:               true,
		Remove:            false, // Don't auto-remove - we manage lifecycle
		RunClaudeUpgrade:  true,  // Run claude upgrade after container startup
		HostDocker:        hostDocker,
		HostDockerTimeout: hostDockerTimeout,
		SSHAgent:          sshAgentEnabled,
		SSHAgentSocket:    sshAgentSocket,
		Environment:       make(map[string]string),
	}

	// Configure timezone to match host
	// This ensures timestamps in container match the user's local time
	if tz := os.Getenv("TZ"); tz != "" {
		containerConfig.Environment["TZ"] = tz
		app.Logger.Debugf("🕐 Setting container timezone: %s", tz)
	} else {
		// Fallback: try to read /etc/timezone or /etc/localtime
		if data, err := os.ReadFile("/etc/timezone"); err == nil {
			tz := strings.TrimSpace(string(data))
			if tz != "" {
				containerConfig.Environment["TZ"] = tz
				app.Logger.Debugf("🕐 Setting container timezone from /etc/timezone: %s", tz)
			}
		}
	}

	// Add mounts
	app.Logger.Info("📁 Configuring container mounts...")
	err = AddMountsToContainer(app, containerConfig, config.Account, mounts, projectDir)
	if err != nil {
		return fmt.Errorf("failed to configure mounts: %w. Check that source directories exist and are accessible", err)
	}

	// Step 6: Lifecycle Management
	var containerID string

	// Check if container already exists
	containerExists, err := app.DockerMgr.IsContainerRunning(dockerCtx, containerName)
	if err != nil {
		app.Logger.Debugf("Failed to check container status: %v", err)
		containerExists = false
	}

	if containerExists {
		// Reuse existing
		app.Logger.Info("♻️ Reusing existing container...")
		status, err := app.DockerMgr.GetContainerStatus(dockerCtx, containerName)
		if err != nil {
			return fmt.Errorf("failed to get container status for reuse: %w", err)
		}
		containerID = status.ID
	} else {
		// Start new or resume stopped
		status, err := app.DockerMgr.GetContainerStatus(dockerCtx, containerName)
		if err != nil {
			app.Logger.Debugf("Failed to get status for potentially stopped container: %v", err)
		}

		if config.SessionPersistence {
			app.Logger.Info("🔄 Starting/Resuming container with session persistence...")
			// StartOrRecoverContainer handles logic for resuming stopped containers
			containerID, err = app.DockerMgr.StartOrRecoverContainer(dockerCtx, containerConfig, config)
		} else {
			app.Logger.Info("🏗️ Starting ephemeral container...")
			// If ephemeral and exists (but stopped/dead), we must clean it first to avoid name conflict
			if status != nil && status.Exists {
				app.Logger.Debug("Removing stopped ephemeral container before recreation...")
				if err := app.DockerMgr.RemoveContainer(dockerCtx, status.ID); err != nil {
					app.Logger.Warnf("Failed to remove stopped container: %v", err)
				}
			}
			containerID, err = app.DockerMgr.StartContainer(dockerCtx, containerConfig)
		}

		if err != nil {
			if dockerCtx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("Docker operation timed out after %s\n💡 For complex builds, increase timeout: --host-docker-timeout 15m", hostDockerTimeout)
			}
			return fmt.Errorf("failed to start container: %w. Check Docker daemon is running and try 'docker system prune'", err)
		}
	}

	// Update session tracking for session persistence
	if config.SessionPersistence {
		config.ContainerID = containerID
	}
	// Always save config to current directory for persistence
	if err := app.ConfigMgr.SaveConfig(config); err != nil {
		app.Logger.Warnf("Failed to save session configuration: %v", err)
	}

	// Also save config to session directory so 'list' command can read metadata
	// calculated session dir: ~/.claude-reactor/{account}/{project-name}-{project-hash}/
	sessionDir := app.AuthMgr.GetProjectSessionDir(config.Account, config.ProjectPath)
	if sessionDir != "" {
		if err := os.MkdirAll(sessionDir, 0755); err == nil {
			sessionConfigPath := filepath.Join(sessionDir, ".claude-reactor")
			// We manually write this for now as ConfigMgr doesn't support custom paths yet
			// In a future refactor, we should add SaveConfigToPath(path, config)
			if data, err := os.ReadFile(".claude-reactor"); err == nil {
				if err := os.WriteFile(sessionConfigPath, data, 0644); err != nil {
					app.Logger.Debugf("Failed to backup config to session dir: %v", err)
				}
			}
		}
	}

	app.Logger.Info("✅ Container started successfully!")

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

		// Conversation control
		// TODO: Fix additional working directories issue before re-enabling --continue support
		app.Logger.Debug("💬 Conversation continuation temporarily disabled due to path issue")

		if app.Debug {
			command = append(command, "-d", "--verbose")
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

	// Create project-specific .claude.json file if it doesn't exist
	// This ensures each project has isolated Claude CLI configuration
	projectClaudeConfig := filepath.Join(claudeSessionDir, ".claude.json")
	if _, err := os.Stat(projectClaudeConfig); os.IsNotExist(err) {
		// Copy from account config as template
		accountConfigPath := app.AuthMgr.GetAccountConfigPath(account)
		if err := app.AuthMgr.CopyMainConfigToAccount(account); err != nil {
			app.Logger.Warnf("Failed to ensure account config exists: %v", err)
		}

		if data, readErr := os.ReadFile(accountConfigPath); readErr == nil {
			if writeErr := os.WriteFile(projectClaudeConfig, data, 0644); writeErr != nil {
				app.Logger.Warnf("Failed to create project-specific .claude.json: %v", writeErr)
			} else {
				app.Logger.Infof("📄 Created project-specific config: %s", projectClaudeConfig)
			}
		} else {
			app.Logger.Warnf("Failed to read account config template: %v", readErr)
		}
	}

	// Mount project-specific .claude.json instead of account-wide config
	// This prevents config file conflicts between different projects
	if _, err := os.Stat(projectClaudeConfig); err == nil {
		err = app.MountMgr.AddMountToConfig(containerConfig, projectClaudeConfig, "/home/claude/.claude.json")
		if err != nil {
			app.Logger.Warnf("Failed to add project Claude config mount: %v", err)
		} else {
			app.Logger.Infof("🔑 Claude config mount: %s -> /home/claude/.claude.json", projectClaudeConfig)
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

	// Add SSH agent mounts if SSH agent forwarding is enabled
	if containerConfig.SSHAgent {
		sshMounts, err := app.ConfigMgr.PrepareSSHMounts(containerConfig.SSHAgent, containerConfig.SSHAgentSocket)
		if err != nil {
			return fmt.Errorf("failed to prepare SSH mounts: %w", err)
		}

		for _, mount := range sshMounts {
			err = app.MountMgr.AddMountToConfig(containerConfig, mount.Source, mount.Target)
			if err != nil {
				app.Logger.Warnf("Failed to add SSH mount %s -> %s: %v", mount.Source, mount.Target, err)
			} else {
				app.Logger.Infof("🔑 SSH mount: %s -> %s", mount.Source, mount.Target)
			}
		}

		// Skip SSH_AUTH_SOCK setup - using direct SSH key mounting instead
		// This is more reliable than agent forwarding for Git operations on Docker Desktop
		if containerConfig.Environment == nil {
			containerConfig.Environment = make(map[string]string)
		}
		app.Logger.Info("🔑 SSH keys configured for Git operations")
	}

	// Add global subagents mount if directory exists
	// Global subagents are stored in ~/.claude/agents/ and available across all projects
	if homeDir, err := os.UserHomeDir(); err == nil {
		globalSubagentsDir := filepath.Join(homeDir, ".claude", "agents")
		if _, err := os.Stat(globalSubagentsDir); err == nil {
			// Ensure the target directory exists in the session directory
			sessionSubagentsDir := filepath.Join(claudeSessionDir, "agents")
			if err := os.MkdirAll(sessionSubagentsDir, 0755); err != nil {
				app.Logger.Warnf("Failed to create session subagents directory: %v", err)
			} else {
				err = app.MountMgr.AddMountToConfig(containerConfig, globalSubagentsDir, "/home/claude/.claude/agents")
				if err != nil {
					app.Logger.Warnf("Failed to add global subagents mount: %v", err)
				} else {
					app.Logger.Infof("🤖 Global subagents mount: %s -> /home/claude/.claude/agents", globalSubagentsDir)
				}
			}
		} else {
			app.Logger.Debugf("Global subagents directory not found: %s", globalSubagentsDir)
		}
	}

	// Add project-specific subagents mount if directory exists
	// Project-specific subagents are stored in {project}/.claude/agents/ and versioned with the repo
	projectSubagentsDir := filepath.Join(projectDir, ".claude", "agents")
	if _, err := os.Stat(projectSubagentsDir); err == nil {
		// The project directory is already mounted at /app or /workspace,
		// so project subagents will be automatically available at /app/.claude/agents
		// or /workspace/.claude/agents. We just log this for visibility.
		app.Logger.Infof("🤖 Project subagents detected: %s (available via project mount)", projectSubagentsDir)
	} else {
		app.Logger.Debugf("Project-specific subagents directory not found: %s", projectSubagentsDir)
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
