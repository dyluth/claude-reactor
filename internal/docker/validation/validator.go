package validation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"

	"claude-reactor/pkg"
)

// RecommendedPackage represents a tool that enhances claude-reactor experience
type RecommendedPackage struct {
	Name        string   // Display name
	Command     string   // Command to test availability
	Description string   // What this tool provides
	Category    string   // Category for grouping
	Priority    string   // high, medium, low
}

// recommendedPackages defines tools that enhance the claude-reactor experience
var recommendedPackages = []RecommendedPackage{
	// Version Control
	{Name: "git", Command: "git --version", Description: "Version control for project management", Category: "version-control", Priority: "high"},
	{Name: "gh", Command: "gh --version", Description: "GitHub CLI for repository operations", Category: "version-control", Priority: "medium"},
	
	// Network Tools
	{Name: "curl", Command: "curl --version", Description: "HTTP client for API requests and downloads", Category: "network", Priority: "high"},
	{Name: "wget", Command: "wget --version", Description: "File downloader for retrieving resources", Category: "network", Priority: "medium"},
	
	// Text Processing
	{Name: "jq", Command: "jq --version", Description: "JSON processor for API responses", Category: "text-processing", Priority: "medium"},
	{Name: "yq", Command: "yq --version", Description: "YAML processor for configuration files", Category: "text-processing", Priority: "low"},
	
	// Build Tools
	{Name: "make", Command: "make --version", Description: "Build automation tool", Category: "build", Priority: "medium"},
	{Name: "docker", Command: "docker --version", Description: "Container runtime (if needed in container)", Category: "build", Priority: "low"},
	
	// Text Editors
	{Name: "nano", Command: "nano --version", Description: "Simple text editor for quick edits", Category: "editors", Priority: "medium"},
	{Name: "vim", Command: "vim --version", Description: "Advanced text editor", Category: "editors", Priority: "low"},
}

// ImageValidator handles Docker image validation and caching
type ImageValidator struct {
	dockerClient client.APIClient
	logger       pkg.Logger
	cacheDir     string
	sessionWarnings map[string]bool // Track warnings shown in this session
}


// NewImageValidator creates a new image validator
func NewImageValidator(dockerClient client.APIClient, logger pkg.Logger) *ImageValidator {
	homeDir, _ := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".claude-reactor", "image-cache")
	
	return &ImageValidator{
		dockerClient: dockerClient,
		logger:       logger,
		cacheDir:     cacheDir,
		sessionWarnings: make(map[string]bool),
	}
}

// ValidateImage validates a Docker image for claude-reactor compatibility
func (v *ImageValidator) ValidateImage(ctx context.Context, imageName string, pullIfNeeded bool) (*pkg.ImageValidationResult, error) {
	v.logger.Debugf("Validating image: %s", imageName)
	
	// Step 1: Ensure image exists locally (pull if needed)
	imageID, err := v.ensureImageExists(ctx, imageName, pullIfNeeded)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure image exists: %w", err)
	}
	
	// Step 2: Get image info and digest
	imageInfo, err := v.dockerClient.ImageInspect(ctx, imageID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect image: %w", err)
	}
	
	digest := v.getImageDigest(imageInfo)
	
	// Step 3: Check cache first
	if cached, err := v.getCachedResult(digest); err == nil && cached != nil {
		v.logger.Debugf("Using cached validation result for image %s (digest: %s)", imageName, digest)
		return cached, nil
	}
	
	// Step 4: Perform validation
	result := &pkg.ImageValidationResult{
		Digest:      digest,
		Architecture: imageInfo.Architecture,
		Platform:    imageInfo.Os,
		Size:        imageInfo.Size,
		ValidatedAt: time.Now().Format(time.RFC3339),
		Warnings:    []string{},
		Errors:      []string{},
		Metadata:    make(map[string]interface{}),
	}
	
	// Step 5: Platform validation
	v.validatePlatform(result)
	
	// Step 6: Check for Claude CLI
	v.validateClaudeCLI(ctx, imageID, result)
	
	// Step 7: Check for recommended packages
	v.checkRecommendedPackages(ctx, imageID, result)
	
	// Step 8: Determine overall compatibility
	result.Compatible = result.IsLinux && result.HasClaude && len(result.Errors) == 0
	
	// Step 9: Cache the result
	if err := v.cacheResult(digest, result); err != nil {
		v.logger.Warnf("Failed to cache validation result: %v", err)
	}
	
	v.logger.Debugf("Image validation complete: compatible=%t, linux=%t, claude=%t", 
		result.Compatible, result.IsLinux, result.HasClaude)
	
	return result, nil
}

// ensureImageExists checks if image exists locally, pulls if needed
func (v *ImageValidator) ensureImageExists(ctx context.Context, imageName string, pullIfNeeded bool) (string, error) {
	// Check if image exists locally
	images, err := v.dockerClient.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list images: %w", err)
	}
	
	// Look for matching image
	for _, image := range images {
		for _, tag := range image.RepoTags {
			if tag == imageName {
				v.logger.Debugf("Image %s found locally (ID: %s)", imageName, image.ID)
				return image.ID, nil
			}
		}
	}
	
	// Image not found locally
	if !pullIfNeeded {
		return "", fmt.Errorf("image %s not found locally and pull not requested", imageName)
	}
	
	// Pull the image
	v.logger.Infof("📦 Pulling image: %s", imageName)
	pullResponse, err := v.dockerClient.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer pullResponse.Close()
	
	// Wait for pull to complete (simplified - in production we'd stream progress)
	// For now, just read the response to completion
	buf := make([]byte, 1024)
	for {
		_, err := pullResponse.Read(buf)
		if err != nil {
			break // EOF or error, either way we're done
		}
	}
	
	// Get the pulled image ID
	images, err = v.dockerClient.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list images after pull: %w", err)
	}
	
	for _, image := range images {
		for _, tag := range image.RepoTags {
			if tag == imageName {
				v.logger.Infof("✅ Image pulled successfully: %s", imageName)
				return image.ID, nil
			}
		}
	}
	
	return "", fmt.Errorf("image %s not found after pull", imageName)
}

// validatePlatform checks if the image is Linux-based
func (v *ImageValidator) validatePlatform(result *pkg.ImageValidationResult) {
	result.IsLinux = strings.ToLower(result.Platform) == "linux"
	
	if !result.IsLinux {
		result.Errors = append(result.Errors, 
			fmt.Sprintf("Unsupported platform: %s. Claude-reactor requires Linux-based images", result.Platform))
	}
	
	// Architecture warnings
	hostArch := "amd64" // TODO: Get from architecture detector
	if result.Architecture != hostArch {
		result.Warnings = append(result.Warnings, 
			fmt.Sprintf("Image architecture (%s) differs from host (%s). This may cause performance issues", 
				result.Architecture, hostArch))
	}
}

// validateClaudeCLI checks if Claude CLI is installed in the image
func (v *ImageValidator) validateClaudeCLI(ctx context.Context, imageID string, result *pkg.ImageValidationResult) {
	// Create a temporary container to test Claude CLI
	containerConfig := &container.Config{
		Image: imageID,
		Cmd:   []string{"claude", "--version"},
		Tty:   false,
	}
	
	containerResp, err := v.dockerClient.ContainerCreate(ctx, containerConfig, nil, nil, nil, "")
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to create test container: %v", err))
		return
	}
	
	// Ensure cleanup
	defer func() {
		v.dockerClient.ContainerRemove(ctx, containerResp.ID, container.RemoveOptions{Force: true})
	}()
	
	// Start the container
	if err := v.dockerClient.ContainerStart(ctx, containerResp.ID, container.StartOptions{}); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to start test container: %v", err))
		return
	}
	
	// Wait for container to finish (with timeout)
	waitCh, errCh := v.dockerClient.ContainerWait(ctx, containerResp.ID, container.WaitConditionNotRunning)
	
	select {
	case waitResp := <-waitCh:
		if waitResp.StatusCode == 0 {
			result.HasClaude = true
			v.logger.Debugf("Claude CLI found and functional in image")
		} else {
			result.HasClaude = false
			result.Errors = append(result.Errors, "Claude CLI not found or not functional. Install Claude CLI in your custom image")
		}
	case err := <-errCh:
		result.Errors = append(result.Errors, fmt.Sprintf("Error checking Claude CLI: %v", err))
	case <-time.After(10 * time.Second):
		result.Errors = append(result.Errors, "Timeout checking Claude CLI")
	}
}

// getImageDigest extracts a consistent digest from image info
func (v *ImageValidator) getImageDigest(imageInfo types.ImageInspect) string {
	// Use image ID as digest - it's already a SHA256 hash
	if imageInfo.ID != "" {
		return imageInfo.ID
	}
	
	// Fallback: create hash from image metadata
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d", 
		imageInfo.Architecture, imageInfo.Os, imageInfo.Size)))
	return hex.EncodeToString(hash[:])
}

// getCachedResult retrieves cached validation result
// Docker image digests are immutable SHA256 hashes - once an image has a specific digest,
// its content will never change. Therefore we can cache validation results for a long time.
func (v *ImageValidator) getCachedResult(digest string) (*pkg.ImageValidationResult, error) {
	if err := os.MkdirAll(v.cacheDir, 0755); err != nil {
		return nil, err
	}
	
	cacheFile := filepath.Join(v.cacheDir, digest+".json")
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, err
	}
	
	var result pkg.ImageValidationResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	
	// Check if cache is still fresh (30 days)
	// Docker image digests are immutable - once validated, they don't change
	validatedAt, err := time.Parse(time.RFC3339, result.ValidatedAt)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp in cache")
	}
	if time.Since(validatedAt) > 30*24*time.Hour {
		return nil, fmt.Errorf("cache expired")
	}
	
	return &result, nil
}

// cacheResult stores validation result in cache
func (v *ImageValidator) cacheResult(digest string, result *pkg.ImageValidationResult) error {
	if err := os.MkdirAll(v.cacheDir, 0755); err != nil {
		return err
	}
	
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	
	cacheFile := filepath.Join(v.cacheDir, digest+".json")
	return os.WriteFile(cacheFile, data, 0644)
}

// checkRecommendedPackages checks for commonly used tools in custom images
func (v *ImageValidator) checkRecommendedPackages(ctx context.Context, imageID string, result *pkg.ImageValidationResult) {
	v.logger.Debugf("Checking recommended packages for image")
	
	availablePackages := []string{}
	missingHighPriority := []string{}
	missingOther := []string{}
	
	// Group packages by priority for better reporting
	for _, pkg := range recommendedPackages {
		available := v.testPackageAvailability(ctx, imageID, pkg)
		
		if available {
			availablePackages = append(availablePackages, pkg.Name)
		} else {
			if pkg.Priority == "high" {
				missingHighPriority = append(missingHighPriority, pkg.Name)
			} else {
				missingOther = append(missingOther, pkg.Name)
			}
		}
	}
	
	// Store package info in metadata
	result.Metadata["packages"] = map[string]interface{}{
		"available": availablePackages,
		"missing_high_priority": missingHighPriority,
		"missing_other": missingOther,
		"total_checked": len(recommendedPackages),
		"total_available": len(availablePackages),
	}
	
	// Add session-aware warnings for missing high-priority packages
	v.addPackageWarnings(result, missingHighPriority, missingOther)
}

// testPackageAvailability tests if a package is available in the image
func (v *ImageValidator) testPackageAvailability(ctx context.Context, imageID string, pkg RecommendedPackage) bool {
	// Create a lightweight container to test package availability
	containerConfig := &container.Config{
		Image: imageID,
		Cmd:   []string{"sh", "-c", pkg.Command + " >/dev/null 2>&1 && echo 'available' || echo 'missing'"},
		Tty:   false,
	}
	
	containerResp, err := v.dockerClient.ContainerCreate(ctx, containerConfig, nil, nil, nil, "")
	if err != nil {
		v.logger.Debugf("Failed to create test container for package %s: %v", pkg.Name, err)
		return false
	}
	
	// Ensure cleanup
	defer func() {
		v.dockerClient.ContainerRemove(ctx, containerResp.ID, container.RemoveOptions{Force: true})
	}()
	
	// Start the container
	if err := v.dockerClient.ContainerStart(ctx, containerResp.ID, container.StartOptions{}); err != nil {
		v.logger.Debugf("Failed to start test container for package %s: %v", pkg.Name, err)
		return false
	}
	
	// Wait for container to finish (with short timeout)
	waitCh, errCh := v.dockerClient.ContainerWait(ctx, containerResp.ID, container.WaitConditionNotRunning)
	
	select {
	case waitResp := <-waitCh:
		// Success (exit code 0) means package is available
		return waitResp.StatusCode == 0
	case err := <-errCh:
		v.logger.Debugf("Error checking package %s: %v", pkg.Name, err)
		return false
	case <-time.After(5 * time.Second):
		v.logger.Debugf("Timeout checking package %s", pkg.Name)
		return false
	}
}

// addPackageWarnings adds session-aware warnings for missing packages
func (v *ImageValidator) addPackageWarnings(result *pkg.ImageValidationResult, missingHighPriority, missingOther []string) {
	// Only warn about high-priority missing packages
	if len(missingHighPriority) > 0 {
		// Create a session key for this warning
		sessionKey := fmt.Sprintf("missing-packages-%s", strings.Join(missingHighPriority, ","))
		
		// Only show warning once per session
		if !v.sessionWarnings[sessionKey] {
			warningMsg := fmt.Sprintf("Missing recommended tools: %s. These enhance the development experience", 
				strings.Join(missingHighPriority, ", "))
			result.Warnings = append(result.Warnings, warningMsg)
			
			// Mark this warning as shown for this session
			v.sessionWarnings[sessionKey] = true
		}
	}
	
	// Add informational note about total packages checked
	if len(missingHighPriority) + len(missingOther) > 0 {
		total := len(missingHighPriority) + len(missingOther)
		infoMsg := fmt.Sprintf("Package check: %d recommended tools missing (use 'docker exec <container> <tool> --version' to verify)", total)
		result.Warnings = append(result.Warnings, infoMsg)
	}
}

// ClearSessionWarnings resets session warning tracking
func (v *ImageValidator) ClearSessionWarnings() {
	v.sessionWarnings = make(map[string]bool)
}

// ClearCache removes all cached validation results
func (v *ImageValidator) ClearCache() error {
	if _, err := os.Stat(v.cacheDir); os.IsNotExist(err) {
		return nil // Cache directory doesn't exist, nothing to clear
	}
	
	return os.RemoveAll(v.cacheDir)
}