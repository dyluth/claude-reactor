package dependency

import (
	"fmt"
	"os"
	"path/filepath"

	"claude-reactor/pkg"
)

// detector implements the PackageManagerDetector interface
type detector struct {
	logger pkg.Logger
}

// NewDetector creates a new package manager detector
func NewDetector(logger pkg.Logger) pkg.PackageManagerDetector {
	return &detector{
		logger: logger,
	}
}

// DetectPackageManagers scans project and returns detected package managers
func (d *detector) DetectPackageManagers(projectPath string) ([]*pkg.PackageManagerInfo, error) {
	d.logger.Debugf("Detecting package managers in: %s", projectPath)
	
	var detected []*pkg.PackageManagerInfo
	
	// Define package manager detection rules
	detectionRules := []struct {
		pmType      string
		name        string
		configFiles []string
		lockFiles   []string
		executable  string
	}{
		{
			pmType:      "go",
			name:        "Go Modules",
			configFiles: []string{"go.mod"},
			lockFiles:   []string{"go.sum"},
			executable:  "go",
		},
		{
			pmType:      "cargo",
			name:        "Cargo",
			configFiles: []string{"Cargo.toml"},
			lockFiles:   []string{"Cargo.lock"},
			executable:  "cargo",
		},
		{
			pmType:      "npm",
			name:        "npm",
			configFiles: []string{"package.json"},
			lockFiles:   []string{"package-lock.json"},
			executable:  "npm",
		},
		{
			pmType:      "yarn",
			name:        "Yarn",
			configFiles: []string{"package.json"},
			lockFiles:   []string{"yarn.lock"},
			executable:  "yarn",
		},
		{
			pmType:      "pnpm",
			name:        "pnpm",
			configFiles: []string{"package.json"},
			lockFiles:   []string{"pnpm-lock.yaml"},
			executable:  "pnpm",
		},
		{
			pmType:      "pip",
			name:        "pip",
			configFiles: []string{"requirements.txt", "requirements.in", "setup.py", "pyproject.toml"},
			lockFiles:   []string{"requirements.txt"},
			executable:  "pip",
		},
		{
			pmType:      "poetry",
			name:        "Poetry",
			configFiles: []string{"pyproject.toml"},
			lockFiles:   []string{"poetry.lock"},
			executable:  "poetry",
		},
		{
			pmType:      "pipenv",
			name:        "Pipenv",
			configFiles: []string{"Pipfile"},
			lockFiles:   []string{"Pipfile.lock"},
			executable:  "pipenv",
		},
		{
			pmType:      "maven",
			name:        "Maven",
			configFiles: []string{"pom.xml"},
			lockFiles:   []string{},
			executable:  "mvn",
		},
		{
			pmType:      "gradle",
			name:        "Gradle",
			configFiles: []string{"build.gradle", "build.gradle.kts"},
			lockFiles:   []string{"gradle.lockfile"},
			executable:  "gradle",
		},
	}
	
	// Check each package manager
	for _, rule := range detectionRules {
		pmInfo := d.checkPackageManager(projectPath, rule.pmType, rule.name, rule.configFiles, rule.lockFiles, rule.executable)
		if pmInfo != nil {
			detected = append(detected, pmInfo)
		}
	}
	
	// Special handling for Node.js package managers (prefer based on lock file)
	detected = d.prioritizeNodePackageManagers(detected)
	
	d.logger.Debugf("Detected %d package managers", len(detected))
	return detected, nil
}

// checkPackageManager checks if a specific package manager is present
func (d *detector) checkPackageManager(projectPath, pmType, name string, configFiles, lockFiles []string, executable string) *pkg.PackageManagerInfo {
	// Check for configuration files
	var foundConfigFiles []string
	var foundLockFiles []string
	
	for _, configFile := range configFiles {
		if d.fileExists(filepath.Join(projectPath, configFile)) {
			foundConfigFiles = append(foundConfigFiles, configFile)
		}
	}
	
	// If no config files found, package manager is not present
	if len(foundConfigFiles) == 0 {
		return nil
	}
	
	// Check for lock files
	for _, lockFile := range lockFiles {
		if d.fileExists(filepath.Join(projectPath, lockFile)) {
			foundLockFiles = append(foundLockFiles, lockFile)
		}
	}
	
	// Check if executable is available
	available := d.isCommandAvailable(executable)
	
	// Get version if available
	version := ""
	if available {
		version = d.getPackageManagerVersion(pmType, executable)
	}
	
	pmInfo := &pkg.PackageManagerInfo{
		Type:        pmType,
		Name:        name,
		ConfigFiles: foundConfigFiles,
		LockFiles:   foundLockFiles,
		Version:     version,
		Available:   available,
	}
	
	d.logger.Debugf("Detected %s: config=%v, lock=%v, available=%v", name, foundConfigFiles, foundLockFiles, available)
	return pmInfo
}

// prioritizeNodePackageManagers handles Node.js package manager priority
func (d *detector) prioritizeNodePackageManagers(detected []*pkg.PackageManagerInfo) []*pkg.PackageManagerInfo {
	var nodeManagers []*pkg.PackageManagerInfo
	var otherManagers []*pkg.PackageManagerInfo
	
	// Separate Node.js package managers from others
	for _, pm := range detected {
		if pm.Type == "npm" || pm.Type == "yarn" || pm.Type == "pnpm" {
			nodeManagers = append(nodeManagers, pm)
		} else {
			otherManagers = append(otherManagers, pm)
		}
	}
	
	// If multiple Node.js package managers detected, prioritize based on lock files
	if len(nodeManagers) > 1 {
		var preferred *pkg.PackageManagerInfo
		
		// Priority order: pnpm > yarn > npm
		for _, pm := range nodeManagers {
			if pm.Type == "pnpm" && len(pm.LockFiles) > 0 {
				preferred = pm
				break
			}
		}
		
		if preferred == nil {
			for _, pm := range nodeManagers {
				if pm.Type == "yarn" && len(pm.LockFiles) > 0 {
					preferred = pm
					break
				}
			}
		}
		
		if preferred == nil {
			for _, pm := range nodeManagers {
				if pm.Type == "npm" && len(pm.LockFiles) > 0 {
					preferred = pm
					break
				}
			}
		}
		
		// If no lock files, prefer by availability and type
		if preferred == nil {
			for _, pmType := range []string{"pnpm", "yarn", "npm"} {
				for _, pm := range nodeManagers {
					if pm.Type == pmType && pm.Available {
						preferred = pm
						break
					}
				}
				if preferred != nil {
					break
				}
			}
		}
		
		// Fallback to first available
		if preferred == nil {
			for _, pm := range nodeManagers {
				if pm.Available {
					preferred = pm
					break
				}
			}
		}
		
		if preferred != nil {
			d.logger.Debugf("Prioritized %s over other Node.js package managers", preferred.Name)
			otherManagers = append(otherManagers, preferred)
		}
	} else {
		otherManagers = append(otherManagers, nodeManagers...)
	}
	
	return otherManagers
}

// GetAvailablePackageManagers returns all package managers available on the system
func (d *detector) GetAvailablePackageManagers() ([]*pkg.PackageManagerInfo, error) {
	d.logger.Debug("Checking available package managers on system")
	
	var available []*pkg.PackageManagerInfo
	
	// Define all supported package managers
	managers := []struct {
		pmType     string
		name       string
		executable string
	}{
		{"go", "Go Modules", "go"},
		{"cargo", "Cargo", "cargo"},
		{"npm", "npm", "npm"},
		{"yarn", "Yarn", "yarn"},
		{"pnpm", "pnpm", "pnpm"},
		{"pip", "pip", "pip"},
		{"poetry", "Poetry", "poetry"},
		{"pipenv", "Pipenv", "pipenv"},
		{"maven", "Maven", "mvn"},
		{"gradle", "Gradle", "gradle"},
	}
	
	for _, mgr := range managers {
		if d.isCommandAvailable(mgr.executable) {
			version := d.getPackageManagerVersion(mgr.pmType, mgr.executable)
			pmInfo := &pkg.PackageManagerInfo{
				Type:        mgr.pmType,
				Name:        mgr.name,
				ConfigFiles: []string{},
				LockFiles:   []string{},
				Version:     version,
				Available:   true,
			}
			available = append(available, pmInfo)
		}
	}
	
	d.logger.Debugf("Found %d available package managers on system", len(available))
	return available, nil
}

// GetPackageManagerByType returns a specific package manager implementation
func (d *detector) GetPackageManagerByType(pmType string) (pkg.PackageManager, error) {
	switch pmType {
	case "go":
		return NewGoManager(d.logger), nil
	case "cargo":
		return NewCargoManager(d.logger), nil
	case "npm":
		return NewNpmManager(d.logger), nil
	case "yarn":
		return NewYarnManager(d.logger), nil
	case "pnpm":
		return NewPnpmManager(d.logger), nil
	case "pip":
		return NewPipManager(d.logger), nil
	case "poetry":
		return NewPoetryManager(d.logger), nil
	case "pipenv":
		return NewPipenvManager(d.logger), nil
	case "maven":
		return NewMavenManager(d.logger), nil
	case "gradle":
		return NewGradleManager(d.logger), nil
	default:
		return nil, fmt.Errorf("unsupported package manager type: %s", pmType)
	}
}

// getPackageManagerVersion gets the version of a specific package manager
func (d *detector) getPackageManagerVersion(pmType, executable string) string {
	var versionArgs []string
	
	switch pmType {
	case "go":
		versionArgs = []string{"version"}
	case "cargo":
		versionArgs = []string{"--version"}
	case "npm", "yarn", "pnpm":
		versionArgs = []string{"--version"}
	case "pip":
		versionArgs = []string{"--version"}
	case "poetry":
		versionArgs = []string{"--version"}
	case "pipenv":
		versionArgs = []string{"--version"}
	case "maven":
		versionArgs = []string{"--version"}
	case "gradle":
		versionArgs = []string{"--version"}
	default:
		versionArgs = []string{"--version"}
	}
	
	version := d.getCommandVersion(executable, versionArgs)
	if version == "" {
		d.logger.Debugf("Could not determine version for %s", executable)
	}
	
	return version
}

// Helper functions (same as in manager.go)
func (d *detector) fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func (d *detector) isCommandAvailable(command string) bool {
	// Use the same implementation as manager.go
	manager := &manager{logger: d.logger}
	return manager.isCommandAvailable(command)
}

func (d *detector) getCommandVersion(command string, versionArgs []string) string {
	// Use the same implementation as manager.go  
	manager := &manager{logger: d.logger}
	return manager.getCommandVersion(command, versionArgs)
}