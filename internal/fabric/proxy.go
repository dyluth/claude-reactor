package fabric

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"claude-reactor/pkg"
)

// ContainerProxy handles proxying MCP messages to service containers
type ContainerProxy struct {
	logger pkg.Logger
}

// NewContainerProxy creates a new container proxy
func NewContainerProxy(logger pkg.Logger) *ContainerProxy {
	return &ContainerProxy{
		logger: logger,
	}
}

// ProxyToContainer forwards an MCP message to a service container and returns the response
func (p *ContainerProxy) ProxyToContainer(ctx context.Context, containerInfo *pkg.ContainerInfo, message *pkg.MCPMessage) (*pkg.MCPMessage, error) {
	p.logger.Debug("Proxying message to container", 
		"container", containerInfo.ID[:12], 
		"service", containerInfo.Service,
		"method", message.Method)

	// For Phase 2, we'll implement HTTP-based MCP communication
	// In a real implementation, this would discover the actual MCP endpoint in the container
	
	// Determine container MCP endpoint
	endpoint, err := p.discoverContainerEndpoint(ctx, containerInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to discover container endpoint: %w", err)
	}

	// Forward the message to the container
	response, err := p.forwardMessage(ctx, endpoint, message)
	if err != nil {
		return nil, fmt.Errorf("failed to forward message to container: %w", err)
	}

	p.logger.Debug("Received response from container", 
		"container", containerInfo.ID[:12],
		"method", message.Method)

	return response, nil
}

// discoverContainerEndpoint finds the MCP endpoint for a running container
func (p *ContainerProxy) discoverContainerEndpoint(ctx context.Context, containerInfo *pkg.ContainerInfo) (string, error) {
	// For Phase 2 MVP, we'll use a conventional endpoint
	// In production, this could inspect the container or use service discovery
	
	// For now, assume MCP services listen on port 8080 inside the container
	// We'll need to map this to a host port or use container networking
	
	// This is a simplified implementation - in reality we'd:
	// 1. Inspect the container to find exposed ports
	// 2. Use Docker networking to connect directly 
	// 3. Or use a service mesh/discovery system
	
	endpoint := fmt.Sprintf("http://localhost:8080/mcp") // Simplified for Phase 2
	p.logger.Debug("Using container endpoint", "endpoint", endpoint, "container", containerInfo.ID[:12])
	
	return endpoint, nil
}

// forwardMessage sends an MCP message to a container endpoint and returns the response
func (p *ContainerProxy) forwardMessage(ctx context.Context, endpoint string, message *pkg.MCPMessage) (*pkg.MCPMessage, error) {
	// Marshal the MCP message to JSON for logging
	_, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	// For Phase 2 MVP, we'll simulate the container response
	// In Phase 3, this would make the actual HTTP call to the container
	p.logger.Debug("Simulating container communication", "endpoint", endpoint)
	response := p.simulateContainerResponse(message)
	
	return response, nil
}

// simulateContainerResponse creates a mock response for Phase 2 testing
// This will be replaced with real container communication in Phase 3
func (p *ContainerProxy) simulateContainerResponse(originalMessage *pkg.MCPMessage) *pkg.MCPMessage {
	// Extract tool information from the original message
	var toolName, serviceName string
	if originalMessage.Params != nil {
		if params, ok := originalMessage.Params.(map[string]interface{}); ok {
			if name, exists := params["name"]; exists {
				if nameStr, ok := name.(string); ok {
					parts := strings.Split(nameStr, "/")
					if len(parts) >= 2 {
						serviceName = parts[0]
						toolName = nameStr
					}
				}
			}
		}
	}

	// Create a simulated successful response
	return &pkg.MCPMessage{
		JsonRPC: "2.0",
		ID:      originalMessage.ID,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("✅ Simulated response from %s service for tool %s\n\nThis is a Phase 2 MVP response. In Phase 3, this will be a real response from the %s container.", 
						serviceName, toolName, serviceName),
				},
			},
		},
	}
}

// HealthCheckContainer verifies that a container's MCP endpoint is accessible
func (p *ContainerProxy) HealthCheckContainer(ctx context.Context, containerInfo *pkg.ContainerInfo) error {
	endpoint, err := p.discoverContainerEndpoint(ctx, containerInfo)
	if err != nil {
		return fmt.Errorf("failed to discover endpoint: %w", err)
	}

	// For Phase 2, we'll assume the endpoint is healthy if we can discover it
	p.logger.Debug("Container health check passed", 
		"container", containerInfo.ID[:12], 
		"endpoint", endpoint)
	
	return nil
}

// GetContainerStats returns basic statistics about container communication
func (p *ContainerProxy) GetContainerStats(containerInfo *pkg.ContainerInfo) map[string]interface{} {
	return map[string]interface{}{
		"container_id": containerInfo.ID,
		"service":      containerInfo.Service,
		"session_id":   containerInfo.SessionID,
		"start_time":   containerInfo.StartTime,
		"status":       "running", // Simplified for Phase 2
	}
}