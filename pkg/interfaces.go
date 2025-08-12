package pkg

import (
	"context"
	"io"
)

// ArchitectureDetector provides methods for detecting system and container architectures
type ArchitectureDetector interface {
	// GetHostArchitecture returns the host system architecture (arm64, amd64, etc.)
	GetHostArchitecture() (string, error)
	
	// GetDockerPlatform returns the Docker platform format (linux/arm64, linux/amd64, etc.)
	GetDockerPlatform() (string, error)
	
	// IsMultiArchSupported checks if the system supports multi-architecture containers
	IsMultiArchSupported() bool
}

// ConfigManager handles configuration loading, validation, and persistence
type ConfigManager interface {
	// LoadConfig loads configuration from file or creates default
	LoadConfig() (*Config, error)
	
	// SaveConfig persists configuration to file
	SaveConfig(config *Config) error
	
	// ValidateConfig validates configuration structure and values
	ValidateConfig(config *Config) error
	
	// GetDefaultConfig returns a default configuration
	GetDefaultConfig() *Config
	
	// AutoDetectVariant detects project type from files in directory
	AutoDetectVariant(projectPath string) (string, error)
	
	// ListAccounts returns available Claude accounts
	ListAccounts() ([]string, error)
}

// DockerManager handles Docker container lifecycle and operations
type DockerManager interface {
	// BuildImage builds a Docker image for the specified variant
	BuildImage(ctx context.Context, variant string, platform string) error
	
	// RebuildImage forces rebuild of Docker image
	RebuildImage(ctx context.Context, variant string, platform string, force bool) error
	
	// StartContainer starts a container with the given configuration
	StartContainer(ctx context.Context, config *ContainerConfig) (string, error)
	
	// StopContainer stops a running container
	StopContainer(ctx context.Context, containerID string) error
	
	// RemoveContainer removes a stopped container
	RemoveContainer(ctx context.Context, containerID string) error
	
	// IsContainerRunning checks if a container is currently running
	IsContainerRunning(ctx context.Context, containerName string) (bool, error)
	
	// GetContainerLogs retrieves logs from a container
	GetContainerLogs(ctx context.Context, containerID string) (io.ReadCloser, error)
	
	// GetContainerStatus returns detailed container status information
	GetContainerStatus(ctx context.Context, containerName string) (*ContainerStatus, error)
	
	// CleanContainer removes specific project/account container
	CleanContainer(ctx context.Context, containerName string) error
	
	// CleanAllContainers removes all claude-reactor containers
	CleanAllContainers(ctx context.Context) error
	
	// CleanImages removes claude-reactor images
	CleanImages(ctx context.Context, all bool) error
	
	// AttachToContainer executes commands in a running container
	AttachToContainer(ctx context.Context, containerName string, command []string, interactive bool) error
	
	// HealthCheck verifies container is healthy and responsive
	HealthCheck(ctx context.Context, containerName string, maxRetries int) error
	
	// ListVariants returns available container variants
	ListVariants() ([]VariantDefinition, error)
	
	// GenerateContainerName creates unique container name with project hash
	GenerateContainerName(projectPath, variant, architecture, account string) string
	
	// GenerateProjectHash creates hash for project directory
	GenerateProjectHash(projectPath string) string
	
	// GetImageName generates image name with architecture
	GetImageName(variant, architecture string) string
	
	// BuildImageWithRegistry builds an image with registry support (Phase 0.1)
	BuildImageWithRegistry(ctx context.Context, variant, platform string, devMode, registryOff, pullLatest bool) error
}

// AuthManager handles Claude CLI authentication
type AuthManager interface {
	// GetAuthConfig returns authentication configuration for the specified account
	GetAuthConfig(account string) (*AuthConfig, error)
	
	// SetupAuth sets up authentication for the specified account
	SetupAuth(account string, apiKey string) error
	
	// ValidateAuth validates authentication for the specified account
	ValidateAuth(account string) error
	
	// IsAuthenticated checks if the specified account is authenticated
	IsAuthenticated(account string) bool
	
	// GetAccountConfigPath returns path to account-specific config directory
	GetAccountConfigPath(account string) string
	
	// SaveAPIKey saves API key to project-specific file
	SaveAPIKey(account, apiKey string) error
	
	// GetAPIKeyFile returns path to account-specific API key file
	GetAPIKeyFile(account string) string
	
	// CopyMainConfigToAccount copies main Claude config to account directory
	CopyMainConfigToAccount(account string) error
}

// Logger provides structured logging interface
type Logger interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Fatal(args ...interface{})
	
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
}

// Config represents the main application configuration
type Config struct {
	Variant     string            `yaml:"variant" validate:"oneof=base go full cloud k8s"`
	Account     string            `yaml:"account,omitempty"`
	DangerMode  bool              `yaml:"danger_mode,omitempty"`
	ProjectPath string            `yaml:"project_path,omitempty"`
	Metadata    map[string]string `yaml:"metadata,omitempty"`
}

// ContainerConfig represents Docker container configuration
type ContainerConfig struct {
	Image       string            `yaml:"image"`
	Name        string            `yaml:"name"`
	Variant     string            `yaml:"variant"`
	Platform    string            `yaml:"platform"`
	Mounts      []Mount           `yaml:"mounts,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty"`
	Ports       []string          `yaml:"ports,omitempty"`
	Command     []string          `yaml:"command,omitempty"`
	Interactive bool              `yaml:"interactive"`
	TTY         bool              `yaml:"tty"`
	Remove      bool              `yaml:"remove"`
}

// Mount represents a container mount point
type Mount struct {
	Source      string `yaml:"source"`
	Target      string `yaml:"target"`
	Type        string `yaml:"type"` // bind, volume, tmpfs
	ReadOnly    bool   `yaml:"read_only,omitempty"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Account   string `yaml:"account"`
	ConfigDir string `yaml:"config_dir"`
	APIKey    string `yaml:"api_key,omitempty"`
	Token     string `yaml:"token,omitempty"`
}

// VariantDefinition represents a container variant configuration
type VariantDefinition struct {
	Name         string            `yaml:"name"`
	Description  string            `yaml:"description"`
	BaseImage    string            `yaml:"base_image"`
	Dockerfile   string            `yaml:"dockerfile"`
	Tools        []string          `yaml:"tools,omitempty"`
	Environment  map[string]string `yaml:"environment,omitempty"`
	Dependencies []string          `yaml:"dependencies,omitempty"`
	Size         string            `yaml:"size,omitempty"`
}

// MountManager handles container mount operations
type MountManager interface {
	// ValidateMountPath validates and expands mount paths
	ValidateMountPath(path string) (string, error)
	
	// AddMountToConfig adds mount configuration to container config
	AddMountToConfig(config *ContainerConfig, sourcePath, targetPath string) error
	
	// GetMountSummary returns formatted summary of mounts
	GetMountSummary(mounts []Mount) string
	
	// UpdateMountSettings updates Claude settings for mounted directories
	UpdateMountSettings(mountPaths []string) error
}

// ContainerStatus represents container state information
type ContainerStatus struct {
	Exists  bool   `yaml:"exists"`
	Running bool   `yaml:"running"`
	Name    string `yaml:"name"`
	Image   string `yaml:"image"`
	ID      string `yaml:"id,omitempty"`
}

// AppContainer holds all application dependencies
type AppContainer struct {
	ArchDetector ArchitectureDetector
	ConfigMgr    ConfigManager
	DockerMgr    DockerManager
	AuthMgr      AuthManager
	MountMgr     MountManager
	Logger       Logger
}