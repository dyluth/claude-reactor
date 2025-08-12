package config

import (
	"fmt"
	"os"
	"path/filepath"
	
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
	// TODO: Implement YAML configuration loading
	m.logger.Debug("Loading configuration (stub implementation)")
	return m.GetDefaultConfig(), nil
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
	
	m.logger.Infof("Configuration saved: variant=%s, account=%s", config.Variant, config.Account)
	return nil
}

// ValidateConfig validates configuration structure and values
func (m *manager) ValidateConfig(config *pkg.Config) error {
	// TODO: Implement configuration validation
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}
	
	validVariants := []string{"base", "go", "full", "cloud", "k8s"}
	for _, variant := range validVariants {
		if config.Variant == variant {
			return nil
		}
	}
	
	return fmt.Errorf("invalid variant: %s", config.Variant)
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