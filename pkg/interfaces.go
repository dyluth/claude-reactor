package pkg

import (
	"context"
	"io"

	"github.com/docker/docker/client"
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

	// StartOrRecoverContainer starts a new container or recovers an existing one based on session persistence
	StartOrRecoverContainer(ctx context.Context, config *ContainerConfig, sessionConfig *Config) (string, error)

	// IsContainerHealthy checks if a container exists and is healthy for session recovery
	IsContainerHealthy(ctx context.Context, containerID string) (bool, error)

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

	// GetClient returns the underlying Docker client for advanced operations
	GetClient() *client.Client
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

	// GetAccountConfigPath returns path to account-specific config file
	GetAccountConfigPath(account string) string

	// GetAccountSessionDir returns path to account-specific Claude session directory
	GetAccountSessionDir(account string) string

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
	Variant            string            `yaml:"variant" validate:"oneof=base go full cloud k8s"`
	Account            string            `yaml:"account,omitempty"`
	DangerMode         bool              `yaml:"danger_mode,omitempty"`
	ProjectPath        string            `yaml:"project_path,omitempty"`
	SessionPersistence bool              `yaml:"session_persistence,omitempty"`
	LastSessionID      string            `yaml:"last_session_id,omitempty"`
	ContainerID        string            `yaml:"container_id,omitempty"`
	Metadata           map[string]string `yaml:"metadata,omitempty"`
}

// ContainerConfig represents Docker container configuration
type ContainerConfig struct {
	Image            string            `yaml:"image"`
	Name             string            `yaml:"name"`
	Variant          string            `yaml:"variant"`
	Platform         string            `yaml:"platform"`
	Mounts           []Mount           `yaml:"mounts,omitempty"`
	Environment      map[string]string `yaml:"environment,omitempty"`
	Ports            []string          `yaml:"ports,omitempty"`
	Command          []string          `yaml:"command,omitempty"`
	Interactive      bool              `yaml:"interactive"`
	TTY              bool              `yaml:"tty"`
	Remove           bool              `yaml:"remove"`
	RunClaudeUpgrade bool              `yaml:"run_claude_upgrade,omitempty"`
}

// Mount represents a container mount point
type Mount struct {
	Source   string `yaml:"source"`
	Target   string `yaml:"target"`
	Type     string `yaml:"type"` // bind, volume, tmpfs
	ReadOnly bool   `yaml:"read_only,omitempty"`
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
	Name              string                 `json:"name"`
	DockerFile        string                 `json:"dockerFile,omitempty"`
	Build             *DevContainerBuild     `json:"build,omitempty"`
	Image             string                 `json:"image,omitempty"`
	Features          map[string]interface{} `json:"features,omitempty"`
	Customizations    *DevContainerCustom    `json:"customizations,omitempty"`
	ForwardPorts      []int                  `json:"forwardPorts,omitempty"`
	PostCreateCommand interface{}            `json:"postCreateCommand,omitempty"`
	PostStartCommand  interface{}            `json:"postStartCommand,omitempty"`
	PostAttachCommand interface{}            `json:"postAttachCommand,omitempty"`
	Mounts            []DevContainerMount    `json:"mounts,omitempty"`
	WorkspaceFolder   string                 `json:"workspaceFolder,omitempty"`
	WorkspaceMount    string                 `json:"workspaceMount,omitempty"`
	RunArgs           []string               `json:"runArgs,omitempty"`
	OverrideCommand   bool                   `json:"overrideCommand,omitempty"`
	ShutdownAction    string                 `json:"shutdownAction,omitempty"`
	UserEnvProbe      string                 `json:"userEnvProbe,omitempty"`
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
	ProjectType string            `json:"projectType"`
	Languages   []string          `json:"languages"`
	Frameworks  []string          `json:"frameworks"`
	Variant     string            `json:"variant"`
	Extensions  []string          `json:"extensions"`
	Features    []string          `json:"features"`
	Tools       []string          `json:"tools"`
	Files       []string          `json:"files"`
	Confidence  float64           `json:"confidence"`
	Metadata    map[string]string `json:"metadata"`
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
	Name          string            `yaml:"name"`
	Description   string            `yaml:"description"`
	Language      string            `yaml:"language"`
	Framework     string            `yaml:"framework,omitempty"`
	Variant       string            `yaml:"variant"`
	Version       string            `yaml:"version"`
	Author        string            `yaml:"author,omitempty"`
	Tags          []string          `yaml:"tags,omitempty"`
	Files         []TemplateFile    `yaml:"files"`
	Variables     []TemplateVar     `yaml:"variables,omitempty"`
	PostCreate    []string          `yaml:"post_create,omitempty"`
	Dependencies  []string          `yaml:"dependencies,omitempty"`
	GitIgnore     []string          `yaml:"git_ignore,omitempty"`
	DevContainer  bool              `yaml:"dev_container,omitempty"`
	Documentation string            `yaml:"documentation,omitempty"`
	Requirements  map[string]string `yaml:"requirements,omitempty"`
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
	Name              string              `json:"name"`
	CurrentVersion    string              `json:"currentVersion"`
	LatestVersion     string              `json:"latestVersion,omitempty"`
	RequestedVersion  string              `json:"requestedVersion,omitempty"`
	Type              string              `json:"type"` // direct, indirect, dev
	PackageManager    string              `json:"packageManager"`
	License           string              `json:"license,omitempty"`
	Homepage          string              `json:"homepage,omitempty"`
	Description       string              `json:"description,omitempty"`
	IsOutdated        bool                `json:"isOutdated"`
	HasVulnerability  bool                `json:"hasVulnerability"`
	VulnerabilityInfo []VulnerabilityInfo `json:"vulnerabilities,omitempty"`
}

// VulnerabilityInfo represents a security vulnerability
type VulnerabilityInfo struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Severity    string  `json:"severity"` // low, moderate, high, critical
	CVSSScore   float64 `json:"cvssScore,omitempty"`
	FixedIn     string  `json:"fixedIn,omitempty"`
	Reference   string  `json:"reference,omitempty"`
}

// PackageManagerInfo represents detected package manager information
type PackageManagerInfo struct {
	Type        string   `json:"type"`        // go, cargo, npm, yarn, pnpm, pip, poetry, pipenv, maven, gradle
	Name        string   `json:"name"`        // Display name
	ConfigFiles []string `json:"configFiles"` // Configuration files found
	LockFiles   []string `json:"lockFiles"`   // Lock files found
	Version     string   `json:"version,omitempty"`
	Available   bool     `json:"available"` // Is the package manager available in PATH
}

// DependencyOperationResult represents the result of a dependency operation
type DependencyOperationResult struct {
	PackageManager      string   `json:"packageManager"`
	Operation           string   `json:"operation"`
	Success             bool     `json:"success"`
	Output              string   `json:"output,omitempty"`
	Error               string   `json:"error,omitempty"`
	DependenciesChanged []string `json:"dependenciesChanged,omitempty"`
	Duration            string   `json:"duration,omitempty"`
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
	ProjectPath          string                `json:"projectPath"`
	GeneratedAt          string                `json:"generatedAt"`
	PackageManagers      []*PackageManagerInfo `json:"packageManagers"`
	TotalDependencies    int                   `json:"totalDependencies"`
	DirectDependencies   int                   `json:"directDependencies"`
	IndirectDependencies int                   `json:"indirectDependencies"`
	OutdatedDependencies int                   `json:"outdatedDependencies"`
	Vulnerabilities      int                   `json:"vulnerabilities"`
	Dependencies         []*DependencyInfo     `json:"dependencies"`
	VulnerabilitySummary map[string]int        `json:"vulnerabilitySummary"` // severity -> count
	LicenseSummary       map[string]int        `json:"licenseSummary"`       // license -> count
	SecurityScore        float64               `json:"securityScore"`        // 0-100
}

// =============================================================================
// Hot Reload & File Watching Interfaces
// =============================================================================

// FileWatcher monitors filesystem changes and triggers appropriate actions
type FileWatcher interface {
	// StartWatching begins monitoring the specified directory for changes
	StartWatching(projectPath string, config *WatchConfig) (*WatchSession, error)

	// StopWatching stops monitoring and cleans up resources
	StopWatching(sessionID string) error

	// GetActiveSessions returns all active watch sessions
	GetActiveSessions() ([]*WatchSession, error)

	// UpdateWatchConfig updates the configuration for an active session
	UpdateWatchConfig(sessionID string, config *WatchConfig) error
}

// WatchConfig defines what files to watch and how to handle changes
type WatchConfig struct {
	// Patterns to include (glob patterns)
	IncludePatterns []string `json:"includePatterns"`

	// Patterns to exclude (glob patterns)
	ExcludePatterns []string `json:"excludePatterns"`

	// Debounce delay in milliseconds (default: 500ms)
	DebounceDelay int `json:"debounceDelay"`

	// Enable recursive watching of subdirectories
	Recursive bool `json:"recursive"`

	// Actions to trigger on file changes
	Actions []*WatchAction `json:"actions"`

	// Project type for language-specific optimizations
	ProjectType string `json:"projectType,omitempty"`

	// Container name to sync changes to
	ContainerName string `json:"containerName,omitempty"`

	// Enable build triggering
	EnableBuild bool `json:"enableBuild"`

	// Enable hot reload for supported frameworks
	EnableHotReload bool `json:"enableHotReload"`

	// Custom build command override
	BuildCommand []string `json:"buildCommand,omitempty"`
}

// WatchAction defines an action to take when files change
type WatchAction struct {
	// Type of action (build, sync, restart, custom)
	Type string `json:"type"`

	// File patterns that trigger this action
	Triggers []string `json:"triggers"`

	// Command to execute for custom actions
	Command []string `json:"command,omitempty"`

	// Working directory for command execution
	WorkingDir string `json:"workingDir,omitempty"`

	// Environment variables for command
	Environment map[string]string `json:"environment,omitempty"`

	// Whether to run action in container vs host
	InContainer bool `json:"inContainer"`

	// Timeout for action execution (milliseconds)
	Timeout int `json:"timeout,omitempty"`

	// Whether action should block further processing
	Blocking bool `json:"blocking"`
}

// WatchSession represents an active file watching session
type WatchSession struct {
	// Unique session identifier
	ID string `json:"id"`

	// Project path being watched
	ProjectPath string `json:"projectPath"`

	// Watch configuration
	Config *WatchConfig `json:"config"`

	// Session start time
	StartTime string `json:"startTime"`

	// Current status (active, paused, error)
	Status string `json:"status"`

	// Statistics about file changes
	Stats *WatchStats `json:"stats"`

	// Last error if any
	LastError string `json:"lastError,omitempty"`

	// Associated container ID
	ContainerID string `json:"containerID,omitempty"`
}

// WatchStats tracks file watching statistics
type WatchStats struct {
	// Total events processed
	TotalEvents int `json:"totalEvents"`

	// Events by type (created, modified, deleted, renamed)
	EventsByType map[string]int `json:"eventsByType"`

	// Actions triggered
	ActionsTriggered int `json:"actionsTriggered"`

	// Successful builds
	SuccessfulBuilds int `json:"successfulBuilds"`

	// Failed builds
	FailedBuilds int `json:"failedBuilds"`

	// Files synced to container
	FilesSynced int `json:"filesSynced"`

	// Last activity timestamp
	LastActivity string `json:"lastActivity"`
}

// FileEvent represents a filesystem change event
type FileEvent struct {
	// Type of event (created, modified, deleted, renamed)
	Type string `json:"type"`

	// Path of the affected file
	Path string `json:"path"`

	// Timestamp of the event
	Timestamp string `json:"timestamp"`

	// File size (for created/modified events)
	Size int64 `json:"size,omitempty"`

	// Previous path (for rename events)
	OldPath string `json:"oldPath,omitempty"`

	// Whether this is a directory
	IsDirectory bool `json:"isDirectory"`
}

// BuildTrigger handles language-specific build operations
type BuildTrigger interface {
	// DetectProjectType identifies the project type and build requirements
	DetectProjectType(projectPath string) (*ProjectBuildInfo, error)

	// GetBuildCommand returns the appropriate build command for the project
	GetBuildCommand(projectPath string, projectType string) ([]string, error)

	// ExecuteBuild runs the build process and returns the result
	ExecuteBuild(projectPath string, buildCmd []string, options *BuildOptions) (*BuildResult, error)

	// GetHotReloadConfig returns hot reload configuration for supported frameworks
	GetHotReloadConfig(projectPath string, projectType string) (*HotReloadConfig, error)

	// ValidateBuildResult checks if the build was successful
	ValidateBuildResult(result *BuildResult) error
}

// ProjectBuildInfo contains detected build information
type ProjectBuildInfo struct {
	// Primary project type (go, rust, nodejs, python, java)
	Type string `json:"type"`

	// Framework or build tool (gin, react, django, spring, etc.)
	Framework string `json:"framework,omitempty"`

	// Version of language/runtime detected
	Version string `json:"version,omitempty"`

	// Default build command
	BuildCommand []string `json:"buildCommand"`

	// Test command
	TestCommand []string `json:"testCommand,omitempty"`

	// Start/serve command
	StartCommand []string `json:"startCommand,omitempty"`

	// Common files to watch for changes
	WatchPatterns []string `json:"watchPatterns"`

	// Common files to ignore
	IgnorePatterns []string `json:"ignorePatterns"`

	// Whether hot reload is supported
	SupportsHotReload bool `json:"supportsHotReload"`

	// Confidence score (0-100)
	Confidence float64 `json:"confidence"`
}

// BuildOptions configures build execution
type BuildOptions struct {
	// Working directory for build
	WorkingDir string `json:"workingDir,omitempty"`

	// Environment variables
	Environment map[string]string `json:"environment,omitempty"`

	// Build timeout in seconds
	Timeout int `json:"timeout,omitempty"`

	// Whether to run build in container
	InContainer bool `json:"inContainer"`

	// Container name (if InContainer is true)
	ContainerName string `json:"containerName,omitempty"`

	// Whether to capture build output
	CaptureOutput bool `json:"captureOutput"`

	// Verbose logging
	Verbose bool `json:"verbose"`
}

// BuildResult represents the result of a build operation
type BuildResult struct {
	// Whether build was successful
	Success bool `json:"success"`

	// Build command executed
	Command []string `json:"command"`

	// Working directory
	WorkingDir string `json:"workingDir"`

	// Build output (stdout + stderr)
	Output string `json:"output"`

	// Error message if failed
	Error string `json:"error,omitempty"`

	// Exit code
	ExitCode int `json:"exitCode"`

	// Build duration
	Duration string `json:"duration"`

	// Timestamp
	Timestamp string `json:"timestamp"`

	// Files generated/modified by build
	GeneratedFiles []string `json:"generatedFiles,omitempty"`
}

// HotReloadConfig defines hot reload behavior for a project
type HotReloadConfig struct {
	// Whether hot reload is enabled
	Enabled bool `json:"enabled"`

	// Port for hot reload server (if applicable)
	Port int `json:"port,omitempty"`

	// Host for hot reload server
	Host string `json:"host,omitempty"`

	// Proxy configuration for API calls
	ProxyConfig map[string]string `json:"proxyConfig,omitempty"`

	// Additional environment variables needed
	Environment map[string]string `json:"environment,omitempty"`

	// Files that trigger full page reload vs hot module replacement
	FullReloadPatterns []string `json:"fullReloadPatterns,omitempty"`

	// Command to start hot reload server
	StartCommand []string `json:"startCommand,omitempty"`
}

// ContainerSync handles syncing files between host and container
type ContainerSync interface {
	// SyncFile synchronizes a single file to the container
	SyncFile(containerID string, hostPath string, containerPath string) error

	// SyncDirectory synchronizes a directory to the container
	SyncDirectory(containerID string, hostPath string, containerPath string, options *SyncOptions) error

	// GetSyncStatus returns the current sync status
	GetSyncStatus(containerID string) (*SyncStatus, error)

	// StartContinuousSync begins continuous file synchronization
	StartContinuousSync(containerID string, mappings []*SyncMapping) (*SyncSession, error)

	// StopContinuousSync stops continuous synchronization
	StopContinuousSync(sessionID string) error
}

// SyncOptions configures file synchronization
type SyncOptions struct {
	// Preserve file permissions
	PreservePermissions bool `json:"preservePermissions"`

	// Preserve file timestamps
	PreserveTimestamps bool `json:"preserveTimestamps"`

	// Delete files in container that don't exist on host
	DeleteExtraneous bool `json:"deleteExtraneous"`

	// Patterns to exclude from sync
	ExcludePatterns []string `json:"excludePatterns"`

	// Whether to sync recursively
	Recursive bool `json:"recursive"`

	// Compression for transfer
	Compress bool `json:"compress"`

	// Dry run (don't actually sync)
	DryRun bool `json:"dryRun"`
}

// SyncMapping defines a host-to-container path mapping
type SyncMapping struct {
	// Host path
	HostPath string `json:"hostPath"`

	// Container path
	ContainerPath string `json:"containerPath"`

	// Sync direction (host-to-container, container-to-host, bidirectional)
	Direction string `json:"direction"`

	// Watch patterns for this mapping
	WatchPatterns []string `json:"watchPatterns,omitempty"`

	// Exclude patterns for this mapping
	ExcludePatterns []string `json:"excludePatterns,omitempty"`
}

// SyncSession represents an active sync session
type SyncSession struct {
	// Session ID
	ID string `json:"id"`

	// Container ID
	ContainerID string `json:"containerID"`

	// Sync mappings
	Mappings []*SyncMapping `json:"mappings"`

	// Session start time
	StartTime string `json:"startTime"`

	// Current status
	Status string `json:"status"`

	// Statistics
	Stats *SyncStats `json:"stats"`
}

// SyncStatus represents current synchronization status
type SyncStatus struct {
	// Container ID
	ContainerID string `json:"containerID"`

	// Whether sync is active
	Active bool `json:"active"`

	// Last sync time
	LastSync string `json:"lastSync,omitempty"`

	// Number of files synced
	FilesSynced int `json:"filesSynced"`

	// Any sync errors
	Errors []string `json:"errors,omitempty"`
}

// SyncStats tracks synchronization statistics
type SyncStats struct {
	// Total files synced
	TotalFiles int `json:"totalFiles"`

	// Total data transferred (bytes)
	BytesTransferred int64 `json:"bytesTransferred"`

	// Sync operations performed
	SyncOperations int `json:"syncOperations"`

	// Failed sync attempts
	FailedSyncs int `json:"failedSyncs"`

	// Average sync time
	AverageSyncTime string `json:"averageSyncTime"`

	// Last sync duration
	LastSyncDuration string `json:"lastSyncDuration"`
}

// HotReloadManager orchestrates file watching, building, and syncing
type HotReloadManager interface {
	// StartHotReload begins hot reload session for a project
	StartHotReload(projectPath string, containerID string, options *HotReloadOptions) (*HotReloadSession, error)

	// StopHotReload stops hot reload session
	StopHotReload(sessionID string) error

	// GetHotReloadSessions returns all active sessions
	GetHotReloadSessions() ([]*HotReloadSession, error)

	// UpdateHotReloadConfig updates configuration for active session
	UpdateHotReloadConfig(sessionID string, options *HotReloadOptions) error

	// GetHotReloadStatus returns current status of a session
	GetHotReloadStatus(sessionID string) (*HotReloadStatus, error)
}

// HotReloadOptions configures hot reload behavior
type HotReloadOptions struct {
	// Auto-detect project type and configure accordingly
	AutoDetect bool `json:"autoDetect"`

	// Custom watch configuration
	WatchConfig *WatchConfig `json:"watchConfig,omitempty"`

	// Build configuration
	BuildOptions *BuildOptions `json:"buildOptions,omitempty"`

	// Sync configuration
	SyncOptions *SyncOptions `json:"syncOptions,omitempty"`

	// Hot reload configuration
	HotReloadConfig *HotReloadConfig `json:"hotReloadConfig,omitempty"`

	// Enable notifications
	EnableNotifications bool `json:"enableNotifications"`

	// Custom notification webhook
	NotificationWebhook string `json:"notificationWebhook,omitempty"`
}

// HotReloadSession represents an active hot reload session
type HotReloadSession struct {
	// Session ID
	ID string `json:"id"`

	// Project path
	ProjectPath string `json:"projectPath"`

	// Container ID
	ContainerID string `json:"containerID"`

	// Project build info
	ProjectInfo *ProjectBuildInfo `json:"projectInfo"`

	// Hot reload options
	Options *HotReloadOptions `json:"options"`

	// Watch session
	WatchSession *WatchSession `json:"watchSession,omitempty"`

	// Sync session
	SyncSession *SyncSession `json:"syncSession,omitempty"`

	// Session start time
	StartTime string `json:"startTime"`

	// Current status
	Status string `json:"status"`

	// Last activity
	LastActivity string `json:"lastActivity,omitempty"`

	// Error information
	Error string `json:"error,omitempty"`
}

// HotReloadStatus provides current status of hot reload session
type HotReloadStatus struct {
	// Session ID
	SessionID string `json:"sessionID"`

	// Overall status (active, paused, error, stopped)
	Status string `json:"status"`

	// File watching status
	WatchingStatus string `json:"watchingStatus"`

	// Build status (idle, building, success, failed)
	BuildStatus string `json:"buildStatus"`

	// Sync status
	SyncStatus string `json:"syncStatus"`

	// Hot reload server status
	HotReloadStatus string `json:"hotReloadStatus"`

	// Recent activities
	RecentActivity []*ActivityEvent `json:"recentActivity,omitempty"`

	// Performance metrics
	Metrics *HotReloadMetrics `json:"metrics,omitempty"`
}

// ActivityEvent represents a hot reload activity
type ActivityEvent struct {
	// Timestamp
	Timestamp string `json:"timestamp"`

	// Event type (file_changed, build_started, build_completed, sync_completed, etc.)
	Type string `json:"type"`

	// Event message
	Message string `json:"message"`

	// Event level (info, warning, error)
	Level string `json:"level"`

	// Additional data
	Data map[string]interface{} `json:"data,omitempty"`
}

// HotReloadMetrics tracks performance metrics
type HotReloadMetrics struct {
	// Total file changes processed
	TotalChanges int `json:"totalChanges"`

	// Average build time
	AverageBuildTime string `json:"averageBuildTime"`

	// Average sync time
	AverageSyncTime string `json:"averageSyncTime"`

	// Success rate for builds
	BuildSuccessRate float64 `json:"buildSuccessRate"`

	// Total uptime
	Uptime string `json:"uptime"`

	// Memory usage (if available)
	MemoryUsage int64 `json:"memoryUsage,omitempty"`

	// CPU usage (if available)
	CPUUsage float64 `json:"cpuUsage,omitempty"`
}

// =============================================================================
// Image Validation Interfaces
// =============================================================================

// ImageValidator handles Docker image validation for custom images
type ImageValidator interface {
	// ValidateImage validates a Docker image for claude-reactor compatibility
	ValidateImage(ctx context.Context, imageName string, pullIfNeeded bool) (*ImageValidationResult, error)

	// ClearCache removes all cached validation results
	ClearCache() error

	// ClearSessionWarnings resets session warning tracking
	ClearSessionWarnings()
}

// ImageValidationResult represents the result of image validation
type ImageValidationResult struct {
	Compatible   bool                   `json:"compatible"`
	Digest       string                 `json:"digest"`
	Architecture string                 `json:"architecture"`
	Platform     string                 `json:"platform"`
	Size         int64                  `json:"size"`
	HasClaude    bool                   `json:"has_claude"`
	IsLinux      bool                   `json:"is_linux"`
	Warnings     []string               `json:"warnings"`
	Errors       []string               `json:"errors"`
	ValidatedAt  string                 `json:"validated_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
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
	FileWatcher     FileWatcher
	BuildTrigger    BuildTrigger
	ContainerSync   ContainerSync
	HotReloadMgr    HotReloadManager
	ImageValidator  ImageValidator
	Logger          Logger
	Debug           bool
}

// FabricOrchestrator interface for the reactor-fabric orchestrator
type FabricOrchestrator interface {
	// Start starts the orchestrator with the given configuration
	Start(ctx context.Context, configPath, listenAddr string) error
}
