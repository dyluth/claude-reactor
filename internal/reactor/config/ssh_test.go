package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"claude-reactor/pkg"
)

// SimpleLogger for testing - doesn't require mock setup
type SimpleLogger struct{}

func (s *SimpleLogger) Debug(args ...interface{})                            {}
func (s *SimpleLogger) Info(args ...interface{})                             {}
func (s *SimpleLogger) Warn(args ...interface{})                             {}
func (s *SimpleLogger) Error(args ...interface{})                            {}
func (s *SimpleLogger) Fatal(args ...interface{})                            {}
func (s *SimpleLogger) Debugf(format string, args ...interface{})            {}
func (s *SimpleLogger) Infof(format string, args ...interface{})             {}
func (s *SimpleLogger) Warnf(format string, args ...interface{})             {}
func (s *SimpleLogger) Errorf(format string, args ...interface{})            {}
func (s *SimpleLogger) Fatalf(format string, args ...interface{})            {}
func (s *SimpleLogger) WithField(key string, value interface{}) pkg.Logger   { return s }
func (s *SimpleLogger) WithFields(fields map[string]interface{}) pkg.Logger { return s }

func TestDetectSSHAgent(t *testing.T) {
	logger := &SimpleLogger{}
	mgr := NewManager(logger).(*manager)

	t.Run("no SSH_AUTH_SOCK set", func(t *testing.T) {
		// Temporarily unset SSH_AUTH_SOCK
		originalValue := os.Getenv("SSH_AUTH_SOCK")
		defer os.Setenv("SSH_AUTH_SOCK", originalValue)
		os.Unsetenv("SSH_AUTH_SOCK")

		_, err := mgr.DetectSSHAgent()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no SSH agent detected")
		assert.Contains(t, err.Error(), "Start SSH agent: eval $(ssh-agent)")
	})

	t.Run("SSH_AUTH_SOCK points to non-existent file", func(t *testing.T) {
		// Temporarily set SSH_AUTH_SOCK to non-existent path
		originalValue := os.Getenv("SSH_AUTH_SOCK")
		defer os.Setenv("SSH_AUTH_SOCK", originalValue)
		os.Setenv("SSH_AUTH_SOCK", "/non/existent/socket")

		_, err := mgr.DetectSSHAgent()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no SSH agent detected")
	})

	t.Run("SSH_AUTH_SOCK points to existing file", func(t *testing.T) {
		// Create a temporary file to simulate SSH agent socket
		tmpDir := t.TempDir()
		socketPath := filepath.Join(tmpDir, "ssh-agent.sock")
		file, err := os.Create(socketPath)
		require.NoError(t, err)
		file.Close()

		// Temporarily set SSH_AUTH_SOCK
		originalValue := os.Getenv("SSH_AUTH_SOCK")
		defer os.Setenv("SSH_AUTH_SOCK", originalValue)
		os.Setenv("SSH_AUTH_SOCK", socketPath)

		result, err := mgr.DetectSSHAgent()
		assert.NoError(t, err)
		assert.Equal(t, socketPath, result)
	})
}

func TestValidateSSHAgent(t *testing.T) {
	logger := &SimpleLogger{}
	mgr := NewManager(logger).(*manager)

	t.Run("empty socket path", func(t *testing.T) {
		err := mgr.ValidateSSHAgent("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SSH agent socket path is empty")
	})

	t.Run("non-existent socket", func(t *testing.T) {
		err := mgr.ValidateSSHAgent("/non/existent/socket")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SSH agent socket not accessible")
		assert.Contains(t, err.Error(), "Ensure SSH agent is running")
	})

	t.Run("existing socket file", func(t *testing.T) {
		// Create a temporary file to simulate SSH agent socket
		tmpDir := t.TempDir()
		socketPath := filepath.Join(tmpDir, "ssh-agent.sock")
		file, err := os.Create(socketPath)
		require.NoError(t, err)
		file.Close()

		err = mgr.ValidateSSHAgent(socketPath)
		assert.NoError(t, err)
	})
}

func TestPrepareSSHMounts(t *testing.T) {
	logger := &SimpleLogger{}
	mgr := NewManager(logger).(*manager)

	t.Run("SSH agent disabled", func(t *testing.T) {
		mounts, err := mgr.PrepareSSHMounts(false, "")
		assert.NoError(t, err)
		assert.Empty(t, mounts)
	})

	t.Run("SSH agent enabled with socket", func(t *testing.T) {
		socketPath := "/tmp/ssh-agent.sock"
		mounts, err := mgr.PrepareSSHMounts(true, socketPath)
		assert.NoError(t, err)
		
		// Should have at least the SSH agent socket mount
		assert.NotEmpty(t, mounts)
		
		// Find SSH agent socket mount
		var agentMount *pkg.Mount
		for i, mount := range mounts {
			if mount.Source == socketPath && mount.Target == "/ssh-agent.sock" {
				agentMount = &mounts[i]
				break
			}
		}
		
		require.NotNil(t, agentMount, "SSH agent socket mount should be present")
		assert.Equal(t, socketPath, agentMount.Source)
		assert.Equal(t, "/ssh-agent.sock", agentMount.Target)
		assert.Equal(t, "bind", agentMount.Type)
		assert.True(t, agentMount.ReadOnly)
	})

	t.Run("SSH agent enabled with SSH config files", func(t *testing.T) {
		// Create temporary SSH directory with config files
		tmpDir := t.TempDir()
		sshDir := filepath.Join(tmpDir, ".ssh")
		err := os.MkdirAll(sshDir, 0755)
		require.NoError(t, err)

		// Create SSH config file
		configFile := filepath.Join(sshDir, "config")
		err = os.WriteFile(configFile, []byte("Host example.com\n  User git\n"), 0644)
		require.NoError(t, err)

		// Create known_hosts file
		knownHostsFile := filepath.Join(sshDir, "known_hosts")
		err = os.WriteFile(knownHostsFile, []byte("example.com ssh-rsa AAAAB3...\n"), 0644)
		require.NoError(t, err)

		// Create .gitconfig file
		gitConfigFile := filepath.Join(tmpDir, ".gitconfig")
		err = os.WriteFile(gitConfigFile, []byte("[user]\n  name = Test User\n"), 0644)
		require.NoError(t, err)

		// Temporarily change HOME to our test directory
		originalHome := os.Getenv("HOME")
		defer os.Setenv("HOME", originalHome)
		os.Setenv("HOME", tmpDir)

		socketPath := "/tmp/ssh-agent.sock"
		mounts, err := mgr.PrepareSSHMounts(true, socketPath)
		assert.NoError(t, err)
		assert.NotEmpty(t, mounts)

		// Should have SSH agent socket, config, known_hosts, and gitconfig mounts
		expectedMounts := map[string]string{
			socketPath:       "/ssh-agent.sock",
			configFile:      "/home/claude/.ssh/config",
			knownHostsFile:  "/home/claude/.ssh/known_hosts",
			gitConfigFile:   "/home/claude/.gitconfig",
		}

		actualMounts := make(map[string]string)
		for _, mount := range mounts {
			actualMounts[mount.Source] = mount.Target
		}

		for source, expectedTarget := range expectedMounts {
			actualTarget, exists := actualMounts[source]
			assert.True(t, exists, "Mount for %s should exist", source)
			assert.Equal(t, expectedTarget, actualTarget, "Target for %s should be %s", source, expectedTarget)
		}

		// All mounts should be read-only
		for _, mount := range mounts {
			assert.True(t, mount.ReadOnly, "Mount %s -> %s should be read-only", mount.Source, mount.Target)
			assert.Equal(t, "bind", mount.Type, "Mount %s -> %s should be bind type", mount.Source, mount.Target)
		}
	})
}

func TestSSHAgentConfigurationPersistence(t *testing.T) {
	logger := &SimpleLogger{}
	mgr := NewManager(logger).(*manager)

	t.Run("save and load SSH agent configuration", func(t *testing.T) {
		// Use temporary directory for config file
		originalDir, _ := os.Getwd()
		tmpDir := t.TempDir()
		defer os.Chdir(originalDir)
		os.Chdir(tmpDir)

		// Create config with SSH agent enabled
		config := &pkg.Config{
			Variant:        "base",
			Account:        "test",
			SSHAgent:       true,
			SSHAgentSocket: "/tmp/ssh-agent.sock",
		}

		// Save configuration
		err := mgr.SaveConfig(config)
		assert.NoError(t, err)

		// Load configuration
		loadedConfig, err := mgr.LoadConfig()
		assert.NoError(t, err)
		assert.True(t, loadedConfig.SSHAgent)
		assert.Equal(t, "/tmp/ssh-agent.sock", loadedConfig.SSHAgentSocket)
	})

	t.Run("load configuration with SSH agent disabled", func(t *testing.T) {
		// Use temporary directory for config file
		originalDir, _ := os.Getwd()
		tmpDir := t.TempDir()
		defer os.Chdir(originalDir)
		os.Chdir(tmpDir)

		// Create config with SSH agent disabled (default)
		config := &pkg.Config{
			Variant: "base",
			Account: "test",
			// SSHAgent defaults to false
		}

		// Save configuration
		err := mgr.SaveConfig(config)
		assert.NoError(t, err)

		// Load configuration
		loadedConfig, err := mgr.LoadConfig()
		assert.NoError(t, err)
		assert.False(t, loadedConfig.SSHAgent)
		assert.Empty(t, loadedConfig.SSHAgentSocket)
	})

	t.Run("parse existing configuration file with SSH agent settings", func(t *testing.T) {
		// Use temporary directory for config file
		originalDir, _ := os.Getwd()
		tmpDir := t.TempDir()
		defer os.Chdir(originalDir)
		os.Chdir(tmpDir)

		// Create a .claude-reactor file with SSH agent settings
		configContent := `variant=go
account=work
ssh_agent=true
ssh_agent_socket=auto
`
		err := os.WriteFile(".claude-reactor", []byte(configContent), 0644)
		require.NoError(t, err)

		// Load configuration
		config, err := mgr.LoadConfig()
		assert.NoError(t, err)
		assert.Equal(t, "go", config.Variant)
		assert.Equal(t, "work", config.Account)
		assert.True(t, config.SSHAgent)
		assert.Equal(t, "auto", config.SSHAgentSocket)
	})
}