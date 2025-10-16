package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCleanCmd(t *testing.T) {
	t.Run("create clean command", func(t *testing.T) {
		app := createMockApp()
		cmd := NewCleanCmd(app)

		assert.Equal(t, "clean", cmd.Use)
		assert.Contains(t, cmd.Short, "Remove containers")
		assert.NotEmpty(t, cmd.Long)
	})

	t.Run("clean command with nil app shows help", func(t *testing.T) {
		cmd := NewCleanCmd(nil)
		
		// This should not panic and should show help
		err := cmd.RunE(cmd, []string{})
		assert.NoError(t, err) // Help returns no error
	})
}

func TestCleanCommandFlags(t *testing.T) {
	t.Run("flag parsing for cleanup levels", func(t *testing.T) {
		app := createMockApp()
		cmd := NewCleanCmd(app)

		// Test sessions flag
		cmd.SetArgs([]string{"--sessions"})
		err := cmd.ParseFlags([]string{"--sessions"})
		require.NoError(t, err)

		sessions, _ := cmd.Flags().GetBool("sessions")
		assert.True(t, sessions)
	})

	t.Run("flag parsing for global scope", func(t *testing.T) {
		app := createMockApp()
		cmd := NewCleanCmd(app)

		// Test global flag
		cmd.SetArgs([]string{"--global"})
		err := cmd.ParseFlags([]string{"--global"})
		require.NoError(t, err)

		global, _ := cmd.Flags().GetBool("global")
		assert.True(t, global)
	})

	t.Run("flag parsing for force mode", func(t *testing.T) {
		app := createMockApp()
		cmd := NewCleanCmd(app)

		// Test force flag
		cmd.SetArgs([]string{"--force"})
		err := cmd.ParseFlags([]string{"--force"})
		require.NoError(t, err)

		force, _ := cmd.Flags().GetBool("force")
		assert.True(t, force)
	})

	t.Run("flag parsing combination", func(t *testing.T) {
		app := createMockApp()
		cmd := NewCleanCmd(app)

		// Test multiple flags
		cmd.SetArgs([]string{"--global", "--auth", "--images", "--force"})
		err := cmd.ParseFlags([]string{"--global", "--auth", "--images", "--force"})
		require.NoError(t, err)

		global, _ := cmd.Flags().GetBool("global")
		auth, _ := cmd.Flags().GetBool("auth")
		images, _ := cmd.Flags().GetBool("images")
		force, _ := cmd.Flags().GetBool("force")

		assert.True(t, global)
		assert.True(t, auth)
		assert.True(t, images)
		assert.True(t, force)
	})
}

func TestCleanCommandHelp(t *testing.T) {
	t.Run("clean command help message", func(t *testing.T) {
		app := createMockApp()
		cmd := NewCleanCmd(app)

		// Test that help text contains expected content
		assert.Contains(t, cmd.Long, "granular cleanup levels")
		assert.Contains(t, cmd.Long, "Current project only")
		assert.Contains(t, cmd.Long, "All projects and accounts")
	})
}

func TestGetDisplayValue(t *testing.T) {
	t.Run("get display value with empty string", func(t *testing.T) {
		result := getDisplayValue("", "default")
		assert.Equal(t, "<default>", result)
	})

	t.Run("get display value with non-empty string", func(t *testing.T) {
		result := getDisplayValue("actual", "default")
		assert.Equal(t, "actual", result)
	})
}

func TestGetCurrentDir(t *testing.T) {
	t.Run("get current directory", func(t *testing.T) {
		dir := getCurrentDir()
		assert.NotEmpty(t, dir)
		assert.NotEqual(t, "unknown", dir)
	})
}