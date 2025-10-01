package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCompletionCmd(t *testing.T) {
	t.Run("create completion command", func(t *testing.T) {
		app := createMockApp()
		cmd := NewCompletionCmd(app)

		assert.Equal(t, "completion [bash|zsh|fish|powershell]", cmd.Use)
		assert.Contains(t, cmd.Short, "completion scripts")
		assert.NotEmpty(t, cmd.Long)
		assert.True(t, cmd.DisableFlagsInUseLine)
		
		// Check valid args
		expectedArgs := []string{"bash", "zsh", "fish", "powershell"}
		assert.Equal(t, expectedArgs, cmd.ValidArgs)
	})
	
	t.Run("completion command with nil app", func(t *testing.T) {
		cmd := NewCompletionCmd(nil)
		
		assert.NotNil(t, cmd)
		assert.Equal(t, "completion [bash|zsh|fish|powershell]", cmd.Use)
	})
}