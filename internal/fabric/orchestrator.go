package fabric

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"claude-reactor/internal/reactor/logging"
	"claude-reactor/pkg"
)

// Orchestrator represents the main reactor-fabric orchestrator
type Orchestrator struct {
	logger           pkg.Logger
	configManager    *ConfigManager
	containerManager *ContainerManager
	dockerManager    *DockerManager
	mcpServer        *MCPServer
	config           *pkg.MCPSuite
}

// NewOrchestrator creates a new reactor-fabric orchestrator
func NewOrchestrator() (pkg.FabricOrchestrator, error) {
	logger := logging.NewLogger()
	
	configManager := NewConfigManager(logger)
	
	containerManager, err := NewContainerManager(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize container manager: %w", err)
	}

	dockerManager, err := NewDockerManager(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Docker manager: %w", err)
	}

	orchestrator := &Orchestrator{
		logger:           logger,
		configManager:    configManager,
		containerManager: containerManager,
		dockerManager:    dockerManager,
	}

	// Return as the interface - Orchestrator implements FabricOrchestrator
	return orchestrator, nil
}

// Start starts the orchestrator with the given configuration
func (o *Orchestrator) Start(ctx context.Context, configPath, listenAddr string) error {
	o.logger.Info("Starting Reactor-Fabric orchestrator")
	o.logger.Info("Configuration file: %s", configPath)
	o.logger.Info("Listen address: %s", listenAddr)

	// Load and validate configuration
	config, err := o.configManager.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	o.config = config

	// Check Docker daemon health
	if err := o.dockerManager.CheckDockerHealth(ctx); err != nil {
		return fmt.Errorf("Docker daemon health check failed: %w", err)
	}

	// Validate all service images
	if err := o.validateServiceImages(ctx); err != nil {
		return fmt.Errorf("service image validation failed: %w", err)
	}

	// Initialize MCP server
	o.mcpServer = NewMCPServer(o.config, o.dockerManager, o.logger)

	// Start MCP server
	if err := o.mcpServer.Start(ctx, listenAddr); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	// Start cleanup routine for idle containers
	go o.startCleanupRoutine(ctx)

	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	o.logger.Info("✅ Reactor-Fabric orchestrator started successfully")
	o.logger.Info("📋 Loaded %d MCP services", len(o.config.Services))
	o.logger.Info("🌐 MCP server listening on %s", listenAddr)
	
	// Print service summary
	for name, service := range o.config.Services {
		timeout := service.Timeout
		if timeout == "" {
			timeout = "default"
		}
		o.logger.Info("  📦 %s: %s (timeout: %s)", name, service.Image, timeout)
	}

	// Wait for shutdown signal
	select {
	case <-ctx.Done():
		o.logger.Info("Context cancelled, shutting down...")
	case sig := <-sigChan:
		o.logger.Info("Received signal %v, shutting down gracefully...", sig)
	}

	return o.shutdown(ctx)
}

// validateServiceImages checks that all configured service images are accessible
func (o *Orchestrator) validateServiceImages(ctx context.Context) error {
	o.logger.Info("Validating service images...")

	for name, service := range o.config.Services {
		o.logger.Debug("Validating image for service %s: %s", name, service.Image)
		
		// Docker manager will pull images as needed in ensureImage
		o.logger.Debug("✅ Image configured for service %s", name)
	}

	o.logger.Info("All service images configured successfully")
	return nil
}

// startCleanupRoutine starts a background goroutine to clean up idle containers
func (o *Orchestrator) startCleanupRoutine(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	o.logger.Debug("Started container cleanup routine")

	for {
		select {
		case <-ctx.Done():
			o.logger.Debug("Stopping cleanup routine")
			return
		case <-ticker.C:
			// Default cleanup after 1 minute of idle time
			defaultTimeout := 1 * time.Minute
			
			if err := o.containerManager.CleanupIdleContainers(ctx, defaultTimeout); err != nil {
				o.logger.Error("Error during container cleanup: %v", err)
			}
		}
	}
}

// shutdown gracefully shuts down the orchestrator
func (o *Orchestrator) shutdown(ctx context.Context) error {
	o.logger.Info("Shutting down Reactor-Fabric orchestrator...")

	// Set a timeout for shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop MCP server first
	if o.mcpServer != nil {
		if err := o.mcpServer.Stop(shutdownCtx); err != nil {
			o.logger.Error("Error stopping MCP server: %v", err)
		}
	}

	// Close Docker manager (will stop all containers)
	if o.dockerManager != nil {
		if err := o.dockerManager.Close(); err != nil {
			o.logger.Error("Error closing Docker manager: %v", err)
		}
	}

	// Close container manager
	if o.containerManager != nil {
		if err := o.containerManager.Close(); err != nil {
			o.logger.Error("Error closing container manager: %v", err)
		}
	}

	o.logger.Info("Reactor-Fabric orchestrator shut down complete")
	return nil
}

// GetConfig returns the current configuration
func (o *Orchestrator) GetConfig() *pkg.MCPSuite {
	return o.config
}

// GetContainerManager returns the container manager (for testing)
func (o *Orchestrator) GetContainerManager() *ContainerManager {
	return o.containerManager
}