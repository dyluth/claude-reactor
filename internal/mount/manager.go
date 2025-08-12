package mount

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"claude-reactor/pkg"
)

// manager implements the MountManager interface
type manager struct {
	logger pkg.Logger
}

// NewManager creates a new mount manager instance
func NewManager(logger pkg.Logger) pkg.MountManager {
	return &manager{
		logger: logger,
	}
}

// ValidateMountPath validates and expands mount paths
func (m *manager) ValidateMountPath(path string) (string, error) {
	// Expand home directory if present
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("unable to get home directory: %w", err)
		}
		path = filepath.Join(homeDir, path[2:])
	}
	
	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("unable to convert to absolute path: %w", err)
	}
	
	// Check if path exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return "", fmt.Errorf("mount path does not exist: %s", absPath)
	}
	
	m.logger.Debugf("Validated mount path: %s -> %s", path, absPath)
	return absPath, nil
}

// AddMountToConfig adds mount configuration to container config
func (m *manager) AddMountToConfig(config *pkg.ContainerConfig, sourcePath, targetPath string) error {
	// Validate source path
	validatedSource, err := m.ValidateMountPath(sourcePath)
	if err != nil {
		return fmt.Errorf("invalid mount source path: %w", err)
	}
	
	// Create mount configuration
	mount := pkg.Mount{
		Source:   validatedSource,
		Target:   targetPath,
		Type:     "bind",
		ReadOnly: false,
	}
	
	// Add to config
	config.Mounts = append(config.Mounts, mount)
	
	m.logger.Debugf("Added mount: %s -> %s", validatedSource, targetPath)
	return nil
}

// GetMountSummary returns formatted summary of mounts
func (m *manager) GetMountSummary(mounts []pkg.Mount) string {
	if len(mounts) == 0 {
		return "No additional mounts"
	}
	
	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Mounts (%d):\n", len(mounts)))
	
	for _, mount := range mounts {
		readOnlyStr := ""
		if mount.ReadOnly {
			readOnlyStr = " (read-only)"
		}
		summary.WriteString(fmt.Sprintf("  %s -> %s%s\n", 
			mount.Source, mount.Target, readOnlyStr))
	}
	
	return strings.TrimSpace(summary.String())
}

// UpdateMountSettings updates Claude settings for mounted directories
func (m *manager) UpdateMountSettings(mountPaths []string) error {
	if len(mountPaths) == 0 {
		m.logger.Debug("No mount paths to update in settings")
		return nil
	}
	
	m.logger.Infof("Updating Claude settings for %d mount paths", len(mountPaths))
	
	// This is a placeholder for the actual Claude settings update logic
	// The bash script updates Claude's settings.json to include mounted directories
	// This would need to be implemented based on Claude CLI's settings format
	
	for _, path := range mountPaths {
		m.logger.Debugf("Would update settings for mount: %s", path)
	}
	
	return nil
}