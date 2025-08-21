package dependency

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"claude-reactor/pkg"
)

// goManager implements PackageManager for Go modules
type goManager struct {
	logger pkg.Logger
}

// NewGoManager creates a new Go package manager
func NewGoManager(logger pkg.Logger) pkg.PackageManager {
	return &goManager{
		logger: logger,
	}
}

func (g *goManager) GetType() string {
	return "go"
}

func (g *goManager) GetName() string {
	return "Go Modules"
}

func (g *goManager) IsAvailable() bool {
	_, err := exec.LookPath("go")
	return err == nil
}

func (g *goManager) GetVersion() (string, error) {
	cmd := exec.Command("go", "version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get Go version: %w", err)
	}
	
	version := strings.TrimSpace(string(output))
	return version, nil
}

func (g *goManager) DetectConfigFiles(projectPath string) []string {
	var configFiles []string
	
	goModPath := filepath.Join(projectPath, "go.mod")
	if g.fileExists(goModPath) {
		configFiles = append(configFiles, "go.mod")
	}
	
	return configFiles
}

func (g *goManager) ListDependencies(projectPath string) ([]*pkg.DependencyInfo, error) {
	g.logger.Debugf("Listing Go dependencies in: %s", projectPath)
	
	// Use go list -json -m all to get module information
	cmd := exec.Command("go", "list", "-json", "-m", "all")
	cmd.Dir = projectPath
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list Go modules: %w", err)
	}
	
	var dependencies []*pkg.DependencyInfo
	decoder := json.NewDecoder(strings.NewReader(string(output)))
	
	for decoder.More() {
		var module struct {
			Path     string `json:"Path"`
			Version  string `json:"Version"`
			Main     bool   `json:"Main"`
			Indirect bool   `json:"Indirect"`
			Replace  *struct {
				Path    string `json:"Path"`
				Version string `json:"Version"`
			} `json:"Replace"`
		}
		
		if err := decoder.Decode(&module); err != nil {
			g.logger.Warnf("Failed to parse Go module info: %v", err)
			continue
		}
		
		// Skip the main module
		if module.Main {
			continue
		}
		
		depType := "direct"
		if module.Indirect {
			depType = "indirect"
		}
		
		name := module.Path
		version := module.Version
		
		// Handle replace directives
		if module.Replace != nil {
			name = module.Replace.Path
			if module.Replace.Version != "" {
				version = module.Replace.Version
			}
		}
		
		dep := &pkg.DependencyInfo{
			Name:            name,
			CurrentVersion:  version,
			Type:           depType,
			PackageManager: "go",
		}
		
		dependencies = append(dependencies, dep)
	}
	
	g.logger.Debugf("Found %d Go dependencies", len(dependencies))
	return dependencies, nil
}

func (g *goManager) InstallDependencies(projectPath string) (*pkg.DependencyOperationResult, error) {
	g.logger.Infof("Installing Go dependencies in: %s", projectPath)
	
	start := time.Now()
	cmd := exec.Command("go", "mod", "download")
	cmd.Dir = projectPath
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	result := &pkg.DependencyOperationResult{
		PackageManager: "Go Modules",
		Operation:      "install",
		Success:        err == nil,
		Output:         string(output),
		Duration:       duration.String(),
	}
	
	if err != nil {
		result.Error = err.Error()
		g.logger.Errorf("Failed to install Go dependencies: %v", err)
	} else {
		g.logger.Infof("Successfully installed Go dependencies")
	}
	
	return result, nil
}

func (g *goManager) UpdateDependencies(projectPath string) (*pkg.DependencyOperationResult, error) {
	g.logger.Infof("Updating Go dependencies in: %s", projectPath)
	
	start := time.Now()
	
	// First, update go.mod with latest versions
	cmd := exec.Command("go", "get", "-u", "./...")
	cmd.Dir = projectPath
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	result := &pkg.DependencyOperationResult{
		PackageManager: "Go Modules",
		Operation:      "update",
		Success:        err == nil,
		Output:         string(output),
		Duration:       duration.String(),
	}
	
	if err != nil {
		result.Error = err.Error()
		g.logger.Errorf("Failed to update Go dependencies: %v", err)
	} else {
		// Run go mod tidy to clean up
		tidyCmd := exec.Command("go", "mod", "tidy")
		tidyCmd.Dir = projectPath
		tidyOutput, tidyErr := tidyCmd.CombinedOutput()
		
		result.Output += "\n" + string(tidyOutput)
		
		if tidyErr != nil {
			result.Success = false
			result.Error = tidyErr.Error()
			g.logger.Errorf("Failed to tidy Go modules: %v", tidyErr)
		} else {
			g.logger.Infof("Successfully updated Go dependencies")
		}
	}
	
	return result, nil
}

func (g *goManager) AuditDependencies(projectPath string) ([]*pkg.VulnerabilityInfo, error) {
	g.logger.Debugf("Auditing Go dependencies in: %s", projectPath)
	
	// Go doesn't have built-in vulnerability scanning like npm audit
	// We could integrate with tools like govulncheck, but for now return empty
	// This would be enhanced in a full implementation
	
	// Check if govulncheck is available
	if _, err := exec.LookPath("govulncheck"); err == nil {
		return g.runGovulncheck(projectPath)
	}
	
	g.logger.Debugf("No Go vulnerability scanner available (govulncheck not found)")
	return []*pkg.VulnerabilityInfo{}, nil
}

func (g *goManager) runGovulncheck(projectPath string) ([]*pkg.VulnerabilityInfo, error) {
	cmd := exec.Command("govulncheck", "./...")
	cmd.Dir = projectPath
	
	output, err := cmd.Output()
	if err != nil {
		// govulncheck returns non-zero exit code when vulnerabilities are found
		if exitErr, ok := err.(*exec.ExitError); ok {
			output = exitErr.Stderr
		}
	}
	
	// Parse govulncheck output (simplified implementation)
	var vulnerabilities []*pkg.VulnerabilityInfo
	
	outputStr := string(output)
	if strings.Contains(outputStr, "Vulnerability") || strings.Contains(outputStr, "vulnerability") {
		// This is a simplified parser - a full implementation would parse JSON output
		vuln := &pkg.VulnerabilityInfo{
			ID:          "GO-VULN-DETECTED",
			Title:       "Go Vulnerability Detected",
			Description: "One or more vulnerabilities found by govulncheck",
			Severity:    "moderate",
			Reference:   "Run 'govulncheck ./...' for details",
		}
		vulnerabilities = append(vulnerabilities, vuln)
	}
	
	return vulnerabilities, nil
}

func (g *goManager) GetOutdatedDependencies(projectPath string) ([]*pkg.DependencyInfo, error) {
	g.logger.Debugf("Checking outdated Go dependencies in: %s", projectPath)
	
	// Use go list -u -m all to check for updates
	cmd := exec.Command("go", "list", "-u", "-json", "-m", "all")
	cmd.Dir = projectPath
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to check Go module updates: %w", err)
	}
	
	var outdated []*pkg.DependencyInfo
	decoder := json.NewDecoder(strings.NewReader(string(output)))
	
	for decoder.More() {
		var module struct {
			Path     string `json:"Path"`
			Version  string `json:"Version"`
			Update   *struct {
				Path    string `json:"Path"`
				Version string `json:"Version"`
			} `json:"Update"`
			Main     bool `json:"Main"`
			Indirect bool `json:"Indirect"`
		}
		
		if err := decoder.Decode(&module); err != nil {
			g.logger.Warnf("Failed to parse Go module update info: %v", err)
			continue
		}
		
		// Skip main module and modules without updates
		if module.Main || module.Update == nil {
			continue
		}
		
		depType := "direct"
		if module.Indirect {
			depType = "indirect"
		}
		
		dep := &pkg.DependencyInfo{
			Name:            module.Path,
			CurrentVersion:  module.Version,
			LatestVersion:   module.Update.Version,
			Type:           depType,
			PackageManager: "go",
			IsOutdated:     true,
		}
		
		outdated = append(outdated, dep)
	}
	
	g.logger.Debugf("Found %d outdated Go dependencies", len(outdated))
	return outdated, nil
}

func (g *goManager) CleanCache(projectPath string) (*pkg.DependencyOperationResult, error) {
	g.logger.Infof("Cleaning Go module cache")
	
	start := time.Now()
	cmd := exec.Command("go", "clean", "-modcache")
	cmd.Dir = projectPath
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	result := &pkg.DependencyOperationResult{
		PackageManager: "Go Modules",
		Operation:      "clean",
		Success:        err == nil,
		Output:         string(output),
		Duration:       duration.String(),
	}
	
	if err != nil {
		result.Error = err.Error()
		g.logger.Errorf("Failed to clean Go module cache: %v", err)
	} else {
		g.logger.Infof("Successfully cleaned Go module cache")
	}
	
	return result, nil
}

// Helper function
func (g *goManager) fileExists(path string) bool {
	manager := &manager{logger: g.logger}
	return manager.fileExists(path)
}