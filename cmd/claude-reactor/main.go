package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"claude-reactor/cmd/claude-reactor/commands"
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
				runCmd := commands.NewRunCmd(app)
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

	// Set version information for commands that need it
	commands.SetVersionInfo(Version, GitCommit, BuildDate)

	// Add subcommands from commands package
	rootCmd.AddCommand(
		commands.NewRunCmd(app),
		commands.NewBuildCmd(app),
		commands.NewConfigCmd(app),
		commands.NewCleanCmd(app),
		commands.NewDevContainerCmd(app),
		commands.NewTemplateCmd(app),
		commands.NewDependencyCmd(app),
		commands.NewHotReloadCmd(app),
		commands.NewDebugCmd(app),
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





