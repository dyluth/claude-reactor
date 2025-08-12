package docker

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	
	"claude-reactor/pkg"
)

// NamingManager handles container and image naming logic
type NamingManager struct {
	logger pkg.Logger
	archDetector pkg.ArchitectureDetector
}

// NewNamingManager creates a new naming manager
func NewNamingManager(logger pkg.Logger, archDetector pkg.ArchitectureDetector) *NamingManager {
	return &NamingManager{
		logger: logger,
		archDetector: archDetector,
	}
}

// GetImageName generates an image name with architecture suffix
// Format: claude-reactor-{variant}-{arch}:{version}
func (nm *NamingManager) GetImageName(variant string) (string, error) {
	arch, err := nm.archDetector.GetHostArchitecture()
	if err != nil {
		return "", fmt.Errorf("failed to get architecture for image name: %w", err)
	}
	
	imageName := fmt.Sprintf("v2-claude-reactor-%s-%s", variant, arch)
	nm.logger.Debugf("Generated image name: %s", imageName)
	
	return imageName, nil
}

// GetContainerName generates a container name with architecture, project hash, and account
// Replicates bash logic: claude-reactor-{variant}-{arch}-{projectHash}-{account}
func (nm *NamingManager) GetContainerName(variant, account string) (string, error) {
	arch, err := nm.archDetector.GetHostArchitecture()
	if err != nil {
		return "", fmt.Errorf("failed to get architecture for container name: %w", err)
	}
	
	// Get project hash from current directory (replicating bash logic)
	projectHash, err := nm.getProjectHash()
	if err != nil {
		return "", fmt.Errorf("failed to get project hash: %w", err)
	}
	
	// Build container name parts
	parts := []string{"v2-claude-reactor", variant, arch, projectHash}
	
	// Add account if specified, otherwise use "default"
	if account != "" {
		parts = append(parts, account)
	} else {
		parts = append(parts, "default")
	}
	
	containerName := strings.Join(parts, "-")
	nm.logger.Debugf("Generated container name: %s (variant=%s, account=%s, hash=%s)", 
		containerName, variant, account, projectHash)
	
	return containerName, nil
}

// getProjectHash generates a hash based on current working directory
// Replicates: PROJECT_HASH=$(echo "$(pwd)" | shasum -a 256 | cut -c1-8)
func (nm *NamingManager) getProjectHash() (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}
	
	return nm.GetProjectHashFromPath(pwd)
}

// GetProjectHashFromPath generates a hash based on the provided path
func (nm *NamingManager) GetProjectHashFromPath(projectPath string) (string, error) {
	// Generate SHA256 hash and take first 8 characters
	hasher := sha256.New()
	hasher.Write([]byte(projectPath))
	hash := fmt.Sprintf("%x", hasher.Sum(nil))
	projectHash := hash[:8]
	
	nm.logger.Debugf("Project path: %s, hash: %s", projectPath, projectHash)
	
	return projectHash, nil
}