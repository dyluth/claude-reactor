package fabric

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"claude-reactor/internal/reactor/logging"
	"claude-reactor/pkg"
)

func TestNewConfigManager(t *testing.T) {
	logger := logging.NewLogger()
	mgr := NewConfigManager(logger)
	
	assert.NotNil(t, mgr)
	assert.Equal(t, logger, mgr.logger)
}

func TestConfigManager_LoadConfig(t *testing.T) {
	logger := logging.NewLogger()
	mgr := NewConfigManager(logger)

	tests := []struct {
		name        string
		configYAML  string
		expectError bool
		expectServices int
	}{
		{
			name: "valid configuration",
			configYAML: `version: "1.0"
orchestrator:
  allowed_mount_roots:
    - "/home/"
    - "/tmp/"
mcp_services:
  filesystem:
    image: "ghcr.io/modelcontextprotocol/server-filesystem:latest"
    timeout: "1m"
  git:
    image: "ghcr.io/modelcontextprotocol/server-git:latest"
    timeout: "5m"`,
			expectError: false,
			expectServices: 2,
		},
		{
			name: "invalid YAML syntax",
			configYAML: `version: "1.0"
orchestrator:
  allowed_mount_roots:
    - "/home/"
    - "/tmp/"
mcp_services:
  filesystem:
    image: "filesystem:latest"
    timeout: "1m"
    invalid_yaml: [unclosed`,
			expectError: true,
			expectServices: 0,
		},
		{
			name: "missing version",
			configYAML: `orchestrator:
  allowed_mount_roots:
    - "/home/"
mcp_services:
  filesystem:
    image: "filesystem:latest"`,
			expectError: true,
			expectServices: 0,
		},
		{
			name: "empty services",
			configYAML: `version: "1.0"
orchestrator:
  allowed_mount_roots:
    - "/home/"
mcp_services: {}`,
			expectError: true,
			expectServices: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "test-config.yaml")
			
			err := os.WriteFile(configFile, []byte(tt.configYAML), 0644)
			require.NoError(t, err)

			// Test loading
			suite, err := mgr.LoadConfig(configFile)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, suite)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, suite)
				assert.Equal(t, tt.expectServices, len(suite.Services))
			}
		})
	}
}

func TestConfigManager_LoadConfig_FileNotFound(t *testing.T) {
	logger := logging.NewLogger()
	mgr := NewConfigManager(logger)

	_, err := mgr.LoadConfig("/nonexistent/file.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "configuration file not found")
}

func TestConfigManager_ValidateConfig(t *testing.T) {
	logger := logging.NewLogger()
	mgr := NewConfigManager(logger)

	tests := []struct {
		name        string
		suite       *pkg.MCPSuite
		expectError bool
		errorContains string
	}{
		{
			name:          "nil configuration",
			suite:         nil,
			expectError:   true,
			errorContains: "cannot be nil",
		},
		{
			name: "missing version",
			suite: &pkg.MCPSuite{
				Version: "",
				Orchestrator: pkg.OrchestratorConfig{
					AllowedMountRoots: []string{"/home/"},
				},
				Services: map[string]pkg.MCPService{
					"test": {Image: "test:latest"},
				},
			},
			expectError:   true,
			errorContains: "version field is required",
		},
		{
			name: "unsupported version",
			suite: &pkg.MCPSuite{
				Version: "2.0",
				Orchestrator: pkg.OrchestratorConfig{
					AllowedMountRoots: []string{"/home/"},
				},
				Services: map[string]pkg.MCPService{
					"test": {Image: "test:latest"},
				},
			},
			expectError:   true,
			errorContains: "unsupported configuration version",
		},
		{
			name: "empty allowed mount roots",
			suite: &pkg.MCPSuite{
				Version: "1.0",
				Orchestrator: pkg.OrchestratorConfig{
					AllowedMountRoots: []string{},
				},
				Services: map[string]pkg.MCPService{
					"test": {Image: "test:latest"},
				},
			},
			expectError:   true,
			errorContains: "allowed_mount_roots cannot be empty",
		},
		{
			name: "no services",
			suite: &pkg.MCPSuite{
				Version: "1.0",
				Orchestrator: pkg.OrchestratorConfig{
					AllowedMountRoots: []string{"/home/"},
				},
				Services: map[string]pkg.MCPService{},
			},
			expectError:   true,
			errorContains: "at least one MCP service must be defined",
		},
		{
			name: "valid configuration",
			suite: &pkg.MCPSuite{
				Version: "1.0",
				Orchestrator: pkg.OrchestratorConfig{
					AllowedMountRoots: []string{"/home/", "/tmp/"},
				},
				Services: map[string]pkg.MCPService{
					"filesystem": {
						Image:   "ghcr.io/modelcontextprotocol/server-filesystem:latest",
						Timeout: "1m",
					},
					"git": {
						Image: "ghcr.io/modelcontextprotocol/server-git:latest",
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mgr.ValidateConfig(tt.suite)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigManager_validateService(t *testing.T) {
	logger := logging.NewLogger()
	mgr := NewConfigManager(logger)

	tests := []struct {
		name        string
		serviceName string
		service     pkg.MCPService
		expectError bool
		errorContains string
	}{
		{
			name:        "empty service name",
			serviceName: "",
			service:     pkg.MCPService{Image: "test:latest"},
			expectError: true,
			errorContains: "service name cannot be empty",
		},
		{
			name:        "reserved service name",
			serviceName: "fabric",
			service:     pkg.MCPService{Image: "test:latest"},
			expectError: true,
			errorContains: "service name 'fabric' is reserved",
		},
		{
			name:        "empty image",
			serviceName: "test",
			service:     pkg.MCPService{Image: ""},
			expectError: true,
			errorContains: "image field is required",
		},
		{
			name:        "invalid image name",
			serviceName: "test",
			service:     pkg.MCPService{Image: "invalid image name"},
			expectError: true,
			errorContains: "invalid Docker image name",
		},
		{
			name:        "invalid timeout",
			serviceName: "test",
			service:     pkg.MCPService{Image: "test:latest", Timeout: "invalid"},
			expectError: true,
			errorContains: "invalid timeout format",
		},
		{
			name:        "valid service",
			serviceName: "filesystem",
			service:     pkg.MCPService{Image: "ghcr.io/modelcontextprotocol/server-filesystem:latest", Timeout: "1m"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mgr.validateService(tt.serviceName, &tt.service)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigManager_isValidDockerImageName(t *testing.T) {
	logger := logging.NewLogger()
	mgr := NewConfigManager(logger)

	tests := []struct {
		name     string
		image    string
		expected bool
	}{
		{"empty string", "", false},
		{"simple name", "ubuntu", true},
		{"with tag", "ubuntu:22.04", true},
		{"with registry", "ghcr.io/user/repo:latest", true},
		{"with spaces", "invalid image", false},
		{"starts with dash", "-invalid", false},
		{"ends with dash", "invalid-", false},
		{"starts with dot", ".invalid", false},
		{"ends with dot", "invalid.", false},
		{"valid complex", "registry.io/namespace/image:v1.0.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mgr.isValidDockerImageName(tt.image)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigManager_GetDefaultConfig(t *testing.T) {
	logger := logging.NewLogger()
	mgr := NewConfigManager(logger)

	config := mgr.GetDefaultConfig()

	assert.NotNil(t, config)
	assert.Equal(t, "1.0", config.Version)
	assert.NotEmpty(t, config.Orchestrator.AllowedMountRoots)
	assert.NotEmpty(t, config.Services)
	
	// Check that default services are present
	assert.Contains(t, config.Services, "filesystem")
	assert.Contains(t, config.Services, "git")
	assert.Contains(t, config.Services, "shell")
}

func TestConfigManager_WriteDefaultConfig(t *testing.T) {
	logger := logging.NewLogger()
	mgr := NewConfigManager(logger)

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "default-config.yaml")

	err := mgr.WriteDefaultConfig(configFile)
	assert.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(configFile)
	assert.NoError(t, err)

	// Verify content can be loaded back
	suite, err := mgr.LoadConfig(configFile)
	assert.NoError(t, err)
	assert.NotNil(t, suite)
	assert.Equal(t, "1.0", suite.Version)
}