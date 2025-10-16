package tests

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"claude-reactor/internal/reactor"
	"claude-reactor/internal/reactor/config"
	"claude-reactor/pkg"
)

// TestHostDockerIntegration tests the end-to-end host Docker functionality
func TestHostDockerIntegration(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	t.Run("host Docker configuration workflow", func(t *testing.T) {
		// Initialize app container
		app, err := reactor.NewAppContainer(false, false, "info")
		require.NoError(t, err)
		require.NotNil(t, app)

		// Test configuration manager with host Docker settings
		configMgr := config.NewManager(app.Logger)

		// Create config with host Docker enabled
		testConfig := configMgr.GetDefaultConfig()
		testConfig.Variant = "go"
		testConfig.HostDocker = true
		testConfig.HostDockerTimeout = "10m"

		// Save configuration
		err = configMgr.SaveConfig(testConfig)
		require.NoError(t, err)

		// Load configuration back
		loadedConfig, err := configMgr.LoadConfig()
		require.NoError(t, err)

		// Verify host Docker settings are preserved
		assert.Equal(t, true, loadedConfig.HostDocker)
		assert.Equal(t, "10m", loadedConfig.HostDockerTimeout)
		assert.Equal(t, "go", loadedConfig.Variant)

		// Verify configuration file content
		configData, err := os.ReadFile(".claude-reactor")
		require.NoError(t, err)

		configStr := string(configData)
		assert.Contains(t, configStr, "variant=go")
		assert.Contains(t, configStr, "host_docker=true")
		assert.Contains(t, configStr, "host_docker_timeout=10m")
	})

	t.Run("lazy Docker initialization with host Docker", func(t *testing.T) {
		// Initialize app container
		app, err := reactor.NewAppContainer(false, false, "info")
		require.NoError(t, err)

		// Initially Docker components should be nil
		assert.Nil(t, app.DockerMgr)
		assert.Nil(t, app.ImageValidator)

		// Try to ensure Docker components (may fail if Docker not available)
		dockerErr := reactor.EnsureDockerComponents(app)

		if dockerErr != nil {
			// Docker not available - this is expected in some test environments
			assert.Contains(t, dockerErr.Error(), "docker", "Error should be Docker-related")
			// Components should still be nil after failed initialization
			assert.Nil(t, app.DockerMgr)
			assert.Nil(t, app.ImageValidator)
		} else {
			// Docker available - components should be initialized
			assert.NotNil(t, app.DockerMgr)
			assert.NotNil(t, app.ImageValidator)
		}
	})

	t.Run("host Docker disabled by default", func(t *testing.T) {
		// Test that host Docker is disabled by default
		configMgr := config.NewManager(&mockLogger{})
		defaultConfig := configMgr.GetDefaultConfig()

		assert.False(t, defaultConfig.HostDocker)
		assert.Empty(t, defaultConfig.HostDockerTimeout)
	})

	t.Run("configuration migration compatibility", func(t *testing.T) {
		// Create an old-style config file without host Docker settings
		oldConfigContent := `variant=base
danger=true
account=test`

		err := os.WriteFile(".claude-reactor", []byte(oldConfigContent), 0644)
		require.NoError(t, err)

		// Load configuration - should handle missing host Docker fields gracefully
		configMgr := config.NewManager(&mockLogger{})
		config, err := configMgr.LoadConfig()
		require.NoError(t, err)

		// Verify existing settings are preserved
		assert.Equal(t, "base", config.Variant)
		assert.Equal(t, true, config.DangerMode)
		assert.Equal(t, "test", config.Account)

		// Verify host Docker settings default to disabled
		assert.False(t, config.HostDocker)
		assert.Empty(t, config.HostDockerTimeout)
	})

	t.Run("configuration persistence with mixed settings", func(t *testing.T) {
		configMgr := config.NewManager(&mockLogger{})

		// Create config with mix of settings
		mixedConfig := configMgr.GetDefaultConfig()
		mixedConfig.Variant = "cloud"
		mixedConfig.Account = "work"
		mixedConfig.DangerMode = false
		mixedConfig.HostDocker = true
		mixedConfig.HostDockerTimeout = "20m"
		mixedConfig.SessionPersistence = true

		// Save and reload
		err := configMgr.SaveConfig(mixedConfig)
		require.NoError(t, err)

		reloadedConfig, err := configMgr.LoadConfig()
		require.NoError(t, err)

		// Verify all settings are preserved correctly
		assert.Equal(t, "cloud", reloadedConfig.Variant)
		assert.Equal(t, "work", reloadedConfig.Account)
		assert.Equal(t, false, reloadedConfig.DangerMode)
		assert.Equal(t, true, reloadedConfig.HostDocker)
		assert.Equal(t, "20m", reloadedConfig.HostDockerTimeout)
		assert.Equal(t, true, reloadedConfig.SessionPersistence)
	})
}

// mockLogger for testing
type mockLogger struct{}

func (m *mockLogger) Debug(args ...interface{})                                    {}
func (m *mockLogger) Info(args ...interface{})                                     {}
func (m *mockLogger) Warn(args ...interface{})                                     {}
func (m *mockLogger) Error(args ...interface{})                                    {}
func (m *mockLogger) Fatal(args ...interface{})                                    {}
func (m *mockLogger) Debugf(format string, args ...interface{})                    {}
func (m *mockLogger) Infof(format string, args ...interface{})                     {}
func (m *mockLogger) Warnf(format string, args ...interface{})                     {}
func (m *mockLogger) Errorf(format string, args ...interface{})                    {}
func (m *mockLogger) Fatalf(format string, args ...interface{})                    {}
func (m *mockLogger) WithField(key string, value interface{}) pkg.Logger          { return m }
func (m *mockLogger) WithFields(fields map[string]interface{}) pkg.Logger         { return m }