package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	
	"claude-reactor/pkg"
)

// manager implements the ConfigManager interface
type manager struct {
	logger pkg.Logger
}

// NewManager creates a new configuration manager
func NewManager(logger pkg.Logger) pkg.ConfigManager {
	return &manager{
		logger: logger,
	}
}

// LoadConfig loads configuration from file or creates default
func (m *manager) LoadConfig() (*pkg.Config, error) {
	config := m.GetDefaultConfig()
	
	// Try to read .claude-reactor file
	if data, err := os.ReadFile(".claude-reactor"); err == nil {
		// Parse bash-style config file
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			
			switch key {
			case "variant":
				config.Variant = value
			case "account":
				config.Account = value
			case "danger":
				config.DangerMode = value == "true"
			case "session_persistence":
				config.SessionPersistence = value == "true"
			case "last_session_id":
				config.LastSessionID = value
			case "container_id":
				config.ContainerID = value
			}
		}
		
		m.logger.Debug("Configuration loaded from .claude-reactor file")
	} else {
		m.logger.Debug("No .claude-reactor file found, using defaults")
	}
	
	return config, nil
}

// SaveConfig persists configuration to file
func (m *manager) SaveConfig(config *pkg.Config) error {
	// Simple stub implementation for backward compatibility
	// Write basic .claude-reactor file in bash script format
	file, err := os.Create(".claude-reactor")
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()
	
	// Write configuration in bash script format for compatibility
	fmt.Fprintf(file, "variant=%s\n", config.Variant)
	if config.Account != "" {
		fmt.Fprintf(file, "account=%s\n", config.Account)
	}
	if config.DangerMode {
		fmt.Fprintf(file, "danger=true\n")
	}
	if config.SessionPersistence {
		fmt.Fprintf(file, "session_persistence=true\n")
	}
	if config.LastSessionID != "" {
		fmt.Fprintf(file, "last_session_id=%s\n", config.LastSessionID)
	}
	if config.ContainerID != "" {
		fmt.Fprintf(file, "container_id=%s\n", config.ContainerID)
	}
	
	m.logger.Infof("Configuration saved: variant=%s, account=%s, session_persistence=%t", config.Variant, config.Account, config.SessionPersistence)
	return nil
}

// ValidateConfig validates configuration structure and values
func (m *manager) ValidateConfig(config *pkg.Config) error {
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}
	
	if config.Variant == "" {
		return fmt.Errorf("image/variant cannot be empty")
	}
	
	// Check if it's a built-in variant
	builtinVariants := []string{"base", "go", "full", "cloud", "k8s"}
	for _, variant := range builtinVariants {
		if config.Variant == variant {
			return nil // Built-in variant is always valid
		}
	}
	
	// If not a built-in variant, treat as custom Docker image
	// Basic validation for Docker image name format
	if !isValidDockerImageName(config.Variant) {
		return fmt.Errorf("invalid image name format: %s. Must be a built-in variant (base, go, full, cloud, k8s) or valid Docker image name", config.Variant)
	}
	
	return nil // Custom image passed basic validation
}

// GetDefaultConfig returns a default configuration with auto-detection
func (m *manager) GetDefaultConfig() *pkg.Config {
	variant, _ := m.AutoDetectVariant("")
	return &pkg.Config{
		Variant:     variant,
		Account:     "",
		DangerMode:  false,
		ProjectPath: "",
		Metadata:    make(map[string]string),
	}
}

// AutoDetectVariant performs basic project type auto-detection
func (m *manager) AutoDetectVariant(projectPath string) (string, error) {
	if projectPath == "" {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			return "base", err
		}
	}
	
	// Check project directory for project indicators
	if _, err := os.Stat(filepath.Join(projectPath, "go.mod")); err == nil {
		m.logger.Debug("Detected Go project (go.mod found)")
		return "go", nil
	}
	
	if _, err := os.Stat(filepath.Join(projectPath, "Cargo.toml")); err == nil {
		m.logger.Debug("Detected Rust project (Cargo.toml found)")
		return "full", nil // Rust is in the full variant
	}
	
	if _, err := os.Stat(filepath.Join(projectPath, "package.json")); err == nil {
		m.logger.Debug("Detected Node.js project (package.json found)")
		return "base", nil // Node.js is in base
	}
	
	if _, err := os.Stat(filepath.Join(projectPath, "requirements.txt")); err == nil {
		m.logger.Debug("Detected Python project (requirements.txt found)")
		return "base", nil // Python is in base
	}
	
	if _, err := os.Stat(filepath.Join(projectPath, "pom.xml")); err == nil {
		m.logger.Debug("Detected Java project (pom.xml found)")
		return "full", nil // Java is in full
	}
	
	// Check for Kubernetes files
	if _, err := os.Stat(filepath.Join(projectPath, "helm")); err == nil {
		return "k8s", nil
	}
	if _, err := os.Stat(filepath.Join(projectPath, "k8s")); err == nil {
		return "k8s", nil
	}
	if _, err := os.Stat(filepath.Join(projectPath, "kubernetes")); err == nil {
		return "k8s", nil
	}
	
	// Check for cloud configuration files
	if _, err := os.Stat(filepath.Join(projectPath, ".aws")); err == nil {
		return "cloud", nil
	}
	if _, err := os.Stat(filepath.Join(projectPath, "terraform")); err == nil {
		return "cloud", nil
	}
	
	m.logger.Debug("No project type detected, using base variant")
	return "base", nil
}

// ListAccounts returns available Claude accounts
func (m *manager) ListAccounts() ([]string, error) {
	// TODO: Implement account listing from ~/.claude-reactor directory
	m.logger.Debug("Listing Claude accounts (stub implementation)")
	return []string{"default"}, nil
}

// isValidDockerImageName validates basic Docker image name format
// Supports formats like: ubuntu, ubuntu:22.04, ghcr.io/org/repo:tag, localhost:5000/image
func isValidDockerImageName(name string) bool {
	if name == "" {
		return false
	}
	
	// Docker image name pattern (simplified but covers most cases)
	// - May contain registry hostname with optional port
	// - May contain namespace/organization
	// - Image name (required)
	// - Optional tag or digest
	pattern := `^(?:[a-zA-Z0-9][a-zA-Z0-9.-]*(?::[0-9]+)?/)?[a-z0-9](?:[a-z0-9._-]*[a-z0-9])?(?:/[a-z0-9](?:[a-z0-9._-]*[a-z0-9])?)*(?::[a-zA-Z0-9][a-zA-Z0-9._-]*)?(?:@sha256:[a-f0-9]{64})?$`
	
	matched, err := regexp.MatchString(pattern, name)
	if err != nil {
		return false
	}
	
	// Additional basic checks
	if matched {
		// Reject if it starts or ends with special characters
		if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") ||
		   strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".") {
			return false
		}
		
		// Reject if it has consecutive dots or slashes
		if strings.Contains(name, "..") || strings.Contains(name, "//") {
			return false
		}
	}
	
	return matched
}