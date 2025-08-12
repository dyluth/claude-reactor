package docker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	
	"claude-reactor/pkg"
)

func TestNewMountManager(t *testing.T) {
	mockLogger := &MockLogger{}
	
	mm := NewMountManager(mockLogger)
	
	assert.NotNil(t, mm)
	assert.Equal(t, mockLogger, mm.logger)
}

func TestMountManager_ExpandPath(t *testing.T) {
	mockLogger := &MockLogger{}
	mm := NewMountManager(mockLogger)
	
	// Set HOME for testing
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", "/test/home")
	
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "expand tilde path",
			input:    "~/Documents",
			expected: "/test/home/Documents",
		},
		{
			name:     "expand tilde root",
			input:    "~",
			expected: "~", // expandPath only handles ~/... not bare ~
		},
		{
			name:     "absolute path unchanged",
			input:    "/absolute/path",
			expected: "/absolute/path",
		},
		{
			name:     "relative path unchanged",
			input:    "relative/path",
			expected: "relative/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mm.expandPath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMountManager_CreateDefaultMounts(t *testing.T) {
	mockLogger := &MockLogger{}
	mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
	mockLogger.On("Warnf", mock.AnythingOfType("string"), mock.Anything).Maybe()
	
	mm := NewMountManager(mockLogger)
	
	t.Run("create default mounts", func(t *testing.T) {
		mounts, err := mm.CreateDefaultMounts("default")
		
		assert.NoError(t, err)
		assert.NotEmpty(t, mounts)
		
		// Should always have project mount
		foundProjectMount := false
		for _, mount := range mounts {
			if mount.Target == "/app" {
				foundProjectMount = true
				assert.Equal(t, "bind", mount.Type)
				assert.False(t, mount.ReadOnly)
				break
			}
		}
		assert.True(t, foundProjectMount, "Should have project mount to /app")
	})
	
	t.Run("account-specific mounts", func(t *testing.T) {
		mounts, err := mm.CreateDefaultMounts("work")
		
		assert.NoError(t, err)
		assert.NotEmpty(t, mounts)
		// Account-specific logic is tested separately
	})
}

func TestMountManager_AddUserMounts(t *testing.T) {
	mockLogger := &MockLogger{}
	mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
	mockLogger.On("Warnf", mock.AnythingOfType("string"), mock.Anything).Maybe()
	
	mm := NewMountManager(mockLogger)
	
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "claude-reactor-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	// Create a subdirectory to mount
	testDir := filepath.Join(tempDir, "test-mount")
	err = os.MkdirAll(testDir, 0755)
	assert.NoError(t, err)
	
	t.Run("add existing directory mount", func(t *testing.T) {
		baseMounts := []pkg.Mount{
			{Source: "/base", Target: "/base", Type: "bind"},
		}
		
		mountPaths := []string{testDir}
		result, err := mm.AddUserMounts(baseMounts, mountPaths)
		
		assert.NoError(t, err)
		assert.Len(t, result, 2) // 1 base + 1 added
		
		// Check the added mount
		addedMount := result[1]
		assert.Equal(t, testDir, addedMount.Source)
		assert.Equal(t, "/mnt/test-mount", addedMount.Target)
		assert.Equal(t, "bind", addedMount.Type)
		assert.False(t, addedMount.ReadOnly)
	})
	
	t.Run("handle non-existent directory", func(t *testing.T) {
		baseMounts := []pkg.Mount{}
		mountPaths := []string{"/non-existent-directory"}
		
		result, err := mm.AddUserMounts(baseMounts, mountPaths)
		
		assert.NoError(t, err)
		assert.Len(t, result, 0) // Should skip non-existent directory
	})
	
	t.Run("handle relative path", func(t *testing.T) {
		// Create a relative directory in current working directory
		currentDir, err := os.Getwd()
		assert.NoError(t, err)
		
		relativeTestDir := filepath.Join(currentDir, "relative-test")
		err = os.MkdirAll(relativeTestDir, 0755)
		assert.NoError(t, err)
		defer os.RemoveAll(relativeTestDir)
		
		baseMounts := []pkg.Mount{}
		mountPaths := []string{"relative-test"}
		
		result, err := mm.AddUserMounts(baseMounts, mountPaths)
		
		assert.NoError(t, err)
		assert.Len(t, result, 1)
		
		// Should be converted to absolute path
		assert.Equal(t, relativeTestDir, result[0].Source)
		assert.Equal(t, "/mnt/relative-test", result[0].Target)
	})
}

func TestMountManager_ValidateMounts(t *testing.T) {
	mockLogger := &MockLogger{}
	mm := NewMountManager(mockLogger)
	
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "claude-reactor-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	t.Run("validate existing mounts", func(t *testing.T) {
		mounts := []pkg.Mount{
			{Source: tempDir, Target: "/test", Type: "bind"},
			{Source: "/var/run/docker.sock", Target: "/var/run/docker.sock", Type: "bind"}, // Should skip validation
		}
		
		err := mm.ValidateMounts(mounts)
		assert.NoError(t, err)
	})
	
	t.Run("fail on non-existent mount", func(t *testing.T) {
		mounts := []pkg.Mount{
			{Source: "/non-existent-path", Target: "/test", Type: "bind"},
		}
		
		err := mm.ValidateMounts(mounts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "mount source does not exist")
	})
}

func TestMountManager_ConvertToDockerMounts(t *testing.T) {
	mockLogger := &MockLogger{}
	mm := NewMountManager(mockLogger)
	
	pkgMounts := []pkg.Mount{
		{Source: "/source1", Target: "/target1", Type: "bind", ReadOnly: false},
		{Source: "/source2", Target: "/target2", Type: "bind", ReadOnly: true},
	}
	
	dockerMounts := mm.ConvertToDockerMounts(pkgMounts)
	
	assert.Len(t, dockerMounts, 2)
	
	assert.Equal(t, "/source1", dockerMounts[0].Source)
	assert.Equal(t, "/target1", dockerMounts[0].Target)
	assert.False(t, dockerMounts[0].ReadOnly)
	
	assert.Equal(t, "/source2", dockerMounts[1].Source)
	assert.Equal(t, "/target2", dockerMounts[1].Target)
	assert.True(t, dockerMounts[1].ReadOnly)
}

func TestMountManager_GetMountSummary(t *testing.T) {
	mockLogger := &MockLogger{}
	mm := NewMountManager(mockLogger)
	
	mounts := []pkg.Mount{
		{Source: "/source1", Target: "/target1", Type: "bind", ReadOnly: false},
		{Source: "/source2", Target: "/target2", Type: "bind", ReadOnly: true},
	}
	
	summary := mm.GetMountSummary(mounts)
	
	assert.Len(t, summary, 2)
	assert.Equal(t, "/source1 -> /target1", summary[0])
	assert.Equal(t, "/source2 -> /target2 (read-only)", summary[1])
}

func TestMountManager_CreateClaudeConfigMounts(t *testing.T) {
	mockLogger := &MockLogger{}
	mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
	
	mm := NewMountManager(mockLogger)
	
	// Set up test environment
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	
	tempHome, err := os.MkdirTemp("", "claude-reactor-home-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempHome)
	os.Setenv("HOME", tempHome)
	
	// Create test Claude config files
	claudeJSON := filepath.Join(tempHome, ".claude.json")
	claudeDir := filepath.Join(tempHome, ".claude")
	
	err = os.WriteFile(claudeJSON, []byte(`{"test": true}`), 0644)
	assert.NoError(t, err)
	
	err = os.MkdirAll(claudeDir, 0755)
	assert.NoError(t, err)
	
	t.Run("default account mounts", func(t *testing.T) {
		mounts, err := mm.createClaudeConfigMounts("")
		
		assert.NoError(t, err)
		assert.Len(t, mounts, 2) // .claude.json + .claude/
		
		// Check .claude.json mount
		found := false
		for _, mount := range mounts {
			if mount.Target == "/home/claude/.claude.json" {
				assert.Equal(t, claudeJSON, mount.Source)
				found = true
				break
			}
		}
		assert.True(t, found, "Should mount .claude.json")
		
		// Check .claude/ mount
		found = false
		for _, mount := range mounts {
			if mount.Target == "/home/claude/.claude" {
				assert.Equal(t, claudeDir, mount.Source)
				found = true
				break
			}
		}
		assert.True(t, found, "Should mount .claude directory")
	})
	
	t.Run("named account mounts", func(t *testing.T) {
		// Create account-specific directories
		reactorDir := filepath.Join(tempHome, ".claude-reactor")
		err := os.MkdirAll(reactorDir, 0755)
		assert.NoError(t, err)
		
		accountClaudeJSON := filepath.Join(reactorDir, ".work-claude.json")
		err = os.WriteFile(accountClaudeJSON, []byte(`{"account": "work"}`), 0644)
		assert.NoError(t, err)
		
		mounts, err := mm.createClaudeConfigMounts("work")
		
		assert.NoError(t, err)
		assert.Len(t, mounts, 1) // Only account-specific .claude.json
		
		mount := mounts[0]
		assert.Equal(t, accountClaudeJSON, mount.Source)
		assert.Equal(t, "/home/claude/.claude.json", mount.Target)
	})
}

func BenchmarkMountManager_CreateDefaultMounts(b *testing.B) {
	mockLogger := &MockLogger{}
	mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
	mockLogger.On("Warnf", mock.AnythingOfType("string"), mock.Anything).Maybe()
	
	mm := NewMountManager(mockLogger)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mm.CreateDefaultMounts("default")
	}
}

func BenchmarkMountManager_ConvertToDockerMounts(b *testing.B) {
	mockLogger := &MockLogger{}
	mm := NewMountManager(mockLogger)
	
	pkgMounts := []pkg.Mount{
		{Source: "/source1", Target: "/target1", Type: "bind", ReadOnly: false},
		{Source: "/source2", Target: "/target2", Type: "bind", ReadOnly: true},
		{Source: "/source3", Target: "/target3", Type: "bind", ReadOnly: false},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mm.ConvertToDockerMounts(pkgMounts)
	}
}