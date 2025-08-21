package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"claude-reactor/pkg"
)

// MockLogger for testing configuration manager
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(args ...interface{}) {
	m.Called(args)
}

func (m *MockLogger) Info(args ...interface{}) {
	m.Called(args)
}

func (m *MockLogger) Warn(args ...interface{}) {
	m.Called(args)
}

func (m *MockLogger) Error(args ...interface{}) {
	m.Called(args)
}

func (m *MockLogger) Fatal(args ...interface{}) {
	m.Called(args)
}

func (m *MockLogger) Debugf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) Infof(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) Warnf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) Errorf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) Fatalf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockLogger) WithField(key string, value interface{}) pkg.Logger {
	args := m.Called(key, value)
	return args.Get(0).(pkg.Logger)
}

func (m *MockLogger) WithFields(fields map[string]interface{}) pkg.Logger {
	args := m.Called(fields)
	return args.Get(0).(pkg.Logger)
}

func TestNewManager(t *testing.T) {
	mockLogger := &MockLogger{}
	
	manager := NewManager(mockLogger)
	
	assert.NotNil(t, manager, "Manager should not be nil")
	assert.Implements(t, (*pkg.ConfigManager)(nil), manager, "Manager should implement ConfigManager interface")
}

func TestManager_LoadConfig(t *testing.T) {
	mockLogger := &MockLogger{}
	mockLogger.On("Debug", mock.Anything).Maybe()
	
	manager := NewManager(mockLogger)
	
	config, err := manager.LoadConfig()
	
	assert.NoError(t, err, "LoadConfig should not error")
	assert.NotNil(t, config, "Config should not be nil")
	assert.Equal(t, "base", config.Variant, "Default variant should be base")
	assert.Empty(t, config.Account, "Default account should be empty")
	assert.False(t, config.DangerMode, "Default danger mode should be false")
	assert.NotNil(t, config.Metadata, "Metadata should be initialized")
}

func TestManager_SaveConfig(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	
	err := os.Chdir(tempDir)
	assert.NoError(t, err, "Should be able to change to temp directory")
	
	mockLogger := &MockLogger{}
	mockLogger.On("Infof", mock.AnythingOfType("string"), mock.Anything).Maybe()
	
	manager := NewManager(mockLogger)
	
	config := &pkg.Config{
		Variant:     "go",
		Account:     "test",
		DangerMode:  true,
		ProjectPath: "/test/path",
		Metadata:    map[string]string{"key": "value"},
	}
	
	err = manager.SaveConfig(config)
	assert.NoError(t, err, "SaveConfig should not error")
	
	// Verify file was created
	assert.FileExists(t, ".claude-reactor", "Config file should be created")
	
	// Verify file contents
	content, err := os.ReadFile(".claude-reactor")
	assert.NoError(t, err, "Should be able to read config file")
	
	contentStr := string(content)
	assert.Contains(t, contentStr, "variant=go", "Should contain variant setting")
	assert.Contains(t, contentStr, "account=test", "Should contain account setting")
	assert.Contains(t, contentStr, "danger=true", "Should contain danger mode setting")
}

func TestManager_SaveConfig_Minimal(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	
	err := os.Chdir(tempDir)
	assert.NoError(t, err, "Should be able to change to temp directory")
	
	mockLogger := &MockLogger{}
	mockLogger.On("Infof", mock.AnythingOfType("string"), mock.Anything).Maybe()
	
	manager := NewManager(mockLogger)
	
	config := &pkg.Config{
		Variant:     "base",
		Account:     "",
		DangerMode:  false,
		ProjectPath: "",
		Metadata:    make(map[string]string),
	}
	
	err = manager.SaveConfig(config)
	assert.NoError(t, err, "SaveConfig should not error")
	
	// Verify file contents contain only variant
	content, err := os.ReadFile(".claude-reactor")
	assert.NoError(t, err, "Should be able to read config file")
	
	contentStr := string(content)
	assert.Contains(t, contentStr, "variant=base", "Should contain variant setting")
	assert.NotContains(t, contentStr, "account=", "Should not contain empty account")
	assert.NotContains(t, contentStr, "danger=", "Should not contain false danger mode")
}

func TestManager_ValidateConfig(t *testing.T) {
	mockLogger := &MockLogger{}
	manager := NewManager(mockLogger)
	
	t.Run("valid config", func(t *testing.T) {
		config := &pkg.Config{
			Variant:     "go",
			Account:     "test",
			DangerMode:  true,
			ProjectPath: "/test/path",
			Metadata:    map[string]string{},
		}
		
		err := manager.ValidateConfig(config)
		assert.NoError(t, err, "Valid config should not error")
	})
	
	t.Run("nil config", func(t *testing.T) {
		err := manager.ValidateConfig(nil)
		assert.Error(t, err, "Nil config should error")
		assert.Contains(t, err.Error(), "cannot be nil", "Error should mention nil")
	})
	
	t.Run("invalid variant", func(t *testing.T) {
		config := &pkg.Config{
			Variant: ".invalid-image-name.",  // Actually invalid Docker image name
		}
		
		err := manager.ValidateConfig(config)
		assert.Error(t, err, "Invalid variant should error")
		assert.Contains(t, err.Error(), "invalid image name format", "Error should mention invalid image name")
	})
	
	t.Run("misleading variant names are valid", func(t *testing.T) {
		// Test case that was causing the original issue - "invalid" is actually a valid Docker image name
		config := &pkg.Config{
			Variant: "invalid",  // This is actually valid as a Docker image name
		}
		
		err := manager.ValidateConfig(config)
		assert.NoError(t, err, "Simple names like 'invalid' are valid Docker image names")
	})
	
	t.Run("all valid variants", func(t *testing.T) {
		validVariants := []string{"base", "go", "full", "cloud", "k8s"}
		
		for _, variant := range validVariants {
			config := &pkg.Config{
				Variant: variant,
			}
			
			err := manager.ValidateConfig(config)
			assert.NoError(t, err, "Variant %s should be valid", variant)
		}
	})
}

func TestManager_GetDefaultConfig(t *testing.T) {
	mockLogger := &MockLogger{}
	mockLogger.On("Debug", mock.Anything).Maybe()
	
	manager := NewManager(mockLogger)
	
	config := manager.GetDefaultConfig()
	
	assert.NotNil(t, config, "Default config should not be nil")
	assert.NotEmpty(t, config.Variant, "Default variant should not be empty")
	assert.Empty(t, config.Account, "Default account should be empty")
	assert.False(t, config.DangerMode, "Default danger mode should be false")
	assert.Empty(t, config.ProjectPath, "Default project path should be empty")
	assert.NotNil(t, config.Metadata, "Metadata should be initialized")
}

func TestManager_AutoDetectVariant(t *testing.T) {
	tests := []struct {
		name          string
		files         []string
		dirs          []string
		expectedVariant string
	}{
		{
			name:            "go project detection",
			files:           []string{"go.mod"},
			expectedVariant: "go",
		},
		{
			name:            "rust project detection", 
			files:           []string{"Cargo.toml"},
			expectedVariant: "full",
		},
		{
			name:            "node.js project detection",
			files:           []string{"package.json"},
			expectedVariant: "base",
		},
		{
			name:            "python project detection",
			files:           []string{"requirements.txt"},
			expectedVariant: "base",
		},
		{
			name:            "java project detection",
			files:           []string{"pom.xml"},
			expectedVariant: "full",
		},
		{
			name:            "kubernetes project detection (helm)",
			dirs:            []string{"helm"},
			expectedVariant: "k8s",
		},
		{
			name:            "kubernetes project detection (k8s)",
			dirs:            []string{"k8s"},
			expectedVariant: "k8s",
		},
		{
			name:            "kubernetes project detection (kubernetes)",
			dirs:            []string{"kubernetes"},
			expectedVariant: "k8s",
		},
		{
			name:            "cloud project detection (.aws)",
			dirs:            []string{".aws"},
			expectedVariant: "cloud",
		},
		{
			name:            "cloud project detection (terraform)",
			dirs:            []string{"terraform"},
			expectedVariant: "cloud",
		},
		{
			name:            "no project indicators",
			files:           []string{},
			expectedVariant: "base",
		},
		{
			name:            "multiple project types (go wins)",
			files:           []string{"go.mod", "package.json", "requirements.txt"},
			expectedVariant: "go",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			originalDir, _ := os.Getwd()
			defer os.Chdir(originalDir)
			
			// Create test files and directories
			for _, file := range tt.files {
				filePath := filepath.Join(tempDir, file)
				err := os.WriteFile(filePath, []byte("test content"), 0644)
				assert.NoError(t, err, "Should be able to create test file %s", file)
			}
			
			for _, dir := range tt.dirs {
				dirPath := filepath.Join(tempDir, dir)
				err := os.MkdirAll(dirPath, 0755)
				assert.NoError(t, err, "Should be able to create test directory %s", dir)
			}
			
			err := os.Chdir(tempDir)
			assert.NoError(t, err, "Should be able to change to temp directory")
			
			mockLogger := &MockLogger{}
			mockLogger.On("Debug", mock.Anything).Maybe()
			
			manager := NewManager(mockLogger).(*manager)
			
			variant, _ := manager.AutoDetectVariant("")
			assert.Equal(t, tt.expectedVariant, variant, "Auto-detection should return correct variant")
		})
	}
}

func TestManager_AutoDetectVariant_Integration(t *testing.T) {
	mockLogger := &MockLogger{}
	mockLogger.On("Debug", mock.Anything).Maybe()
	
	manager := NewManager(mockLogger)
	
	// Test with current directory (should detect go.mod)
	config := manager.GetDefaultConfig()
	
	// The exact variant depends on current directory, but should be one of the valid ones
	validVariants := []string{"base", "go", "full", "cloud", "k8s"}
	assert.Contains(t, validVariants, config.Variant, "Auto-detected variant should be valid")
}

func BenchmarkManager_AutoDetectVariant(b *testing.B) {
	mockLogger := &MockLogger{}
	mockLogger.On("Debug", mock.Anything).Maybe()
	
	manager := NewManager(mockLogger).(*manager)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.AutoDetectVariant("")
	}
}

func TestIsValidDockerImageName(t *testing.T) {
	tests := []struct {
		name     string
		imageName string
		expected  bool
	}{
		// Valid simple images
		{"ubuntu", "ubuntu", true},
		{"alpine", "alpine", true},
		{"nginx", "nginx", true},
		
		// Valid images with tags
		{"ubuntu with tag", "ubuntu:22.04", true},
		{"alpine with tag", "alpine:3.18", true},
		{"node with tag", "node:18-alpine", true},
		
		// Valid registry images
		{"docker hub with namespace", "library/ubuntu", true},
		{"docker hub with namespace and tag", "library/ubuntu:latest", true},
		{"custom registry", "ghcr.io/dyluth/claude-reactor-base", true},
		{"custom registry with tag", "ghcr.io/dyluth/claude-reactor-base:latest", true},
		{"localhost registry", "localhost:5000/myimage", true},
		{"registry with port", "registry.example.com:443/namespace/image:tag", true},
		
		// Valid digests
		{"image with digest", "ubuntu@sha256:1234567890123456789012345678901234567890123456789012345678901234", true},
		{"registry image with digest", "ghcr.io/org/repo@sha256:1234567890123456789012345678901234567890123456789012345678901234", true},
		
		// Invalid cases
		{"empty string", "", false},
		{"starts with dash", "-invalid", false},
		{"ends with dash", "invalid-", false},
		{"starts with dot", ".invalid", false},
		{"ends with dot", "invalid.", false},
		{"consecutive dots", "invalid..image", false},
		{"consecutive slashes", "invalid//image", false},
		{"uppercase in name", "Ubuntu", false},
		{"invalid tag format", "ubuntu:TAG-", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidDockerImageName(tt.imageName)
			assert.Equal(t, tt.expected, result, "Image name: %s", tt.imageName)
		})
	}
}

func BenchmarkManager_ValidateConfig(b *testing.B) {
	mockLogger := &MockLogger{}
	manager := NewManager(mockLogger)
	
	config := &pkg.Config{
		Variant:     "go",
		Account:     "test",
		DangerMode:  true,
		ProjectPath: "/test/path",
		Metadata:    make(map[string]string),
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.ValidateConfig(config)
	}
}