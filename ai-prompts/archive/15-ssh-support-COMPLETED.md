# Feature: SSH Support (Agent Forwarding) for Secure Git Operations

## 1. Description
SSH Agent Forwarding enables secure git operations within Claude-Reactor containers by forwarding the host's SSH agent socket to the container. This eliminates the need to copy private keys into containers while providing seamless git push/pull functionality. Users can leverage their existing SSH key management workflows, including hardware tokens and encrypted keys with passphrases, without compromising security.

## Goal statement
To enable secure git operations within Claude-Reactor containers by forwarding SSH authentication through the host's SSH agent, eliminating private key exposure while maintaining full git workflow functionality.

## Project Analysis & current state

### Technology & architecture
- **Go CLI framework**: Cobra for command-line interface (`cmd/claude-reactor/commands/run.go`)
- **Configuration management**: YAML-based config persistence (`internal/reactor/config/manager.go`)
- **Docker container management**: Docker SDK for Go (`internal/reactor/docker/manager.go`)
- **Mount system**: Existing mount abstraction (`internal/reactor/mount/`)
- **Logging**: Structured logging via `internal/reactor/logging/`
- **Flag-based configuration**: Following `--host-docker-timeout` pattern for persistence

### current state
Currently, Claude-Reactor containers have no access to SSH authentication, preventing git push/pull operations. Users must either:
- Use HTTPS git remotes with stored credentials (less secure)
- Manually configure git credentials inside containers (inconvenient)
- Copy SSH keys into containers (security risk)

The existing configuration system supports string-based parameters with persistence (e.g., `host_docker_timeout`), and the mount system can handle file system mounts.

## context & problem definition

### problem statement
Developers using Claude-Reactor cannot perform git push/pull operations because containers lack SSH key access. This forces users to either compromise security by copying private keys or use less secure HTTPS authentication methods, disrupting standard git workflows.

### success criteria
- Users can perform `git push` and `git pull` operations from within Claude-Reactor containers
- No private keys are copied or exposed within containers
- SSH agent forwarding works across macOS and Linux platforms
- Configuration persists across container restarts for seamless reuse
- Clear error messages guide users through SSH agent setup issues
- Integration takes <5 minutes for users with existing SSH agents

## technical requirements

### functional requirements
- [ ] Add `--ssh-agent` flag for auto-detecting host SSH agent
- [ ] Support `--ssh-agent=path` for explicit SSH agent socket paths
- [ ] Forward SSH agent socket to container with proper permissions
- [ ] Mount essential SSH files (`~/.ssh/config`, `~/.ssh/known_hosts`) read-only
- [ ] Mount `~/.gitconfig` read-only for complete git workflow
- [ ] Persist SSH agent configuration for container reuse scenarios
- [ ] Validate SSH agent connectivity before container startup
- [ ] Provide helpful error messages for common SSH agent misconfigurations
- [ ] Integrate with smart container reuse logic (recreate when flag specified)
- [ ] Comprehensive documentation and operational tooling via Makefile
- [ ] Developer onboarding experience <10 minutes from clone to running

### non-functional requirements
- **Security**: No private keys copied to containers, read-only mounts for SSH files
- **Performance**: SSH agent validation <1 second, no impact on container startup
- **Compatibility**: Support macOS and Linux SSH agent implementations
- **Usability**: Auto-detection works for 90% of standard SSH agent setups
- **Reliability**: Graceful handling of agent disconnection or restart
- **Operations**: All common tasks accessible via single Makefile commands
- **Documentation**: Comprehensive docs/ structure with setup, deployment, troubleshooting guides
- **Developer Experience**: <10 minutes from git clone to running locally

### Technical Constraints
- Docker socket mounting limitations on different platforms
- SSH agent socket path variations across operating systems
- Container user permission requirements for SSH socket access
- Read-only mount restrictions for security compliance

## Data & Database changes

### Data model updates
Configuration struct updates in `pkg/interfaces.go`:

```go
type Config struct {
    // ... existing fields ...
    SSHAgent       bool   `yaml:"ssh_agent,omitempty"`
    SSHAgentSocket string `yaml:"ssh_agent_socket,omitempty"` // "auto" or explicit path
}

type ContainerConfig struct {
    // ... existing fields ...
    SSHAgent       bool   `yaml:"ssh_agent,omitempty"`
    SSHAgentSocket string `yaml:"ssh_agent_socket,omitempty"`
}
```

### Data migration plan
N/A - New fields added to existing configuration structures with omitempty tags for backward compatibility.

## API & Backend changes

### Data access pattern
Configuration accessed via existing `ConfigManager.LoadConfig()` and `ConfigManager.SaveConfig()` methods using YAML serialization.

### server actions
New functions in `internal/reactor/config/manager.go`:
- `DetectSSHAgent() (string, error)` - Auto-detect SSH agent socket
- `ValidateSSHAgent(socketPath string) error` - Test agent connectivity
- `PrepareSSHMounts(sshAgent bool, socketPath string) ([]Mount, error)` - Prepare SSH-related mounts

New functions in `cmd/claude-reactor/commands/run.go`:
- Handle `--ssh-agent` flag parsing and validation
- Integration with existing configuration persistence logic

### Database queries
N/A - File-based YAML configuration storage.

### API Routes
N/A - CLI application with no HTTP API.

## frontend changes

### New components
N/A - CLI application with no frontend.

### Page updates
N/A - CLI application with no frontend.

## Implementation plan

### phase 1 - Core SSH Agent Detection and Validation
- [ ] Add SSH agent detection logic for macOS and Linux
- [ ] Implement agent connectivity validation with `ssh-add -l`
- [ ] Create comprehensive error messages for common failure scenarios
- [ ] Add unit tests for agent detection across different socket configurations
- [ ] Handle platform-specific SSH agent socket paths and naming

### phase 2 - Configuration and CLI Integration  
- [ ] Add `--ssh-agent` flag to run command with auto-detect and explicit path support
- [ ] Extend configuration structures to store SSH agent settings
- [ ] Implement configuration persistence following `host_docker_timeout` pattern
- [ ] Integrate with smart container reuse logic (recreate when flag specified)
- [ ] Add configuration validation and migration support

### phase 3 - Container Mount Integration
- [ ] Implement SSH agent socket mounting with proper permissions
- [ ] Add read-only mounting for `~/.ssh/config` and `~/.ssh/known_hosts`
- [ ] Add read-only mounting for `~/.gitconfig` for complete git workflow
- [ ] Ensure container `claude` user can access SSH agent socket
- [ ] Handle Docker platform differences for socket mounting

### phase 4 - Testing and Documentation
- [ ] Create comprehensive unit tests for all SSH agent functionality
- [ ] Add integration tests with real SSH agents and git operations
- [ ] Write user documentation for SSH agent setup and troubleshooting
- [ ] Create docs/ structure with README, DEVELOPMENT, DEPLOYMENT, OPERATIONS guides
- [ ] Implement comprehensive Makefile with all development, testing, and deployment commands
- [ ] Validate <10 minute developer onboarding experience
- [ ] Document troubleshooting procedures and common issues

## 5. Testing Strategy

### Unit Tests
- `DetectSSHAgent()` function with mocked environment variables and file system
- `ValidateSSHAgent()` with various socket states (working, dead, permission denied)
- Configuration persistence for different SSH agent settings
- CLI flag parsing for both auto-detect and explicit path scenarios
- Error message generation for different failure modes

### Integration Tests
- End-to-end SSH agent forwarding with real agent and container
- Git operations (clone, push, pull) within container using forwarded agent
- Container reuse scenarios with persisted SSH agent configuration
- Cross-platform testing on macOS and Linux environments
- Mount validation for SSH config files and git config

### End-to-End (E2E) Tests
1. **Complete git workflow**: Start SSH agent, add key, run claude-reactor with --ssh-agent, perform git clone/push/pull operations
2. **Agent auto-detection**: Test with standard SSH agent setups on different platforms
3. **Configuration persistence**: Run with --ssh-agent, stop container, run with no flags, verify agent still forwarded
4. **Error scenarios**: Test with no agent, dead agent, empty agent, invalid socket paths

## 6. Security Considerations

### Authentication & Authorization
- SSH agent forwarding maintains host authentication model
- Container cannot access private keys directly, only through agent
- Read-only mounts prevent container from modifying SSH configuration
- Agent socket mounted with minimal necessary permissions

### Data Validation & Sanitization
- Socket path validation to prevent directory traversal attacks
- SSH agent response validation before considering agent "working"
- File existence and permission checks before mounting SSH configuration files
- Environment variable sanitization for SSH_AUTH_SOCK detection

### Potential Vulnerabilities
- **Socket hijacking**: Container could potentially abuse SSH agent access - mitigated by read-only socket mount
- **Configuration exposure**: SSH config may contain sensitive hostnames/aliases - acceptable for development use case
- **Agent flooding**: Container could overwhelm agent with requests - acceptable risk for development workflow
- **Path injection**: Explicit socket paths need validation - mitigated by path sanitization

## 7. Rollout & Deployment

### Feature Flags
No feature flags needed - opt-in functionality via `--ssh-agent` flag.

### Deployment Steps
1. Update configuration structures and add SSH agent detection logic
2. Add CLI flags and configuration persistence
3. Implement container mounting for SSH agent and related files
4. Update documentation and examples
5. Release with comprehensive testing across platforms

### Rollback Plan
- Feature is opt-in via flags, no impact on existing functionality
- Remove `--ssh-agent` flag and related configuration if issues arise
- Existing containers without SSH agent forwarding continue working unchanged
- Configuration backward compatibility maintained with omitempty tags

## 8. Open Questions & Assumptions

### Assumptions
- Users have SSH agents configured for their development workflow
- Standard SSH agent socket locations are consistent enough for auto-detection
- Docker socket mounting security is acceptable for development use case
- Read-only mounts provide sufficient security for SSH configuration files
- Most users will use auto-detection rather than explicit socket paths

### Implementation Notes
- SSH agent socket paths vary significantly across platforms but follow detectable patterns
- Container user permissions for SSH socket access may require careful Docker configuration
- Git workflows benefit significantly from SSH config and known_hosts mounting beyond just agent forwarding
- Error messages should educate users about SSH agent setup rather than attempting automatic fixes
- Configuration persistence enables seamless container reuse while maintaining security