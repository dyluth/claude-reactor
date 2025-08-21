package hotreload

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"claude-reactor/pkg"
)

// buildTrigger implements pkg.BuildTrigger for language-specific build operations
type buildTrigger struct {
	logger pkg.Logger
}

// NewBuildTrigger creates a new build trigger implementation
func NewBuildTrigger(logger pkg.Logger) pkg.BuildTrigger {
	return &buildTrigger{
		logger: logger,
	}
}

func (bt *buildTrigger) DetectProjectType(projectPath string) (*pkg.ProjectBuildInfo, error) {
	info := &pkg.ProjectBuildInfo{
		WatchPatterns:  []string{"**/*.{go,js,ts,py,rs,java,kt}"},
		IgnorePatterns: []string{"node_modules/", ".git/", "target/", "build/", "dist/", "__pycache__/"},
	}
	
	// Check for Go project
	if bt.fileExists(filepath.Join(projectPath, "go.mod")) {
		info.Type = "go"
		info.Framework = "go"
		info.BuildCommand = []string{"go", "build", "."}
		info.TestCommand = []string{"go", "test", "./..."}
		info.WatchPatterns = []string{"**/*.go", "go.mod", "go.sum"}
		info.SupportsHotReload = true
		info.Confidence = 95.0
		
		// Check for specific Go frameworks
		if bt.containsImport(projectPath, "github.com/gin-gonic/gin") {
			info.Framework = "gin"
			info.StartCommand = []string{"go", "run", "."}
		} else if bt.containsImport(projectPath, "github.com/gorilla/mux") {
			info.Framework = "gorilla"
			info.StartCommand = []string{"go", "run", "."}
		}
		return info, nil
	}
	
	// Check for Node.js project
	if bt.fileExists(filepath.Join(projectPath, "package.json")) {
		info.Type = "nodejs"
		info.Framework = "node"
		info.BuildCommand = []string{"npm", "run", "build"}
		info.TestCommand = []string{"npm", "test"}
		info.StartCommand = []string{"npm", "start"}
		info.WatchPatterns = []string{"**/*.{js,ts,jsx,tsx}", "package.json"}
		info.SupportsHotReload = true
		info.Confidence = 90.0
		
		// Detect specific frameworks
		packageJson := bt.readPackageJson(projectPath)
		if strings.Contains(packageJson, "\"react\"") {
			info.Framework = "react"
			info.StartCommand = []string{"npm", "run", "dev"}
		} else if strings.Contains(packageJson, "\"next\"") {
			info.Framework = "nextjs"
			info.StartCommand = []string{"npm", "run", "dev"}
		} else if strings.Contains(packageJson, "\"vue\"") {
			info.Framework = "vue"
			info.StartCommand = []string{"npm", "run", "serve"}
		} else if strings.Contains(packageJson, "\"express\"") {
			info.Framework = "express"
			info.StartCommand = []string{"npm", "run", "dev"}
		}
		return info, nil
	}
	
	// Check for Rust project
	if bt.fileExists(filepath.Join(projectPath, "Cargo.toml")) {
		info.Type = "rust"
		info.Framework = "cargo"
		info.BuildCommand = []string{"cargo", "build"}
		info.TestCommand = []string{"cargo", "test"}
		info.StartCommand = []string{"cargo", "run"}
		info.WatchPatterns = []string{"**/*.rs", "Cargo.toml", "Cargo.lock"}
		info.SupportsHotReload = false
		info.Confidence = 95.0
		return info, nil
	}
	
	// Check for Python project
	if bt.fileExists(filepath.Join(projectPath, "requirements.txt")) || 
		bt.fileExists(filepath.Join(projectPath, "pyproject.toml")) ||
		bt.fileExists(filepath.Join(projectPath, "Pipfile")) {
		info.Type = "python"
		info.Framework = "python"
		info.BuildCommand = []string{"python", "-m", "pip", "install", "-r", "requirements.txt"}
		info.TestCommand = []string{"python", "-m", "pytest"}
		info.WatchPatterns = []string{"**/*.py", "requirements.txt", "pyproject.toml"}
		info.SupportsHotReload = true
		info.Confidence = 85.0
		
		// Check for specific frameworks
		if bt.containsPythonPackage(projectPath, "django") {
			info.Framework = "django"
			info.StartCommand = []string{"python", "manage.py", "runserver"}
		} else if bt.containsPythonPackage(projectPath, "flask") {
			info.Framework = "flask"
			info.StartCommand = []string{"python", "app.py"}
		} else if bt.containsPythonPackage(projectPath, "fastapi") {
			info.Framework = "fastapi"
			info.StartCommand = []string{"uvicorn", "main:app", "--reload"}
		}
		return info, nil
	}
	
	// Check for Java project
	if bt.fileExists(filepath.Join(projectPath, "pom.xml")) {
		info.Type = "java"
		info.Framework = "maven"
		info.BuildCommand = []string{"mvn", "compile"}
		info.TestCommand = []string{"mvn", "test"}
		info.WatchPatterns = []string{"**/*.java", "pom.xml"}
		info.SupportsHotReload = false
		info.Confidence = 90.0
		
		// Check for Spring Boot
		pomContent := bt.readFile(filepath.Join(projectPath, "pom.xml"))
		if strings.Contains(pomContent, "spring-boot") {
			info.Framework = "springboot"
			info.StartCommand = []string{"mvn", "spring-boot:run"}
			info.SupportsHotReload = true
		}
		return info, nil
	}
	
	if bt.fileExists(filepath.Join(projectPath, "build.gradle")) || 
		bt.fileExists(filepath.Join(projectPath, "build.gradle.kts")) {
		info.Type = "java"
		info.Framework = "gradle"
		info.BuildCommand = []string{"./gradlew", "build"}
		info.TestCommand = []string{"./gradlew", "test"}
		info.WatchPatterns = []string{"**/*.java", "build.gradle", "build.gradle.kts"}
		info.SupportsHotReload = false
		info.Confidence = 90.0
		return info, nil
	}
	
	// Default unknown project
	info.Type = "unknown"
	info.Framework = "generic"
	info.BuildCommand = []string{"echo", "No build command detected"}
	info.Confidence = 10.0
	return info, nil
}

func (bt *buildTrigger) GetBuildCommand(projectPath string, projectType string) ([]string, error) {
	info, err := bt.DetectProjectType(projectPath)
	if err != nil {
		return nil, err
	}
	
	if projectType != "" && projectType != info.Type {
		// Override detected type with user-specified type
		switch projectType {
		case "go":
			return []string{"go", "build", "."}, nil
		case "nodejs":
			return []string{"npm", "run", "build"}, nil
		case "rust":
			return []string{"cargo", "build"}, nil
		case "python":
			return []string{"python", "-m", "pip", "install", "-r", "requirements.txt"}, nil
		case "java":
			if bt.fileExists(filepath.Join(projectPath, "pom.xml")) {
				return []string{"mvn", "compile"}, nil
			}
			return []string{"./gradlew", "build"}, nil
		default:
			return []string{"echo", "Unknown project type: " + projectType}, nil
		}
	}
	
	return info.BuildCommand, nil
}

// NeedsRebuild determines if a rebuild is needed based on changed files
func (bt *buildTrigger) NeedsRebuild(changedFiles []string, buildInfo *pkg.ProjectBuildInfo) bool {
	if len(changedFiles) == 0 || buildInfo == nil {
		return false
	}
	
	relevantExts := getRelevantFileExtensions(buildInfo.Type)
	if len(relevantExts) == 0 {
		// Unknown project type - always rebuild
		return true
	}
	
	for _, file := range changedFiles {
		if isRelevantFile(file, buildInfo.Type) {
			return true
		}
	}
	
	return false
}

func (bt *buildTrigger) ExecuteBuild(projectPath string, buildCmd []string, options *pkg.BuildOptions) (*pkg.BuildResult, error) {
	if len(buildCmd) == 0 {
		bt.logger.Warnf("No build command specified, returning success")
		return &pkg.BuildResult{
			Success:   true,
			Command:   []string{},
			Output:    "No build command specified",
			Duration:  "0s",
			Timestamp: time.Now().Format(time.RFC3339),
			ExitCode:  0,
		}, nil
	}
	
	start := time.Now()
	
	// Set default options
	if options == nil {
		options = &pkg.BuildOptions{
			WorkingDir:    projectPath,
			CaptureOutput: true,
			Timeout:       300, // 5 minutes default
		}
	}
	
	if options.WorkingDir == "" {
		options.WorkingDir = projectPath
	}
	
	// Validate working directory exists
	if _, err := os.Stat(options.WorkingDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("working directory does not exist: %s", options.WorkingDir)
	}
	
	bt.logger.Infof("Executing build: %v in %s", buildCmd, options.WorkingDir)
	
	// Create command
	cmd := exec.Command(buildCmd[0], buildCmd[1:]...)
	cmd.Dir = options.WorkingDir
	
	// Set environment variables
	if options.Environment != nil {
		env := os.Environ()
		for key, value := range options.Environment {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
		cmd.Env = env
	}
	
	// Execute command
	var output []byte
	var err error
	
	if options.CaptureOutput {
		output, err = cmd.CombinedOutput()
	} else {
		err = cmd.Run()
	}
	
	duration := time.Since(start)
	
	// Create result
	result := &pkg.BuildResult{
		Success:     err == nil,
		Command:     buildCmd,
		WorkingDir:  options.WorkingDir,
		Output:      string(output),
		Duration:    duration.String(),
		Timestamp:   time.Now().Format(time.RFC3339),
		ExitCode:    0,
	}
	
	if err != nil {
		result.Error = err.Error()
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.ExitCode = 1
		}
		bt.logger.Errorf("Build failed: %v", err)
	} else {
		bt.logger.Infof("Build succeeded in %v", duration)
	}
	
	return result, nil
}

func (bt *buildTrigger) GetHotReloadConfig(projectPath string, projectType string) (*pkg.HotReloadConfig, error) {
	info, err := bt.DetectProjectType(projectPath)
	if err != nil {
		return nil, err
	}
	
	config := &pkg.HotReloadConfig{
		Enabled: info.SupportsHotReload,
		Host:    "localhost",
	}
	
	switch info.Framework {
	case "react":
		config.Port = 3000
		config.StartCommand = []string{"npm", "run", "dev"}
		config.FullReloadPatterns = []string{"public/**/*", "package.json"}
		
	case "nextjs":
		config.Port = 3000
		config.StartCommand = []string{"npm", "run", "dev"}
		config.FullReloadPatterns = []string{"next.config.js", "package.json"}
		
	case "vue":
		config.Port = 8080
		config.StartCommand = []string{"npm", "run", "serve"}
		config.FullReloadPatterns = []string{"vue.config.js", "package.json"}
		
	case "gin":
		config.Port = 8080
		config.StartCommand = []string{"go", "run", "."}
		config.FullReloadPatterns = []string{"go.mod", "go.sum"}
		
	case "django":
		config.Port = 8000
		config.StartCommand = []string{"python", "manage.py", "runserver"}
		config.FullReloadPatterns = []string{"settings.py", "requirements.txt"}
		
	case "flask":
		config.Port = 5000
		config.StartCommand = []string{"python", "app.py"}
		config.Environment = map[string]string{
			"FLASK_ENV": "development",
			"FLASK_DEBUG": "1",
		}
		
	case "fastapi":
		config.Port = 8000
		config.StartCommand = []string{"uvicorn", "main:app", "--reload", "--host", "0.0.0.0"}
		
	case "springboot":
		config.Port = 8080
		config.StartCommand = []string{"mvn", "spring-boot:run"}
		config.Environment = map[string]string{
			"SPRING_PROFILES_ACTIVE": "development",
		}
		
	default:
		if !info.SupportsHotReload {
			config.Enabled = false
		}
	}
	
	return config, nil
}

func (bt *buildTrigger) ValidateBuildResult(result *pkg.BuildResult) error {
	if result == nil {
		return fmt.Errorf("build result is nil")
	}
	
	if !result.Success {
		return fmt.Errorf("build failed: %s", result.Error)
	}
	
	if result.ExitCode != 0 {
		return fmt.Errorf("build exited with code %d", result.ExitCode)
	}
	
	return nil
}

// Helper functions

func (bt *buildTrigger) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (bt *buildTrigger) readFile(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(content)
}

func (bt *buildTrigger) readPackageJson(projectPath string) string {
	return bt.readFile(filepath.Join(projectPath, "package.json"))
}

func (bt *buildTrigger) containsImport(projectPath, importPath string) bool {
	goModPath := filepath.Join(projectPath, "go.mod")
	content := bt.readFile(goModPath)
	return strings.Contains(content, importPath)
}

func (bt *buildTrigger) containsPythonPackage(projectPath, packageName string) bool {
	// Check requirements.txt
	reqPath := filepath.Join(projectPath, "requirements.txt")
	if content := bt.readFile(reqPath); content != "" {
		if strings.Contains(content, packageName) {
			return true
		}
	}
	
	// Check pyproject.toml
	pyprojectPath := filepath.Join(projectPath, "pyproject.toml")
	if content := bt.readFile(pyprojectPath); content != "" {
		if strings.Contains(content, packageName) {
			return true
		}
	}
	
	return false
}

// getRelevantFileExtensions returns file extensions that are relevant for the project type
func getRelevantFileExtensions(projectType string) []string {
	switch projectType {
	case "go":
		return []string{".go", ".mod", ".sum"}
	case "nodejs":
		return []string{".js", ".ts", ".jsx", ".tsx", ".json", ".mjs"}
	case "python":
		return []string{".py", ".pyx", ".pyi", ".txt", ".toml", ".cfg"}
	case "rust":
		return []string{".rs", ".toml"}
	case "java":
		return []string{".java", ".xml", ".gradle", ".properties"}
	default:
		return []string{}
	}
}

// isRelevantFile checks if a file path contains relevant extensions for the project type
func isRelevantFile(filePath, projectType string) bool {
	relevantExts := getRelevantFileExtensions(projectType)
	if len(relevantExts) == 0 {
		return false
	}
	
	ext := filepath.Ext(filePath)
	for _, relevantExt := range relevantExts {
		if ext == relevantExt {
			return true
		}
	}
	
	return false
}