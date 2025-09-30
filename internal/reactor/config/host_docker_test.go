package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"claude-reactor/pkg"
)

func TestHostDockerConfigurationPersistence(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	manager := NewManager(&mockLogger{})

	t.Run("save and load host Docker configuration", func(t *testing.T) {
		config := &pkg.Config{
			Variant:           "go",
			HostDocker:        true,
			HostDockerTimeout: "10m",
		}

		// Save configuration
		err := manager.SaveConfig(config)
		require.NoError(t, err)

		// Load configuration
		loadedConfig, err := manager.LoadConfig()
		require.NoError(t, err)

		// Verify host Docker settings are preserved
		assert.Equal(t, true, loadedConfig.HostDocker)
		assert.Equal(t, "10m", loadedConfig.HostDockerTimeout)
		assert.Equal(t, "go", loadedConfig.Variant)
	})

	t.Run("load configuration with host Docker disabled", func(t *testing.T) {
		config := &pkg.Config{
			Variant:    "base",
			HostDocker: false,
		}

		// Save configuration
		err := manager.SaveConfig(config)
		require.NoError(t, err)

		// Load configuration
		loadedConfig, err := manager.LoadConfig()
		require.NoError(t, err)

		// Verify host Docker is disabled
		assert.Equal(t, false, loadedConfig.HostDocker)
		assert.Equal(t, "", loadedConfig.HostDockerTimeout)
	})

	t.Run("parse existing configuration file with host Docker settings", func(t *testing.T) {
		// Create a config file with host Docker settings
		configContent := `variant=go
host_docker=true
host_docker_timeout=15m
danger=true
account=test`

		err := os.WriteFile(".claude-reactor", []byte(configContent), 0644)
		require.NoError(t, err)

		// Load configuration
		config, err := manager.LoadConfig()
		require.NoError(t, err)

		// Verify all settings are loaded correctly
		assert.Equal(t, "go", config.Variant)
		assert.Equal(t, true, config.HostDocker)
		assert.Equal(t, "15m", config.HostDockerTimeout)
		assert.Equal(t, true, config.DangerMode)
		assert.Equal(t, "test", config.Account)
	})

	t.Run("configuration file without host Docker settings", func(t *testing.T) {
		// Create a config file without host Docker settings
		configContent := `variant=base
danger=false`

		err := os.WriteFile(".claude-reactor", []byte(configContent), 0644)
		require.NoError(t, err)

		// Load configuration
		config, err := manager.LoadConfig()
		require.NoError(t, err)

		// Verify host Docker settings default to false/empty
		assert.Equal(t, false, config.HostDocker)
		assert.Equal(t, "", config.HostDockerTimeout)
	})

	t.Run("save configuration preserves existing non-host-docker settings", func(t *testing.T) {
		// Create initial config
		initialConfig := &pkg.Config{
			Variant:           "base",
			Account:           "work",
			DangerMode:        true,
			SessionPersistence: true,
			LastSessionID:     "session-123",
		}

		err := manager.SaveConfig(initialConfig)
		require.NoError(t, err)

		// Update with host Docker settings
		updatedConfig := &pkg.Config{
			Variant:           "go",
			Account:           "work",
			DangerMode:        true,
			SessionPersistence: true,
			LastSessionID:     "session-123",
			HostDocker:        true,
			HostDockerTimeout: "20m",
		}

		err = manager.SaveConfig(updatedConfig)
		require.NoError(t, err)

		// Load and verify all settings are preserved
		loadedConfig, err := manager.LoadConfig()
		require.NoError(t, err)

		assert.Equal(t, "go", loadedConfig.Variant)
		assert.Equal(t, "work", loadedConfig.Account)
		assert.Equal(t, true, loadedConfig.DangerMode)
		assert.Equal(t, true, loadedConfig.SessionPersistence)
		assert.Equal(t, "session-123", loadedConfig.LastSessionID)
		assert.Equal(t, true, loadedConfig.HostDocker)
		assert.Equal(t, "20m", loadedConfig.HostDockerTimeout)
	})
}

func TestHostDockerConfigurationDefaults(t *testing.T) {
	manager := NewManager(&mockLogger{})

	t.Run("default configuration has host Docker disabled", func(t *testing.T) {
		config := manager.GetDefaultConfig()

		assert.Equal(t, false, config.HostDocker)
		assert.Equal(t, "", config.HostDockerTimeout)
	})
}

// mockLogger for testing
type mockLogger struct{}

func (m *mockLogger) Debug(args ...interface{})                                 {}
func (m *mockLogger) Info(args ...interface{})                                  {}
func (m *mockLogger) Warn(args ...interface{})                                  {}
func (m *mockLogger) Error(args ...interface{})                                 {}
func (m *mockLogger) Fatal(args ...interface{})                                 {}
func (m *mockLogger) Debugf(format string, args ...interface{})                 {}
func (m *mockLogger) Infof(format string, args ...interface{})                  {}
func (m *mockLogger) Warnf(format string, args ...interface{})                  {}
func (m *mockLogger) Errorf(format string, args ...interface{})                 {}
func (m *mockLogger) Fatalf(format string, args ...interface{})                 {}
func (m *mockLogger) WithField(key string, value interface{}) pkg.Logger        { return m }
func (m *mockLogger) WithFields(fields map[string]interface{}) pkg.Logger       { return m }