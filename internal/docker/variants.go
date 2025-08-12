package docker

import (
	"fmt"
	"strings"
	
	"claude-reactor/pkg"
)

// VariantManager handles Docker container variant definitions and validation
type VariantManager struct {
	logger pkg.Logger
	variants map[string]*pkg.VariantDefinition
}

// NewVariantManager creates a new variant manager with built-in variant definitions
func NewVariantManager(logger pkg.Logger) *VariantManager {
	vm := &VariantManager{
		logger: logger,
		variants: make(map[string]*pkg.VariantDefinition),
	}
	
	// Initialize built-in variants (replicating the bash script's variants)
	vm.initializeBuiltinVariants()
	
	return vm
}

// ValidateVariant checks if a variant name is valid and supported
func (vm *VariantManager) ValidateVariant(variant string) error {
	if variant == "" {
		return fmt.Errorf("variant cannot be empty")
	}
	
	if _, exists := vm.variants[variant]; !exists {
		available := vm.GetAvailableVariants()
		return fmt.Errorf("invalid variant '%s'. Available variants: %s", 
			variant, strings.Join(available, ", "))
	}
	
	return nil
}

// GetVariantDefinition returns the definition for a specific variant
func (vm *VariantManager) GetVariantDefinition(variant string) (*pkg.VariantDefinition, error) {
	if err := vm.ValidateVariant(variant); err != nil {
		return nil, err
	}
	
	definition := vm.variants[variant]
	vm.logger.Debugf("Retrieved variant definition: %s", variant)
	
	return definition, nil
}

// GetAvailableVariants returns a list of all available variant names
func (vm *VariantManager) GetAvailableVariants() []string {
	variants := make([]string, 0, len(vm.variants))
	for name := range vm.variants {
		variants = append(variants, name)
	}
	return variants
}

// GetVariantDescription returns a human-readable description of a variant
func (vm *VariantManager) GetVariantDescription(variant string) string {
	if definition, exists := vm.variants[variant]; exists {
		return definition.Description
	}
	return "Unknown variant"
}

// GetVariantSize returns the estimated size of a variant
func (vm *VariantManager) GetVariantSize(variant string) string {
	if definition, exists := vm.variants[variant]; exists {
		return definition.Size
	}
	return "Unknown"
}

// GetVariantTools returns the list of tools included in a variant
func (vm *VariantManager) GetVariantTools(variant string) []string {
	if definition, exists := vm.variants[variant]; exists {
		return definition.Tools
	}
	return []string{}
}

// initializeBuiltinVariants sets up the standard container variants
// This replicates the variants from the original bash script and Dockerfile
func (vm *VariantManager) initializeBuiltinVariants() {
	// Define base tools first
	baseTools := []string{
		"node.js", "python3", "uv", "git", "ripgrep", "jq", 
		"kubectl", "github-cli", "docker-cli",
	}
	
	// Define Go tools (need to make a copy to avoid slice sharing issues)
	goTools := make([]string, len(baseTools))
	copy(goTools, baseTools)
	goTools = append(goTools, "go", "gopls", "delve", "staticcheck", "golangci-lint")
	
	// Define full tools (Go + Rust/Java/DB)
	fullTools := make([]string, len(goTools))
	copy(fullTools, goTools)
	fullTools = append(fullTools,
		"rust", "cargo", "java", "maven", "gradle",
		"mysql-client", "postgresql-client", "redis-tools", "sqlite3")
	
	// Define cloud tools (full + cloud CLIs)
	cloudTools := make([]string, len(fullTools))
	copy(cloudTools, fullTools)
	cloudTools = append(cloudTools, "aws-cli", "gcloud", "azure-cli", "terraform")
	
	// Define k8s tools (full + k8s tools)
	k8sTools := make([]string, len(fullTools))
	copy(k8sTools, fullTools)
	k8sTools = append(k8sTools, "helm", "k9s", "kubectx", "kubens", "kustomize", "stern")
	
	vm.variants["base"] = &pkg.VariantDefinition{
		Name:        "base",
		Description: "Node.js, Python (with pip + uv), basic development tools",
		BaseImage:   "debian:bullseye-slim",
		Dockerfile:  "Dockerfile",
		Tools:       baseTools,
		Size:        "~500MB",
		Dependencies: []string{},
	}
	
	vm.variants["go"] = &pkg.VariantDefinition{
		Name:        "go", 
		Description: "Base + Go toolchain and development utilities",
		BaseImage:   "base",
		Dockerfile:  "Dockerfile",
		Tools:       goTools,
		Size:        "~800MB",
		Dependencies: []string{"base"},
	}
	
	vm.variants["full"] = &pkg.VariantDefinition{
		Name:        "full",
		Description: "Go + Rust, Java, database clients",
		BaseImage:   "go", 
		Dockerfile:  "Dockerfile",
		Tools:       fullTools,
		Size:        "~1.2GB",
		Dependencies: []string{"base", "go"},
	}
	
	vm.variants["cloud"] = &pkg.VariantDefinition{
		Name:        "cloud",
		Description: "Full + AWS/GCP/Azure CLIs", 
		BaseImage:   "full",
		Dockerfile:  "Dockerfile",
		Tools:       cloudTools,
		Size:        "~1.5GB", 
		Dependencies: []string{"base", "go", "full"},
	}
	
	vm.variants["k8s"] = &pkg.VariantDefinition{
		Name:        "k8s",
		Description: "Full + Enhanced Kubernetes tools (helm, k9s, stern)",
		BaseImage:   "full",
		Dockerfile:  "Dockerfile", 
		Tools:       k8sTools,
		Size:        "~1.4GB",
		Dependencies: []string{"base", "go", "full"},
	}
	
	vm.logger.Debugf("Initialized %d built-in variants", len(vm.variants))
}