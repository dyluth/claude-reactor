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

// cargoManager implements PackageManager for Cargo (Rust)
type cargoManager struct {
	logger pkg.Logger
}

// NewCargoManager creates a new Cargo package manager
func NewCargoManager(logger pkg.Logger) pkg.PackageManager {
	return &cargoManager{
		logger: logger,
	}
}

func (c *cargoManager) GetType() string {
	return "cargo"
}

func (c *cargoManager) GetName() string {
	return "Cargo"
}

func (c *cargoManager) IsAvailable() bool {
	_, err := exec.LookPath("cargo")
	return err == nil
}

func (c *cargoManager) GetVersion() (string, error) {
	cmd := exec.Command("cargo", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get Cargo version: %w", err)
	}
	
	version := strings.TrimSpace(string(output))
	return version, nil
}

func (c *cargoManager) DetectConfigFiles(projectPath string) []string {
	var configFiles []string
	
	cargoTomlPath := filepath.Join(projectPath, "Cargo.toml")
	if c.fileExists(cargoTomlPath) {
		configFiles = append(configFiles, "Cargo.toml")
	}
	
	return configFiles
}

func (c *cargoManager) ListDependencies(projectPath string) ([]*pkg.DependencyInfo, error) {
	c.logger.Debugf("Listing Cargo dependencies in: %s", projectPath)
	
	// Use cargo metadata to get dependency information
	cmd := exec.Command("cargo", "metadata", "--format-version", "1")
	cmd.Dir = projectPath
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get Cargo metadata: %w", err)
	}
	
	var metadata struct {
		Packages []struct {
			Name         string `json:"name"`
			Version      string `json:"version"`
			Dependencies []struct {
				Name     string `json:"name"`
				Req      string `json:"req"`
				Kind     interface{} `json:"kind"`
				Optional bool   `json:"optional"`
			} `json:"dependencies"`
		} `json:"packages"`
		Resolve struct {
			Nodes []struct {
				Id           string   `json:"id"`
				Dependencies []string `json:"dependencies"`
			} `json:"nodes"`
		} `json:"resolve"`
		WorkspaceMembers []string `json:"workspace_members"`
	}
	
	if err := json.Unmarshal(output, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse Cargo metadata: %w", err)
	}
	
	var dependencies []*pkg.DependencyInfo
	
	// Create a map to track workspace packages
	workspacePackages := make(map[string]bool)
	for _, member := range metadata.WorkspaceMembers {
		workspacePackages[member] = true
	}
	
	// Process all packages that are not workspace members
	for _, pkgInfo := range metadata.Packages {
		pkgId := fmt.Sprintf("%s %s", pkgInfo.Name, pkgInfo.Version)
		if workspacePackages[pkgId] {
			continue // Skip workspace packages
		}
		
		// Determine dependency type based on usage
		depType := "indirect"
		// This is simplified - a full implementation would analyze the dependency graph
		depType = "direct" // For now, assume direct
		
		dep := &pkg.DependencyInfo{
			Name:            pkgInfo.Name,
			CurrentVersion:  pkgInfo.Version,
			Type:           depType,
			PackageManager: "cargo",
		}
		
		dependencies = append(dependencies, dep)
	}
	
	c.logger.Debugf("Found %d Cargo dependencies", len(dependencies))
	return dependencies, nil
}

func (c *cargoManager) InstallDependencies(projectPath string) (*pkg.DependencyOperationResult, error) {
	c.logger.Infof("Installing Cargo dependencies in: %s", projectPath)
	
	start := time.Now()
	cmd := exec.Command("cargo", "fetch")
	cmd.Dir = projectPath
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	result := &pkg.DependencyOperationResult{
		PackageManager: "Cargo",
		Operation:      "install",
		Success:        err == nil,
		Output:         string(output),
		Duration:       duration.String(),
	}
	
	if err != nil {
		result.Error = err.Error()
		c.logger.Errorf("Failed to install Cargo dependencies: %v", err)
	} else {
		c.logger.Infof("Successfully installed Cargo dependencies")
	}
	
	return result, nil
}

func (c *cargoManager) UpdateDependencies(projectPath string) (*pkg.DependencyOperationResult, error) {
	c.logger.Infof("Updating Cargo dependencies in: %s", projectPath)
	
	start := time.Now()
	cmd := exec.Command("cargo", "update")
	cmd.Dir = projectPath
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	result := &pkg.DependencyOperationResult{
		PackageManager: "Cargo",
		Operation:      "update",
		Success:        err == nil,
		Output:         string(output),
		Duration:       duration.String(),
	}
	
	if err != nil {
		result.Error = err.Error()
		c.logger.Errorf("Failed to update Cargo dependencies: %v", err)
	} else {
		c.logger.Infof("Successfully updated Cargo dependencies")
	}
	
	return result, nil
}

func (c *cargoManager) AuditDependencies(projectPath string) ([]*pkg.VulnerabilityInfo, error) {
	c.logger.Debugf("Auditing Cargo dependencies in: %s", projectPath)
	
	// Check if cargo-audit is available
	if _, err := exec.LookPath("cargo-audit"); err != nil {
		c.logger.Debugf("cargo-audit not available, skipping vulnerability check")
		return []*pkg.VulnerabilityInfo{}, nil
	}
	
	cmd := exec.Command("cargo", "audit", "--json")
	cmd.Dir = projectPath
	
	output, err := cmd.Output()
	if err != nil {
		// cargo audit returns non-zero when vulnerabilities are found
		if exitErr, ok := err.(*exec.ExitError); ok {
			output = exitErr.Stderr
		}
		if len(output) == 0 {
			return nil, fmt.Errorf("failed to run cargo audit: %w", err)
		}
	}
	
	var auditResult struct {
		Vulnerabilities []struct {
			Advisory struct {
				Id          string   `json:"id"`
				Title       string   `json:"title"`
				Description string   `json:"description"`
				URL         string   `json:"url"`
				Severity    string   `json:"severity"`
				Categories  []string `json:"categories"`
			} `json:"advisory"`
			Versions struct {
				Patched   []string `json:"patched"`
				Unaffected []string `json:"unaffected"`
			} `json:"versions"`
		} `json:"vulnerabilities"`
	}
	
	if err := json.Unmarshal(output, &auditResult); err != nil {
		c.logger.Warnf("Failed to parse cargo audit output: %v", err)
		return []*pkg.VulnerabilityInfo{}, nil
	}
	
	var vulnerabilities []*pkg.VulnerabilityInfo
	
	for _, vuln := range auditResult.Vulnerabilities {
		severity := strings.ToLower(vuln.Advisory.Severity)
		if severity == "" {
			severity = "moderate"
		}
		
		fixedIn := ""
		if len(vuln.Versions.Patched) > 0 {
			fixedIn = vuln.Versions.Patched[0]
		}
		
		vulnInfo := &pkg.VulnerabilityInfo{
			ID:          vuln.Advisory.Id,
			Title:       vuln.Advisory.Title,
			Description: vuln.Advisory.Description,
			Severity:    severity,
			FixedIn:     fixedIn,
			Reference:   vuln.Advisory.URL,
		}
		
		vulnerabilities = append(vulnerabilities, vulnInfo)
	}
	
	c.logger.Debugf("Found %d Cargo vulnerabilities", len(vulnerabilities))
	return vulnerabilities, nil
}

func (c *cargoManager) GetOutdatedDependencies(projectPath string) ([]*pkg.DependencyInfo, error) {
	c.logger.Debugf("Checking outdated Cargo dependencies in: %s", projectPath)
	
	// Cargo doesn't have a built-in "outdated" command like npm
	// We could use cargo-outdated plugin, but for now return empty
	// This would require installing: cargo install --locked cargo-outdated
	
	if _, err := exec.LookPath("cargo-outdated"); err != nil {
		c.logger.Debugf("cargo-outdated not available, skipping outdated check")
		return []*pkg.DependencyInfo{}, nil
	}
	
	cmd := exec.Command("cargo", "outdated", "--format", "json")
	cmd.Dir = projectPath
	
	output, err := cmd.Output()
	if err != nil {
		c.logger.Warnf("Failed to check outdated Cargo dependencies: %v", err)
		return []*pkg.DependencyInfo{}, nil
	}
	
	var outdatedResult struct {
		Dependencies []struct {
			Name     string `json:"name"`
			Project  string `json:"project"`
			Compat   string `json:"compat"`
			Latest   string `json:"latest"`
			Kind     string `json:"kind"`
		} `json:"dependencies"`
	}
	
	if err := json.Unmarshal(output, &outdatedResult); err != nil {
		c.logger.Warnf("Failed to parse cargo-outdated output: %v", err)
		return []*pkg.DependencyInfo{}, nil
	}
	
	var outdated []*pkg.DependencyInfo
	
	for _, dep := range outdatedResult.Dependencies {
		depType := "direct"
		if dep.Kind == "dev" {
			depType = "dev"
		}
		
		depInfo := &pkg.DependencyInfo{
			Name:            dep.Name,
			CurrentVersion:  dep.Project,
			LatestVersion:   dep.Latest,
			RequestedVersion: dep.Compat,
			Type:           depType,
			PackageManager: "cargo",
			IsOutdated:     true,
		}
		
		outdated = append(outdated, depInfo)
	}
	
	c.logger.Debugf("Found %d outdated Cargo dependencies", len(outdated))
	return outdated, nil
}

func (c *cargoManager) CleanCache(projectPath string) (*pkg.DependencyOperationResult, error) {
	c.logger.Infof("Cleaning Cargo cache")
	
	start := time.Now()
	cmd := exec.Command("cargo", "clean")
	cmd.Dir = projectPath
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	result := &pkg.DependencyOperationResult{
		PackageManager: "Cargo",
		Operation:      "clean",
		Success:        err == nil,
		Output:         string(output),
		Duration:       duration.String(),
	}
	
	if err != nil {
		result.Error = err.Error()
		c.logger.Errorf("Failed to clean Cargo cache: %v", err)
	} else {
		c.logger.Infof("Successfully cleaned Cargo cache")
	}
	
	return result, nil
}

// Helper function
func (c *cargoManager) fileExists(path string) bool {
	manager := &manager{logger: c.logger}
	return manager.fileExists(path)
}