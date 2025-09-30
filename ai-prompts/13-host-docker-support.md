# Feature: Host Docker Socket Access Support

## 1. Description

Add optional host Docker socket access to claude-reactor containers, enabling advanced use cases where containers need to interact with the host Docker daemon. This provides host-level Docker API access for container management, building, and orchestration within Claude-reactor workflows.

## Goal Statement

Enable secure, optional host Docker socket access for claude-reactor containers with explicit user consent, clear security warnings, and proper permission handling while maintaining the current secure-by-default behavior for users who don't need host Docker access.

## Project Analysis & Current State

### Technology & Architecture
- **Current State**: Claude-reactor runs in isolated containers without host Docker access
- **Target State**: Optional host Docker socket mounting with proper security controls
- **Security Model**: Explicit opt-in with runtime warnings about host-level privileges
- **Flag Independence**: Separate from `--danger` flag, can be used independently or together

### Current Limitations
- No access to host Docker daemon from within containers launched by `run` command
- Cannot build Docker images or manage host containers from within running containers
- `run` command limited to container-internal operations only
- Missing infrastructure for host-level Docker operations within user containers

### Command-Specific Docker Requirements
- **`run`**: Launches user containers that may need host Docker access ✅ `--host-docker` needed
- **`clean`**: Manages claude-reactor containers using existing Docker API ❌ No host access needed
- **`info`**: Shows system information and tests Docker connectivity ❌ No host access needed
- **`config`**: Configuration management only ❌ No Docker needed
- **`completion`**: Shell completion only ❌ No Docker needed

**Rationale**: Only the `run` command launches containers where users might need to build images, manage containers, or perform Docker operations. Other commands operate on claude-reactor's own infrastructure and don't require host-level Docker privileges.

## Context & Problem Definition

### Problem Statement
**Who**: Advanced users requiring host Docker capabilities, CI/CD workflows, container orchestration
**What**: Need host Docker daemon access from within claude-reactor containers for building images, managing containers, and orchestration tasks
**Why**: Current isolated container model limits advanced Docker workflows that require host daemon interaction

### Current Pain Points
- **Limited Docker Capabilities**: Cannot build images or manage host containers
- **Advanced Workflow Restrictions**: CI/CD and orchestration use cases not supported
- **User Confusion**: Unclear when Docker operations require host access
- **Security Uncertainty**: No clear guidance on security implications

### Success Criteria
- [ ] **Optional Host Access**: Flag-controlled host Docker socket mounting
- [ ] **Security Transparency**: Clear warnings about host-level privileges granted
- [ ] **Independent Configuration**: Works with/without `--danger` flag
- [ ] **Graceful Fallback**: Clear error messages when host Docker unavailable
- [ ] **Persistent Configuration**: Save host Docker preference per project
- [ ] **Comprehensive Documentation**: Security implications and setup requirements
- [ ] **Container Runtime Support**: Proper Dockerfile and runtime configuration

## Technical Requirements

### Functional Requirements

#### CLI Interface
- [ ] Add `--host-docker` flag to `run` command only
- [ ] Add `--host-docker-timeout` flag for configurable Docker operation timeout
- [ ] Flag independence: allow `--danger`, `--host-docker`, both, or neither
- [ ] Configuration persistence: save `host_docker=true` and `host_docker_timeout=` in `.claude-reactor`
- [ ] Runtime warning: display security notice when flag is active
- [ ] Help text: clear documentation of security implications and timeout behavior

#### Docker Detection & Validation
- [ ] Detect Docker socket availability at `/var/run/docker.sock`
- [ ] Validate Docker group membership for current user
- [ ] Check Docker daemon connectivity and API version
- [ ] Configurable timeout for Docker operations (default: 5m, use "0" to disable)
- [ ] Graceful error handling when host Docker unavailable
- [ ] Distinguished error messages: flag not set vs. Docker unavailable

#### Security Controls
- [ ] Explicit user consent required via `--host-docker` flag
- [ ] Runtime security warning displayed when enabled
- [ ] Clear documentation of host-level privileges granted
- [ ] No auto-detection or implicit host Docker access
- [ ] Audit trail: log when host Docker access is enabled

#### Container Infrastructure
- [ ] Dockerfile: Docker group creation with configurable GID
- [ ] Runtime: Proper socket mounting and group permissions
- [ ] User setup: Add `claude` user to docker group
- [ ] GID handling: Support host Docker GID detection and matching

### Non-Functional Requirements

#### Security
- **Principle of Least Privilege**: No host Docker access unless explicitly requested
- **Clear Warnings**: Prominent security notices when host access enabled
- **Audit Trail**: Log all host Docker operations for security review
- **Documentation**: Comprehensive security implications and best practices

#### Usability
- **Clear Error Messages**: Distinguish between "flag not set" and "Docker unavailable"
- **Consistent Interface**: Same flag behavior across all commands
- **Configuration Persistence**: Remember user preference per project
- **Help Integration**: Clear documentation in `--help` output

#### Reliability
- **Graceful Degradation**: Fall back to non-host operations when host Docker unavailable
- **Error Recovery**: Clear guidance on resolving Docker access issues
- **Timeout Handling**: 30-second timeout for all Docker operations

### Technical Constraints
- Must maintain backward compatibility with existing configurations
- Must not enable host Docker access by default
- Must work with existing Docker socket mounting patterns
- Must support both rootful and rootless Docker configurations

## Security Analysis

### Security Implications

#### Host-Level Privileges
- **Docker Socket Access**: Equivalent to root access on host system
- **Container Escape**: Can create privileged containers and mount host filesystem
- **Network Access**: Can access host network and other containers
- **File System Access**: Can mount and access any host directory
- **Process Control**: Can view and interact with host processes

#### Attack Vectors
- **Malicious Image Execution**: Host Docker access allows running any container
- **Data Exfiltration**: Can mount host directories and access sensitive data
- **Privilege Escalation**: Can create privileged containers for host access
- **Network Reconnaissance**: Access to host network and container networks
- **Resource Exhaustion**: Can consume host Docker resources

#### Mitigation Strategies
- **Explicit Consent**: Require `--host-docker` flag, never default
- **Runtime Warnings**: Display security implications when enabled
- **Documentation**: Comprehensive security guidance and best practices
- **Audit Logging**: Log all host Docker operations
- **Timeout Controls**: Configurable timeout for Docker operations (default 5m, disable with "0")

### Security Model

```
User Intent → Flag Required → Warning Displayed → Host Access Granted
    ↓              ↓              ↓                ↓
   User         --host-docker    Security        Docker Socket
  Decision       Flag Set        Warning         Mounted
```

## Implementation Plan

### Phase 1 - CLI Flag and Configuration Support
**Goal**: Add CLI flag support and configuration persistence

#### Flag Implementation
- [ ] Add `--host-docker` flag to `run` command only
- [ ] Add `--host-docker-timeout` flag with duration parsing (default "5m", "0" disables)
- [ ] Update flag parsing logic in run command handler
- [ ] Add flags to configuration struct and persistence
- [ ] Implement independent flag behavior (separate from `--danger`)

#### Configuration Updates
- [ ] Add `host_docker` field to configuration struct
- [ ] Add `host_docker_timeout` field to configuration struct
- [ ] Update `.claude-reactor` file format to include host Docker preferences
- [ ] Implement configuration migration for existing config files
- [ ] Add validation for host Docker configuration values and timeout parsing

#### Example Configuration
```bash
# .claude-reactor
variant=go
danger=true
host_docker=true
host_docker_timeout=10m
account=work
```

### Phase 2 - Docker Detection and Validation
**Goal**: Implement host Docker detection and validation logic

#### Detection Functions
- [ ] `detectDockerSocket()`: Check for `/var/run/docker.sock`
- [ ] `validateDockerAccess()`: Test Docker API connectivity
- [ ] `checkDockerGroupMembership()`: Verify user in docker group
- [ ] `getDockerDaemonInfo()`: Retrieve Docker daemon information

#### Error Handling
- [ ] Distinguish "host Docker not requested" vs "host Docker unavailable"
- [ ] Provide clear guidance for resolving Docker access issues
- [ ] Implement graceful fallback to non-host operations
- [ ] Add troubleshooting information to error messages

#### Detection Logic Flow
```go
func ensureHostDockerAccess(hostDockerRequested bool, timeout time.Duration) error {
    if !hostDockerRequested {
        return initializeIsolatedDocker() // Current behavior
    }

    if !dockerSocketExists() {
        return fmt.Errorf("host Docker requested but socket not available")
    }

    if !validateDockerAccess() {
        return fmt.Errorf("host Docker socket found but access denied")
    }

    displaySecurityWarning()
    return initializeHostDocker(timeout)
}
```

### Phase 3 - Security Warnings and User Experience
**Goal**: Implement security warnings and user guidance

#### Runtime Security Warning
```
⚠️  WARNING: HOST DOCKER ACCESS ENABLED
🔒 This grants claude-reactor container HOST-LEVEL Docker privileges:
   • Can create/manage ANY container on the host
   • Can mount/access ANY host directory
   • Can access host network and other containers
   • Equivalent to ROOT access on the host system
💡 Only enable for trusted workflows requiring Docker management
```

#### Warning Implementation
- [ ] Display warning once per session when `--host-docker` enabled
- [ ] Add flag to suppress warning for automated workflows
- [ ] Include warning in help text and documentation
- [ ] Log warning display in audit trail

#### User Experience Improvements
- [ ] Clear help text explaining host Docker implications
- [ ] Examples in help output showing safe vs. host Docker usage
- [ ] Troubleshooting section for common Docker access issues
- [ ] Migration guide for existing users

### Phase 4 - Container Infrastructure Updates
**Goal**: Update Dockerfile and runtime configuration for host Docker support

#### Dockerfile Updates
```dockerfile
# Support configurable Docker GID for host compatibility
ARG DOCKER_GID=999
RUN groupadd -g ${DOCKER_GID} docker || groupmod -g ${DOCKER_GID} docker
RUN usermod -aG docker claude

# Install Docker CLI for host Docker operations
RUN curl -fsSL https://download.docker.com/linux/static/stable/x86_64/docker-20.10.17.tgz | \
    tar xzf - --strip 1 -C /usr/local/bin docker/docker

# Validate Docker client installation
RUN docker --version
```

#### Runtime Configuration
```bash
# Host Docker socket mounting pattern
docker run \
  -v /var/run/docker.sock:/var/run/docker.sock \
  --group-add docker \
  claude-reactor run --host-docker

# GID detection and handling
DOCKER_GID=$(stat -f %g /var/run/docker.sock 2>/dev/null || echo 999)
docker build --build-arg DOCKER_GID=$DOCKER_GID -t claude-reactor .
```

#### Documentation Updates
- [ ] Container runtime requirements for host Docker
- [ ] Docker GID detection and configuration
- [ ] Security considerations for production deployments
- [ ] Troubleshooting guide for permission issues

### Phase 5 - Testing and Validation
**Goal**: Comprehensive testing of host Docker functionality

#### Unit Tests
- [ ] Flag parsing and configuration persistence
- [ ] Docker detection and validation logic
- [ ] Security warning display and suppression
- [ ] Error handling for various Docker unavailability scenarios

#### Integration Tests
- [ ] Host Docker socket mounting and access
- [ ] Docker group membership validation
- [ ] Container creation and management with host Docker
- [ ] Security warning display in real workflows

#### Security Tests
- [ ] Verify no host Docker access without explicit flag
- [ ] Validate security warnings are displayed
- [ ] Test graceful fallback when host Docker unavailable
- [ ] Verify audit logging of host Docker operations

## Usage Examples

### Basic Usage (No Host Docker)
```bash
# Standard isolated container operation
claude-reactor run --image go

# Danger mode without host Docker
claude-reactor run --danger
```

### Host Docker Access
```bash
# Enable host Docker access (with security warning)
claude-reactor run --host-docker

# Configure timeout for long builds (default: 5m)
claude-reactor run --host-docker --host-docker-timeout 15m

# Disable timeout for complex builds (use with caution)
claude-reactor run --host-docker --host-docker-timeout 0

# Combined with danger mode
claude-reactor run --danger --host-docker --host-docker-timeout 10m

# Persistent configuration
echo "host_docker=true" >> .claude-reactor
echo "host_docker_timeout=15m" >> .claude-reactor
claude-reactor run  # Uses saved preferences
```

### Container Runtime Setup
```bash
# Detect host Docker GID
DOCKER_GID=$(stat -f %g /var/run/docker.sock 2>/dev/null || echo 999)

# Build container with correct Docker group
docker build --build-arg DOCKER_GID=$DOCKER_GID -t claude-reactor .

# Run with host Docker socket mounted
docker run \
  -v /var/run/docker.sock:/var/run/docker.sock \
  --group-add docker \
  claude-reactor run --host-docker --host-docker-timeout 10m
```

## Timeout Configuration Best Practices

### Recommended Timeout Values

#### Default Settings
- **Default timeout**: `5m` - Suitable for most Docker operations (pulls, starts, simple builds)
- **Conservative approach**: Start with default and increase as needed
- **Disable only when necessary**: Use `0` for complex builds requiring unlimited time

#### Operation-Specific Guidelines
```bash
# Quick operations (pulls, starts) - Default is adequate
claude-reactor run --host-docker  # Uses 5m default

# Simple builds (< 10 minutes) - Moderate timeout
claude-reactor run --host-docker --host-docker-timeout 10m

# Complex builds (10-30 minutes) - Extended timeout
claude-reactor run --host-docker --host-docker-timeout 30m

# Large multi-stage builds or slow networks - Disable timeout
claude-reactor run --host-docker --host-docker-timeout 0  # Use with caution
```

#### Configuration Strategy
```bash
# Project-specific timeout in .claude-reactor
echo "host_docker_timeout=15m" >> .claude-reactor  # For complex Go builds
echo "host_docker_timeout=30m" >> .claude-reactor  # For large Docker builds
echo "host_docker_timeout=0" >> .claude-reactor    # For CI/CD pipelines
```

### Timeout Behavior

#### When Timeout is Reached
- **Operation cancelled**: Docker operations are interrupted gracefully
- **Clear error message**: User informed about timeout and can retry with longer duration
- **Container cleanup**: Any partially created containers are removed
- **Retry guidance**: Error message suggests appropriate timeout value

#### Disable Timeout Considerations
```bash
# ⚠️  CAUTION: Disabling timeout removes operation safety net
claude-reactor run --host-docker --host-docker-timeout 0

# Scenarios where disabling timeout is appropriate:
# • Large multi-gigabyte base images in slow networks
# • Complex CI/CD pipelines with many build stages
# • Development environments with known long-running operations
# • Automated scripts where hanging is preferable to premature cancellation
```

#### Timeout Format
- **Go duration format**: `1m30s`, `2h45m`, `10s`, `1h`
- **Common values**: `30s`, `1m`, `5m`, `15m`, `30m`, `1h`, `2h`
- **Disable timeout**: `0` or `0s`
- **Invalid formats**: Rejected with clear error message

## Error Handling and User Guidance

### Error Categories

#### 1. Host Docker Not Requested (Current Behavior)
```
Error: docker not available: failed to connect to Docker daemon
💡 For basic operations, this is expected. Use --host-docker if you need Docker access.
```

#### 2. Host Docker Requested But Socket Missing
```
Error: host Docker requested but socket not available
💡 Mount Docker socket: -v /var/run/docker.sock:/var/run/docker.sock
💡 Add docker group: --group-add docker
💡 See documentation: claude-reactor help docker-setup
```

#### 3. Host Docker Socket Present But Access Denied
```
Error: host Docker socket found but access denied
💡 Check docker group membership: groups $USER
💡 Verify socket permissions: ls -la /var/run/docker.sock
💡 See troubleshooting: claude-reactor help docker-troubleshooting
```

#### 4. Docker Operation Timeout
```
Error: Docker operation timed out after 5m0s
💡 For complex builds, increase timeout: --host-docker-timeout 15m
💡 For unlimited time, disable timeout: --host-docker-timeout 0
💡 Save preference: echo "host_docker_timeout=15m" >> .claude-reactor
```

#### 5. Invalid Timeout Format
```
Error: invalid timeout format "5min": time: unknown unit "min" in duration "5min"
💡 Use Go duration format: 5m, 1h30m, 30s
💡 Valid examples: 30s, 5m, 1h, 2h30m
💡 Disable timeout: 0
```

## Documentation Requirements

### Security Documentation
- [ ] **Host Docker Security Implications**: Comprehensive guide to security risks
- [ ] **When to Use Host Docker**: Appropriate use cases and alternatives
- [ ] **Security Best Practices**: Guidelines for safe host Docker usage
- [ ] **Threat Model**: Analysis of attack vectors and mitigations

### Setup Documentation
- [ ] **Container Runtime Setup**: Complete guide to Docker socket mounting
- [ ] **GID Configuration**: Docker group setup and troubleshooting
- [ ] **Platform-Specific Guides**: Linux, macOS, Windows Docker setup
- [ ] **Production Deployment**: Security considerations for production use

### Troubleshooting Documentation
- [ ] **Common Issues**: Docker socket permissions, group membership
- [ ] **Error Resolution**: Step-by-step guides for each error type
- [ ] **Diagnostic Commands**: Tools for diagnosing Docker access issues
- [ ] **FAQ**: Frequently asked questions about host Docker access

## Testing Strategy

### Unit Tests
- Flag parsing and validation for `--host-docker` in `run` command only
- Configuration persistence and migration for host Docker preference
- Docker detection and validation functions for host socket access
- Security warning display logic for `run` command
- Error message generation and categorization for host Docker scenarios

### Integration Tests
- Host Docker socket detection and access
- Docker group membership validation
- Container operations with host Docker access
- Security warning display in real workflows
- Configuration persistence across sessions

### Security Tests
- Verify no host Docker access without explicit consent
- Validate security warnings are always displayed
- Test audit logging of host Docker operations
- Verify graceful fallback when host Docker unavailable
- Test attack scenarios and mitigation effectiveness

## Security Considerations

### Threat Modeling

#### High-Risk Scenarios
- **Malicious Container Execution**: Host Docker allows running any container image
- **Container Escape**: Privileged containers can access host filesystem
- **Data Exfiltration**: Host directory mounting enables data access
- **Network Reconnaissance**: Access to host and container networks
- **Resource Exhaustion**: Unlimited host Docker resource consumption

#### Mitigation Controls
- **Explicit Consent**: `--host-docker` flag required, never default
- **Security Warnings**: Prominent warnings about host-level access
- **Audit Logging**: Log all host Docker operations for review
- **Timeout Controls**: Configurable timeout on Docker operations (default 5m, disable with "0")
- **Documentation**: Comprehensive security guidance

### Security Model Validation

#### Principle of Least Privilege
- No host Docker access unless explicitly requested
- Clear separation between isolated and host Docker modes
- Minimal additional privileges granted

#### Defense in Depth
- Multiple validation layers for host Docker access
- Clear error messages for troubleshooting
- Comprehensive documentation of security implications
- Audit logging for security monitoring

## Open Questions & Assumptions

### Assumptions
- Users understand Docker security implications when using `--host-docker`
- Host Docker socket mounting follows standard Docker patterns
- Docker group membership can be configured correctly
- Security warnings are sufficient for informed consent

### Implementation Notes
- This feature significantly expands claude-reactor's capabilities
- Security warning must be prominent and cannot be easily dismissed
- Error messages must clearly distinguish between different failure modes
- Documentation must emphasize security implications
- Testing must cover both functionality and security aspects

### Future Considerations
- Docker-over-TCP support for remote Docker daemons
- Rootless Docker support and configuration
- Container runtime security profiles (AppArmor, SELinux)
- Integration with container security scanning tools
- Advanced audit logging and monitoring capabilities