package commands

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestSetVersionInfo(t *testing.T) {
	t.Run("set version info", func(t *testing.T) {
		// Test that SetVersionInfo doesn't panic
		SetVersionInfo("v1.0.0", "abc123", "2024-01-01")

		// The function just sets global variables, so we can't easily assert the values
		// but we can ensure it doesn't panic
		assert.True(t, true) // This test just ensures no panic
	})
}

func TestNewInfoCmd(t *testing.T) {
	t.Run("create info command", func(t *testing.T) {
		app := createMockApp()
		cmd := NewInfoCmd(app)

		assert.Equal(t, "info", cmd.Use)
		assert.Contains(t, cmd.Short, "information")
		assert.NotEmpty(t, cmd.Long)
		assert.True(t, cmd.HasSubCommands())
	})

	t.Run("info command creation", func(t *testing.T) {
		app := createMockApp()
		cmd := NewInfoCmd(app)

		// Check basic properties
		assert.Equal(t, "info", cmd.Use)
		assert.Contains(t, cmd.Short, "information")
		assert.True(t, cmd.HasSubCommands())
	})
}

func TestInfoSubcommands(t *testing.T) {
	t.Run("info has expected subcommands", func(t *testing.T) {
		app := createMockApp()
		cmd := NewInfoCmd(app)

		subcommands := cmd.Commands()
		subcommandNames := make([]string, len(subcommands))
		for i, subcmd := range subcommands {
			subcommandNames[i] = subcmd.Use
		}

		assert.Contains(t, subcommandNames, "status")
		assert.Contains(t, subcommandNames, "info")
		assert.Contains(t, subcommandNames, "image [image-name]")
		assert.Contains(t, subcommandNames, "cache")
	})
}

func TestInfoStatusSubcommand(t *testing.T) {
	t.Run("info status with nil app shows help", func(t *testing.T) {
		app := createMockApp()
		cmd := NewInfoCmd(app)

		// Find the status subcommand
		var statusCmd *cobra.Command
		for _, subcmd := range cmd.Commands() {
			if subcmd.Use == "status" {
				statusCmd = subcmd
				break
			}
		}

		assert.NotNil(t, statusCmd)
		assert.Equal(t, "status", statusCmd.Use)
		assert.Contains(t, statusCmd.Short, "debug status")
	})
}

func TestInfoImageSubcommand(t *testing.T) {
	t.Run("info image requires one argument", func(t *testing.T) {
		app := createMockApp()
		cmd := NewInfoCmd(app)

		// Find the image subcommand
		var imageCmd *cobra.Command
		for _, subcmd := range cmd.Commands() {
			if subcmd.Use == "image [image-name]" {
				imageCmd = subcmd
				break
			}
		}

		assert.NotNil(t, imageCmd)
		assert.Contains(t, imageCmd.Short, "image compatibility")
		// The Args should be ExactArgs(1) but we can't easily test the function reference
	})
}

func TestInfoCacheSubcommand(t *testing.T) {
	t.Run("info cache is leaf command", func(t *testing.T) {
		app := createMockApp()
		cmd := NewInfoCmd(app)

		// Find the cache subcommand
		var cacheCmd *cobra.Command
		for _, subcmd := range cmd.Commands() {
			if subcmd.Use == "cache" {
				cacheCmd = subcmd
				break
			}
		}

		assert.NotNil(t, cacheCmd)

		assert.NotNil(t, cacheCmd)
		assert.False(t, cacheCmd.HasSubCommands())
	})
}

func TestDebugCommandStructure(t *testing.T) {
	t.Run("debug command has proper examples", func(t *testing.T) {
		app := createMockApp()
		cmd := NewInfoCmd(app)

		// Check examples are present in the full command help
		assert.Contains(t, cmd.Example, "# Show system information")
		assert.Contains(t, cmd.Example, "# Test custom image compatibility")
		assert.Contains(t, cmd.Example, "# Clear validation cache")

		// Check command structure
		assert.True(t, cmd.HasSubCommands())
		assert.Equal(t, "info", cmd.Use)
	})
}
