package mount

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"claude-reactor/pkg"
	"claude-reactor/pkg/mocks"
)

func TestNewManager(t *testing.T) {
	mockLogger := &mocks.MockLogger{}

	mgr := NewManager(mockLogger)

	assert.NotNil(t, mgr)
	assert.IsType(t, &manager{}, mgr)

	// Verify internal structure
	impl := mgr.(*manager)
	assert.Equal(t, mockLogger, impl.logger)
}

func TestManager_ValidateMountPath(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		setupPath    func() (string, func(), error)
		expectError  bool
		errorMessage string
	}{
		{
			name: "valid existing directory",
			path: "/valid/path",
			setupPath: func() (string, func(), error) {
				tempDir, err := os.MkdirTemp("", "mount-test-*")
				if err != nil {
					return "", nil, err
				}
				return tempDir, func() { os.RemoveAll(tempDir) }, nil
			},
			expectError: false,
		},
		{
			name: "valid existing file",
			path: "/valid/file.txt",
			setupPath: func() (string, func(), error) {
				tempFile, err := os.CreateTemp("", "mount-test-*.txt")
				if err != nil {
					return "", nil, err
				}
				tempFile.Close()
				return tempFile.Name(), func() { os.Remove(tempFile.Name()) }, nil
			},
			expectError: false,
		},
		{
			name:         "non-existent path",
			path:         "/non/existent/path",
			setupPath:    func() (string, func(), error) { return "/non/existent/path", func() {}, nil },
			expectError:  true,
			errorMessage: "path does not exist",
		},
		{
			name:         "empty path",
			path:         "",
			setupPath:    func() (string, func(), error) { return "", func() {}, nil },
			expectError:  true,
			errorMessage: "path cannot be empty",
		},
		{
			name:         "relative path",
			path:         "relative/path",
			setupPath:    func() (string, func(), error) { return "relative/path", func() {}, nil },
			expectError:  true,
			errorMessage: "path must be absolute",
		},
		{
			name: "home directory expansion",
			path: "~/Documents",
			setupPath: func() (string, func(), error) {
				home, err := os.UserHomeDir()
				if err != nil {
					return "", nil, err
				}
				docsPath := filepath.Join(home, "Documents")
				// Create Documents directory for the test
				if err := os.MkdirAll(docsPath, 0755); err != nil {
					return "", nil, err
				}
				return docsPath, func() { os.RemoveAll(docsPath) }, nil
			},
			expectError: false, // Should expand and validate
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup path if needed
			actualPath, cleanup, err := tt.setupPath()
			if err != nil {
				t.Skip("Failed to setup test path:", err)
			}
			defer cleanup()

			// Setup mocks
			mockLogger := &mocks.MockLogger{}
			mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()

			mgr := &manager{
				logger: mockLogger,
			}

			// Execute
			var result string
			if tt.name == "home directory expansion" {
				result, err = mgr.ValidateMountPath(tt.path)
			} else {
				result, err = mgr.ValidateMountPath(actualPath)
			}

			// Verify
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, result)
				assert.True(t, filepath.IsAbs(result), "Result should be absolute path")
			}

			// Verify mock expectations
			mockLogger.AssertExpectations(t)
		})
	}
}

func TestManager_AddMountToConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *pkg.ContainerConfig
		sourcePath  string
		targetPath  string
		setupMocks  func(*mocks.MockLogger)
		expectError bool
	}{
		{
			name: "add new mount to empty config",
			config: &pkg.ContainerConfig{
				Image:       "test-image",
				Mounts:      []pkg.Mount{},
			},
			sourcePath: "/tmp",
			targetPath: "/container/path",
			setupMocks: func(mockLogger *mocks.MockLogger) {
				mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: false,
		},
		{
			name: "add mount to existing mounts",
			config: &pkg.ContainerConfig{
				Image:       "test-image",
				Mounts: []pkg.Mount{
					{Source: "/tmp", Target: "/existing/target", Type: "bind"},
				},
			},
			sourcePath: "/usr",
			targetPath: "/new/target",
			setupMocks: func(mockLogger *mocks.MockLogger) {
				mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: false,
		},
		{
			name: "add duplicate mount",
			config: &pkg.ContainerConfig{
				Image:       "test-image",
				Mounts: []pkg.Mount{
					{Source: "/tmp", Target: "/duplicate/target", Type: "bind"},
				},
			},
			sourcePath: "/tmp",
			targetPath: "/duplicate/target",
			setupMocks: func(mockLogger *mocks.MockLogger) {
				mockLogger.On("Warnf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: false, // Should handle gracefully
		},
		{
			name:        "nil config",
			config:      nil,
			sourcePath:  "/tmp",
			targetPath:  "/container/path",
			setupMocks:  func(mockLogger *mocks.MockLogger) {},
			expectError: true,
		},
		{
			name: "empty source path",
			config: &pkg.ContainerConfig{
				Image:       "test-image",
				Mounts:      []pkg.Mount{},
			},
			sourcePath: "",
			targetPath: "/container/path",
			setupMocks: func(mockLogger *mocks.MockLogger) {
				mockLogger.On("Errorf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: true,
		},
		{
			name: "empty target path",
			config: &pkg.ContainerConfig{
				Image:       "test-image",
				Mounts:      []pkg.Mount{},
			},
			sourcePath: "/tmp",
			targetPath: "",
			setupMocks: func(mockLogger *mocks.MockLogger) {
				mockLogger.On("Errorf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockLogger := &mocks.MockLogger{}
			tt.setupMocks(mockLogger)

			mgr := &manager{
				logger: mockLogger,
			}

			// Store original mount count
			var originalMountCount int
			if tt.config != nil {
				originalMountCount = len(tt.config.Mounts)
			}

			// Execute
			err := mgr.AddMountToConfig(tt.config, tt.sourcePath, tt.targetPath)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.config != nil {
					// Check if mount was added (unless it's a duplicate)
					if tt.name == "add duplicate mount" {
						// Mount count should remain the same for duplicates
						assert.Equal(t, originalMountCount, len(tt.config.Mounts))
					} else {
						// Mount should be added
						assert.Len(t, tt.config.Mounts, originalMountCount+1)
						
						// Verify the new mount
						newMount := tt.config.Mounts[len(tt.config.Mounts)-1]
						assert.Equal(t, tt.sourcePath, newMount.Source)
						assert.Equal(t, tt.targetPath, newMount.Target)
						assert.Equal(t, "bind", newMount.Type)
					}
				}
			}

			// Verify mock expectations
			mockLogger.AssertExpectations(t)
		})
	}
}

func TestManager_GetMountSummary(t *testing.T) {
	tests := []struct {
		name     string
		mounts   []pkg.Mount
		expected string
	}{
		{
			name:     "empty mounts",
			mounts:   []pkg.Mount{},
			expected: "No mounts configured",
		},
		{
			name: "single mount",
			mounts: []pkg.Mount{
				{Source: "/host/path", Target: "/container/path", Type: "bind"},
			},
			expected: "1 mount: /host/path -> /container/path",
		},
		{
			name: "multiple mounts",
			mounts: []pkg.Mount{
				{Source: "/host/path1", Target: "/container/path1", Type: "bind"},
				{Source: "/host/path2", Target: "/container/path2", Type: "bind"},
				{Source: "/host/path3", Target: "/container/path3", Type: "bind"},
			},
			expected: "3 mounts: /host/path1 -> /container/path1, /host/path2 -> /container/path2, /host/path3 -> /container/path3",
		},
		{
			name: "mounts with different types",
			mounts: []pkg.Mount{
				{Source: "/host/path1", Target: "/container/path1", Type: "bind"},
				{Source: "volume-name", Target: "/container/path2", Type: "volume"},
			},
			expected: "2 mounts: /host/path1 -> /container/path1, volume-name -> /container/path2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLogger := &mocks.MockLogger{}

			mgr := &manager{
				logger: mockLogger,
			}

			result := mgr.GetMountSummary(tt.mounts)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestManager_UpdateMountSettings(t *testing.T) {
	tests := []struct {
		name        string
		mountPaths  []string
		setupMocks  func(*mocks.MockLogger)
		expectError bool
	}{
		{
			name:       "update with valid mount paths",
			mountPaths: []string{"/path1", "/path2"},
			setupMocks: func(mockLogger *mocks.MockLogger) {
				mockLogger.On("Infof", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: false,
		},
		{
			name:       "update with no mount paths",
			mountPaths: []string{},
			setupMocks: func(mockLogger *mocks.MockLogger) {
				mockLogger.On("Debug", mock.AnythingOfType("string")).Maybe()
			},
			expectError: false,
		},
		{
			name:       "update with nil mount paths",
			mountPaths: nil,
			setupMocks: func(mockLogger *mocks.MockLogger) {
				mockLogger.On("Debug", mock.AnythingOfType("string")).Maybe()
			},
			expectError: false,
		},
		{
			name:       "update with mixed valid and empty paths",
			mountPaths: []string{"/valid/path", "", "/another/path"},
			setupMocks: func(mockLogger *mocks.MockLogger) {
				mockLogger.On("Infof", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Warnf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: false, // Should filter out empty paths
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockLogger := &mocks.MockLogger{}
			tt.setupMocks(mockLogger)

			mgr := &manager{
				logger: mockLogger,
			}

			// Execute
			err := mgr.UpdateMountSettings(tt.mountPaths)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify mock expectations
			mockLogger.AssertExpectations(t)
		})
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected func() string
	}{
		{
			name: "expand tilde to home directory",
			path: "~/Documents",
			expected: func() string {
				home, _ := os.UserHomeDir()
				return filepath.Join(home, "Documents")
			},
		},
		{
			name: "expand tilde alone",
			path: "~",
			expected: func() string {
				home, _ := os.UserHomeDir()
				return home
			},
		},
		{
			name:     "absolute path unchanged",
			path:     "/absolute/path",
			expected: func() string { return "/absolute/path" },
		},
		{
			name:     "relative path unchanged",
			path:     "relative/path",
			expected: func() string { return "relative/path" },
		},
		{
			name:     "empty path unchanged",
			path:     "",
			expected: func() string { return "" },
		},
		{
			name:     "path with tilde in middle unchanged",
			path:     "/path/~middle/path",
			expected: func() string { return "/path/~middle/path" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandPath(tt.path)
			expected := tt.expected()
			assert.Equal(t, expected, result)
		})
	}
}

func TestIsAbsolutePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "absolute unix path",
			path:     "/absolute/unix/path",
			expected: true,
		},
		{
			name:     "relative path",
			path:     "relative/path",
			expected: false,
		},
		{
			name:     "current directory",
			path:     "./current",
			expected: false,
		},
		{
			name:     "parent directory",
			path:     "../parent",
			expected: false,
		},
		{
			name:     "empty path",
			path:     "",
			expected: false,
		},
		{
			name:     "root path",
			path:     "/",
			expected: true,
		},
		{
			name:     "home path expanded",
			path:     "~/Documents",
			expected: false, // Before expansion
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAbsolutePath(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMountExists(t *testing.T) {
	tests := []struct {
		name       string
		mounts     []pkg.Mount
		sourcePath string
		targetPath string
		expected   bool
	}{
		{
			name: "mount exists with exact match",
			mounts: []pkg.Mount{
				{Source: "/host/path", Target: "/container/path", Type: "bind"},
				{Source: "/other/path", Target: "/other/target", Type: "bind"},
			},
			sourcePath: "/host/path",
			targetPath: "/container/path",
			expected:   true,
		},
		{
			name: "mount does not exist",
			mounts: []pkg.Mount{
				{Source: "/host/path", Target: "/container/path", Type: "bind"},
			},
			sourcePath: "/different/path",
			targetPath: "/different/target",
			expected:   false,
		},
		{
			name: "source matches but target differs",
			mounts: []pkg.Mount{
				{Source: "/host/path", Target: "/container/path", Type: "bind"},
			},
			sourcePath: "/host/path",
			targetPath: "/different/target",
			expected:   false,
		},
		{
			name: "target matches but source differs",
			mounts: []pkg.Mount{
				{Source: "/host/path", Target: "/container/path", Type: "bind"},
			},
			sourcePath: "/different/source",
			targetPath: "/container/path",
			expected:   false,
		},
		{
			name:       "empty mounts list",
			mounts:     []pkg.Mount{},
			sourcePath: "/tmp",
			targetPath: "/container/path",
			expected:   false,
		},
		{
			name:       "nil mounts list",
			mounts:     nil,
			sourcePath: "/tmp",
			targetPath: "/container/path",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mountExists(tt.mounts, tt.sourcePath, tt.targetPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "clean simple path",
			path:     "/simple/path",
			expected: "/simple/path",
		},
		{
			name:     "clean path with double slashes",
			path:     "/path//with/double//slashes",
			expected: "/path/with/double/slashes",
		},
		{
			name:     "clean path with current directory",
			path:     "/path/./with/current",
			expected: "/path/with/current",
		},
		{
			name:     "clean path with parent directory",
			path:     "/path/parent/../with/parent",
			expected: "/path/with/parent",
		},
		{
			name:     "clean complex path",
			path:     "/path/./complex/../path//with/everything",
			expected: "/path/path/with/everything",
		},
		{
			name:     "clean relative path",
			path:     "relative/./path/../clean",
			expected: "relative/clean",
		},
		{
			name:     "clean empty path",
			path:     "",
			expected: ".",
		},
		{
			name:     "clean root path",
			path:     "/",
			expected: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizePath(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}