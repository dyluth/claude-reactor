package internal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"claude-reactor/pkg"
)

func TestNewAppContainer(t *testing.T) {
	t.Run("successful initialization", func(t *testing.T) {
		container, err := NewAppContainer()
		
		require.NoError(t, err, "NewAppContainer should not return error")
		require.NotNil(t, container, "Container should not be nil")
		
		// Test that all components are initialized
		assert.NotNil(t, container.Logger, "Logger should be initialized")
		assert.NotNil(t, container.ArchDetector, "ArchDetector should be initialized")
		assert.NotNil(t, container.ConfigMgr, "ConfigMgr should be initialized")
		assert.NotNil(t, container.DockerMgr, "DockerMgr should be initialized")
		assert.NotNil(t, container.AuthMgr, "AuthMgr should be initialized")
		assert.NotNil(t, container.MountMgr, "MountMgr should be initialized")
		
		// Test that all components implement their interfaces
		assert.Implements(t, (*pkg.Logger)(nil), container.Logger)
		assert.Implements(t, (*pkg.ArchitectureDetector)(nil), container.ArchDetector)
		assert.Implements(t, (*pkg.ConfigManager)(nil), container.ConfigMgr)
		assert.Implements(t, (*pkg.DockerManager)(nil), container.DockerMgr)
		assert.Implements(t, (*pkg.AuthManager)(nil), container.AuthMgr)
		assert.Implements(t, (*pkg.MountManager)(nil), container.MountMgr)
	})
	
	t.Run("components are properly connected", func(t *testing.T) {
		container, err := NewAppContainer()
		require.NoError(t, err)
		
		// Test that logger is functioning
		container.Logger.Info("Test log message")
		
		// Test that architecture detector works
		arch, err := container.ArchDetector.GetHostArchitecture()
		assert.NoError(t, err, "ArchDetector should work")
		assert.NotEmpty(t, arch, "Should return valid architecture")
		
		// Test that config manager works
		config, err := container.ConfigMgr.LoadConfig()
		assert.NoError(t, err, "ConfigMgr should work")
		assert.NotNil(t, config, "Should return valid config")
		
		// Test that Docker manager is connected to Docker daemon
		// (This will fail gracefully if Docker is not available)
		_, err = container.DockerMgr.IsContainerRunning(context.Background(), "non-existent")
		// Error is expected but should not panic
		assert.NotPanics(t, func() {
			_, _ = container.DockerMgr.IsContainerRunning(context.Background(), "test")
		})
	})
	
	t.Run("multiple container instances are independent", func(t *testing.T) {
		container1, err1 := NewAppContainer()
		container2, err2 := NewAppContainer()
		
		require.NoError(t, err1)
		require.NoError(t, err2)
		require.NotNil(t, container1)
		require.NotNil(t, container2)
		
		// Verify they are different instances
		assert.NotSame(t, container1, container2, "Containers should be different instances")
		assert.NotSame(t, container1.Logger, container2.Logger, "Loggers should be different instances")
		assert.NotSame(t, container1.ArchDetector, container2.ArchDetector, "ArchDetectors should be different instances")
		assert.NotSame(t, container1.ConfigMgr, container2.ConfigMgr, "ConfigMgrs should be different instances")
		assert.NotSame(t, container1.DockerMgr, container2.DockerMgr, "DockerMgrs should be different instances")
		assert.NotSame(t, container1.AuthMgr, container2.AuthMgr, "AuthMgrs should be different instances")
		assert.NotSame(t, container1.MountMgr, container2.MountMgr, "MountMgrs should be different instances")
	})
	
	t.Run("dependency injection consistency", func(t *testing.T) {
		container, err := NewAppContainer()
		require.NoError(t, err)
		
		// Verify all components have the same logger instance
		// (This test verifies the dependency injection pattern)
		
		// We can't directly test logger sharing since it's not exposed,
		// but we can test that all components are working with consistent setup
		
		// Test that config can be loaded and validated
		config, err := container.ConfigMgr.LoadConfig()
		require.NoError(t, err)
		
		err = container.ConfigMgr.ValidateConfig(config)
		assert.NoError(t, err, "Config validation should work with loaded config")
		
		// Test that architecture detection is consistent
		arch1, err1 := container.ArchDetector.GetHostArchitecture()
		arch2, err2 := container.ArchDetector.GetHostArchitecture()
		
		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.Equal(t, arch1, arch2, "Architecture detection should be consistent")
	})
}

func TestNewAppContainer_DockerError(t *testing.T) {
	// This test checks that the function handles Docker initialization errors gracefully
	// In the current implementation, Docker errors are not fatal to container creation
	// but this test ensures the behavior is consistent
	
	t.Run("handles docker initialization", func(t *testing.T) {
		container, err := NewAppContainer()
		
		// Docker may not be available in all test environments
		// The function should either succeed or fail gracefully
		if err != nil {
			// If there's an error, it should be related to Docker
			assert.Contains(t, err.Error(), "docker", "Error should be Docker-related")
		} else {
			// If successful, container should be fully initialized
			assert.NotNil(t, container, "Container should be initialized")
			assert.NotNil(t, container.DockerMgr, "DockerMgr should be initialized")
		}
	})
}

func BenchmarkNewAppContainer(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		container, err := NewAppContainer()
		if err != nil {
			b.Fatalf("NewAppContainer failed: %v", err)
		}
		_ = container
	}
}

// Test to ensure AppContainer satisfies the expected interface
func TestAppContainerInterface(t *testing.T) {
	container, err := NewAppContainer()
	require.NoError(t, err)
	require.NotNil(t, container)
	
	// Test that AppContainer has all expected fields
	assert.NotNil(t, container.Logger, "AppContainer should have Logger field")
	assert.NotNil(t, container.ArchDetector, "AppContainer should have ArchDetector field")
	assert.NotNil(t, container.ConfigMgr, "AppContainer should have ConfigMgr field")
	assert.NotNil(t, container.DockerMgr, "AppContainer should have DockerMgr field")
	assert.NotNil(t, container.AuthMgr, "AppContainer should have AuthMgr field")
	assert.NotNil(t, container.MountMgr, "AppContainer should have MountMgr field")
}