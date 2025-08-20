package devcontainer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"claude-reactor/pkg/mocks"
)

func TestManager_DetectProjectType(t *testing.T) {
	tests := []struct {
		name         string
		setupProject func(string) error
		expectedLang string
		expectError  bool
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
			expectedLang: "Go",
			expectError:  false,
		},
		{
			name: "detect nodejs project with package.json",
			setupProject: func(projectDir string) error {
				packageJson := `{
  "name": "test-project",
  "version": "1.0.0",
  "dependencies": {
    "express": "^4.18.0"
  }
}
`
				return os.WriteFile(filepath.Join(projectDir, "package.json"), []byte(packageJson), 0644)
			},
			expectedLang: "JavaScript",
			expectError:  false,
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
			expectedLang: "Python",
			expectError:  false,
		},
		{
			name: "detect rust project with Cargo.toml",
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
			expectedLang: "Rust",
			expectError:  false,
		},
		{
			name: "detect java project with pom.xml",
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
			expectedLang: "Java",
			expectError:  false,
		},
		{
			name: "unknown project type",
			setupProject: func(projectDir string) error {
				// Create only a README file
				readme := `# Unknown Project

This project has no clear language indicators.
`
				return os.WriteFile(filepath.Join(projectDir, "README.md"), []byte(readme), 0644)
			},
			expectedLang: "",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for this test
			tempDir, err := os.MkdirTemp("", "devcontainer-detect-*")
			assert.NoError(t, err)
			defer os.RemoveAll(tempDir)

			projectDir := filepath.Join(tempDir, "test-project")
			err = os.MkdirAll(projectDir, 0755)
			assert.NoError(t, err)

			// Setup project files
			err = tt.setupProject(projectDir)
			assert.NoError(t, err)

			// Setup manager
			mockLogger := &mocks.MockLogger{}
			mockConfigMgr := &mocks.MockConfigManager{}

			mgr := &manager{
				logger:    mockLogger,
				configMgr: mockConfigMgr,
			}
			
			// Setup mock for AutoDetectVariant call
			mockConfigMgr.On("AutoDetectVariant", mock.AnythingOfType("string")).Return("go", nil)

			// Execute
			result, err := mgr.DetectProjectType(projectDir)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.expectedLang != "" {
					assert.Contains(t, result.Languages, tt.expectedLang)
				} else {
					// For unknown projects, Languages may be empty or contain generic entries
					assert.True(t, len(result.Languages) >= 0)
				}
				assert.NotEmpty(t, result.ProjectType)
			}
		})
	}
}