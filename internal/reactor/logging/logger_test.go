package logging

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewLogger(t *testing.T) {
	logger := NewLogger()
	assert.NotNil(t, logger)
	
	// Test that it implements the interface
	logger.Info("Test message")
	logger.Debugf("Test formatted message: %s", "test")
}

func TestGetLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected logrus.Level
	}{
		{
			name:     "debug level",
			envValue: "DEBUG",
			expected: logrus.DebugLevel,
		},
		{
			name:     "info level",
			envValue: "INFO", 
			expected: logrus.InfoLevel,
		},
		{
			name:     "warn level",
			envValue: "WARN",
			expected: logrus.WarnLevel,
		},
		{
			name:     "warning level",
			envValue: "WARNING",
			expected: logrus.WarnLevel,
		},
		{
			name:     "error level",
			envValue: "ERROR",
			expected: logrus.ErrorLevel,
		},
		{
			name:     "fatal level",
			envValue: "FATAL",
			expected: logrus.FatalLevel,
		},
		{
			name:     "default level (empty)",
			envValue: "",
			expected: logrus.InfoLevel,
		},
		{
			name:     "default level (invalid)",
			envValue: "INVALID",
			expected: logrus.InfoLevel,
		},
		{
			name:     "case insensitive",
			envValue: "debug",
			expected: logrus.DebugLevel, // Should work case-insensitively
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			if tt.envValue != "" {
				os.Setenv("CLAUDE_REACTOR_LOG_LEVEL", tt.envValue)
				defer os.Unsetenv("CLAUDE_REACTOR_LOG_LEVEL")
			}
			
			level := getLogLevel()
			assert.Equal(t, tt.expected, level)
		})
	}
}

func TestLogger_WithField(t *testing.T) {
	logger := NewLogger()
	
	fieldLogger := logger.WithField("test", "value")
	assert.NotNil(t, fieldLogger)
	
	// Test that it still implements the interface
	fieldLogger.Info("Test message with field")
}

func TestLogger_WithFields(t *testing.T) {
	logger := NewLogger()
	
	fields := map[string]interface{}{
		"field1": "value1",
		"field2": 42,
		"field3": true,
	}
	
	fieldsLogger := logger.WithFields(fields)
	assert.NotNil(t, fieldsLogger)
	
	// Test that it still implements the interface
	fieldsLogger.Info("Test message with multiple fields")
}

func TestLogger_AllMethods(t *testing.T) {
	logger := NewLogger()
	
	// Test all logging methods don't panic
	assert.NotPanics(t, func() {
		logger.Debug("Debug message")
		logger.Info("Info message")
		logger.Warn("Warning message")
		logger.Error("Error message")
		
		logger.Debugf("Debug formatted: %s", "test")
		logger.Infof("Info formatted: %d", 42)
		logger.Warnf("Warning formatted: %t", true)
		logger.Errorf("Error formatted: %v", []string{"a", "b"})
	})
}

func BenchmarkLogger_Info(b *testing.B) {
	logger := NewLogger()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("Benchmark test message")
	}
}

func BenchmarkLogger_WithField(b *testing.B) {
	logger := NewLogger()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.WithField("test", "value").Info("Benchmark test message")
	}
}