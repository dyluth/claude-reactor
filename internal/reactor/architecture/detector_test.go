package architecture

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	
	"claude-reactor/pkg"
)

// MockLogger implements the Logger interface for testing
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(args ...interface{})                            { m.Called(args...) }
func (m *MockLogger) Info(args ...interface{})                             { m.Called(args...) }
func (m *MockLogger) Warn(args ...interface{})                             { m.Called(args...) }
func (m *MockLogger) Error(args ...interface{})                            { m.Called(args...) }
func (m *MockLogger) Fatal(args ...interface{})                            { m.Called(args...) }
func (m *MockLogger) Debugf(format string, args ...interface{})            { m.Called(format, args) }
func (m *MockLogger) Infof(format string, args ...interface{})             { m.Called(format, args) }
func (m *MockLogger) Warnf(format string, args ...interface{})             { m.Called(format, args) }
func (m *MockLogger) Errorf(format string, args ...interface{})            { m.Called(format, args) }
func (m *MockLogger) Fatalf(format string, args ...interface{})            { m.Called(format, args) }
func (m *MockLogger) WithField(key string, value interface{}) pkg.Logger   { return m }
func (m *MockLogger) WithFields(fields map[string]interface{}) pkg.Logger  { return m }

func TestDetector_GetHostArchitecture(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		wantErr  bool
	}{
		{
			name:     "current runtime architecture",
			expected: runtime.GOARCH,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLogger := &MockLogger{}
			mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Return()
			
			d := NewDetector(mockLogger)
			
			arch, err := d.GetHostArchitecture()
			
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			
			assert.NoError(t, err)
			assert.NotEmpty(t, arch)
			
			// Verify it's one of the supported architectures
			supportedArch := []string{"amd64", "arm64", "i386", "arm"}
			assert.Contains(t, supportedArch, arch)
		})
	}
}

func TestDetector_GetDockerPlatform(t *testing.T) {
	tests := []struct {
		name     string
		expected map[string]string // maps architecture to expected platform
	}{
		{
			name: "architecture to platform mapping",
			expected: map[string]string{
				"amd64": "linux/amd64",
				"arm64": "linux/arm64",
				"i386":  "linux/386",
				"arm":   "linux/arm/v7",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLogger := &MockLogger{}
			mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Return()
			
			d := NewDetector(mockLogger)
			
			platform, err := d.GetDockerPlatform()
			assert.NoError(t, err)
			assert.NotEmpty(t, platform)
			assert.True(t, platform == "linux/amd64" || platform == "linux/arm64" || 
				platform == "linux/386" || platform == "linux/arm/v7")
		})
	}
}

func TestDetector_IsMultiArchSupported(t *testing.T) {
	mockLogger := &MockLogger{}
	d := NewDetector(mockLogger)
	
	supported := d.IsMultiArchSupported()
	assert.True(t, supported, "Multi-arch should be supported by default")
}

func TestGetContainerName(t *testing.T) {
	tests := []struct {
		name      string
		variant   string
		account   string
		mockArch  string
		expected  string
		wantErr   bool
	}{
		{
			name:     "with account",
			variant:  "go",
			account:  "work",
			mockArch: "arm64",
			expected: "claude-reactor-go-arm64-work",
			wantErr:  false,
		},
		{
			name:     "without account uses default",
			variant:  "base",
			account:  "",
			mockArch: "amd64",
			expected: "claude-reactor-base-amd64-default",
			wantErr:  false,
		},
		{
			name:     "cloud variant with personal account",
			variant:  "cloud",
			account:  "personal",
			mockArch: "arm64",
			expected: "claude-reactor-cloud-arm64-personal",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock architecture detector
			mockArchDetector := &MockArchDetector{}
			mockArchDetector.On("GetHostArchitecture").Return(tt.mockArch, nil)
			
			name, err := GetContainerName(tt.variant, tt.account, mockArchDetector)
			
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, name)
		})
	}
}

func TestGetImageName(t *testing.T) {
	tests := []struct {
		name     string
		variant  string
		mockArch string
		expected string
		wantErr  bool
	}{
		{
			name:     "go variant arm64",
			variant:  "go",
			mockArch: "arm64",
			expected: "claude-reactor-go:arm64",
			wantErr:  false,
		},
		{
			name:     "base variant amd64",
			variant:  "base",
			mockArch: "amd64",
			expected: "claude-reactor-base:amd64",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockArchDetector := &MockArchDetector{}
			mockArchDetector.On("GetHostArchitecture").Return(tt.mockArch, nil)
			
			name, err := GetImageName(tt.variant, mockArchDetector)
			
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, name)
		})
	}
}

// MockArchDetector for testing utility functions
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

// Benchmark tests
func BenchmarkDetector_GetHostArchitecture(b *testing.B) {
	mockLogger := &MockLogger{}
	mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Return()
	
	d := NewDetector(mockLogger)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = d.GetHostArchitecture()
	}
}

func BenchmarkDetector_GetDockerPlatform(b *testing.B) {
	mockLogger := &MockLogger{}
	mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Return()
	
	d := NewDetector(mockLogger)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = d.GetDockerPlatform()
	}
}