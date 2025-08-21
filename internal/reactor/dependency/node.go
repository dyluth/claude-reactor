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

// yarnManager implements PackageManager for Yarn
type yarnManager struct {
	logger pkg.Logger
}

// NewYarnManager creates a new Yarn package manager
func NewYarnManager(logger pkg.Logger) pkg.PackageManager {
	return &yarnManager{
		logger: logger,
	}
}

func (y *yarnManager) GetType() string {
	return "yarn"
}

func (y *yarnManager) GetName() string {
	return "Yarn"
}

func (y *yarnManager) IsAvailable() bool {
	_, err := exec.LookPath("yarn")
	return err == nil
}

func (y *yarnManager) GetVersion() (string, error) {
	cmd := exec.Command("yarn", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get Yarn version: %w", err)
	}
	
	version := strings.TrimSpace(string(output))
	return version, nil
}

func (y *yarnManager) DetectConfigFiles(projectPath string) []string {
	var configFiles []string
	
	packageJsonPath := filepath.Join(projectPath, "package.json")
	if y.fileExists(packageJsonPath) {
		configFiles = append(configFiles, "package.json")
	}
	
	return configFiles
}

func (y *yarnManager) ListDependencies(projectPath string) ([]*pkg.DependencyInfo, error) {
	y.logger.Debugf("Listing Yarn dependencies in: %s", projectPath)
	
	// Use yarn list --json to get dependency information
	cmd := exec.Command("yarn", "list", "--json", "--depth=0")
	cmd.Dir = projectPath
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list Yarn dependencies: %w", err)
	}
	
	// Yarn output is JSON Lines format
	lines := strings.Split(string(output), "\n")
	var dependencies []*pkg.DependencyInfo
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		var entry struct {
			Type string `json:"type"`
			Data struct {
				Trees []struct {
					Name     string `json:"name"`
					Children []interface{} `json:"children"`
				} `json:"trees"`
			} `json:"data"`
		}
		
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		
		if entry.Type == "tree" {
			for _, tree := range entry.Data.Trees {
				// Parse name@version format
				parts := strings.Split(tree.Name, "@")
				if len(parts) >= 2 {
					name := strings.Join(parts[:len(parts)-1], "@")
					version := parts[len(parts)-1]
					
					depType := "direct"
					if len(tree.Children) > 0 {
						depType = "indirect"
					}
					
					dep := &pkg.DependencyInfo{
						Name:            name,
						CurrentVersion:  version,
						Type:           depType,
						PackageManager: "yarn",
					}
					
					dependencies = append(dependencies, dep)
				}
			}
		}
	}
	
	y.logger.Debugf("Found %d Yarn dependencies", len(dependencies))
	return dependencies, nil
}

func (y *yarnManager) InstallDependencies(projectPath string) (*pkg.DependencyOperationResult, error) {
	y.logger.Infof("Installing Yarn dependencies in: %s", projectPath)
	
	start := time.Now()
	cmd := exec.Command("yarn", "install")
	cmd.Dir = projectPath
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	result := &pkg.DependencyOperationResult{
		PackageManager: "Yarn",
		Operation:      "install",
		Success:        err == nil,
		Output:         string(output),
		Duration:       duration.String(),
	}
	
	if err != nil {
		result.Error = err.Error()
		y.logger.Errorf("Failed to install Yarn dependencies: %v", err)
	} else {
		y.logger.Infof("Successfully installed Yarn dependencies")
	}
	
	return result, nil
}

func (y *yarnManager) UpdateDependencies(projectPath string) (*pkg.DependencyOperationResult, error) {
	y.logger.Infof("Updating Yarn dependencies in: %s", projectPath)
	
	start := time.Now()
	cmd := exec.Command("yarn", "upgrade")
	cmd.Dir = projectPath
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	result := &pkg.DependencyOperationResult{
		PackageManager: "Yarn",
		Operation:      "update",
		Success:        err == nil,
		Output:         string(output),
		Duration:       duration.String(),
	}
	
	if err != nil {
		result.Error = err.Error()
		y.logger.Errorf("Failed to update Yarn dependencies: %v", err)
	} else {
		y.logger.Infof("Successfully updated Yarn dependencies")
	}
	
	return result, nil
}

func (y *yarnManager) AuditDependencies(projectPath string) ([]*pkg.VulnerabilityInfo, error) {
	y.logger.Debugf("Auditing Yarn dependencies in: %s", projectPath)
	
	cmd := exec.Command("yarn", "audit", "--json")
	cmd.Dir = projectPath
	
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			output = exitErr.Stderr
		}
		if len(output) == 0 {
			return []*pkg.VulnerabilityInfo{}, nil
		}
	}
	
	// Yarn audit output is JSON Lines
	lines := strings.Split(string(output), "\n")
	var vulnerabilities []*pkg.VulnerabilityInfo
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		var entry struct {
			Type string `json:"type"`
			Data struct {
				Advisory struct {
					ID       int    `json:"id"`
					Title    string `json:"title"`
					Severity string `json:"severity"`
					URL      string `json:"url"`
				} `json:"advisory"`
			} `json:"data"`
		}
		
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		
		if entry.Type == "auditAdvisory" {
			vuln := &pkg.VulnerabilityInfo{
				ID:          fmt.Sprintf("YARN-%d", entry.Data.Advisory.ID),
				Title:       entry.Data.Advisory.Title,
				Severity:    strings.ToLower(entry.Data.Advisory.Severity),
				Reference:   entry.Data.Advisory.URL,
			}
			
			vulnerabilities = append(vulnerabilities, vuln)
		}
	}
	
	y.logger.Debugf("Found %d Yarn vulnerabilities", len(vulnerabilities))
	return vulnerabilities, nil
}

func (y *yarnManager) GetOutdatedDependencies(projectPath string) ([]*pkg.DependencyInfo, error) {
	y.logger.Debugf("Checking outdated Yarn dependencies in: %s", projectPath)
	
	cmd := exec.Command("yarn", "outdated", "--json")
	cmd.Dir = projectPath
	
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			output = exitErr.Stderr
		}
		if len(output) == 0 {
			return []*pkg.DependencyInfo{}, nil
		}
	}
	
	// Parse Yarn outdated JSON output (simplified)
	var outdated []*pkg.DependencyInfo
	
	// Yarn outdated format varies by version, this is a simplified implementation
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}
		
		// This would need proper parsing of Yarn's outdated format
		// For now, return empty list
	}
	
	y.logger.Debugf("Found %d outdated Yarn dependencies", len(outdated))
	return outdated, nil
}

func (y *yarnManager) CleanCache(projectPath string) (*pkg.DependencyOperationResult, error) {
	y.logger.Infof("Cleaning Yarn cache")
	
	start := time.Now()
	cmd := exec.Command("yarn", "cache", "clean")
	cmd.Dir = projectPath
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	result := &pkg.DependencyOperationResult{
		PackageManager: "Yarn",
		Operation:      "clean",
		Success:        err == nil,
		Output:         string(output),
		Duration:       duration.String(),
	}
	
	if err != nil {
		result.Error = err.Error()
		y.logger.Errorf("Failed to clean Yarn cache: %v", err)
	} else {
		y.logger.Infof("Successfully cleaned Yarn cache")
	}
	
	return result, nil
}

func (y *yarnManager) fileExists(path string) bool {
	manager := &manager{logger: y.logger}
	return manager.fileExists(path)
}

// pnpmManager implements PackageManager for pnpm
type pnpmManager struct {
	logger pkg.Logger
}

// NewPnpmManager creates a new pnpm package manager
func NewPnpmManager(logger pkg.Logger) pkg.PackageManager {
	return &pnpmManager{
		logger: logger,
	}
}

func (p *pnpmManager) GetType() string {
	return "pnpm"
}

func (p *pnpmManager) GetName() string {
	return "pnpm"
}

func (p *pnpmManager) IsAvailable() bool {
	_, err := exec.LookPath("pnpm")
	return err == nil
}

func (p *pnpmManager) GetVersion() (string, error) {
	cmd := exec.Command("pnpm", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get pnpm version: %w", err)
	}
	
	version := strings.TrimSpace(string(output))
	return version, nil
}

func (p *pnpmManager) DetectConfigFiles(projectPath string) []string {
	var configFiles []string
	
	packageJsonPath := filepath.Join(projectPath, "package.json")
	if p.fileExists(packageJsonPath) {
		configFiles = append(configFiles, "package.json")
	}
	
	return configFiles
}

func (p *pnpmManager) ListDependencies(projectPath string) ([]*pkg.DependencyInfo, error) {
	p.logger.Debugf("Listing pnpm dependencies in: %s", projectPath)
	
	// Use pnpm list --json --depth=0
	cmd := exec.Command("pnpm", "list", "--json", "--depth=0")
	cmd.Dir = projectPath
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list pnpm dependencies: %w", err)
	}
	
	var listResult []struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
		DevDependencies map[string]struct {
			Version string `json:"version"`
		} `json:"devDependencies"`
	}
	
	if err := json.Unmarshal(output, &listResult); err != nil {
		return nil, fmt.Errorf("failed to parse pnpm list output: %w", err)
	}
	
	var dependencies []*pkg.DependencyInfo
	
	if len(listResult) > 0 {
		result := listResult[0]
		
		// Process regular dependencies
		for name, info := range result.Dependencies {
			dep := &pkg.DependencyInfo{
				Name:            name,
				CurrentVersion:  info.Version,
				Type:           "direct",
				PackageManager: "pnpm",
			}
			dependencies = append(dependencies, dep)
		}
		
		// Process dev dependencies
		for name, info := range result.DevDependencies {
			dep := &pkg.DependencyInfo{
				Name:            name,
				CurrentVersion:  info.Version,
				Type:           "dev",
				PackageManager: "pnpm",
			}
			dependencies = append(dependencies, dep)
		}
	}
	
	p.logger.Debugf("Found %d pnpm dependencies", len(dependencies))
	return dependencies, nil
}

func (p *pnpmManager) InstallDependencies(projectPath string) (*pkg.DependencyOperationResult, error) {
	p.logger.Infof("Installing pnpm dependencies in: %s", projectPath)
	
	start := time.Now()
	cmd := exec.Command("pnpm", "install")
	cmd.Dir = projectPath
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	result := &pkg.DependencyOperationResult{
		PackageManager: "pnpm",
		Operation:      "install",
		Success:        err == nil,
		Output:         string(output),
		Duration:       duration.String(),
	}
	
	if err != nil {
		result.Error = err.Error()
		p.logger.Errorf("Failed to install pnpm dependencies: %v", err)
	} else {
		p.logger.Infof("Successfully installed pnpm dependencies")
	}
	
	return result, nil
}

func (p *pnpmManager) UpdateDependencies(projectPath string) (*pkg.DependencyOperationResult, error) {
	p.logger.Infof("Updating pnpm dependencies in: %s", projectPath)
	
	start := time.Now()
	cmd := exec.Command("pnpm", "update")
	cmd.Dir = projectPath
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	result := &pkg.DependencyOperationResult{
		PackageManager: "pnpm",
		Operation:      "update",
		Success:        err == nil,
		Output:         string(output),
		Duration:       duration.String(),
	}
	
	if err != nil {
		result.Error = err.Error()
		p.logger.Errorf("Failed to update pnpm dependencies: %v", err)
	} else {
		p.logger.Infof("Successfully updated pnpm dependencies")
	}
	
	return result, nil
}

func (p *pnpmManager) AuditDependencies(projectPath string) ([]*pkg.VulnerabilityInfo, error) {
	p.logger.Debugf("Auditing pnpm dependencies in: %s", projectPath)
	
	cmd := exec.Command("pnpm", "audit", "--json")
	cmd.Dir = projectPath
	
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			output = exitErr.Stderr
		}
		if len(output) == 0 {
			return []*pkg.VulnerabilityInfo{}, nil
		}
	}
	
	var auditResult struct {
		Advisories map[string]struct {
			ID       string `json:"id"`
			Title    string `json:"title"`
			Severity string `json:"severity"`
			URL      string `json:"url"`
		} `json:"advisories"`
	}
	
	if err := json.Unmarshal(output, &auditResult); err != nil {
		p.logger.Warnf("Failed to parse pnpm audit output: %v", err)
		return []*pkg.VulnerabilityInfo{}, nil
	}
	
	var vulnerabilities []*pkg.VulnerabilityInfo
	
	for _, advisory := range auditResult.Advisories {
		vuln := &pkg.VulnerabilityInfo{
			ID:          advisory.ID,
			Title:       advisory.Title,
			Severity:    strings.ToLower(advisory.Severity),
			Reference:   advisory.URL,
		}
		
		vulnerabilities = append(vulnerabilities, vuln)
	}
	
	p.logger.Debugf("Found %d pnpm vulnerabilities", len(vulnerabilities))
	return vulnerabilities, nil
}

func (p *pnpmManager) GetOutdatedDependencies(projectPath string) ([]*pkg.DependencyInfo, error) {
	p.logger.Debugf("Checking outdated pnpm dependencies in: %s", projectPath)
	
	cmd := exec.Command("pnpm", "outdated", "--format=json")
	cmd.Dir = projectPath
	
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			output = exitErr.Stderr
		}
		if len(output) == 0 {
			return []*pkg.DependencyInfo{}, nil
		}
	}
	
	var outdatedMap map[string]struct {
		Current string `json:"current"`
		Latest  string `json:"latest"`
		Wanted  string `json:"wanted"`
		IsDeprecated bool `json:"isDeprecated"`
	}
	
	if err := json.Unmarshal(output, &outdatedMap); err != nil {
		p.logger.Warnf("Failed to parse pnpm outdated output: %v", err)
		return []*pkg.DependencyInfo{}, nil
	}
	
	var outdated []*pkg.DependencyInfo
	
	for name, info := range outdatedMap {
		dep := &pkg.DependencyInfo{
			Name:            name,
			CurrentVersion:  info.Current,
			LatestVersion:   info.Latest,
			RequestedVersion: info.Wanted,
			Type:           "direct",
			PackageManager: "pnpm",
			IsOutdated:     true,
		}
		
		outdated = append(outdated, dep)
	}
	
	p.logger.Debugf("Found %d outdated pnpm dependencies", len(outdated))
	return outdated, nil
}

func (p *pnpmManager) CleanCache(projectPath string) (*pkg.DependencyOperationResult, error) {
	p.logger.Infof("Cleaning pnpm cache")
	
	start := time.Now()
	cmd := exec.Command("pnpm", "store", "prune")
	cmd.Dir = projectPath
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	result := &pkg.DependencyOperationResult{
		PackageManager: "pnpm",
		Operation:      "clean",
		Success:        err == nil,
		Output:         string(output),
		Duration:       duration.String(),
	}
	
	if err != nil {
		result.Error = err.Error()
		p.logger.Errorf("Failed to clean pnpm cache: %v", err)
	} else {
		p.logger.Infof("Successfully cleaned pnpm cache")
	}
	
	return result, nil
}

func (p *pnpmManager) fileExists(path string) bool {
	manager := &manager{logger: p.logger}
	return manager.fileExists(path)
}