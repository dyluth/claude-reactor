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

// pipManager implements PackageManager for pip
type pipManager struct {
	logger pkg.Logger
}

// NewPipManager creates a new pip package manager
func NewPipManager(logger pkg.Logger) pkg.PackageManager {
	return &pipManager{
		logger: logger,
	}
}

func (p *pipManager) GetType() string {
	return "pip"
}

func (p *pipManager) GetName() string {
	return "pip"
}

func (p *pipManager) IsAvailable() bool {
	_, err := exec.LookPath("pip")
	return err == nil
}

func (p *pipManager) GetVersion() (string, error) {
	cmd := exec.Command("pip", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get pip version: %w", err)
	}
	
	version := strings.TrimSpace(string(output))
	return version, nil
}

func (p *pipManager) DetectConfigFiles(projectPath string) []string {
	var configFiles []string
	
	files := []string{"requirements.txt", "requirements.in", "setup.py", "pyproject.toml"}
	for _, file := range files {
		if p.fileExists(filepath.Join(projectPath, file)) {
			configFiles = append(configFiles, file)
		}
	}
	
	return configFiles
}

func (p *pipManager) ListDependencies(projectPath string) ([]*pkg.DependencyInfo, error) {
	p.logger.Debugf("Listing pip dependencies in: %s", projectPath)
	
	// Use pip list --format=json to get installed packages
	cmd := exec.Command("pip", "list", "--format=json")
	cmd.Dir = projectPath
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list pip packages: %w", err)
	}
	
	var packages []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	
	if err := json.Unmarshal(output, &packages); err != nil {
		return nil, fmt.Errorf("failed to parse pip list output: %w", err)
	}
	
	var dependencies []*pkg.DependencyInfo
	
	// Read requirements.txt to determine which are direct dependencies
	directDeps := p.getDirectDependencies(projectPath)
	
	for _, pkgInfo := range packages {
		// Skip standard library packages
		if p.isStandardLibrary(pkgInfo.Name) {
			continue
		}
		
		depType := "indirect"
		if directDeps[strings.ToLower(pkgInfo.Name)] {
			depType = "direct"
		}
		
		dep := &pkg.DependencyInfo{
			Name:            pkgInfo.Name,
			CurrentVersion:  pkgInfo.Version,
			Type:           depType,
			PackageManager: "pip",
		}
		
		dependencies = append(dependencies, dep)
	}
	
	p.logger.Debugf("Found %d pip dependencies", len(dependencies))
	return dependencies, nil
}

func (p *pipManager) getDirectDependencies(projectPath string) map[string]bool {
	directDeps := make(map[string]bool)
	
	// Check requirements.txt
	reqPath := filepath.Join(projectPath, "requirements.txt")
	if p.fileExists(reqPath) {
		// This is simplified - a full implementation would parse requirements.txt properly
		// For now, we'll assume all requirements are direct dependencies
		cmd := exec.Command("pip", "freeze", "--local")
		cmd.Dir = projectPath
		if output, err := cmd.Output(); err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" && !strings.HasPrefix(line, "#") {
					if idx := strings.Index(line, "=="); idx > 0 {
						pkg := strings.ToLower(strings.TrimSpace(line[:idx]))
						directDeps[pkg] = true
					}
				}
			}
		}
	}
	
	return directDeps
}

func (p *pipManager) isStandardLibrary(packageName string) bool {
	// List of common standard library packages to exclude
	stdLib := []string{"pip", "setuptools", "wheel", "distlib", "pkg-resources"}
	
	packageLower := strings.ToLower(packageName)
	for _, std := range stdLib {
		if packageLower == std {
			return true
		}
	}
	
	return false
}

func (p *pipManager) InstallDependencies(projectPath string) (*pkg.DependencyOperationResult, error) {
	p.logger.Infof("Installing pip dependencies in: %s", projectPath)
	
	start := time.Now()
	
	// Try requirements.txt first
	reqPath := filepath.Join(projectPath, "requirements.txt")
	var cmd *exec.Cmd
	
	if p.fileExists(reqPath) {
		cmd = exec.Command("pip", "install", "-r", "requirements.txt")
	} else {
		cmd = exec.Command("pip", "install", ".")
	}
	
	cmd.Dir = projectPath
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	result := &pkg.DependencyOperationResult{
		PackageManager: "pip",
		Operation:      "install",
		Success:        err == nil,
		Output:         string(output),
		Duration:       duration.String(),
	}
	
	if err != nil {
		result.Error = err.Error()
		p.logger.Errorf("Failed to install pip dependencies: %v", err)
	} else {
		p.logger.Infof("Successfully installed pip dependencies")
	}
	
	return result, nil
}

func (p *pipManager) UpdateDependencies(projectPath string) (*pkg.DependencyOperationResult, error) {
	p.logger.Infof("Updating pip dependencies in: %s", projectPath)
	
	start := time.Now()
	
	// pip doesn't have a direct "update all" command, so we'll upgrade pip first
	cmd := exec.Command("pip", "install", "--upgrade", "pip")
	cmd.Dir = projectPath
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	result := &pkg.DependencyOperationResult{
		PackageManager: "pip",
		Operation:      "update",
		Success:        err == nil,
		Output:         string(output),
		Duration:       duration.String(),
	}
	
	if err != nil {
		result.Error = err.Error()
		p.logger.Errorf("Failed to update pip: %v", err)
	} else {
		p.logger.Infof("Successfully updated pip")
	}
	
	return result, nil
}

func (p *pipManager) AuditDependencies(projectPath string) ([]*pkg.VulnerabilityInfo, error) {
	p.logger.Debugf("Auditing pip dependencies in: %s", projectPath)
	
	// Check if pip-audit is available
	if _, err := exec.LookPath("pip-audit"); err != nil {
		p.logger.Debugf("pip-audit not available, skipping vulnerability check")
		return []*pkg.VulnerabilityInfo{}, nil
	}
	
	cmd := exec.Command("pip-audit", "--format", "json")
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
	
	var auditResult []struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		ID          string `json:"id"`
		Description string `json:"description"`
		Severity    string `json:"severity"`
		FixVersions []string `json:"fix_versions"`
	}
	
	if err := json.Unmarshal(output, &auditResult); err != nil {
		p.logger.Warnf("Failed to parse pip-audit output: %v", err)
		return []*pkg.VulnerabilityInfo{}, nil
	}
	
	var vulnerabilities []*pkg.VulnerabilityInfo
	
	for _, vuln := range auditResult {
		fixedIn := ""
		if len(vuln.FixVersions) > 0 {
			fixedIn = vuln.FixVersions[0]
		}
		
		vulnInfo := &pkg.VulnerabilityInfo{
			ID:          vuln.ID,
			Title:       fmt.Sprintf("Vulnerability in %s", vuln.Name),
			Description: vuln.Description,
			Severity:    strings.ToLower(vuln.Severity),
			FixedIn:     fixedIn,
		}
		
		vulnerabilities = append(vulnerabilities, vulnInfo)
	}
	
	p.logger.Debugf("Found %d pip vulnerabilities", len(vulnerabilities))
	return vulnerabilities, nil
}

func (p *pipManager) GetOutdatedDependencies(projectPath string) ([]*pkg.DependencyInfo, error) {
	p.logger.Debugf("Checking outdated pip dependencies in: %s", projectPath)
	
	cmd := exec.Command("pip", "list", "--outdated", "--format=json")
	cmd.Dir = projectPath
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to check outdated pip packages: %w", err)
	}
	
	var outdatedPackages []struct {
		Name           string `json:"name"`
		Version        string `json:"version"`
		LatestVersion  string `json:"latest_version"`
		LatestFiletype string `json:"latest_filetype"`
	}
	
	if err := json.Unmarshal(output, &outdatedPackages); err != nil {
		return nil, fmt.Errorf("failed to parse pip outdated output: %w", err)
	}
	
	var outdated []*pkg.DependencyInfo
	
	for _, pkgInfo := range outdatedPackages {
		if p.isStandardLibrary(pkgInfo.Name) {
			continue
		}
		
		dep := &pkg.DependencyInfo{
			Name:            pkgInfo.Name,
			CurrentVersion:  pkgInfo.Version,
			LatestVersion:   pkgInfo.LatestVersion,
			Type:           "direct", // Simplified
			PackageManager: "pip",
			IsOutdated:     true,
		}
		
		outdated = append(outdated, dep)
	}
	
	p.logger.Debugf("Found %d outdated pip dependencies", len(outdated))
	return outdated, nil
}

func (p *pipManager) CleanCache(projectPath string) (*pkg.DependencyOperationResult, error) {
	p.logger.Infof("Cleaning pip cache")
	
	start := time.Now()
	cmd := exec.Command("pip", "cache", "purge")
	cmd.Dir = projectPath
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	result := &pkg.DependencyOperationResult{
		PackageManager: "pip",
		Operation:      "clean",
		Success:        err == nil,
		Output:         string(output),
		Duration:       duration.String(),
	}
	
	if err != nil {
		result.Error = err.Error()
		p.logger.Errorf("Failed to clean pip cache: %v", err)
	} else {
		p.logger.Infof("Successfully cleaned pip cache")
	}
	
	return result, nil
}

// Helper function
func (p *pipManager) fileExists(path string) bool {
	manager := &manager{logger: p.logger}
	return manager.fileExists(path)
}

// poetryManager implements PackageManager for Poetry
type poetryManager struct {
	logger pkg.Logger
}

// NewPoetryManager creates a new Poetry package manager
func NewPoetryManager(logger pkg.Logger) pkg.PackageManager {
	return &poetryManager{
		logger: logger,
	}
}

func (p *poetryManager) GetType() string {
	return "poetry"
}

func (p *poetryManager) GetName() string {
	return "Poetry"
}

func (p *poetryManager) IsAvailable() bool {
	_, err := exec.LookPath("poetry")
	return err == nil
}

func (p *poetryManager) GetVersion() (string, error) {
	cmd := exec.Command("poetry", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get Poetry version: %w", err)
	}
	
	version := strings.TrimSpace(string(output))
	return version, nil
}

func (p *poetryManager) DetectConfigFiles(projectPath string) []string {
	var configFiles []string
	
	pyprojectPath := filepath.Join(projectPath, "pyproject.toml")
	if p.fileExists(pyprojectPath) {
		configFiles = append(configFiles, "pyproject.toml")
	}
	
	return configFiles
}

func (p *poetryManager) ListDependencies(projectPath string) ([]*pkg.DependencyInfo, error) {
	p.logger.Debugf("Listing Poetry dependencies in: %s", projectPath)
	
	cmd := exec.Command("poetry", "show", "--format=json")
	cmd.Dir = projectPath
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list Poetry dependencies: %w", err)
	}
	
	var packages []struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Category    string `json:"category"`
		Description string `json:"description"`
	}
	
	if err := json.Unmarshal(output, &packages); err != nil {
		return nil, fmt.Errorf("failed to parse Poetry show output: %w", err)
	}
	
	var dependencies []*pkg.DependencyInfo
	
	for _, pkgInfo := range packages {
		depType := "direct"
		if pkgInfo.Category == "dev" {
			depType = "dev"
		}
		
		dep := &pkg.DependencyInfo{
			Name:            pkgInfo.Name,
			CurrentVersion:  pkgInfo.Version,
			Description:     pkgInfo.Description,
			Type:           depType,
			PackageManager: "poetry",
		}
		
		dependencies = append(dependencies, dep)
	}
	
	p.logger.Debugf("Found %d Poetry dependencies", len(dependencies))
	return dependencies, nil
}

func (p *poetryManager) InstallDependencies(projectPath string) (*pkg.DependencyOperationResult, error) {
	p.logger.Infof("Installing Poetry dependencies in: %s", projectPath)
	
	start := time.Now()
	cmd := exec.Command("poetry", "install")
	cmd.Dir = projectPath
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	result := &pkg.DependencyOperationResult{
		PackageManager: "Poetry",
		Operation:      "install",
		Success:        err == nil,
		Output:         string(output),
		Duration:       duration.String(),
	}
	
	if err != nil {
		result.Error = err.Error()
		p.logger.Errorf("Failed to install Poetry dependencies: %v", err)
	} else {
		p.logger.Infof("Successfully installed Poetry dependencies")
	}
	
	return result, nil
}

func (p *poetryManager) UpdateDependencies(projectPath string) (*pkg.DependencyOperationResult, error) {
	p.logger.Infof("Updating Poetry dependencies in: %s", projectPath)
	
	start := time.Now()
	cmd := exec.Command("poetry", "update")
	cmd.Dir = projectPath
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	result := &pkg.DependencyOperationResult{
		PackageManager: "Poetry",
		Operation:      "update",
		Success:        err == nil,
		Output:         string(output),
		Duration:       duration.String(),
	}
	
	if err != nil {
		result.Error = err.Error()
		p.logger.Errorf("Failed to update Poetry dependencies: %v", err)
	} else {
		p.logger.Infof("Successfully updated Poetry dependencies")
	}
	
	return result, nil
}

func (p *poetryManager) AuditDependencies(projectPath string) ([]*pkg.VulnerabilityInfo, error) {
	// Poetry doesn't have built-in vulnerability scanning
	// Could integrate with safety or pip-audit
	p.logger.Debugf("Poetry vulnerability scanning not implemented")
	return []*pkg.VulnerabilityInfo{}, nil
}

func (p *poetryManager) GetOutdatedDependencies(projectPath string) ([]*pkg.DependencyInfo, error) {
	p.logger.Debugf("Checking outdated Poetry dependencies in: %s", projectPath)
	
	cmd := exec.Command("poetry", "show", "--outdated", "--format=json")
	cmd.Dir = projectPath
	
	_, err := cmd.Output()
	if err != nil {
		// Poetry might not support --format=json with --outdated
		return []*pkg.DependencyInfo{}, nil
	}
	
	// This would need proper JSON parsing for Poetry's outdated format
	p.logger.Debugf("Poetry outdated parsing not fully implemented")
	return []*pkg.DependencyInfo{}, nil
}

func (p *poetryManager) CleanCache(projectPath string) (*pkg.DependencyOperationResult, error) {
	p.logger.Infof("Cleaning Poetry cache")
	
	start := time.Now()
	cmd := exec.Command("poetry", "cache", "clear", "--all", ".")
	cmd.Dir = projectPath
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	result := &pkg.DependencyOperationResult{
		PackageManager: "Poetry",
		Operation:      "clean",
		Success:        err == nil,
		Output:         string(output),
		Duration:       duration.String(),
	}
	
	if err != nil {
		result.Error = err.Error()
		p.logger.Errorf("Failed to clean Poetry cache: %v", err)
	} else {
		p.logger.Infof("Successfully cleaned Poetry cache")
	}
	
	return result, nil
}

func (p *poetryManager) fileExists(path string) bool {
	manager := &manager{logger: p.logger}
	return manager.fileExists(path)
}

// pipenvManager implements PackageManager for Pipenv
type pipenvManager struct {
	logger pkg.Logger
}

// NewPipenvManager creates a new Pipenv package manager
func NewPipenvManager(logger pkg.Logger) pkg.PackageManager {
	return &pipenvManager{
		logger: logger,
	}
}

func (p *pipenvManager) GetType() string {
	return "pipenv"
}

func (p *pipenvManager) GetName() string {
	return "Pipenv"
}

func (p *pipenvManager) IsAvailable() bool {
	_, err := exec.LookPath("pipenv")
	return err == nil
}

func (p *pipenvManager) GetVersion() (string, error) {
	cmd := exec.Command("pipenv", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get Pipenv version: %w", err)
	}
	
	version := strings.TrimSpace(string(output))
	return version, nil
}

func (p *pipenvManager) DetectConfigFiles(projectPath string) []string {
	var configFiles []string
	
	pipfilePath := filepath.Join(projectPath, "Pipfile")
	if p.fileExists(pipfilePath) {
		configFiles = append(configFiles, "Pipfile")
	}
	
	return configFiles
}

// Simplified implementations for Pipenv (similar patterns to pip/poetry)
func (p *pipenvManager) ListDependencies(projectPath string) ([]*pkg.DependencyInfo, error) {
	// Use pipenv graph or list command
	p.logger.Debugf("Pipenv dependency listing not fully implemented")
	return []*pkg.DependencyInfo{}, nil
}

func (p *pipenvManager) InstallDependencies(projectPath string) (*pkg.DependencyOperationResult, error) {
	start := time.Now()
	cmd := exec.Command("pipenv", "install")
	cmd.Dir = projectPath
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	return &pkg.DependencyOperationResult{
		PackageManager: "Pipenv",
		Operation:      "install",
		Success:        err == nil,
		Output:         string(output),
		Error:          func() string { if err != nil { return err.Error() }; return "" }(),
		Duration:       duration.String(),
	}, nil
}

func (p *pipenvManager) UpdateDependencies(projectPath string) (*pkg.DependencyOperationResult, error) {
	start := time.Now()
	cmd := exec.Command("pipenv", "update")
	cmd.Dir = projectPath
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	return &pkg.DependencyOperationResult{
		PackageManager: "Pipenv",
		Operation:      "update",
		Success:        err == nil,
		Output:         string(output),
		Error:          func() string { if err != nil { return err.Error() }; return "" }(),
		Duration:       duration.String(),
	}, nil
}

func (p *pipenvManager) AuditDependencies(projectPath string) ([]*pkg.VulnerabilityInfo, error) {
	return []*pkg.VulnerabilityInfo{}, nil
}

func (p *pipenvManager) GetOutdatedDependencies(projectPath string) ([]*pkg.DependencyInfo, error) {
	return []*pkg.DependencyInfo{}, nil
}

func (p *pipenvManager) CleanCache(projectPath string) (*pkg.DependencyOperationResult, error) {
	start := time.Now()
	cmd := exec.Command("pipenv", "clean")
	cmd.Dir = projectPath
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	return &pkg.DependencyOperationResult{
		PackageManager: "Pipenv",
		Operation:      "clean",
		Success:        err == nil,
		Output:         string(output),
		Error:          func() string { if err != nil { return err.Error() }; return "" }(),
		Duration:       duration.String(),
	}, nil
}

func (p *pipenvManager) fileExists(path string) bool {
	manager := &manager{logger: p.logger}
	return manager.fileExists(path)
}