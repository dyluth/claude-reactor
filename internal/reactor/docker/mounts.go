package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/mount"
	
	"claude-reactor/pkg"
)

// MountManager handles Docker container mount configuration
type MountManager struct {
	logger pkg.Logger
}

// NewMountManager creates a new mount manager
func NewMountManager(logger pkg.Logger) *MountManager {
	return &MountManager{
		logger: logger,
	}
}

// CreateDefaultMounts creates the standard mount configuration for Claude containers
func (mm *MountManager) CreateDefaultMounts(account string) ([]pkg.Mount, error) {
	mounts := []pkg.Mount{}
	
	// 1. Project mount - Current directory to /app
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}
	
	mounts = append(mounts, pkg.Mount{
		Source: currentDir,
		Target: "/app",
		Type:   "bind",
		ReadOnly: false,
	})
	
	mm.logger.Debugf("Added project mount: %s -> /app", currentDir)

	// 2. Kubernetes config mount (read-only)
	kubeConfig := filepath.Join(os.Getenv("HOME"), ".kube")
	if _, err := os.Stat(kubeConfig); err == nil {
		mounts = append(mounts, pkg.Mount{
			Source: kubeConfig,
			Target: "/home/claude/.kube",
			Type:   "bind",
			ReadOnly: true,
		})
		mm.logger.Debugf("Added Kubernetes config mount")
	}
	
	// 3. Git config mount (read-only)
	gitConfig := filepath.Join(os.Getenv("HOME"), ".gitconfig")
	if _, err := os.Stat(gitConfig); err == nil {
		mounts = append(mounts, pkg.Mount{
			Source: gitConfig,
			Target: "/home/claude/.gitconfig",
			Type:   "bind",
			ReadOnly: true,
		})
		mm.logger.Debugf("Added Git config mount")
	}
	
	// 4. Claude config mount (account-specific)
	claudeMounts, err := mm.createClaudeConfigMounts(account)
	if err != nil {
		mm.logger.Warnf("Failed to create Claude config mounts: %v", err)
	} else {
		mounts = append(mounts, claudeMounts...)
	}
	
	return mounts, nil
}

// AddUserMounts adds user-specified mount paths to the mount configuration
func (mm *MountManager) AddUserMounts(mounts []pkg.Mount, mountPaths []string) ([]pkg.Mount, error) {
	for _, mountPath := range mountPaths {
		// Expand tilde if present
		expandedPath := mm.expandPath(mountPath)
		
		// Convert to absolute path if relative
		if !filepath.IsAbs(expandedPath) {
			currentDir, err := os.Getwd()
			if err != nil {
				return nil, fmt.Errorf("failed to get current directory: %w", err)
			}
			expandedPath = filepath.Join(currentDir, expandedPath)
		}
		
		// Check if directory exists
		if _, err := os.Stat(expandedPath); os.IsNotExist(err) {
			mm.logger.Warnf("Mount path does not exist: %s", expandedPath)
			continue
		}
		
		// Create mount point using basename
		mountName := filepath.Base(expandedPath)
		targetPath := filepath.Join("/mnt", mountName)
		
		mount := pkg.Mount{
			Source: expandedPath,
			Target: targetPath,
			Type:   "bind",
			ReadOnly: false,
		}
		
		mounts = append(mounts, mount)
		mm.logger.Debugf("Added user mount: %s -> %s", expandedPath, targetPath)
	}
	
	return mounts, nil
}

// ValidateMounts checks that all mount sources exist and are accessible
func (mm *MountManager) ValidateMounts(mounts []pkg.Mount) error {
	for _, mount := range mounts {
		// Skip validation for Docker socket and other system mounts
		if mount.Source == "/var/run/docker.sock" {
			continue
		}
		
		// Check if source exists
		if _, err := os.Stat(mount.Source); os.IsNotExist(err) {
			return fmt.Errorf("mount source does not exist: %s", mount.Source)
		}
		
		// Check if source is readable (skip for sockets as they can't be opened)
		if info, err := os.Stat(mount.Source); err == nil {
			// Skip accessibility check for sockets as they can't be opened with os.Open()
			if info.Mode()&os.ModeSocket != 0 {
				// This is a socket file, skip the open test
				continue
			}
		}
		
		file, err := os.Open(mount.Source)
		if err != nil {
			return fmt.Errorf("mount source is not accessible: %s (%v)", mount.Source, err)
		}
		file.Close()
	}
	
	return nil
}

// ConvertToDockerMounts converts pkg.Mount to Docker SDK mount.Mount
func (mm *MountManager) ConvertToDockerMounts(pkgMounts []pkg.Mount) []mount.Mount {
	dockerMounts := make([]mount.Mount, len(pkgMounts))
	
	for i, pkgMount := range pkgMounts {
		dockerMounts[i] = mount.Mount{
			Type:     mount.Type(pkgMount.Type),
			Source:   pkgMount.Source,
			Target:   pkgMount.Target,
			ReadOnly: pkgMount.ReadOnly,
		}
	}
	
	return dockerMounts
}

// GetMountSummary returns a human-readable summary of mounts
func (mm *MountManager) GetMountSummary(mounts []pkg.Mount) []string {
	summary := make([]string, 0, len(mounts))
	
	for _, mount := range mounts {
		readOnlyStr := ""
		if mount.ReadOnly {
			readOnlyStr = " (read-only)"
		}
		summary = append(summary, fmt.Sprintf("%s -> %s%s", mount.Source, mount.Target, readOnlyStr))
	}
	
	return summary
}

// createClaudeConfigMounts creates account-specific Claude configuration mounts
func (mm *MountManager) createClaudeConfigMounts(account string) ([]pkg.Mount, error) {
	mounts := []pkg.Mount{}
	
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		return mounts, fmt.Errorf("HOME environment variable not set")
	}
	
	// Determine Claude config directory based on account
	var claudeConfigDir string
	if account != "" && account != "default" {
		// Named account - use isolated directory
		claudeConfigDir = filepath.Join(homeDir, ".claude-reactor", fmt.Sprintf(".%s-claude.json", account))
		claudeDotDir := filepath.Join(homeDir, ".claude-reactor", fmt.Sprintf(".%s-claude", account))
		
		// Check for account-specific config file
		if _, err := os.Stat(claudeConfigDir); err == nil {
			mounts = append(mounts, pkg.Mount{
				Source: claudeConfigDir,
				Target: "/home/claude/.claude.json",
				Type:   "bind",
				ReadOnly: false,
			})
			mm.logger.Debugf("Added account-specific Claude config: %s", account)
		}
		
		// Always mount account-specific .claude directory (create if it doesn't exist)
		if _, err := os.Stat(claudeDotDir); os.IsNotExist(err) {
			// Create the directory if it doesn't exist
			if err := os.MkdirAll(claudeDotDir, 0755); err != nil {
				mm.logger.Warnf("Failed to create account-specific .claude directory: %v", err)
			} else {
				mm.logger.Debugf("Created missing account-specific .claude directory: %s", claudeDotDir)
			}
		}
		
		// Always add the mount for account-specific .claude directory (now that it exists)
		mounts = append(mounts, pkg.Mount{
			Source: claudeDotDir,
			Target: "/home/claude/.claude",
			Type:   "bind",
			ReadOnly: false,
		})
		mm.logger.Debugf("Added account-specific .claude directory mount: %s", account)
	} else {
		// Default account - use main Claude directories
		claudeJSON := filepath.Join(homeDir, ".claude.json")
		claudeDotDir := filepath.Join(homeDir, ".claude")
		
		// Check for main Claude config file
		if _, err := os.Stat(claudeJSON); err == nil {
			mounts = append(mounts, pkg.Mount{
				Source: claudeJSON,
				Target: "/home/claude/.claude.json",
				Type:   "bind",
				ReadOnly: false,
			})
			mm.logger.Debugf("Added main Claude config file")
		}
		
		// Always mount .claude directory (create if it doesn't exist)
		if _, err := os.Stat(claudeDotDir); os.IsNotExist(err) {
			// Create the directory if it doesn't exist
			if err := os.MkdirAll(claudeDotDir, 0755); err != nil {
				mm.logger.Warnf("Failed to create .claude directory: %v", err)
			} else {
				mm.logger.Debugf("Created missing .claude directory: %s", claudeDotDir)
			}
		}
		
		// Always add the mount for .claude directory (now that it exists)
		mounts = append(mounts, pkg.Mount{
			Source: claudeDotDir,
			Target: "/home/claude/.claude",
			Type:   "bind",
			ReadOnly: false,
		})
		mm.logger.Debugf("Added main .claude directory mount")
	}
	
	return mounts, nil
}

// expandPath expands tilde (~) to home directory
func (mm *MountManager) expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir := os.Getenv("HOME")
		if homeDir != "" {
			return filepath.Join(homeDir, path[2:])
		}
	}
	return path
}