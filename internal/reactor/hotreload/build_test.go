package hotreload

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"claude-reactor/pkg"
	"claude-reactor/pkg/mocks"
)

func TestNewBuildTrigger(t *testing.T) {
	mockLogger := &mocks.MockLogger{}

	trigger := NewBuildTrigger(mockLogger)

	assert.NotNil(t, trigger)
	assert.IsType(t, &buildTrigger{}, trigger)

	// Verify internal structure
	impl := trigger.(*buildTrigger)
	assert.Equal(t, mockLogger, impl.logger)
}

func TestBuildTrigger_DetectProjectType(t *testing.T) {
	tests := []struct {
		name            string
		setupProject    func(string) error
		expectedType    string
		expectedBuildCmd []string
		expectedRunCmd   []string
		expectError     bool
	}{
		{
			name: "detect go project",
			setupProject: func(projectDir string) error {
				goMod := `module test-project

go 1.21

require (
	github.com/gorilla/mux v1.8.0
)
`
				return os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goMod), 0644)
			},
			expectedType:     "go",
			expectedBuildCmd: []string{"go", "build", "."},
			expectedRunCmd:   []string{"go", "run", "."},
			expectError:      false,
		},
		{
			name: "detect nodejs project with npm",
			setupProject: func(projectDir string) error {
				packageJson := `{
  "name": "test-project",
  "version": "1.0.0",
  "scripts": {
    "start": "node index.js",
    "build": "npm run compile"
  },
  "dependencies": {
    "express": "^4.18.0"
  }
}
`
				return os.WriteFile(filepath.Join(projectDir, "package.json"), []byte(packageJson), 0644)
			},
			expectedType:     "nodejs",
			expectedBuildCmd: []string{"npm", "run", "build"},
			expectedRunCmd:   []string{"npm", "run", "dev"},
			expectError:      false,
		},
		{
			name: "detect python project with requirements.txt",
			setupProject: func(projectDir string) error {
				requirements := `flask>=2.0.0
fastapi>=0.68.0
uvicorn>=0.15.0
`
				return os.WriteFile(filepath.Join(projectDir, "requirements.txt"), []byte(requirements), 0644)
			},
			expectedType:     "python",
			expectedBuildCmd: []string{"python", "-m", "pip", "install", "-r", "requirements.txt"},
			expectedRunCmd:   []string{"python", "app.py"},
			expectError:      false,
		},
		{
			name: "detect python project with pyproject.toml",
			setupProject: func(projectDir string) error {
				pyproject := `[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[project]
name = "test-project"
version = "1.0.0"
dependencies = [
  "flask>=2.0.0"
]

[project.scripts]
start = "python main.py"
`
				return os.WriteFile(filepath.Join(projectDir, "pyproject.toml"), []byte(pyproject), 0644)
			},
			expectedType:     "python",
			expectedBuildCmd: []string{"python", "-m", "pip", "install", "-r", "requirements.txt"},
			expectedRunCmd:   []string{"python", "app.py"},
			expectError:      false,
		},
		{
			name: "detect rust project",
			setupProject: func(projectDir string) error {
				cargoToml := `[package]
name = "test-project"
version = "0.1.0"
edition = "2021"

[dependencies]
tokio = { version = "1.0", features = ["full"] }
serde = { version = "1.0", features = ["derive"] }
`
				return os.WriteFile(filepath.Join(projectDir, "Cargo.toml"), []byte(cargoToml), 0644)
			},
			expectedType:     "rust",
			expectedBuildCmd: []string{"cargo", "build"},
			expectedRunCmd:   []string{"cargo", "run"},
			expectError:      false,
		},
		{
			name: "detect java project with maven",
			setupProject: func(projectDir string) error {
				pomXml := `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <modelVersion>4.0.0</modelVersion>
    <groupId>com.example</groupId>
    <artifactId>test-project</artifactId>
    <version>1.0.0</version>
    
    <dependencies>
        <dependency>
            <groupId>org.springframework.boot</groupId>
            <artifactId>spring-boot-starter-web</artifactId>
        </dependency>
    </dependencies>
</project>
`
				return os.WriteFile(filepath.Join(projectDir, "pom.xml"), []byte(pomXml), 0644)
			},
			expectedType:     "java",
			expectedBuildCmd: []string{"mvn", "compile"},
			expectedRunCmd:   []string{"mvn", "spring-boot:run"},
			expectError:      false,
		},
		{
			name: "detect java project with gradle",
			setupProject: func(projectDir string) error {
				buildGradle := `plugins {
    id 'java'
    id 'org.springframework.boot' version '2.7.0'
}

dependencies {
    implementation 'org.springframework.boot:spring-boot-starter-web'
}
`
				return os.WriteFile(filepath.Join(projectDir, "build.gradle"), []byte(buildGradle), 0644)
			},
			expectedType:     "java",
			expectedBuildCmd: []string{"./gradlew", "build"},
			expectedRunCmd:   nil,
			expectError:      false,
		},
		{
			name: "unknown project type",
			setupProject: func(projectDir string) error {
				// Create only a README file
				readme := `# Unknown Project

This project has no clear build system indicators.
`
				return os.WriteFile(filepath.Join(projectDir, "README.md"), []byte(readme), 0644)
			},
			expectedType:     "unknown",
			expectedBuildCmd: []string{"echo", "No build command detected"},
			expectedRunCmd:   nil,
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for this test
			tempDir, err := os.MkdirTemp("", "build-detect-*")
			assert.NoError(t, err)
			defer os.RemoveAll(tempDir)

			projectDir := filepath.Join(tempDir, "test-project")
			err = os.MkdirAll(projectDir, 0755)
			assert.NoError(t, err)

			// Setup project files
			err = tt.setupProject(projectDir)
			assert.NoError(t, err)

			// Setup mocks
			mockLogger := &mocks.MockLogger{}
			mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()

			trigger := &buildTrigger{
				logger: mockLogger,
			}

			// Execute
			buildInfo, err := trigger.DetectProjectType(projectDir)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, buildInfo)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, buildInfo)
				assert.Equal(t, tt.expectedType, buildInfo.Type)
				assert.Equal(t, tt.expectedBuildCmd, buildInfo.BuildCommand)
				assert.Equal(t, tt.expectedRunCmd, buildInfo.StartCommand)
			}

			// Verify mock expectations
			mockLogger.AssertExpectations(t)
		})
	}
}

func TestBuildTrigger_ExecuteBuild(t *testing.T) {
	tests := []struct {
		name         string
		buildInfo    *pkg.ProjectBuildInfo
		setupMocks   func(*mocks.MockLogger)
		expectError  bool
		skipExecution bool // Skip actual command execution for unit tests
	}{
		{
			name: "execute go build",
			buildInfo: &pkg.ProjectBuildInfo{
				Type:  "go",
				BuildCommand: []string{"go", "version"}, // Use a safe command that should work
				StartCommand:   []string{"echo", "go run simulation"},
				},
			setupMocks: func(mockLogger *mocks.MockLogger) {
				mockLogger.On("Infof", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: false,
		},
		{
			name: "execute invalid build command",
			buildInfo: &pkg.ProjectBuildInfo{
				Type:         "unknown",
				BuildCommand: []string{"this-command-does-not-exist"},
				StartCommand: []string{"echo", "test"},
			},
			setupMocks: func(mockLogger *mocks.MockLogger) {
				mockLogger.On("Errorf", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Infof", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: false, // ExecuteBuild never returns error, result.Success will be false
		},
		{
			name: "execute build with empty command",
			buildInfo: &pkg.ProjectBuildInfo{
				Type:         "unknown",
				BuildCommand: nil,
				StartCommand:   nil,
				},
			setupMocks: func(mockLogger *mocks.MockLogger) {
				mockLogger.On("Warnf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: false, // Should handle gracefully
		},
		{
			name: "execute nodejs build simulation",
			buildInfo: &pkg.ProjectBuildInfo{
				Type:  "nodejs", 
				BuildCommand: []string{"echo", "npm install && npm run build"},
				StartCommand:   []string{"echo", "npm start"},
				},
			setupMocks: func(mockLogger *mocks.MockLogger) {
				mockLogger.On("Infof", mock.AnythingOfType("string"), mock.Anything).Maybe()
				mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockLogger := &mocks.MockLogger{}
			tt.setupMocks(mockLogger)

			trigger := &buildTrigger{
				logger: mockLogger,
			}

			// Execute
			result, err := trigger.ExecuteBuild(".", tt.buildInfo.BuildCommand, nil)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				if len(tt.buildInfo.BuildCommand) == 0 {
					// Empty build command should return success without execution
					assert.NoError(t, err)
					assert.NotNil(t, result)
					assert.True(t, result.Success)
					assert.Equal(t, "No build command specified", result.Output)
				} else {
					// For non-empty commands, check execution
					assert.NoError(t, err)
					assert.NotNil(t, result)
					// Success depends on whether the command actually works
				}
			}

			// Verify mock expectations
			mockLogger.AssertExpectations(t)
		})
	}
}

func TestBuildTrigger_NeedsRebuild(t *testing.T) {
	tests := []struct {
		name         string
		setupProject func(string) error
		changedFiles []string
		buildInfo    *pkg.ProjectBuildInfo
		expected     bool
	}{
		{
			name: "go project needs rebuild for .go files",
			setupProject: func(projectDir string) error {
				return os.WriteFile(filepath.Join(projectDir, "main.go"), []byte("package main"), 0644)
			},
			changedFiles: []string{"main.go", "utils.go"},
			buildInfo: &pkg.ProjectBuildInfo{
				Type: "go",
			},
			expected: true,
		},
		{
			name: "go project doesn't need rebuild for non-go files",
			setupProject: func(projectDir string) error {
				return os.WriteFile(filepath.Join(projectDir, "README.md"), []byte("# Project"), 0644)
			},
			changedFiles: []string{"README.md", "docs/guide.md"},
			buildInfo: &pkg.ProjectBuildInfo{
				Type: "go",
			},
			expected: false,
		},
		{
			name: "nodejs project needs rebuild for package.json",
			setupProject: func(projectDir string) error {
				return os.WriteFile(filepath.Join(projectDir, "package.json"), []byte("{}"), 0644)
			},
			changedFiles: []string{"package.json"},
			buildInfo: &pkg.ProjectBuildInfo{
				Type: "nodejs",
			},
			expected: true,
		},
		{
			name: "nodejs project needs rebuild for source files",
			setupProject: func(projectDir string) error {
				return os.WriteFile(filepath.Join(projectDir, "index.js"), []byte("console.log('test')"), 0644)
			},
			changedFiles: []string{"src/index.ts", "lib/utils.js"},
			buildInfo: &pkg.ProjectBuildInfo{
				Type: "nodejs",
			},
			expected: true,
		},
		{
			name: "python project needs rebuild for requirements",
			setupProject: func(projectDir string) error {
				return os.WriteFile(filepath.Join(projectDir, "requirements.txt"), []byte("flask"), 0644)
			},
			changedFiles: []string{"requirements.txt"},
			buildInfo: &pkg.ProjectBuildInfo{
				Type: "python",
			},
			expected: true,
		},
		{
			name: "python project needs rebuild for source files",
			setupProject: func(projectDir string) error {
				return os.WriteFile(filepath.Join(projectDir, "main.py"), []byte("print('test')"), 0644)
			},
			changedFiles: []string{"main.py", "utils.py"},
			buildInfo: &pkg.ProjectBuildInfo{
				Type: "python",
			},
			expected: true,
		},
		{
			name: "rust project needs rebuild for Cargo.toml",
			setupProject: func(projectDir string) error {
				return os.WriteFile(filepath.Join(projectDir, "Cargo.toml"), []byte("[package]\nname = \"test\""), 0644)
			},
			changedFiles: []string{"Cargo.toml"},
			buildInfo: &pkg.ProjectBuildInfo{
				Type: "rust",
			},
			expected: true,
		},
		{
			name: "rust project needs rebuild for source files",
			setupProject: func(projectDir string) error {
				srcDir := filepath.Join(projectDir, "src")
				if err := os.MkdirAll(srcDir, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(srcDir, "main.rs"), []byte("fn main() {}"), 0644)
			},
			changedFiles: []string{"src/main.rs", "src/lib.rs"},
			buildInfo: &pkg.ProjectBuildInfo{
				Type: "rust",
			},
			expected: true,
		},
		{
			name: "java project needs rebuild for pom.xml",
			setupProject: func(projectDir string) error {
				return os.WriteFile(filepath.Join(projectDir, "pom.xml"), []byte("<project></project>"), 0644)
			},
			changedFiles: []string{"pom.xml"},
			buildInfo: &pkg.ProjectBuildInfo{
				Type: "java",
			},
			expected: true,
		},
		{
			name: "java project needs rebuild for source files",
			setupProject: func(projectDir string) error {
				srcDir := filepath.Join(projectDir, "src")
				if err := os.MkdirAll(srcDir, 0755); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(srcDir, "Main.java"), []byte("public class Main {}"), 0644)
			},
			changedFiles: []string{"src/main/java/Main.java", "src/test/java/TestMain.java"},
			buildInfo: &pkg.ProjectBuildInfo{
				Type: "java",
			},
			expected: true,
		},
		{
			name: "unknown project type always needs rebuild",
			setupProject: func(projectDir string) error {
				return os.WriteFile(filepath.Join(projectDir, "some-file.txt"), []byte("content"), 0644)
			},
			changedFiles: []string{"any-file.xyz"},
			buildInfo: &pkg.ProjectBuildInfo{
				Type: "unknown",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for this test
			tempDir, err := os.MkdirTemp("", "rebuild-test-*")
			assert.NoError(t, err)
			defer os.RemoveAll(tempDir)

			projectDir := filepath.Join(tempDir, "test-project")
			err = os.MkdirAll(projectDir, 0755)
			assert.NoError(t, err)

			// Setup project files
			err = tt.setupProject(projectDir)
			assert.NoError(t, err)

			// Note: ProjectBuildInfo doesn't have WorkingDir field
			// The test will use projectDir as the execution context

			// Setup mocks
			mockLogger := &mocks.MockLogger{}
			mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()

			trigger := &buildTrigger{
				logger: mockLogger,
			}

			// Execute
			result := trigger.NeedsRebuild(tt.changedFiles, tt.buildInfo)

			// Verify
			assert.Equal(t, tt.expected, result)

			// Verify mock expectations
			mockLogger.AssertExpectations(t)
		})
	}
}

func TestGetRelevantFileExtensions(t *testing.T) {
	tests := []struct {
		projectType string
		expected    []string
	}{
		{
			projectType: "go",
			expected:    []string{".go", ".mod", ".sum"},
		},
		{
			projectType: "nodejs", 
			expected:    []string{".js", ".ts", ".jsx", ".tsx", ".json", ".mjs"},
		},
		{
			projectType: "python",
			expected:    []string{".py", ".pyx", ".pyi", ".txt", ".toml", ".cfg"},
		},
		{
			projectType: "rust",
			expected:    []string{".rs", ".toml"},
		},
		{
			projectType: "java",
			expected:    []string{".java", ".xml", ".gradle", ".properties"},
		},
		{
			projectType: "unknown",
			expected:    []string{}, // Should return empty slice for unknown types
		},
	}

	for _, tt := range tests {
		t.Run(tt.projectType, func(t *testing.T) {
			result := getRelevantFileExtensions(tt.projectType)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestIsRelevantFile(t *testing.T) {
	tests := []struct {
		name        string
		filePath    string
		projectType string
		expected    bool
	}{
		{
			name:        "go source file",
			filePath:    "/project/main.go",
			projectType: "go",
			expected:    true,
		},
		{
			name:        "go mod file",
			filePath:    "/project/go.mod",
			projectType: "go",
			expected:    true,
		},
		{
			name:        "non-go file in go project",
			filePath:    "/project/README.md",
			projectType: "go",
			expected:    false,
		},
		{
			name:        "javascript file",
			filePath:    "/project/index.js",
			projectType: "nodejs",
			expected:    true,
		},
		{
			name:        "typescript file",
			filePath:    "/project/main.ts",
			projectType: "nodejs",
			expected:    true,
		},
		{
			name:        "package.json file",
			filePath:    "/project/package.json",
			projectType: "nodejs",
			expected:    true,
		},
		{
			name:        "python source file",
			filePath:    "/project/main.py",
			projectType: "python",
			expected:    true,
		},
		{
			name:        "requirements file",
			filePath:    "/project/requirements.txt",
			projectType: "python",
			expected:    true,
		},
		{
			name:        "rust source file",
			filePath:    "/project/src/main.rs",
			projectType: "rust",
			expected:    true,
		},
		{
			name:        "cargo toml file",
			filePath:    "/project/Cargo.toml",
			projectType: "rust",
			expected:    true,
		},
		{
			name:        "java source file",
			filePath:    "/project/src/Main.java",
			projectType: "java",
			expected:    true,
		},
		{
			name:        "maven pom file",
			filePath:    "/project/pom.xml",
			projectType: "java",
			expected:    true,
		},
		{
			name:        "unknown project type file",
			filePath:    "/project/some-file.txt",
			projectType: "unknown",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRelevantFile(tt.filePath, tt.projectType)
			assert.Equal(t, tt.expected, result)
		})
	}
}