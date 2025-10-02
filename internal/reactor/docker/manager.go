package docker

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"crypto/rand"
	"math/big"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/moby/term"
	
	"claude-reactor/pkg"
)

// manager implements the DockerManager interface
type manager struct {
	client client.APIClient
	logger pkg.Logger
}

// NewManager creates a new Docker manager with Docker client
func NewManager(logger pkg.Logger) (pkg.DockerManager, error) {
	// Initialize Docker client from environment with API version negotiation
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	
	logger.Debug("Docker client initialized successfully")
	
	// Validate Docker connection
	ctx := context.Background()
	_, err = cli.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Docker daemon: %w", err)
	}
	
	logger.Debug("Docker daemon connection validated")
	
	return &manager{
		client: cli,
		logger: logger,
	}, nil
}

// BuildImage builds a Docker image for the specified variant
func (m *manager) BuildImage(ctx context.Context, variant string, platform string) error {
	m.logger.Infof("Building Docker image: variant=%s, platform=%s", variant, platform)
	
	// Create VariantManager for validation
	variantMgr := NewVariantManager(m.logger)
	
	// Validate variant before proceeding
	if err := variantMgr.ValidateVariant(variant); err != nil {
		return fmt.Errorf("invalid variant: %w", err)
	}
	
	m.logger.Debugf("Using variant: %s (%s)", variant, variantMgr.GetVariantDescription(variant))
	
	// Create NamingManager for image name generation
	archDetector := &basicArchDetector{}
	namingMgr := NewNamingManager(m.logger, archDetector)
	
	imageName, err := namingMgr.GetImageName(variant)
	if err != nil {
		return fmt.Errorf("failed to generate image name: %w", err)
	}
	
	// Find project root directory (where Dockerfile is located)
	projectRoot, err := m.findProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}
	
	// Create build context from project root directory
	buildContext, err := m.createBuildContext(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to create build context: %w", err)
	}
	defer buildContext.Close()
	
	// Build image with Docker SDK
	buildOptions := types.ImageBuildOptions{
		Tags:       []string{imageName + ":latest"},
		Target:     variant,
		Platform:   platform,
		Dockerfile: "Dockerfile", // Use the main Dockerfile
		Remove:     true,
		ForceRemove: true,
	}
	
	m.logger.Debugf("Starting Docker build with options: %+v", buildOptions)
	
	buildResponse, err := m.client.ImageBuild(ctx, buildContext, buildOptions)
	if err != nil {
		return fmt.Errorf("failed to build Docker image: %w", err)
	}
	defer buildResponse.Body.Close()
	
	// Stream build output and check for errors
	if err := m.streamBuildOutput(buildResponse.Body); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}
	
	m.logger.Infof("Successfully built image: %s", imageName)
	return nil
}

// StartContainer starts a container with the given configuration
func (m *manager) StartContainer(ctx context.Context, config *pkg.ContainerConfig) (string, error) {
	m.logger.Infof("Starting container: %s", config.Name)
	
	// Create mount manager for handling mounts
	mountMgr := NewMountManager(m.logger)
	
	// Use mounts from config, or create default mounts if nil (not explicitly set to empty)
	configMounts := config.Mounts
	if config.Mounts == nil {
		m.logger.Debugf("No mounts specified in config, creating default mounts")
		// Extract account from container name if present
		// Container names follow: claude-reactor-variant-arch-hash-account
		account := "default"
		nameParts := strings.Split(config.Name, "-")
		if len(nameParts) >= 6 {
			account = nameParts[5]
		}
		
		var err error
		configMounts, err = mountMgr.CreateDefaultMounts(account)
		if err != nil {
			return "", fmt.Errorf("failed to create default mounts: %w", err)
		}
	}
	
	// Validate mounts before proceeding
	if err := mountMgr.ValidateMounts(configMounts); err != nil {
		m.logger.Warnf("Mount validation failed: %v", err)
		// Continue anyway, as some mounts may be optional
	}
	
	// Convert pkg.Mount to Docker SDK mount.Mount
	mounts := mountMgr.ConvertToDockerMounts(configMounts)
	
	// Log mount summary
	if len(mounts) > 0 {
		summary := mountMgr.GetMountSummary(configMounts)
		m.logger.Debugf("Container mounts:")
		for _, mount := range summary {
			m.logger.Debugf("  %s", mount)
		}
	}
	
	// Convert environment map to []string
	env := make([]string, 0, len(config.Environment))
	for key, value := range config.Environment {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	
	// Create container configuration
	containerConfig := &container.Config{
		Image:      config.Image,
		Env:        env,
		Cmd:        config.Command,
		Tty:        config.TTY,
		OpenStdin:  config.Interactive,
		StdinOnce:  config.Interactive,
		AttachStdin: config.Interactive,
		AttachStdout: true,
		AttachStderr: true,
	}
	
	// Create host configuration
	hostConfig := &container.HostConfig{
		Mounts:      mounts,
		AutoRemove:  false, // We'll manage removal manually
		NetworkMode: "bridge", // Default network mode
	}
	
	// Create container
	m.logger.Debugf("Creating container with image: %s", config.Image)
	resp, err := m.client.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, config.Name)
	if err != nil {
		// Check for SSH agent socket mounting issues (common with Docker Desktop on macOS)
		if strings.Contains(err.Error(), "socket_mnt") && strings.Contains(err.Error(), "bind source path does not exist") {
			return "", fmt.Errorf("failed to create container: SSH agent socket mounting failed\n"+
				"💡 This is a known issue with Docker Desktop on macOS\n"+
				"💡 Try without SSH agent: remove --ssh-agent flag\n"+
				"💡 Or use a different SSH agent socket path\n"+
				"Original error: %w", err)
		}
		return "", fmt.Errorf("failed to create container: %w", err)
	}
	
	// Start container
	m.logger.Debugf("Starting container with ID: %s", resp.ID)
	if err := m.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		// Clean up the created container if start fails
		m.client.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
		return "", fmt.Errorf("failed to start container: %w", err)
	}
	
	m.logger.Infof("Successfully started container: %s (ID: %s)", config.Name, resp.ID[:12])
	
	// Ensure required directories exist in container
	if err := m.ensureContainerDirectories(ctx, resp.ID); err != nil {
		m.logger.Warnf("Failed to create container directories (non-fatal): %v", err)
		// Don't fail container startup for directory creation issues
	}
	
	// Run claude upgrade if requested and container has claude CLI
	if config.RunClaudeUpgrade {
		m.logger.Info("Running claude upgrade in container...")
		if err := m.runClaudeUpgrade(ctx, resp.ID); err != nil {
			m.logger.Warnf("Claude upgrade failed (non-fatal): %v", err)
			// Don't fail container startup for upgrade issues
		}
	}
	
	return resp.ID, nil
}

// runClaudeUpgrade executes claude upgrade in the container after startup
func (m *manager) runClaudeUpgrade(ctx context.Context, containerID string) error {
	// Wait a moment for container to be fully ready
	time.Sleep(2 * time.Second)
	
	// Create exec configuration using container package types
	execConfig := container.ExecOptions{
		Cmd:    []string{"claude", "upgrade"},
		Detach: true, // Run in background
	}
	
	// Create exec instance
	execResp, err := m.client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		// Claude CLI may not be available in this container - not a fatal error
		m.logger.Debugf("Could not create claude upgrade exec (claude CLI may not be available): %v", err)
		return nil
	}
	
	// Start exec (fire and forget - don't wait for completion) 
	err = m.client.ContainerExecStart(ctx, execResp.ID, container.ExecStartOptions{
		Detach: true,
	})
	if err != nil {
		m.logger.Debugf("Could not start claude upgrade (non-fatal): %v", err)
		return nil
	}
	
	m.logger.Info("✅ Claude upgrade initiated in container")
	return nil
}

// ensureContainerDirectories creates required directories in the container
func (m *manager) ensureContainerDirectories(ctx context.Context, containerID string) error {
	// Create .claude-reactor directory to prevent "Path not found" errors
	// This directory is needed by Claude CLI for authentication and config operations
	execConfig := container.ExecOptions{
		Cmd:    []string{"mkdir", "-p", "/home/claude/.claude-reactor"},
		Detach: false,
	}
	
	// Create exec instance
	execResp, err := m.client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create directory creation exec: %w", err)
	}
	
	// Start exec and wait for completion
	err = m.client.ContainerExecStart(ctx, execResp.ID, container.ExecStartOptions{
		Detach: false,
	})
	if err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}
	
	m.logger.Debugf("✅ Created required directories in container")
	return nil
}

// StopContainer stops a running container
func (m *manager) StopContainer(ctx context.Context, containerID string) error {
	m.logger.Infof("Stopping container: %s", containerID[:12])
	
	// Set a reasonable timeout for graceful shutdown (30 seconds)
	timeout := 30
	stopOptions := container.StopOptions{
		Timeout: &timeout,
	}
	
	if err := m.client.ContainerStop(ctx, containerID, stopOptions); err != nil {
		// Check if container is already stopped
		if strings.Contains(err.Error(), "container already stopped") ||
		   strings.Contains(err.Error(), "not running") {
			m.logger.Debugf("Container %s is already stopped", containerID[:12])
			return nil
		}
		return fmt.Errorf("failed to stop container %s: %w", containerID[:12], err)
	}
	
	m.logger.Infof("Successfully stopped container: %s", containerID[:12])
	return nil
}

// RemoveContainer removes a stopped container
func (m *manager) RemoveContainer(ctx context.Context, containerID string) error {
	m.logger.Infof("Removing container: %s", containerID[:12])
	
	removeOptions := container.RemoveOptions{
		RemoveVolumes: true,  // Remove associated volumes
		Force:         false, // Don't force remove running containers by default
	}
	
	if err := m.client.ContainerRemove(ctx, containerID, removeOptions); err != nil {
		// Check if container is still running and needs force removal
		if strings.Contains(err.Error(), "cannot remove a running container") {
			m.logger.Warnf("Container %s is still running, forcing removal", containerID[:12])
			removeOptions.Force = true
			if err := m.client.ContainerRemove(ctx, containerID, removeOptions); err != nil {
				return fmt.Errorf("failed to force remove container %s: %w", containerID[:12], err)
			}
		} else if strings.Contains(err.Error(), "no such container") {
			m.logger.Debugf("Container %s is already removed", containerID[:12])
			return nil
		} else {
			return fmt.Errorf("failed to remove container %s: %w", containerID[:12], err)
		}
	}
	
	m.logger.Infof("Successfully removed container: %s", containerID[:12])
	return nil
}

// IsContainerRunning checks if a container is currently running
func (m *manager) IsContainerRunning(ctx context.Context, containerName string) (bool, error) {
	m.logger.Debugf("Checking if container is running: %s", containerName)
	
	// List containers with the given name
	containers, err := m.client.ContainerList(ctx, container.ListOptions{
		All: true, // Include stopped containers too
	})
	if err != nil {
		return false, fmt.Errorf("failed to list containers: %w", err)
	}
	
	// Look for container with matching name
	for _, container := range containers {
		for _, name := range container.Names {
			// Docker container names start with "/", so we need to trim it
			cleanName := strings.TrimPrefix(name, "/")
			if cleanName == containerName {
				// Check if container is running
				return container.State == "running", nil
			}
		}
	}
	
	// Container not found
	return false, nil
}

// GetContainerLogs retrieves logs from a container
func (m *manager) GetContainerLogs(ctx context.Context, containerID string) (io.ReadCloser, error) {
	// TODO: Implement container log retrieval with Docker SDK
	m.logger.Debugf("Getting logs for container: %s", containerID)
	return nil, nil // Stub implementation
}

// createBuildContext creates a tar archive of the build context
func (m *manager) createBuildContext(contextDir string) (io.ReadCloser, error) {
	// Create a pipe for the tar stream
	pr, pw := io.Pipe()
	
	go func() {
		defer pw.Close()
		
		tarWriter := tar.NewWriter(pw)
		defer tarWriter.Close()
		
		err := filepath.Walk(contextDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			
			// Skip unwanted files and directories
			relPath, err := filepath.Rel(contextDir, path)
			if err != nil {
				return err
			}
			
			// Skip .git, dist, and other unwanted directories
			if m.shouldSkipPath(relPath) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			
			// Create tar header
			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}
			header.Name = relPath
			
			// Write header
			if err := tarWriter.WriteHeader(header); err != nil {
				return err
			}
			
			// Write file content if it's a regular file
			if info.Mode().IsRegular() {
				file, err := os.Open(path)
				if err != nil {
					return err
				}
				defer file.Close()
				
				if _, err := io.Copy(tarWriter, file); err != nil {
					return err
				}
			}
			
			return nil
		})
		
		if err != nil {
			pw.CloseWithError(err)
		}
	}()
	
	return pr, nil
}

// shouldSkipPath determines if a path should be skipped in build context
func (m *manager) shouldSkipPath(path string) bool {
	// Exact matches
	if path == ".git" || path == "dist" || path == ".claude-reactor" || path == "tests/results" {
		return true
	}
	
	// Prefix matches
	prefixPatterns := []string{
		".git/",
		".claude-reactor-",
		"dist/",
		"tests/results/",
		"tests/fixtures/",
		"claude-reactor-go",
		"node_modules",
	}
	
	for _, prefix := range prefixPatterns {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	
	// Suffix matches
	suffixPatterns := []string{
		".DS_Store",
		"Thumbs.db",
	}
	
	for _, suffix := range suffixPatterns {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	
	// Contains matches
	if strings.Contains(path, "/node_modules") {
		return true
	}
	
	return false
}

// streamBuildOutput streams Docker build output and parses for errors
func (m *manager) streamBuildOutput(reader io.Reader) error {
	decoder := json.NewDecoder(reader)
	
	for decoder.More() {
		var message map[string]interface{}
		if err := decoder.Decode(&message); err != nil {
			return fmt.Errorf("failed to decode build output: %w", err)
		}
		
		// Check for error message
		if errMsg, ok := message["error"].(string); ok {
			return fmt.Errorf("build error: %s", errMsg)
		}
		
		// Log stream messages
		if stream, ok := message["stream"].(string); ok {
			stream = strings.TrimSpace(stream)
			if stream != "" {
				m.logger.Debugf("Build: %s", stream)
			}
		}
	}
	
	return nil
}

// basicArchDetector is a simple implementation for internal use
type basicArchDetector struct{}

func (d *basicArchDetector) GetHostArchitecture() (string, error) {
	// Use the same logic as architecture package
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		return "amd64", nil
	case "arm64":
		return "arm64", nil
	default:
		return arch, nil
	}
}

func (d *basicArchDetector) GetDockerPlatform() (string, error) {
	arch, err := d.GetHostArchitecture()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("linux/%s", arch), nil
}

func (d *basicArchDetector) IsMultiArchSupported() bool {
	return true
}

// RebuildImage forces rebuild of Docker image
func (m *manager) RebuildImage(ctx context.Context, variant string, platform string, force bool) error {
	if force {
		// Remove existing image first
		imageName := m.GetImageName(variant, "")
		_, err := m.client.ImageRemove(ctx, imageName, image.RemoveOptions{
			Force:         true,
			PruneChildren: true,
		})
		if err != nil {
			m.logger.Warnf("Failed to remove existing image: %v", err)
		}
	}
	
	return m.BuildImage(ctx, variant, platform)
}

// GetContainerStatus returns detailed container status information
func (m *manager) GetContainerStatus(ctx context.Context, containerName string) (*pkg.ContainerStatus, error) {
	containers, err := m.client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}
	
	for _, container := range containers {
		for _, name := range container.Names {
			// Docker container names start with '/'
			if strings.TrimPrefix(name, "/") == containerName {
				return &pkg.ContainerStatus{
					Exists:  true,
					Running: container.State == "running",
					Name:    containerName,
					Image:   container.Image,
					ID:      container.ID,
				}, nil
			}
		}
	}
	
	return &pkg.ContainerStatus{
		Exists:  false,
		Running: false,
		Name:    containerName,
	}, nil
}

// CleanContainer removes specific project/account container
func (m *manager) CleanContainer(ctx context.Context, containerName string) error {
	status, err := m.GetContainerStatus(ctx, containerName)
	if err != nil {
		return fmt.Errorf("failed to get container status: %w", err)
	}
	
	if !status.Exists {
		m.logger.Debugf("Container %s does not exist", containerName)
		return nil
	}
	
	if status.Running {
		if err := m.StopContainer(ctx, status.ID); err != nil {
			return fmt.Errorf("failed to stop container: %w", err)
		}
	}
	
	return m.RemoveContainer(ctx, status.ID)
}

// CleanAllContainers removes all claude-reactor containers
func (m *manager) CleanAllContainers(ctx context.Context) error {
	containers, err := m.client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}
	
	var errors []error
	for _, container := range containers {
		for _, name := range container.Names {
			containerName := strings.TrimPrefix(name, "/")
			if strings.HasPrefix(containerName, "claude-reactor-") {
				m.logger.Infof("Removing container: %s", containerName)
				if err := m.CleanContainer(ctx, containerName); err != nil {
					errors = append(errors, fmt.Errorf("failed to clean %s: %w", containerName, err))
				}
			}
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("errors during cleanup: %v", errors)
	}
	
	return nil
}

// AttachToContainer executes commands in a running container using Docker SDK exec
func (m *manager) AttachToContainer(ctx context.Context, containerName string, command []string, interactive bool) error {
	m.logger.Debugf("Attaching to container %s with command: %v (interactive: %t)", containerName, command, interactive)
	
	// Get container ID from name
	containerID, err := m.getContainerIDByName(ctx, containerName)
	if err != nil {
		return fmt.Errorf("failed to find container %s: %w", containerName, err)
	}
	
	if interactive {
		return m.attachInteractive(ctx, containerID, command)
	} else {
		return m.attachNonInteractive(ctx, containerID, command)
	}
}

// attachInteractive handles interactive container attachment with TTY
func (m *manager) attachInteractive(ctx context.Context, containerID string, command []string) error {
	m.logger.Debugf("Creating interactive exec for container %s", containerID[:12])
	
	// Create exec configuration
	execConfig := container.ExecOptions{
		Cmd:          command,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
	}
	
	// Create exec instance
	execResp, err := m.client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create exec instance: %w", err)
	}
	
	// Start exec with hijacked connection for interactive I/O
	execStartCheck := container.ExecStartOptions{
		Tty: true,
	}
	
	hijackedResp, err := m.client.ContainerExecAttach(ctx, execResp.ID, execStartCheck)
	if err != nil {
		return fmt.Errorf("failed to attach to exec instance: %w", err)
	}
	defer hijackedResp.Close()
	
	// Start the exec instance
	if err := m.client.ContainerExecStart(ctx, execResp.ID, execStartCheck); err != nil {
		return fmt.Errorf("failed to start exec instance: %w", err)
	}
	
	m.logger.Info("✅ Successfully attached to container - press Ctrl+C to disconnect")
	
	// Set up proper terminal handling for keystroke interpretation
	fd := os.Stdin.Fd()
	var oldState *term.State
	
	// Check if stdin is a terminal and set raw mode
	if term.IsTerminal(fd) {
		oldState, err = term.MakeRaw(fd)
		if err != nil {
			m.logger.Warnf("Failed to set terminal to raw mode: %v", err)
		} else {
			// Ensure we restore terminal state on exit
			defer func() {
				if oldState != nil {
					term.RestoreTerminal(fd, oldState)
				}
			}()
		}
		
		// Sync terminal size to prevent display issues
		if err := m.resizeContainerTTY(ctx, execResp.ID); err != nil {
			m.logger.Debugf("Failed to resize container TTY: %v", err)
		}
	}
	
	// Set up signal handling to properly restore terminal on interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	// Handle I/O with proper error handling and buffering
	inputDone := make(chan error, 1)
	outputDone := make(chan error, 1)
	
	// Set up terminal resize handling to prevent display issues
	if term.IsTerminal(fd) {
		resizeChan := make(chan os.Signal, 1)
		signal.Notify(resizeChan, syscall.SIGWINCH)
		
		go func() {
			for range resizeChan {
				if err := m.resizeContainerTTY(ctx, execResp.ID); err != nil {
					m.logger.Debugf("Failed to resize container TTY: %v", err)
				}
			}
		}()
	}
	
	// Copy output from container to stdout with improved buffering
	go func() {
		// Use a buffered copy to reduce flickering
		buf := make([]byte, 1024)
		for {
			n, err := hijackedResp.Reader.Read(buf)
			if n > 0 {
				if _, writeErr := os.Stdout.Write(buf[:n]); writeErr != nil {
					outputDone <- writeErr
					return
				}
			}
			if err != nil {
				outputDone <- err
				return
			}
		}
	}()
	
	// Copy input from stdin to container with improved buffering
	go func() {
		_, err := io.Copy(hijackedResp.Conn, os.Stdin)
		inputDone <- err
	}()
	
	// Wait for completion or signal
	select {
	case <-sigChan:
		m.logger.Debug("Received interrupt signal, disconnecting...")
		// Restore terminal state before exiting
		if oldState != nil {
			term.RestoreTerminal(fd, oldState)
		}
		return fmt.Errorf("interrupted by user")
		
	case err := <-inputDone:
		if err != nil {
			m.logger.Debugf("Input stream ended: %v", err)
		}
		
	case err := <-outputDone:
		if err != nil {
			m.logger.Debugf("Output stream ended: %v", err)
		}
	}
	
	// Check if exec is still running
	inspectResp, err := m.client.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return fmt.Errorf("failed to inspect exec instance: %w", err)
	}
	
	if inspectResp.Running {
		m.logger.Debug("Exec still running, waiting for completion...")
		// Give it a moment to complete
		time.Sleep(1 * time.Second)
	}
	
	m.logger.Debug("Interactive session completed")
	return nil
}

// resizeContainerTTY synchronizes the container's TTY size with the host terminal
func (m *manager) resizeContainerTTY(ctx context.Context, execID string) error {
	fd := os.Stdin.Fd()
	if !term.IsTerminal(fd) {
		return nil
	}
	
	// Get current terminal size
	ws, err := term.GetWinsize(fd)
	if err != nil {
		return fmt.Errorf("failed to get terminal size: %w", err)
	}
	
	// Resize the container's TTY
	resizeOptions := container.ResizeOptions{
		Height: uint(ws.Height),
		Width:  uint(ws.Width),
	}
	
	if err := m.client.ContainerExecResize(ctx, execID, resizeOptions); err != nil {
		return fmt.Errorf("failed to resize container TTY: %w", err)
	}
	
	return nil
}

// attachNonInteractive handles non-interactive command execution
func (m *manager) attachNonInteractive(ctx context.Context, containerID string, command []string) error {
	m.logger.Debugf("Creating non-interactive exec for container %s", containerID[:12])
	
	// Create exec configuration
	execConfig := container.ExecOptions{
		Cmd:          command,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
	}
	
	// Create exec instance
	execResp, err := m.client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create exec instance: %w", err)
	}
	
	// Start exec
	execStartCheck := container.ExecStartOptions{
		Tty: false,
	}
	
	hijackedResp, err := m.client.ContainerExecAttach(ctx, execResp.ID, execStartCheck)
	if err != nil {
		return fmt.Errorf("failed to attach to exec instance: %w", err)
	}
	defer hijackedResp.Close()
	
	// Start the exec instance
	if err := m.client.ContainerExecStart(ctx, execResp.ID, execStartCheck); err != nil {
		return fmt.Errorf("failed to start exec instance: %w", err)
	}
	
	// Copy output to stdout/stderr
	io.Copy(os.Stdout, hijackedResp.Reader)
	
	// Wait for completion and get exit code
	inspectResp, err := m.client.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return fmt.Errorf("failed to inspect exec instance: %w", err)
	}
	
	if inspectResp.ExitCode != 0 {
		return fmt.Errorf("command exited with code %d", inspectResp.ExitCode)
	}
	
	return nil
}

// getContainerIDByName retrieves container ID by name
func (m *manager) getContainerIDByName(ctx context.Context, containerName string) (string, error) {
	containers, err := m.client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}
	
	for _, cont := range containers {
		for _, name := range cont.Names {
			// Container names include leading slash
			if strings.TrimPrefix(name, "/") == containerName {
				return cont.ID, nil
			}
		}
	}
	
	return "", fmt.Errorf("container %s not found", containerName)
}

// HealthCheck verifies container is healthy and responsive
func (m *manager) HealthCheck(ctx context.Context, containerName string, maxRetries int) error {
	// TODO: Implement container health checking with retries
	m.logger.Debugf("Checking health of container %s (max retries: %d)", containerName, maxRetries)
	return nil // Stub implementation
}

// ListVariants returns available container variants
func (m *manager) ListVariants() ([]pkg.VariantDefinition, error) {
	// Return hard-coded variant definitions for now
	variants := []pkg.VariantDefinition{
		{
			Name:        "base",
			Description: "Node.js, Python with pip + uv, basic development tools",
			Size:        "~500MB",
			Tools:       []string{"node", "python", "pip", "uv", "git", "curl"},
		},
		{
			Name:        "go",
			Description: "Base + Go toolchain and development utilities",
			Size:        "~800MB", 
			Tools:       []string{"node", "python", "pip", "uv", "git", "curl", "go", "delve"},
		},
		{
			Name:        "full",
			Description: "Go + Rust, Java, database clients",
			Size:        "~1.2GB",
			Tools:       []string{"node", "python", "go", "rust", "java", "maven", "psql", "mysql"},
		},
		{
			Name:        "cloud",
			Description: "Full + AWS/GCP/Azure CLIs",
			Size:        "~1.5GB",
			Tools:       []string{"aws", "gcloud", "az", "terraform", "kubectl"},
		},
		{
			Name:        "k8s",
			Description: "Full + Enhanced Kubernetes tools",
			Size:        "~1.4GB",
			Tools:       []string{"kubectl", "helm", "k9s", "stern", "kubectx"},
		},
	}
	
	return variants, nil
}

// GenerateContainerName creates unique container name with project hash
func (m *manager) GenerateContainerName(projectPath, variant, architecture, account string) string {
	namingMgr := NewNamingManager(m.logger, &basicArchDetector{})
	containerName, err := namingMgr.GetContainerName(variant, account)
	if err != nil {
		m.logger.Errorf("Failed to generate container name: %v", err)
		return fmt.Sprintf("claude-reactor-%s-%s", variant, account)
	}
	return containerName
}

// GenerateProjectHash creates hash for project directory
func (m *manager) GenerateProjectHash(projectPath string) string {
	namingMgr := NewNamingManager(m.logger, &basicArchDetector{})
	if projectPath == "" {
		projectPath, _ = os.Getwd()
	}
	hash, err := namingMgr.GetProjectHashFromPath(projectPath)
	if err != nil {
		m.logger.Errorf("Failed to generate project hash: %v", err)
		return "default"
	}
	return hash
}

// GetImageName generates image name with architecture
func (m *manager) GetImageName(variant, architecture string) string {
	namingMgr := NewNamingManager(m.logger, &basicArchDetector{})
	imageName, _ := namingMgr.GetImageName(variant)
	return imageName
}

// CleanImages removes claude-reactor images
func (m *manager) CleanImages(ctx context.Context, all bool) error {
	// List all images
	images, err := m.client.ImageList(ctx, image.ListOptions{All: true})
	if err != nil {
		return fmt.Errorf("failed to list images: %w", err)
	}
	
	var cleanedCount int
	var errors []error
	
	for _, img := range images {
		// Check if this is a claude-reactor image
		isClaudeImage := false
		
		// Check repository tags
		for _, tag := range img.RepoTags {
			if strings.HasPrefix(tag, "claude-reactor-") {
				isClaudeImage = true
				break
			}
		}
		
		// Also check repository digests if no tags
		if !isClaudeImage && len(img.RepoTags) == 0 {
			for _, digest := range img.RepoDigests {
				if strings.Contains(digest, "claude-reactor") {
					isClaudeImage = true
					break
				}
			}
		}
		
		// Skip if not a claude-reactor image and we're not cleaning all
		if !isClaudeImage && !all {
			continue
		}
		
		// Skip if not a claude-reactor image and we are only cleaning claude-reactor images
		if !isClaudeImage {
			continue
		}
		
		m.logger.Infof("Removing image: %s", img.ID[:12])
		
		// Remove the image
		_, err := m.client.ImageRemove(ctx, img.ID, image.RemoveOptions{
			Force:         false, // Don't force remove by default
			PruneChildren: true,  // Remove untagged parent images
		})
		
		if err != nil {
			if strings.Contains(err.Error(), "image is being used") {
				m.logger.Warnf("Image %s is being used by containers, skipping", img.ID[:12])
			} else {
				errors = append(errors, fmt.Errorf("failed to remove image %s: %w", img.ID[:12], err))
			}
		} else {
			cleanedCount++
		}
	}
	
	m.logger.Infof("✅ Cleaned %d images", cleanedCount)
	
	if len(errors) > 0 {
		return fmt.Errorf("encountered %d errors during image cleanup: %v", len(errors), errors)
	}
	
	return nil
}

// findProjectRoot finds the directory containing the Dockerfile
func (m *manager) findProjectRoot() (string, error) {
	// Try to find Dockerfile relative to the binary location first
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		dockerfilePath := filepath.Join(execDir, "Dockerfile")
		if _, err := os.Stat(dockerfilePath); err == nil {
			m.logger.Debugf("Found Dockerfile at binary location: %s", execDir)
			return execDir, nil
		}
	}
	
	// Start from current directory and walk up
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}
	
	// Walk up the directory tree looking for Dockerfile
	dir := currentDir
	for {
		dockerfilePath := filepath.Join(dir, "Dockerfile")
		if _, err := os.Stat(dockerfilePath); err == nil {
			// Found Dockerfile
			m.logger.Debugf("Found Dockerfile in directory tree: %s", dir)
			return dir, nil
		}
		
		// Check if we've reached the root
		parent := filepath.Dir(dir)
		if parent == dir {
			// We've reached the root directory
			break
		}
		dir = parent
	}
	
	// Fallback: Try common project locations
	possibleRoots := []string{"/app", ".", filepath.Dir(currentDir)}
	for _, root := range possibleRoots {
		dockerfilePath := filepath.Join(root, "Dockerfile")
		if _, err := os.Stat(dockerfilePath); err == nil {
			m.logger.Debugf("Found Dockerfile at fallback location: %s", root)
			return root, nil
		}
	}
	
	// Final fallback to current directory
	m.logger.Warnf("Dockerfile not found anywhere, using current directory: %s", currentDir)
	return currentDir, nil
}

// Registry support functions for Phase 0.1

// shouldUseRegistry determines if registry should be used based on flags and environment
func (m *manager) shouldUseRegistry(devMode, registryOff bool) bool {
	if devMode || registryOff {
		return false
	}
	
	// Check environment variable
	useRegistry := os.Getenv("CLAUDE_REACTOR_USE_REGISTRY")
	if useRegistry == "false" || useRegistry == "0" {
		return false
	}
	
	// Default to true
	return true
}

// getRegistryImageName generates the registry image name
func (m *manager) getRegistryImageName(variant string) string {
	registry := os.Getenv("CLAUDE_REACTOR_REGISTRY")
	if registry == "" {
		registry = "ghcr.io/dyluth/claude-reactor"
	}
	
	tag := os.Getenv("CLAUDE_REACTOR_TAG")
	if tag == "" {
		tag = "latest"
	}
	
	return fmt.Sprintf("%s/%s:%s", registry, variant, tag)
}

// tryPullFromRegistry attempts to pull an image from registry
func (m *manager) tryPullFromRegistry(ctx context.Context, variant string) error {
	registryImageName := m.getRegistryImageName(variant)
	
	m.logger.Infof("📦 Attempting to pull %s variant from registry...", variant)
	m.logger.Debugf("Registry image: %s", registryImageName)
	
	// Pull from registry
	pullResponse, err := m.client.ImagePull(ctx, registryImageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull from registry: %w", err)
	}
	defer pullResponse.Close()
	
	// Read and discard the pull output (we could stream it if verbose)
	_, err = io.Copy(io.Discard, pullResponse)
	if err != nil {
		return fmt.Errorf("failed to complete registry pull: %w", err)
	}
	
	// Get the local image name
	namingMgr := NewNamingManager(m.logger, &basicArchDetector{})
	localImageName, err := namingMgr.GetImageName(variant)
	if err != nil {
		return fmt.Errorf("failed to generate local image name: %w", err)
	}
	
	// Tag the pulled image with local naming convention
	err = m.client.ImageTag(ctx, registryImageName, localImageName+":latest")
	if err != nil {
		return fmt.Errorf("failed to tag pulled image: %w", err)
	}
	
	err = m.client.ImageTag(ctx, registryImageName, localImageName)
	if err != nil {
		return fmt.Errorf("failed to tag pulled image: %w", err)
	}
	
	m.logger.Infof("✅ Successfully pulled %s variant from registry", variant)
	return nil
}

// imageExistsLocally checks if an image exists locally
func (m *manager) imageExistsLocally(ctx context.Context, imageName string) (bool, error) {
	_, _, err := m.client.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		if client.IsErrNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// BuildImageWithRegistry builds an image with registry support
func (m *manager) BuildImageWithRegistry(ctx context.Context, variant, platform string, devMode, registryOff, pullLatest bool) error {
	// Get image name
	namingMgr := NewNamingManager(m.logger, &basicArchDetector{})
	imageName, err := namingMgr.GetImageName(variant)
	if err != nil {
		return fmt.Errorf("failed to generate image name: %w", err)
	}
	
	// Force pull if requested
	if pullLatest {
		m.logger.Info("⬇️ Force pulling latest image from registry...")
		// Remove local image first
		_, err := m.client.ImageRemove(ctx, imageName, image.RemoveOptions{Force: true})
		if err != nil && !client.IsErrNotFound(err) {
			m.logger.Debugf("Could not remove local image for force pull: %v", err)
		}
	}
	
	// Check if image exists locally (unless force pull)
	if !pullLatest {
		exists, err := m.imageExistsLocally(ctx, imageName)
		if err != nil {
			return fmt.Errorf("failed to check if image exists: %w", err)
		}
		if exists {
			m.logger.Debugf("Image %s already exists locally", imageName)
			return nil
		}
	}
	
	// Try registry first if enabled
	if m.shouldUseRegistry(devMode, registryOff) {
		err := m.tryPullFromRegistry(ctx, variant)
		if err != nil {
			m.logger.Infof("❌ Failed to pull from registry: %v", err)
			m.logger.Info("🔨 Falling back to local build...")
		} else {
			// Successfully pulled from registry
			return nil
		}
	} else if devMode {
		m.logger.Info("🔨 Dev mode enabled - building locally")
	} else if registryOff {
		m.logger.Info("🔨 Registry disabled - building locally")
	}
	
	// Fall back to local build
	return m.BuildImage(ctx, variant, platform)
}

// GetClient returns the underlying Docker client for advanced operations
func (m *manager) GetClient() *client.Client {
	if c, ok := m.client.(*client.Client); ok {
		return c
	}
	return nil
}

// StartOrRecoverContainer starts a new container or recovers an existing one based on session persistence
func (m *manager) StartOrRecoverContainer(ctx context.Context, config *pkg.ContainerConfig, sessionConfig *pkg.Config) (string, error) {
	// If session persistence is disabled, always start fresh
	if !sessionConfig.SessionPersistence {
		return m.StartContainer(ctx, config)
	}

	// Try to recover existing container by saved ContainerID
	if sessionConfig.ContainerID != "" {
		healthy, err := m.IsContainerHealthy(ctx, sessionConfig.ContainerID)
		if err != nil {
			m.logger.Warnf("Failed to check container health: %v", err)
		} else if healthy {
			// Safely truncate session ID for display
			displaySessionID := sessionConfig.LastSessionID
			if len(displaySessionID) > 8 {
				displaySessionID = displaySessionID[:8]
			}
			m.logger.Infof("🔄 Recovering existing session: %s", displaySessionID)
			return sessionConfig.ContainerID, nil
		}
		m.logger.Info("Previous container unhealthy, checking for container by name")
	}

	// Try to recover existing container by name (in case ContainerID was lost but container still exists)
	status, err := m.GetContainerStatus(ctx, config.Name)
	if err != nil {
		m.logger.Debugf("Failed to check container status by name: %v", err)
	} else if status.Exists {
		if status.Running {
			m.logger.Infof("🔄 Found existing running container: %s", config.Name)
			// Update session config with the found container ID
			sessionConfig.ContainerID = status.ID
			return status.ID, nil
		} else {
			m.logger.Infof("🔄 Found existing stopped container, starting it: %s", config.Name)
			// Start the existing container
			if err := m.client.ContainerStart(ctx, status.ID, container.StartOptions{}); err != nil {
				m.logger.Warnf("Failed to start existing container: %v", err)
				// Remove the problematic container and create a new one
				m.logger.Info("Removing problematic container to create fresh one")
				m.client.ContainerRemove(ctx, status.ID, container.RemoveOptions{Force: true})
			} else {
				m.logger.Infof("✅ Successfully started existing container: %s", config.Name)
				// Update session config with the container ID
				sessionConfig.ContainerID = status.ID
				return status.ID, nil
			}
		}
	}

	// Start new container (no existing container found or existing one was problematic)
	containerID, err := m.StartContainer(ctx, config)
	if err != nil {
		return "", err
	}

	// Generate new session ID and update config
	sessionID, err := generateSessionID()
	if err != nil {
		m.logger.Warnf("Failed to generate session ID: %v", err)
		sessionID = fmt.Sprintf("session-%d", time.Now().Unix())
	}

	sessionConfig.LastSessionID = sessionID
	sessionConfig.ContainerID = containerID

	// Safely truncate session ID for display
	displaySessionID := sessionID
	if len(displaySessionID) > 8 {
		displaySessionID = displaySessionID[:8]
	}
	m.logger.Infof("💾 New session created: %s", displaySessionID)
	return containerID, nil
}

// IsContainerHealthy checks if a container exists and is healthy for session recovery
func (m *manager) IsContainerHealthy(ctx context.Context, containerID string) (bool, error) {
	// Check if container exists and is running
	containerInfo, err := m.client.ContainerInspect(ctx, containerID)
	if err != nil {
		// Container doesn't exist or can't be inspected
		return false, nil
	}

	// Check if container is running
	if !containerInfo.State.Running {
		m.logger.Debug("Container exists but is not running")
		return false, nil
	}

	// Check if container is responsive (simple health check)
	// Try to execute a basic command to verify the container is responsive
	execConfig := container.ExecOptions{
		Cmd:          []string{"echo", "health-check"},
		AttachStdout: true,
		AttachStderr: true,
	}

	execID, err := m.client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		m.logger.Debug("Container exec create failed - container unhealthy")
		return false, nil
	}

	execAttach, err := m.client.ContainerExecAttach(ctx, execID.ID, container.ExecAttachOptions{})
	if err != nil {
		m.logger.Debug("Container exec attach failed - container unhealthy")
		return false, nil
	}
	defer execAttach.Close()

	// Start the exec with a timeout
	execCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = m.client.ContainerExecStart(execCtx, execID.ID, container.ExecStartOptions{})
	if err != nil {
		m.logger.Debug("Container exec start failed - container unhealthy")
		return false, nil
	}

	// Container is running and responsive
	m.logger.Debug("Container health check passed")
	return true, nil
}

// initializeClaudeConfig copies the Claude configuration files from temporary mounts to proper locations
func (m *manager) initializeClaudeConfig(ctx context.Context, containerID string) error {
	// Copy Claude config from temporary mount to proper location
	copyConfigCmd := []string{"sh", "-c", `
		# Copy Claude config if the seed file exists
		if [ -f /tmp/claude-config-seed.json ]; then
			cp /tmp/claude-config-seed.json /home/claude/.claude.json
			chmod 644 /home/claude/.claude.json
			echo "Copied Claude config to /home/claude/.claude.json"
		fi
		
		# Copy credentials file if it exists
		if [ -f /home/claude/.credentials.json ]; then
			chmod 600 /home/claude/.credentials.json
			echo "Set credentials file permissions"
		fi
	`}
	
	// Create exec configuration
	execConfig := container.ExecOptions{
		Cmd:    copyConfigCmd,
		Detach: false,
	}
	
	// Create exec instance
	execResp, err := m.client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create config copy exec: %w", err)
	}
	
	// Start the exec instance
	if err := m.client.ContainerExecStart(ctx, execResp.ID, container.ExecStartOptions{}); err != nil {
		return fmt.Errorf("failed to copy config files: %w", err)
	}
	
	m.logger.Debug("Claude configuration files initialized")
	return nil
}

// generateSessionID creates a unique session identifier
func generateSessionID() (string, error) {
	// Generate a random 16-character hex string
	bytes := make([]byte, 8)
	for i := range bytes {
		n, err := rand.Int(rand.Reader, big.NewInt(256))
		if err != nil {
			return "", err
		}
		bytes[i] = byte(n.Int64())
	}
	return fmt.Sprintf("%x", bytes), nil
}

