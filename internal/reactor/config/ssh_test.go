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
		
		// SSH agent socket mounting is intentionally skipped due to Docker Desktop limitations
		// So we should NOT find an SSH agent socket mount
		for _, mount := range mounts {
			assert.NotEqual(t, socketPath, mount.Source, "SSH agent socket should not be mounted due to Docker Desktop limitations")
			assert.NotEqual(t, "/ssh-agent.sock", mount.Target, "SSH agent socket mount should not be present")
		}
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

		// SSH agent socket should NOT be mounted (due to Docker Desktop limitations)
		for _, mount := range mounts {
			assert.NotEqual(t, socketPath, mount.Source, "SSH agent socket should not be mounted")
			assert.NotEqual(t, "/ssh-agent.sock", mount.Target, "SSH agent socket mount should not be present")
		}

		// Should have config, known_hosts, and gitconfig mounts
		// Note: SSH config will be a filtered temporary file, not the original
		expectedTargets := map[string]bool{
			"/home/claude/.ssh/config":      false,
			"/home/claude/.ssh/known_hosts": false,
			"/home/claude/.gitconfig":       false,
		}

		actualMounts := make(map[string]string)
		for _, mount := range mounts {
			actualMounts[mount.Source] = mount.Target
			if _, exists := expectedTargets[mount.Target]; exists {
				expectedTargets[mount.Target] = true
			}
		}

		// Verify that known_hosts and gitconfig are mounted with original sources
		assert.True(t, expectedTargets["/home/claude/.ssh/known_hosts"], "known_hosts should be mounted")
		assert.True(t, expectedTargets["/home/claude/.gitconfig"], "gitconfig should be mounted")
		assert.True(t, expectedTargets["/home/claude/.ssh/config"], "SSH config should be mounted")

		// Verify known_hosts and gitconfig use original files as source
		assert.Equal(t, "/home/claude/.ssh/known_hosts", actualMounts[knownHostsFile])
		assert.Equal(t, "/home/claude/.gitconfig", actualMounts[gitConfigFile])

		// SSH config should use a temporary filtered file (not the original)
		var configMountSource string
		for source, target := range actualMounts {
			if target == "/home/claude/.ssh/config" {
				configMountSource = source
				break
			}
		}
		assert.NotEmpty(t, configMountSource, "SSH config mount should exist")
		assert.NotEqual(t, configFile, configMountSource, "SSH config should use filtered temporary file, not original")
		assert.Contains(t, configMountSource, "claude-reactor-ssh-config", "SSH config source should be a claude-reactor temporary file")

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