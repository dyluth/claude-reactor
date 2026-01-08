package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"claude-reactor/cmd/claude-reactor/commands"
	"claude-reactor/internal/reactor"
	"claude-reactor/internal/reactor/logging"
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

	// Special case: if user just wants help, create command without app initialization
	for _, arg := range os.Args[1:] {
		if arg == "--help" || arg == "-h" || arg == "help" {
			rootCmd := newRootCmd(nil) // Create with nil app for help
			return rootCmd.ExecuteContext(ctx)
		}
	}

	// Parse flags to get debug/verbose/log-level for app initialization
	tempCmd := &cobra.Command{Use: "claude-reactor"}
	tempCmd.PersistentFlags().Bool("debug", false, "Enable debug mode")
	tempCmd.PersistentFlags().Bool("verbose", false, "Enable verbose output")
	tempCmd.PersistentFlags().String("log-level", "info", "Set log level")
	tempCmd.PersistentFlags().Bool("version", false, "Print version information")
	tempCmd.SilenceErrors = true
	tempCmd.SilenceUsage = true
	// Ignore errors here as we might have other flags not defined in tempCmd
	_ = tempCmd.ParseFlags(os.Args[1:])

	debug, _ := tempCmd.PersistentFlags().GetBool("debug")
	verbose, _ := tempCmd.PersistentFlags().GetBool("verbose")
	logLevel, _ := tempCmd.PersistentFlags().GetString("log-level")

	// Initialize app upfront with parsed flags
	app, err := reactor.NewAppContainer(debug, verbose, logLevel)
	if err != nil {
		return fmt.Errorf("failed to initialize application: %w", err)
	}

	// Create root command with initialized app
	rootCmd := newRootCmd(app)
	return rootCmd.ExecuteContext(ctx)
}

func newRootCmd(app *pkg.AppContainer) *cobra.Command {
	var rootCmd = &cobra.Command{
		Use:   "claude-reactor",
		Short: "A simple, safe way to run Claude CLI in Docker containers with account isolation",
		Long: `Claude-Reactor provides a secure way to run Claude CLI in Docker containers
with proper account isolation. It offers multiple pre-built container variants
for different development needs while maintaining security and simplicity.`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", Version, GitCommit, BuildDate),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if app != nil {
				// Re-sync logger flags if they changed (e.g. from command line specific overrides)
				debug, _ := cmd.Flags().GetBool("debug")
				verbose, _ := cmd.Flags().GetBool("verbose")
				logLevel, _ := cmd.Flags().GetString("log-level")

				// Only update if different to avoid overhead
				if debug != app.Debug {
					app.Logger = logging.NewLoggerWithFlags(debug, verbose, logLevel)
					app.Debug = debug
					// Update config manager logger
					if app.ConfigMgr != nil {
						// We can't easily swap logger in existing manager without interface change/method
						// But for now, app.Logger is the source of truth for new components
					}
				}
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			// Handle deprecated flags with clear migration guidance
			if listVariants, _ := cmd.Flags().GetBool("list-variants"); listVariants {
				fmt.Fprintf(os.Stderr, "❌ The --list-variants flag has been removed. Use:\n")
				fmt.Fprintf(os.Stderr, "   claude-reactor info\n")
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
			if app != nil {
				config, err := app.ConfigMgr.LoadConfig()
				if err == nil && (config.Variant != "" || config.Account != "") {
					// Configuration exists, default to run command
					app.Logger.Info("🚀 Found existing configuration, running container...")
					runCmd := commands.NewRunCmd(app)
					if runErr := runCmd.RunE(cmd, args); runErr != nil {
						cmd.PrintErrf("Run failed: %v\n", runErr)
						os.Exit(1)
					}
					return
				}
			}

			// No configuration found, show help
			cmd.Help()
		},
	}

	// Global flags
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug mode")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().String("log-level", "info", "Set log level (debug, info, warn, error)")

	// Deprecated flags (hidden, show clear migration error)
	rootCmd.Flags().Bool("list-variants", false, "Removed: use 'debug info'")
	rootCmd.Flags().Bool("show-config", false, "Removed: use 'config show'")
	rootCmd.Flags().String("variant", "", "Removed: use 'run --image'")
	rootCmd.Flags().MarkHidden("list-variants")
	rootCmd.Flags().MarkHidden("show-config")
	rootCmd.Flags().MarkHidden("variant")

	// Version command
	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("claude-reactor version %s\n", Version)
			fmt.Printf("Git commit: %s\n", GitCommit)
			fmt.Printf("Build date: %s\n", BuildDate)
			fmt.Printf("Go version: %s\n", runtime.Version())
			fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		},
	}
	rootCmd.AddCommand(versionCmd)

	// Set version information for commands that need it
	commands.SetVersionInfo(Version, GitCommit, BuildDate)

	// Add subcommands from commands package
	rootCmd.AddCommand(
		commands.NewRunCmd(app),
		commands.NewConfigCmd(app),
		commands.NewCleanCmd(app),
		commands.NewInfoCmd(app),
		commands.NewListCmd(app),
		commands.NewCompletionCmd(app),
	)

	return rootCmd
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
