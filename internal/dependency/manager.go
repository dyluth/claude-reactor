package dependency

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"claude-reactor/pkg"
)

// manager implements the UnifiedDependencyManager interface
type manager struct {
	logger    pkg.Logger
	detector  pkg.PackageManagerDetector
	managers  map[string]pkg.PackageManager
}

// NewManager creates a new unified dependency manager
func NewManager(logger pkg.Logger) pkg.UnifiedDependencyManager {
	detector := NewDetector(logger)
	
	// Initialize package managers
	managers := make(map[string]pkg.PackageManager)
	managers["go"] = NewGoManager(logger)
	managers["cargo"] = NewCargoManager(logger)
	managers["npm"] = NewNpmManager(logger)
	managers["yarn"] = NewYarnManager(logger)
	managers["pnpm"] = NewPnpmManager(logger)
	managers["pip"] = NewPipManager(logger)
	managers["poetry"] = NewPoetryManager(logger)
	managers["pipenv"] = NewPipenvManager(logger)
	managers["maven"] = NewMavenManager(logger)
	managers["gradle"] = NewGradleManager(logger)
	
	return &manager{
		logger:   logger,
		detector: detector,
		managers: managers,
	}
}

// DetectProjectDependencies analyzes project and detects all package managers and dependencies
func (m *manager) DetectProjectDependencies(projectPath string) ([]*pkg.PackageManagerInfo, []*pkg.DependencyInfo, error) {
	m.logger.Infof("Detecting dependencies in project: %s", projectPath)
	
	// Detect available package managers
	packageManagers, err := m.detector.DetectPackageManagers(projectPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to detect package managers: %w", err)
	}
	
	if len(packageManagers) == 0 {
		m.logger.Infof("No package managers detected in project")
		return []*pkg.PackageManagerInfo{}, []*pkg.DependencyInfo{}, nil
	}
	
	// Collect all dependencies
	var allDependencies []*pkg.DependencyInfo
	
	for _, pmInfo := range packageManagers {
		if !pmInfo.Available {
			m.logger.Warnf("Package manager %s detected but not available in PATH", pmInfo.Name)
			continue
		}
		
		manager, exists := m.managers[pmInfo.Type]
		if !exists {
			m.logger.Warnf("No implementation found for package manager: %s", pmInfo.Type)
			continue
		}
		
		dependencies, err := manager.ListDependencies(projectPath)
		if err != nil {
			m.logger.Warnf("Failed to list dependencies for %s: %v", pmInfo.Name, err)
			continue
		}
		
		allDependencies = append(allDependencies, dependencies...)
	}
	
	m.logger.Infof("Detected %d package managers and %d dependencies", len(packageManagers), len(allDependencies))
	return packageManagers, allDependencies, nil
}

// InstallAllDependencies installs dependencies for all detected package managers
func (m *manager) InstallAllDependencies(projectPath string) ([]*pkg.DependencyOperationResult, error) {
	m.logger.Infof("Installing dependencies for all package managers in: %s", projectPath)
	
	packageManagers, err := m.detector.DetectPackageManagers(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect package managers: %w", err)
	}
	
	var results []*pkg.DependencyOperationResult
	
	for _, pmInfo := range packageManagers {
		if !pmInfo.Available {
			result := &pkg.DependencyOperationResult{
				PackageManager: pmInfo.Name,
				Operation:      "install",
				Success:        false,
				Error:          fmt.Sprintf("Package manager %s not available in PATH", pmInfo.Name),
			}
			results = append(results, result)
			continue
		}
		
		manager, exists := m.managers[pmInfo.Type]
		if !exists {
			result := &pkg.DependencyOperationResult{
				PackageManager: pmInfo.Name,
				Operation:      "install",
				Success:        false,
				Error:          fmt.Sprintf("No implementation found for package manager: %s", pmInfo.Type),
			}
			results = append(results, result)
			continue
		}
		
		result, err := manager.InstallDependencies(projectPath)
		if err != nil {
			result = &pkg.DependencyOperationResult{
				PackageManager: pmInfo.Name,
				Operation:      "install",
				Success:        false,
				Error:          err.Error(),
			}
		}
		
		results = append(results, result)
	}
	
	return results, nil
}

// UpdateAllDependencies updates dependencies for all detected package managers
func (m *manager) UpdateAllDependencies(projectPath string) ([]*pkg.DependencyOperationResult, error) {
	m.logger.Infof("Updating dependencies for all package managers in: %s", projectPath)
	
	packageManagers, err := m.detector.DetectPackageManagers(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect package managers: %w", err)
	}
	
	var results []*pkg.DependencyOperationResult
	
	for _, pmInfo := range packageManagers {
		if !pmInfo.Available {
			result := &pkg.DependencyOperationResult{
				PackageManager: pmInfo.Name,
				Operation:      "update",
				Success:        false,
				Error:          fmt.Sprintf("Package manager %s not available in PATH", pmInfo.Name),
			}
			results = append(results, result)
			continue
		}
		
		manager, exists := m.managers[pmInfo.Type]
		if !exists {
			result := &pkg.DependencyOperationResult{
				PackageManager: pmInfo.Name,
				Operation:      "update",
				Success:        false,
				Error:          fmt.Sprintf("No implementation found for package manager: %s", pmInfo.Type),
			}
			results = append(results, result)
			continue
		}
		
		result, err := manager.UpdateDependencies(projectPath)
		if err != nil {
			result = &pkg.DependencyOperationResult{
				PackageManager: pmInfo.Name,
				Operation:      "update",
				Success:        false,
				Error:          err.Error(),
			}
		}
		
		results = append(results, result)
	}
	
	return results, nil
}

// AuditAllDependencies performs security audit across all package managers
func (m *manager) AuditAllDependencies(projectPath string) ([]*pkg.VulnerabilityInfo, error) {
	m.logger.Infof("Auditing dependencies for all package managers in: %s", projectPath)
	
	packageManagers, err := m.detector.DetectPackageManagers(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect package managers: %w", err)
	}
	
	var allVulnerabilities []*pkg.VulnerabilityInfo
	
	for _, pmInfo := range packageManagers {
		if !pmInfo.Available {
			m.logger.Warnf("Package manager %s not available for audit", pmInfo.Name)
			continue
		}
		
		manager, exists := m.managers[pmInfo.Type]
		if !exists {
			m.logger.Warnf("No implementation found for package manager: %s", pmInfo.Type)
			continue
		}
		
		vulnerabilities, err := manager.AuditDependencies(projectPath)
		if err != nil {
			m.logger.Warnf("Failed to audit dependencies for %s: %v", pmInfo.Name, err)
			continue
		}
		
		allVulnerabilities = append(allVulnerabilities, vulnerabilities...)
	}
	
	return allVulnerabilities, nil
}

// GetAllOutdatedDependencies finds outdated dependencies across all package managers
func (m *manager) GetAllOutdatedDependencies(projectPath string) ([]*pkg.DependencyInfo, error) {
	m.logger.Infof("Checking for outdated dependencies in: %s", projectPath)
	
	packageManagers, err := m.detector.DetectPackageManagers(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect package managers: %w", err)
	}
	
	var allOutdated []*pkg.DependencyInfo
	
	for _, pmInfo := range packageManagers {
		if !pmInfo.Available {
			m.logger.Warnf("Package manager %s not available for outdated check", pmInfo.Name)
			continue
		}
		
		manager, exists := m.managers[pmInfo.Type]
		if !exists {
			m.logger.Warnf("No implementation found for package manager: %s", pmInfo.Type)
			continue
		}
		
		outdated, err := manager.GetOutdatedDependencies(projectPath)
		if err != nil {
			m.logger.Warnf("Failed to check outdated dependencies for %s: %v", pmInfo.Name, err)
			continue
		}
		
		allOutdated = append(allOutdated, outdated...)
	}
	
	return allOutdated, nil
}

// CleanAllCaches cleans caches for all detected package managers
func (m *manager) CleanAllCaches(projectPath string) ([]*pkg.DependencyOperationResult, error) {
	m.logger.Infof("Cleaning caches for all package managers in: %s", projectPath)
	
	packageManagers, err := m.detector.DetectPackageManagers(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect package managers: %w", err)
	}
	
	var results []*pkg.DependencyOperationResult
	
	for _, pmInfo := range packageManagers {
		if !pmInfo.Available {
			result := &pkg.DependencyOperationResult{
				PackageManager: pmInfo.Name,
				Operation:      "clean",
				Success:        false,
				Error:          fmt.Sprintf("Package manager %s not available in PATH", pmInfo.Name),
			}
			results = append(results, result)
			continue
		}
		
		manager, exists := m.managers[pmInfo.Type]
		if !exists {
			result := &pkg.DependencyOperationResult{
				PackageManager: pmInfo.Name,
				Operation:      "clean",
				Success:        false,
				Error:          fmt.Sprintf("No implementation found for package manager: %s", pmInfo.Type),
			}
			results = append(results, result)
			continue
		}
		
		result, err := manager.CleanCache(projectPath)
		if err != nil {
			result = &pkg.DependencyOperationResult{
				PackageManager: pmInfo.Name,
				Operation:      "clean",
				Success:        false,
				Error:          err.Error(),
			}
		}
		
		results = append(results, result)
	}
	
	return results, nil
}

// GenerateDependencyReport creates a comprehensive dependency report
func (m *manager) GenerateDependencyReport(projectPath string) (*pkg.DependencyReport, error) {
	m.logger.Infof("Generating dependency report for: %s", projectPath)
	
	startTime := time.Now()
	
	// Detect dependencies
	packageManagers, dependencies, err := m.DetectProjectDependencies(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect dependencies: %w", err)
	}
	
	// Get vulnerabilities
	vulnerabilities, err := m.AuditAllDependencies(projectPath)
	if err != nil {
		m.logger.Warnf("Failed to audit dependencies: %v", err)
		vulnerabilities = []*pkg.VulnerabilityInfo{}
	}
	
	// Get outdated dependencies
	outdated, err := m.GetAllOutdatedDependencies(projectPath)
	if err != nil {
		m.logger.Warnf("Failed to check outdated dependencies: %v", err)
		outdated = []*pkg.DependencyInfo{}
	}
	
	// Calculate statistics
	directDeps := 0
	indirectDeps := 0
	vulnerabilityMap := make(map[string]int)
	licenseMap := make(map[string]int)
	
	for _, dep := range dependencies {
		if dep.Type == "direct" {
			directDeps++
		} else {
			indirectDeps++
		}
		
		if dep.License != "" {
			licenseMap[dep.License]++
		}
		
		for _, vuln := range dep.VulnerabilityInfo {
			vulnerabilityMap[vuln.Severity]++
		}
	}
	
	// Update outdated status
	outdatedSet := make(map[string]bool)
	for _, dep := range outdated {
		outdatedSet[dep.Name] = true
	}
	
	for _, dep := range dependencies {
		if outdatedSet[dep.Name] {
			dep.IsOutdated = true
		}
	}
	
	// Calculate security score (0-100, higher is better)
	securityScore := m.calculateSecurityScore(dependencies, vulnerabilities)
	
	report := &pkg.DependencyReport{
		ProjectPath:          projectPath,
		GeneratedAt:          startTime.Format(time.RFC3339),
		PackageManagers:      packageManagers,
		TotalDependencies:    len(dependencies),
		DirectDependencies:   directDeps,
		IndirectDependencies: indirectDeps,
		OutdatedDependencies: len(outdated),
		Vulnerabilities:      len(vulnerabilities),
		Dependencies:         dependencies,
		VulnerabilitySummary: vulnerabilityMap,
		LicenseSummary:       licenseMap,
		SecurityScore:        securityScore,
	}
	
	m.logger.Infof("Generated dependency report: %d dependencies, %d vulnerabilities, %.1f security score", 
		len(dependencies), len(vulnerabilities), securityScore)
	
	return report, nil
}

// calculateSecurityScore calculates a security score from 0-100 (higher is better)
func (m *manager) calculateSecurityScore(dependencies []*pkg.DependencyInfo, vulnerabilities []*pkg.VulnerabilityInfo) float64 {
	if len(dependencies) == 0 {
		return 100.0
	}
	
	score := 100.0
	
	// Deduct points for vulnerabilities
	criticalCount := 0
	highCount := 0
	moderateCount := 0
	lowCount := 0
	
	for _, vuln := range vulnerabilities {
		switch strings.ToLower(vuln.Severity) {
		case "critical":
			criticalCount++
			score -= 20.0
		case "high":
			highCount++
			score -= 10.0
		case "moderate":
			moderateCount++
			score -= 5.0
		case "low":
			lowCount++
			score -= 1.0
		}
	}
	
	// Deduct points for outdated dependencies
	outdatedCount := 0
	for _, dep := range dependencies {
		if dep.IsOutdated {
			outdatedCount++
		}
	}
	
	if len(dependencies) > 0 {
		outdatedRatio := float64(outdatedCount) / float64(len(dependencies))
		score -= outdatedRatio * 20.0 // Up to 20 points for outdated deps
	}
	
	// Ensure score is within bounds
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	
	return score
}

// Helper function to execute commands with timeout and logging
func (m *manager) execCommand(projectPath, command string, args []string, timeout time.Duration) (*pkg.DependencyOperationResult, error) {
	start := time.Now()
	
	cmd := exec.Command(command, args...)
	cmd.Dir = projectPath
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	result := &pkg.DependencyOperationResult{
		PackageManager: command,
		Operation:      strings.Join(args, " "),
		Success:        err == nil,
		Output:         string(output),
		Duration:       duration.String(),
	}
	
	if err != nil {
		result.Error = err.Error()
		m.logger.Debugf("Command failed: %s %v, error: %v, output: %s", command, args, err, string(output))
	} else {
		m.logger.Debugf("Command succeeded: %s %v, duration: %v", command, args, duration)
	}
	
	return result, err
}

// Helper function to check if a command is available in PATH
func (m *manager) isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// Helper function to get command version
func (m *manager) getCommandVersion(command string, versionArgs []string) string {
	cmd := exec.Command(command, versionArgs...)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	
	version := strings.TrimSpace(string(output))
	// Extract version number from output (basic implementation)
	lines := strings.Split(version, "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	
	return version
}

// Helper function to check if file exists
func (m *manager) fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// Helper function to find files with pattern
func (m *manager) findFiles(projectPath, pattern string) []string {
	var files []string
	
	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking
		}
		
		if info.IsDir() {
			// Skip common directories that don't contain package manager configs
			dirName := filepath.Base(path)
			if dirName == ".git" || dirName == "node_modules" || dirName == "target" || dirName == ".venv" {
				return filepath.SkipDir
			}
			return nil
		}
		
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			return nil
		}
		
		if matched {
			relPath, err := filepath.Rel(projectPath, path)
			if err != nil {
				relPath = path
			}
			files = append(files, relPath)
		}
		
		return nil
	})
	
	if err != nil {
		m.logger.Debugf("Error walking directory %s: %v", projectPath, err)
	}
	
	return files
}