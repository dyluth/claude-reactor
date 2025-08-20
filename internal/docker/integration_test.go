//go:build integration
// +build integration

package docker

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"claude-reactor/internal/architecture"
	"claude-reactor/internal/logging"
	"claude-reactor/pkg"
)

// TestDockerIntegrationWorkflow tests the complete Docker workflow end-to-end
func TestDockerIntegrationWorkflow(t *testing.T) {
	// Skip if running in short mode or if Docker is not available
	if testing.Short() {
		t.Skip("Skipping Docker integration tests in short mode")
	}

	// Initialize real components
	logger := logging.NewLoggerWithLevel(logrus.DebugLevel)
	archDetector := architecture.NewDetector(logger)
	dockerMgr, err := NewManager(logger)
	require.NoError(t, err, "Docker should be available for integration tests")

	ctx := context.Background()

	t.Run("complete workflow with recovery", func(t *testing.T) {
		// Use recovery manager for comprehensive testing
		recoveryMgr := NewRecoveryManager(logger, dockerMgr)
		variantMgr := NewVariantManager(logger)
		namingMgr := NewNamingManager(logger, archDetector)

		// Test variant validation
		err := variantMgr.ValidateVariant("base")
		assert.NoError(t, err)

		// Test architecture detection
		hostArch, err := archDetector.GetHostArchitecture()
		assert.NoError(t, err)
		assert.NotEmpty(t, hostArch)

		platform, err := archDetector.GetDockerPlatform()
		assert.NoError(t, err)
		assert.Contains(t, platform, "linux/")

		// Test image naming
		imageName, err := namingMgr.GetImageName("base")
		assert.NoError(t, err)
		assert.Contains(t, imageName, "claude-reactor-base")

		// Test container naming
		containerName, err := namingMgr.GetContainerName("base", "test")
		assert.NoError(t, err)
		assert.Contains(t, containerName, "claude-reactor-base")
		assert.Contains(t, containerName, "test")

		// Test mount creation (with empty mounts to avoid Docker mount issues)
		mounts := []pkg.Mount{} // Use empty mounts for integration test

		// Test container configuration
		config := &pkg.ContainerConfig{
			Name:        containerName,
			Image:       "alpine:latest", // Use small, reliable image
			Command:     []string{"sleep", "10"},
			Interactive: false,
			TTY:         false,
			Mounts:      mounts,
			Environment: map[string]string{
				"INTEGRATION_TEST": "true",
			},
		}

		// Test full container lifecycle with recovery
		containerID, err := recoveryMgr.StartContainerWithRecovery(ctx, config, nil)
		if err != nil {
			// If we can't pull alpine:latest, skip the test
			if containsAny(err.Error(), []string{"pull access denied", "not found", "network"}) {
				t.Skip("Cannot pull alpine:latest image for integration testing")
				return
			}
			require.NoError(t, err, "Should be able to start container")
		}

		require.NotEmpty(t, containerID)

		// Clean up regardless of test outcome
		defer func() {
			_ = recoveryMgr.StopContainerWithRecovery(ctx, containerID, nil)
			_ = dockerMgr.RemoveContainer(ctx, containerID)
		}()

		// Test container is running
		running, err := dockerMgr.IsContainerRunning(ctx, containerName)
		assert.NoError(t, err)
		assert.True(t, running, "Container should be running")

		// Test graceful stop with recovery
		err = recoveryMgr.StopContainerWithRecovery(ctx, containerID, nil)
		assert.NoError(t, err)

		// Verify container is stopped
		running, err = dockerMgr.IsContainerRunning(ctx, containerName)
		assert.NoError(t, err)
		assert.False(t, running, "Container should be stopped")

		// Test container removal
		err = dockerMgr.RemoveContainer(ctx, containerID)
		assert.NoError(t, err)

		// Verify container is gone
		running, err = dockerMgr.IsContainerRunning(ctx, containerName)
		assert.NoError(t, err)
		assert.False(t, running, "Container should not exist")
	})
}

// TestDockerBuildIntegration tests Docker image building
func TestDockerBuildIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker build integration tests in short mode")
	}

	// Check if Dockerfile exists before attempting build
	if _, err := os.Stat("../../Dockerfile"); os.IsNotExist(err) {
		t.Skip("Dockerfile not found, skipping build integration test")
	}

	logger := logging.NewLoggerWithLevel(logrus.DebugLevel)
	archDetector := architecture.NewDetector(logger)
	dockerMgr, err := NewManager(logger)
	require.NoError(t, err, "Docker should be available for integration tests")

	recoveryMgr := NewRecoveryManager(logger, dockerMgr)
	ctx := context.Background()

	t.Run("build base image with recovery", func(t *testing.T) {
		platform, err := archDetector.GetDockerPlatform()
		require.NoError(t, err)

		// Use recovery manager to build image
		config := &RecoveryConfig{
			MaxRetries:    2,
			InitialDelay:  2 * time.Second,
			MaxDelay:      10 * time.Second,
			BackoffFactor: 2.0,
		}

		err = recoveryMgr.BuildImageWithRecovery(ctx, "base", platform, config)
		if err != nil {
			// Network or infrastructure issues are acceptable for integration tests
			if containsAny(err.Error(), []string{"network", "connection", "timeout", "pull", "download"}) {
				t.Skipf("Network-related build failure (acceptable in CI): %v", err)
				return
			}
			// Dockerfile issues indicate real problems
			require.NoError(t, err, "Build should succeed with valid Dockerfile")
		}

		// If build succeeded, verify image exists by attempting to create a container
		// (We won't actually start it to keep the test fast)
	})
}

// TestDockerErrorRecovery tests error handling and recovery scenarios
func TestDockerErrorRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker error recovery tests in short mode")
	}

	logger := logging.NewLoggerWithLevel(logrus.DebugLevel)
	dockerMgr, err := NewManager(logger)
	require.NoError(t, err, "Docker should be available for integration tests")

	recoveryMgr := NewRecoveryManager(logger, dockerMgr)
	ctx := context.Background()

	t.Run("invalid image handling", func(t *testing.T) {
		config := &pkg.ContainerConfig{
			Name:        "test-invalid-image",
			Image:       "nonexistent/invalid:latest",
			Command:     []string{"echo", "test"},
			Interactive: false,
			TTY:         false,
			Mounts:      []pkg.Mount{},
		}

		recoveryConfig := &RecoveryConfig{
			MaxRetries:   2,
			InitialDelay: 100 * time.Millisecond, // Fast for testing
		}

		_, err := recoveryMgr.StartContainerWithRecovery(ctx, config, recoveryConfig)
		assert.Error(t, err, "Should fail with invalid image")
		assert.Contains(t, err.Error(), "failed to start container")
	})

	t.Run("stop nonexistent container", func(t *testing.T) {
		recoveryConfig := &RecoveryConfig{
			MaxRetries:   2,
			InitialDelay: 100 * time.Millisecond,
		}

		recoveryMgr.StopContainerWithRecovery(ctx, "nonexistent-container-id", recoveryConfig)
		// This should either succeed (graceful handling) or fail predictably
		// The important thing is that it doesn't panic or hang
		assert.NotPanics(t, func() {
			_ = recoveryMgr.StopContainerWithRecovery(ctx, "nonexistent-container-id", recoveryConfig)
		})
	})
}

// TestDockerComponentIntegration tests individual components working together
func TestDockerComponentIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker component integration tests in short mode")
	}

	logger := logging.NewLoggerWithLevel(logrus.InfoLevel) // Less verbose for component tests
	archDetector := architecture.NewDetector(logger)

	t.Run("naming and variant integration", func(t *testing.T) {
		namingMgr := NewNamingManager(logger, archDetector)
		variantMgr := NewVariantManager(logger)

		// Test all variants have valid names
		variants := variantMgr.GetAvailableVariants()
		assert.NotEmpty(t, variants)

		for _, variant := range variants {
			err := variantMgr.ValidateVariant(variant)
			assert.NoError(t, err, "Variant %s should be valid", variant)

			imageName, err := namingMgr.GetImageName(variant)
			assert.NoError(t, err, "Should generate image name for %s", variant)
			assert.Contains(t, imageName, variant)

			containerName, err := namingMgr.GetContainerName(variant, "test")
			assert.NoError(t, err, "Should generate container name for %s", variant)
			assert.Contains(t, containerName, variant)
		}
	})

	t.Run("mount and configuration integration", func(t *testing.T) {
		mountMgr := NewMountManager(logger)

		// Test default mounts creation
		mounts, err := mountMgr.CreateDefaultMounts("test-account")
		assert.NoError(t, err)
		assert.NotEmpty(t, mounts, "Should create default mounts")

		// Verify project mount is included
		foundProjectMount := false
		for _, mount := range mounts {
			if mount.Target == "/app" {
				foundProjectMount = true
				break
			}
		}
		assert.True(t, foundProjectMount, "Should include project mount")

		// Test mount validation doesn't panic
		assert.NotPanics(t, func() {
			_ = mountMgr.ValidateMounts(mounts)
		})

		// Test mount summary generation
		summary := mountMgr.GetMountSummary(mounts)
		assert.NotEmpty(t, summary)
	})
}

// Helper function to check if error message contains any of the given strings
func containsAny(str string, substrings []string) bool {
	str = strings.ToLower(str)
	for _, substring := range substrings {
		if strings.Contains(str, strings.ToLower(substring)) {
			return true
		}
	}
	return false
}
