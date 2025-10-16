package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfigCmd(t *testing.T) {
	t.Run("create config command", func(t *testing.T) {
		app := createMockApp()
		cmd := NewConfigCmd(app)

		assert.Equal(t, "config", cmd.Use)
		assert.Contains(t, cmd.Short, "configuration")
		assert.NotEmpty(t, cmd.Long)
		
		// Should have subcommands
		assert.True(t, cmd.HasSubCommands())
		
		// Check for expected subcommands
		subcommands := cmd.Commands()
		subcommandNames := make([]string, len(subcommands))
		for i, subcmd := range subcommands {
			subcommandNames[i] = subcmd.Use
		}
		
		assert.Contains(t, subcommandNames, "show")
		assert.Contains(t, subcommandNames, "validate")
		assert.Contains(t, subcommandNames, "set [key] [value]")
	})
}

func TestConfigShowCmd(t *testing.T) {
	t.Run("config show with nil app shows help", func(t *testing.T) {
		cmd := newConfigShowCmd(nil)
		
		err := cmd.RunE(cmd, []string{})
		assert.NoError(t, err) // Help returns no error
	})

	t.Run("config show command created correctly", func(t *testing.T) {
		app := createMockApp()
		cmd := newConfigShowCmd(app)
		
		assert.Equal(t, "show", cmd.Use)
		assert.Contains(t, cmd.Short, "configuration")
	})
}

func TestConfigValidateCmd(t *testing.T) {
	t.Run("config validate with nil app shows help", func(t *testing.T) {
		cmd := newConfigValidateCmd(nil)
		
		err := cmd.RunE(cmd, []string{})
		assert.NoError(t, err) // Help returns no error
	})
}

func TestConfigSetCmd(t *testing.T) {
	t.Run("config set with nil app shows help", func(t *testing.T) {
		cmd := newConfigSetCmd(nil)
		
		err := cmd.RunE(cmd, []string{})
		assert.NoError(t, err) // Help returns no error
	})

	t.Run("config set command structure", func(t *testing.T) {
		app := createMockApp()
		cmd := newConfigSetCmd(app)
		
		// Check command properties
		assert.Equal(t, "set [key] [value]", cmd.Use)
		assert.Contains(t, cmd.Short, "configuration value")
	})
}

func TestGetDisplayValueConfig(t *testing.T) {
	t.Run("get display value with empty string", func(t *testing.T) {
		result := getDisplayValue("", "default")
		assert.Equal(t, "<default>", result)
	})

	t.Run("get display value with non-empty string", func(t *testing.T) {
		result := getDisplayValue("actual", "default")
		assert.Equal(t, "actual", result)
	})
}

func TestGetCurrentDirConfig(t *testing.T) {
	t.Run("get current directory", func(t *testing.T) {
		dir := getCurrentDir()
		assert.NotEmpty(t, dir)
		assert.NotEqual(t, "<unknown>", dir)
	})
}