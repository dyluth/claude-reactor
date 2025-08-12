package docker

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	
	"claude-reactor/pkg"
)

// MockLogger for testing
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(args ...interface{})                           { m.Called(args...) }
func (m *MockLogger) Info(args ...interface{})                            { m.Called(args...) }
func (m *MockLogger) Warn(args ...interface{})                            { m.Called(args...) }
func (m *MockLogger) Error(args ...interface{})                           { m.Called(args...) }
func (m *MockLogger) Fatal(args ...interface{})                           { m.Called(args...) }
func (m *MockLogger) Debugf(format string, args ...interface{})           { m.Called(format, args) }
func (m *MockLogger) Infof(format string, args ...interface{})            { m.Called(format, args) }
func (m *MockLogger) Warnf(format string, args ...interface{})            { m.Called(format, args) }
func (m *MockLogger) Errorf(format string, args ...interface{})           { m.Called(format, args) }
func (m *MockLogger) Fatalf(format string, args ...interface{})           { m.Called(format, args) }
func (m *MockLogger) WithField(key string, value interface{}) pkg.Logger  { return m }
func (m *MockLogger) WithFields(fields map[string]interface{}) pkg.Logger { return m }

// MockArchDetector for testing
type MockArchDetector struct {
	mock.Mock
}

func (m *MockArchDetector) GetHostArchitecture() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockArchDetector) GetDockerPlatform() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockArchDetector) IsMultiArchSupported() bool {
	args := m.Called()
	return args.Bool(0)
}

func TestNamingManager_GetImageName(t *testing.T) {
	tests := []struct {
		name     string
		variant  string
		mockArch string
		expected string
		wantErr  bool
	}{
		{
			name:     "go variant with arm64",
			variant:  "go",
			mockArch: "arm64",
			expected: "claude-reactor-go-arm64",
			wantErr:  false,
		},
		{
			name:     "base variant with amd64",
			variant:  "base",
			mockArch: "amd64",
			expected: "claude-reactor-base-amd64",
			wantErr:  false,
		},
		{
			name:     "full variant with arm64",
			variant:  "full",
			mockArch: "arm64",
			expected: "claude-reactor-full-arm64",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLogger := &MockLogger{}
			mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Return()
			
			mockArchDetector := &MockArchDetector{}
			mockArchDetector.On("GetHostArchitecture").Return(tt.mockArch, nil)
			
			nm := NewNamingManager(mockLogger, mockArchDetector)
			
			result, err := nm.GetImageName(tt.variant)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
			
			mockArchDetector.AssertExpectations(t)
		})
	}
}

func TestNamingManager_GetContainerName(t *testing.T) {
	tests := []struct {
		name     string
		variant  string
		account  string
		mockArch string
		wantErr  bool
		validate func(t *testing.T, result string)
	}{
		{
			name:     "go variant with work account",
			variant:  "go",
			account:  "work",
			mockArch: "arm64",
			wantErr:  false,
			validate: func(t *testing.T, result string) {
				assert.Contains(t, result, "claude-reactor-go-arm64")
				assert.Contains(t, result, "work")
				// Should have project hash (8 characters)
				parts := strings.Split(result, "-")
				assert.Len(t, parts, 6) // claude-reactor-go-arm64-{hash}-work
				assert.Len(t, parts[4], 8) // project hash should be 8 chars at index 4
			},
		},
		{
			name:     "base variant with empty account (uses default)",
			variant:  "base",
			account:  "",
			mockArch: "amd64",
			wantErr:  false,
			validate: func(t *testing.T, result string) {
				assert.Contains(t, result, "claude-reactor-base-amd64")
				assert.Contains(t, result, "default")
			},
		},
		{
			name:     "full variant with personal account",
			variant:  "full",
			account:  "personal",
			mockArch: "arm64",
			wantErr:  false,
			validate: func(t *testing.T, result string) {
				assert.Contains(t, result, "claude-reactor-full-arm64")
				assert.Contains(t, result, "personal")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLogger := &MockLogger{}
			mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Return()
			
			mockArchDetector := &MockArchDetector{}
			mockArchDetector.On("GetHostArchitecture").Return(tt.mockArch, nil)
			
			nm := NewNamingManager(mockLogger, mockArchDetector)
			
			result, err := nm.GetContainerName(tt.variant, tt.account)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, result)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
			
			mockArchDetector.AssertExpectations(t)
		})
	}
}

func TestNamingManager_GetProjectHash(t *testing.T) {
	mockLogger := &MockLogger{}
	mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Return()
	
	mockArchDetector := &MockArchDetector{}
	
	nm := NewNamingManager(mockLogger, mockArchDetector)
	
	hash, err := nm.getProjectHash()
	
	assert.NoError(t, err)
	assert.Len(t, hash, 8, "Project hash should be 8 characters long")
	assert.Regexp(t, "^[a-f0-9]{8}$", hash, "Project hash should be 8 lowercase hex characters")
}

func BenchmarkNamingManager_GetContainerName(b *testing.B) {
	mockLogger := &MockLogger{}
	mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Return()
	
	mockArchDetector := &MockArchDetector{}
	mockArchDetector.On("GetHostArchitecture").Return("arm64", nil)
	
	nm := NewNamingManager(mockLogger, mockArchDetector)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = nm.GetContainerName("go", "test")
	}
}