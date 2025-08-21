package docker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types"

	"claude-reactor/pkg"
)

// RecoveryManager handles error recovery and retry logic for Docker operations
type RecoveryManager struct {
	logger    pkg.Logger
	dockerMgr pkg.DockerManager
}

// NewRecoveryManager creates a new recovery manager
func NewRecoveryManager(logger pkg.Logger, dockerMgr pkg.DockerManager) *RecoveryManager {
	return &RecoveryManager{
		logger:    logger,
		dockerMgr: dockerMgr,
	}
}

// RecoveryConfig defines retry and timeout settings
type RecoveryConfig struct {
	MaxRetries      int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	HealthCheckMaxRetries int
	HealthCheckDelay      time.Duration
}

// DefaultRecoveryConfig returns sensible default recovery settings
func DefaultRecoveryConfig() *RecoveryConfig {
	return &RecoveryConfig{
		MaxRetries:            3,
		InitialDelay:          1 * time.Second,
		MaxDelay:              10 * time.Second,
		BackoffFactor:         2.0,
		HealthCheckMaxRetries: 5,
		HealthCheckDelay:      1 * time.Second,
	}
}

// StartContainerWithRecovery starts a container with comprehensive error recovery
func (rm *RecoveryManager) StartContainerWithRecovery(ctx context.Context, config *pkg.ContainerConfig, recoveryConfig *RecoveryConfig) (string, error) {
	if recoveryConfig == nil {
		recoveryConfig = DefaultRecoveryConfig()
	}

	rm.logger.Debugf("Starting container with recovery: %s", config.Name)

	// Check for existing container first
	containerID, err := rm.handleExistingContainer(ctx, config.Name)
	if err != nil {
		return "", fmt.Errorf("failed to handle existing container: %w", err)
	}
	
	if containerID != "" {
		// Existing container found and handled
		return containerID, nil
	}

	// No existing container, start new one with retry logic
	var lastErr error
	delay := recoveryConfig.InitialDelay
	
	for attempt := 1; attempt <= recoveryConfig.MaxRetries; attempt++ {
		rm.logger.Debugf("Container start attempt %d/%d", attempt, recoveryConfig.MaxRetries)
		
		containerID, err := rm.dockerMgr.StartContainer(ctx, config)
		if err != nil {
			lastErr = err
			rm.logger.Warnf("Container start attempt %d failed: %v", attempt, err)
			
			// Check if this is a retryable error
			if !rm.isRetryableError(err) {
				rm.logger.Debugf("Error is not retryable, aborting")
				return "", fmt.Errorf("container start failed (non-retryable): %w", err)
			}
			
			if attempt < recoveryConfig.MaxRetries {
				rm.logger.Debugf("Waiting %v before retry", delay)
				select {
				case <-ctx.Done():
					return "", ctx.Err()
				case <-time.After(delay):
				}
				
				// Calculate next delay with exponential backoff
				delay = time.Duration(float64(delay) * recoveryConfig.BackoffFactor)
				if delay > recoveryConfig.MaxDelay {
					delay = recoveryConfig.MaxDelay
				}
				continue
			}
		} else {
			// Container started successfully, perform health check
			if err := rm.performHealthCheck(ctx, config.Name, recoveryConfig); err != nil {
				rm.logger.Errorf("Container health check failed: %v", err)
				// Clean up the unhealthy container
				_ = rm.dockerMgr.StopContainer(ctx, containerID)
				_ = rm.dockerMgr.RemoveContainer(ctx, containerID)
				return "", fmt.Errorf("container started but failed health check: %w", err)
			}
			
			rm.logger.Infof("✅ Container started successfully: %s", containerID[:12])
			return containerID, nil
		}
	}
	
	return "", fmt.Errorf("failed to start container after %d attempts: %w", recoveryConfig.MaxRetries, lastErr)
}

// handleExistingContainer checks for and handles existing containers
func (rm *RecoveryManager) handleExistingContainer(ctx context.Context, containerName string) (string, error) {
	running, err := rm.dockerMgr.IsContainerRunning(ctx, containerName)
	if err != nil {
		return "", err
	}

	if running {
		rm.logger.Infof("Found running container: %s", containerName)
		
		// Get container ID for running container
		// We need to list containers to get the ID from the name
		containers, err := rm.listContainersByName(ctx, containerName)
		if err != nil {
			return "", fmt.Errorf("failed to get container ID: %w", err)
		}
		
		if len(containers) > 0 {
			containerID := containers[0].ID
			rm.logger.Infof("✅ Reusing existing running container: %s", containerID[:12])
			return containerID, nil
		}
	} else {
		// Check if there's a stopped container we need to clean up
		containers, err := rm.listContainersByName(ctx, containerName)
		if err != nil {
			return "", err
		}
		
		for _, container := range containers {
			if container.State != "running" {
				rm.logger.Debugf("Found stopped container %s, removing", container.ID[:12])
				if err := rm.dockerMgr.RemoveContainer(ctx, container.ID); err != nil {
					rm.logger.Warnf("Failed to remove stopped container: %v", err)
				}
			}
		}
	}

	return "", nil // No existing container to reuse
}

// performHealthCheck verifies the container is responsive
func (rm *RecoveryManager) performHealthCheck(ctx context.Context, containerName string, config *RecoveryConfig) error {
	rm.logger.Debugf("Performing container health check: %s", containerName)
	
	for attempt := 1; attempt <= config.HealthCheckMaxRetries; attempt++ {
		// Simple health check - verify container is still running
		running, err := rm.dockerMgr.IsContainerRunning(ctx, containerName)
		if err != nil {
			rm.logger.Debugf("Health check attempt %d failed: %v", attempt, err)
		} else if running {
			rm.logger.Debugf("✅ Container health check passed")
			return nil
		}
		
		if attempt == config.HealthCheckMaxRetries {
			return fmt.Errorf("container health check failed after %d attempts", config.HealthCheckMaxRetries)
		}
		
		rm.logger.Debugf("Health check attempt %d failed, retrying...", attempt)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(config.HealthCheckDelay):
		}
	}
	
	return fmt.Errorf("container health check timed out")
}

// BuildImageWithRecovery builds a Docker image with retry logic
func (rm *RecoveryManager) BuildImageWithRecovery(ctx context.Context, variant, platform string, config *RecoveryConfig) error {
	if config == nil {
		config = DefaultRecoveryConfig()
	}

	rm.logger.Infof("Building image with recovery: variant=%s, platform=%s", variant, platform)
	
	var lastErr error
	delay := config.InitialDelay
	
	for attempt := 1; attempt <= config.MaxRetries; attempt++ {
		rm.logger.Debugf("Image build attempt %d/%d", attempt, config.MaxRetries)
		
		err := rm.dockerMgr.BuildImage(ctx, variant, platform)
		if err != nil {
			lastErr = err
			rm.logger.Warnf("Image build attempt %d failed: %v", attempt, err)
			
			// Check if this is a retryable error
			if !rm.isBuildRetryableError(err) {
				rm.logger.Debugf("Build error is not retryable, aborting")
				return fmt.Errorf("image build failed (non-retryable): %w", err)
			}
			
			if attempt < config.MaxRetries {
				rm.logger.Debugf("Waiting %v before build retry", delay)
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(delay):
				}
				
				// Calculate next delay with exponential backoff
				delay = time.Duration(float64(delay) * config.BackoffFactor)
				if delay > config.MaxDelay {
					delay = config.MaxDelay
				}
				continue
			}
		} else {
			rm.logger.Infof("✅ Image built successfully: %s-%s", variant, platform)
			return nil
		}
	}
	
	return fmt.Errorf("failed to build image after %d attempts: %w", config.MaxRetries, lastErr)
}

// StopContainerWithRecovery stops a container with error recovery
func (rm *RecoveryManager) StopContainerWithRecovery(ctx context.Context, containerID string, config *RecoveryConfig) error {
	if config == nil {
		config = DefaultRecoveryConfig()
	}

	var lastErr error
	delay := config.InitialDelay
	
	for attempt := 1; attempt <= config.MaxRetries; attempt++ {
		err := rm.dockerMgr.StopContainer(ctx, containerID)
		if err != nil {
			lastErr = err
			
			// Some stop errors are acceptable (container already stopped)
			if rm.isStopAcceptableError(err) {
				rm.logger.Debugf("Container stop completed with acceptable error: %v", err)
				return nil
			}
			
			if attempt < config.MaxRetries {
				rm.logger.Warnf("Stop attempt %d failed, retrying: %v", attempt, err)
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(delay):
				}
				delay = time.Duration(float64(delay) * config.BackoffFactor)
				if delay > config.MaxDelay {
					delay = config.MaxDelay
				}
				continue
			}
		} else {
			return nil
		}
	}
	
	return fmt.Errorf("failed to stop container after %d attempts: %w", config.MaxRetries, lastErr)
}

// isRetryableError determines if a container start error is worth retrying
func (rm *RecoveryManager) isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := strings.ToLower(err.Error())
	
	// Network/connectivity issues - retry
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "network") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "temporary failure") {
		return true
	}
	
	// Docker daemon issues - retry
	if strings.Contains(errStr, "daemon") ||
		strings.Contains(errStr, "server error") {
		return true
	}
	
	// Resource issues - retry
	if strings.Contains(errStr, "no space left") ||
		strings.Contains(errStr, "insufficient") {
		return true
	}
	
	// Configuration errors - don't retry
	if strings.Contains(errStr, "invalid") ||
		strings.Contains(errStr, "not found") ||
		strings.Contains(errStr, "bind source path does not exist") {
		return false
	}
	
	// Default to retryable for unknown errors
	return true
}

// isBuildRetryableError determines if a build error is worth retrying
func (rm *RecoveryManager) isBuildRetryableError(err error) bool {
	if err == nil {
		return false
	}
	
	errStr := strings.ToLower(err.Error())
	
	// Network issues during build - retry
	if strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "network") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "pull") ||
		strings.Contains(errStr, "download") {
		return true
	}
	
	// Dockerfile syntax errors - don't retry
	if strings.Contains(errStr, "dockerfile") ||
		strings.Contains(errStr, "syntax") ||
		strings.Contains(errStr, "unknown instruction") {
		return false
	}
	
	// Default to retryable for unknown build errors
	return true
}

// isStopAcceptableError determines if a stop error is acceptable
func (rm *RecoveryManager) isStopAcceptableError(err error) bool {
	if err == nil {
		return true
	}
	
	errStr := strings.ToLower(err.Error())
	
	// Already stopped errors are acceptable
	if strings.Contains(errStr, "already stopped") ||
		strings.Contains(errStr, "not running") ||
		strings.Contains(errStr, "no such container") {
		return true
	}
	
	return false
}

// listContainersByName lists containers matching a specific name
func (rm *RecoveryManager) listContainersByName(ctx context.Context, name string) ([]types.Container, error) {
	// This is a helper method - in a real implementation we'd need to access
	// the Docker client directly or extend the DockerManager interface
	// For now, we'll simulate this functionality
	
	// Note: This would need to be implemented by accessing the Docker client
	// directly or extending the DockerManager interface to include ListContainers
	return []types.Container{}, nil
}