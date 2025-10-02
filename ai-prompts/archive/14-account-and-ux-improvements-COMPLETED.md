# Feature: Account Isolation and UX Improvements

## 1. Description

Implement complete account isolation for Claude CLI with improved directory structure, session persistence, authentication management, and enhanced user experience. This feature provides secure account separation with project-specific session storage, simplified container lifecycle management, and comprehensive listing capabilities to manage multiple accounts and projects efficiently.

## Goal statement

To provide complete isolation between different Claude accounts while sharing underlying container infrastructure, enabling users to work with multiple Claude accounts across different projects with persistent authentication, session data, and simplified container management.

## Project Analysis & current state

### Technology & architecture

**Current Architecture:**
- **Go-based CLI**: Using Cobra framework for command structure (`/app/cmd/claude-reactor/`)
- **Configuration Management**: Simple file-based config in `/app/internal/reactor/config/manager.go`
- **Authentication**: Account-specific auth handling in `/app/internal/reactor/auth/manager.go`
- **Docker Integration**: Container management via `/app/internal/reactor/docker/`
- **Mount Management**: File system mounting logic in `/app/internal/reactor/mount/`

**Key Files:**
- `/app/cmd/claude-reactor/commands/run.go` - Main container lifecycle management
- `/app/internal/reactor/config/manager.go` - Configuration persistence 
- `/app/internal/reactor/auth/manager.go` - Authentication and account management
- `/app/internal/reactor/docker/naming.go` - Container naming logic
- `/app/CLAUDE.md` - Project documentation and workflows

**Current Container Architecture:**
- Multi-stage Dockerfile with 5 variants (base, go, full, cloud, k8s)
- Container naming: `claude-reactor-{variant}-{architecture}-{project-hash}-{account}`
- Host Docker support with security warnings and timeout controls

### current state

**Authentication Structure (Current):**
- Global Claude config in `~/.claude.json`
- Account-specific configs: `~/.claude-reactor/.{account}-claude.json`
- API key files: `~/.claude-reactor/.claude-reactor-{account}-env`
- Session data mounted to `/home/claude/.claude` in containers

**Configuration Management:**
- Project-level `.claude-reactor` file stores variant, danger mode, host docker settings
- Basic account field exists but account switching not fully implemented
- No project-specific session isolation

**Container Lifecycle:**
- Containers named with project hash but account isolation incomplete
- Existing containers cause conflicts when recreating
- No smart reuse logic based on command arguments

## context & problem definition

### problem statement

**Who**: Developers using multiple Claude accounts (work, personal) across different projects requiring secure isolation and persistent authentication.

**What**: Current implementation lacks complete account isolation, has authentication persistence issues requiring re-login on container restart, and provides poor user experience for managing multiple projects and accounts.

**Why**: Users need to maintain separate Claude contexts for different purposes (work projects, personal projects) while avoiding authentication friction and ensuring complete data isolation between accounts.

### success criteria

- [ ] **Complete Account Isolation**: Each account has isolated authentication, session data, and container infrastructure
- [ ] **Persistent Authentication**: No re-authentication required when restarting containers for the same account
- [ ] **Smart Container Reuse**: Containers reused when `claude-reactor run` called with no arguments, recreated when arguments provided
- [ ] **Project Session Isolation**: Each project/account combination maintains separate conversation history and settings
- [ ] **Improved Clean Commands**: Granular cleanup options for containers, sessions, and authentication data
- [ ] **Enhanced Listing**: Clear visibility into accounts, projects, and container status
- [ ] **Backward Compatibility**: New structure only, no migration of existing data

## technical requirements

### functional requirements

#### Account Management
- [ ] Default account uses `$USER` environment variable, fallback to `"user"` if unavailable
- [ ] Account-specific directory structure: `~/.claude-reactor/{account}/`
- [ ] Account-specific Claude configuration mounting for persistent authentication
- [ ] Account-specific session directories: `~/.claude-reactor/{account}/{project-name}-{project-hash}/`
- [ ] Container naming: `claude-reactor-{variant}-{architecture}-{project-hash}-{account}`

#### Project Session Isolation  
- [ ] Project hash generated from absolute path (8-character hash)
- [ ] Project-specific session directories contain Claude conversation history
- [ ] Project-specific `.claude-reactor` config moved from local directory to session directory
- [ ] Each project/account combination has isolated container and session data

#### Container Lifecycle Management
- [ ] Smart container reuse: reuse existing container when `claude-reactor run` called with no arguments
- [ ] Container recreation: force recreate when any arguments passed to `claude-reactor run`
- [ ] Container conflict resolution: handle existing containers gracefully
- [ ] Configuration drift detection: verify container compatibility before reuse

#### Enhanced Clean Commands
- [ ] `claude-reactor clean` - containers only (current behavior)
- [ ] `claude-reactor clean --sessions` - containers + session data
- [ ] `claude-reactor clean --auth` - containers + session data + credentials  
- [ ] `claude-reactor clean --all` - everything including global cache
- [ ] Clear help text explaining what each level cleans

#### List Command Implementation
- [ ] `claude-reactor list` - flat table view showing accounts, projects, containers, last used
- [ ] `claude-reactor list --json` - JSON output for scripting
- [ ] Display account, project name, full path, container count, last used timestamp
- [ ] No filtering initially (can be added later)

#### Authentication Improvements
- [ ] Mount correct Claude credential files for persistent authentication
- [ ] Support both `~/.claude.json` and `~/.claude/` directory structures
- [ ] Ensure same credentials file used across all projects for same account
- [ ] Handle OAuth tokens and API keys correctly

### non-functional requirements

#### Security
- **Account Isolation**: Complete separation of authentication and session data between accounts
- **File Permissions**: Proper permissions on credential and session files (0600 for secrets)
- **Container Security**: Maintain existing security model with explicit opt-in for host Docker

#### Performance  
- **Fast Container Startup**: Smart reuse reduces container creation overhead
- **Efficient Storage**: Session data organized by project to avoid conflicts
- **Quick Listing**: List command responds in < 2 seconds for reasonable number of projects

#### Usability
- **Intuitive Commands**: Clear distinction between container cleanup levels
- **Helpful Error Messages**: Guide users when authentication or container issues occur
- **Consistent Interface**: Same command patterns across all operations

#### Operations
- **Comprehensive Makefile**: All development, testing, and deployment commands
- **Documentation**: Update CLAUDE.md with new account and session management workflows
- **Developer Experience**: Maintain <10 minute setup time for new developers

### Technical Constraints

- Must not break existing container infrastructure or host Docker functionality
- Must maintain compatibility with current Makefile and build system
- Must work with all existing container variants (base, go, full, cloud, k8s)
- Must support both ARM64 and AMD64 architectures
- No automatic migration of existing data - new structure only

## Data & Database changes

### Data model updates

**New Directory Structure:**
```
~/.claude-reactor/
├── {account}/                              # Account-specific session data
│   ├── {project-name}-{project-hash}/      # Project-specific sessions
│   │   ├── .claude-reactor                 # Project config (moved from local dir)
│   │   ├── projects/                       # Claude conversation history
│   │   ├── shell-snapshots/               # Shell session data
│   │   └── todos/                         # Project todos
│   └── another-project-{hash}/
├── .{account}-claude.json                  # Account-specific Claude config
├── .claude-reactor-{account}-env           # Account-specific API keys
└── .default-claude.json                   # Default account config
```

**Configuration Schema Updates:**
```go
type Config struct {
    Variant              string  // Container variant
    Account              string  // Account name ($USER default)
    DangerMode          bool    // Danger mode setting
    HostDocker          bool    // Host Docker access
    HostDockerTimeout   string  // Host Docker timeout
    SessionPersistence  bool    // Enable session persistence
    LastSessionID       string  // Last session identifier  
    ContainerID         string  // Associated container ID
    ProjectPath         string  // Project absolute path
    ProjectHash         string  // 8-character hash of project path
    Metadata            map[string]string // Additional metadata
}
```

**List Command Data Structure:**
```go
type ProjectInfo struct {
    Account       string    `json:"account"`
    ProjectName   string    `json:"project_name"`
    ProjectHash   string    `json:"project_hash"`
    ProjectPath   string    `json:"project_path"`
    Containers    []string  `json:"containers"`
    LastUsed      time.Time `json:"last_used"`
    SessionDir    string    `json:"session_dir"`
}

type ListResponse struct {
    Projects []ProjectInfo `json:"projects"`
    Summary  struct {
        TotalAccounts   int `json:"total_accounts"`
        TotalProjects   int `json:"total_projects"`
        TotalContainers int `json:"total_containers"`
    } `json:"summary"`
}
```

### Data migration plan

**N/A** - No migration required. New structure only per requirements. Existing `.claude-reactor` files and session data remain unused.

## API & Backend changes

### Data access pattern

**File System Operations**: Direct file system access for configuration and session management using Go's `os` and `filepath` packages with proper error handling and atomic operations.

**Docker API Integration**: Continue using existing Docker client for container management with enhanced naming and lifecycle logic.

### server actions

#### Account Management Functions
```go
// GetDefaultAccount returns $USER or "user" fallback
func GetDefaultAccount() string

// NormalizeAccount handles empty account defaulting
func NormalizeAccount(account string) string

// GetAccountSessionDir returns account-specific session directory
func GetAccountSessionDir(account string) string

// GetProjectSessionDir returns project-specific session directory  
func GetProjectSessionDir(account, projectPath string) string
```

#### Project Hash Functions
```go
// GenerateProjectHash creates 8-character hash from absolute path
func GenerateProjectHash(projectPath string) string

// GetProjectNameFromPath extracts project name from path
func GetProjectNameFromPath(projectPath string) string

// GetProjectInfo extracts project metadata from path
func GetProjectInfo(projectPath string) (*ProjectInfo, error)
```

#### Container Lifecycle Functions
```go
// ShouldReuseContainer determines if existing container should be reused
func ShouldReuseContainer(hasArgs bool, existingContainer *ContainerInfo) bool

// GetContainerByName retrieves container info by exact name match
func GetContainerByName(containerName string) (*ContainerInfo, error)

// ValidateContainerConfig checks if existing container config matches current
func ValidateContainerConfig(container *ContainerInfo, config *Config) error
```

#### List Command Functions
```go
// ListProjects scans ~/.claude-reactor for all accounts and projects
func ListProjects() ([]ProjectInfo, error)

// GetProjectLastUsed determines last used timestamp from container/session data
func GetProjectLastUsed(sessionDir string, containers []string) time.Time

// FormatProjectList renders projects in table format
func FormatProjectList(projects []ProjectInfo) string

// FormatProjectListJSON renders projects in JSON format
func FormatProjectListJSON(projects []ProjectInfo) ([]byte, error)
```

### Database queries

**N/A** - File system based storage only. No database queries required.

### API Routes  

**N/A** - CLI application. No HTTP API routes.

## frontend changes

### New components

**N/A** - CLI application. No frontend components.

### Page updates

**N/A** - CLI application. No web interface.

## Implementation plan

### phase 1 - Core Account and Directory Structure

**Goal**: Implement new account-based directory structure and default account logic

**Tasks**:
- [ ] Update `GetDefaultAccount()` to use `$USER` with `"user"` fallback
- [ ] Implement `GenerateProjectHash()` for 8-character hash from absolute path
- [ ] Create `GetProjectSessionDir()` for account/project-specific session paths
- [ ] Update `AuthManager.GetAccountSessionDir()` to use new structure
- [ ] Update mount logic to use project-specific session directories
- [ ] Move `.claude-reactor` config to session directory instead of local directory
- [ ] Test account isolation with multiple test accounts

### phase 2 - Smart Container Lifecycle Management

**Goal**: Implement intelligent container reuse and recreation logic  

**Tasks**:
- [ ] Add argument detection to `run` command (check if any flags/args passed)
- [ ] Implement `ShouldReuseContainer()` logic based on arguments
- [ ] Add `GetContainerByName()` for existing container detection
- [ ] Implement `ValidateContainerConfig()` for configuration drift detection
- [ ] Update container startup logic to handle reuse vs recreation
- [ ] Add proper error handling for container name conflicts
- [ ] Test container reuse scenarios with different configurations

### phase 3 - Enhanced Authentication and Credential Management

**Goal**: Fix authentication persistence and credential mounting

**Tasks**:
- [ ] Research and identify correct Claude CLI credential file locations
- [ ] Update mount logic to include all necessary credential files
- [ ] Ensure account-specific credential isolation
- [ ] Test authentication persistence across container restarts
- [ ] Verify OAuth token and API key handling
- [ ] Document credential file requirements in CLAUDE.md

### phase 4 - List Command Implementation

**Goal**: Provide visibility into accounts, projects, and containers

**Tasks**:
- [ ] Create new `list` command in `/app/cmd/claude-reactor/commands/list.go`
- [ ] Implement `ListProjects()` to scan `~/.claude-reactor` structure
- [ ] Add project metadata extraction (name, hash, path, containers)
- [ ] Implement flat table formatting for default output
- [ ] Add `--json` flag for JSON output format
- [ ] Include last used timestamp calculation from container/session data
- [ ] Test with multiple accounts and projects

### phase 5 - Enhanced Clean Command

**Goal**: Provide granular cleanup options for different data types

**Tasks**:
- [ ] Add `--sessions` flag to clean command for session data cleanup
- [ ] Add `--auth` flag for authentication data cleanup  
- [ ] Add `--all` flag for complete cleanup including cache
- [ ] Implement session directory cleanup logic
- [ ] Implement authentication file cleanup logic
- [ ] Update help text to clearly explain each cleanup level
- [ ] Add confirmation prompts for destructive operations
- [ ] Test cleanup operations with various data scenarios

### phase 6 - Documentation and Operations

**Goal**: Update documentation and ensure operational excellence

**Tasks**:
- [ ] Update `/app/CLAUDE.md` with new account and session management workflows
- [ ] Add troubleshooting section for authentication and container issues
- [ ] Update Makefile targets for new testing scenarios
- [ ] Create example workflows for multi-account usage
- [ ] Add developer onboarding documentation for new structure
- [ ] Validate <10 minute developer onboarding experience
- [ ] Document best practices for account and project management

## 5. Testing Strategy

### Unit Tests

**Account Management Functions**:
- `GetDefaultAccount()` with various `$USER` values and fallback scenarios
- `GenerateProjectHash()` with different project paths ensuring 8-character output
- `GetProjectSessionDir()` path generation with various account/project combinations
- `NormalizeAccount()` handling of empty, whitespace, and valid account names

**Container Lifecycle Functions**:
- `ShouldReuseContainer()` logic with different argument combinations
- `ValidateContainerConfig()` with matching and mismatched configurations
- Container name generation with various account/project/variant combinations

**List Command Functions**:
- `ListProjects()` with various directory structures and missing directories
- Project metadata extraction with edge cases (special characters, long paths)
- Table and JSON formatting with different data sets

**Configuration Management**:
- Session directory creation and permission handling
- Config file movement from local directory to session directory
- Account-specific config isolation and validation

### Integration Tests

**Multi-Account Workflows**:
- Create containers with different accounts and verify isolation
- Switch between accounts and verify correct authentication and session data
- Test account-specific credential mounting and persistence

**Container Lifecycle Integration**:
- Test container reuse when calling `claude-reactor run` with no arguments
- Test container recreation when calling `claude-reactor run` with arguments
- Verify container name conflicts are handled gracefully

**Clean Command Integration**:
- Test each cleanup level (`clean`, `--sessions`, `--auth`, `--all`)
- Verify only intended data is removed at each level
- Test cleanup with multiple accounts and projects

**Authentication Persistence**:
- Verify authentication persists across container restarts
- Test credential file mounting for all supported Claude CLI storage locations
- Validate account isolation prevents credential leakage

### End-to-End (E2E) Tests

**Multi-Account Development Workflow**:
1. Initialize default account in project A
2. Create and use work account in project B  
3. Switch back to default account in project A
4. Verify separate session data and authentication
5. Use list command to view all accounts and projects
6. Clean specific account data and verify isolation

**Container Lifecycle E2E**:
1. Start container with `claude-reactor run`
2. Verify container reuse with subsequent `claude-reactor run`
3. Force recreation with `claude-reactor run --image go`
4. Verify new container created with correct configuration

**Authentication Persistence E2E**:
1. Authenticate with Claude CLI in container
2. Exit and restart container
3. Verify authentication persists without re-login
4. Test with different accounts in different projects

## 6. Security Considerations

### Authentication & Authorization

**Account Isolation**: Each account has completely separate authentication credentials and session data with no cross-account access possible.

**File Permissions**: 
- Session directories: `0755` (readable by user)
- Credential files: `0600` (readable by user only)
- API key files: `0600` (readable by user only)

**Container Security**: Maintain existing security model with explicit opt-in for host Docker access and prominent security warnings.

### Data Validation & Sanitization

**Account Name Validation**: Sanitize account names to prevent directory traversal attacks and ensure valid filesystem names.

**Project Path Validation**: Validate project paths are absolute and within expected boundaries to prevent malicious path manipulation.

**Configuration Validation**: Validate all configuration values against expected schemas to prevent injection attacks.

### Potential Vulnerabilities

**Directory Traversal**: Account names and project paths must be validated to prevent accessing files outside intended directories.

**Credential Leakage**: Ensure account-specific credentials are properly isolated and cannot be accessed by other accounts.

**Container Privilege Escalation**: Maintain existing container security model and warnings for host Docker access.

**Mitigation Strategies**:
- Strict input validation for all user-provided data
- Proper file permissions on all credential and session files
- Regular security audits of credential mounting logic
- Clear documentation of security implications

## 7. Rollout & Deployment

### Feature Flags

**N/A** - No feature flags required. New structure is used immediately without migration.

### Deployment Steps

1. **Standard Build Process**: Use existing `make build` and CI/CD pipeline
2. **No Database Migration**: File system changes only, no special deployment steps
3. **Backward Compatibility**: Existing containers continue to work during transition
4. **Documentation Update**: Update CLAUDE.md with new workflows

### Rollback Plan

**Immediate Rollback**: 
- Revert to previous version using standard deployment rollback
- Existing containers and old `.claude-reactor` files continue to work
- No data loss as new structure is additive

**Partial Feature Disable**: Not applicable as changes are structural improvements rather than feature flags.

## 8. Open Questions & Assumptions

### Assumptions

- Users understand that existing `.claude-reactor` files in project directories will no longer be used
- Claude CLI credential file locations are consistent across installations and can be reliably detected
- Container reuse based on command arguments provides good user experience balance
- 8-character project hash provides sufficient uniqueness for typical project sets
- Flat table view is preferred over hierarchical view for list command default output

### Implementation Notes

- Project hash collision handling: Current 8-character hash provides 16^8 = ~4.3 billion combinations, sufficient for typical use
- Container name length: New naming scheme may approach Docker's container name limits with very long account names
- Session data growth: No automatic cleanup of old session data - users must manually clean
- Cross-platform compatibility: Path handling must work correctly on Windows, macOS, and Linux
- Performance with many projects: List command performance with hundreds of projects needs monitoring

### Future Considerations

- **Automatic session cleanup**: Add commands to clean old/unused session data
- **Project templates**: Pre-configured project types with specific container variants
- **Team sharing**: Mechanisms to share project configurations across team members
- **Cloud synchronization**: Integration with cloud storage for session data backup
- **Enhanced security**: Additional security controls for sensitive projects or accounts