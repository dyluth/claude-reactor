package dependency

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"claude-reactor/pkg"
)

// mavenManager implements PackageManager for Maven
type mavenManager struct {
	logger pkg.Logger
}

// NewMavenManager creates a new Maven package manager
func NewMavenManager(logger pkg.Logger) pkg.PackageManager {
	return &mavenManager{
		logger: logger,
	}
}

func (m *mavenManager) GetType() string {
	return "maven"
}

func (m *mavenManager) GetName() string {
	return "Maven"
}

func (m *mavenManager) IsAvailable() bool {
	_, err := exec.LookPath("mvn")
	return err == nil
}

func (m *mavenManager) GetVersion() (string, error) {
	cmd := exec.Command("mvn", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get Maven version: %w", err)
	}
	
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0]), nil
	}
	
	return strings.TrimSpace(string(output)), nil
}

func (m *mavenManager) DetectConfigFiles(projectPath string) []string {
	var configFiles []string
	
	pomPath := filepath.Join(projectPath, "pom.xml")
	if m.fileExists(pomPath) {
		configFiles = append(configFiles, "pom.xml")
	}
	
	return configFiles
}

func (m *mavenManager) ListDependencies(projectPath string) ([]*pkg.DependencyInfo, error) {
	m.logger.Debugf("Listing Maven dependencies in: %s", projectPath)
	
	// Use mvn dependency:list to get dependencies
	cmd := exec.Command("mvn", "dependency:list", "-DoutputFile=/dev/stdout", "-Dsilent=true", "-DexcludeTransitive=false")
	cmd.Dir = projectPath
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list Maven dependencies: %w", err)
	}
	
	var dependencies []*pkg.DependencyInfo
	
	// Parse Maven dependency list output
	lines := strings.Split(string(output), "\n")
	depRegex := regexp.MustCompile(`^\[INFO\]\s+([^:]+):([^:]+):([^:]+):([^:]+)(?::([^:]+))?$`)
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		matches := depRegex.FindStringSubmatch(line)
		if len(matches) >= 5 {
			groupId := matches[1]
			artifactId := matches[2]
			packaging := matches[3]
			version := matches[4]
			scope := ""
			
			if len(matches) > 5 && matches[5] != "" {
				scope = matches[5]
			}
			
			// Skip test dependencies in main listing
			if scope == "test" {
				continue
			}
			
			name := fmt.Sprintf("%s:%s", groupId, artifactId)
			depType := "direct"
			if scope != "" && scope != "compile" {
				depType = "indirect"
			}
			
			dep := &pkg.DependencyInfo{
				Name:            name,
				CurrentVersion:  version,
				Type:           depType,
				PackageManager: "maven",
				Description:     fmt.Sprintf("Packaging: %s, Scope: %s", packaging, scope),
			}
			
			dependencies = append(dependencies, dep)
		}
	}
	
	m.logger.Debugf("Found %d Maven dependencies", len(dependencies))
	return dependencies, nil
}

func (m *mavenManager) InstallDependencies(projectPath string) (*pkg.DependencyOperationResult, error) {
	m.logger.Infof("Installing Maven dependencies in: %s", projectPath)
	
	start := time.Now()
	cmd := exec.Command("mvn", "dependency:resolve")
	cmd.Dir = projectPath
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	result := &pkg.DependencyOperationResult{
		PackageManager: "Maven",
		Operation:      "install",
		Success:        err == nil,
		Output:         string(output),
		Duration:       duration.String(),
	}
	
	if err != nil {
		result.Error = err.Error()
		m.logger.Errorf("Failed to install Maven dependencies: %v", err)
	} else {
		m.logger.Infof("Successfully installed Maven dependencies")
	}
	
	return result, nil
}

func (m *mavenManager) UpdateDependencies(projectPath string) (*pkg.DependencyOperationResult, error) {
	m.logger.Infof("Updating Maven dependencies in: %s", projectPath)
	
	start := time.Now()
	
	// Use versions plugin to update dependencies
	cmd := exec.Command("mvn", "versions:use-latest-versions", "-DallowMajorUpdates=false")
	cmd.Dir = projectPath
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	result := &pkg.DependencyOperationResult{
		PackageManager: "Maven",
		Operation:      "update",
		Success:        err == nil,
		Output:         string(output),
		Duration:       duration.String(),
	}
	
	if err != nil {
		result.Error = err.Error()
		m.logger.Errorf("Failed to update Maven dependencies: %v", err)
	} else {
		m.logger.Infof("Successfully updated Maven dependencies")
	}
	
	return result, nil
}

func (m *mavenManager) AuditDependencies(projectPath string) ([]*pkg.VulnerabilityInfo, error) {
	m.logger.Debugf("Auditing Maven dependencies in: %s", projectPath)
	
	// Check if OWASP dependency-check plugin is configured
	// This is a simplified implementation
	cmd := exec.Command("mvn", "org.owasp:dependency-check-maven:check", "-DfailOnError=false")
	cmd.Dir = projectPath
	
	output, err := cmd.Output()
	if err != nil {
		// Plugin might not be configured
		m.logger.Debugf("Maven vulnerability scanning not configured (OWASP dependency-check)")
		return []*pkg.VulnerabilityInfo{}, nil
	}
	
	// Parse OWASP dependency-check report (simplified)
	var vulnerabilities []*pkg.VulnerabilityInfo
	
	if strings.Contains(string(output), "vulnerabilities found") {
		// This would need proper parsing of the OWASP report
		vuln := &pkg.VulnerabilityInfo{
			ID:          "MAVEN-VULN-DETECTED",
			Title:       "Maven Vulnerability Detected",
			Description: "One or more vulnerabilities found by OWASP dependency-check",
			Severity:    "moderate",
			Reference:   "Check target/dependency-check-report.html for details",
		}
		vulnerabilities = append(vulnerabilities, vuln)
	}
	
	m.logger.Debugf("Found %d Maven vulnerabilities", len(vulnerabilities))
	return vulnerabilities, nil
}

func (m *mavenManager) GetOutdatedDependencies(projectPath string) ([]*pkg.DependencyInfo, error) {
	m.logger.Debugf("Checking outdated Maven dependencies in: %s", projectPath)
	
	// Use versions plugin to check for updates
	cmd := exec.Command("mvn", "versions:display-dependency-updates", "-DprocessDependencies=true")
	cmd.Dir = projectPath
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to check Maven dependency updates: %w", err)
	}
	
	var outdated []*pkg.DependencyInfo
	
	// Parse Maven versions plugin output (simplified)
	lines := strings.Split(string(output), "\n")
	updateRegex := regexp.MustCompile(`^\[INFO\]\s+([^:]+):([^:]+).*?([0-9][^:]*)\s+->\s+([0-9][^:]*)\s*$`)
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		matches := updateRegex.FindStringSubmatch(line)
		if len(matches) >= 5 {
			groupId := matches[1]
			artifactId := matches[2]
			currentVersion := matches[3]
			latestVersion := matches[4]
			
			name := fmt.Sprintf("%s:%s", groupId, artifactId)
			
			dep := &pkg.DependencyInfo{
				Name:            name,
				CurrentVersion:  currentVersion,
				LatestVersion:   latestVersion,
				Type:           "direct",
				PackageManager: "maven",
				IsOutdated:     true,
			}
			
			outdated = append(outdated, dep)
		}
	}
	
	m.logger.Debugf("Found %d outdated Maven dependencies", len(outdated))
	return outdated, nil
}

func (m *mavenManager) CleanCache(projectPath string) (*pkg.DependencyOperationResult, error) {
	m.logger.Infof("Cleaning Maven cache")
	
	start := time.Now()
	cmd := exec.Command("mvn", "dependency:purge-local-repository")
	cmd.Dir = projectPath
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	result := &pkg.DependencyOperationResult{
		PackageManager: "Maven",
		Operation:      "clean",
		Success:        err == nil,
		Output:         string(output),
		Duration:       duration.String(),
	}
	
	if err != nil {
		result.Error = err.Error()
		m.logger.Errorf("Failed to clean Maven cache: %v", err)
	} else {
		m.logger.Infof("Successfully cleaned Maven cache")
	}
	
	return result, nil
}

func (m *mavenManager) fileExists(path string) bool {
	manager := &manager{logger: m.logger}
	return manager.fileExists(path)
}

// gradleManager implements PackageManager for Gradle
type gradleManager struct {
	logger pkg.Logger
}

// NewGradleManager creates a new Gradle package manager
func NewGradleManager(logger pkg.Logger) pkg.PackageManager {
	return &gradleManager{
		logger: logger,
	}
}

func (g *gradleManager) GetType() string {
	return "gradle"
}

func (g *gradleManager) GetName() string {
	return "Gradle"
}

func (g *gradleManager) IsAvailable() bool {
	// Check for gradle or gradlew
	if _, err := exec.LookPath("gradle"); err == nil {
		return true
	}
	if _, err := exec.LookPath("gradlew"); err == nil {
		return true
	}
	return false
}

func (g *gradleManager) GetVersion() (string, error) {
	// Try gradlew first, then gradle
	commands := [][]string{
		{"./gradlew", "--version"},
		{"gradle", "--version"},
	}
	
	for _, cmd := range commands {
		if output, err := exec.Command(cmd[0], cmd[1:]...).Output(); err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(line, "Gradle") {
					return strings.TrimSpace(line), nil
				}
			}
		}
	}
	
	return "", fmt.Errorf("failed to get Gradle version")
}

func (g *gradleManager) DetectConfigFiles(projectPath string) []string {
	var configFiles []string
	
	buildFiles := []string{"build.gradle", "build.gradle.kts"}
	for _, file := range buildFiles {
		if g.fileExists(filepath.Join(projectPath, file)) {
			configFiles = append(configFiles, file)
		}
	}
	
	return configFiles
}

func (g *gradleManager) ListDependencies(projectPath string) ([]*pkg.DependencyInfo, error) {
	g.logger.Debugf("Listing Gradle dependencies in: %s", projectPath)
	
	// Use gradle dependencies task
	cmd := g.getGradleCommand(projectPath, "dependencies", "--configuration=compileClasspath")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list Gradle dependencies: %w", err)
	}
	
	var dependencies []*pkg.DependencyInfo
	
	// Parse Gradle dependencies output (simplified)
	lines := strings.Split(string(output), "\n")
	depRegex := regexp.MustCompile(`^[+\\\-\s]*([^:]+):([^:]+):([^:]+)`)
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		matches := depRegex.FindStringSubmatch(line)
		if len(matches) >= 4 {
			groupId := matches[1]
			artifactId := matches[2]
			version := matches[3]
			
			name := fmt.Sprintf("%s:%s", groupId, artifactId)
			
			// Determine dependency type based on tree structure
			depType := "direct"
			if strings.Contains(line, "\\---") || strings.Contains(line, "+---") {
				if strings.Count(line, " ") > 5 {
					depType = "indirect"
				}
			}
			
			dep := &pkg.DependencyInfo{
				Name:            name,
				CurrentVersion:  version,
				Type:           depType,
				PackageManager: "gradle",
			}
			
			dependencies = append(dependencies, dep)
		}
	}
	
	g.logger.Debugf("Found %d Gradle dependencies", len(dependencies))
	return dependencies, nil
}

func (g *gradleManager) InstallDependencies(projectPath string) (*pkg.DependencyOperationResult, error) {
	g.logger.Infof("Installing Gradle dependencies in: %s", projectPath)
	
	start := time.Now()
	cmd := g.getGradleCommand(projectPath, "build", "--no-daemon")
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	result := &pkg.DependencyOperationResult{
		PackageManager: "Gradle",
		Operation:      "install",
		Success:        err == nil,
		Output:         string(output),
		Duration:       duration.String(),
	}
	
	if err != nil {
		result.Error = err.Error()
		g.logger.Errorf("Failed to install Gradle dependencies: %v", err)
	} else {
		g.logger.Infof("Successfully installed Gradle dependencies")
	}
	
	return result, nil
}

func (g *gradleManager) UpdateDependencies(projectPath string) (*pkg.DependencyOperationResult, error) {
	g.logger.Infof("Updating Gradle dependencies in: %s", projectPath)
	
	start := time.Now()
	
	// Use refreshDependencies task or dependency updates plugin
	cmd := g.getGradleCommand(projectPath, "build", "--refresh-dependencies", "--no-daemon")
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	result := &pkg.DependencyOperationResult{
		PackageManager: "Gradle",
		Operation:      "update",
		Success:        err == nil,
		Output:         string(output),
		Duration:       duration.String(),
	}
	
	if err != nil {
		result.Error = err.Error()
		g.logger.Errorf("Failed to update Gradle dependencies: %v", err)
	} else {
		g.logger.Infof("Successfully updated Gradle dependencies")
	}
	
	return result, nil
}

func (g *gradleManager) AuditDependencies(projectPath string) ([]*pkg.VulnerabilityInfo, error) {
	g.logger.Debugf("Auditing Gradle dependencies in: %s", projectPath)
	
	// Check if OWASP dependency-check plugin is configured
	cmd := g.getGradleCommand(projectPath, "dependencyCheckAnalyze", "--no-daemon")
	
	output, err := cmd.Output()
	if err != nil {
		// Plugin might not be configured
		g.logger.Debugf("Gradle vulnerability scanning not configured (OWASP dependency-check)")
		return []*pkg.VulnerabilityInfo{}, nil
	}
	
	// Parse OWASP dependency-check report (simplified)
	var vulnerabilities []*pkg.VulnerabilityInfo
	
	if strings.Contains(string(output), "vulnerabilities found") {
		vuln := &pkg.VulnerabilityInfo{
			ID:          "GRADLE-VULN-DETECTED",
			Title:       "Gradle Vulnerability Detected",
			Description: "One or more vulnerabilities found by OWASP dependency-check",
			Severity:    "moderate",
			Reference:   "Check build/reports/dependency-check-report.html for details",
		}
		vulnerabilities = append(vulnerabilities, vuln)
	}
	
	g.logger.Debugf("Found %d Gradle vulnerabilities", len(vulnerabilities))
	return vulnerabilities, nil
}

func (g *gradleManager) GetOutdatedDependencies(projectPath string) ([]*pkg.DependencyInfo, error) {
	g.logger.Debugf("Checking outdated Gradle dependencies in: %s", projectPath)
	
	// Use dependency updates plugin if available
	cmd := g.getGradleCommand(projectPath, "dependencyUpdates", "--no-daemon")
	
	output, err := cmd.Output()
	if err != nil {
		g.logger.Debugf("Gradle dependency updates plugin not available")
		return []*pkg.DependencyInfo{}, nil
	}
	
	// Parse dependency updates output (simplified)
	var outdated []*pkg.DependencyInfo
	
	// This would need proper parsing of the dependency updates report
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "->") {
			// This is a simplified parser - a full implementation would parse the report properly
		}
	}
	
	g.logger.Debugf("Found %d outdated Gradle dependencies", len(outdated))
	return outdated, nil
}

func (g *gradleManager) CleanCache(projectPath string) (*pkg.DependencyOperationResult, error) {
	g.logger.Infof("Cleaning Gradle cache")
	
	start := time.Now()
	cmd := g.getGradleCommand(projectPath, "clean", "--no-daemon")
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	
	result := &pkg.DependencyOperationResult{
		PackageManager: "Gradle",
		Operation:      "clean",
		Success:        err == nil,
		Output:         string(output),
		Duration:       duration.String(),
	}
	
	if err != nil {
		result.Error = err.Error()
		g.logger.Errorf("Failed to clean Gradle cache: %v", err)
	} else {
		g.logger.Infof("Successfully cleaned Gradle cache")
	}
	
	return result, nil
}

func (g *gradleManager) getGradleCommand(projectPath string, args ...string) *exec.Cmd {
	// Prefer gradlew if available
	gradlewPath := filepath.Join(projectPath, "gradlew")
	if g.fileExists(gradlewPath) {
		cmd := exec.Command("./gradlew", args...)
		cmd.Dir = projectPath
		return cmd
	}
	
	// Fall back to gradle
	cmd := exec.Command("gradle", args...)
	cmd.Dir = projectPath
	return cmd
}

func (g *gradleManager) fileExists(path string) bool {
	manager := &manager{logger: g.logger}
	return manager.fileExists(path)
}