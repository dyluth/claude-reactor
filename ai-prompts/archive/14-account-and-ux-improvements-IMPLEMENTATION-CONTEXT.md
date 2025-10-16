# Implementation Context for Account and UX Improvements

## Critical Implementation Context

This document contains essential context needed to implement the Account and UX Improvements feature defined in `14-account-and-ux-improvements.md`. Read this FIRST before beginning implementation.

## Project State at Design Time

### Repository Context
- **Branch**: `to-go-pruned` 
- **Last Commit**: `2f9690c feat: Complete radical simplification and host Docker support`
- **Status**: Clean working directory
- **Language**: Go 1.23.0 with Go modules

### Key Architecture Files (MUST READ FIRST)
```bash
# Essential files to understand before implementing:
/app/CLAUDE.md                                    # Project overview and workflows
/app/cmd/claude-reactor/commands/run.go           # Main container lifecycle (lines 500-520 critical)
/app/internal/reactor/config/manager.go           # Configuration persistence
/app/internal/reactor/auth/manager.go             # Authentication management  
/app/internal/reactor/docker/naming.go            # Container naming logic
/app/ai-prompts/13-host-docker-support.md         # Recently implemented host Docker feature
```

### Current Authentication Structure (CRITICAL)
Based on research from web search and code analysis:

**Claude CLI Credential Storage:**
- **Primary**: `~/.claude.json` - Main Claude CLI config (recommended for reliability)
- **Alternative**: `~/.claude/claude.json` - Managed by CLI
- **Project Settings**: `.claude/settings.json` (project level)
- **Local Settings**: `.claude/settings.local.json` (ignored by git)

**Current claude-reactor Auth Files:**
- Global config: `~/.claude-reactor/.claude.json`
- Account configs: `~/.claude-reactor/.{account}-claude.json`
- API keys: `~/.claude-reactor/.claude-reactor-{account}-env`

**AUTHENTICATION PERSISTENCE ISSUE ROOT CAUSE:**
The current mount in `/app/cmd/claude-reactor/commands/run.go:502` mounts to `/home/claude/.claude` but may be missing the actual credential files that Claude CLI expects. The web search revealed Claude uses a hierarchical config system with multiple file locations.

## Critical Code Patterns

### Current Container Naming (lines in docker/naming.go)
```go
// Current pattern: claude-reactor-{variant}-{architecture}-{project-hash}-{account}
// Project hash is 8 characters from absolute path
```

### Current Configuration Loading Pattern
```go
// From config/manager.go lines 26-73
func (m *manager) LoadConfig() (*pkg.Config, error) {
    // Reads .claude-reactor file from current directory
    // Parses bash-style key=value format
}
```

### Current Authentication Pattern  
```go
// From auth/manager.go lines 147-154
func (m *manager) GetAccountConfigPath(account string) string {
    normalizedAccount := normalizeAccount(account)
    if normalizedAccount == "default" {
        return filepath.Join(m.claudeReactorDir, ".default-claude.json")
    }
    return filepath.Join(m.claudeReactorDir, fmt.Sprintf(".%s-claude.json", normalizedAccount))
}
```

## Implementation Priorities (CRITICAL ORDER)

### Phase 1: MUST Fix Authentication First
**BLOCKER**: Current authentication persistence issue prevents proper user experience.

1. **Research exact Claude CLI credential files** - Run `claude config list` to see actual file locations
2. **Update mount logic** in `run.go:500-504` to mount ALL necessary credential files
3. **Test authentication persistence** across container restarts

### Phase 2: Directory Structure Changes
**DEPENDENCY**: Authentication must work before changing directory structure.

1. **Move `.claude-reactor` config** from project directory to session directory
2. **Implement project hash generation** from absolute path
3. **Create account-specific session directories**

### Phase 3: Container Lifecycle Intelligence  
**DEPENDENCY**: Directory structure must be in place first.

1. **Add argument detection** to run command
2. **Implement smart reuse logic**
3. **Handle container name conflicts**

## Specific Implementation Notes

### Default Account Logic (EXACT IMPLEMENTATION)
```go
func GetDefaultAccount() string {
    if user := os.Getenv("USER"); user != "" {
        return user
    }
    return "user"  // Exact fallback requested
}
```

### Project Hash Generation (EXACT SPECIFICATION)
```go
func GenerateProjectHash(projectPath string) string {
    // Must be 8 characters
    // Must use absolute path
    // Use SHA256 and take first 8 chars
    hash := sha256.Sum256([]byte(projectPath))
    return fmt.Sprintf("%x", hash)[:8]
}
```

### Container Reuse Logic (EXACT BEHAVIOR)
```go
func ShouldReuseContainer(hasArgs bool) bool {
    // Reuse ONLY when `claude-reactor run` called with NO arguments
    // Recreate when ANY arguments passed
    return !hasArgs
}
```

## Known Issues to Address

### 1. Container Name Conflict (REPORTED BUG)
```bash
# Current error when container exists:
Error: failed to start container: failed to create container: Error response from daemon: Conflict. The container name "/claude-reactor-base-arm64-5f98d015-default" is already in use
```
**Root Cause**: No logic to handle existing containers
**Solution**: Implement smart reuse logic in Phase 3

### 2. Authentication Persistence (REPORTED BUG)  
```bash
# Current behavior: Re-authentication required every container restart
# Expected: Persistent authentication across restarts for same account
```
**Root Cause**: Incorrect credential file mounting
**Solution**: Fix in Phase 1 (highest priority)

### 3. Session Data Isolation (DESIGN REQUIREMENT)
```bash
# Current: All projects share session data
# Required: Project-specific session isolation
```
**Root Cause**: Single session directory for all projects
**Solution**: Implement in Phase 2

## Testing Requirements

### Authentication Testing (CRITICAL)
```bash
# Must verify these scenarios work:
1. Start container, authenticate with Claude
2. Stop container  
3. Start container again
4. Verify no re-authentication required
5. Test with multiple accounts
6. Test with both OAuth and API key auth
```

### Multi-Account Testing
```bash
# Directory structure after implementation:
~/.claude-reactor/
├── cam/                                    # $USER account
│   ├── claude-reactor-5f98d015/           # This project session
│   │   ├── .claude-reactor                # Moved from local directory
│   │   ├── projects/                      # Claude conversation history
│   │   └── shell-snapshots/
│   └── other-project-a1b2c3d4/
├── work/                                  # Work account
│   └── frontend-e5f6g7h8/
├── .cam-claude.json                       # Account-specific Claude config
├── .work-claude.json
└── .default-claude.json
```

## Build and Development Context

### Current Build System
```bash
# Use existing Makefile for all operations:
make test-unit                    # Fast unit tests (5 seconds)
make test                        # Complete test suite
make build                       # Build claude-reactor binary
make clean-all                   # Complete cleanup
```

### Development Workflow
```bash
# Current development pattern:
go run cmd/claude-reactor/main.go run        # Test locally
./claude-reactor                             # Use built binary
make test-unit                               # Validate changes
```

## Integration Points

### Host Docker Integration (ALREADY IMPLEMENTED)
- **File**: `ai-prompts/13-host-docker-support.md`
- **Status**: Recently completed, working correctly
- **Integration**: Must maintain all host Docker functionality
- **Warning System**: Must preserve security warnings

### Makefile Integration (REQUIRED)
- **Requirement**: All operations via Makefile commands
- **Current**: 25+ targets available
- **Needed**: No changes to build system required

### Container Variants (MAINTAIN COMPATIBILITY)
- **Variants**: base, go, full, cloud, k8s
- **Architecture**: ARM64 and AMD64 support
- **Registry**: GitHub Container Registry integration
- **Requirement**: All variants must work with new account system

## Error Handling Patterns

### Configuration Errors (EXISTING PATTERN)
```go
return fmt.Errorf("failed to load configuration: %w. Try running 'claude-reactor config validate' to check setup", err)
```

### Docker Errors (EXISTING PATTERN)  
```go
return fmt.Errorf("failed to start container: %w. Check Docker daemon is running and try 'docker system prune'", err)
```

### Authentication Errors (NEW PATTERN NEEDED)
```go
return fmt.Errorf("authentication config not found for account: %s. Try running 'claude-reactor run --account %s --interactive-login'", account, account)
```

## Performance Considerations

### List Command Performance (NEW REQUIREMENT)
- **Target**: < 2 seconds response time
- **Challenge**: Scanning multiple account directories
- **Solution**: Concurrent directory scanning if needed

### Container Startup Performance (MAINTAIN)
- **Current**: Smart registry pulls with local fallback
- **Requirement**: Maintain or improve startup times
- **New Challenge**: Container reuse logic must be fast

## Security Context (CRITICAL)

### File Permissions (EXACT REQUIREMENTS)
```bash
# Session directories: 0755 (user readable)
chmod 755 ~/.claude-reactor/{account}/

# Credential files: 0600 (user only)
chmod 600 ~/.claude-reactor/.{account}-claude.json
chmod 600 ~/.claude-reactor/.claude-reactor-{account}-env
```

### Account Isolation (CRITICAL REQUIREMENT)
- **Requirement**: Complete separation between accounts
- **Test**: Account A cannot access Account B data
- **Implementation**: Separate directories, configs, containers

## Backward Compatibility Context

### No Migration Policy (EXACT REQUIREMENT)
- **Policy**: No automatic migration of existing data
- **Reason**: Explicitly requested by user
- **Implementation**: New structure only, ignore old files
- **User Impact**: Users must reconfigure in new structure

### Existing Container Compatibility
- **Requirement**: Existing containers continue to work
- **Challenge**: Container naming scheme changes
- **Solution**: Graceful handling of old container names

## Dependencies and Prerequisites

### Go Dependencies (CURRENT)
```go
// From go.mod - these are already available:
github.com/docker/docker v28.3.3+incompatible
github.com/sirupsen/logrus v1.9.3
github.com/spf13/cobra v1.9.1
github.com/stretchr/testify v1.10.0
gopkg.in/yaml.v3 v3.0.1
```

### System Dependencies
- **Docker**: Must be running for container operations
- **Claude CLI**: Must be installed in containers
- **File System**: Must support symlinks and proper permissions

### Environment Variables
- **$USER**: Primary source for default account
- **$HOME**: For ~/.claude-reactor directory location
- **Docker Environment**: All existing Docker environment variables

## Implementation Validation Checklist

Before considering implementation complete, verify:

### Core Functionality
- [ ] Authentication persists across container restarts
- [ ] Container name conflicts resolved gracefully  
- [ ] Multiple accounts work in isolation
- [ ] Project-specific session directories created
- [ ] List command shows all accounts/projects
- [ ] Clean command levels work correctly

### Integration Testing
- [ ] All existing Makefile targets still work
- [ ] Host Docker functionality preserved
- [ ] All container variants work with new account system
- [ ] No regression in container startup performance

### Security Validation  
- [ ] Account isolation verified (cannot access other account data)
- [ ] File permissions set correctly on all credential files
- [ ] No credential leakage between accounts

This context document should provide everything needed to successfully implement the Account and UX Improvements feature after any interruption or context reset.