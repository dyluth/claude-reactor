package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	
	"claude-reactor/pkg"
)

func TestNewVariantManager(t *testing.T) {
	mockLogger := &MockLogger{}
	mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Return()
	
	vm := NewVariantManager(mockLogger)
	
	assert.NotNil(t, vm)
	assert.NotNil(t, vm.variants)
	assert.Greater(t, len(vm.variants), 0, "Should have built-in variants")
	
	// Verify all expected variants are present
	expectedVariants := []string{"base", "go", "full", "cloud", "k8s"}
	for _, variant := range expectedVariants {
		_, exists := vm.variants[variant]
		assert.True(t, exists, "Variant %s should exist", variant)
	}
}

func TestVariantManager_ValidateVariant(t *testing.T) {
	mockLogger := &MockLogger{}
	mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Return()
	
	vm := NewVariantManager(mockLogger)
	
	tests := []struct {
		name     string
		variant  string
		wantErr  bool
		errMsg   string
	}{
		{
			name:    "valid base variant",
			variant: "base",
			wantErr: false,
		},
		{
			name:    "valid go variant", 
			variant: "go",
			wantErr: false,
		},
		{
			name:    "valid full variant",
			variant: "full", 
			wantErr: false,
		},
		{
			name:    "valid cloud variant",
			variant: "cloud",
			wantErr: false,
		},
		{
			name:    "valid k8s variant",
			variant: "k8s",
			wantErr: false,
		},
		{
			name:    "empty variant",
			variant: "",
			wantErr: true,
			errMsg:  "variant cannot be empty",
		},
		{
			name:    "invalid variant",
			variant: "invalid",
			wantErr: true,
			errMsg:  "invalid variant 'invalid'",
		},
		{
			name:    "case sensitive - uppercase",
			variant: "BASE",
			wantErr: true,
			errMsg:  "invalid variant 'BASE'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := vm.ValidateVariant(tt.variant)
			
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestVariantManager_GetVariantDefinition(t *testing.T) {
	mockLogger := &MockLogger{}
	mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Return()
	
	vm := NewVariantManager(mockLogger)
	
	tests := []struct {
		name     string
		variant  string
		wantErr  bool
		validate func(t *testing.T, def *pkg.VariantDefinition)
	}{
		{
			name:    "get base variant definition",
			variant: "base",
			wantErr: false,
			validate: func(t *testing.T, def *pkg.VariantDefinition) {
				assert.Equal(t, "base", def.Name)
				assert.Contains(t, def.Description, "Node.js")
				assert.Contains(t, def.Tools, "node.js")
				assert.Contains(t, def.Tools, "python3")
				assert.Equal(t, "~500MB", def.Size)
			},
		},
		{
			name:    "get go variant definition", 
			variant: "go",
			wantErr: false,
			validate: func(t *testing.T, def *pkg.VariantDefinition) {
				assert.Equal(t, "go", def.Name)
				assert.Contains(t, def.Description, "Go toolchain")
				assert.Contains(t, def.Tools, "go")
				assert.Contains(t, def.Tools, "gopls")
				assert.Equal(t, "~800MB", def.Size)
				assert.Contains(t, def.Dependencies, "base")
			},
		},
		{
			name:    "get full variant definition",
			variant: "full", 
			wantErr: false,
			validate: func(t *testing.T, def *pkg.VariantDefinition) {
				assert.Equal(t, "full", def.Name)
				assert.Contains(t, def.Description, "Rust")
				assert.Contains(t, def.Description, "Java")
				assert.Contains(t, def.Tools, "rust")
				assert.Contains(t, def.Tools, "java")
				assert.Contains(t, def.Tools, "mysql-client")
				assert.Equal(t, "~1.2GB", def.Size)
			},
		},
		{
			name:    "invalid variant",
			variant: "nonexistent", 
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def, err := vm.GetVariantDefinition(tt.variant)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, def)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, def)
				if tt.validate != nil {
					tt.validate(t, def)
				}
			}
		})
	}
}

func TestVariantManager_GetAvailableVariants(t *testing.T) {
	mockLogger := &MockLogger{}
	mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Return()
	
	vm := NewVariantManager(mockLogger)
	
	variants := vm.GetAvailableVariants()
	
	assert.NotEmpty(t, variants)
	assert.Equal(t, 5, len(variants), "Should have 5 built-in variants")
	
	// Check that all expected variants are present
	expectedVariants := map[string]bool{
		"base":  false,
		"go":    false, 
		"full":  false,
		"cloud": false,
		"k8s":   false,
	}
	
	for _, variant := range variants {
		if _, exists := expectedVariants[variant]; exists {
			expectedVariants[variant] = true
		}
	}
	
	// Verify all expected variants were found
	for variant, found := range expectedVariants {
		assert.True(t, found, "Variant %s should be in available variants", variant)
	}
}

func TestVariantManager_GetVariantDescription(t *testing.T) {
	mockLogger := &MockLogger{}
	mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Return()
	
	vm := NewVariantManager(mockLogger)
	
	tests := []struct {
		name     string
		variant  string
		expected string
	}{
		{
			name:     "base variant description",
			variant:  "base",
			expected: "Node.js, Python",
		},
		{
			name:     "go variant description", 
			variant:  "go",
			expected: "Go toolchain",
		},
		{
			name:     "cloud variant description",
			variant:  "cloud", 
			expected: "AWS/GCP/Azure",
		},
		{
			name:     "unknown variant",
			variant:  "unknown",
			expected: "Unknown variant",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			description := vm.GetVariantDescription(tt.variant)
			assert.Contains(t, description, tt.expected)
		})
	}
}

func TestVariantManager_GetVariantSize(t *testing.T) {
	mockLogger := &MockLogger{}
	mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Return()
	
	vm := NewVariantManager(mockLogger)
	
	tests := []struct {
		name     string
		variant  string
		expected string
	}{
		{
			name:     "base variant size",
			variant:  "base", 
			expected: "~500MB",
		},
		{
			name:     "go variant size",
			variant:  "go",
			expected: "~800MB",
		},
		{
			name:     "full variant size",
			variant:  "full",
			expected: "~1.2GB", 
		},
		{
			name:     "unknown variant size",
			variant:  "unknown",
			expected: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := vm.GetVariantSize(tt.variant)
			assert.Equal(t, tt.expected, size)
		})
	}
}

func TestVariantManager_GetVariantTools(t *testing.T) {
	mockLogger := &MockLogger{}
	mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Return()
	
	vm := NewVariantManager(mockLogger)
	
	tests := []struct {
		name           string
		variant        string
		shouldContain  []string
		shouldNotContain []string
	}{
		{
			name:    "base variant tools",
			variant: "base",
			shouldContain: []string{"node.js", "python3", "git", "kubectl"},
			shouldNotContain: []string{"go", "rust", "java"},
		},
		{
			name:    "go variant tools",
			variant: "go", 
			shouldContain: []string{"node.js", "python3", "go", "gopls"},
			shouldNotContain: []string{"rust", "java", "aws-cli"},
		},
		{
			name:    "full variant tools",
			variant: "full",
			shouldContain: []string{"go", "rust", "java", "mysql-client"},
			shouldNotContain: []string{"aws-cli", "helm"},
		},
		{
			name:    "cloud variant tools",
			variant: "cloud",
			shouldContain: []string{"go", "rust", "java", "aws-cli", "terraform"},
			shouldNotContain: []string{"helm", "k9s"},
		},
		{
			name:    "k8s variant tools",
			variant: "k8s", 
			shouldContain: []string{"go", "rust", "java", "helm", "k9s", "stern"},
			shouldNotContain: []string{"aws-cli", "terraform"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tools := vm.GetVariantTools(tt.variant)
			
			for _, tool := range tt.shouldContain {
				assert.Contains(t, tools, tool, "Variant %s should contain tool %s", tt.variant, tool)
			}
			
			for _, tool := range tt.shouldNotContain {
				assert.NotContains(t, tools, tool, "Variant %s should not contain tool %s", tt.variant, tool)
			}
		})
	}
	
	// Test unknown variant
	unknownTools := vm.GetVariantTools("unknown")
	assert.Empty(t, unknownTools)
}

func BenchmarkVariantManager_ValidateVariant(b *testing.B) {
	mockLogger := &MockLogger{}
	mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Return()
	
	vm := NewVariantManager(mockLogger)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = vm.ValidateVariant("go")
	}
}

func BenchmarkVariantManager_GetVariantDefinition(b *testing.B) {
	mockLogger := &MockLogger{}
	mockLogger.On("Debugf", mock.AnythingOfType("string"), mock.Anything).Return()
	
	vm := NewVariantManager(mockLogger)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = vm.GetVariantDefinition("full")
	}
}