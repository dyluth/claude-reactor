package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

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
	assert.NotEmpty(t, impl.claudeReactorDir)
}

func TestManager_GetAuthConfig(t *testing.T) {
	tests := []struct {
		name        string
		account     string
		setupFiles  func(string) error
		setupMocks  func(*mocks.MockLogger)
		expectError bool
		expectNil   bool
	}{
		{
			name:    "get default account auth config",
			account: "default",
			setupFiles: func(dir string) error {
				configPath := filepath.Join(dir, ".default-claude.json")
				config := map[string]interface{}{
					"apiKey":    "test-api-key",
					"endpoint":  "https://api.anthropic.com",
					"projectId": "test-project",
				}
				data, _ := json.Marshal(config)
				return os.WriteFile(configPath, data, 0644)
			},
			setupMocks: func(mockLogger *mocks.MockLogger) {
			mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: false,
			expectNil:   false,
		},
		{
			name:    "get named account auth config",
			account: "work",
			setupFiles: func(dir string) error {
				configPath := filepath.Join(dir, ".work-claude.json")
				config := map[string]interface{}{
					"apiKey":    "work-api-key",
					"endpoint":  "https://api.anthropic.com",
					"projectId": "work-project",
				}
				data, _ := json.Marshal(config)
				return os.WriteFile(configPath, data, 0644)
			},
			setupMocks: func(mockLogger *mocks.MockLogger) {
			mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: false,
			expectNil:   false,
		},
		{
			name:    "get non-existent account config",
			account: "nonexistent",
			setupFiles: func(dir string) error {
				// Don't create any config file
				return nil
			},
			setupMocks: func(mockLogger *mocks.MockLogger) {
			mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: true,
			expectNil:   true,
		},
		{
			name:    "get config with invalid json",
			account: "invalid",
			setupFiles: func(dir string) error {
				configPath := filepath.Join(dir, ".invalid-claude.json")
				invalidJson := `{"apiKey": "test", "invalid": json}`
				return os.WriteFile(configPath, []byte(invalidJson), 0644)
			},
			setupMocks: func(mockLogger *mocks.MockLogger) {
				mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Errorf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Warnf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: true,
			expectNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for this test
			tempDir, err := os.MkdirTemp("", "auth-test-*")
			assert.NoError(t, err)
			defer os.RemoveAll(tempDir)

			// Setup test files
			err = tt.setupFiles(tempDir)
			assert.NoError(t, err)

			// Setup mocks
			mockLogger := &mocks.MockLogger{}
			tt.setupMocks(mockLogger)

			mgr := &manager{
				logger:           mockLogger,
				claudeReactorDir: tempDir,
			}

			// Execute
			config, err := mgr.GetAuthConfig(tt.account)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.expectNil {
				assert.Nil(t, config)
			} else {
				assert.NotNil(t, config)
				assert.NotEmpty(t, config.APIKey)
				assert.NotEmpty(t, config.ConfigDir)
			}

			// Verify mock expectations
			mockLogger.AssertExpectations(t)
		})
	}
}

func TestManager_SetupAuth(t *testing.T) {
	tests := []struct {
		name        string
		account     string
		apiKey      string
		setupFiles  func(string) error
		setupMocks  func(*mocks.MockLogger)
		expectError bool
	}{
		{
			name:    "setup auth for new account",
			account: "new-account",
			apiKey:  "sk-ant-test-key",
			setupFiles: func(dir string) error {
				// Create main claude config to copy from
				mainConfigPath := filepath.Join(dir, ".claude.json")
				config := map[string]interface{}{
					"endpoint":  "https://api.anthropic.com",
					"projectId": "main-project",
				}
				data, _ := json.Marshal(config)
				return os.WriteFile(mainConfigPath, data, 0644)
			},
			setupMocks: func(mockLogger *mocks.MockLogger) {
				mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Infof", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Warnf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: false,
		},
		{
			name:    "setup auth for existing account",
			account: "existing",
			apiKey:  "sk-ant-existing-key",
			setupFiles: func(dir string) error {
				// Create existing account config
				configPath := filepath.Join(dir, ".existing-claude.json")
				config := map[string]interface{}{
					"apiKey":    "old-api-key",
					"endpoint":  "https://api.anthropic.com",
					"projectId": "existing-project",
				}
				data, _ := json.Marshal(config)
				return os.WriteFile(configPath, data, 0644)
			},
			setupMocks: func(mockLogger *mocks.MockLogger) {
				mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Infof", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Warnf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: false,
		},
		{
			name:    "setup auth with empty API key",
			account: "test-account",
			apiKey:  "",
			setupFiles: func(dir string) error {
				return nil
			},
			setupMocks: func(mockLogger *mocks.MockLogger) {
				mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Errorf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Warnf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: true,
		},
		{
			name:    "setup auth with empty account",
			account: "",
			apiKey:  "sk-ant-test-key",
			setupFiles: func(dir string) error {
				return nil
			},
			setupMocks: func(mockLogger *mocks.MockLogger) {
				mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Errorf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Warnf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for this test
			tempDir, err := os.MkdirTemp("", "auth-setup-*")
			assert.NoError(t, err)
			defer os.RemoveAll(tempDir)

			// Setup test files
			err = tt.setupFiles(tempDir)
			assert.NoError(t, err)

			// Setup mocks
			mockLogger := &mocks.MockLogger{}
			tt.setupMocks(mockLogger)

			mgr := &manager{
				logger:           mockLogger,
				claudeReactorDir: tempDir,
			}

			// Execute
			err = mgr.SetupAuth(tt.account, tt.apiKey)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify config file was created/updated
				configPath := mgr.GetAccountConfigPath(tt.account)
				assert.FileExists(t, configPath)

				// Verify API key was saved to separate file
				if tt.apiKey != "" {
					apiKeyFile := mgr.GetAPIKeyFile(tt.account)
					assert.FileExists(t, apiKeyFile)

					content, err := os.ReadFile(apiKeyFile)
					assert.NoError(t, err)
					assert.Equal(t, tt.apiKey, string(content))
				}
			}

			// Verify mock expectations
			mockLogger.AssertExpectations(t)
		})
	}
}

func TestManager_ValidateAuth(t *testing.T) {
	tests := []struct {
		name        string
		account     string
		setupFiles  func(string) error
		setupMocks  func(*mocks.MockLogger)
		expectError bool
	}{
		{
			name:    "validate existing auth",
			account: "valid-account",
			setupFiles: func(dir string) error {
				// Create valid config
				configPath := filepath.Join(dir, ".valid-account-claude.json")
				config := map[string]interface{}{
					"endpoint":  "https://api.anthropic.com",
					"projectId": "valid-project",
				}
				data, _ := json.Marshal(config)
				if err := os.WriteFile(configPath, data, 0644); err != nil {
					return err
				}

				// Create API key file
				apiKeyPath := filepath.Join(dir, ".claude-reactor-valid-account-env")
				return os.WriteFile(apiKeyPath, []byte("sk-ant-valid-key"), 0644)
			},
			setupMocks: func(mockLogger *mocks.MockLogger) {
			mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: false,
		},
		{
			name:    "validate missing auth config",
			account: "missing-account",
			setupFiles: func(dir string) error {
				// Don't create any files
				return nil
			},
			setupMocks: func(mockLogger *mocks.MockLogger) {
				mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Errorf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Warnf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: true,
		},
		{
			name:    "validate auth with missing API key",
			account: "no-key-account",
			setupFiles: func(dir string) error {
				// Create config but no API key file
				configPath := filepath.Join(dir, ".no-key-account-claude.json")
				config := map[string]interface{}{
					"endpoint":  "https://api.anthropic.com",
					"projectId": "no-key-project",
				}
				data, _ := json.Marshal(config)
				return os.WriteFile(configPath, data, 0644)
			},
			setupMocks: func(mockLogger *mocks.MockLogger) {
			mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Warnf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: false, // Should validate config even without API key file
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for this test
			tempDir, err := os.MkdirTemp("", "auth-validate-*")
			assert.NoError(t, err)
			defer os.RemoveAll(tempDir)

			// Setup test files
			err = tt.setupFiles(tempDir)
			assert.NoError(t, err)

			// Setup mocks
			mockLogger := &mocks.MockLogger{}
			tt.setupMocks(mockLogger)

			mgr := &manager{
				logger:           mockLogger,
				claudeReactorDir: tempDir,
			}

			// Execute
			err = mgr.ValidateAuth(tt.account)

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

func TestManager_IsAuthenticated(t *testing.T) {
	tests := []struct {
		name       string
		account    string
		setupFiles func(string) error
		expected   bool
	}{
		{
			name:    "authenticated account",
			account: "authenticated",
			setupFiles: func(dir string) error {
				configPath := filepath.Join(dir, ".authenticated-claude.json")
				config := map[string]interface{}{
					"endpoint":  "https://api.anthropic.com",
					"projectId": "auth-project",
				}
				data, _ := json.Marshal(config)
				return os.WriteFile(configPath, data, 0644)
			},
			expected: true,
		},
		{
			name:    "not authenticated account",
			account: "not-authenticated",
			setupFiles: func(dir string) error {
				// Don't create config file
				return nil
			},
			expected: false,
		},
		{
			name:    "account with invalid config",
			account: "invalid-config",
			setupFiles: func(dir string) error {
				configPath := filepath.Join(dir, ".invalid-config-claude.json")
				invalidJson := `{"invalid": json}`
				return os.WriteFile(configPath, []byte(invalidJson), 0644)
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for this test
			tempDir, err := os.MkdirTemp("", "auth-is-auth-*")
			assert.NoError(t, err)
			defer os.RemoveAll(tempDir)

			// Setup test files
			err = tt.setupFiles(tempDir)
			assert.NoError(t, err)

			// Setup mocks
			mockLogger := &mocks.MockLogger{}
			mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()

			mgr := &manager{
				logger:           mockLogger,
				claudeReactorDir: tempDir,
			}

			// Execute
			result := mgr.IsAuthenticated(tt.account)

			// Verify
			assert.Equal(t, tt.expected, result)
			
			// Verify mock expectations
			mockLogger.AssertExpectations(t)
		})
	}
}

func TestManager_GetAccountConfigPath(t *testing.T) {
	// Get expected default account for empty string
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		currentUser = "user" // fallback
	}
	
	tests := []struct {
		name     string
		account  string
		expected string
	}{
		{
			name:     "default account",
			account:  "default",
			expected: ".default-claude.json",
		},
		{
			name:     "named account",
			account:  "work",
			expected: ".work-claude.json",
		},
		{
			name:     "empty account defaults to current user",
			account:  "",
			expected: fmt.Sprintf(".%s-claude.json", currentUser),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLogger := &mocks.MockLogger{}
			baseDir := "/test/dir"

			mgr := &manager{
				logger:           mockLogger,
				claudeReactorDir: baseDir,
			}

			result := mgr.GetAccountConfigPath(tt.account)
			expected := filepath.Join(baseDir, tt.expected)
			assert.Equal(t, expected, result)
		})
	}
}

func TestManager_SaveAPIKey(t *testing.T) {
	tests := []struct {
		name        string
		account     string
		apiKey      string
		setupMocks  func(*mocks.MockLogger)
		expectError bool
	}{
		{
			name:    "save API key for valid account",
			account: "test-account",
			apiKey:  "sk-ant-test-key-123",
			setupMocks: func(mockLogger *mocks.MockLogger) {
			mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: false,
		},
		{
			name:    "save API key with empty account",
			account: "",
			apiKey:  "sk-ant-test-key-123",
			setupMocks: func(mockLogger *mocks.MockLogger) {
				mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Errorf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Warnf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: true,
		},
		{
			name:    "save empty API key",
			account: "test-account",
			apiKey:  "",
			setupMocks: func(mockLogger *mocks.MockLogger) {
				mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Errorf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Warnf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for this test
			tempDir, err := os.MkdirTemp("", "auth-save-key-*")
			assert.NoError(t, err)
			defer os.RemoveAll(tempDir)

			// Setup mocks
			mockLogger := &mocks.MockLogger{}
			tt.setupMocks(mockLogger)

			mgr := &manager{
				logger:           mockLogger,
				claudeReactorDir: tempDir,
			}

			// Execute
			err = mgr.SaveAPIKey(tt.account, tt.apiKey)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify API key file was created
				apiKeyFile := mgr.GetAPIKeyFile(tt.account)
				assert.FileExists(t, apiKeyFile)

				// Verify content
				content, err := os.ReadFile(apiKeyFile)
				assert.NoError(t, err)
				assert.Equal(t, tt.apiKey, string(content))
			}

			// Verify mock expectations
			mockLogger.AssertExpectations(t)
		})
	}
}

func TestManager_GetAPIKeyFile(t *testing.T) {
	// Get expected default account for empty string
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		currentUser = "user" // fallback
	}
	
	tests := []struct {
		name     string
		account  string
		expected string
	}{
		{
			name:     "default account API key file",
			account:  "default",
			expected: ".claude-reactor-default-env",
		},
		{
			name:     "named account API key file",
			account:  "work",
			expected: ".claude-reactor-work-env",
		},
		{
			name:     "empty account defaults to current user",
			account:  "",
			expected: fmt.Sprintf(".claude-reactor-%s-env", currentUser),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLogger := &mocks.MockLogger{}
			baseDir := "/test/dir"

			mgr := &manager{
				logger:           mockLogger,
				claudeReactorDir: baseDir,
			}

			result := mgr.GetAPIKeyFile(tt.account)
			expected := filepath.Join(baseDir, tt.expected)
			assert.Equal(t, expected, result)
		})
	}
}

func TestManager_CopyMainConfigToAccount(t *testing.T) {
	tests := []struct {
		name        string
		account     string
		setupFiles  func(string) error
		setupMocks  func(*mocks.MockLogger)
		expectError bool
	}{
		{
			name:    "copy existing main config",
			account: "new-account",
			setupFiles: func(dir string) error {
				mainConfigPath := filepath.Join(dir, ".claude.json")
				config := map[string]interface{}{
					"endpoint":    "https://api.anthropic.com",
					"projectId":   "main-project",
					"preferences": map[string]interface{}{"theme": "dark"},
				}
				data, _ := json.Marshal(config)
				return os.WriteFile(mainConfigPath, data, 0644)
			},
			setupMocks: func(mockLogger *mocks.MockLogger) {
			mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Infof", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: false,
		},
		{
			name:    "copy non-existent main config",
			account: "test-account",
			setupFiles: func(dir string) error {
				// Don't create main config
				return nil
			},
			setupMocks: func(mockLogger *mocks.MockLogger) {
			mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Warnf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Infof", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: false, // Should create default config
		},
		{
			name:    "copy with existing account config",
			account: "existing-account",
			setupFiles: func(dir string) error {
				// Create main config
				mainConfigPath := filepath.Join(dir, ".claude.json")
				mainConfig := map[string]interface{}{
					"endpoint":  "https://api.anthropic.com",
					"projectId": "main-project",
				}
				mainData, _ := json.Marshal(mainConfig)
				if err := os.WriteFile(mainConfigPath, mainData, 0644); err != nil {
					return err
				}

				// Create existing account config
				accountConfigPath := filepath.Join(dir, ".existing-account-claude.json")
				existingConfig := map[string]interface{}{
					"endpoint":  "https://api.anthropic.com",
					"projectId": "existing-project",
				}
				existingData, _ := json.Marshal(existingConfig)
				return os.WriteFile(accountConfigPath, existingData, 0644)
			},
			setupMocks: func(mockLogger *mocks.MockLogger) {
			mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Warnf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Infof", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: false, // Should skip copying
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for this test
			tempDir, err := os.MkdirTemp("", "auth-copy-*")
			assert.NoError(t, err)
			defer os.RemoveAll(tempDir)

			// Setup test files
			err = tt.setupFiles(tempDir)
			assert.NoError(t, err)

			// Setup mocks
			mockLogger := &mocks.MockLogger{}
			tt.setupMocks(mockLogger)

			mgr := &manager{
				logger:           mockLogger,
				claudeReactorDir: tempDir,
			}

			// Execute
			err = mgr.CopyMainConfigToAccount(tt.account)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify account config was created
				accountConfigPath := mgr.GetAccountConfigPath(tt.account)
				assert.FileExists(t, accountConfigPath)
			}

			// Verify mock expectations
			mockLogger.AssertExpectations(t)
		})
	}
}

func TestEnsureClaudeReactorDir(t *testing.T) {
	// Create temporary directory for this test
	tempDir, err := os.MkdirTemp("", "ensure-dir-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	claudeReactorDir := filepath.Join(tempDir, ".claude-reactor")

	// Verify directory doesn't exist initially
	assert.NoDirExists(t, claudeReactorDir)

	// Execute
	err = ensureClaudeReactorDir(claudeReactorDir)

	// Verify
	assert.NoError(t, err)
	assert.DirExists(t, claudeReactorDir)

	// Test calling again on existing directory
	err = ensureClaudeReactorDir(claudeReactorDir)
	assert.NoError(t, err)
	assert.DirExists(t, claudeReactorDir)
}

func TestNormalizeAccount(t *testing.T) {
	// Get expected default account for empty string
	currentUser := os.Getenv("USER")
	if currentUser == "" {
		currentUser = "user" // fallback
	}
	
	tests := []struct {
		name     string
		account  string
		expected string
	}{
		{
			name:     "normal account name",
			account:  "work",
			expected: "work",
		},
		{
			name:     "empty account name",
			account:  "",
			expected: currentUser,
		},
		{
			name:     "whitespace account name",
			account:  "   ",
			expected: currentUser,
		},
		{
			name:     "account name with spaces",
			account:  "my work",
			expected: "my work",
		},
		{
			name:     "default account name",
			account:  "default",
			expected: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := &manager{logger: &mocks.MockLogger{}}
			result := mgr.normalizeAccount(tt.account)
			assert.Equal(t, tt.expected, result)
		})
	}
}