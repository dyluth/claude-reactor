package pkg

import (
	"net"
	"time"
	"crypto/rand"
	"fmt"
)

// MCPSuite is the top-level structure for the claude-mcp-suite.yaml file.
type MCPSuite struct {
	Version      string                `yaml:"version"`
	Orchestrator OrchestratorConfig    `yaml:"orchestrator"`
	Services     map[string]MCPService `yaml:"mcp_services"`
}

// OrchestratorConfig holds global settings for the orchestrator itself.
type OrchestratorConfig struct {
	AllowedMountRoots []string `yaml:"allowed_mount_roots"`
}

// MCPService defines a single, orchestrable MCP agent.
type MCPService struct {
	Image               string                 `yaml:"image"`
	Config              map[string]interface{} `yaml:"config,omitempty"`
	Timeout             string                 `yaml:"timeout,omitempty"` // e.g., "1m", "5m30s"
	ServiceType         string                 `yaml:"service_type,omitempty"` // "llm_agent" or "tool_service"
	ContainerStrategy   string                 `yaml:"container_strategy,omitempty"` // "fresh_per_call", "reuse_per_session", "smart_refresh"
	MaxCallsPerContainer int                   `yaml:"max_calls_per_container,omitempty"` // For smart_refresh strategy
	MaxContainerAge     string                 `yaml:"max_container_age,omitempty"` // For smart_refresh strategy
	MemoryThreshold     string                 `yaml:"memory_threshold,omitempty"` // For smart_refresh strategy
}

// ClientContext holds the state for a single connected client session.
type ClientContext struct {
	SessionID string
	Mounts    []Mount
}

// Note: Mount and FabricOrchestrator are defined in interfaces.go


// ServiceRequest represents a request to spawn a service container
type ServiceRequest struct {
	ServiceName string
	SessionID   string
	ClientMounts []Mount
}

// ValidationResult represents the result of configuration validation
type ValidationResult struct {
	Valid  bool
	Errors []ValidationError
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
	Code    string
}

// MCP Communication Types

// MCPMessage represents a JSON-RPC 2.0 message for MCP communication
type MCPMessage struct {
	JsonRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents a JSON-RPC 2.0 error
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCP Error Codes (as specified in the design document)
const (
	ErrorDockerDaemonUnresponsive = -32001
	ErrorContainerStartFailure    = -32002
	ErrorInvalidSuiteConfig       = -32003
	ErrorSecurityViolation        = -32004
	ErrorServiceNotFound          = -32005
	ErrorInternalError            = -32603
)

// Service Types
const (
	ServiceTypeLLMAgent    = "llm_agent"
	ServiceTypeToolService = "tool_service"
)

// Container Strategies
const (
	StrategyFreshPerCall     = "fresh_per_call"
	StrategyReusePerSession  = "reuse_per_session" 
	StrategySmartRefresh     = "smart_refresh"
)

// ClientSession represents an active client connection and session state
type ClientSession struct {
	ID            string
	Conn          net.Conn
	ClientContext *ClientContext
	CreatedAt     time.Time
	LastActivity  time.Time
}

// ContainerInfo updated with proper field names matching the spec
type ContainerInfo struct {
	ID        string
	Name      string
	Service   string
	SessionID string
	Image     string
	StartTime time.Time
	CallCount int       // Track number of tool calls handled
	LastUsed  time.Time // Track last activity for smart refresh
}

// GenerateSessionID creates a unique session identifier
func GenerateSessionID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}