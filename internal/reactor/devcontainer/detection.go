package devcontainer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	
	"claude-reactor/pkg"
)

// detectGoProject detects Go project characteristics
func (m *manager) detectGoProject(projectPath string, result *pkg.ProjectDetectionResult) error {
	goModPath := filepath.Join(projectPath, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		return fmt.Errorf("go.mod not found")
	}
	
	result.Files = append(result.Files, "go.mod")
	
	// Check for go.sum
	if _, err := os.Stat(filepath.Join(projectPath, "go.sum")); err == nil {
		result.Files = append(result.Files, "go.sum")
	}
	
	// Check for common Go project patterns
	if _, err := os.Stat(filepath.Join(projectPath, "main.go")); err == nil {
		result.Features = append(result.Features, "executable")
		result.Files = append(result.Files, "main.go")
	}
	
	if _, err := os.Stat(filepath.Join(projectPath, "cmd")); err == nil {
		result.Features = append(result.Features, "multi-command")
	}
	
	// Check for web frameworks
	goModContent, err := os.ReadFile(goModPath)
	if err == nil {
		content := string(goModContent)
		if strings.Contains(content, "github.com/gin-gonic/gin") {
			result.Frameworks = append(result.Frameworks, "Gin")
		}
		if strings.Contains(content, "github.com/gorilla/mux") {
			result.Frameworks = append(result.Frameworks, "Gorilla Mux")
		}
		if strings.Contains(content, "github.com/spf13/cobra") {
			result.Frameworks = append(result.Frameworks, "Cobra CLI")
		}
	}
	
	result.Tools = append(result.Tools, "go", "gofmt", "golint")
	result.Metadata["gomod_path"] = goModPath
	
	return nil
}

// detectRustProject detects Rust project characteristics
func (m *manager) detectRustProject(projectPath string, result *pkg.ProjectDetectionResult) error {
	cargoPath := filepath.Join(projectPath, "Cargo.toml")
	if _, err := os.Stat(cargoPath); os.IsNotExist(err) {
		return fmt.Errorf("Cargo.toml not found")
	}
	
	result.Files = append(result.Files, "Cargo.toml")
	
	// Check for Cargo.lock
	if _, err := os.Stat(filepath.Join(projectPath, "Cargo.lock")); err == nil {
		result.Files = append(result.Files, "Cargo.lock")
	}
	
	// Check for src directory
	if _, err := os.Stat(filepath.Join(projectPath, "src")); err == nil {
		result.Features = append(result.Features, "source-directory")
	}
	
	// Check for main.rs vs lib.rs
	if _, err := os.Stat(filepath.Join(projectPath, "src", "main.rs")); err == nil {
		result.Features = append(result.Features, "executable")
		result.Files = append(result.Files, "src/main.rs")
	}
	
	if _, err := os.Stat(filepath.Join(projectPath, "src", "lib.rs")); err == nil {
		result.Features = append(result.Features, "library")
		result.Files = append(result.Files, "src/lib.rs")
	}
	
	result.Tools = append(result.Tools, "cargo", "rustc", "rustfmt")
	result.Metadata["cargo_path"] = cargoPath
	
	return nil
}

// detectNodeProject detects Node.js project characteristics
func (m *manager) detectNodeProject(projectPath string, result *pkg.ProjectDetectionResult) error {
	packagePath := filepath.Join(projectPath, "package.json")
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		return fmt.Errorf("package.json not found")
	}
	
	result.Files = append(result.Files, "package.json")
	
	// Parse package.json for more details
	packageData, err := os.ReadFile(packagePath)
	if err == nil {
		var pkg map[string]interface{}
		if json.Unmarshal(packageData, &pkg) == nil {
			
			// Check dependencies for frameworks
			if deps, ok := pkg["dependencies"].(map[string]interface{}); ok {
				if _, hasReact := deps["react"]; hasReact {
					result.Frameworks = append(result.Frameworks, "React")
				}
				if _, hasVue := deps["vue"]; hasVue {
					result.Frameworks = append(result.Frameworks, "Vue")
				}
				if _, hasExpress := deps["express"]; hasExpress {
					result.Frameworks = append(result.Frameworks, "Express")
				}
				if _, hasNext := deps["next"]; hasNext {
					result.Frameworks = append(result.Frameworks, "Next.js")
				}
			}
			
			// Check scripts
			if scripts, ok := pkg["scripts"].(map[string]interface{}); ok {
				for script := range scripts {
					result.Features = append(result.Features, fmt.Sprintf("script-%s", script))
				}
			}
		}
	}
	
	// Check for TypeScript
	if _, err := os.Stat(filepath.Join(projectPath, "tsconfig.json")); err == nil {
		result.Languages = append(result.Languages, "TypeScript")
		result.Files = append(result.Files, "tsconfig.json")
	}
	
	// Check for yarn.lock vs package-lock.json
	if _, err := os.Stat(filepath.Join(projectPath, "yarn.lock")); err == nil {
		result.Tools = append(result.Tools, "yarn")
		result.Files = append(result.Files, "yarn.lock")
	} else if _, err := os.Stat(filepath.Join(projectPath, "package-lock.json")); err == nil {
		result.Tools = append(result.Tools, "npm")
		result.Files = append(result.Files, "package-lock.json")
	}
	
	result.Metadata["package_path"] = packagePath
	
	return nil
}

// detectPythonProject detects Python project characteristics
func (m *manager) detectPythonProject(projectPath string, result *pkg.ProjectDetectionResult) error {
	// Check for various Python dependency files
	pythonFiles := []string{"requirements.txt", "pyproject.toml", "setup.py", "Pipfile", "poetry.lock"}
	found := false
	
	for _, file := range pythonFiles {
		if _, err := os.Stat(filepath.Join(projectPath, file)); err == nil {
			result.Files = append(result.Files, file)
			found = true
		}
	}
	
	if !found {
		return fmt.Errorf("no Python dependency files found")
	}
	
	// Check for common Python frameworks
	if _, err := os.Stat(filepath.Join(projectPath, "manage.py")); err == nil {
		result.Frameworks = append(result.Frameworks, "Django")
		result.Files = append(result.Files, "manage.py")
	}
	
	// Check requirements.txt for frameworks
	reqPath := filepath.Join(projectPath, "requirements.txt")
	if reqData, err := os.ReadFile(reqPath); err == nil {
		content := string(reqData)
		if strings.Contains(content, "flask") {
			result.Frameworks = append(result.Frameworks, "Flask")
		}
		if strings.Contains(content, "fastapi") {
			result.Frameworks = append(result.Frameworks, "FastAPI")
		}
		if strings.Contains(content, "django") {
			result.Frameworks = append(result.Frameworks, "Django")
		}
	}
	
	// Check for virtual environment indicators
	if _, err := os.Stat(filepath.Join(projectPath, "venv")); err == nil {
		result.Features = append(result.Features, "virtual-environment")
	}
	
	result.Tools = append(result.Tools, "python", "pip")
	
	return nil
}

// detectJavaProject detects Java project characteristics
func (m *manager) detectJavaProject(projectPath string, result *pkg.ProjectDetectionResult) error {
	// Check for Maven
	if _, err := os.Stat(filepath.Join(projectPath, "pom.xml")); err == nil {
		result.Files = append(result.Files, "pom.xml")
		result.Tools = append(result.Tools, "maven")
		result.Features = append(result.Features, "maven-project")
		
		// Parse pom.xml for framework detection
		if pomData, err := os.ReadFile(filepath.Join(projectPath, "pom.xml")); err == nil {
			content := string(pomData)
			if strings.Contains(content, "spring-boot") {
				result.Frameworks = append(result.Frameworks, "Spring Boot")
			}
			if strings.Contains(content, "spring-framework") {
				result.Frameworks = append(result.Frameworks, "Spring")
			}
		}
		
		return nil
	}
	
	// Check for Gradle
	gradleFiles := []string{"build.gradle", "build.gradle.kts", "gradlew"}
	for _, file := range gradleFiles {
		if _, err := os.Stat(filepath.Join(projectPath, file)); err == nil {
			result.Files = append(result.Files, file)
			result.Tools = append(result.Tools, "gradle")
			result.Features = append(result.Features, "gradle-project")
			return nil
		}
	}
	
	return fmt.Errorf("no Java build files found")
}