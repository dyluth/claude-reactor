package commands

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewListCmd(t *testing.T) {
	t.Run("create list command", func(t *testing.T) {
		app := createMockApp()
		cmd := NewListCmd(app)

		assert.Equal(t, "list", cmd.Use)
		assert.Contains(t, cmd.Short, "List accounts")
		assert.NotEmpty(t, cmd.Long)
	})

	t.Run("list command with nil app shows help", func(t *testing.T) {
		cmd := NewListCmd(nil)
		
		err := cmd.RunE(cmd, []string{})
		assert.NoError(t, err) // Help returns no error
	})
}

func TestListCommandFlags(t *testing.T) {
	t.Run("json flag parsing", func(t *testing.T) {
		app := createMockApp()
		cmd := NewListCmd(app)

		// Test json flag
		cmd.SetArgs([]string{"--json"})
		err := cmd.ParseFlags([]string{"--json"})
		assert.NoError(t, err)

		jsonOutput, _ := cmd.Flags().GetBool("json")
		assert.True(t, jsonOutput)
	})

	t.Run("short json flag parsing", func(t *testing.T) {
		app := createMockApp()
		cmd := NewListCmd(app)

		// Test short json flag
		cmd.SetArgs([]string{"-j"})
		err := cmd.ParseFlags([]string{"-j"})
		assert.NoError(t, err)

		jsonOutput, _ := cmd.Flags().GetBool("json")
		assert.True(t, jsonOutput)
	})
}

func TestParseProjectDirName(t *testing.T) {
	t.Run("valid project directory name", func(t *testing.T) {
		projectName, projectHash, err := parseProjectDirName("my-project-a1b2c3d4")
		
		assert.NoError(t, err)
		assert.Equal(t, "my-project", projectName)
		assert.Equal(t, "a1b2c3d4", projectHash)
	})

	t.Run("project name with multiple dashes", func(t *testing.T) {
		projectName, projectHash, err := parseProjectDirName("my-complex-project-name-f7894af8")
		
		assert.NoError(t, err)
		assert.Equal(t, "my-complex-project-name", projectName)
		assert.Equal(t, "f7894af8", projectHash)
	})

	t.Run("invalid format - no dash", func(t *testing.T) {
		_, _, err := parseProjectDirName("invalidname")
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid format")
	})

	t.Run("invalid format - hash too short", func(t *testing.T) {
		_, _, err := parseProjectDirName("project-abc123")
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid hash")
	})

	t.Run("invalid format - hash too long", func(t *testing.T) {
		_, _, err := parseProjectDirName("project-abc123def456")
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid hash")
	})
}

func TestFormatRelativeTime(t *testing.T) {
	t.Run("just now", func(t *testing.T) {
		now := time.Now()
		result := formatRelativeTime(now)
		assert.Equal(t, "just now", result)
	})

	t.Run("minutes ago", func(t *testing.T) {
		fiveMinutesAgo := time.Now().Add(-5 * time.Minute)
		result := formatRelativeTime(fiveMinutesAgo)
		assert.Equal(t, "5m ago", result)
	})

	t.Run("hours ago", func(t *testing.T) {
		twoHoursAgo := time.Now().Add(-2 * time.Hour)
		result := formatRelativeTime(twoHoursAgo)
		assert.Equal(t, "2h ago", result)
	})

	t.Run("days ago", func(t *testing.T) {
		threeDaysAgo := time.Now().Add(-3 * 24 * time.Hour)
		result := formatRelativeTime(threeDaysAgo)
		assert.Equal(t, "3d ago", result)
	})

	t.Run("weeks ago - shows date", func(t *testing.T) {
		twoWeeksAgo := time.Now().Add(-14 * 24 * time.Hour)
		result := formatRelativeTime(twoWeeksAgo)
		// Should show date format
		assert.Contains(t, result, "-")
		assert.Len(t, result, 10) // YYYY-MM-DD format
	})
}

func TestTruncate(t *testing.T) {
	t.Run("string shorter than limit", func(t *testing.T) {
		result := truncate("short", 10)
		assert.Equal(t, "short", result)
	})

	t.Run("string equal to limit", func(t *testing.T) {
		result := truncate("exactly10c", 10)
		assert.Equal(t, "exactly10c", result)
	})

	t.Run("string longer than limit", func(t *testing.T) {
		result := truncate("this is a very long string", 10)
		assert.Equal(t, "this is...", result)
		assert.Len(t, result, 10)
	})

	t.Run("string longer than limit - edge case", func(t *testing.T) {
		result := truncate("abcdefghijklmnop", 5)
		assert.Equal(t, "ab...", result)
		assert.Len(t, result, 5)
	})
}

func TestListCommandCreation(t *testing.T) {
	t.Run("list command created with flags", func(t *testing.T) {
		app := createMockApp()
		cmd := NewListCmd(app)
		
		// Check that flags exist
		jsonFlag := cmd.Flags().Lookup("json")
		assert.NotNil(t, jsonFlag)
		assert.Equal(t, "j", jsonFlag.Shorthand)
		
		// Check help content
		assert.Contains(t, cmd.Long, "account isolation")
		assert.Contains(t, cmd.Long, "container status")
	})
}

