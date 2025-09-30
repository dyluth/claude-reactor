package auth

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	
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
	normalizedAccount := m.normalizeAccount(account)
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
	
	normalizedAccount := m.normalizeAccount(account)
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
	normalizedAccount := m.normalizeAccount(account)
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
	normalizedAccount := m.normalizeAccount(account)
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
	normalizedAccount := m.normalizeAccount(account)
	if normalizedAccount == "default" {
		return filepath.Join(m.claudeReactorDir, ".default-claude.json")
	}
	return filepath.Join(m.claudeReactorDir, fmt.Sprintf(".%s-claude.json", normalizedAccount))
}

// GetAccountSessionDir returns path to account-specific Claude session directory
// This directory contains all Claude CLI session data: projects/, shell-snapshots/, todos/, etc.
func (m *manager) GetAccountSessionDir(account string) string {
	normalizedAccount := m.normalizeAccount(account)
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
	
	normalizedAccount := m.normalizeAccount(account)
	filename := m.GetAPIKeyFile(normalizedAccount)
	m.logger.Debugf("Saving API key for account %s to %s", normalizedAccount, filename)
	
	return os.WriteFile(filename, []byte(apiKey), 0644)
}

// GetAPIKeyFile returns path to account-specific API key file
func (m *manager) GetAPIKeyFile(account string) string {
	normalizedAccount := m.normalizeAccount(account)
	if normalizedAccount == "default" {
		return filepath.Join(m.claudeReactorDir, ".claude-reactor-default-env")
	}
	return filepath.Join(m.claudeReactorDir, fmt.Sprintf(".claude-reactor-%s-env", normalizedAccount))
}

// CopyMainConfigToAccount copies main Claude config to account directory
func (m *manager) CopyMainConfigToAccount(account string) error {
	normalizedAccount := m.normalizeAccount(account)
	m.logger.Infof("Copying main Claude config to account: %s", normalizedAccount)
	
	// Check if account config already exists
	accountConfigPath := m.GetAccountConfigPath(normalizedAccount)
	if _, err := os.Stat(accountConfigPath); err == nil {
		m.logger.Warnf("Account config already exists: %s", accountConfigPath)
		return nil // Skip copying
	}
	
	// Try to read main claude config
	homeDir, _ := os.UserHomeDir()
	mainConfigPath := filepath.Join(homeDir, ".claude.json")
	var config map[string]interface{}
	
	if data, err := os.ReadFile(mainConfigPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			m.logger.Warnf("Failed to parse main claude config: %v", err)
		} else {
			// Filter out host-specific paths that don't exist in container
			// Only keep /app project and container-safe global settings
			if projects, ok := config["projects"].(map[string]interface{}); ok {
				containerProjects := map[string]interface{}{}
				if appProject, exists := projects["/app"]; exists {
					containerProjects["/app"] = appProject
				} else {
					// Create minimal /app project if it doesn't exist
					containerProjects["/app"] = map[string]interface{}{
						"allowedTools":                           []interface{}{},
						"history":                               []interface{}{},
						"mcpContextUris":                        []interface{}{},
						"mcpServers":                            map[string]interface{}{},
						"enabledMcpjsonServers":                 []interface{}{},
						"disabledMcpjsonServers":                []interface{}{},
						"hasTrustDialogAccepted":                false,
						"hasTrustDialogHooksAccepted":           false,
						"projectOnboardingSeenCount":            0,
						"hasClaudeMdExternalIncludesApproved":   false,
						"hasClaudeMdExternalIncludesWarningShown": false,
					}
				}
				config["projects"] = containerProjects
			}
			
			// Remove any host-specific additional directories or working directories
			delete(config, "additionalDirectories")
			delete(config, "workingDirectories")
		}
	} else {
		m.logger.Warnf("Main claude config not found: %s", mainConfigPath)
		// Create complete config with ALL fields Claude CLI expects to prevent any writes
		// Generate a stable userID based on the account name to avoid changes
		hasher := sha256.New()
		hasher.Write([]byte(normalizedAccount + "-claude-reactor"))
		userID := fmt.Sprintf("%x", hasher.Sum(nil))
		
		config = map[string]interface{}{
			"numStartups":                   1,
			"installMethod":                 "unknown", 
			"autoUpdates":                   true,
			"endpoint":                      "https://api.anthropic.com",
			"projectId":                     "default-project",
			"userID":                        userID,
			"firstStartTime":                time.Now().UTC().Format(time.RFC3339),
			"isQualifiedForDataSharing":     false,
			"hasCompletedOnboarding":        true,
			"lastOnboardingVersion":         "1.0.71",
			"bypassPermissionsModeAccepted": true,
			"fallbackAvailableWarningThreshold": 0.5,
			"subscriptionNoticeCount":       0,
			"hasAvailableSubscription":      false,
			"projects": map[string]interface{}{
				"/app": map[string]interface{}{
					"allowedTools":                           []interface{}{},
					"history":                               []interface{}{},
					"mcpContextUris":                        []interface{}{},
					"mcpServers":                            map[string]interface{}{},
					"enabledMcpjsonServers":                 []interface{}{},
					"disabledMcpjsonServers":                []interface{}{},
					"hasTrustDialogAccepted":                false,
					"hasTrustDialogHooksAccepted":           false,
					"projectOnboardingSeenCount":            0,
					"hasClaudeMdExternalIncludesApproved":   false,
					"hasClaudeMdExternalIncludesWarningShown": false,
				},
			},
		}
	}
	
	// Write account-specific config atomically to prevent corruption
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	// Use atomic write via temporary file to prevent corruption
	tempPath := accountConfigPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary config: %w", err)
	}
	
	// Atomic rename to final location
	if err := os.Rename(tempPath, accountConfigPath); err != nil {
		os.Remove(tempPath) // Clean up temp file on failure
		return fmt.Errorf("failed to finalize account config: %w", err)
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

// GetDefaultAccount returns $USER or "user" fallback per requirements
func (m *manager) GetDefaultAccount() string {
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	return "user"  // Exact fallback requested
}

// normalizeAccount normalizes account name, using new default account logic
func (m *manager) normalizeAccount(account string) string {
	if strings.TrimSpace(account) == "" {
		return m.GetDefaultAccount()
	}
	return account
}

// GenerateProjectHash creates 8-character hash from absolute path
func GenerateProjectHash(projectPath string) string {
	// Must be 8 characters
	// Must use absolute path  
	// Use SHA256 and take first 8 chars
	hash := sha256.Sum256([]byte(projectPath))
	return fmt.Sprintf("%x", hash)[:8]
}

// GetProjectNameFromPath extracts project name from path
func GetProjectNameFromPath(projectPath string) string {
	return filepath.Base(projectPath)
}

// GetProjectSessionDir returns project-specific session directory  
func (m *manager) GetProjectSessionDir(account, projectPath string) string {
	normalizedAccount := m.normalizeAccount(account)
	projectName := GetProjectNameFromPath(projectPath)
	projectHash := GenerateProjectHash(projectPath)
	sessionDirName := fmt.Sprintf("%s-%s", projectName, projectHash)
	return filepath.Join(m.claudeReactorDir, normalizedAccount, sessionDirName)
}