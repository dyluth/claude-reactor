package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	
	"claude-reactor/pkg"
)

// manager implements the AuthManager interface
type manager struct {
	logger           pkg.Logger
	claudeReactorDir string
}

// NewManager creates a new authentication manager
func NewManager(logger pkg.Logger) pkg.AuthManager {
	homeDir, _ := os.UserHomeDir()
	claudeReactorDir := filepath.Join(homeDir, ".claude-reactor")
	
	return &manager{
		logger:           logger,
		claudeReactorDir: claudeReactorDir,
	}
}

// GetAuthConfig returns authentication configuration for the specified account
func (m *manager) GetAuthConfig(account string) (*pkg.AuthConfig, error) {
	normalizedAccount := normalizeAccount(account)
	configPath := m.GetAccountConfigPath(normalizedAccount)
	
	m.logger.Debugf("Loading auth config from: %s", configPath)
	
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("authentication config not found for account: %s", account)
	}
	
	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read auth config: %w", err)
	}
	
	// Parse JSON
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		m.logger.Errorf("Failed to parse auth config: %v", err)
		return nil, fmt.Errorf("failed to parse auth config: %w", err)
	}
	
	// Build auth config
	authConfig := &pkg.AuthConfig{
		Account:   normalizedAccount,
		ConfigDir: m.claudeReactorDir,
	}
	
	if apiKey, ok := config["apiKey"].(string); ok {
		authConfig.APIKey = apiKey
	}
	
	return authConfig, nil
}

// SetupAuth sets up authentication for the specified account  
func (m *manager) SetupAuth(account string, apiKey string) error {
	if account == "" {
		m.logger.Errorf("Account cannot be empty")
		return fmt.Errorf("account cannot be empty")
	}
	
	if apiKey == "" {
		m.logger.Errorf("API key cannot be empty")
		return fmt.Errorf("API key cannot be empty")
	}
	
	normalizedAccount := normalizeAccount(account)
	m.logger.Infof("Setting up authentication for account: %s", normalizedAccount)
	
	// Ensure claude-reactor directory exists
	if err := ensureClaudeReactorDir(m.claudeReactorDir); err != nil {
		return fmt.Errorf("failed to create claude-reactor directory: %w", err)
	}
	
	// Copy main config to account if it doesn't exist
	if err := m.CopyMainConfigToAccount(normalizedAccount); err != nil {
		return fmt.Errorf("failed to copy main config: %w", err)
	}
	
	// Save API key to separate file
	if err := m.SaveAPIKey(normalizedAccount, apiKey); err != nil {
		return fmt.Errorf("failed to save API key: %w", err)
	}
	
	return nil
}

// ValidateAuth validates authentication for the specified account
func (m *manager) ValidateAuth(account string) error {
	normalizedAccount := normalizeAccount(account)
	m.logger.Debugf("Validating authentication for account: %s", normalizedAccount)
	
	// Check if config file exists
	configPath := m.GetAccountConfigPath(normalizedAccount)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		m.logger.Errorf("Authentication config not found for account: %s", normalizedAccount)
		return fmt.Errorf("authentication config not found for account: %s", normalizedAccount)
	}
	
	// Check if API key file exists (optional)
	apiKeyFile := m.GetAPIKeyFile(normalizedAccount)
	if _, err := os.Stat(apiKeyFile); os.IsNotExist(err) {
		m.logger.Warnf("API key file not found for account: %s", normalizedAccount)
		// Don't fail, as API key might be in config file
	}
	
	return nil
}

// IsAuthenticated checks if the specified account is authenticated
func (m *manager) IsAuthenticated(account string) bool {
	normalizedAccount := normalizeAccount(account)
	m.logger.Debugf("Checking authentication status for account: %s", normalizedAccount)
	
	configPath := m.GetAccountConfigPath(normalizedAccount)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return false
	}
	
	// Try to parse the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false
	}
	
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return false
	}
	
	return true
}

// GetAccountConfigPath returns path to account-specific config file
func (m *manager) GetAccountConfigPath(account string) string {
	normalizedAccount := normalizeAccount(account)
	if normalizedAccount == "default" {
		return filepath.Join(m.claudeReactorDir, ".default-claude.json")
	}
	return filepath.Join(m.claudeReactorDir, fmt.Sprintf(".%s-claude.json", normalizedAccount))
}

// GetAccountSessionDir returns path to account-specific Claude session directory
// This directory contains all Claude CLI session data: projects/, shell-snapshots/, todos/, etc.
func (m *manager) GetAccountSessionDir(account string) string {
	normalizedAccount := normalizeAccount(account)
	return filepath.Join(m.claudeReactorDir, normalizedAccount)
}

// SaveAPIKey saves API key to project-specific file
func (m *manager) SaveAPIKey(account, apiKey string) error {
	if account == "" {
		m.logger.Errorf("Account cannot be empty")
		return fmt.Errorf("account cannot be empty")
	}
	
	if apiKey == "" {
		m.logger.Errorf("API key cannot be empty") 
		return fmt.Errorf("API key cannot be empty")
	}
	
	normalizedAccount := normalizeAccount(account)
	filename := m.GetAPIKeyFile(normalizedAccount)
	m.logger.Debugf("Saving API key for account %s to %s", normalizedAccount, filename)
	
	return os.WriteFile(filename, []byte(apiKey), 0644)
}

// GetAPIKeyFile returns path to account-specific API key file
func (m *manager) GetAPIKeyFile(account string) string {
	normalizedAccount := normalizeAccount(account)
	if normalizedAccount == "default" {
		return filepath.Join(m.claudeReactorDir, ".claude-reactor-default-env")
	}
	return filepath.Join(m.claudeReactorDir, fmt.Sprintf(".claude-reactor-%s-env", normalizedAccount))
}

// CopyMainConfigToAccount copies main Claude config to account directory
func (m *manager) CopyMainConfigToAccount(account string) error {
	normalizedAccount := normalizeAccount(account)
	m.logger.Infof("Copying main Claude config to account: %s", normalizedAccount)
	
	// Check if account config already exists
	accountConfigPath := m.GetAccountConfigPath(normalizedAccount)
	if _, err := os.Stat(accountConfigPath); err == nil {
		m.logger.Warnf("Account config already exists: %s", accountConfigPath)
		return nil // Skip copying
	}
	
	// Try to read main claude config
	mainConfigPath := filepath.Join(m.claudeReactorDir, ".claude.json")
	var config map[string]interface{}
	
	if data, err := os.ReadFile(mainConfigPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			m.logger.Warnf("Failed to parse main claude config: %v", err)
		}
	} else {
		m.logger.Warnf("Main claude config not found: %s", mainConfigPath)
		// Create default config
		config = map[string]interface{}{
			"endpoint":  "https://api.anthropic.com",
			"projectId": "default-project",
		}
	}
	
	// Write account-specific config
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	if err := os.WriteFile(accountConfigPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write account config: %w", err)
	}
	
	m.logger.Infof("Created account config: %s", accountConfigPath)
	m.logger.Debugf("Config content: %s", string(data))
	
	return nil
}

// Helper functions

// ensureClaudeReactorDir creates the claude-reactor directory if it doesn't exist
func ensureClaudeReactorDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// normalizeAccount normalizes account name, defaulting to "default" for empty strings
func normalizeAccount(account string) string {
	if strings.TrimSpace(account) == "" {
		return "default"
	}
	return account
}