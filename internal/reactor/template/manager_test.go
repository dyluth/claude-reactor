package template

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	
	"claude-reactor/pkg/mocks"
)

func TestNewManager(t *testing.T) {
	mockLogger := &mocks.MockLogger{}
	mockConfigMgr := &mocks.MockConfigManager{}
	mockDevContMgr := &mocks.MockDevContainerManager{}
	
	mgr := NewManager(mockLogger, mockConfigMgr, mockDevContMgr)
	
	assert.NotNil(t, mgr)
	assert.IsType(t, &manager{}, mgr)
	
	// Verify internal structure
	impl := mgr.(*manager)
	assert.Equal(t, mockLogger, impl.logger)
	assert.Equal(t, mockConfigMgr, impl.configMgr)
	assert.Equal(t, mockDevContMgr, impl.devContMgr)
	assert.NotEmpty(t, impl.templatesDir)
}

func TestManager_ListTemplates(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(string) error
		expectError bool
		expectCount int
	}{
		{
			name: "empty templates directory",
			setupFunc: func(dir string) error {
				return os.MkdirAll(dir, 0755)
			},
			expectError: false,
			expectCount: 9, // Built-in templates only
		},
		{
			name: "templates directory with custom template",
			setupFunc: func(dir string) error {
				if err := os.MkdirAll(dir, 0755); err != nil {
					return err
				}
				
				// First ensure builtin templates are created
				mockLogger := &mocks.MockLogger{}
				mockLogger.On("Infof", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Warnf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mgr := &manager{
					logger:       mockLogger,
					configMgr:    &mocks.MockConfigManager{},
					devContMgr:   &mocks.MockDevContainerManager{},
					templatesDir: dir,
				}
				if err := mgr.initializeBuiltinTemplates(); err != nil {
					return err
				}
				
				// Then create a custom template
				templateDir := filepath.Join(dir, "custom-template")
				if err := os.MkdirAll(templateDir, 0755); err != nil {
					return err
				}
				
				templateYaml := `name: custom-template
description: Custom test template
language: go
variant: go
files:
  - path: main.go
    content: |
      package main
      func main() {}
`
				return os.WriteFile(filepath.Join(templateDir, "template.yaml"), []byte(templateYaml), 0644)
			},
			expectError: false,
			expectCount: 10, // Built-in + 1 custom
		},
		{
			name: "templates directory does not exist",
			setupFunc: func(dir string) error {
				// Don't create the directory
				return nil
			},
			expectError: false,
			expectCount: 9, // Built-in templates only (directory will be created)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for this test
			tempDir, err := os.MkdirTemp("", "template-test-*")
			assert.NoError(t, err)
			defer os.RemoveAll(tempDir)
			
			// Setup mocks
			mockLogger := &mocks.MockLogger{}
			mockLogger.On("Infof", mock.AnythingOfType("string"), mock.Anything).Maybe()
			mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			mockLogger.On("Warnf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			mockConfigMgr := &mocks.MockConfigManager{}
			mockDevContMgr := &mocks.MockDevContainerManager{}
			
			// Create manager with custom templates directory
			mgr := &manager{
				logger:       mockLogger,
				configMgr:    mockConfigMgr,
				devContMgr:   mockDevContMgr,
				templatesDir: tempDir,
			}
			
			// Setup test scenario
			err = tt.setupFunc(tempDir)
			assert.NoError(t, err)
			
			// Execute
			templates, err := mgr.ListTemplates()
			
			// Verify
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, templates)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, templates)
				assert.Len(t, templates, tt.expectCount)
				
				// Verify built-in templates are always present
				builtinNames := []string{"go-api", "go-cli", "rust-cli", "rust-lib", "node-api", "react-app", "python-api", "python-cli", "java-spring"}
				for _, builtinName := range builtinNames {
					found := false
					for _, template := range templates {
						if template.Name == builtinName {
							found = true
							break
						}
					}
					assert.True(t, found, "Built-in template %s should be present", builtinName)
				}
			}
		})
	}
}

func TestManager_GetTemplate(t *testing.T) {
	tests := []struct {
		name         string
		templateName string
		setupFunc    func(string) error
		expectError  bool
		expectedName string
	}{
		{
			name:         "get built-in go-api template",
			templateName: "go-api",
			setupFunc:    func(dir string) error { return os.MkdirAll(dir, 0755) },
			expectError:  false,
			expectedName: "go-api",
		},
		{
			name:         "get built-in node-api template",
			templateName: "node-api",
			setupFunc:    func(dir string) error { return os.MkdirAll(dir, 0755) },
			expectError:  false,
			expectedName: "node-api",
		},
		{
			name:         "get non-existent template",
			templateName: "nonexistent",
			setupFunc:    func(dir string) error { return os.MkdirAll(dir, 0755) },
			expectError:  true,
		},
		{
			name:         "get custom template",
			templateName: "custom",
			setupFunc: func(dir string) error {
				if err := os.MkdirAll(dir, 0755); err != nil {
					return err
				}
				
				templateDir := filepath.Join(dir, "custom")
				if err := os.MkdirAll(templateDir, 0755); err != nil {
					return err
				}
				
				templateYaml := `name: custom
description: Custom test template
language: go
variant: go
files:
  - path: main.go
    content: |
      package main
      func main() {}
`
				return os.WriteFile(filepath.Join(templateDir, "template.yaml"), []byte(templateYaml), 0644)
			},
			expectError:  false,
			expectedName: "custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for this test
			tempDir, err := os.MkdirTemp("", "template-test-*")
			assert.NoError(t, err)
			defer os.RemoveAll(tempDir)
			
			// Setup mocks
			mockLogger := &mocks.MockLogger{}
			mockLogger.On("Infof", mock.AnythingOfType("string"), mock.Anything).Maybe()
			mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			mockLogger.On("Warnf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			mockConfigMgr := &mocks.MockConfigManager{}
			mockDevContMgr := &mocks.MockDevContainerManager{}
			
			// Create manager with custom templates directory
			mgr := &manager{
				logger:       mockLogger,
				configMgr:    mockConfigMgr,
				devContMgr:   mockDevContMgr,
				templatesDir: tempDir,
			}
			
			// Setup test scenario
			err = tt.setupFunc(tempDir)
			assert.NoError(t, err)
			
			// Execute
			template, err := mgr.GetTemplate(tt.templateName)
			
			// Verify
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, template)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, template)
				assert.Equal(t, tt.expectedName, template.Name)
				assert.NotEmpty(t, template.Description)
				assert.NotEmpty(t, template.Language)
				assert.NotEmpty(t, template.Variant)
				assert.NotEmpty(t, template.Files)
			}
		})
	}
}

func TestManager_CreateFromTemplate(t *testing.T) {
	tests := []struct {
		name         string
		templateName string
		projectPath  string
		variables    map[string]string
		setupMocks   func(*mocks.MockLogger, *mocks.MockConfigManager, *mocks.MockDevContainerManager)
		expectError  bool
	}{
		{
			name:         "create go project from template",
			templateName: "go-api",
			projectPath:  "test-project",
			variables: map[string]string{
				"ProjectName": "test-project",
				"Language":    "go",
			},
			setupMocks: func(mockLogger *mocks.MockLogger, mockConfigMgr *mocks.MockConfigManager, mockDevContMgr *mocks.MockDevContainerManager) {
				mockLogger.On("Infof", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockDevContMgr.On("GenerateDevContainer", mock.AnythingOfType("string"), mock.AnythingOfType("*pkg.Config")).Return(nil).Maybe()
			},
			expectError: false,
		},
		{
			name:         "create project with invalid template",
			templateName: "nonexistent",
			projectPath:  "test-project",
			variables: map[string]string{
				"ProjectName": "test-project",
				"Language":    "unknown",
			},
			setupMocks: func(mockLogger *mocks.MockLogger, mockConfigMgr *mocks.MockConfigManager, mockDevContMgr *mocks.MockDevContainerManager) {
				mockLogger.On("Errorf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for this test
			tempDir, err := os.MkdirTemp("", "template-test-*")
			assert.NoError(t, err)
			defer os.RemoveAll(tempDir)
			
			// Create project directory inside temp dir
			projectPath := filepath.Join(tempDir, tt.projectPath)
			
			// Setup mocks
			mockLogger := &mocks.MockLogger{}
			mockLogger.On("Infof", mock.AnythingOfType("string"), mock.Anything).Maybe()
			mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			mockLogger.On("Warnf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			mockConfigMgr := &mocks.MockConfigManager{}
			mockDevContMgr := &mocks.MockDevContainerManager{}
			tt.setupMocks(mockLogger, mockConfigMgr, mockDevContMgr)
			
			// Create manager with custom templates directory
			mgr := &manager{
				logger:       mockLogger,
				configMgr:    mockConfigMgr,
				devContMgr:   mockDevContMgr,
				templatesDir: filepath.Join(tempDir, "templates"),
			}
			
			// Execute
			result, err := mgr.ScaffoldProject(tt.templateName, projectPath, "test-project", tt.variables)
			_ = result // Avoid unused variable error
			
			// Verify
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				
				// Verify project files were created
				assert.DirExists(t, projectPath)
				
				// Check for expected files based on template
				if tt.templateName == "go-api" {
					assert.FileExists(t, filepath.Join(projectPath, "go.mod"))
					assert.FileExists(t, filepath.Join(projectPath, "main.go"))
				}
			}
			
			// Verify mock expectations
			mockLogger.AssertExpectations(t)
			mockConfigMgr.AssertExpectations(t)
			mockDevContMgr.AssertExpectations(t)
		})
	}
}

func TestManager_ValidateTemplate(t *testing.T) {
	tests := []struct {
		name         string
		templateDir  string
		setupFunc    func(string) error
		expectError  bool
		errorMessage string
	}{
		{
			name:        "valid template",
			templateDir: "valid-template",
			setupFunc: func(dir string) error {
				templateYaml := `name: valid-template
description: A valid test template
language: go
variant: go
files:
  - path: main.go
    content: |
      package main
      func main() {}
`
				return os.WriteFile(filepath.Join(dir, "template.yaml"), []byte(templateYaml), 0644)
			},
			expectError: false,
		},
		{
			name:        "template without yaml file",
			templateDir: "invalid-template",
			setupFunc: func(dir string) error {
				return nil // Don't create template.yaml
			},
			expectError:  true,
			errorMessage: "template.yaml not found",
		},
		{
			name:        "template with invalid yaml",
			templateDir: "invalid-yaml",
			setupFunc: func(dir string) error {
				invalidYaml := `name: invalid
description: Invalid YAML
language: go
variant: go
files:
  - name: main.go
    content: |
      invalid yaml structure
    invalid_field: 
      - nested
        - improperly
`
				return os.WriteFile(filepath.Join(dir, "template.yaml"), []byte(invalidYaml), 0644)
			},
			expectError:  true,
			errorMessage: "failed to parse template.yaml",
		},
		{
			name:        "template missing required fields",
			templateDir: "missing-fields",
			setupFunc: func(dir string) error {
				incompleteYaml := `name: incomplete
# Missing description, language, variant, and files
`
				return os.WriteFile(filepath.Join(dir, "template.yaml"), []byte(incompleteYaml), 0644)
			},
			expectError:  true,
			errorMessage: "template validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for this test
			tempDir, err := os.MkdirTemp("", "template-validate-*")
			assert.NoError(t, err)
			defer os.RemoveAll(tempDir)
			
			templateDir := filepath.Join(tempDir, tt.templateDir)
			err = os.MkdirAll(templateDir, 0755)
			assert.NoError(t, err)
			
			// Setup test scenario
			err = tt.setupFunc(templateDir)
			assert.NoError(t, err)
			
			// Setup mocks
			mockLogger := &mocks.MockLogger{}
			mockLogger.On("Infof", mock.AnythingOfType("string"), mock.Anything).Maybe()
			mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			mockLogger.On("Warnf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			mockConfigMgr := &mocks.MockConfigManager{}
			mockDevContMgr := &mocks.MockDevContainerManager{}
			
			mgr := &manager{
				logger:       mockLogger,
				configMgr:    mockConfigMgr,
				devContMgr:   mockDevContMgr,
				templatesDir: tempDir,
			}
			
			// First load the template, then validate it
			template, err := mgr.GetTemplate(tt.templateDir)
			if err != nil && tt.expectError {
				assert.Error(t, err)
				return // Expected error in loading
			}
			assert.NoError(t, err)
			
			// Execute validation
			err = mgr.ValidateTemplate(template)
			
			// Verify
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestManager_InstallTemplate(t *testing.T) {
	tests := []struct {
		name        string
		templateURL string
		setupMocks  func(*mocks.MockLogger)
		expectError bool
	}{
		{
			name:        "install template from valid path",
			templateURL: "/valid/template/path",
			setupMocks: func(mockLogger *mocks.MockLogger) {
				mockLogger.On("Infof", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Errorf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: true, // Will fail because path doesn't exist, but tests the flow
		},
		{
			name:        "install template from empty path",
			templateURL: "",
			setupMocks: func(mockLogger *mocks.MockLogger) {
				mockLogger.On("Errorf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for this test
			tempDir, err := os.MkdirTemp("", "template-install-*")
			assert.NoError(t, err)
			defer os.RemoveAll(tempDir)
			
			// Setup mocks
			mockLogger := &mocks.MockLogger{}
			mockLogger.On("Infof", mock.AnythingOfType("string"), mock.Anything).Maybe()
			mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			mockLogger.On("Warnf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			mockConfigMgr := &mocks.MockConfigManager{}
			mockDevContMgr := &mocks.MockDevContainerManager{}
			tt.setupMocks(mockLogger)
			
			mgr := &manager{
				logger:       mockLogger,
				configMgr:    mockConfigMgr,
				devContMgr:   mockDevContMgr,
				templatesDir: tempDir,
			}
			
			// Execute
			err = mgr.InstallTemplate(tt.templateURL)
			
			// Verify
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			
			mockLogger.AssertExpectations(t)
		})
	}
}

func TestManager_UninstallTemplate(t *testing.T) {
	tests := []struct {
		name         string
		templateName string
		setupFunc    func(string) error
		setupMocks   func(*mocks.MockLogger)
		expectError  bool
	}{
		{
			name:         "uninstall built-in template",
			templateName: "go-api",
			setupFunc:    func(dir string) error { return os.MkdirAll(dir, 0755) },
			setupMocks: func(mockLogger *mocks.MockLogger) {
				mockLogger.On("Errorf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: true, // Cannot uninstall built-in templates
		},
		{
			name:         "uninstall custom template",
			templateName: "custom",
			setupFunc: func(dir string) error {
				templateDir := filepath.Join(dir, "custom")
				if err := os.MkdirAll(templateDir, 0755); err != nil {
					return err
				}
				
				templateYaml := `name: custom
description: Custom template to uninstall
language: go
variant: go
files:
  - path: main.go
    content: package main
`
				return os.WriteFile(filepath.Join(templateDir, "template.yaml"), []byte(templateYaml), 0644)
			},
			setupMocks: func(mockLogger *mocks.MockLogger) {
				mockLogger.On("Infof", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: false,
		},
		{
			name:         "uninstall non-existent template",
			templateName: "nonexistent",
			setupFunc:    func(dir string) error { return os.MkdirAll(dir, 0755) },
			setupMocks: func(mockLogger *mocks.MockLogger) {
				mockLogger.On("Errorf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for this test
			tempDir, err := os.MkdirTemp("", "template-uninstall-*")
			assert.NoError(t, err)
			defer os.RemoveAll(tempDir)
			
			// Setup test scenario
			err = tt.setupFunc(tempDir)
			assert.NoError(t, err)
			
			// Setup mocks
			mockLogger := &mocks.MockLogger{}
			mockLogger.On("Infof", mock.AnythingOfType("string"), mock.Anything).Maybe()
			mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			mockLogger.On("Warnf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			mockConfigMgr := &mocks.MockConfigManager{}
			mockDevContMgr := &mocks.MockDevContainerManager{}
			tt.setupMocks(mockLogger)
			
			mgr := &manager{
				logger:       mockLogger,
				configMgr:    mockConfigMgr,
				devContMgr:   mockDevContMgr,
				templatesDir: tempDir,
			}
			
			// Execute
			err = mgr.UninstallTemplate(tt.templateName)
			
			// Verify
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Verify template directory was removed
				assert.NoDirExists(t, filepath.Join(tempDir, tt.templateName))
			}
			
			mockLogger.AssertExpectations(t)
		})
	}
}