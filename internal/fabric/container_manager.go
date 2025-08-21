package fabric

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"

	"claude-reactor/pkg"
)

// ContainerManager manages the lifecycle of MCP service containers
type ContainerManager struct {
	client          *client.Client
	logger          pkg.Logger
	containers      map[string]*pkg.ContainerInfo
	containersMutex sync.RWMutex
}

// NewContainerManager creates a new container manager
func NewContainerManager(logger pkg.Logger) (*ContainerManager, error) {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Test connection to Docker daemon
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = dockerClient.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Docker daemon: %w", err)
	}

	return &ContainerManager{
		client:     dockerClient,
		logger:     logger,
		containers: make(map[string]*pkg.ContainerInfo),
	}, nil
}

// StartContainer creates and starts a new MCP service container
func (cm *ContainerManager) StartContainer(ctx context.Context, req *pkg.ServiceRequest, service *pkg.MCPService) (*pkg.ContainerInfo, error) {
	cm.logger.Info("Starting container for service %s (session: %s)", req.ServiceName, req.SessionID)

	// Generate unique container name
	containerName := fmt.Sprintf("reactor-fabric-%s-%s", req.ServiceName, req.SessionID[:8])

	// Prepare mount configuration
	mounts := make([]mount.Mount, len(req.ClientMounts))
	for i, clientMount := range req.ClientMounts {
		mounts[i] = mount.Mount{
			Type:     mount.TypeBind,
			Source:   clientMount.Source,
			Target:   clientMount.Target,
			ReadOnly: clientMount.ReadOnly,
		}
		cm.logger.Debug("Adding mount: %s -> %s (readonly: %t)", clientMount.Source, clientMount.Target, clientMount.ReadOnly)
	}

	// Create container configuration
	config := &container.Config{
		Image: service.Image,
		Env:   []string{},
		// Add any service-specific environment variables from service.Config
	}

	// Add service config as environment variables if present
	if service.Config != nil {
		for key, value := range service.Config {
			if strValue, ok := value.(string); ok {
				config.Env = append(config.Env, fmt.Sprintf("%s=%s", strings.ToUpper(key), strValue))
			}
		}
	}

	hostConfig := &container.HostConfig{
		Mounts:      mounts,
		AutoRemove:  false, // We'll handle removal manually
		NetworkMode: "bridge",
		// Add resource constraints if needed
		Resources: container.Resources{
			Memory: 512 * 1024 * 1024, // 512MB limit
		},
	}

	// Create the container
	resp, err := cm.client.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Start the container
	if err := cm.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		// Clean up the created container if start fails
		cm.client.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// Wait for container to be running and healthy
	if err := cm.waitForContainerReady(ctx, resp.ID, 30*time.Second); err != nil {
		// Clean up if container doesn't become ready
		cm.stopAndRemoveContainer(ctx, resp.ID)
		return nil, fmt.Errorf("container failed to become ready: %w", err)
	}

	// Create container info
	info := &pkg.ContainerInfo{
		ID:        resp.ID,
		Name:      containerName,
		Image:     service.Image,
		Service:   req.ServiceName,
		SessionID: req.SessionID,
		StartTime: time.Now(),
	}

	// Store container info
	cm.containersMutex.Lock()
	cm.containers[resp.ID] = info
	cm.containersMutex.Unlock()

	cm.logger.Info("Successfully started container %s for service %s", resp.ID[:12], req.ServiceName)
	return info, nil
}

// StopContainer stops and removes a container
func (cm *ContainerManager) StopContainer(ctx context.Context, containerID string) error {
	cm.logger.Info("Stopping container %s", containerID[:12])

	if err := cm.stopAndRemoveContainer(ctx, containerID); err != nil {
		return err
	}

	// Remove from tracking
	cm.containersMutex.Lock()
	delete(cm.containers, containerID)
	cm.containersMutex.Unlock()

	return nil
}

// stopAndRemoveContainer handles the actual stopping and removal
func (cm *ContainerManager) stopAndRemoveContainer(ctx context.Context, containerID string) error {
	// Stop the container with a timeout
	timeout := 10
	if err := cm.client.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout}); err != nil {
		cm.logger.Warn("Failed to stop container %s gracefully: %v", containerID[:12], err)
	}

	// Remove the container
	if err := cm.client.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force: true, // Force remove even if it's still running
	}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	cm.logger.Info("Container %s stopped and removed", containerID[:12])
	return nil
}

// waitForContainerReady waits for container to be in running state
func (cm *ContainerManager) waitForContainerReady(ctx context.Context, containerID string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for container to be ready")
		case <-ticker.C:
			inspect, err := cm.client.ContainerInspect(ctx, containerID)
			if err != nil {
				return fmt.Errorf("failed to inspect container: %w", err)
			}

			if inspect.State.Running {
				cm.logger.Debug("Container %s is now running", containerID[:12])
				return nil
			}

			if inspect.State.Dead || inspect.State.ExitCode != 0 {
				return fmt.Errorf("container exited unexpectedly with code %d", inspect.State.ExitCode)
			}
		}
	}
}

// GetContainerInfo retrieves information about a tracked container
func (cm *ContainerManager) GetContainerInfo(containerID string) (*pkg.ContainerInfo, bool) {
	cm.containersMutex.RLock()
	defer cm.containersMutex.RUnlock()
	
	info, exists := cm.containers[containerID]
	return info, exists
}

// ListContainers returns all currently tracked containers
func (cm *ContainerManager) ListContainers() []*pkg.ContainerInfo {
	cm.containersMutex.RLock()
	defer cm.containersMutex.RUnlock()

	containers := make([]*pkg.ContainerInfo, 0, len(cm.containers))
	for _, info := range cm.containers {
		containers = append(containers, info)
	}
	return containers
}

// UpdateLastUsed updates the last used timestamp for a container
func (cm *ContainerManager) UpdateLastUsed(containerID string) {
	cm.containersMutex.Lock()
	defer cm.containersMutex.Unlock()

	if info, exists := cm.containers[containerID]; exists {
		// Update start time as proxy for last used time
		info.StartTime = time.Now()
	}
}

// CleanupIdleContainers removes containers that have been idle for too long
func (cm *ContainerManager) CleanupIdleContainers(ctx context.Context, maxIdleTime time.Duration) error {
	cm.containersMutex.RLock()
	var containersToCleanup []string
	
	for id, info := range cm.containers {
		if time.Since(info.StartTime) > maxIdleTime {
			containersToCleanup = append(containersToCleanup, id)
		}
	}
	cm.containersMutex.RUnlock()

	// Clean up idle containers
	for _, containerID := range containersToCleanup {
		cm.logger.Info("Cleaning up idle container %s (service: %s)", containerID[:12], cm.containers[containerID].Service)
		if err := cm.StopContainer(ctx, containerID); err != nil {
			cm.logger.Error("Failed to cleanup container %s: %v", containerID[:12], err)
		}
	}

	if len(containersToCleanup) > 0 {
		cm.logger.Info("Cleaned up %d idle containers", len(containersToCleanup))
	}

	return nil
}

// PullImage ensures the Docker image is available locally
func (cm *ContainerManager) PullImage(ctx context.Context, imageName string) error {
	cm.logger.Info("Pulling Docker image: %s", imageName)

	reader, err := cm.client.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer reader.Close()

	// Read the response to ensure pull completes
	// In a real implementation, you might want to parse this and show progress
	_, err = io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read image pull response: %w", err)
	}

	cm.logger.Info("Successfully pulled image: %s", imageName)
	return nil
}

// ValidateImage checks if an image exists locally or can be pulled
func (cm *ContainerManager) ValidateImage(ctx context.Context, imageName string) error {
	// First, check if image exists locally
	images, err := cm.client.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list images: %w", err)
	}

	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == imageName {
				cm.logger.Debug("Image %s found locally", imageName)
				return nil
			}
		}
	}

	// Image not found locally, try to inspect it remotely
	_, _, err = cm.client.ImageInspectWithRaw(ctx, imageName)
	if err == nil {
		return nil // Image exists remotely
	}

	return fmt.Errorf("image %s not found locally or remotely", imageName)
}

// Close cleans up the container manager
func (cm *ContainerManager) Close() error {
	// Stop all tracked containers
	ctx := context.Background()
	for containerID := range cm.containers {
		if err := cm.StopContainer(ctx, containerID); err != nil {
			cm.logger.Error("Failed to stop container %s during cleanup: %v", containerID[:12], err)
		}
	}

	return cm.client.Close()
}