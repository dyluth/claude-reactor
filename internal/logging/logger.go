package logging

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	
	"claude-reactor/pkg"
)

// logger wraps logrus to implement our Logger interface
type logger struct {
	*logrus.Logger
}

// NewLogger creates a new structured logger with appropriate configuration
func NewLogger() pkg.Logger {
	l := logrus.New()
	
	// Set output format
	l.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "15:04:05",
		ForceColors:     true,
		DisableQuote:    true,
	})
	
	// Set log level based on environment
	level := getLogLevel()
	l.SetLevel(level)
	
	// Set output to stdout (not stderr) for better CLI experience
	l.SetOutput(os.Stdout)
	
	return &logger{Logger: l}
}

// getLogLevel determines the appropriate log level from environment variables
func getLogLevel() logrus.Level {
	// Check for CLAUDE_REACTOR_LOG_LEVEL environment variable
	envLevel := strings.ToUpper(os.Getenv("CLAUDE_REACTOR_LOG_LEVEL"))
	
	switch envLevel {
	case "DEBUG":
		return logrus.DebugLevel
	case "INFO":
		return logrus.InfoLevel
	case "WARN", "WARNING":
		return logrus.WarnLevel
	case "ERROR":
		return logrus.ErrorLevel
	case "FATAL":
		return logrus.FatalLevel
	default:
		// Default to INFO level for production use
		return logrus.InfoLevel
	}
}

// WithField creates a new logger with a single field
func (l *logger) WithField(key string, value interface{}) pkg.Logger {
	return &logger{Logger: l.Logger.WithField(key, value).Logger}
}

// WithFields creates a new logger with multiple fields
func (l *logger) WithFields(fields map[string]interface{}) pkg.Logger {
	logrusFields := make(logrus.Fields)
	for k, v := range fields {
		logrusFields[k] = v
	}
	return &logger{Logger: l.Logger.WithFields(logrusFields).Logger}
}

// Implement the pkg.Logger interface methods that aren't directly available from logrus

// Debug logs a debug message
func (l *logger) Debug(args ...interface{}) {
	l.Logger.Debug(args...)
}

// Info logs an info message
func (l *logger) Info(args ...interface{}) {
	l.Logger.Info(args...)
}

// Warn logs a warning message
func (l *logger) Warn(args ...interface{}) {
	l.Logger.Warn(args...)
}

// Error logs an error message
func (l *logger) Error(args ...interface{}) {
	l.Logger.Error(args...)
}

// Fatal logs a fatal message and exits
func (l *logger) Fatal(args ...interface{}) {
	l.Logger.Fatal(args...)
}

// Debugf logs a formatted debug message
func (l *logger) Debugf(format string, args ...interface{}) {
	l.Logger.Debugf(format, args...)
}

// Infof logs a formatted info message
func (l *logger) Infof(format string, args ...interface{}) {
	l.Logger.Infof(format, args...)
}

// Warnf logs a formatted warning message
func (l *logger) Warnf(format string, args ...interface{}) {
	l.Logger.Warnf(format, args...)
}

// Errorf logs a formatted error message
func (l *logger) Errorf(format string, args ...interface{}) {
	l.Logger.Errorf(format, args...)
}

// Fatalf logs a formatted fatal message and exits
func (l *logger) Fatalf(format string, args ...interface{}) {
	l.Logger.Fatalf(format, args...)
}