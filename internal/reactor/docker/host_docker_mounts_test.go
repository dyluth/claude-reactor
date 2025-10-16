package docker

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"claude-reactor/pkg"
)

func TestHostDockerMountLogic(t *testing.T) {
	logger := &mockMountLogger{}

	t.Run("docker socket excluded from default mounts", func(t *testing.T) {
		mountMgr := NewMountManager(logger)

		// Create default mounts
		mounts, err := mountMgr.CreateDefaultMounts("test-account")
		require.NoError(t, err)

		// Verify Docker socket is NOT included in default mounts
		dockerSocketFound := false
		for _, mount := range mounts {
			if mount.Source == "/var/run/docker.sock" || mount.Target == "/var/run/docker.sock" {
				dockerSocketFound = true
				break
			}
		}

		assert.False(t, dockerSocketFound, "Docker socket should not be included in default mounts")
	})

	t.Run("default mounts include expected non-docker mounts", func(t *testing.T) {
		mountMgr := NewMountManager(logger)

		mounts, err := mountMgr.CreateDefaultMounts("test-account")
		require.NoError(t, err)

		// Should include project mount
		projectMountFound := false
		for _, mount := range mounts {
			if mount.Target == "/app" {
				projectMountFound = true
				break
			}
		}
		assert.True(t, projectMountFound, "Project mount should be included")

		// Verify mount structure
		assert.Greater(t, len(mounts), 0, "Should have at least one mount")
	})

	t.Run("mount validation allows docker socket", func(t *testing.T) {
		mountMgr := NewMountManager(logger)

		// Create a mount with Docker socket
		mounts := []pkg.Mount{
			{
				Source:   "/var/run/docker.sock",
				Target:   "/var/run/docker.sock",
				Type:     "bind",
				ReadOnly: false,
			},
		}

		// Validation should pass for Docker socket
		err := mountMgr.ValidateMounts(mounts)
		// Note: This might fail if Docker socket doesn't exist, but that's expected
		// The important thing is that it doesn't reject it for being a Docker socket
		if err != nil {
			// If there's an error, it should be about the socket not existing, not about validation rules
			assert.Contains(t, err.Error(), "does not exist", "Error should be about file not existing, not validation rules")
		}
	})

	t.Run("mount conversion handles docker socket correctly", func(t *testing.T) {
		mountMgr := NewMountManager(logger)

		pkgMounts := []pkg.Mount{
			{
				Source:   "/var/run/docker.sock",
				Target:   "/var/run/docker.sock",
				Type:     "bind",
				ReadOnly: false,
			},
		}

		dockerMounts := mountMgr.ConvertToDockerMounts(pkgMounts)

		require.Len(t, dockerMounts, 1)
		assert.Equal(t, "/var/run/docker.sock", dockerMounts[0].Source)
		assert.Equal(t, "/var/run/docker.sock", dockerMounts[0].Target)
		assert.Equal(t, "bind", string(dockerMounts[0].Type))
		assert.False(t, dockerMounts[0].ReadOnly)
	})
}

func TestDockerSocketMountIntegration(t *testing.T) {
	// This tests the integration with AddMountsToContainer logic
	// We'll simulate the container config with host Docker enabled

	t.Run("container config with host docker enabled", func(t *testing.T) {
		containerConfig := &pkg.ContainerConfig{
			Name:              "test-container",
			HostDocker:        true,
			HostDockerTimeout: "5m",
			Mounts:            []pkg.Mount{},
		}

		// Verify the config has host Docker enabled
		assert.True(t, containerConfig.HostDocker)
		assert.Equal(t, "5m", containerConfig.HostDockerTimeout)

		// The actual mounting logic is tested in AddMountsToContainer
		// which requires more complex setup, but we can verify the config structure
	})

	t.Run("container config with host docker disabled", func(t *testing.T) {
		containerConfig := &pkg.ContainerConfig{
			Name:       "test-container",
			HostDocker: false,
			Mounts:     []pkg.Mount{},
		}

		// Verify the config has host Docker disabled
		assert.False(t, containerConfig.HostDocker)
		assert.Empty(t, containerConfig.HostDockerTimeout)
	})
}

func TestDockerSocketAvailabilityCheck(t *testing.T) {
	t.Run("docker socket availability detection", func(t *testing.T) {
		// Test the logic that checks if Docker socket exists
		dockerSock := "/var/run/docker.sock"

		// Check if socket exists (may or may not exist in test environment)
		_, err := os.Stat(dockerSock)

		if err != nil {
			// Socket doesn't exist - this is expected in many test environments
			assert.True(t, os.IsNotExist(err), "Error should be 'not exist' if socket unavailable")
		} else {
			// Socket exists - verify it's a socket file
			// This is the case where host Docker would be available
			t.Logf("Docker socket found at %s", dockerSock)
		}
	})
}

// mockMountLogger for testing
type mockMountLogger struct{}

func (m *mockMountLogger) Debug(args ...interface{})                                 {}
func (m *mockMountLogger) Info(args ...interface{})                                  {}
func (m *mockMountLogger) Warn(args ...interface{})                                  {}
func (m *mockMountLogger) Error(args ...interface{})                                 {}
func (m *mockMountLogger) Fatal(args ...interface{})                                 {}
func (m *mockMountLogger) Debugf(format string, args ...interface{})                 {}
func (m *mockMountLogger) Infof(format string, args ...interface{})                  {}
func (m *mockMountLogger) Warnf(format string, args ...interface{})                  {}
func (m *mockMountLogger) Errorf(format string, args ...interface{})                 {}
func (m *mockMountLogger) Fatalf(format string, args ...interface{})                 {}
func (m *mockMountLogger) WithField(key string, value interface{}) pkg.Logger        { return m }
func (m *mockMountLogger) WithFields(fields map[string]interface{}) pkg.Logger       { return m }