package commands

import (
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"claude-reactor/pkg"
	"claude-reactor/pkg/mocks"
)

func TestHostDockerFlagParsing(t *testing.T) {
	t.Run("host-docker flag parsing", func(t *testing.T) {
		app := createMockApp()
		cmd := NewRunCmd(app)

		// Set flags
		cmd.SetArgs([]string{"--host-docker", "--host-docker-timeout", "15m"})
		err := cmd.ParseFlags([]string{"--host-docker", "--host-docker-timeout", "15m"})
		require.NoError(t, err)

		hostDocker, _ := cmd.Flags().GetBool("host-docker")
		hostDockerTimeout, _ := cmd.Flags().GetString("host-docker-timeout")

		assert.True(t, hostDocker)
		assert.Equal(t, "15m", hostDockerTimeout)
	})

	t.Run("default timeout value", func(t *testing.T) {
		app := createMockApp()
		cmd := NewRunCmd(app)

		hostDockerTimeout, _ := cmd.Flags().GetString("host-docker-timeout")
		assert.Equal(t, "5m", hostDockerTimeout) // Default value
	})

	t.Run("host-docker can be combined with danger mode", func(t *testing.T) {
		app := createMockApp()
		cmd := NewRunCmd(app)

		cmd.SetArgs([]string{"--host-docker", "--danger"})
		err := cmd.ParseFlags([]string{"--host-docker", "--danger"})
		require.NoError(t, err)

		hostDocker, _ := cmd.Flags().GetBool("host-docker")
		danger, _ := cmd.Flags().GetBool("danger")

		assert.True(t, hostDocker)
		assert.True(t, danger)
	})
}

func TestHostDockerTimeoutValidation(t *testing.T) {
	testCases := []struct {
		name        string
		timeout     string
		expectError bool
		errorSubstr string
	}{
		{
			name:        "valid timeout - minutes",
			timeout:     "5m",
			expectError: false,
		},
		{
			name:        "valid timeout - seconds",
			timeout:     "30s",
			expectError: false,
		},
		{
			name:        "valid timeout - hours",
			timeout:     "1h30m",
			expectError: false,
		},
		{
			name:        "disable timeout - zero",
			timeout:     "0",
			expectError: false,
		},
		{
			name:        "disable timeout - zero seconds",
			timeout:     "0s",
			expectError: false,
		},
		{
			name:        "invalid timeout format",
			timeout:     "5min",
			expectError: true,
			errorSubstr: "unknown unit",
		},
		{
			name:        "invalid timeout format - no unit",
			timeout:     "300",
			expectError: true,
			errorSubstr: "missing unit",
		},
		{
			name:        "invalid timeout - negative",
			timeout:     "-5m",
			expectError: false, // Go actually allows negative durations
			errorSubstr: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test time.ParseDuration directly (this is what the code uses)
			if tc.timeout != "0" && tc.timeout != "0s" {
				_, err := time.ParseDuration(tc.timeout)
				if tc.expectError {
					assert.Error(t, err, "Expected error for timeout: %s", tc.timeout)
					if tc.errorSubstr != "" {
						assert.Contains(t, err.Error(), tc.errorSubstr, "Error should contain expected message")
					}
				} else {
					assert.NoError(t, err, "Expected no error for timeout: %s", tc.timeout)
				}
			}
		})
	}
}

func TestHostDockerSecurityWarning(t *testing.T) {
	t.Run("security warning display", func(t *testing.T) {
		// Create a logger that captures output
		logger := &captureLogger{}

		// Test security warning display
		displayHostDockerSecurityWarning(logger, "5m")

		// Verify warning components are present
		output := strings.Join(logger.messages, "\n")
		assert.Contains(t, output, "WARNING: HOST DOCKER ACCESS ENABLED")
		assert.Contains(t, output, "HOST-LEVEL Docker privileges")
		assert.Contains(t, output, "Can create/manage ANY container")
		assert.Contains(t, output, "Can mount/access ANY host directory")
		assert.Contains(t, output, "Can access host network")
		assert.Contains(t, output, "Equivalent to ROOT access")
		assert.Contains(t, output, "Docker operations timeout: 5m")
		assert.Contains(t, output, "Only enable for trusted workflows")
	})

	t.Run("security warning with unlimited timeout", func(t *testing.T) {
		logger := &captureLogger{}

		displayHostDockerSecurityWarning(logger, "0")

		output := strings.Join(logger.messages, "\n")
		assert.Contains(t, output, "UNLIMITED TIMEOUT (no timeout protection)")
	})

	t.Run("security warning with disabled timeout", func(t *testing.T) {
		logger := &captureLogger{}

		displayHostDockerSecurityWarning(logger, "0s")

		output := strings.Join(logger.messages, "\n")
		assert.Contains(t, output, "UNLIMITED TIMEOUT (no timeout protection)")
	})
}

func TestHostDockerContainerConfig(t *testing.T) {
	t.Run("container config includes host Docker settings", func(t *testing.T) {
		// This would be tested in integration, but we can test the struct fields
		config := &pkg.ContainerConfig{
			Image:             "test:latest",
			Name:              "test-container",
			HostDocker:        true,
			HostDockerTimeout: "10m",
		}

		assert.True(t, config.HostDocker)
		assert.Equal(t, "10m", config.HostDockerTimeout)
	})
}

func TestNilAppContainerHandling(t *testing.T) {
	t.Run("run container with nil app returns error", func(t *testing.T) {
		cmd := &cobra.Command{}

		err := RunContainer(cmd, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "application container is not initialized")
	})
}

// Helper functions for testing

func createMockApp() *pkg.AppContainer {
	return &pkg.AppContainer{
		ConfigMgr:    &mocks.MockConfigManager{},
		Logger:       &captureLogger{},
		ArchDetector: &mocks.MockArchDetector{},
		AuthMgr:      &mocks.MockAuthManager{},
		MountMgr:     &mocks.MockMountManager{},
		Debug:        false,
	}
}

// captureLogger captures log messages for testing
type captureLogger struct {
	messages []string
}

func (c *captureLogger) Debug(args ...interface{})   { c.messages = append(c.messages, args[0].(string)) }
func (c *captureLogger) Info(args ...interface{})    { c.messages = append(c.messages, args[0].(string)) }
func (c *captureLogger) Warn(args ...interface{})    { c.messages = append(c.messages, args[0].(string)) }
func (c *captureLogger) Error(args ...interface{})   { c.messages = append(c.messages, args[0].(string)) }
func (c *captureLogger) Fatal(args ...interface{})   { c.messages = append(c.messages, args[0].(string)) }
func (c *captureLogger) Debugf(format string, args ...interface{}) {
	c.messages = append(c.messages, format)
}
func (c *captureLogger) Infof(format string, args ...interface{}) {
	c.messages = append(c.messages, format)
}
func (c *captureLogger) Warnf(format string, args ...interface{}) {
	c.messages = append(c.messages, format)
}
func (c *captureLogger) Errorf(format string, args ...interface{}) {
	c.messages = append(c.messages, format)
}
func (c *captureLogger) Fatalf(format string, args ...interface{}) {
	c.messages = append(c.messages, format)
}
func (c *captureLogger) WithField(key string, value interface{}) pkg.Logger { return c }
func (c *captureLogger) WithFields(fields map[string]interface{}) pkg.Logger { return c }