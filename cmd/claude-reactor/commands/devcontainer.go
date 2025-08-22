package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"claude-reactor/pkg"
)

// NewDevContainerCmd creates the devcontainer command for VS Code integration
func NewDevContainerCmd(app *pkg.AppContainer) *cobra.Command {
	var devcontainerCmd = &cobra.Command{
		Use:   "devcontainer",
		Short: "VS Code Dev Container integration",
		Long:  `Generate and manage VS Code Dev Container configurations for seamless IDE integration.`,
	}
	
	// Generate subcommand
	generateCmd := &cobra.Command{
		Use:   "generate [project-path]",
		Short: "Generate .devcontainer configuration",
		Long: `Generate VS Code Dev Container configuration based on project detection.
		
This command will:
- Detect your project type (Go, Rust, Node.js, Python, Java, etc.)
- Select the appropriate claude-reactor container variant
- Create .devcontainer/devcontainer.json with optimal settings
- Configure VS Code extensions for your tech stack
- Set up development tools and debugging`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return generateDevContainer(cmd, args, app)
		},
	}
	
	// Validate subcommand
	validateCmd := &cobra.Command{
		Use:   "validate [project-path]",
		Short: "Validate existing .devcontainer configuration",
		Long:  "Check if existing .devcontainer configuration is valid and properly structured.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return validateDevContainer(cmd, args, app)
		},
	}
	
	// Update subcommand
	updateCmd := &cobra.Command{
		Use:   "update [project-path]",
		Short: "Update existing .devcontainer configuration",
		Long:  "Update existing .devcontainer configuration with latest templates and settings.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return updateDevContainer(cmd, args, app)
		},
	}
	
	// Remove subcommand
	removeCmd := &cobra.Command{
		Use:   "remove [project-path]",
		Short: "Remove .devcontainer configuration",
		Long:  "Remove .devcontainer directory and configurations from project.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return removeDevContainer(cmd, args, app)
		},
	}
	
	// Info subcommand
	infoCmd := &cobra.Command{
		Use:   "info [project-path]",
		Short: "Analyze project for devcontainer configuration",
		Long:  "Analyze project structure and show recommended devcontainer settings.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return showDevContainerInfo(cmd, args, app)
		},
	}
	
	// Add flags
	generateCmd.Flags().String("image", "", "Force specific container image (base, go, full, cloud, k8s)")
	generateCmd.Flags().Bool("force", false, "Overwrite existing .devcontainer configuration")
	
	// Help subcommand
	helpCmd := &cobra.Command{
		Use:   "help",
		Short: "Detailed VS Code Dev Container setup guide",
		Long:  "Show comprehensive guide for setting up VS Code Dev Containers with claude-reactor.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showVSCodeHelp(cmd, args, app)
		},
	}
	
	// Add subcommands
	devcontainerCmd.AddCommand(
		generateCmd,
		validateCmd,
		updateCmd,
		removeCmd,
		infoCmd,
		helpCmd,
	)
	
	return devcontainerCmd
}

// generateDevContainer generates a new .devcontainer configuration
func generateDevContainer(cmd *cobra.Command, args []string, app *pkg.AppContainer) error {
	projectPath := getCurrentDir()
	if len(args) > 0 {
		projectPath = args[0]
	}
	
	// Load current config or use default
	config, err := app.ConfigMgr.LoadConfig()
	if err != nil {
		config = app.ConfigMgr.GetDefaultConfig()
	}
	config.ProjectPath = projectPath
	
	// Handle explicit image flag
	if image, _ := cmd.Flags().GetString("image"); image != "" {
		if _, err := app.ConfigMgr.AutoDetectVariant(""); err != nil {
			return fmt.Errorf("invalid image '%s': %w", image, err)
		}
		config.Variant = image
	}
	
	// Check if devcontainer already exists
	force, _ := cmd.Flags().GetBool("force")
	devcontainerPath := filepath.Join(projectPath, ".devcontainer")
	if _, err := os.Stat(devcontainerPath); err == nil && !force {
		return fmt.Errorf(".devcontainer already exists at %s (use --force to overwrite)", devcontainerPath)
	}
	
	app.Logger.Infof("🔧 Generating VS Code Dev Container configuration...")
	
	// Generate devcontainer configuration
	if err := app.DevContainerMgr.GenerateDevContainer(projectPath, config); err != nil {
		return fmt.Errorf("failed to generate devcontainer: %w", err)
	}
	
	app.Logger.Infof("✅ Successfully generated .devcontainer configuration!")
	app.Logger.Infof("")
	app.Logger.Infof("📋 VS Code Setup Instructions:")
	app.Logger.Infof("   1. Install 'Dev Containers' extension: ms-vscode-remote.remote-containers")
	app.Logger.Infof("   2. Open this project in VS Code: code .")
	app.Logger.Infof("   3. VS Code will show: 'Folder contains a Dev Container configuration file'")
	app.Logger.Infof("   4. Click 'Reopen in Container' to launch your development environment")
	app.Logger.Infof("")
	app.Logger.Infof("🔧 Troubleshooting:")
	app.Logger.Infof("   • If notification missing: Command Palette → 'Dev Containers: Reopen in Container'")
	app.Logger.Infof("   • If 'Dockerfile does not exist': Ensure you opened VS Code from the project root directory")
	app.Logger.Infof("   • If build fails: Ensure Docker is running and try 'Rebuild Container'")
	app.Logger.Infof("   • If extensions missing: Check 'Extensions' view, they install automatically")
	app.Logger.Infof("   • To rebuild: Command Palette → 'Dev Containers: Rebuild Container'")
	app.Logger.Infof("   • To exit container: Command Palette → 'Dev Containers: Reopen Folder Locally'")
	app.Logger.Infof("   • Path issues: Run 'claude-reactor devcontainer validate' to check configuration")
	app.Logger.Infof("")
	app.Logger.Infof("🚀 You're all set for containerized development!")
	
	return nil
}

// validateDevContainer validates existing .devcontainer configuration
func validateDevContainer(cmd *cobra.Command, args []string, app *pkg.AppContainer) error {
	projectPath := getCurrentDir()
	if len(args) > 0 {
		projectPath = args[0]
	}
	
	app.Logger.Infof("🔍 Validating .devcontainer configuration...")
	
	if err := app.DevContainerMgr.ValidateDevContainer(projectPath); err != nil {
		app.Logger.Errorf("❌ Validation failed: %v", err)
		return err
	}
	
	app.Logger.Infof("✅ DevContainer configuration is valid!")
	return nil
}

// updateDevContainer updates existing .devcontainer configuration
func updateDevContainer(cmd *cobra.Command, args []string, app *pkg.AppContainer) error {
	projectPath := getCurrentDir()
	if len(args) > 0 {
		projectPath = args[0]
	}
	
	// Load current config or use default
	config, err := app.ConfigMgr.LoadConfig()
	if err != nil {
		config = app.ConfigMgr.GetDefaultConfig()
	}
	config.ProjectPath = projectPath
	
	app.Logger.Infof("🔄 Updating .devcontainer configuration...")
	
	if err := app.DevContainerMgr.UpdateDevContainer(projectPath, config); err != nil {
		return fmt.Errorf("failed to update devcontainer: %w", err)
	}
	
	app.Logger.Infof("✅ Successfully updated .devcontainer configuration!")
	return nil
}

// removeDevContainer removes .devcontainer configuration
func removeDevContainer(cmd *cobra.Command, args []string, app *pkg.AppContainer) error {
	projectPath := getCurrentDir()
	if len(args) > 0 {
		projectPath = args[0]
	}
	
	app.Logger.Infof("🗑️  Removing .devcontainer configuration...")
	
	if err := app.DevContainerMgr.RemoveDevContainer(projectPath); err != nil {
		return fmt.Errorf("failed to remove devcontainer: %w", err)
	}
	
	return nil
}

// showDevContainerInfo analyzes project for devcontainer configuration
func showDevContainerInfo(cmd *cobra.Command, args []string, app *pkg.AppContainer) error {
	projectPath := getCurrentDir()
	if len(args) > 0 {
		projectPath = args[0]
	}
	
	fmt.Printf("🔍 DevContainer Project Analysis\n")
	fmt.Printf("═══════════════════════════════\n\n")
	
	fmt.Printf("📁 Project Path: %s\n\n", projectPath)
	
	// Detect project type
	detection, err := app.DevContainerMgr.DetectProjectType(projectPath)
	if err != nil {
		return fmt.Errorf("failed to detect project type: %w", err)
	}
	
	fmt.Printf("🎯 Detected Project Type: %s\n", detection.ProjectType)
	fmt.Printf("🖼️  Recommended Image: %s\n", detection.Variant)
	
	if detection.Confidence > 0 {
		fmt.Printf("📊 Detection Confidence: %.0f%%\n", detection.Confidence*100)
	}
	
	if len(detection.Files) > 0 {
		fmt.Printf("\n📄 Key Files Found:\n")
		for _, file := range detection.Files {
			fmt.Printf("  • %s\n", file)
		}
	}
	
	if len(detection.Extensions) > 0 {
		fmt.Printf("\n🔌 Recommended VS Code Extensions:\n")
		count := len(detection.Extensions)
		if count > 10 {
			count = 10 // Show first 10
		}
		for i := 0; i < count; i++ {
			fmt.Printf("  • %s\n", detection.Extensions[i])
		}
		if len(detection.Extensions) > 10 {
			fmt.Printf("  ... and %d more extensions\n", len(detection.Extensions)-10)
		}
	}
	
	fmt.Printf("\n💡 Run 'claude-reactor devcontainer generate' to create .devcontainer configuration\n")
	
	return nil
}

// showVSCodeHelp displays comprehensive VS Code Dev Container setup guide
func showVSCodeHelp(cmd *cobra.Command, args []string, app *pkg.AppContainer) error {
	helpText := `
🏗️  VS Code Dev Container Integration with Claude-Reactor

═══════════════════════════════════════════════════════════════════════════════

Claude-reactor provides seamless VS Code Dev Container integration, allowing you
to develop in a fully-featured containerized environment with all the tools,
extensions, and configuration your project needs.

═══════════════════════════════════════════════════════════════════════════════

🚀 QUICK START

1. Generate devcontainer for your project:
   claude-reactor devcontainer generate

2. Open project in VS Code:
   code .

3. When prompted, click "Reopen in Container"

4. Start coding with full claude-reactor toolchain!

That's it! Your development environment is ready with all tools, extensions,
and Claude CLI access.

═══════════════════════════════════════════════════════════════════════════════

🔧 DETAILED WORKFLOW

Step 1: Project Analysis
   • Run: claude-reactor devcontainer info
   • Review detected project type, variant, and recommended extensions
   
Step 2: Generate Configuration  
   • Run: claude-reactor devcontainer generate
   • Creates .devcontainer/devcontainer.json with optimal settings
   • Force overwrite existing: claude-reactor devcontainer generate --force
   • Specify image: claude-reactor devcontainer generate --image go

Step 3: Open in VS Code
   • Launch VS Code in project directory: code .
   • VS Code detects .devcontainer/devcontainer.json automatically
   • Shows notification: "Folder contains a Dev Container configuration file"

Step 4: Container Setup
   • Click "Reopen in Container" in notification
   • Alternative: Command Palette (Ctrl/Cmd+Shift+P) → "Dev Containers: Reopen in Container"
   • VS Code downloads/builds the container image (first time: ~2-3 minutes)
   • Container starts with your project mounted
   • Extensions install automatically

Step 5: Development
   • Full access to development tools (Go, Rust, Node.js, Python, etc.)
   • Claude CLI available: claude --help
   • Git integration works seamlessly  
   • Port forwarding for web applications
   • Terminal access to containerized environment
   • File system changes persist
   • VS Code features work normally (IntelliSense, debugging, etc.)

═══════════════════════════════════════════════════════════════════════════════

🛠️  FEATURES & CAPABILITIES

Automatic Configuration:
   • Project type detection (Go, Rust, Python, Node.js, Java, C++, etc.)
   • Variant selection (base, go, full, cloud, k8s)
   • VS Code extension recommendations
   • Development tool setup
   • Port forwarding configuration

Language Support:
   • Go: Full toolchain, debugging, testing, go mod support
   • Rust: Cargo, rust-analyzer, debugging
   • Node.js: npm/yarn, debugging, TypeScript support  
   • Python: pip/conda, debugging, Jupyter support
   • Java: Maven/Gradle, debugging, Spring Boot support
   • And many more...

Claude Integration:
   • Claude CLI pre-installed and authenticated
   • Access to all claude-reactor functionality
   • Seamless AI-assisted development workflow

═══════════════════════════════════════════════════════════════════════════════

🔍 TROUBLESHOOTING

Problem: "Reopen in Container" notification doesn't appear
Solution: 
   • Ensure .devcontainer/devcontainer.json exists
   • Reload VS Code: Ctrl/Cmd+Shift+P → "Developer: Reload Window"
   • Manually trigger: Command Palette → "Dev Containers: Reopen in Container"

Problem: Container build fails
Solutions:
   • Ensure Docker is running: docker ps
   • Check Docker daemon: docker info
   • Try rebuilding: Command Palette → "Dev Containers: Rebuild Container"
   • Check .devcontainer/devcontainer.json syntax
   • Run: claude-reactor devcontainer validate

Problem: Extensions don't install
Solutions:
   • Reload window: Command Palette → "Developer: Reload Window" 
   • Manual install: Extensions view → search and install
   • Check internet connection
   • Try rebuilding container

Problem: File changes don't persist
Solution:
   • VS Code Dev Containers mount your project directory automatically
   • Changes to files in your project are persistent
   • Changes outside your project (like /usr/local) are not persistent
   • This is normal and intended behavior

Problem: Poor performance
Solutions:
   • Ensure adequate Docker resources (4GB+ RAM recommended)
   • Close unnecessary applications
   • Use Docker Desktop resource limits appropriately
   • Consider using smaller container variant (base instead of full)

Problem: Port forwarding issues
Solutions:
   • VS Code should automatically forward ports
   • Manual forward: Command Palette → "Ports: Focus on Ports View"
   • Check application is binding to 0.0.0.0, not localhost
   • Ensure firewall allows connections

═══════════════════════════════════════════════════════════════════════════════

💡 TIPS & BEST PRACTICES

1. First-Time Setup:
   • Install Dev Containers extension first: ms-vscode-remote.remote-containers
   • Ensure Docker is running before opening container
   • Be patient on first build (downloads container image)

2. Daily Workflow:
   • Open Command Palette: Ctrl/Cmd+Shift+P
   • Open terminal: Ctrl/Cmd+backtick
   • Switch between local/container: Command Palette → "Dev Containers: Reopen Folder Locally"

2. Multiple Projects:
   • Each project gets its own devcontainer configuration
   • VS Code remembers container settings per project
   • Switch between projects seamlessly

3. Team Collaboration:
   • Commit .devcontainer/ to version control
   • All team members get identical development environment
   • No more "works on my machine" issues

4. Performance Optimization:
   • First container build is slow (~2-3 minutes)
   • Subsequent builds are fast (~30 seconds) due to Docker layer caching
   • Keep Docker running to avoid startup delays

5. Custom Extensions:
   • Edit .devcontainer/devcontainer.json to add custom extensions
   • Run: claude-reactor devcontainer update to refresh configuration

═══════════════════════════════════════════════════════════════════════════════

📚 USEFUL COMMANDS

claude-reactor devcontainer info              # Analyze current project
claude-reactor devcontainer generate          # Create .devcontainer config  
claude-reactor devcontainer generate --force  # Overwrite existing config
claude-reactor devcontainer validate          # Check config validity
claude-reactor devcontainer update            # Update existing config
claude-reactor devcontainer remove            # Delete .devcontainer directory

VS Code Command Palette (Ctrl/Cmd+Shift+P):
   • Dev Containers: Reopen in Container
   • Dev Containers: Reopen Folder Locally  
   • Dev Containers: Rebuild Container
   • Dev Containers: Show Container Log
   • Ports: Focus on Ports View

═══════════════════════════════════════════════════════════════════════════════

🌟 WHY USE DEV CONTAINERS?

Benefits:
   • Consistent development environment across team
   • No "works on my machine" issues
   • Easy onboarding for new team members
   • Isolate project dependencies
   • Full-featured IDE experience in container
   • Claude CLI integration for AI-assisted development

Perfect For:
   • Team projects with complex dependencies
   • Multi-language projects  
   • Projects requiring specific tool versions
   • CI/CD environment matching
   • Client work with isolation requirements
   • Educational environments

Experience:
   • Feels like local development
   • All VS Code features work normally
   • Fast file access and editing
   • Port forwarding works for web applications
   • Extensions provide full functionality

Happy containerized coding! 🚀

Run 'claude-reactor devcontainer help' anytime to see this guide.

`
	
	fmt.Print(helpText)
	return nil
}

