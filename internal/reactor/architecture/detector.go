package architecture

import (
	"fmt"
	"runtime"
	"strings"

	"claude-reactor/pkg"
)

// detector implements the ArchitectureDetector interface
type detector struct {
	logger pkg.Logger
}

// NewDetector creates a new architecture detector
func NewDetector(logger pkg.Logger) pkg.ArchitectureDetector {
	return &detector{
		logger: logger,
	}
}

// GetHostArchitecture returns the host system architecture
// Replicates the detect_architecture() bash function
func (d *detector) GetHostArchitecture() (string, error) {
	arch := runtime.GOARCH
	
	d.logger.Debugf("Detected raw architecture: %s", arch)
	
	switch arch {
	case "amd64":
		return "amd64", nil
	case "arm64":
		return "arm64", nil
	case "386":
		return "i386", nil
	case "arm":
		return "arm", nil
	default:
		return "", fmt.Errorf("unsupported architecture: %s", arch)
	}
}

// GetDockerPlatform returns the Docker platform format
// Replicates the get_docker_platform() bash function
func (d *detector) GetDockerPlatform() (string, error) {
	arch, err := d.GetHostArchitecture()
	if err != nil {
		return "", fmt.Errorf("failed to detect host architecture: %w", err)
	}
	
	var platform string
	switch arch {
	case "amd64":
		platform = "linux/amd64"
	case "arm64":
		platform = "linux/arm64"
	case "i386":
		platform = "linux/386"
	case "arm":
		platform = "linux/arm/v7"
	default:
		return "", fmt.Errorf("unsupported architecture for Docker platform: %s", arch)
	}
	
	d.logger.Debugf("Docker platform: %s", platform)
	return platform, nil
}

// IsMultiArchSupported checks if the system supports multi-architecture containers
func (d *detector) IsMultiArchSupported() bool {
	// Most modern Docker installations support multi-arch
	// This could be enhanced to actually check Docker buildx availability
	return true
}

// GetContainerName generates a container name with architecture suffix
// Following the pattern from bash: claude-reactor-{variant}-{arch}-{account}
func GetContainerName(variant, account string, archDetector pkg.ArchitectureDetector) (string, error) {
	arch, err := archDetector.GetHostArchitecture()
	if err != nil {
		return "", fmt.Errorf("failed to get architecture for container name: %w", err)
	}
	
	// Base container name parts
	parts := []string{"claude-reactor", variant, arch}
	
	// Add account if specified, otherwise use "default"
	if account != "" {
		parts = append(parts, account)
	} else {
		parts = append(parts, "default")
	}
	
	return strings.Join(parts, "-"), nil
}

// GetImageName generates an image name with architecture suffix
func GetImageName(variant string, archDetector pkg.ArchitectureDetector) (string, error) {
	arch, err := archDetector.GetHostArchitecture()
	if err != nil {
		return "", fmt.Errorf("failed to get architecture for image name: %w", err)
	}
	
	return fmt.Sprintf("claude-reactor-%s:%s", variant, arch), nil
}