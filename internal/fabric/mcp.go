package fabric

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"claude-reactor/pkg"
)

// MCPServer implements the MCP server for reactor-fabric orchestrator
type MCPServer struct {
	suite         *pkg.MCPSuite
	dockerManager *DockerManager
	proxy         *ContainerProxy
	logger        pkg.Logger
	sessions      map[string]*pkg.ClientSession
	containers    map[string]*pkg.ContainerInfo
	sessionMutex  sync.RWMutex
	listener      net.Listener
	stopChan      chan struct{}
}

// NewMCPServer creates a new MCP server instance
func NewMCPServer(suite *pkg.MCPSuite, dockerManager *DockerManager, logger pkg.Logger) *MCPServer {
	return &MCPServer{
		suite:         suite,
		dockerManager: dockerManager,
		proxy:         NewContainerProxy(logger),
		logger:        logger,
		sessions:      make(map[string]*pkg.ClientSession),
		containers:    make(map[string]*pkg.ContainerInfo),
		stopChan:      make(chan struct{}),
	}
}

// Start starts the MCP server on the specified address
func (s *MCPServer) Start(ctx context.Context, address string) error {
	s.logger.Info("Starting MCP server", "address", address)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", address, err)
	}

	s.listener = listener
	s.logger.Info("MCP server listening", "address", listener.Addr().String())

	// Start connection handler in goroutine
	go s.handleConnections(ctx)

	return nil
}

// Stop stops the MCP server and cleans up resources
func (s *MCPServer) Stop(ctx context.Context) error {
	s.logger.Info("Stopping MCP server")

	close(s.stopChan)

	if s.listener != nil {
		s.listener.Close()
	}

	// Clean up all active sessions and containers
	s.sessionMutex.Lock()
	defer s.sessionMutex.Unlock()

	for sessionID, session := range s.sessions {
		s.logger.Info("Cleaning up session", "sessionID", sessionID)
		
		// Stop any containers for this session
		for containerID, container := range s.containers {
			if container.SessionID == sessionID {
				if err := s.dockerManager.StopService(ctx, container); err != nil {
					s.logger.Warn("Failed to stop container during shutdown", 
						"containerID", containerID, "error", err)
				}
				delete(s.containers, containerID)
			}
		}

		// Close session connection
		if session.Conn != nil {
			session.Conn.Close()
		}
		delete(s.sessions, sessionID)
	}

	s.logger.Info("MCP server stopped")
	return nil
}

// handleConnections accepts and handles incoming connections
func (s *MCPServer) handleConnections(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				select {
				case <-s.stopChan:
					return
				default:
					s.logger.Error("Failed to accept connection", "error", err)
					continue
				}
			}

			// Handle each connection in a separate goroutine
			go s.handleClientConnection(ctx, conn)
		}
	}
}

// handleClientConnection handles a single client connection
func (s *MCPServer) handleClientConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	sessionID := pkg.GenerateSessionID()
	s.logger.Info("New client connection", "sessionID", sessionID, "remoteAddr", conn.RemoteAddr())

	// Create session
	session := &pkg.ClientSession{
		ID:        sessionID,
		Conn:      conn,
		CreatedAt: time.Now(),
		LastActivity: time.Now(),
	}

	// Register session
	s.sessionMutex.Lock()
	s.sessions[sessionID] = session
	s.sessionMutex.Unlock()

	// Handle messages from this client
	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		default:
			var message pkg.MCPMessage
			if err := decoder.Decode(&message); err != nil {
				s.logger.Debug("Client disconnected", "sessionID", sessionID, "error", err)
				s.cleanupSession(ctx, sessionID)
				return
			}

			// Update last activity
			s.sessionMutex.Lock()
			session.LastActivity = time.Now()
			s.sessionMutex.Unlock()

			// Handle the message
			response, err := s.handleMessage(ctx, sessionID, &message)
			if err != nil {
				// Send error response
				errorResponse := &pkg.MCPMessage{
					JsonRPC: "2.0",
					ID:      message.ID,
					Error: &pkg.MCPError{
						Code:    pkg.ErrorInternalError,
						Message: err.Error(),
					},
				}
				encoder.Encode(errorResponse)
				continue
			}

			// Send response if we have one
			if response != nil {
				if err := encoder.Encode(response); err != nil {
					s.logger.Error("Failed to send response", "sessionID", sessionID, "error", err)
					return
				}
			}
		}
	}
}

// handleMessage processes a single MCP message
func (s *MCPServer) handleMessage(ctx context.Context, sessionID string, message *pkg.MCPMessage) (*pkg.MCPMessage, error) {
	s.logger.Debug("Handling message", "sessionID", sessionID, "method", message.Method, "id", message.ID)

	// Handle fabric/registerClient specially
	if message.Method == "tools/call" && message.Params != nil {
		if params, ok := message.Params.(map[string]interface{}); ok {
			if name, exists := params["name"]; exists && name == "fabric/registerClient" {
				return s.handleRegisterClient(ctx, sessionID, message, params)
			}
		}
	}

	// Handle initialize method
	if message.Method == "initialize" {
		return s.handleInitialize(sessionID, message)
	}

	// Handle other tool calls by delegating to appropriate service
	if message.Method == "tools/call" {
		return s.handleToolCall(ctx, sessionID, message)
	}

	// Handle unknown methods
	return nil, fmt.Errorf("unsupported method: %s", message.Method)
}

// handleRegisterClient handles the fabric/registerClient tool call
func (s *MCPServer) handleRegisterClient(ctx context.Context, sessionID string, message *pkg.MCPMessage, params map[string]interface{}) (*pkg.MCPMessage, error) {
	s.logger.Info("Handling registerClient", "sessionID", sessionID)

	// Extract mounts from arguments
	arguments, exists := params["arguments"]
	if !exists {
		return nil, fmt.Errorf("missing arguments in registerClient call")
	}

	args, ok := arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments format")
	}

	mountsData, exists := args["mounts"]
	if !exists {
		return nil, fmt.Errorf("missing mounts in registerClient arguments")
	}

	// Parse mounts
	mountsArray, ok := mountsData.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid mounts format")
	}

	var mounts []pkg.Mount
	for _, mountData := range mountsArray {
		mountMap, ok := mountData.(map[string]interface{})
		if !ok {
			continue
		}

		mount := pkg.Mount{
			Source:   getString(mountMap, "source"),
			Target:   getString(mountMap, "target"),
			ReadOnly: getBool(mountMap, "readOnly"),
		}

		// Validate mount path against allowed roots
		configManager := NewConfigManager(s.logger)
		if err := configManager.ValidateMountPath(s.suite, mount.Source); err != nil {
			return nil, fmt.Errorf("mount validation failed: %w", err)
		}

		mounts = append(mounts, mount)
	}

	// Update session with client context
	s.sessionMutex.Lock()
	if session, exists := s.sessions[sessionID]; exists {
		session.ClientContext = &pkg.ClientContext{
			SessionID: sessionID,
			Mounts:    mounts,
		}
	}
	s.sessionMutex.Unlock()

	s.logger.Info("Client registered successfully", "sessionID", sessionID, "mounts", len(mounts))

	// Return success response
	return &pkg.MCPMessage{
		JsonRPC: "2.0",
		ID:      message.ID,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("Successfully registered client with session ID: %s", sessionID),
				},
			},
		},
	}, nil
}

// handleInitialize handles MCP initialize method
func (s *MCPServer) handleInitialize(sessionID string, message *pkg.MCPMessage) (*pkg.MCPMessage, error) {
	s.logger.Debug("Handling initialize", "sessionID", sessionID)

	// Build list of available tools from all services
	var tools []map[string]interface{}
	
	// Add fabric/registerClient tool
	tools = append(tools, map[string]interface{}{
		"name":        "fabric/registerClient",
		"description": "Register client session with mount information",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"mounts": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"source":   map[string]interface{}{"type": "string"},
							"target":   map[string]interface{}{"type": "string"},
							"readOnly": map[string]interface{}{"type": "boolean", "default": false},
						},
						"required": []string{"source", "target"},
					},
				},
			},
			"required": []string{"mounts"},
		},
	})

	// Add service-specific tools (these would be dynamically discovered)
	for serviceName := range s.suite.Services {
		tools = append(tools, map[string]interface{}{
			"name":        fmt.Sprintf("%s/invoke", serviceName),
			"description": fmt.Sprintf("Invoke tools from %s service", serviceName),
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"tool":      map[string]interface{}{"type": "string"},
					"arguments": map[string]interface{}{"type": "object"},
				},
				"required": []string{"tool"},
			},
		})
	}

	return &pkg.MCPMessage{
		JsonRPC: "2.0",
		ID:      message.ID,
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "reactor-fabric",
				"version": "1.0.0",
			},
			"tools": tools,
		},
	}, nil
}

// handleToolCall handles tool calls by routing to appropriate service
func (s *MCPServer) handleToolCall(ctx context.Context, sessionID string, message *pkg.MCPMessage) (*pkg.MCPMessage, error) {
	params, ok := message.Params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid params format")
	}

	toolName, exists := params["name"]
	if !exists {
		return nil, fmt.Errorf("missing tool name")
	}

	toolNameStr, ok := toolName.(string)
	if !ok {
		return nil, fmt.Errorf("invalid tool name format")
	}

	// Parse service name from tool call (format: serviceName/invoke or serviceName/toolName)
	parts := strings.Split(toolNameStr, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid tool name format: %s", toolNameStr)
	}

	serviceName := parts[0]
	
	// Check if service exists in configuration
	service, exists := s.suite.Services[serviceName]
	if !exists {
		return &pkg.MCPMessage{
			JsonRPC: "2.0",
			ID:      message.ID,
			Error: &pkg.MCPError{
				Code:    pkg.ErrorServiceNotFound,
				Message: fmt.Sprintf("Service %s not found", serviceName),
			},
		}, nil
	}

	// Get session context
	s.sessionMutex.RLock()
	session, sessionExists := s.sessions[sessionID]
	s.sessionMutex.RUnlock()

	if !sessionExists || session.ClientContext == nil {
		return &pkg.MCPMessage{
			JsonRPC: "2.0",
			ID:      message.ID,
			Error: &pkg.MCPError{
				Code:    pkg.ErrorInternalError,
				Message: "Session not properly registered. Please call fabric/registerClient first.",
			},
		}, nil
	}

	// Apply service detector for container strategy decisions
	detector := NewServiceDetector(s.logger)
	detector.ApplyDefaults(&service)

	// Determine container key based on strategy
	containerKey := detector.GetContainerKey(serviceName, sessionID, service.ContainerStrategy)
	container, containerExists := s.containers[containerKey]

	// Check if we need to refresh an existing container (smart_refresh strategy)
	if containerExists && service.ContainerStrategy == pkg.StrategySmartRefresh {
		if s.shouldRefreshContainer(container, &service) {
			s.logger.Info("Refreshing container due to smart refresh criteria", 
				"service", serviceName, 
				"container", container.ID[:12],
				"calls", container.CallCount)
			
			// Remove old container
			if err := s.dockerManager.StopService(ctx, container); err != nil {
				s.logger.Warn("Failed to stop old container during refresh", "error", err)
			}
			delete(s.containers, containerKey)
			containerExists = false
		}
	}

	if !containerExists {
		// Start a new container for this service
		s.logger.Info("Starting new container for service", 
			"service", serviceName, 
			"session", sessionID,
			"strategy", service.ContainerStrategy)

		var err error
		container, err = s.dockerManager.StartService(ctx, serviceName, &service, session.ClientContext)
		if err != nil {
			return &pkg.MCPMessage{
				JsonRPC: "2.0",
				ID:      message.ID,
				Error: &pkg.MCPError{
					Code:    pkg.ErrorContainerStartFailure,
					Message: fmt.Sprintf("Failed to start service container: %v", err),
				},
			}, nil
		}

		// Initialize container tracking for smart refresh
		container.CallCount = 0
		container.LastUsed = time.Now()
		
		// Store container info
		s.containers[containerKey] = container
		s.logger.Info("Service container started", 
			"service", serviceName, 
			"container", container.ID[:12],
			"strategy", service.ContainerStrategy)
	}

	// Update container usage tracking
	container.CallCount++
	container.LastUsed = time.Now()
	s.containers[containerKey] = container

	// Health check the container before proxying
	if err := s.proxy.HealthCheckContainer(ctx, container); err != nil {
		return &pkg.MCPMessage{
			JsonRPC: "2.0",
			ID:      message.ID,
			Error: &pkg.MCPError{
				Code:    pkg.ErrorContainerStartFailure,
				Message: fmt.Sprintf("Container health check failed: %v", err),
			},
		}, nil
	}

	// Proxy the tool call to the container
	s.logger.Info("Proxying tool call to container", "service", serviceName, "tool", toolNameStr, "session", sessionID)
	
	response, err := s.proxy.ProxyToContainer(ctx, container, message)
	if err != nil {
		return &pkg.MCPMessage{
			JsonRPC: "2.0",
			ID:      message.ID,
			Error: &pkg.MCPError{
				Code:    pkg.ErrorInternalError,
				Message: fmt.Sprintf("Failed to proxy to container: %v", err),
			},
		}, nil
	}

	s.logger.Info("Tool call completed successfully", "service", serviceName, "tool", toolNameStr, "session", sessionID)
	return response, nil
}

// shouldRefreshContainer determines if a container should be refreshed based on smart refresh criteria
func (s *MCPServer) shouldRefreshContainer(container *pkg.ContainerInfo, service *pkg.MCPService) bool {
	// Check call count threshold
	if service.MaxCallsPerContainer > 0 && container.CallCount >= service.MaxCallsPerContainer {
		s.logger.Debug("Container refresh due to call count threshold", 
			"container", container.ID[:12],
			"calls", container.CallCount, 
			"threshold", service.MaxCallsPerContainer)
		return true
	}
	
	// Check container age threshold
	if service.MaxContainerAge != "" {
		maxAge, err := time.ParseDuration(service.MaxContainerAge)
		if err == nil {
			containerAge := time.Since(container.StartTime)
			if containerAge >= maxAge {
				s.logger.Debug("Container refresh due to age threshold", 
					"container", container.ID[:12],
					"age", containerAge, 
					"threshold", maxAge)
				return true
			}
		}
	}
	
	// TODO: Check memory threshold when Docker stats API is available
	// This would require implementing container memory monitoring
	if service.MemoryThreshold != "" {
		s.logger.Debug("Memory threshold checking not yet implemented", 
			"container", container.ID[:12],
			"threshold", service.MemoryThreshold)
		// For now, we don't refresh based on memory
	}
	
	return false
}

// cleanupSession removes a session and its associated containers
func (s *MCPServer) cleanupSession(ctx context.Context, sessionID string) {
	s.sessionMutex.Lock()
	defer s.sessionMutex.Unlock()

	s.logger.Info("Cleaning up session", "sessionID", sessionID)

	// Stop containers for this session
	for containerID, container := range s.containers {
		if container.SessionID == sessionID {
			if err := s.dockerManager.StopService(ctx, container); err != nil {
				s.logger.Warn("Failed to stop container during cleanup", 
					"containerID", containerID, "error", err)
			}
			delete(s.containers, containerID)
		}
	}

	// Remove session
	delete(s.sessions, sessionID)
}

// Helper functions for type conversion
func getString(m map[string]interface{}, key string) string {
	if v, exists := m[key]; exists {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getBool(m map[string]interface{}, key string) bool {
	if v, exists := m[key]; exists {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}