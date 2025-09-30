package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"claude-reactor/pkg"
)

// NewConfigCmd creates the config command for managing configuration
func NewConfigCmd(app *pkg.AppContainer) *cobra.Command {
	var configCmd = &cobra.Command{
		Use:   "config",
		Short: "Manage claude-reactor configuration",
		Long: `Display and manage claude-reactor configuration settings.
View current configuration, account settings, and project-specific preferences.`,
	}

	configCmd.AddCommand(
		newConfigShowCmd(app),
		newConfigValidateCmd(app),
		newConfigSetCmd(app),
	)

	return configCmd
}

func newConfigShowCmd(app *pkg.AppContainer) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showEnhancedConfig(cmd, app)
		},
	}
}

func newConfigValidateCmd(app *pkg.AppContainer) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return validateConfig(cmd, app)
		},
	}
}

func newConfigSetCmd(app *pkg.AppContainer) *cobra.Command {
	return &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return setConfigValue(cmd, args, app)
		},
	}
}

// showEnhancedConfig displays the current configuration
func showEnhancedConfig(cmd *cobra.Command, app *pkg.AppContainer) error {
	config, err := app.ConfigMgr.LoadConfig()
	if err != nil {
		app.Logger.Warnf("Could not load config: %v", err)
		config = app.ConfigMgr.GetDefaultConfig()
	}

	fmt.Printf("📋 Claude-Reactor Configuration\n")
	fmt.Printf("═══════════════════════════════\n\n")
	
	fmt.Printf("🖼️  Image/Variant: %s\n", getDisplayValue(config.Variant, "auto-detect"))
	fmt.Printf("👤 Account: %s\n", getDisplayValue(config.Account, "default"))
	fmt.Printf("🔥 Danger Mode: %t\n", config.DangerMode)
	fmt.Printf("🐳 Host Docker: %t\n", config.HostDocker)
	if config.HostDocker {
		fmt.Printf("⏰ Host Docker Timeout: %s\n", getDisplayValue(config.HostDockerTimeout, "5m"))
	}
	fmt.Printf("💾 Session Persistence: %t\n", config.SessionPersistence)
	if config.SessionPersistence {
		fmt.Printf("🔗 Last Session ID: %s\n", getDisplayValue(config.LastSessionID, "none"))
		fmt.Printf("📦 Container ID: %s\n", getDisplayValue(config.ContainerID, "none"))
	}
	
	// Show current directory and project detection
	fmt.Printf("\n📁 Current Directory: %s\n", getCurrentDir())
	
	// Show auto-detected variant
	detectedVariant, err := app.ConfigMgr.AutoDetectVariant("")
	if err == nil {
		fmt.Printf("🔍 Auto-detected: %s\n", detectedVariant)
	}

	return nil
}

func getDisplayValue(value, defaultDisplay string) string {
	if value == "" {
		return fmt.Sprintf("<%s>", defaultDisplay)
	}
	return value
}

func getCurrentDir() string {
	if dir, err := os.Getwd(); err == nil {
		return dir
	}
	return "<unknown>"
}

// validateConfig validates the current configuration
func validateConfig(cmd *cobra.Command, app *pkg.AppContainer) error {
	config, err := app.ConfigMgr.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	if err := app.ConfigMgr.ValidateConfig(config); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	
	fmt.Println("Configuration is valid ✓")
	return nil
}

// setConfigValue sets a configuration value and optionally persists it
func setConfigValue(cmd *cobra.Command, args []string, app *pkg.AppContainer) error {
	key := args[0]
	value := args[1]

	// Load current config
	config, err := app.ConfigMgr.LoadConfig()
	if err != nil {
		config = app.ConfigMgr.GetDefaultConfig()
	}

	// Handle special keys
	switch key {
	case "danger":
		if value == "true" || value == "1" || value == "on" {
			config.DangerMode = true
		} else {
			config.DangerMode = false
		}
	case "variant", "image":
		config.Variant = value
	case "account":
		config.Account = value
	case "session_persistence":
		if value == "true" || value == "1" || value == "on" {
			config.SessionPersistence = true
		} else {
			config.SessionPersistence = false
			// Clear session data when disabling persistence
			config.LastSessionID = ""
			config.ContainerID = ""
		}
	case "container_id":
		config.ContainerID = value
	case "last_session_id":
		config.LastSessionID = value
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	// Save the updated configuration
	if err := app.ConfigMgr.SaveConfig(config); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	app.Logger.Infof("Set %s = %s", key, value)
	return nil
}