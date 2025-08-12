package devcontainer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	
	"claude-reactor/pkg"
)

// manager implements the DevContainerManager interface
type manager struct {
	logger pkg.Logger
	configMgr pkg.ConfigManager
}

// NewManager creates a new DevContainer manager
func NewManager(logger pkg.Logger, configMgr pkg.ConfigManager) pkg.DevContainerManager {
	return &manager{
		logger: logger,
		configMgr: configMgr,
	}
}

// GenerateDevContainer creates .devcontainer configuration based on project detection
func (m *manager) GenerateDevContainer(projectPath string, config *pkg.Config) error {
	if projectPath == "" {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}
	
	m.logger.Infof("Generating devcontainer configuration for project: %s", projectPath)
	
	// Detect project type with enhanced detection
	detection, err := m.DetectProjectType(projectPath)
	if err != nil {
		return fmt.Errorf("failed to detect project type: %w", err)
	}
	
	// Create .devcontainer directory
	devcontainerDir := filepath.Join(projectPath, ".devcontainer")
	if err := os.MkdirAll(devcontainerDir, 0755); err != nil {
		return fmt.Errorf("failed to create .devcontainer directory: %w", err)
	}
	
	// Generate devcontainer configuration
	devConfig, err := m.createDevContainerConfigForProject(detection, config)
	if err != nil {
		return fmt.Errorf("failed to create devcontainer config: %w", err)
	}
	
	// Write devcontainer.json
	configBytes, err := m.CreateDevContainerConfig(devConfig)
	if err != nil {
		return fmt.Errorf("failed to serialize devcontainer config: %w", err)
	}
	
	configPath := filepath.Join(devcontainerDir, "devcontainer.json")
	if err := os.WriteFile(configPath, configBytes, 0644); err != nil {
		return fmt.Errorf("failed to write devcontainer.json: %w", err)
	}
	
	m.logger.Infof("Successfully created devcontainer configuration")
	m.logger.Infof("  - Project Type: %s", detection.ProjectType)
	m.logger.Infof("  - Variant: %s", detection.Variant)
	m.logger.Infof("  - Extensions: %d", len(detection.Extensions))
	m.logger.Infof("  - Config: %s", configPath)
	
	return nil
}

// ValidateDevContainer validates existing .devcontainer configuration
func (m *manager) ValidateDevContainer(projectPath string) error {
	devcontainerPath := filepath.Join(projectPath, ".devcontainer", "devcontainer.json")
	
	if _, err := os.Stat(devcontainerPath); os.IsNotExist(err) {
		return fmt.Errorf("devcontainer.json not found at %s", devcontainerPath)
	}
	
	// Read and validate JSON structure
	data, err := os.ReadFile(devcontainerPath)
	if err != nil {
		return fmt.Errorf("failed to read devcontainer.json: %w", err)
	}
	
	var config pkg.DevContainerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("invalid devcontainer.json format: %w", err)
	}
	
	// Basic validation
	if config.Name == "" {
		return fmt.Errorf("devcontainer.json missing required 'name' field")
	}
	
	if config.Image == "" && config.DockerFile == "" && config.Build == nil {
		return fmt.Errorf("devcontainer.json must specify either 'image', 'dockerFile', or 'build'")
	}
	
	// Validate Dockerfile path if using build
	if config.Build != nil && config.Build.DockerFile != "" {
		devcontainerDir := filepath.Join(projectPath, ".devcontainer")
		dockerfilePath := filepath.Join(devcontainerDir, config.Build.DockerFile)
		
		if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
			m.logger.Warnf("⚠️  Dockerfile not found at %s", dockerfilePath)
			m.logger.Infof("💡 Expected Dockerfile location: %s", dockerfilePath)
			m.logger.Infof("💡 Actual project structure:")
			m.logger.Infof("   Current directory: %s", projectPath)
			if _, err := os.Stat(filepath.Join(projectPath, "Dockerfile")); err == nil {
				m.logger.Infof("   ✅ Dockerfile found at: %s/Dockerfile", projectPath)
				m.logger.Infof("   💡 Try: Open VS Code from the project root directory where Dockerfile exists")
			} else {
				m.logger.Infof("   ❌ No Dockerfile found in project root")
			}
		} else {
			m.logger.Infof("✅ Dockerfile found at: %s", dockerfilePath)
		}
	}
	
	m.logger.Infof("DevContainer configuration is valid: %s", config.Name)
	return nil
}

// GetExtensionsForProject returns recommended VS Code extensions for detected project type
func (m *manager) GetExtensionsForProject(projectType string, variant string) ([]string, error) {
	extensions := []string{}
	
	// Base extensions for all projects
	extensions = append(extensions, 
		"ms-vscode-remote.remote-containers",
		"ms-vscode.vscode-json",
		"redhat.vscode-yaml",
		"ms-vscode.remote-repositories",
	)
	
	// Language-specific extensions
	switch projectType {
	case "go":
		extensions = append(extensions,
			"golang.Go",
			"golang.go-nightly",
			"ms-vscode.vscode-go",
		)
	case "rust":
		extensions = append(extensions,
			"rust-lang.rust-analyzer",
			"tamasfe.even-better-toml",
			"serayuzgur.crates",
		)
	case "node", "javascript", "typescript":
		extensions = append(extensions,
			"ms-vscode.vscode-typescript-next",
			"ms-vscode.vscode-eslint",
			"esbenp.prettier-vscode",
			"ms-vscode.vscode-npm-script",
		)
	case "python":
		extensions = append(extensions,
			"ms-python.python",
			"ms-python.vscode-pylance",
			"ms-python.black-formatter",
			"ms-python.flake8",
		)
	case "java":
		extensions = append(extensions,
			"redhat.java",
			"vscjava.vscode-java-pack",
			"vscjava.vscode-gradle",
			"vscjava.vscode-maven",
		)
	}
	
	// Variant-specific extensions
	switch variant {
	case "cloud":
		extensions = append(extensions,
			"ms-vscode.vscode-docker",
			"ms-azuretools.vscode-docker",
			"amazonwebservices.aws-toolkit-vscode",
			"googlecloudtools.cloudcode",
		)
	case "k8s":
		extensions = append(extensions,
			"ms-kubernetes-tools.vscode-kubernetes-tools",
			"ms-vscode.vscode-docker",
			"tim-koehler.helm-intellisense",
		)
	}
	
	// Git and development workflow extensions
	extensions = append(extensions,
		"eamodio.gitlens",
		"github.vscode-pull-request-github",
		"ms-vscode.vscode-git-graph",
	)
	
	return extensions, nil
}

// CreateDevContainerConfig generates devcontainer.json content
func (m *manager) CreateDevContainerConfig(config *pkg.DevContainerConfig) ([]byte, error) {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal devcontainer config: %w", err)
	}
	
	return data, nil
}

// DetectProjectType performs enhanced project detection for VS Code integration
func (m *manager) DetectProjectType(projectPath string) (*pkg.ProjectDetectionResult, error) {
	if projectPath == "" {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
	}
	
	result := &pkg.ProjectDetectionResult{
		Languages:  []string{},
		Frameworks: []string{},
		Extensions: []string{},
		Features:   []string{},
		Tools:      []string{},
		Files:      []string{},
		Metadata:   make(map[string]string),
		Confidence: 0.0,
	}
	
	// Use existing auto-detection as base
	variant, err := m.configMgr.AutoDetectVariant(projectPath)
	if err != nil {
		m.logger.Warnf("Auto-detection warning: %v", err)
	}
	result.Variant = variant
	
	// Enhanced project type detection
	if err := m.detectGoProject(projectPath, result); err == nil {
		result.ProjectType = "go"
		result.Languages = append(result.Languages, "Go")
		result.Confidence = 0.9
	}
	
	if err := m.detectRustProject(projectPath, result); err == nil {
		result.ProjectType = "rust"
		result.Languages = append(result.Languages, "Rust")
		result.Confidence = 0.9
	}
	
	if err := m.detectNodeProject(projectPath, result); err == nil {
		result.ProjectType = "node"
		result.Languages = append(result.Languages, "JavaScript", "TypeScript")
		result.Confidence = 0.8
	}
	
	if err := m.detectPythonProject(projectPath, result); err == nil {
		result.ProjectType = "python"
		result.Languages = append(result.Languages, "Python")
		result.Confidence = 0.8
	}
	
	if err := m.detectJavaProject(projectPath, result); err == nil {
		result.ProjectType = "java"
		result.Languages = append(result.Languages, "Java")
		result.Confidence = 0.8
	}
	
	// Get extensions based on detected project type
	extensions, err := m.GetExtensionsForProject(result.ProjectType, result.Variant)
	if err != nil {
		m.logger.Warnf("Failed to get extensions: %v", err)
	} else {
		result.Extensions = extensions
	}
	
	// If no specific type detected, default to base
	if result.ProjectType == "" {
		result.ProjectType = "base"
		result.Confidence = 0.5
		result.Extensions, _ = m.GetExtensionsForProject("base", variant)
	}
	
	return result, nil
}

// UpdateDevContainer updates existing .devcontainer configuration
func (m *manager) UpdateDevContainer(projectPath string, config *pkg.Config) error {
	// Check if devcontainer exists
	if err := m.ValidateDevContainer(projectPath); err != nil {
		return fmt.Errorf("cannot update: %w", err)
	}
	
	// Regenerate configuration
	return m.GenerateDevContainer(projectPath, config)
}

// RemoveDevContainer removes .devcontainer directory and configurations
func (m *manager) RemoveDevContainer(projectPath string) error {
	devcontainerDir := filepath.Join(projectPath, ".devcontainer")
	
	if _, err := os.Stat(devcontainerDir); os.IsNotExist(err) {
		m.logger.Infof("No .devcontainer directory to remove")
		return nil
	}
	
	if err := os.RemoveAll(devcontainerDir); err != nil {
		return fmt.Errorf("failed to remove .devcontainer directory: %w", err)
	}
	
	m.logger.Infof("Successfully removed .devcontainer directory")
	return nil
}

// createDevContainerConfigForProject creates a devcontainer config based on project detection
func (m *manager) createDevContainerConfigForProject(detection *pkg.ProjectDetectionResult, config *pkg.Config) (*pkg.DevContainerConfig, error) {
	// Base devcontainer configuration
	devConfig := &pkg.DevContainerConfig{
		Name: fmt.Sprintf("Claude Reactor %s (%s)", strings.Title(detection.Variant), strings.Title(detection.ProjectType)),
		Build: &pkg.DevContainerBuild{
			DockerFile: "../Dockerfile",
			Context: "..",
			Target: detection.Variant,
		},
		WorkspaceFolder: "/workspaces/${localWorkspaceFolderBasename}",
		Customizations: &pkg.DevContainerCustom{
			VSCode: &pkg.VSCodeCustomization{
				Extensions: detection.Extensions,
				Settings: map[string]interface{}{
					"terminal.integrated.shell.linux": "/bin/bash",
					"files.watcherExclude": map[string]bool{
						"**/.git/objects/**": true,
						"**/.git/subtree-cache/**": true,
						"**/node_modules/*/**": true,
						"**/.hg/store/**": true,
					},
				},
			},
		},
		Mounts: []pkg.DevContainerMount{
			{
				Source: "${localEnv:HOME}/.claude-reactor",
				Target: "/home/claude/.claude-reactor",
				Type:   "bind",
			},
			{
				Source: "${localEnv:HOME}/.gitconfig",
				Target: "/home/claude/.gitconfig",
				Type:   "bind",
			},
		},
		ForwardPorts:      []int{3000, 8000, 8080, 9000},
		PostCreateCommand: []string{"echo", "DevContainer ready for development!"},
		ShutdownAction:    "stopContainer",
		UserEnvProbe:      "loginInteractiveShell",
	}
	
	// Add language-specific settings
	m.addLanguageSpecificSettings(devConfig, detection)
	
	return devConfig, nil
}

// addLanguageSpecificSettings adds language-specific VS Code settings
func (m *manager) addLanguageSpecificSettings(config *pkg.DevContainerConfig, detection *pkg.ProjectDetectionResult) {
	settings := config.Customizations.VSCode.Settings
	
	switch detection.ProjectType {
	case "go":
		settings["go.toolsManagement.checkForUpdates"] = "local"
		settings["go.useLanguageServer"] = true
		settings["go.gopath"] = "/go"
		settings["go.goroot"] = "/usr/local/go"
		
	case "rust":
		settings["rust-analyzer.server.path"] = "rust-analyzer"
		settings["rust-analyzer.checkOnSave.command"] = "cargo check"
		
	case "node":
		settings["typescript.preferences.importModuleSpecifier"] = "relative"
		settings["typescript.updateImportsOnFileMove.enabled"] = "always"
		settings["npm.enableScriptExplorer"] = true
		
	case "python":
		settings["python.defaultInterpreterPath"] = "/usr/local/bin/python"
		settings["python.linting.enabled"] = true
		settings["python.linting.flake8Enabled"] = true
		settings["python.formatting.provider"] = "black"
		
	case "java":
		settings["java.home"] = "/usr/lib/jvm/java-17-openjdk"
		settings["java.configuration.runtimes"] = []map[string]string{
			{"name": "JavaSE-17", "path": "/usr/lib/jvm/java-17-openjdk"},
		}
	}
	
	// Add variant-specific settings
	switch detection.Variant {
	case "cloud":
		settings["docker.showStartPage"] = false
		settings["aws.telemetry"] = false
		
	case "k8s":
		settings["vs-kubernetes"] = map[string]interface{}{
			"vs-kubernetes.kubectl-path": "/usr/local/bin/kubectl",
			"vs-kubernetes.helm-path": "/usr/local/bin/helm",
		}
	}
}