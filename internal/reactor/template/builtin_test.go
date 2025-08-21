package template

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	
	"claude-reactor/pkg"
	"claude-reactor/pkg/mocks"
)

func TestManager_initializeBuiltinTemplates(t *testing.T) {
	// Create temporary directory for this test
	tempDir, err := os.MkdirTemp("", "builtin-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	// Setup mocks
	mockLogger := &mocks.MockLogger{}
	mockConfigMgr := &mocks.MockConfigManager{}
	mockDevContMgr := &mocks.MockDevContainerManager{}
	
	// Expect logging calls
	mockLogger.On("Infof", mock.AnythingOfType("string"), mock.Anything).Maybe()
	mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Maybe()
	
	mgr := &manager{
		logger:       mockLogger,
		configMgr:    mockConfigMgr,
		devContMgr:   mockDevContMgr,
		templatesDir: tempDir,
	}
	
	// Execute
	err = mgr.initializeBuiltinTemplates()
	
	// Verify
	assert.NoError(t, err)
	
	// Verify templates were created
	expectedTemplates := []string{
		"go-api", "go-cli", "rust-cli", "rust-lib", 
		"node-api", "react-app", "python-api", "python-cli", "java-spring",
	}
	
	for _, templateName := range expectedTemplates {
		templateDir := filepath.Join(tempDir, templateName)
		assert.DirExists(t, templateDir, "Template directory for %s should exist", templateName)
		assert.FileExists(t, filepath.Join(templateDir, "template.yaml"), "Template YAML for %s should exist", templateName)
	}
	
	mockLogger.AssertExpectations(t)
}

func TestManager_createGoAPITemplate(t *testing.T) {
	mockLogger := &mocks.MockLogger{}
	mockConfigMgr := &mocks.MockConfigManager{}
	mockDevContMgr := &mocks.MockDevContainerManager{}
	
	mgr := &manager{
		logger:     mockLogger,
		configMgr:  mockConfigMgr,
		devContMgr: mockDevContMgr,
	}
	
	template := mgr.createGoAPITemplate()
	
	assert.NotNil(t, template)
	assert.Equal(t, "go-api", template.Name)
	assert.Equal(t, "Go REST API with Gorilla Mux and structured logging", template.Description)
	assert.Equal(t, "go", template.Language)
	assert.Equal(t, "gorilla/mux", template.Framework)
	assert.Equal(t, "go", template.Variant)
	assert.Equal(t, "1.0.0", template.Version)
	assert.Equal(t, "claude-reactor", template.Author)
	assert.Contains(t, template.Tags, "go")
	assert.Contains(t, template.Tags, "api")
	assert.Contains(t, template.Tags, "rest")
	assert.Contains(t, template.Tags, "builtin")
	assert.NotEmpty(t, template.Files)
	
	// Verify essential files are present
	fileNames := make(map[string]bool)
	for _, file := range template.Files {
		fileNames[file.Path] = true
	}
	
	assert.True(t, fileNames["go.mod"], "go.mod should be present")
	assert.True(t, fileNames["main.go"], "main.go should be present")
	assert.True(t, fileNames["README.md"], "README.md should be present")
}

func TestManager_createGoCLITemplate(t *testing.T) {
	mockLogger := &mocks.MockLogger{}
	mockConfigMgr := &mocks.MockConfigManager{}
	mockDevContMgr := &mocks.MockDevContainerManager{}
	
	mgr := &manager{
		logger:     mockLogger,
		configMgr:  mockConfigMgr,
		devContMgr: mockDevContMgr,
	}
	
	template := mgr.createGoCLITemplate()
	
	assert.NotNil(t, template)
	assert.Equal(t, "go-cli", template.Name)
	assert.Equal(t, "Go CLI application with Cobra framework", template.Description)
	assert.Equal(t, "go", template.Language)
	assert.Equal(t, "cobra", template.Framework)
	assert.Equal(t, "go", template.Variant)
	assert.Contains(t, template.Tags, "go")
	assert.Contains(t, template.Tags, "cli")
	assert.Contains(t, template.Tags, "cobra")
	assert.NotEmpty(t, template.Files)
	
	// Verify CLI-specific files
	fileNames := make(map[string]bool)
	for _, file := range template.Files {
		fileNames[file.Path] = true
	}
	
	assert.True(t, fileNames["go.mod"], "go.mod should be present")
	assert.True(t, fileNames["main.go"], "main.go should be present")
	assert.True(t, fileNames["cmd/root.go"], "cmd/root.go should be present")
}

func TestManager_createRustCLITemplate(t *testing.T) {
	mockLogger := &mocks.MockLogger{}
	mockConfigMgr := &mocks.MockConfigManager{}
	mockDevContMgr := &mocks.MockDevContainerManager{}
	
	mgr := &manager{
		logger:     mockLogger,
		configMgr:  mockConfigMgr,
		devContMgr: mockDevContMgr,
	}
	
	template := mgr.createRustCLITemplate()
	
	assert.NotNil(t, template)
	assert.Equal(t, "rust-cli", template.Name)
	assert.Equal(t, "Rust CLI application with clap argument parser", template.Description)
	assert.Equal(t, "rust", template.Language)
	assert.Equal(t, "clap", template.Framework)
	assert.Equal(t, "full", template.Variant) // Rust requires full variant
	assert.Contains(t, template.Tags, "rust")
	assert.Contains(t, template.Tags, "cli")
	assert.NotEmpty(t, template.Files)
	
	// Verify Rust-specific files
	fileNames := make(map[string]bool)
	for _, file := range template.Files {
		fileNames[file.Path] = true
	}
	
	assert.True(t, fileNames["Cargo.toml"], "Cargo.toml should be present")
	assert.True(t, fileNames["src/main.rs"], "src/main.rs should be present")
}

func TestManager_createNodeAPITemplate(t *testing.T) {
	mockLogger := &mocks.MockLogger{}
	mockConfigMgr := &mocks.MockConfigManager{}
	mockDevContMgr := &mocks.MockDevContainerManager{}
	
	mgr := &manager{
		logger:     mockLogger,
		configMgr:  mockConfigMgr,
		devContMgr: mockDevContMgr,
	}
	
	template := mgr.createNodeAPITemplate()
	
	assert.NotNil(t, template)
	assert.Equal(t, "node-api", template.Name)
	assert.Equal(t, "Node.js REST API with Express and TypeScript", template.Description)
	assert.Equal(t, "node", template.Language)
	assert.Equal(t, "express", template.Framework)
	assert.Equal(t, "base", template.Variant)
	assert.Contains(t, template.Tags, "node")
	assert.Contains(t, template.Tags, "api")
	assert.Contains(t, template.Tags, "typescript")
	assert.NotEmpty(t, template.Files)
	
	// Verify Node.js-specific files
	fileNames := make(map[string]bool)
	for _, file := range template.Files {
		fileNames[file.Path] = true
	}
	
	assert.True(t, fileNames["package.json"], "package.json should be present")
	assert.True(t, fileNames["tsconfig.json"], "tsconfig.json should be present")
	assert.True(t, fileNames["src/index.ts"], "src/index.ts should be present")
}

func TestManager_createPythonAPITemplate(t *testing.T) {
	mockLogger := &mocks.MockLogger{}
	mockConfigMgr := &mocks.MockConfigManager{}
	mockDevContMgr := &mocks.MockDevContainerManager{}
	
	mgr := &manager{
		logger:     mockLogger,
		configMgr:  mockConfigMgr,
		devContMgr: mockDevContMgr,
	}
	
	template := mgr.createPythonAPITemplate()
	
	assert.NotNil(t, template)
	assert.Equal(t, "python-api", template.Name)
	assert.Equal(t, "Python REST API with FastAPI and modern tooling", template.Description)
	assert.Equal(t, "python", template.Language)
	assert.Equal(t, "fastapi", template.Framework)
	assert.Equal(t, "base", template.Variant)
	assert.Contains(t, template.Tags, "python")
	assert.Contains(t, template.Tags, "api")
	assert.Contains(t, template.Tags, "fastapi")
	assert.NotEmpty(t, template.Files)
	
	// Verify Python-specific files
	fileNames := make(map[string]bool)
	for _, file := range template.Files {
		fileNames[file.Path] = true
	}
	
	assert.True(t, fileNames["requirements.txt"], "requirements.txt should be present")
	assert.True(t, fileNames["main.py"], "main.py should be present")
	assert.True(t, fileNames["README.md"], "README.md should be present")
}

func TestManager_createJavaSpringTemplate(t *testing.T) {
	mockLogger := &mocks.MockLogger{}
	mockConfigMgr := &mocks.MockConfigManager{}
	mockDevContMgr := &mocks.MockDevContainerManager{}
	
	mgr := &manager{
		logger:     mockLogger,
		configMgr:  mockConfigMgr,
		devContMgr: mockDevContMgr,
	}
	
	template := mgr.createJavaSpringTemplate()
	
	assert.NotNil(t, template)
	assert.Equal(t, "java-spring", template.Name)
	assert.Equal(t, "Java Spring Boot REST API", template.Description)
	assert.Equal(t, "java", template.Language)
	assert.Equal(t, "spring-boot", template.Framework)
	assert.Equal(t, "full", template.Variant) // Java requires full variant
	assert.Contains(t, template.Tags, "java")
	assert.Contains(t, template.Tags, "spring")
	assert.Contains(t, template.Tags, "api")
	assert.NotEmpty(t, template.Files)
	
	// Verify Java-specific files
	fileNames := make(map[string]bool)
	for _, file := range template.Files {
		fileNames[file.Path] = true
	}
	
	assert.True(t, fileNames["pom.xml"], "pom.xml should be present")
}

func TestManager_saveBuiltinTemplate(t *testing.T) {
	// Create temporary directory for this test
	tempDir, err := os.MkdirTemp("", "builtin-save-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	mockLogger := &mocks.MockLogger{}
	mockConfigMgr := &mocks.MockConfigManager{}
	mockDevContMgr := &mocks.MockDevContainerManager{}
	
	mgr := &manager{
		logger:       mockLogger,
		configMgr:    mockConfigMgr,
		devContMgr:   mockDevContMgr,
		templatesDir: tempDir,
	}
	
	// Create a test template
	template := &pkg.ProjectTemplate{
		Name:        "test-template",
		Description: "Test template for saving",
		Language:    "go",
		Variant:     "go",
		Version:     "1.0.0",
		Author:      "test",
		Files: []pkg.TemplateFile{
			{
				Path: "test.txt",
				Content: "This is a test file",
			},
		},
	}
	
	// Execute
	err = mgr.saveBuiltinTemplate(template)
	
	// Verify
	assert.NoError(t, err)
	
	// Check template directory was created
	templateDir := filepath.Join(tempDir, template.Name)
	assert.DirExists(t, templateDir)
	
	// Check template.yaml was created
	templateFile := filepath.Join(templateDir, "template.yaml")
	assert.FileExists(t, templateFile)
	
	// Verify template YAML contains the file definition  
	yamlContent, err := os.ReadFile(templateFile)
	assert.NoError(t, err)
	assert.Contains(t, string(yamlContent), "test.txt")
	assert.Contains(t, string(yamlContent), "This is a test file")
}

func TestBuiltinTemplatesCompleteness(t *testing.T) {
	mockLogger := &mocks.MockLogger{}
	mockConfigMgr := &mocks.MockConfigManager{}
	mockDevContMgr := &mocks.MockDevContainerManager{}
	
	mgr := &manager{
		logger:     mockLogger,
		configMgr:  mockConfigMgr,
		devContMgr: mockDevContMgr,
	}
	
	// Test all template creation functions
	templates := []*pkg.ProjectTemplate{
		mgr.createGoAPITemplate(),
		mgr.createGoCLITemplate(),
		mgr.createRustCLITemplate(),
		mgr.createRustLibTemplate(),
		mgr.createNodeAPITemplate(),
		mgr.createReactAppTemplate(),
		mgr.createPythonAPITemplate(),
		mgr.createPythonCLITemplate(),
		mgr.createJavaSpringTemplate(),
	}
	
	// Verify each template has required fields
	for _, template := range templates {
		assert.NotEmpty(t, template.Name, "Template should have a name")
		assert.NotEmpty(t, template.Description, "Template should have a description")
		assert.NotEmpty(t, template.Language, "Template should have a language")
		assert.NotEmpty(t, template.Variant, "Template should have a variant")
		assert.NotEmpty(t, template.Version, "Template should have a version")
		assert.NotEmpty(t, template.Author, "Template should have an author")
		assert.NotEmpty(t, template.Tags, "Template should have tags")
		assert.NotEmpty(t, template.Files, "Template should have files")
		
		// Verify at least one file exists
		assert.GreaterOrEqual(t, len(template.Files), 1, "Template should have at least one file")
		
		// Verify each file has content
		for _, file := range template.Files {
			assert.NotEmpty(t, file.Path, "File should have a name")
			assert.NotEmpty(t, file.Content, "File should have content")
		}
	}
	
	// Verify we have templates for major languages
	languages := make(map[string]bool)
	for _, template := range templates {
		languages[template.Language] = true
	}
	
	expectedLanguages := []string{"go", "rust", "node", "python", "java"}
	for _, lang := range expectedLanguages {
		assert.True(t, languages[lang], "Should have at least one template for language: %s", lang)
	}
}