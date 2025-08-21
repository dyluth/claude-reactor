package devcontainer

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
	mockConfigMgr := &mocks.MockConfigManager{}

	mgr := NewManager(mockLogger, mockConfigMgr)

	assert.NotNil(t, mgr)
	assert.IsType(t, &manager{}, mgr)

	// Verify internal structure
	impl := mgr.(*manager)
	assert.Equal(t, mockLogger, impl.logger)
	assert.Equal(t, mockConfigMgr, impl.configMgr)
}

func TestManager_GenerateDevContainer(t *testing.T) {
	tests := []struct {
		name        string
		projectPath string
		config      *pkg.Config
		setupProject func(string) error
		setupMocks  func(*mocks.MockLogger, *mocks.MockConfigManager)
		expectError bool
	}{
		{
			name:        "generate for go project",
			projectPath: "",
			config:      &pkg.Config{Variant: "go"},
			setupProject: func(projectDir string) error {
				goMod := `module test-project

go 1.21
`
				return os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goMod), 0644)
			},
			setupMocks: func(mockLogger *mocks.MockLogger, mockConfigMgr *mocks.MockConfigManager) {
				mockLogger.On("Infof", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockConfigMgr.On("AutoDetectVariant", mock.AnythingOfType("string")).Return("go", nil).Maybe()
			},
			expectError: false,
		},
		{
			name:        "generate for nodejs project",
			projectPath: "",
			config:      &pkg.Config{Variant: "base"},
			setupProject: func(projectDir string) error {
				packageJson := `{"name": "test-project", "version": "1.0.0"}`
				return os.WriteFile(filepath.Join(projectDir, "package.json"), []byte(packageJson), 0644)
			},
			setupMocks: func(mockLogger *mocks.MockLogger, mockConfigMgr *mocks.MockConfigManager) {
				mockLogger.On("Infof", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockConfigMgr.On("AutoDetectVariant", mock.AnythingOfType("string")).Return("base", nil).Maybe()
			},
			expectError: false,
		},
		{
			name:        "generate for python project",
			projectPath: "",
			config:      &pkg.Config{Variant: "base"},
			setupProject: func(projectDir string) error {
				requirements := "flask>=2.0.0"
				return os.WriteFile(filepath.Join(projectDir, "requirements.txt"), []byte(requirements), 0644)
			},
			setupMocks: func(mockLogger *mocks.MockLogger, mockConfigMgr *mocks.MockConfigManager) {
				mockLogger.On("Infof", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockConfigMgr.On("AutoDetectVariant", mock.AnythingOfType("string")).Return("base", nil).Maybe()
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for this test
			tempDir, err := os.MkdirTemp("", "devcontainer-gen-*")
			assert.NoError(t, err)
			defer os.RemoveAll(tempDir)

			projectDir := filepath.Join(tempDir, "test-project")
			err = os.MkdirAll(projectDir, 0755)
			assert.NoError(t, err)

			// Setup project files
			err = tt.setupProject(projectDir)
			assert.NoError(t, err)

			// Change to project directory for empty projectPath test
			originalDir, _ := os.Getwd()
			defer os.Chdir(originalDir)
			os.Chdir(projectDir)

			// Setup mocks
			mockLogger := &mocks.MockLogger{}
			mockConfigMgr := &mocks.MockConfigManager{}
			tt.setupMocks(mockLogger, mockConfigMgr)

			mgr := &manager{
				logger:    mockLogger,
				configMgr: mockConfigMgr,
			}

			// Execute
			err = mgr.GenerateDevContainer(tt.projectPath, tt.config)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify .devcontainer directory was created
				devcontainerDir := filepath.Join(projectDir, ".devcontainer")
				assert.DirExists(t, devcontainerDir)

				// Verify devcontainer.json was created
				devcontainerJson := filepath.Join(devcontainerDir, "devcontainer.json")
				assert.FileExists(t, devcontainerJson)
			}

			// Verify mock expectations
			mockLogger.AssertExpectations(t)
			mockConfigMgr.AssertExpectations(t)
		})
	}
}

func TestManager_ValidateDevContainer(t *testing.T) {
	tests := []struct {
		name         string
		setupProject func(string) error
		expectError  bool
	}{
		{
			name: "validate existing devcontainer",
			setupProject: func(projectDir string) error {
				devcontainerDir := filepath.Join(projectDir, ".devcontainer")
				if err := os.MkdirAll(devcontainerDir, 0755); err != nil {
					return err
				}

				devcontainerJson := `{
  "name": "Test Project",
  "image": "claude-reactor-v2-base"
}
`
				return os.WriteFile(filepath.Join(devcontainerDir, "devcontainer.json"), []byte(devcontainerJson), 0644)
			},
			expectError: false,
		},
		{
			name: "validate missing devcontainer",
			setupProject: func(projectDir string) error {
				// Don't create any devcontainer files
				return nil
			},
			expectError: true,
		},
		{
			name: "validate invalid devcontainer json",
			setupProject: func(projectDir string) error {
				devcontainerDir := filepath.Join(projectDir, ".devcontainer")
				if err := os.MkdirAll(devcontainerDir, 0755); err != nil {
					return err
				}

				invalidJson := `{invalid json}`
				return os.WriteFile(filepath.Join(devcontainerDir, "devcontainer.json"), []byte(invalidJson), 0644)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for this test
			tempDir, err := os.MkdirTemp("", "devcontainer-validate-*")
			assert.NoError(t, err)
			defer os.RemoveAll(tempDir)

			projectDir := filepath.Join(tempDir, "test-project")
			err = os.MkdirAll(projectDir, 0755)
			assert.NoError(t, err)

			// Setup project files
			err = tt.setupProject(projectDir)
			assert.NoError(t, err)

			// Setup manager
			mockLogger := &mocks.MockLogger{}
			mockLogger.On("Infof", mock.AnythingOfType("string"), mock.Anything).Maybe()
			mockLogger.On("Errorf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			mockConfigMgr := &mocks.MockConfigManager{}

			mgr := &manager{
				logger:    mockLogger,
				configMgr: mockConfigMgr,
			}

			// Execute
			err = mgr.ValidateDevContainer(projectDir)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestManager_GetExtensionsForProject(t *testing.T) {
	tests := []struct {
		name        string
		projectType string
		variant     string
		expected    []string
	}{
		{
			name:        "go project extensions",
			projectType: "go",
			variant:     "go",
			expected:    []string{"golang.Go"},
		},
		{
			name:        "nodejs project extensions",
			projectType: "node",
			variant:     "base",
			expected:    []string{"ms-vscode.vscode-typescript-next", "esbenp.prettier-vscode"},
		},
		{
			name:        "python project extensions",
			projectType: "python",
			variant:     "base",
			expected:    []string{"ms-python.python", "ms-python.flake8"},
		},
		{
			name:        "rust project extensions",
			projectType: "rust",
			variant:     "full",
			expected:    []string{"rust-lang.rust-analyzer"},
		},
		{
			name:        "java project extensions",
			projectType: "java",
			variant:     "full",
			expected:    []string{"redhat.java", "vscjava.vscode-java-pack"},
		},
		{
			name:        "unknown project extensions",
			projectType: "unknown",
			variant:     "base",
			expected:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLogger := &mocks.MockLogger{}
			mockConfigMgr := &mocks.MockConfigManager{}

			mgr := &manager{
				logger:    mockLogger,
				configMgr: mockConfigMgr,
			}

			result, err := mgr.GetExtensionsForProject(tt.projectType, tt.variant)

			assert.NoError(t, err)
			// Check that all expected extensions are present (result may have additional extensions)
			for _, expected := range tt.expected {
				assert.Contains(t, result, expected, "Expected extension %s not found", expected)
			}
		})
	}
}

func TestManager_UpdateDevContainer(t *testing.T) {
	// Create temporary directory for this test
	tempDir, err := os.MkdirTemp("", "devcontainer-update-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	projectDir := filepath.Join(tempDir, "test-project")
	err = os.MkdirAll(projectDir, 0755)
	assert.NoError(t, err)

	// Create existing devcontainer
	devcontainerDir := filepath.Join(projectDir, ".devcontainer")
	err = os.MkdirAll(devcontainerDir, 0755)
	assert.NoError(t, err)

	originalJson := `{
  "name": "Original Project",
  "image": "claude-reactor-v2-base"
}
`
	err = os.WriteFile(filepath.Join(devcontainerDir, "devcontainer.json"), []byte(originalJson), 0644)
	assert.NoError(t, err)

	// Setup manager
	mockLogger := &mocks.MockLogger{}
	mockConfigMgr := &mocks.MockConfigManager{}
	mockLogger.On("Infof", mock.AnythingOfType("string"), mock.Anything).Maybe()
	mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
	mockConfigMgr.On("AutoDetectVariant", mock.AnythingOfType("string")).Return("go", nil).Maybe()

	mgr := &manager{
		logger:    mockLogger,
		configMgr: mockConfigMgr,
	}

	config := &pkg.Config{Variant: "go"}

	// Execute
	err = mgr.UpdateDevContainer(projectDir, config)

	// Verify
	assert.NoError(t, err)

	// Verify devcontainer.json was updated
	devcontainerJson := filepath.Join(devcontainerDir, "devcontainer.json")
	assert.FileExists(t, devcontainerJson)

	// Verify mock expectations
	mockLogger.AssertExpectations(t)
}

func TestManager_RemoveDevContainer(t *testing.T) {
	// Create temporary directory for this test
	tempDir, err := os.MkdirTemp("", "devcontainer-remove-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	projectDir := filepath.Join(tempDir, "test-project")
	err = os.MkdirAll(projectDir, 0755)
	assert.NoError(t, err)

	// Create devcontainer directory and files
	devcontainerDir := filepath.Join(projectDir, ".devcontainer")
	err = os.MkdirAll(devcontainerDir, 0755)
	assert.NoError(t, err)

	devcontainerJson := `{"name": "Test Project"}`
	err = os.WriteFile(filepath.Join(devcontainerDir, "devcontainer.json"), []byte(devcontainerJson), 0644)
	assert.NoError(t, err)

	// Verify it exists before removal
	assert.DirExists(t, devcontainerDir)

	// Setup manager
	mockLogger := &mocks.MockLogger{}
	mockLogger.On("Infof", mock.AnythingOfType("string"), mock.Anything).Maybe()
	mockConfigMgr := &mocks.MockConfigManager{}

	mgr := &manager{
		logger:    mockLogger,
		configMgr: mockConfigMgr,
	}

	// Execute
	err = mgr.RemoveDevContainer(projectDir)

	// Verify
	assert.NoError(t, err)

	// Verify devcontainer directory was removed
	assert.NoDirExists(t, devcontainerDir)
}