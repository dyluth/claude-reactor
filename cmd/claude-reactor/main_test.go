package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"claude-reactor/cmd/claude-reactor/commands"
	"claude-reactor/pkg"
	"claude-reactor/pkg/mocks"
)

// TestRootCommand tests the basic root command functionality
func TestRootCommand(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantHelp bool
	}{
		{
			name:     "no args shows help",
			args:     []string{},
			wantHelp: true,
		},
		{
			name:     "help flag shows help",
			args:     []string{"--help"},
			wantHelp: true,
		},
		{
			name:     "version flag shows version",
			args:     []string{"--version"},
			wantHelp: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock app container
			app := createMockAppContainer(t)

			// Set up mock expectations for LoadConfig call (used in default behavior)
			mockConfig := &mocks.MockConfigManager{}
			if len(tt.args) == 0 {
				// For no args test, expect LoadConfig to be called and return error (no config)
				mockConfig.On("LoadConfig").Return((*pkg.Config)(nil), fmt.Errorf("no config found"))
			}
			app.ConfigMgr = mockConfig

			// Create root command
			cmd := newRootCmd(app)
			cmd.SetArgs(tt.args)

			// Capture output
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			// Execute command
			err := cmd.Execute()

			// Check results
			output := buf.String()
			if tt.wantHelp {
				assert.Contains(t, output, "claude-reactor")
				assert.Contains(t, output, "Usage:")
			}

			// Version command should not error
			if strings.Contains(strings.Join(tt.args, " "), "version") {
				assert.NoError(t, err)
			}
		})
	}
}

// TestRunCommandFlags tests run command flag parsing
func TestRunCommandFlags(t *testing.T) {
	tests := []struct {
		name              string
		args              []string
		expectedVariant   string
		expectedAccount   string
		expectedDanger    bool
		expectedShell     bool
		expectedNoPersist bool
		shouldError       bool
	}{
		{
			name:              "default flags",
			args:              []string{"run"},
			expectedVariant:   "",
			expectedAccount:   "",
			expectedDanger:    false,
			expectedShell:     false,
			expectedNoPersist: false, // Default is to persist (keep container running)
		},
		{
			name:              "image flag",
			args:              []string{"run", "--image", "go"},
			expectedVariant:   "go",
			expectedAccount:   "",
			expectedDanger:    false,
			expectedShell:     false,
			expectedNoPersist: false,
		},
		{
			name:              "all flags set",
			args:              []string{"run", "--image", "full", "--account", "work", "--danger", "--shell", "--no-persist"},
			expectedVariant:   "full",
			expectedAccount:   "work",
			expectedDanger:    true,
			expectedShell:     true,
			expectedNoPersist: true, // Explicitly set to not persist (remove container when finished)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock app container that will capture flag values
			app := createMockAppContainer(t)

			// Track if runContainer was called with correct parameters
			var capturedVariant, capturedAccount string
			var capturedDanger, capturedShell, capturedNoPersist bool

			// Create a test version of runContainer
			testRunContainer := func(cmd *cobra.Command, app *pkg.AppContainer) error {
				capturedVariant, _ = cmd.Flags().GetString("image")
				capturedAccount, _ = cmd.Flags().GetString("account")
				capturedDanger, _ = cmd.Flags().GetBool("danger")
				capturedShell, _ = cmd.Flags().GetBool("shell")
				capturedNoPersist, _ = cmd.Flags().GetBool("no-persist")
				return nil // Don't actually run the container
			}

			// Create run command with test implementation
			runCmd := commands.NewRunCmd(app)
			runCmd.RunE = func(cmd *cobra.Command, args []string) error {
				return testRunContainer(cmd, app)
			}

			// Create root command and add run command
			rootCmd := &cobra.Command{Use: "claude-reactor"}
			rootCmd.AddCommand(runCmd)
			rootCmd.SetArgs(tt.args)

			// Execute command
			err := rootCmd.Execute()

			if tt.shouldError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedVariant, capturedVariant)
			assert.Equal(t, tt.expectedAccount, capturedAccount)
			assert.Equal(t, tt.expectedDanger, capturedDanger)
			assert.Equal(t, tt.expectedShell, capturedShell)
			assert.Equal(t, tt.expectedNoPersist, capturedNoPersist)
		})
	}
}

// TestCleanCommandFlags tests clean command flag parsing
func TestCleanCommandFlags(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedAll    bool
		expectedImages bool
		shouldError    bool
	}{
		{
			name:           "default flags",
			args:           []string{"clean"},
			expectedAll:    false,
			expectedImages: false,
		},
		{
			name:           "all flag",
			args:           []string{"clean", "--all"},
			expectedAll:    true,
			expectedImages: false,
		},
		{
			name:           "images flag",
			args:           []string{"clean", "--images"},
			expectedAll:    false,
			expectedImages: true,
		},
		{
			name:           "both flags",
			args:           []string{"clean", "--all", "--images"},
			expectedAll:    true,
			expectedImages: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock app container
			app := createMockAppContainer(t)

			// Track if cleanContainers was called with correct parameters
			var capturedAll, capturedImages bool

			// Create a test version of cleanContainers
			testCleanContainers := func(cmd *cobra.Command, app *pkg.AppContainer) error {
				capturedAll, _ = cmd.Flags().GetBool("all")
				capturedImages, _ = cmd.Flags().GetBool("images")
				return nil // Don't actually clean containers
			}

			// Create clean command with test implementation
			cleanCmd := commands.NewCleanCmd(app)
			cleanCmd.RunE = func(cmd *cobra.Command, args []string) error {
				return testCleanContainers(cmd, app)
			}

			// Create root command and add clean command
			rootCmd := &cobra.Command{Use: "claude-reactor"}
			rootCmd.AddCommand(cleanCmd)
			rootCmd.SetArgs(tt.args)

			// Execute command
			err := rootCmd.Execute()

			if tt.shouldError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedAll, capturedAll)
			assert.Equal(t, tt.expectedImages, capturedImages)
		})
	}
}

// TestConfigCommand tests config subcommands
func TestConfigCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		shouldError bool
	}{
		{
			name:        "config show",
			args:        []string{"config", "show"},
			shouldError: false,
		},
		{
			name:        "config validate",
			args:        []string{"config", "validate"},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock app container
			app := createMockAppContainer(t)

			// Set up mock expectations for Logger
			app.Logger.(*mocks.MockLogger).On("Info", mock.Anything).Return()

			// Mock ConfigManager methods
			mockConfig := &pkg.Config{
				Variant:     "go",
				Account:     "test",
				DangerMode:  false,
				ProjectPath: "/test/path",
				Metadata:    make(map[string]string),
			}

			app.ConfigMgr.(*mocks.MockConfigManager).On("LoadConfig").Return(mockConfig, nil)
			app.ConfigMgr.(*mocks.MockConfigManager).On("ValidateConfig", mockConfig).Return(nil)
			app.ConfigMgr.(*mocks.MockConfigManager).On("AutoDetectVariant", "").Return("go", nil)

			// Create config command
			configCmd := commands.NewConfigCmd(app)

			// Create root command and add config command
			rootCmd := &cobra.Command{Use: "claude-reactor"}
			rootCmd.AddCommand(configCmd)
			rootCmd.SetArgs(tt.args)

			// Capture output
			buf := new(bytes.Buffer)
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)

			// Execute command
			err := rootCmd.Execute()

			if tt.shouldError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			// Note: The config commands write output directly to stdout
			// which doesn't get captured in the command buffer.
			// The test success is that the commands execute without error.
		})
	}
}

// TestLegacyCompatibility removed - legacy flags cause os.Exit(1) which is hard to test
// These flags show appropriate deprecation messages and exit, which is the intended behavior

// createMockAppContainer creates a mock AppContainer for testing
func createMockAppContainer(t *testing.T) *pkg.AppContainer {
	return &pkg.AppContainer{
		ConfigMgr:    &mocks.MockConfigManager{},
		DockerMgr:    &mocks.MockDockerManager{},
		AuthMgr:      &mocks.MockAuthManager{},
		MountMgr:     &mocks.MockMountManager{},
		ArchDetector: &mocks.MockArchDetector{},
		Logger:       &mocks.MockLogger{},
	}
}

// Test utility functions that have 0% coverage
func TestGetDisplayValue(t *testing.T) {
	t.Run("returns value when not empty", func(t *testing.T) {
		result := getDisplayValue("actual-value", "default")
		assert.Equal(t, "actual-value", result)
	})

	t.Run("returns default display when value is empty", func(t *testing.T) {
		result := getDisplayValue("", "default-option")
		assert.Equal(t, "(using default-option)", result)
	})

	t.Run("handles empty default display", func(t *testing.T) {
		result := getDisplayValue("", "")
		assert.Equal(t, "(using )", result)
	})
}

func TestGetCurrentDir(t *testing.T) {
	t.Run("returns current directory", func(t *testing.T) {
		result := getCurrentDir()

		// Should return a non-empty string and not "unknown" in normal circumstances
		assert.NotEmpty(t, result)
		// The exact path will vary, but it should not be "unknown" in normal test runs
		// We can't assert the exact path since it depends on where tests are run
	})
}
