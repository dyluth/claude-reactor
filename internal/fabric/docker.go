package fabric

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"

	"claude-reactor/pkg"
)

// DockerManager handles Docker container lifecycle operations
type DockerManager struct {
	client *client.Client
	logger pkg.Logger
}

// NewDockerManager creates a new Docker manager
func NewDockerManager(logger pkg.Logger) (*DockerManager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &DockerManager{
		client: cli,
		logger: logger,
	}, nil
}

// StartService starts a new container for the specified MCP service
func (d *DockerManager) StartService(ctx context.Context, serviceName string, service *pkg.MCPService, clientCtx *pkg.ClientContext) (*pkg.ContainerInfo, error) {
	d.logger.Info("Starting service container", "service", serviceName, "image", service.Image, "session", clientCtx.SessionID)

	// Create context with timeout for Docker operations
	dockerCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Pull image if needed
	if err := d.ensureImage(dockerCtx, service.Image); err != nil {
		return nil, fmt.Errorf("failed to ensure image: %w", err)
	}

	// Prepare container configuration
	containerConfig := &container.Config{
		Image: service.Image,
		Env:   []string{
			fmt.Sprintf("MCP_SESSION_ID=%s", clientCtx.SessionID),
		},
		Labels: map[string]string{
			"reactor-fabric.service":    serviceName,
			"reactor-fabric.session":    clientCtx.SessionID,
			"reactor-fabric.created":    time.Now().Format(time.RFC3339),
		},
	}

	// Prepare host configuration with mounts
	hostConfig := &container.HostConfig{
		Mounts:      d.buildMounts(clientCtx.Mounts),
		AutoRemove:  true,
		NetworkMode: "bridge",
	}

	// Generate unique container name
	containerName := fmt.Sprintf("reactor-fabric-%s-%s-%d", serviceName, clientCtx.SessionID[:8], time.Now().Unix())

	// Create container
	resp, err := d.client.ContainerCreate(dockerCtx, containerConfig, hostConfig, nil, nil, containerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Start container
	if err := d.client.ContainerStart(dockerCtx, resp.ID, container.StartOptions{}); err != nil {
		// Clean up the container if start fails
		d.client.ContainerRemove(dockerCtx, resp.ID, container.RemoveOptions{Force: true})
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// Wait for container to be healthy/ready
	if err := d.waitForContainer(dockerCtx, resp.ID); err != nil {
		// Clean up the container if health check fails
		d.client.ContainerRemove(dockerCtx, resp.ID, container.RemoveOptions{Force: true})
		return nil, fmt.Errorf("container failed health check: %w", err)
	}

	d.logger.Info("Service container started successfully", 
		"service", serviceName, 
		"container", resp.ID[:12], 
		"session", clientCtx.SessionID)

	return &pkg.ContainerInfo{
		ID:        resp.ID,
		Name:      containerName,
		Service:   serviceName,
		SessionID: clientCtx.SessionID,
		Image:     service.Image,
		StartTime: time.Now(),
	}, nil
}

// StopService stops and removes a service container
func (d *DockerManager) StopService(ctx context.Context, containerInfo *pkg.ContainerInfo) error {
	d.logger.Info("Stopping service container", 
		"service", containerInfo.Service, 
		"container", containerInfo.ID[:12])

	// Create context with timeout
	dockerCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Stop the container
	timeout := 10
	if err := d.client.ContainerStop(dockerCtx, containerInfo.ID, container.StopOptions{
		Timeout: &timeout,
	}); err != nil {
		d.logger.Warn("Failed to stop container gracefully, forcing removal", "error", err)
	}

	// Remove the container
	if err := d.client.ContainerRemove(dockerCtx, containerInfo.ID, container.RemoveOptions{
		Force: true,
	}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	d.logger.Info("Service container stopped and removed", 
		"service", containerInfo.Service, 
		"container", containerInfo.ID[:12])

	return nil
}

// GetContainerLogs retrieves logs from a container
func (d *DockerManager) GetContainerLogs(ctx context.Context, containerID string) (string, error) {
	dockerCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       "100",
	}

	logs, err := d.client.ContainerLogs(dockerCtx, containerID, options)
	if err != nil {
		return "", fmt.Errorf("failed to get container logs: %w", err)
	}
	defer logs.Close()

	logData, err := io.ReadAll(logs)
	if err != nil {
		return "", fmt.Errorf("failed to read container logs: %w", err)
	}

	return string(logData), nil
}

// ListServiceContainers lists all reactor-fabric managed containers
func (d *DockerManager) ListServiceContainers(ctx context.Context) ([]pkg.ContainerInfo, error) {
	dockerCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	containers, err := d.client.ContainerList(dockerCtx, container.ListOptions{
		All: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var serviceContainers []pkg.ContainerInfo
	for _, c := range containers {
		// Check if this is a reactor-fabric managed container
		if service, exists := c.Labels["reactor-fabric.service"]; exists {
			info := pkg.ContainerInfo{
				ID:      c.ID,
				Name:    strings.TrimPrefix(c.Names[0], "/"),
				Service: service,
				Image:   c.Image,
			}

			if sessionID, exists := c.Labels["reactor-fabric.session"]; exists {
				info.SessionID = sessionID
			}

			if createdStr, exists := c.Labels["reactor-fabric.created"]; exists {
				if created, err := time.Parse(time.RFC3339, createdStr); err == nil {
					info.StartTime = created
				}
			}

			serviceContainers = append(serviceContainers, info)
		}
	}

	return serviceContainers, nil
}

// CheckDockerHealth verifies Docker daemon is accessible
func (d *DockerManager) CheckDockerHealth(ctx context.Context) error {
	dockerCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := d.client.Ping(dockerCtx)
	if err != nil {
		return fmt.Errorf("Docker daemon unresponsive: %w", err)
	}

	return nil
}

// ensureImage pulls an image if it doesn't exist locally
func (d *DockerManager) ensureImage(ctx context.Context, imageName string) error {
	// Check if image exists locally
	_, _, err := d.client.ImageInspectWithRaw(ctx, imageName)
	if err == nil {
		// Image exists locally
		return nil
	}

	d.logger.Info("Pulling Docker image", "image", imageName)

	// Pull the image
	reader, err := d.client.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer reader.Close()

	// Wait for pull to complete
	_, err = io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to complete image pull: %w", err)
	}

	d.logger.Info("Successfully pulled Docker image", "image", imageName)
	return nil
}

// buildMounts converts client mounts to Docker mount specifications
func (d *DockerManager) buildMounts(clientMounts []pkg.Mount) []mount.Mount {
	var mounts []mount.Mount

	for _, m := range clientMounts {
		mountType := mount.TypeBind
		if strings.HasPrefix(m.Source, "volume://") {
			mountType = mount.TypeVolume
			m.Source = strings.TrimPrefix(m.Source, "volume://")
		}

		mounts = append(mounts, mount.Mount{
			Type:     mountType,
			Source:   m.Source,
			Target:   m.Target,
			ReadOnly: m.ReadOnly,
		})
	}

	return mounts
}

// waitForContainer waits for a container to be ready
func (d *DockerManager) waitForContainer(ctx context.Context, containerID string) error {
	// Simple readiness check - wait for container to be running
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		inspect, err := d.client.ContainerInspect(ctx, containerID)
		if err != nil {
			return fmt.Errorf("failed to inspect container: %w", err)
		}

		if inspect.State.Running {
			d.logger.Debug("Container is running", "container", containerID[:12])
			return nil
		}

		if inspect.State.Dead || inspect.State.Status == "exited" {
			logs, _ := d.GetContainerLogs(ctx, containerID)
			return fmt.Errorf("container failed to start (status: %s), logs: %s", 
				inspect.State.Status, logs)
		}

		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("container failed to become ready within timeout")
}

// Close closes the Docker client connection
func (d *DockerManager) Close() error {
	if d.client != nil {
		return d.client.Close()
	}
	return nil
}