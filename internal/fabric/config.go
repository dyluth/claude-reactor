package fabric

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"claude-reactor/internal/reactor/logging"
	"claude-reactor/pkg"
)

// ConfigManager handles MCP suite configuration parsing and validation
type ConfigManager struct {
	logger   pkg.Logger
	detector *ServiceDetector
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(logger pkg.Logger) *ConfigManager {
	return &ConfigManager{
		logger:   logger,
		detector: NewServiceDetector(logger),
	}
}

// LoadConfig loads and parses a claude-mcp-suite.yaml file
func (c *ConfigManager) LoadConfig(filepath string) (*pkg.MCPSuite, error) {
	c.logger.Info("Loading MCP suite configuration from %s", filepath)

	// Check if file exists
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found: %s", filepath)
	}

	// Read the file
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Parse YAML
	var suite pkg.MCPSuite
	if err := yaml.Unmarshal(data, &suite); err != nil {
		return nil, fmt.Errorf("failed to parse YAML configuration: %w", err)
	}

	// Validate the configuration
	if err := c.ValidateConfig(&suite); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	c.logger.Info("Successfully loaded configuration with %d services", len(suite.Services))
	return &suite, nil
}

// ValidateConfig validates the parsed configuration for correctness
func (c *ConfigManager) ValidateConfig(suite *pkg.MCPSuite) error {
	if suite == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	// Validate version
	if suite.Version == "" {
		return fmt.Errorf("version field is required")
	}

	// Validate supported versions
	if suite.Version != "1.0" {
		return fmt.Errorf("unsupported configuration version: %s (supported: 1.0)", suite.Version)
	}

	// Validate orchestrator config
	if err := c.validateOrchestratorConfig(&suite.Orchestrator); err != nil {
		return fmt.Errorf("orchestrator configuration error: %w", err)
	}

	// Validate services
	if len(suite.Services) == 0 {
		return fmt.Errorf("at least one MCP service must be defined")
	}

	for name, service := range suite.Services {
		// Apply intelligent defaults before validation
		c.detector.ApplyDefaults(&service)
		
		// Update the service in the map with applied defaults
		suite.Services[name] = service
		
		if err := c.validateService(name, &service); err != nil {
			return fmt.Errorf("service %s validation error: %w", name, err)
		}
		
		// Validate container strategy configuration
		if err := c.detector.ValidateContainerStrategy(&service); err != nil {
			return fmt.Errorf("service %s container strategy error: %w", name, err)
		}
	}

	return nil
}

// validateOrchestratorConfig validates the orchestrator-specific configuration
func (c *ConfigManager) validateOrchestratorConfig(config *pkg.OrchestratorConfig) error {
	if len(config.AllowedMountRoots) == 0 {
		return fmt.Errorf("allowed_mount_roots cannot be empty (security requirement)")
	}

	// Validate mount roots
	for i, root := range config.AllowedMountRoots {
		if !filepath.IsAbs(root) {
			return fmt.Errorf("allowed_mount_roots[%d] must be absolute path: %s", i, root)
		}

		// Clean the path and ensure it ends with /
		cleaned := filepath.Clean(root)
		if !strings.HasSuffix(cleaned, "/") {
			config.AllowedMountRoots[i] = cleaned + "/"
		}
	}

	return nil
}

// validateService validates an individual MCP service configuration
func (c *ConfigManager) validateService(name string, service *pkg.MCPService) error {
	// Validate service name
	if name == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	// Validate reserved names
	reservedNames := []string{"fabric", "orchestrator", "system"}
	for _, reserved := range reservedNames {
		if name == reserved {
			return fmt.Errorf("service name '%s' is reserved", name)
		}
	}

	// Validate image
	if service.Image == "" {
		return fmt.Errorf("image field is required")
	}

	// Basic Docker image name validation
	if !c.isValidDockerImageName(service.Image) {
		return fmt.Errorf("invalid Docker image name: %s", service.Image)
	}

	// Validate timeout if provided
	if service.Timeout != "" {
		if _, err := time.ParseDuration(service.Timeout); err != nil {
			return fmt.Errorf("invalid timeout format '%s': %w", service.Timeout, err)
		}
	}

	return nil
}

// isValidDockerImageName performs basic validation of Docker image names
func (c *ConfigManager) isValidDockerImageName(name string) bool {
	if name == "" {
		return false
	}

	// Basic checks - this could be made more comprehensive
	if strings.Contains(name, " ") {
		return false
	}

	// Must not start or end with certain characters
	if strings.HasPrefix(name, "-") || strings.HasPrefix(name, ".") {
		return false
	}

	if strings.HasSuffix(name, "-") || strings.HasSuffix(name, ".") {
		return false
	}

	return true
}

// GetDefaultConfig returns a default configuration for demonstration
func (c *ConfigManager) GetDefaultConfig() *pkg.MCPSuite {
	return &pkg.MCPSuite{
		Version: "1.0",
		Orchestrator: pkg.OrchestratorConfig{
			AllowedMountRoots: []string{
				"/home/",
				"/Users/",
				"/tmp/",
			},
		},
		Services: map[string]pkg.MCPService{
			"filesystem": {
				Image:   "ghcr.io/modelcontextprotocol/server-filesystem:latest",
				Timeout: "1m",
			},
			"git": {
				Image:   "ghcr.io/modelcontextprotocol/server-git:latest",
				Timeout: "5m",
			},
			"shell": {
				Image:   "ghcr.io/modelcontextprotocol/server-shell:latest",
				Timeout: "1m",
			},
		},
	}
}

// ValidateMountPath validates a requested mount path against allowed roots
// This implements the security requirement from the specification
func (c *ConfigManager) ValidateMountPath(suite *pkg.MCPSuite, requestedPath string) error {
	absPath, err := filepath.Abs(requestedPath)
	if err != nil {
		return fmt.Errorf("invalid mount path: %w", err)
	}

	for _, allowedRoot := range suite.Orchestrator.AllowedMountRoots {
		absRoot, err := filepath.Abs(allowedRoot)
		if err != nil {
			continue
		}

		// Check if the requested path is within the allowed root
		rel, err := filepath.Rel(absRoot, absPath)
		if err != nil {
			continue
		}

		// If the relative path doesn't start with "..", it's within the allowed root
		if !strings.HasPrefix(rel, "..") {
			c.logger.Debug("Mount path validated", "path", absPath, "root", absRoot)
			return nil
		}
	}

	return fmt.Errorf("mount path %s not allowed by configuration", absPath)
}

// WriteDefaultConfig writes a default configuration file
func (c *ConfigManager) WriteDefaultConfig(filepath string) error {
	config := c.GetDefaultConfig()
	
	_, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	// Add comments to make it more user-friendly
	configWithComments := `# Reactor-Fabric Service Suite Configuration
version: "1.0"

# Global settings for the orchestrator
orchestrator:
  # Allowed host paths that clients can request to mount.
  # IMPORTANT: For security, keep this as restrictive as possible.
  allowed_mount_roots:
    - "/home/"
    - "/Users/"
    - "/tmp/"

# Available MCP service definitions
mcp_services:
  # Provides basic filesystem operations.
  filesystem:
    image: "ghcr.io/modelcontextprotocol/server-filesystem:latest"
    timeout: "1m" # Default idle timeout

  # Provides tools for interacting with a Git repository.
  git:
    image: "ghcr.io/modelcontextprotocol/server-git:latest"
    timeout: "5m"

  # Provides tools for running shell commands.
  # Note: This is powerful and potentially dangerous.
  shell:
    image: "ghcr.io/modelcontextprotocol/server-shell:latest"
    timeout: "1m"

  # An example of a claude-reactor based service.
  # This agent would be specialised for Python development.
  python_expert:
    image: "ghcr.io/dyluth/claude-reactor/python:latest"
    timeout: "10m"
    config:
      account: "work_account"
      danger_mode: false
      specialty: "Expert in Python, Django, and data analysis."
`

	if err := os.WriteFile(filepath, []byte(configWithComments), 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	c.logger.Info("Default configuration written to %s", filepath)
	return nil
}

// NewStandaloneLogger creates a simple logger for validation commands that don't need full logging infrastructure
func NewStandaloneLogger() pkg.Logger {
	return logging.NewLogger()
}