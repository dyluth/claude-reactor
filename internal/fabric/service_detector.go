package fabric

import (
	"fmt"
	"strings"
	"time"

	"claude-reactor/pkg"
)

// ServiceDetector handles automatic detection of service types and default strategies
type ServiceDetector struct {
	logger pkg.Logger
}

// NewServiceDetector creates a new service detector
func NewServiceDetector(logger pkg.Logger) *ServiceDetector {
	return &ServiceDetector{
		logger: logger,
	}
}

// DetectServiceType automatically determines if a service is an LLM agent or traditional tool service
func (d *ServiceDetector) DetectServiceType(service *pkg.MCPService) string {
	// Return explicit service type if provided
	if service.ServiceType != "" {
		d.logger.Debug("Using explicit service type", "type", service.ServiceType, "image", service.Image)
		return service.ServiceType
	}

	// Auto-detect based on image name patterns
	imageLower := strings.ToLower(service.Image)
	
	// LLM Agent patterns
	llmPatterns := []string{
		"claude-reactor",
		"claude-",
		"llm-",
		"gpt-",
		"anthropic",
		"openai",
		"chatgpt",
		"assistant",
		"agent",
		"expert",
	}

	for _, pattern := range llmPatterns {
		if strings.Contains(imageLower, pattern) {
			d.logger.Debug("Auto-detected LLM agent service", "pattern", pattern, "image", service.Image)
			return pkg.ServiceTypeLLMAgent
		}
	}

	// Traditional tool service patterns
	toolPatterns := []string{
		"server-filesystem",
		"server-git", 
		"server-shell",
		"server-database",
		"mcp-server",
		"tool-",
		"util-",
	}

	for _, pattern := range toolPatterns {
		if strings.Contains(imageLower, pattern) {
			d.logger.Debug("Auto-detected tool service", "pattern", pattern, "image", service.Image)
			return pkg.ServiceTypeToolService
		}
	}

	// Default to tool service for unknown patterns
	d.logger.Debug("Defaulting to tool service (unknown pattern)", "image", service.Image)
	return pkg.ServiceTypeToolService
}

// GetDefaultContainerStrategy returns the default container strategy for a service type
func (d *ServiceDetector) GetDefaultContainerStrategy(serviceType string) string {
	switch serviceType {
	case pkg.ServiceTypeLLMAgent:
		// LLM agents should get fresh containers to maintain full context window
		return pkg.StrategyFreshPerCall
	case pkg.ServiceTypeToolService:
		// Traditional tools can reuse containers for performance
		return pkg.StrategyReusePerSession
	default:
		// Safe default
		return pkg.StrategyReusePerSession
	}
}

// ApplyDefaults applies smart defaults to a service configuration
func (d *ServiceDetector) ApplyDefaults(service *pkg.MCPService) {
	// Detect service type if not specified
	detectedType := d.DetectServiceType(service)
	if service.ServiceType == "" {
		service.ServiceType = detectedType
	}

	// Apply default container strategy if not specified
	if service.ContainerStrategy == "" {
		service.ContainerStrategy = d.GetDefaultContainerStrategy(detectedType)
		d.logger.Debug("Applied default container strategy", 
			"service_type", detectedType, 
			"strategy", service.ContainerStrategy,
			"image", service.Image)
	}

	// Apply smart refresh defaults for smart_refresh strategy
	if service.ContainerStrategy == pkg.StrategySmartRefresh {
		d.applySmartRefreshDefaults(service, detectedType)
	}

	// Apply timeout defaults based on service type
	if service.Timeout == "" {
		d.applyTimeoutDefaults(service, detectedType)
	}
}

// applySmartRefreshDefaults sets reasonable defaults for smart refresh strategy
func (d *ServiceDetector) applySmartRefreshDefaults(service *pkg.MCPService, serviceType string) {
	if serviceType == pkg.ServiceTypeLLMAgent {
		// LLM agents: Conservative refresh to preserve context
		if service.MaxCallsPerContainer == 0 {
			service.MaxCallsPerContainer = 5
		}
		if service.MaxContainerAge == "" {
			service.MaxContainerAge = "20m"
		}
		if service.MemoryThreshold == "" {
			service.MemoryThreshold = "400MB"
		}
	} else {
		// Tool services: More aggressive reuse for performance
		if service.MaxCallsPerContainer == 0 {
			service.MaxCallsPerContainer = 20
		}
		if service.MaxContainerAge == "" {
			service.MaxContainerAge = "30m"
		}
		if service.MemoryThreshold == "" {
			service.MemoryThreshold = "300MB"
		}
	}

	d.logger.Debug("Applied smart refresh defaults", 
		"service_type", serviceType,
		"max_calls", service.MaxCallsPerContainer,
		"max_age", service.MaxContainerAge,
		"memory_threshold", service.MemoryThreshold)
}

// applyTimeoutDefaults sets reasonable timeout defaults based on service type
func (d *ServiceDetector) applyTimeoutDefaults(service *pkg.MCPService, serviceType string) {
	if serviceType == pkg.ServiceTypeLLMAgent {
		// LLM agents typically need longer timeouts for complex reasoning
		service.Timeout = "10m"
	} else {
		// Tool services are typically faster
		service.Timeout = "2m"
	}

	d.logger.Debug("Applied timeout default", 
		"service_type", serviceType, 
		"timeout", service.Timeout)
}

// ValidateContainerStrategy ensures the container strategy configuration is valid
func (d *ServiceDetector) ValidateContainerStrategy(service *pkg.MCPService) error {
	validStrategies := []string{
		pkg.StrategyFreshPerCall,
		pkg.StrategyReusePerSession,
		pkg.StrategySmartRefresh,
	}

	// Check if strategy is valid
	valid := false
	for _, validStrategy := range validStrategies {
		if service.ContainerStrategy == validStrategy {
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("invalid container strategy '%s', must be one of: %v", 
			service.ContainerStrategy, validStrategies)
	}

	// Validate smart refresh specific configuration
	if service.ContainerStrategy == pkg.StrategySmartRefresh {
		if err := d.validateSmartRefreshConfig(service); err != nil {
			return fmt.Errorf("smart refresh configuration error: %w", err)
		}
	}

	return nil
}

// validateSmartRefreshConfig validates smart refresh specific settings
func (d *ServiceDetector) validateSmartRefreshConfig(service *pkg.MCPService) error {
	// Validate MaxCallsPerContainer
	if service.MaxCallsPerContainer < 1 {
		return fmt.Errorf("max_calls_per_container must be at least 1, got %d", service.MaxCallsPerContainer)
	}

	// Validate MaxContainerAge
	if service.MaxContainerAge != "" {
		if _, err := time.ParseDuration(service.MaxContainerAge); err != nil {
			return fmt.Errorf("invalid max_container_age format '%s': %w", service.MaxContainerAge, err)
		}
	}

	// Note: MemoryThreshold validation would require parsing memory units (MB, GB, etc.)
	// For now, we'll just check it's not empty when specified
	if service.MemoryThreshold != "" {
		// Basic validation - should contain a number and unit
		if !strings.Contains(service.MemoryThreshold, "MB") && 
		   !strings.Contains(service.MemoryThreshold, "GB") {
			return fmt.Errorf("memory_threshold must specify units (MB or GB), got '%s'", service.MemoryThreshold)
		}
	}

	return nil
}

// GetContainerKey generates a unique key for container tracking based on strategy
func (d *ServiceDetector) GetContainerKey(serviceName, sessionID, strategy string) string {
	switch strategy {
	case pkg.StrategyFreshPerCall:
		// Fresh containers get unique keys each time
		return fmt.Sprintf("%s-%s-%d", serviceName, sessionID, time.Now().UnixNano())
	case pkg.StrategyReusePerSession, pkg.StrategySmartRefresh:
		// Reused containers use the same key per session
		return fmt.Sprintf("%s-%s", serviceName, sessionID)
	default:
		// Safe default
		return fmt.Sprintf("%s-%s", serviceName, sessionID)
	}
}