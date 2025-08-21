package template

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
	
	"claude-reactor/pkg"
)

// manager implements the TemplateManager interface
type manager struct {
	logger      pkg.Logger
	configMgr   pkg.ConfigManager
	devContMgr  pkg.DevContainerManager
	templatesDir string
}

// NewManager creates a new template manager
func NewManager(logger pkg.Logger, configMgr pkg.ConfigManager, devContMgr pkg.DevContainerManager) pkg.TemplateManager {
	homeDir, _ := os.UserHomeDir()
	templatesDir := filepath.Join(homeDir, ".claude-reactor", "templates")
	
	return &manager{
		logger:       logger,
		configMgr:    configMgr,
		devContMgr:   devContMgr,
		templatesDir: templatesDir,
	}
}

// ListTemplates returns all available project templates
func (m *manager) ListTemplates() ([]*pkg.ProjectTemplate, error) {
	if err := m.ensureTemplatesDir(); err != nil {
		return nil, fmt.Errorf("failed to ensure templates directory: %w", err)
	}
	
	var templates []*pkg.ProjectTemplate
	
	err := filepath.Walk(m.templatesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if info.Name() == "template.yaml" || info.Name() == "template.yml" {
			template, err := m.loadTemplateFromFile(path)
			if err != nil {
				m.logger.Warnf("Failed to load template from %s: %v", path, err)
				return nil // Continue processing other templates
			}
			templates = append(templates, template)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to scan templates directory: %w", err)
	}
	
	// Sort templates by name
	sort.Slice(templates, func(i, j int) bool {
		return templates[i].Name < templates[j].Name
	})
	
	return templates, nil
}

// GetTemplate retrieves a specific template by name
func (m *manager) GetTemplate(name string) (*pkg.ProjectTemplate, error) {
	templates, err := m.ListTemplates()
	if err != nil {
		return nil, err
	}
	
	for _, template := range templates {
		if template.Name == name {
			return template, nil
		}
	}
	
	return nil, fmt.Errorf("template '%s' not found", name)
}

// ValidateTemplate validates template structure and content
func (m *manager) ValidateTemplate(template *pkg.ProjectTemplate) error {
	if template.Name == "" {
		return fmt.Errorf("template name is required")
	}
	
	if template.Language == "" {
		return fmt.Errorf("template language is required")
	}
	
	if template.Variant == "" {
		return fmt.Errorf("template variant is required")
	}
	
	// Validate variant is supported
	supportedVariants := []string{"base", "go", "full", "cloud", "k8s"}
	validVariant := false
	for _, variant := range supportedVariants {
		if template.Variant == variant {
			validVariant = true
			break
		}
	}
	if !validVariant {
		return fmt.Errorf("unsupported variant: %s", template.Variant)
	}
	
	// Validate template variables
	for _, variable := range template.Variables {
		if variable.Name == "" {
			return fmt.Errorf("template variable name is required")
		}
		if variable.Type == "" {
			variable.Type = "string" // Default type
		}
		if variable.Type == "choice" && len(variable.Choices) == 0 {
			return fmt.Errorf("choice type variable '%s' must have choices", variable.Name)
		}
	}
	
	// Validate files
	if len(template.Files) == 0 {
		return fmt.Errorf("template must have at least one file")
	}
	
	for _, file := range template.Files {
		if file.Path == "" {
			return fmt.Errorf("template file path is required")
		}
		if file.Content == "" && file.Source == "" {
			return fmt.Errorf("template file '%s' must have either content or source", file.Path)
		}
	}
	
	return nil
}

// ScaffoldProject creates a new project from template
func (m *manager) ScaffoldProject(templateName, projectPath, projectName string, variables map[string]string) (*pkg.ProjectScaffoldResult, error) {
	template, err := m.GetTemplate(templateName)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}
	
	if err := m.ValidateProjectName(projectName); err != nil {
		return nil, fmt.Errorf("invalid project name: %w", err)
	}
	
	// Ensure project directory exists
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create project directory: %w", err)
	}
	
	// Get template variables with defaults
	templateVars, err := m.GetTemplateVariables(template, projectName)
	if err != nil {
		return nil, fmt.Errorf("failed to get template variables: %w", err)
	}
	
	// Merge provided variables with defaults
	for key, value := range variables {
		templateVars[key] = value
	}
	
	result := &pkg.ProjectScaffoldResult{
		ProjectPath:  projectPath,
		TemplateName: templateName,
		ProjectName:  projectName,
		Language:     template.Language,
		Framework:    template.Framework,
		Variant:      template.Variant,
		Variables:    templateVars,
		FilesCreated: []string{},
	}
	
	m.logger.Infof("Scaffolding project '%s' using template '%s'", projectName, templateName)
	
	// Create files from template
	for _, file := range template.Files {
		if err := m.createFileFromTemplate(file, projectPath, templateVars); err != nil {
			return result, fmt.Errorf("failed to create file '%s': %w", file.Path, err)
		}
		result.FilesCreated = append(result.FilesCreated, file.Path)
	}
	
	// Create .gitignore
	if len(template.GitIgnore) > 0 {
		gitignorePath := filepath.Join(projectPath, ".gitignore")
		if err := m.createGitIgnore(gitignorePath, template.GitIgnore); err != nil {
			m.logger.Warnf("Failed to create .gitignore: %v", err)
		} else {
			result.FilesCreated = append(result.FilesCreated, ".gitignore")
		}
	}
	
	// Create .claude-reactor config
	if err := m.createClaudeReactorConfig(projectPath, template.Variant); err != nil {
		m.logger.Warnf("Failed to create .claude-reactor config: %v", err)
	} else {
		result.FilesCreated = append(result.FilesCreated, ".claude-reactor")
	}
	
	// Generate devcontainer if requested
	if template.DevContainer {
		config := &pkg.Config{
			Variant:     template.Variant,
			ProjectPath: projectPath,
		}
		if err := m.devContMgr.GenerateDevContainer(projectPath, config); err != nil {
			m.logger.Warnf("Failed to generate devcontainer: %v", err)
		} else {
			result.DevContainerGen = true
		}
	}
	
	// Initialize git repository
	if err := m.initializeGitRepo(projectPath); err != nil {
		m.logger.Warnf("Failed to initialize git repository: %v", err)
	} else {
		result.GitInitialized = true
	}
	
	// Run post-create commands
	if len(template.PostCreate) > 0 {
		if err := m.runPostCreateCommands(projectPath, template.PostCreate, templateVars); err != nil {
			m.logger.Warnf("Failed to run post-create commands: %v", err)
		} else {
			result.PostCreateRan = true
		}
	}
	
	m.logger.Infof("Successfully scaffolded project '%s'", projectName)
	m.logger.Infof("  - Template: %s", templateName)
	m.logger.Infof("  - Language: %s", template.Language)
	m.logger.Infof("  - Variant: %s", template.Variant)
	m.logger.Infof("  - Files Created: %d", len(result.FilesCreated))
	
	return result, nil
}

// InteractiveScaffold runs interactive project creation wizard
func (m *manager) InteractiveScaffold(projectPath string) (*pkg.ProjectScaffoldResult, error) {
	scanner := bufio.NewScanner(os.Stdin)
	
	// List available templates
	templates, err := m.ListTemplates()
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}
	
	if len(templates) == 0 {
		return nil, fmt.Errorf("no templates available")
	}
	
	// Display templates
	fmt.Println("Available templates:")
	for i, template := range templates {
		fmt.Printf("  %d. %s (%s) - %s\n", i+1, template.Name, template.Language, template.Description)
	}
	
	// Select template
	fmt.Print("Select template (1-", len(templates), "): ")
	scanner.Scan()
	var templateIndex int
	if _, err := fmt.Sscanf(scanner.Text(), "%d", &templateIndex); err != nil || templateIndex < 1 || templateIndex > len(templates) {
		return nil, fmt.Errorf("invalid template selection")
	}
	
	selectedTemplate := templates[templateIndex-1]
	
	// Get project name
	fmt.Print("Project name: ")
	scanner.Scan()
	projectName := strings.TrimSpace(scanner.Text())
	
	if err := m.ValidateProjectName(projectName); err != nil {
		return nil, fmt.Errorf("invalid project name: %w", err)
	}
	
	// Collect template variables
	variables := make(map[string]string)
	
	for _, variable := range selectedTemplate.Variables {
		if variable.Type == "choice" {
			fmt.Printf("%s (%s) [choices: %s]: ", variable.Description, variable.Name, strings.Join(variable.Choices, ", "))
		} else {
			defaultStr := ""
			if variable.Default != nil {
				defaultStr = fmt.Sprintf(" [default: %v]", variable.Default)
			}
			fmt.Printf("%s (%s)%s: ", variable.Description, variable.Name, defaultStr)
		}
		
		scanner.Scan()
		value := strings.TrimSpace(scanner.Text())
		
		if value == "" && variable.Default != nil {
			value = fmt.Sprintf("%v", variable.Default)
		}
		
		if value == "" && variable.Required {
			return nil, fmt.Errorf("variable '%s' is required", variable.Name)
		}
		
		if variable.Type == "choice" && value != "" {
			validChoice := false
			for _, choice := range variable.Choices {
				if value == choice {
					validChoice = true
					break
				}
			}
			if !validChoice {
				return nil, fmt.Errorf("invalid choice for '%s': %s", variable.Name, value)
			}
		}
		
		if value != "" {
			variables[variable.Name] = value
		}
	}
	
	// Create project directory
	fullProjectPath := filepath.Join(projectPath, projectName)
	
	// Confirm creation
	fmt.Printf("Create project '%s' at '%s'? (y/N): ", projectName, fullProjectPath)
	scanner.Scan()
	confirmation := strings.ToLower(strings.TrimSpace(scanner.Text()))
	
	if confirmation != "y" && confirmation != "yes" {
		return nil, fmt.Errorf("project creation cancelled")
	}
	
	return m.ScaffoldProject(selectedTemplate.Name, fullProjectPath, projectName, variables)
}

// ValidateProjectName checks if project name is valid
func (m *manager) ValidateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	
	if len(name) > 100 {
		return fmt.Errorf("project name too long (max 100 characters)")
	}
	
	// Check for valid characters (alphanumeric, dash, underscore)
	validName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("project name can only contain letters, numbers, dashes, and underscores")
	}
	
	// Cannot start with dash or underscore
	if strings.HasPrefix(name, "-") || strings.HasPrefix(name, "_") {
		return fmt.Errorf("project name cannot start with dash or underscore")
	}
	
	return nil
}

// GetTemplateVariables extracts variables from template and provides defaults
func (m *manager) GetTemplateVariables(template *pkg.ProjectTemplate, projectName string) (map[string]string, error) {
	variables := make(map[string]string)
	
	// Add standard variables
	variables["PROJECT_NAME"] = projectName
	variables["PROJECT_NAME_UPPER"] = strings.ToUpper(projectName)
	variables["PROJECT_NAME_LOWER"] = strings.ToLower(projectName)
	variables["PROJECT_NAME_TITLE"] = strings.Title(projectName)
	
	// Add template-specific variables with defaults
	for _, variable := range template.Variables {
		if variable.Default != nil {
			variables[variable.Name] = fmt.Sprintf("%v", variable.Default)
		}
	}
	
	return variables, nil
}

// RenderTemplate processes template variables in content
func (m *manager) RenderTemplate(content string, variables map[string]string) (string, error) {
	tmpl, err := template.New("template").Parse(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}
	
	var result strings.Builder
	if err := tmpl.Execute(&result, variables); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	
	return result.String(), nil
}

// ensureTemplatesDir creates the templates directory if it doesn't exist
func (m *manager) ensureTemplatesDir() error {
	if err := os.MkdirAll(m.templatesDir, 0755); err != nil {
		return fmt.Errorf("failed to create templates directory: %w", err)
	}
	
	// Initialize with built-in templates if directory is empty
	entries, err := os.ReadDir(m.templatesDir)
	if err != nil {
		return err
	}
	
	if len(entries) == 0 {
		return m.initializeBuiltinTemplates()
	}
	
	return nil
}

// loadTemplateFromFile loads a template from a YAML file
func (m *manager) loadTemplateFromFile(path string) (*pkg.ProjectTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file: %w", err)
	}
	
	var template pkg.ProjectTemplate
	if err := yaml.Unmarshal(data, &template); err != nil {
		return nil, fmt.Errorf("failed to parse template YAML: %w", err)
	}
	
	if err := m.ValidateTemplate(&template); err != nil {
		return nil, fmt.Errorf("invalid template: %w", err)
	}
	
	return &template, nil
}

// createFileFromTemplate creates a file from template configuration
func (m *manager) createFileFromTemplate(file pkg.TemplateFile, projectPath string, variables map[string]string) error {
	filePath := filepath.Join(projectPath, file.Path)
	
	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	var content string
	var err error
	
	if file.Content != "" {
		content = file.Content
	} else if file.Source != "" {
		// Load content from source file
		sourceData, err := os.ReadFile(filepath.Join(filepath.Dir(filePath), file.Source))
		if err != nil {
			return fmt.Errorf("failed to read source file: %w", err)
		}
		content = string(sourceData)
	}
	
	// Process template variables if this is a template file
	if file.Template {
		content, err = m.RenderTemplate(content, variables)
		if err != nil {
			return fmt.Errorf("failed to render template: %w", err)
		}
	}
	
	// Write file
	fileMode := os.FileMode(0644)
	if file.Executable {
		fileMode = 0755
	}
	
	if err := os.WriteFile(filePath, []byte(content), fileMode); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	return nil
}

// createGitIgnore creates a .gitignore file
func (m *manager) createGitIgnore(path string, entries []string) error {
	content := strings.Join(entries, "\n") + "\n"
	return os.WriteFile(path, []byte(content), 0644)
}

// initializeGitRepo initializes a git repository
func (m *manager) initializeGitRepo(projectPath string) error {
	// Check if git is available
	if _, err := os.Stat("/usr/bin/git"); os.IsNotExist(err) {
		return fmt.Errorf("git not available")
	}
	
	// Initialize repo (this would need proper git command execution)
	// For now, just create .git directory placeholder
	gitDir := filepath.Join(projectPath, ".git")
	return os.MkdirAll(gitDir, 0755)
}

// runPostCreateCommands executes post-creation commands
func (m *manager) runPostCreateCommands(projectPath string, commands []string, variables map[string]string) error {
	// This would execute the commands in the project directory
	// For now, just log what would be executed
	m.logger.Infof("Would execute post-create commands:")
	for _, cmd := range commands {
		renderedCmd, err := m.RenderTemplate(cmd, variables)
		if err != nil {
			return err
		}
		m.logger.Infof("  - %s", renderedCmd)
	}
	return nil
}

// initializeBuiltinTemplates creates default templates - implementation in builtin.go

// CreateTemplate creates a new template from existing project
func (m *manager) CreateTemplate(projectPath, templateName string) (*pkg.ProjectTemplate, error) {
	if err := m.ValidateProjectName(templateName); err != nil {
		return nil, fmt.Errorf("invalid template name: %w", err)
	}
	
	// Detect project type
	detection, err := m.devContMgr.DetectProjectType(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect project type: %w", err)
	}
	
	template := &pkg.ProjectTemplate{
		Name:        templateName,
		Description: fmt.Sprintf("Generated template for %s project", detection.ProjectType),
		Language:    detection.ProjectType,
		Framework:   strings.Join(detection.Frameworks, ","),
		Variant:     detection.Variant,
		Version:     "1.0.0",
		Tags:        []string{"generated"},
		Files:       []pkg.TemplateFile{},
		Variables:   []pkg.TemplateVar{},
		DevContainer: true,
	}
	
	// Scan project files and create template files
	err = filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip certain directories and files
		relPath, _ := filepath.Rel(projectPath, path)
		if m.shouldSkipFile(relPath, info) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		
		if !info.IsDir() {
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", path, err)
			}
			
			templateFile := pkg.TemplateFile{
				Path:     relPath,
				Content:  string(content),
				Template: m.isTemplateFile(relPath, string(content)),
			}
			
			template.Files = append(template.Files, templateFile)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to scan project: %w", err)
	}
	
	// Save template
	templateDir := filepath.Join(m.templatesDir, templateName)
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create template directory: %w", err)
	}
	
	templateFile := filepath.Join(templateDir, "template.yaml")
	data, err := yaml.Marshal(template)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal template: %w", err)
	}
	
	if err := os.WriteFile(templateFile, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to save template: %w", err)
	}
	
	m.logger.Infof("Created template '%s' from project at %s", templateName, projectPath)
	return template, nil
}

// InstallTemplate installs template from file or URL
func (m *manager) InstallTemplate(source string) error {
	// For now, just support local file installation
	if !strings.HasSuffix(source, ".yaml") && !strings.HasSuffix(source, ".yml") {
		return fmt.Errorf("only YAML template files are supported")
	}
	
	template, err := m.loadTemplateFromFile(source)
	if err != nil {
		return fmt.Errorf("failed to load template from source: %w", err)
	}
	
	// Create template directory
	templateDir := filepath.Join(m.templatesDir, template.Name)
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		return fmt.Errorf("failed to create template directory: %w", err)
	}
	
	// Copy template file
	templateFile := filepath.Join(templateDir, "template.yaml")
	sourceData, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}
	
	if err := os.WriteFile(templateFile, sourceData, 0644); err != nil {
		return fmt.Errorf("failed to install template: %w", err)
	}
	
	m.logger.Infof("Installed template '%s' from %s", template.Name, source)
	return nil
}

// UninstallTemplate removes a template
func (m *manager) UninstallTemplate(name string) error {
	templateDir := filepath.Join(m.templatesDir, name)
	
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		return fmt.Errorf("template '%s' not found", name)
	}
	
	if err := os.RemoveAll(templateDir); err != nil {
		return fmt.Errorf("failed to remove template: %w", err)
	}
	
	m.logger.Infof("Uninstalled template '%s'", name)
	return nil
}

// GetTemplatesForLanguage returns templates for specific language
func (m *manager) GetTemplatesForLanguage(language string) ([]*pkg.ProjectTemplate, error) {
	templates, err := m.ListTemplates()
	if err != nil {
		return nil, err
	}
	
	var filtered []*pkg.ProjectTemplate
	for _, template := range templates {
		if strings.EqualFold(template.Language, language) {
			filtered = append(filtered, template)
		}
	}
	
	return filtered, nil
}

// GetRecommendedTemplate suggests best template for project type
func (m *manager) GetRecommendedTemplate(detection *pkg.ProjectDetectionResult) (*pkg.ProjectTemplate, error) {
	templates, err := m.GetTemplatesForLanguage(detection.ProjectType)
	if err != nil {
		return nil, err
	}
	
	if len(templates) == 0 {
		return nil, fmt.Errorf("no templates found for language: %s", detection.ProjectType)
	}
	
	// Find best matching template based on frameworks and variant
	var bestTemplate *pkg.ProjectTemplate
	bestScore := 0.0
	
	for _, template := range templates {
		score := 0.0
		
		// Match variant
		if template.Variant == detection.Variant {
			score += 0.5
		}
		
		// Match frameworks
		if template.Framework != "" {
			templateFrameworks := strings.Split(template.Framework, ",")
			for _, tf := range templateFrameworks {
				for _, df := range detection.Frameworks {
					if strings.EqualFold(strings.TrimSpace(tf), df) {
						score += 0.3
					}
				}
			}
		}
		
		// Prefer newer versions
		if template.Version != "" {
			score += 0.1
		}
		
		if score > bestScore {
			bestScore = score
			bestTemplate = template
		}
	}
	
	if bestTemplate == nil {
		bestTemplate = templates[0] // Fallback to first template
	}
	
	return bestTemplate, nil
}

// shouldSkipFile determines if a file should be skipped during template creation
func (m *manager) shouldSkipFile(relPath string, info os.FileInfo) bool {
	skipDirs := []string{".git", "node_modules", "target", ".idea", ".vscode", "build", "dist"}
	skipFiles := []string{".DS_Store", "Thumbs.db", ".env", ".claude-reactor"}
	
	for _, dir := range skipDirs {
		if strings.Contains(relPath, dir) {
			return true
		}
	}
	
	for _, file := range skipFiles {
		if strings.Contains(relPath, file) {
			return true
		}
	}
	
	return false
}

// isTemplateFile determines if file content should be templated
func (m *manager) isTemplateFile(path, content string) bool {
	// Check if content contains template variables
	templatePattern := regexp.MustCompile(`\{\{\.[\w_]+\}\}`)
	return templatePattern.MatchString(content)
}

// createClaudeReactorConfig creates .claude-reactor config file in the specified directory
func (m *manager) createClaudeReactorConfig(projectPath, variant string) error {
	configPath := filepath.Join(projectPath, ".claude-reactor")
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()
	
	// Write configuration in bash script format for compatibility
	fmt.Fprintf(file, "variant=%s\n", variant)
	
	return nil
}