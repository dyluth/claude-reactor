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

// DevContainerConfig represents VS Code devcontainer.json configuration
type DevContainerConfig struct {
	Name               string                 `json:"name"`
	DockerFile         string                 `json:"dockerFile,omitempty"`
	Build              *DevContainerBuild     `json:"build,omitempty"`
	Image              string                 `json:"image,omitempty"`
	Features           map[string]interface{} `json:"features,omitempty"`
	Customizations     *DevContainerCustom    `json:"customizations,omitempty"`
	ForwardPorts       []int                  `json:"forwardPorts,omitempty"`
	PostCreateCommand  interface{}            `json:"postCreateCommand,omitempty"`
	PostStartCommand   interface{}            `json:"postStartCommand,omitempty"`
	PostAttachCommand  interface{}            `json:"postAttachCommand,omitempty"`
	Mounts             []DevContainerMount    `json:"mounts,omitempty"`
	WorkspaceFolder    string                 `json:"workspaceFolder,omitempty"`
	WorkspaceMount     string                 `json:"workspaceMount,omitempty"`
	RunArgs            []string               `json:"runArgs,omitempty"`
	OverrideCommand    bool                   `json:"overrideCommand,omitempty"`
	ShutdownAction     string                 `json:"shutdownAction,omitempty"`
	UserEnvProbe       string                 `json:"userEnvProbe,omitempty"`
}

// DevContainerBuild represents build configuration
type DevContainerBuild struct {
	DockerFile string            `json:"dockerfile"`
	Context    string            `json:"context,omitempty"`
	Args       map[string]string `json:"args,omitempty"`
	Target     string            `json:"target,omitempty"`
}

// DevContainerCustom represents VS Code customizations
type DevContainerCustom struct {
	VSCode *VSCodeCustomization `json:"vscode,omitempty"`
}

// VSCodeCustomization represents VS Code-specific settings
type VSCodeCustomization struct {
	Extensions []string               `json:"extensions,omitempty"`
	Settings   map[string]interface{} `json:"settings,omitempty"`
}

// DevContainerMount represents mount configuration for devcontainers
type DevContainerMount struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
}

// ProjectDetectionResult contains enhanced project detection information
type ProjectDetectionResult struct {
	ProjectType    string            `json:"projectType"`
	Languages      []string          `json:"languages"`
	Frameworks     []string          `json:"frameworks"`
	Variant        string            `json:"variant"`
	Extensions     []string          `json:"extensions"`
	Features       []string          `json:"features"`
	Tools          []string          `json:"tools"`
	Files          []string          `json:"files"`
	Confidence     float64           `json:"confidence"`
	Metadata       map[string]string `json:"metadata"`
}

// DevContainerManager handles VS Code Dev Container integration
type DevContainerManager interface {
	// GenerateDevContainer creates .devcontainer configuration based on project detection
	GenerateDevContainer(projectPath string, config *Config) error
	
	// ValidateDevContainer validates existing .devcontainer configuration
	ValidateDevContainer(projectPath string) error
	
	// GetExtensionsForProject returns recommended VS Code extensions for detected project type
	GetExtensionsForProject(projectType string, variant string) ([]string, error)
	
	// CreateDevContainerConfig generates devcontainer.json content
	CreateDevContainerConfig(config *DevContainerConfig) ([]byte, error)
	
	// DetectProjectType performs enhanced project detection for VS Code integration
	DetectProjectType(projectPath string) (*ProjectDetectionResult, error)
	
	// UpdateDevContainer updates existing .devcontainer configuration
	UpdateDevContainer(projectPath string, config *Config) error
	
	// RemoveDevContainer removes .devcontainer directory and configurations
	RemoveDevContainer(projectPath string) error
}

// ProjectTemplate represents a project template configuration
type ProjectTemplate struct {
	Name           string            `yaml:"name"`
	Description    string            `yaml:"description"`
	Language       string            `yaml:"language"`
	Framework      string            `yaml:"framework,omitempty"`
	Variant        string            `yaml:"variant"`
	Version        string            `yaml:"version"`
	Author         string            `yaml:"author,omitempty"`
	Tags           []string          `yaml:"tags,omitempty"`
	Files          []TemplateFile    `yaml:"files"`
	Variables      []TemplateVar     `yaml:"variables,omitempty"`
	PostCreate     []string          `yaml:"post_create,omitempty"`
	Dependencies   []string          `yaml:"dependencies,omitempty"`
	GitIgnore      []string          `yaml:"git_ignore,omitempty"`
	DevContainer   bool              `yaml:"dev_container,omitempty"`
	Documentation  string            `yaml:"documentation,omitempty"`
	Requirements   map[string]string `yaml:"requirements,omitempty"`
}

// TemplateFile represents a file in a project template
type TemplateFile struct {
	Path        string `yaml:"path"`
	Content     string `yaml:"content,omitempty"`
	Source      string `yaml:"source,omitempty"`
	Executable  bool   `yaml:"executable,omitempty"`
	Template    bool   `yaml:"template,omitempty"`
	Conditional string `yaml:"conditional,omitempty"`
}

// TemplateVar represents a template variable
type TemplateVar struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"description"`
	Type        string      `yaml:"type"` // string, bool, int, choice
	Default     interface{} `yaml:"default,omitempty"`
	Required    bool        `yaml:"required,omitempty"`
	Choices     []string    `yaml:"choices,omitempty"`
	Validation  string      `yaml:"validation,omitempty"`
}

// ProjectScaffoldResult represents the result of project scaffolding
type ProjectScaffoldResult struct {
	ProjectPath     string            `json:"projectPath"`
	TemplateName    string            `json:"templateName"`
	ProjectName     string            `json:"projectName"`
	Language        string            `json:"language"`
	Framework       string            `json:"framework,omitempty"`
	Variant         string            `json:"variant"`
	FilesCreated    []string          `json:"filesCreated"`
	Variables       map[string]string `json:"variables"`
	PostCreateRan   bool              `json:"postCreateRan"`
	DevContainerGen bool              `json:"devContainerGenerated"`
	GitInitialized  bool              `json:"gitInitialized"`
}

// TemplateManager handles project template management and scaffolding
type TemplateManager interface {
	// ListTemplates returns all available project templates
	ListTemplates() ([]*ProjectTemplate, error)
	
	// GetTemplate retrieves a specific template by name
	GetTemplate(name string) (*ProjectTemplate, error)
	
	// ValidateTemplate validates template structure and content
	ValidateTemplate(template *ProjectTemplate) error
	
	// ScaffoldProject creates a new project from template
	ScaffoldProject(templateName, projectPath, projectName string, variables map[string]string) (*ProjectScaffoldResult, error)
	
	// InteractiveScaffold runs interactive project creation wizard
	InteractiveScaffold(projectPath string) (*ProjectScaffoldResult, error)
	
	// CreateTemplate creates a new template from existing project
	CreateTemplate(projectPath, templateName string) (*ProjectTemplate, error)
	
	// InstallTemplate installs template from file or URL
	InstallTemplate(source string) error
	
	// UninstallTemplate removes a template
	UninstallTemplate(name string) error
	
	// GetTemplatesForLanguage returns templates for specific language
	GetTemplatesForLanguage(language string) ([]*ProjectTemplate, error)
	
	// GetRecommendedTemplate suggests best template for project type
	GetRecommendedTemplate(detection *ProjectDetectionResult) (*ProjectTemplate, error)
	
	// RenderTemplate processes template variables in content
	RenderTemplate(content string, variables map[string]string) (string, error)
	
	// ValidateProjectName checks if project name is valid
	ValidateProjectName(name string) error
	
	// GetTemplateVariables extracts variables from template and provides defaults
	GetTemplateVariables(template *ProjectTemplate, projectName string) (map[string]string, error)
}

// DependencyInfo represents a project dependency with version and metadata
type DependencyInfo struct {
	Name            string    `json:"name"`
	CurrentVersion  string    `json:"currentVersion"`
	LatestVersion   string    `json:"latestVersion,omitempty"`
	RequestedVersion string   `json:"requestedVersion,omitempty"`
	Type            string    `json:"type"` // direct, indirect, dev
	PackageManager  string    `json:"packageManager"`
	License         string    `json:"license,omitempty"`
	Homepage        string    `json:"homepage,omitempty"`
	Description     string    `json:"description,omitempty"`
	IsOutdated      bool      `json:"isOutdated"`
	HasVulnerability bool     `json:"hasVulnerability"`
	VulnerabilityInfo []VulnerabilityInfo `json:"vulnerabilities,omitempty"`
}

// VulnerabilityInfo represents a security vulnerability
type VulnerabilityInfo struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Severity    string `json:"severity"` // low, moderate, high, critical
	CVSSScore   float64 `json:"cvssScore,omitempty"`
	FixedIn     string `json:"fixedIn,omitempty"`
	Reference   string `json:"reference,omitempty"`
}

// PackageManagerInfo represents detected package manager information
type PackageManagerInfo struct {
	Type        string   `json:"type"`        // go, cargo, npm, yarn, pnpm, pip, poetry, pipenv, maven, gradle
	Name        string   `json:"name"`        // Display name
	ConfigFiles []string `json:"configFiles"` // Configuration files found
	LockFiles   []string `json:"lockFiles"`   // Lock files found
	Version     string   `json:"version,omitempty"`
	Available   bool     `json:"available"`   // Is the package manager available in PATH
}

// DependencyOperationResult represents the result of a dependency operation
type DependencyOperationResult struct {
	PackageManager string   `json:"packageManager"`
	Operation      string   `json:"operation"`
	Success        bool     `json:"success"`
	Output         string   `json:"output,omitempty"`
	Error          string   `json:"error,omitempty"`
	DependenciesChanged []string `json:"dependenciesChanged,omitempty"`
	Duration       string   `json:"duration,omitempty"`
}

// PackageManager defines operations for a specific package manager
type PackageManager interface {
	// GetType returns the package manager type (go, cargo, npm, etc.)
	GetType() string
	
	// GetName returns the display name
	GetName() string
	
	// IsAvailable checks if the package manager is available in the system
	IsAvailable() bool
	
	// GetVersion returns the package manager version
	GetVersion() (string, error)
	
	// DetectConfigFiles finds configuration files for this package manager
	DetectConfigFiles(projectPath string) []string
	
	// ListDependencies returns all dependencies
	ListDependencies(projectPath string) ([]*DependencyInfo, error)
	
	// InstallDependencies installs all dependencies
	InstallDependencies(projectPath string) (*DependencyOperationResult, error)
	
	// UpdateDependencies updates dependencies to latest versions
	UpdateDependencies(projectPath string) (*DependencyOperationResult, error)
	
	// AuditDependencies checks for security vulnerabilities
	AuditDependencies(projectPath string) ([]*VulnerabilityInfo, error)
	
	// GetOutdatedDependencies finds dependencies with newer versions available
	GetOutdatedDependencies(projectPath string) ([]*DependencyInfo, error)
	
	// CleanCache cleans the package manager cache
	CleanCache(projectPath string) (*DependencyOperationResult, error)
}

// PackageManagerDetector detects available package managers in a project
type PackageManagerDetector interface {
	// DetectPackageManagers scans project and returns detected package managers
	DetectPackageManagers(projectPath string) ([]*PackageManagerInfo, error)
	
	// GetAvailablePackageManagers returns all package managers available on the system
	GetAvailablePackageManagers() ([]*PackageManagerInfo, error)
	
	// GetPackageManagerByType returns a specific package manager implementation
	GetPackageManagerByType(pmType string) (PackageManager, error)
}

// UnifiedDependencyManager provides unified operations across package managers
type UnifiedDependencyManager interface {
	// DetectProjectDependencies analyzes project and detects all package managers and dependencies
	DetectProjectDependencies(projectPath string) ([]*PackageManagerInfo, []*DependencyInfo, error)
	
	// InstallAllDependencies installs dependencies for all detected package managers
	InstallAllDependencies(projectPath string) ([]*DependencyOperationResult, error)
	
	// UpdateAllDependencies updates dependencies for all detected package managers
	UpdateAllDependencies(projectPath string) ([]*DependencyOperationResult, error)
	
	// AuditAllDependencies performs security audit across all package managers
	AuditAllDependencies(projectPath string) ([]*VulnerabilityInfo, error)
	
	// GetAllOutdatedDependencies finds outdated dependencies across all package managers
	GetAllOutdatedDependencies(projectPath string) ([]*DependencyInfo, error)
	
	// CleanAllCaches cleans caches for all detected package managers
	CleanAllCaches(projectPath string) ([]*DependencyOperationResult, error)
	
	// GenerateDependencyReport creates a comprehensive dependency report
	GenerateDependencyReport(projectPath string) (*DependencyReport, error)
}

// DependencyReport represents a comprehensive project dependency analysis
type DependencyReport struct {
	ProjectPath         string                    `json:"projectPath"`
	GeneratedAt         string                    `json:"generatedAt"`
	PackageManagers     []*PackageManagerInfo     `json:"packageManagers"`
	TotalDependencies   int                       `json:"totalDependencies"`
	DirectDependencies  int                       `json:"directDependencies"`
	IndirectDependencies int                      `json:"indirectDependencies"`
	OutdatedDependencies int                      `json:"outdatedDependencies"`
	Vulnerabilities     int                       `json:"vulnerabilities"`
	Dependencies        []*DependencyInfo         `json:"dependencies"`
	VulnerabilitySummary map[string]int           `json:"vulnerabilitySummary"` // severity -> count
	LicenseSummary      map[string]int            `json:"licenseSummary"`       // license -> count
	SecurityScore       float64                   `json:"securityScore"`        // 0-100
}

// AppContainer holds all application dependencies
type AppContainer struct {
	ArchDetector    ArchitectureDetector
	ConfigMgr       ConfigManager
	DockerMgr       DockerManager
	AuthMgr         AuthManager
	MountMgr        MountManager
	DevContainerMgr DevContainerManager
	TemplateMgr     TemplateManager
	DependencyMgr   UnifiedDependencyManager
	Logger          Logger
}