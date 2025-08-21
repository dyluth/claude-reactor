package validation

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"claude-reactor/pkg"
)

// MockLogger for testing
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(args ...interface{}) { m.Called(args) }
func (m *MockLogger) Info(args ...interface{})  { m.Called(args) }
func (m *MockLogger) Warn(args ...interface{})  { m.Called(args) }
func (m *MockLogger) Error(args ...interface{}) { m.Called(args) }
func (m *MockLogger) Fatal(args ...interface{}) { m.Called(args) }

func (m *MockLogger) Debugf(format string, args ...interface{}) { m.Called(format, args) }
func (m *MockLogger) Infof(format string, args ...interface{})  { m.Called(format, args) }
func (m *MockLogger) Warnf(format string, args ...interface{})  { m.Called(format, args) }
func (m *MockLogger) Errorf(format string, args ...interface{}) { m.Called(format, args) }
func (m *MockLogger) Fatalf(format string, args ...interface{}) { m.Called(format, args) }

func (m *MockLogger) WithField(key string, value interface{}) pkg.Logger {
	args := m.Called(key, value)
	return args.Get(0).(pkg.Logger)
}

func (m *MockLogger) WithFields(fields map[string]interface{}) pkg.Logger {
	args := m.Called(fields)
	return args.Get(0).(pkg.Logger)
}

func TestRecommendedPackages(t *testing.T) {
	// Test that recommended packages are properly defined
	assert.Greater(t, len(recommendedPackages), 5, "Should have several recommended packages")

	// Test high priority packages exist
	var highPriorityCount int
	var mediumPriorityCount int
	var lowPriorityCount int
	
	for _, pkg := range recommendedPackages {
		assert.NotEmpty(t, pkg.Name, "Package name should not be empty")
		assert.NotEmpty(t, pkg.Command, "Package command should not be empty")
		assert.NotEmpty(t, pkg.Description, "Package description should not be empty")
		assert.NotEmpty(t, pkg.Category, "Package category should not be empty")
		assert.Contains(t, []string{"high", "medium", "low"}, pkg.Priority, "Package priority should be valid")

		switch pkg.Priority {
		case "high":
			highPriorityCount++
		case "medium":
			mediumPriorityCount++
		case "low":
			lowPriorityCount++
		}
	}

	assert.Greater(t, highPriorityCount, 0, "Should have at least one high priority package")
	assert.Greater(t, mediumPriorityCount, 0, "Should have at least one medium priority package")
	assert.Greater(t, lowPriorityCount, 0, "Should have at least one low priority package")

	// Test specific critical packages
	packageNames := make(map[string]bool)
	for _, pkg := range recommendedPackages {
		packageNames[pkg.Name] = true
	}

	assert.True(t, packageNames["git"], "Should include git")
	assert.True(t, packageNames["curl"], "Should include curl")
	assert.True(t, packageNames["make"], "Should include make")
	assert.True(t, packageNames["nano"], "Should include nano")
}

func TestRecommendedPackageCategories(t *testing.T) {
	categories := make(map[string]int)
	for _, pkg := range recommendedPackages {
		categories[pkg.Category]++
	}

	// Test that we have diverse categories
	assert.Greater(t, categories["version-control"], 0, "Should have version control tools")
	assert.Greater(t, categories["network"], 0, "Should have network tools")
	assert.Greater(t, categories["text-processing"], 0, "Should have text processing tools")
	assert.Greater(t, categories["build"], 0, "Should have build tools")
	assert.Greater(t, categories["editors"], 0, "Should have editors")
}

func TestClearSessionWarnings(t *testing.T) {
	mockLogger := &MockLogger{}
	
	// Create a validator with some session warnings
	validator := &ImageValidator{
		logger:          mockLogger,
		cacheDir:        "/tmp/test-cache",
		sessionWarnings: make(map[string]bool),
	}

	// Add some session warnings
	validator.sessionWarnings["test-warning-1"] = true
	validator.sessionWarnings["test-warning-2"] = true

	assert.Len(t, validator.sessionWarnings, 2, "Should have 2 warnings")

	// Clear warnings
	validator.ClearSessionWarnings()

	assert.Len(t, validator.sessionWarnings, 0, "Should have no warnings after clearing")
	assert.NotNil(t, validator.sessionWarnings, "Session warnings map should still exist")
}

func TestAddPackageWarnings_OncePerSession(t *testing.T) {
	mockLogger := &MockLogger{}
	validator := &ImageValidator{
		logger:          mockLogger,
		cacheDir:        "/tmp/test-cache",
		sessionWarnings: make(map[string]bool),
	}

	result := &pkg.ImageValidationResult{
		Warnings: []string{},
	}

	missingHighPriority := []string{"git", "curl"}
	missingOther := []string{"vim", "nano"}

	// First call should add warnings
	validator.addPackageWarnings(result, missingHighPriority, missingOther)

	assert.Len(t, result.Warnings, 2, "Should add 2 warnings (missing tools + package check)")
	assert.Contains(t, result.Warnings[0], "git, curl")
	assert.Contains(t, result.Warnings[0], "Missing recommended tools")
	assert.Contains(t, result.Warnings[1], "4 recommended tools missing")

	// Verify session warning was tracked
	expectedKey := "missing-packages-git,curl"
	assert.True(t, validator.sessionWarnings[expectedKey], "Should track session warning")

	// Second call with same missing packages should not add duplicate warnings
	result2 := &pkg.ImageValidationResult{
		Warnings: []string{},
	}

	validator.addPackageWarnings(result2, missingHighPriority, missingOther)

	// Should only add the informational message, not the duplicate warning
	assert.Len(t, result2.Warnings, 1, "Should only add info message on second call")
	assert.Contains(t, result2.Warnings[0], "4 recommended tools missing")
	assert.NotContains(t, result2.Warnings[0], "Missing recommended tools")
}

func TestAddPackageWarnings_NoHighPriorityMissing(t *testing.T) {
	mockLogger := &MockLogger{}
	validator := &ImageValidator{
		logger:          mockLogger,
		cacheDir:        "/tmp/test-cache",
		sessionWarnings: make(map[string]bool),
	}

	result := &pkg.ImageValidationResult{
		Warnings: []string{},
	}

	missingHighPriority := []string{} // No high priority missing
	missingOther := []string{"vim", "nano"}

	validator.addPackageWarnings(result, missingHighPriority, missingOther)

	assert.Len(t, result.Warnings, 1, "Should only add info message")
	assert.Contains(t, result.Warnings[0], "2 recommended tools missing")
	assert.NotContains(t, result.Warnings[0], "Missing recommended tools")
}

func TestAddPackageWarnings_NoMissingPackages(t *testing.T) {
	mockLogger := &MockLogger{}
	validator := &ImageValidator{
		logger:          mockLogger,
		cacheDir:        "/tmp/test-cache",
		sessionWarnings: make(map[string]bool),
	}

	result := &pkg.ImageValidationResult{
		Warnings: []string{},
	}

	missingHighPriority := []string{}
	missingOther := []string{}

	validator.addPackageWarnings(result, missingHighPriority, missingOther)

	assert.Len(t, result.Warnings, 0, "Should not add any warnings when no packages missing")
}

func TestAddPackageWarnings_DifferentMissingPackages(t *testing.T) {
	mockLogger := &MockLogger{}
	validator := &ImageValidator{
		logger:          mockLogger,
		cacheDir:        "/tmp/test-cache",
		sessionWarnings: make(map[string]bool),
	}

	// First warning for git, curl
	result1 := &pkg.ImageValidationResult{Warnings: []string{}}
	validator.addPackageWarnings(result1, []string{"git", "curl"}, []string{})

	assert.Len(t, result1.Warnings, 2, "Should add 2 warnings (missing tools + package check)")
	assert.Contains(t, result1.Warnings[0], "git, curl")
	assert.Contains(t, result1.Warnings[0], "Missing recommended tools")
	assert.Contains(t, result1.Warnings[1], "2 recommended tools missing")

	// Second warning for different packages should show
	result2 := &pkg.ImageValidationResult{Warnings: []string{}}
	validator.addPackageWarnings(result2, []string{"make", "wget"}, []string{})

	assert.Len(t, result2.Warnings, 2, "Should add 2 warnings (missing tools + package check)")
	assert.Contains(t, result2.Warnings[0], "make, wget")
	assert.Contains(t, result2.Warnings[0], "Missing recommended tools")
	assert.Contains(t, result2.Warnings[1], "2 recommended tools missing")

	// Third warning for same as first should not show duplicate warning
	result3 := &pkg.ImageValidationResult{Warnings: []string{}}
	validator.addPackageWarnings(result3, []string{"git", "curl"}, []string{"vim"})

	assert.Len(t, result3.Warnings, 1, "Should only add info message for repeated warning")
	assert.Contains(t, result3.Warnings[0], "3 recommended tools missing")
	assert.NotContains(t, result3.Warnings[0], "Missing recommended tools")
}

func TestPackageCategorizationLogic(t *testing.T) {
	mockLogger := &MockLogger{}
	validator := &ImageValidator{
		logger:          mockLogger,
		cacheDir:        "/tmp/test-cache",
		sessionWarnings: make(map[string]bool),
	}

	// Test that we can properly categorize packages by priority
	highPriority := []string{}
	mediumPriority := []string{}
	lowPriority := []string{}

	for _, pkg := range recommendedPackages {
		switch pkg.Priority {
		case "high":
			highPriority = append(highPriority, pkg.Name)
		case "medium":
			mediumPriority = append(mediumPriority, pkg.Name)
		case "low":
			lowPriority = append(lowPriority, pkg.Name)
		}
	}

	// Test with only high priority missing (should warn)
	result1 := &pkg.ImageValidationResult{Warnings: []string{}}
	validator.addPackageWarnings(result1, highPriority, []string{})

	if len(highPriority) > 0 {
		assert.Greater(t, len(result1.Warnings), 0, "Should warn about high priority packages")
		assert.Contains(t, result1.Warnings[0], "Missing recommended tools")
	}

	// Test with only low priority missing (should not warn about missing tools)
	result2 := &pkg.ImageValidationResult{Warnings: []string{}}
	validator.addPackageWarnings(result2, []string{}, lowPriority)

	if len(lowPriority) > 0 {
		assert.Len(t, result2.Warnings, 1, "Should only show info message")
		assert.Contains(t, result2.Warnings[0], "recommended tools missing")
		assert.NotContains(t, result2.Warnings[0], "Missing recommended tools")
	}
}

func BenchmarkAddPackageWarnings(b *testing.B) {
	mockLogger := &MockLogger{}
	validator := &ImageValidator{
		logger:          mockLogger,
		cacheDir:        "/tmp/test-cache",
		sessionWarnings: make(map[string]bool),
	}

	missingHighPriority := []string{"git", "curl", "make"}
	missingOther := []string{"vim", "nano", "yq"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := &pkg.ImageValidationResult{Warnings: []string{}}
		validator.addPackageWarnings(result, missingHighPriority, missingOther)
	}
}

func BenchmarkSessionWarningLookup(b *testing.B) {
	mockLogger := &MockLogger{}
	validator := &ImageValidator{
		logger:          mockLogger,
		cacheDir:        "/tmp/test-cache",
		sessionWarnings: make(map[string]bool),
	}

	// Pre-populate with many warnings
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("warning-%d", i)
		validator.sessionWarnings[key] = true
	}

	testKey := "missing-packages-git,curl"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.sessionWarnings[testKey]
	}
}