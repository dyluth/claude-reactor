package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"claude-reactor/pkg"
)

// NewTemplateCmd creates the template command for project scaffolding
func NewTemplateCmd(app *pkg.AppContainer) *cobra.Command {
	var templateCmd = &cobra.Command{
		Use:   "template",
		Short: "Project template management and scaffolding",
		Long: `The template command provides intelligent project scaffolding and template management.
		
Create new projects from templates, manage custom templates, and scaffold projects
with best practices and proper configuration for claude-reactor development.`,
		Example: `# List available templates
claude-reactor template list

# Create new project from template
claude-reactor template new go-api my-api

# Interactive project creation
claude-reactor template init

# Show template details
claude-reactor template show go-api`,
	}

	templateCmd.AddCommand(
		newTemplateListCmd(app),
		newTemplateShowCmd(app),
		newTemplateNewCmd(app),
		newTemplateInitCmd(app),
		newTemplateCreateCmd(app),
		newTemplateInstallCmd(app),
		newTemplateUninstallCmd(app),
		newTemplateValidateCmd(app),
	)

	return templateCmd
}

// newTemplateListCmd lists available templates
func newTemplateListCmd(app *pkg.AppContainer) *cobra.Command {
	var listCmd = &cobra.Command{
		Use:   "list [language]",
		Short: "List available project templates",
		Long: `Lists all available project templates or filters by language.
		
Templates are organized by language and framework, each providing a complete
project structure with best practices and proper claude-reactor integration.`,
		Example: `# List all templates
claude-reactor template list

# List Go templates only  
claude-reactor template list go

# List with details
claude-reactor template list --detailed`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return listTemplates(cmd, args, app)
		},
	}

	listCmd.Flags().BoolP("detailed", "d", false, "Show detailed template information")
	return listCmd
}

// newTemplateShowCmd shows template details
func newTemplateShowCmd(app *pkg.AppContainer) *cobra.Command {
	var showCmd = &cobra.Command{
		Use:   "show <template-name>",
		Short: "Show detailed information about a template",
		Long: `Displays comprehensive information about a specific template including:
- Description and metadata
- Files that will be created
- Template variables
- Post-creation commands
- Requirements and dependencies`,
		Example: `# Show Go API template details
claude-reactor template show go-api

# Show with file contents
claude-reactor template show go-api --files`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return showTemplate(cmd, args, app)
		},
	}

	showCmd.Flags().Bool("files", false, "Show file contents preview")
	return showCmd
}

// newTemplateNewCmd creates new project from template
func newTemplateNewCmd(app *pkg.AppContainer) *cobra.Command {
	var newCmd = &cobra.Command{
		Use:   "new <template-name> <project-name> [path]",
		Short: "Create new project from template",
		Long: `Creates a new project from the specified template with intelligent scaffolding.
		
The command will:
1. Create project directory structure
2. Generate files with template variables
3. Set up .claude-reactor configuration
4. Initialize git repository (optional)
5. Generate devcontainer configuration (optional)
6. Run post-creation commands`,
		Example: `# Create Go API project
claude-reactor template new go-api my-api

# Create in specific directory
claude-reactor template new go-api my-api ./projects/

# Skip git initialization
claude-reactor template new go-api my-api --no-git

# Set template variables
claude-reactor template new go-api my-api --var PORT=3000 --var AUTHOR="John Doe"`,
		Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			return createFromTemplate(cmd, args, app)
		},
	}

	newCmd.Flags().Bool("no-git", false, "Skip git repository initialization")
	newCmd.Flags().Bool("no-devcontainer", false, "Skip devcontainer generation")
	newCmd.Flags().StringSlice("var", []string{}, "Set template variables (key=value)")
	newCmd.Flags().Bool("force", false, "Overwrite existing directory")
	return newCmd
}

// newTemplateInitCmd runs interactive project creation
func newTemplateInitCmd(app *pkg.AppContainer) *cobra.Command {
	var initCmd = &cobra.Command{
		Use:   "init [path]",
		Short: "Interactive project initialization wizard",
		Long: `Runs an interactive wizard to create a new project.
		
The wizard will:
1. Show available templates
2. Help you select the best template
3. Collect project information
4. Set template variables
5. Create the project with all configurations`,
		Example: `# Start interactive wizard in current directory
claude-reactor template init

# Start wizard in specific directory
claude-reactor template init ./projects/`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return interactiveInit(cmd, args, app)
		},
	}

	return initCmd
}

// newTemplateCreateCmd creates template from existing project
func newTemplateCreateCmd(app *pkg.AppContainer) *cobra.Command {
	var createCmd = &cobra.Command{
		Use:   "create <template-name> [project-path]",
		Short: "Create template from existing project",
		Long: `Creates a new template based on an existing project structure.
		
This is useful for:
- Creating custom templates from your projects
- Sharing project structures with team members
- Building organization-specific templates`,
		Example: `# Create template from current directory
claude-reactor template create my-template

# Create template from specific project
claude-reactor template create my-template ./my-project/

# Include custom metadata
claude-reactor template create my-template --description "My custom template" --author "John Doe"`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return createTemplate(cmd, args, app)
		},
	}

	createCmd.Flags().String("description", "", "Template description")
	createCmd.Flags().String("author", "", "Template author")
	createCmd.Flags().StringSlice("tags", []string{}, "Template tags")
	return createCmd
}

// newTemplateInstallCmd installs template from file or URL
func newTemplateInstallCmd(app *pkg.AppContainer) *cobra.Command {
	var installCmd = &cobra.Command{
		Use:   "install <source>",
		Short: "Install template from file or URL",
		Long: `Installs a template from a local file or remote URL.
		
Supported sources:
- Local YAML files (template.yaml)
- Git repositories (future)
- Template registries (future)`,
		Example: `# Install from local file
claude-reactor template install ./my-template.yaml

# Install from URL (future)
claude-reactor template install https://example.com/template.yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return installTemplate(cmd, args, app)
		},
	}

	return installCmd
}

// newTemplateUninstallCmd removes a template
func newTemplateUninstallCmd(app *pkg.AppContainer) *cobra.Command {
	var uninstallCmd = &cobra.Command{
		Use:   "uninstall <template-name>",
		Short: "Remove an installed template",
		Long: `Removes a template from the local template registry.
		
This will permanently delete the template configuration and files.
Built-in templates cannot be uninstalled.`,
		Example: `# Remove custom template
claude-reactor template uninstall my-custom-template

# Force removal without confirmation
claude-reactor template uninstall my-template --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return uninstallTemplate(cmd, args, app)
		},
	}

	uninstallCmd.Flags().Bool("force", false, "Force removal without confirmation")
	return uninstallCmd
}

// newTemplateValidateCmd validates a template
func newTemplateValidateCmd(app *pkg.AppContainer) *cobra.Command {
	var validateCmd = &cobra.Command{
		Use:   "validate <template-file>",
		Short: "Validate template configuration",
		Long: `Validates a template configuration file for correctness.
		
Checks:
- YAML syntax
- Required fields
- File references
- Variable definitions
- Template syntax`,
		Example: `# Validate local template file
claude-reactor template validate ./template.yaml

# Validate installed template
claude-reactor template validate go-api`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return validateTemplate(cmd, args, app)
		},
	}

	return validateCmd
}

// listTemplates lists available project templates
func listTemplates(cmd *cobra.Command, args []string, app *pkg.AppContainer) error {
	detailed, _ := cmd.Flags().GetBool("detailed")
	
	var templates []*pkg.ProjectTemplate
	var err error
	
	if len(args) > 0 {
		// Filter by language
		language := args[0]
		templates, err = app.TemplateMgr.GetTemplatesForLanguage(language)
		if err != nil {
			return fmt.Errorf("failed to get templates for language %s: %w", language, err)
		}
		app.Logger.Infof("📋 Available %s templates:", strings.Title(language))
	} else {
		// Get all templates
		templates, err = app.TemplateMgr.ListTemplates()
		if err != nil {
			return fmt.Errorf("failed to list templates: %w", err)
		}
		app.Logger.Infof("📋 Available project templates:")
	}
	
	if len(templates) == 0 {
		fmt.Println("\n❌ No templates found")
		fmt.Println("💡 Templates will be automatically created on first use")
		return nil
	}
	
	if detailed {
		for _, template := range templates {
			fmt.Printf("\n🎯 %s (%s)\n", template.Name, template.Language)
			fmt.Printf("   Description: %s\n", template.Description)
			fmt.Printf("   Framework: %s\n", template.Framework)
			fmt.Printf("   Variant: %s\n", template.Variant)
			fmt.Printf("   Version: %s\n", template.Version)
			if len(template.Tags) > 0 {
				fmt.Printf("   Tags: %s\n", strings.Join(template.Tags, ", "))
			}
			fmt.Printf("   Files: %d\n", len(template.Files))
			if len(template.Variables) > 0 {
				fmt.Printf("   Variables: %d\n", len(template.Variables))
			}
			if template.DevContainer {
				fmt.Printf("   ✅ Includes VS Code Dev Container\n")
			}
		}
	} else {
		// Group by language
		languageGroups := make(map[string][]*pkg.ProjectTemplate)
		for _, template := range templates {
			languageGroups[template.Language] = append(languageGroups[template.Language], template)
		}
		
		for language, langTemplates := range languageGroups {
			fmt.Printf("\n📁 %s:\n", strings.Title(language))
			for _, template := range langTemplates {
				framework := ""
				if template.Framework != "" {
					framework = fmt.Sprintf(" (%s)", template.Framework)
				}
				devcontainer := ""
				if template.DevContainer {
					devcontainer = " 📦"
				}
				fmt.Printf("  • %s%s - %s%s\n", template.Name, framework, template.Description, devcontainer)
			}
		}
		
		fmt.Printf("\n💡 Use 'claude-reactor template show <template-name>' for details\n")
		fmt.Printf("💡 Use 'claude-reactor template new <template-name> <project-name>' to create project\n")
	}
	
	return nil
}

// showTemplate shows detailed information about a template
func showTemplate(cmd *cobra.Command, args []string, app *pkg.AppContainer) error {
	templateName := args[0]
	showFiles, _ := cmd.Flags().GetBool("files")
	
	template, err := app.TemplateMgr.GetTemplate(templateName)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}
	
	fmt.Printf("🎯 Template: %s\n", template.Name)
	fmt.Printf("📝 Description: %s\n", template.Description)
	fmt.Printf("🔧 Language: %s\n", template.Language)
	if template.Framework != "" {
		fmt.Printf("🏗️  Framework: %s\n", template.Framework)
	}
	fmt.Printf("📦 Variant: %s\n", template.Variant)
	fmt.Printf("🏷️  Version: %s\n", template.Version)
	if template.Author != "" {
		fmt.Printf("👤 Author: %s\n", template.Author)
	}
	if len(template.Tags) > 0 {
		fmt.Printf("🏷️  Tags: %s\n", strings.Join(template.Tags, ", "))
	}
	
	if template.DevContainer {
		fmt.Printf("✅ Includes VS Code Dev Container integration\n")
	}
	
	if len(template.Variables) > 0 {
		fmt.Printf("\n📋 Template Variables:\n")
		for _, variable := range template.Variables {
			defaultStr := ""
			if variable.Default != nil {
				defaultStr = fmt.Sprintf(" (default: %v)", variable.Default)
			}
			requiredStr := ""
			if variable.Required {
				requiredStr = " *required*"
			}
			fmt.Printf("  • %s (%s): %s%s%s\n", variable.Name, variable.Type, variable.Description, defaultStr, requiredStr)
			if variable.Type == "choice" && len(variable.Choices) > 0 {
				fmt.Printf("    Choices: %s\n", strings.Join(variable.Choices, ", "))
			}
		}
	}
	
	fmt.Printf("\n📁 Files (%d):\n", len(template.Files))
	for _, file := range template.Files {
		templateStr := ""
		if file.Template {
			templateStr = " [templated]"
		}
		execStr := ""
		if file.Executable {
			execStr = " [executable]"
		}
		fmt.Printf("  • %s%s%s\n", file.Path, templateStr, execStr)
		
		if showFiles && file.Content != "" {
			lines := strings.Split(file.Content, "\n")
			preview := lines
			if len(lines) > 10 {
				preview = lines[:10]
				fmt.Printf("    Preview (first 10 lines):\n")
			} else {
				fmt.Printf("    Content:\n")
			}
			for _, line := range preview {
				fmt.Printf("      %s\n", line)
			}
			if len(lines) > 10 {
				fmt.Printf("    ... (%d more lines)\n", len(lines)-10)
			}
			fmt.Println()
		}
	}
	
	if len(template.GitIgnore) > 0 {
		fmt.Printf("\n🚫 Git Ignore Patterns (%d):\n", len(template.GitIgnore))
		for _, pattern := range template.GitIgnore {
			if pattern != "" {
				fmt.Printf("  • %s\n", pattern)
			}
		}
	}
	
	if len(template.PostCreate) > 0 {
		fmt.Printf("\n⚡ Post-Creation Commands:\n")
		for _, cmd := range template.PostCreate {
			fmt.Printf("  • %s\n", cmd)
		}
	}
	
	if len(template.Dependencies) > 0 {
		fmt.Printf("\n📦 Dependencies:\n")
		for _, dep := range template.Dependencies {
			fmt.Printf("  • %s\n", dep)
		}
	}
	
	fmt.Printf("\n💡 Create project: claude-reactor template new %s <project-name>\n", template.Name)
	
	return nil
}

// createFromTemplate creates a new project from template
func createFromTemplate(cmd *cobra.Command, args []string, app *pkg.AppContainer) error {
	templateName := args[0]
	projectName := args[1]
	
	projectPath := "."
	if len(args) > 2 {
		projectPath = args[2]
	}
	
	// Get flags
	force, _ := cmd.Flags().GetBool("force")
	noGit, _ := cmd.Flags().GetBool("no-git")
	noDevcontainer, _ := cmd.Flags().GetBool("no-devcontainer")
	vars, _ := cmd.Flags().GetStringSlice("var")
	
	// Parse template variables
	variables := make(map[string]string)
	for _, v := range vars {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 {
			variables[parts[0]] = parts[1]
		} else {
			app.Logger.Warnf("Invalid variable format: %s (expected key=value)", v)
		}
	}
	
	// Build full project path
	fullProjectPath := filepath.Join(projectPath, projectName)
	
	// Check if directory exists
	if _, err := os.Stat(fullProjectPath); err == nil && !force {
		return fmt.Errorf("directory %s already exists (use --force to overwrite)", fullProjectPath)
	}
	
	app.Logger.Infof("🏗️ Creating project '%s' from template '%s'", projectName, templateName)
	
	// Create project from template
	result, err := app.TemplateMgr.ScaffoldProject(templateName, fullProjectPath, projectName, variables)
	if err != nil {
		return fmt.Errorf("failed to scaffold project: %w", err)
	}
	
	// Display results
	fmt.Printf("\n✅ Project '%s' created successfully!\n", projectName)
	fmt.Printf("📁 Location: %s\n", result.ProjectPath)
	fmt.Printf("🎯 Template: %s (%s)\n", result.TemplateName, result.Language)
	fmt.Printf("📦 Variant: %s\n", result.Variant)
	fmt.Printf("📄 Files Created: %d\n", len(result.FilesCreated))
	
	if result.DevContainerGen && !noDevcontainer {
		fmt.Printf("✅ VS Code Dev Container configured\n")
	}
	
	if result.GitInitialized && !noGit {
		fmt.Printf("✅ Git repository initialized\n")
	}
	
	if result.PostCreateRan {
		fmt.Printf("✅ Post-creation commands executed\n")
	}
	
	fmt.Printf("\n🚀 Next Steps:\n")
	fmt.Printf("  1. cd %s\n", projectName)
	if result.Language == "go" || result.Language == "rust" || result.Language == "node" {
		fmt.Printf("  2. Install dependencies (see README.md)\n")
	}
	if result.DevContainerGen {
		fmt.Printf("  3. Open in VS Code: code .\n")
		fmt.Printf("  4. Reopen in Dev Container when prompted\n")
	} else {
		fmt.Printf("  3. Start development with: claude-reactor run\n")
	}
	
	return nil
}

// interactiveInit runs the interactive project creation wizard
func interactiveInit(cmd *cobra.Command, args []string, app *pkg.AppContainer) error {
	projectPath := "."
	if len(args) > 0 {
		projectPath = args[0]
	}
	
	app.Logger.Infof("🧙 Starting interactive project creation wizard...")
	
	result, err := app.TemplateMgr.InteractiveScaffold(projectPath)
	if err != nil {
		return fmt.Errorf("interactive scaffolding failed: %w", err)
	}
	
	// Display results (same as createFromTemplate)
	fmt.Printf("\n✅ Project '%s' created successfully!\n", result.ProjectName)
	fmt.Printf("📁 Location: %s\n", result.ProjectPath)
	fmt.Printf("🎯 Template: %s (%s)\n", result.TemplateName, result.Language)
	fmt.Printf("📦 Variant: %s\n", result.Variant)
	fmt.Printf("📄 Files Created: %d\n", len(result.FilesCreated))
	
	fmt.Printf("\n🚀 Next Steps:\n")
	fmt.Printf("  1. cd %s\n", result.ProjectName)
	if result.DevContainerGen {
		fmt.Printf("  2. Open in VS Code: code .\n")
		fmt.Printf("  3. Reopen in Dev Container when prompted\n")
	} else {
		fmt.Printf("  2. Start development with: claude-reactor run\n")
	}
	
	return nil
}

// createTemplate creates a template from existing project
func createTemplate(cmd *cobra.Command, args []string, app *pkg.AppContainer) error {
	templateName := args[0]
	projectPath := "."
	if len(args) > 1 {
		projectPath = args[1]
	}
	
	description, _ := cmd.Flags().GetString("description")
	author, _ := cmd.Flags().GetString("author")
	tags, _ := cmd.Flags().GetStringSlice("tags")
	
	app.Logger.Infof("🏗️ Creating template '%s' from project at %s", templateName, projectPath)
	
	template, err := app.TemplateMgr.CreateTemplate(projectPath, templateName)
	if err != nil {
		return fmt.Errorf("failed to create template: %w", err)
	}
	
	// Update template metadata if provided
	if description != "" {
		template.Description = description
	}
	if author != "" {
		template.Author = author
	}
	if len(tags) > 0 {
		template.Tags = append(template.Tags, tags...)
	}
	
	fmt.Printf("\n✅ Template '%s' created successfully!\n", template.Name)
	fmt.Printf("📝 Description: %s\n", template.Description)
	fmt.Printf("🔧 Language: %s\n", template.Language)
	fmt.Printf("📦 Variant: %s\n", template.Variant)
	fmt.Printf("📄 Files: %d\n", len(template.Files))
	
	fmt.Printf("\n💡 Use template: claude-reactor template new %s <project-name>\n", template.Name)
	
	return nil
}

// installTemplate installs a template from file or URL
func installTemplate(cmd *cobra.Command, args []string, app *pkg.AppContainer) error {
	source := args[0]
	
	app.Logger.Infof("📦 Installing template from %s", source)
	
	if err := app.TemplateMgr.InstallTemplate(source); err != nil {
		return fmt.Errorf("failed to install template: %w", err)
	}
	
	fmt.Printf("✅ Template installed successfully!\n")
	fmt.Printf("💡 Use 'claude-reactor template list' to see all available templates\n")
	
	return nil
}

// uninstallTemplate removes a template
func uninstallTemplate(cmd *cobra.Command, args []string, app *pkg.AppContainer) error {
	templateName := args[0]
	force, _ := cmd.Flags().GetBool("force")
	
	if !force {
		// Check if template exists
		_, err := app.TemplateMgr.GetTemplate(templateName)
		if err != nil {
			return fmt.Errorf("template '%s' not found", templateName)
		}
		
		fmt.Printf("⚠️  This will permanently remove template '%s'\n", templateName)
		fmt.Print("Are you sure? (y/N): ")
		
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("❌ Cancelled")
			return nil
		}
	}
	
	app.Logger.Infof("🗑️ Removing template '%s'", templateName)
	
	if err := app.TemplateMgr.UninstallTemplate(templateName); err != nil {
		return fmt.Errorf("failed to uninstall template: %w", err)
	}
	
	fmt.Printf("✅ Template '%s' removed successfully!\n", templateName)
	
	return nil
}

// validateTemplate validates a template configuration
func validateTemplate(cmd *cobra.Command, args []string, app *pkg.AppContainer) error {
	templateSource := args[0]
	
	app.Logger.Infof("🔍 Validating template: %s", templateSource)
	
	// Check if it's a template name or file path
	var template *pkg.ProjectTemplate
	var err error
	
	if strings.HasSuffix(templateSource, ".yaml") || strings.HasSuffix(templateSource, ".yml") {
		// Validate file directly
		fmt.Printf("⚠️  Direct file validation not yet implemented\n")
		fmt.Printf("💡 Use: claude-reactor template install %s && claude-reactor template validate <template-name>\n", templateSource)
		return nil
	} else {
		// Get installed template
		template, err = app.TemplateMgr.GetTemplate(templateSource)
		if err != nil {
			return fmt.Errorf("failed to get template: %w", err)
		}
	}
	
	// Validate template
	if err := app.TemplateMgr.ValidateTemplate(template); err != nil {
		fmt.Printf("❌ Template validation failed:\n")
		fmt.Printf("   %v\n", err)
		return nil
	}
	
	fmt.Printf("✅ Template '%s' is valid!\n", template.Name)
	fmt.Printf("📝 Description: %s\n", template.Description)
	fmt.Printf("🔧 Language: %s\n", template.Language)
	fmt.Printf("📦 Variant: %s\n", template.Variant)
	fmt.Printf("📄 Files: %d\n", len(template.Files))
	if len(template.Variables) > 0 {
		fmt.Printf("🔧 Variables: %d\n", len(template.Variables))
	}
	
	return nil
}