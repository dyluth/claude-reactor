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

// npmManager implements PackageManager for npm
type npmManager struct {
	logger pkg.Logger
}

// NewNpmManager creates a new npm package manager
func NewNpmManager(logger pkg.Logger) pkg.PackageManager {
	return &npmManager{
		logger: logger,
	}
}

func (n *npmManager) GetType() string {
	return "npm"
}

func (n *npmManager) GetName() string {
	return "npm"
}

func (n *npmManager) IsAvailable() bool {
	_, err := exec.LookPath("npm")
	return err == nil
}

func (n *npmManager) GetVersion() (string, error) {
	cmd := exec.Command("npm", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get npm version: %w", err)
	}
	
	version := strings.TrimSpace(string(output))
	return version, nil
}

func (n *npmManager) DetectConfigFiles(projectPath string) []string {
	var configFiles []string
	
	packageJsonPath := filepath.Join(projectPath, "package.json")
	if n.fileExists(packageJsonPath) {
		configFiles = append(configFiles, "package.json")
	}
	
	return configFiles
}

func (n *npmManager) ListDependencies(projectPath string) ([]*pkg.DependencyInfo, error) {
	n.logger.Debugf("Listing npm dependencies in: %s", projectPath)
	
	// Use npm list --json to get dependency information
	cmd := exec.Command("npm", "list", "--json", "--all")
	cmd.Dir = projectPath
	
	output, err := cmd.Output()
	if err != nil {
		// npm list returns non-zero exit code for missing peer dependencies, but output is still valid
		if exitErr, ok := err.(*exec.ExitError); ok {
			output = exitErr.Stderr
			if len(output) == 0 {
				return nil, fmt.Errorf("failed to list npm dependencies: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to list npm dependencies: %w", err)
		}
	}
	
	var npmList struct {
		Dependencies map[string]struct {
			Version      string                 `json:"version"`
			Dependencies map[string]interface{} `json:"dependencies"`
			DevDependencies map[string]interface{} `json:"devDependencies"`
		} `json:"dependencies"`
	}
	
	if err := json.Unmarshal(output, &npmList); err != nil {
		return nil, fmt.Errorf("failed to parse npm list output: %w", err)
	}
	
	var dependencies []*pkg.DependencyInfo
	
	// Parse dependencies recursively
	n.parseDependencies(npmList.Dependencies, "", &dependencies)
	
	n.logger.Debugf("Found %d npm dependencies", len(dependencies))
	return dependencies, nil
}

func (n *npmManager) parseDependencies(deps map[string]struct {
	Version      string                 `json:"version"`
	Dependencies map[string]interface{} `json:"dependencies"`
	DevDependencies map[string]interface{} `json:"devDependencies"`
}, parentName string, result *[]*pkg.DependencyInfo) {
	
	for name, info := range deps {
		depType := "direct"
		if parentName != "" {
			depType = "indirect"
		}
		
		dep := &pkg.DependencyInfo{
			Name:            name,
			CurrentVersion:  info.Version,
			Type:           depType,
			PackageManager: "npm",
		}
		
		*result = append(*result, dep)
		
		// Recursively parse nested dependencies
		if info.Dependencies != nil {
			// Convert interface{} map to proper structure
			nestedDeps := make(map[string]struct {
				Version      string                 `json:"version"`
				Dependencies map[string]interface{} `json:"dependencies"`
				DevDependencies map[string]interface{} `json:"devDependencies"`
			})
			
			for nestedName, nestedInfo := range info.Dependencies {
				if nestedData, ok := nestedInfo.(map[string]interface{}); ok {
					if version, hasVersion := nestedData["version"].(string); hasVersion {
						nestedDeps[nestedName] = struct {
							Version      string                 `json:"version"`
							Dependencies map[string]interface{} `json:"dependencies"`
							DevDependencies map[string]interface{} `json:"devDependencies"`
						}{
							Version:      version,
							Dependencies: nil,
							DevDependencies: nil,
						}
					}
				}
			}
			
			n.parseDependencies(nestedDeps, name, result)
		}
	}
}

func (n *npmManager) InstallDependencies(projectPath string) (*pkg.DependencyOperationResult, error) {
	n.logger.Infof("Installing npm dependencies in: %s", projectPath)
	
	start := time.Now()
	cmd := exec.Command("npm", "install")
	cmd.Dir = projectPath
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	result := &pkg.DependencyOperationResult{
		PackageManager: "npm",
		Operation:      "install",
		Success:        err == nil,
		Output:         string(output),
		Duration:       duration.String(),
	}
	
	if err != nil {
		result.Error = err.Error()
		n.logger.Errorf("Failed to install npm dependencies: %v", err)
	} else {
		n.logger.Infof("Successfully installed npm dependencies")
	}
	
	return result, nil
}

func (n *npmManager) UpdateDependencies(projectPath string) (*pkg.DependencyOperationResult, error) {
	n.logger.Infof("Updating npm dependencies in: %s", projectPath)
	
	start := time.Now()
	cmd := exec.Command("npm", "update")
	cmd.Dir = projectPath
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	result := &pkg.DependencyOperationResult{
		PackageManager: "npm",
		Operation:      "update",
		Success:        err == nil,
		Output:         string(output),
		Duration:       duration.String(),
	}
	
	if err != nil {
		result.Error = err.Error()
		n.logger.Errorf("Failed to update npm dependencies: %v", err)
	} else {
		n.logger.Infof("Successfully updated npm dependencies")
	}
	
	return result, nil
}

func (n *npmManager) AuditDependencies(projectPath string) ([]*pkg.VulnerabilityInfo, error) {
	n.logger.Debugf("Auditing npm dependencies in: %s", projectPath)
	
	cmd := exec.Command("npm", "audit", "--json")
	cmd.Dir = projectPath
	
	output, err := cmd.Output()
	if err != nil {
		// npm audit returns non-zero exit code when vulnerabilities are found
		if exitErr, ok := err.(*exec.ExitError); ok {
			output = exitErr.Stderr
		}
		if len(output) == 0 {
			return nil, fmt.Errorf("failed to run npm audit: %w", err)
		}
	}
	
	var auditResult struct {
		Vulnerabilities map[string]struct {
			Severity string `json:"severity"`
			Title    string `json:"title"`
			URL      string `json:"url"`
			Via      []interface{} `json:"via"`
		} `json:"vulnerabilities"`
	}
	
	if err := json.Unmarshal(output, &auditResult); err != nil {
		n.logger.Warnf("Failed to parse npm audit output: %v", err)
		return []*pkg.VulnerabilityInfo{}, nil
	}
	
	var vulnerabilities []*pkg.VulnerabilityInfo
	
	for pkgName, vuln := range auditResult.Vulnerabilities {
		vulnInfo := &pkg.VulnerabilityInfo{
			ID:          pkgName + "-npm-vuln",
			Title:       vuln.Title,
			Description: fmt.Sprintf("Vulnerability in %s", pkgName),
			Severity:    vuln.Severity,
			Reference:   vuln.URL,
		}
		
		vulnerabilities = append(vulnerabilities, vulnInfo)
	}
	
	n.logger.Debugf("Found %d npm vulnerabilities", len(vulnerabilities))
	return vulnerabilities, nil
}

func (n *npmManager) GetOutdatedDependencies(projectPath string) ([]*pkg.DependencyInfo, error) {
	n.logger.Debugf("Checking outdated npm dependencies in: %s", projectPath)
	
	cmd := exec.Command("npm", "outdated", "--json")
	cmd.Dir = projectPath
	
	output, err := cmd.Output()
	if err != nil {
		// npm outdated returns non-zero when outdated packages exist
		if exitErr, ok := err.(*exec.ExitError); ok {
			output = exitErr.Stderr
		}
		if len(output) == 0 {
			return []*pkg.DependencyInfo{}, nil // No outdated packages
		}
	}
	
	var outdatedMap map[string]struct {
		Current string `json:"current"`
		Wanted  string `json:"wanted"`
		Latest  string `json:"latest"`
		Type    string `json:"type"`
	}
	
	if err := json.Unmarshal(output, &outdatedMap); err != nil {
		n.logger.Warnf("Failed to parse npm outdated output: %v", err)
		return []*pkg.DependencyInfo{}, nil
	}
	
	var outdated []*pkg.DependencyInfo
	
	for name, info := range outdatedMap {
		depType := "direct"
		if info.Type == "devDependencies" {
			depType = "dev"
		}
		
		dep := &pkg.DependencyInfo{
			Name:            name,
			CurrentVersion:  info.Current,
			LatestVersion:   info.Latest,
			RequestedVersion: info.Wanted,
			Type:           depType,
			PackageManager: "npm",
			IsOutdated:     true,
		}
		
		outdated = append(outdated, dep)
	}
	
	n.logger.Debugf("Found %d outdated npm dependencies", len(outdated))
	return outdated, nil
}

func (n *npmManager) CleanCache(projectPath string) (*pkg.DependencyOperationResult, error) {
	n.logger.Infof("Cleaning npm cache")
	
	start := time.Now()
	cmd := exec.Command("npm", "cache", "clean", "--force")
	cmd.Dir = projectPath
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	result := &pkg.DependencyOperationResult{
		PackageManager: "npm",
		Operation:      "clean",
		Success:        err == nil,
		Output:         string(output),
		Duration:       duration.String(),
	}
	
	if err != nil {
		result.Error = err.Error()
		n.logger.Errorf("Failed to clean npm cache: %v", err)
	} else {
		n.logger.Infof("Successfully cleaned npm cache")
	}
	
	return result, nil
}

// Helper function
func (n *npmManager) fileExists(path string) bool {
	manager := &manager{logger: n.logger}
	return manager.fileExists(path)
}