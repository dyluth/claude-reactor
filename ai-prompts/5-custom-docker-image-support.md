# Feature: Custom Docker Image Support

## 1. Description
Enable users to specify custom Docker images as base environments for their claude-reactor containers, moving beyond the predefined variants (base, go, full, cloud, k8s) to support personalized development environments, organizational standards, and specialized toolchains.

## Goal statement
To allow users to run claude-reactor with any compatible Docker image while maintaining security, functionality, and user experience consistency.

## Project Analysis & current state

### Technology & architecture
- **Go-based CLI**: Cobra framework with structured command handling (`cmd/` directory)
- **Docker Integration**: Docker SDK for Go in `internal/docker/` package
- **Container Management**: `internal/docker/container.go` handles container lifecycle
- **Image Management**: `internal/docker/build.go` manages multi-stage builds from our Dockerfile
- **Configuration System**: `internal/config/` manages variant selection and container settings
- **Architecture Detection**: `internal/architecture/` handles multi-arch image selection

### current state
Currently, users select from 5 predefined container variants:
1. **base**: Node.js, Python, basic development tools (~500MB)
2. **go**: Base + Go toolchain (~800MB)
3. **full**: Go + Rust, Java, database clients (~1.2GB)
4. **cloud**: Full + AWS/GCP/Azure CLIs (~1.5GB)
5. **k8s**: Full + Enhanced Kubernetes tools (~1.4GB)

All images are built from our multi-stage Dockerfile with the `claude-reactor-{variant}-{arch}` naming convention.

## context & problem definition

### problem statement
**Who**: Developers, DevOps engineers, and teams using claude-reactor
**What**: Are constrained by predefined container variants that may not match their specific development environments, organizational standards, or specialized tool requirements
**Why**: Leading to reduced productivity, inability to use proprietary tools, conflicts with company Docker image policies, and friction in adoption

### success criteria
- [ ] Users can successfully run claude-reactor with 95% of publicly available base images (ubuntu, alpine, debian, etc.)
- [ ] Image validation prevents 100% of incompatible images from starting (those without basic shell/filesystem requirements)
- [ ] Custom image startup time remains within 10% of standard variant startup time
- [ ] Zero security regressions compared to standard variants
- [ ] Documentation enables users to create compatible custom images in <30 minutes

## technical requirements

### functional requirements
- [ ] Add `--image` flag to `run` command accepting Docker image references
- [ ] Support both local images and registry images (with pull functionality)
- [ ] Validate custom images for claude-reactor compatibility before container creation
- [ ] Maintain all existing mounting functionality (project files, claude config, additional mounts)
- [ ] Preserve authentication and account isolation features
- [ ] Support both tagged images (`myorg/devenv:latest`) and digest references
- [ ] Provide clear error messages for incompatible images
- [ ] Allow fallback to standard variants if custom image fails
- [ ] Comprehensive documentation and operational tooling via Makefile
- [ ] Developer onboarding experience <10 minutes from clone to running

### non-functional requirements
- **Performance**: Custom image container startup < 15 seconds (vs ~10 seconds for standard variants)
- **Security**: No privilege escalation beyond standard variant security model
- **Compatibility**: Support for major base images (Ubuntu 20.04+, Alpine 3.15+, Debian 11+)
- **Reliability**: 99.9% success rate for validated compatible images
- **Usability**: Intuitive error messages and troubleshooting guidance
- **Operations**: All common tasks accessible via single Makefile commands
- **Documentation**: Comprehensive docs/ structure with setup, deployment, troubleshooting guides
- **Developer Experience**: <10 minutes from git clone to running locally

### Technical Constraints
- Must work with existing Docker daemon requirements
- Cannot require root privileges beyond standard Docker access
- Must support both ARM64 and AMD64 architectures
- Limited to images with basic POSIX shell and filesystem structure
- Must maintain backward compatibility with existing CLI interface
- Memory overhead for validation <50MB
- Cannot modify user's custom images (read-only approach)

## Data & Database changes

### Data model updates
N/A - No database changes required. Configuration will be handled in-memory and through existing config file system.

### Data migration plan
N/A - No data migration needed.

## API & Backend changes

### Data access pattern
Extend existing configuration management in `internal/config/manager.go` to handle custom image specifications alongside variant selection.

### server actions
1. **Image Validation Service**:
   - Input: Docker image reference (string)
   - Output: Validation result (compatible/incompatible + reasons)
   - Logic: Pull image, inspect filesystem, check for required tools

2. **Custom Image Container Creator**:
   - Input: Image reference, project path, configuration
   - Output: Container instance or error
   - Logic: Create container with custom image instead of built variant

3. **Image Information Inspector**:
   - Input: Docker image reference
   - Output: Image metadata (size, arch, base OS, available tools)
   - Logic: Inspect image layers and extract useful information

### Database queries
N/A - No database queries required.

### API Routes
N/A - CLI-only feature, no web API needed.

## frontend changes

### New components
N/A - This is a CLI feature with no web frontend.

### Page updates
N/A - CLI interface only.

## Implementation plan

### phase 1 - CLI Interface & Basic Integration
- [ ] Add `--image` flag to `run` command in `cmd/run.go`
- [ ] Extend configuration system to handle custom image references
- [ ] Update container creation logic to use custom images
- [ ] Basic error handling for invalid image references
- [ ] Unit tests for CLI argument parsing and configuration handling

### phase 2 - Image Validation & Compatibility
- [ ] Design and implement image validation service in `internal/docker/validation.go`
- [ ] Create compatibility checker for basic requirements (shell, filesystem, permissions)
- [ ] Add image inspection utilities for gathering metadata
- [ ] Implement validation caching to avoid repeated checks
- [ ] Comprehensive unit tests for validation logic

### phase 3 - Enhanced Container Integration
- [ ] Ensure proper mounting of project files, claude config, and additional volumes
- [ ] Validate authentication and account isolation with custom images
- [ ] Add image pull functionality with progress reporting
- [ ] Implement fallback mechanisms for failed custom images
- [ ] Integration tests with various real-world base images

### phase 4 - User Experience & Documentation
- [ ] Create comprehensive documentation for custom image usage
- [ ] Add example Dockerfiles for common use cases
- [ ] Implement helpful error messages and troubleshooting guidance
- [ ] Add `--validate-image` command for testing compatibility without running
- [ ] Create docs/ structure with README, DEVELOPMENT, DEPLOYMENT, OPERATIONS guides
- [ ] Implement comprehensive Makefile with all development, testing, and deployment commands
- [ ] Validate <10 minute developer onboarding experience
- [ ] Document troubleshooting procedures and common issues

## 5. Testing Strategy
### Unit Tests
- **CLI Argument Parsing**: Test `--image` flag handling with various input formats
- **Configuration Management**: Test custom image config storage and retrieval
- **Image Validation**: Test compatibility checking with mock Docker images
- **Container Creation**: Test container setup with custom vs standard images
- **Error Handling**: Test all error scenarios with invalid images/references

### Integration Tests
- **Docker Integration**: Test actual container creation with real custom images
- **Mounting System**: Verify all mount points work correctly with custom images
- **Authentication Flow**: Test account isolation and auth config with custom images
- **Multi-Architecture**: Test ARM64 and AMD64 custom image support
- **Image Registry**: Test pulling from public and private registries

### End-to-End (E2E) Tests
- **Standard Workflow**: User runs `./claude-reactor run --image ubuntu:22.04`
- **Validation Failure**: User tries incompatible image, receives clear error message
- **Custom Development**: User creates and uses custom image with specialized tools
- **Fallback Scenario**: Custom image fails, user falls back to standard variant
- **Multi-Project**: Different custom images for different projects in same session

## 6. Security Considerations
### Authentication & Authorization
- Maintain existing account-based isolation with custom images
- Ensure custom images cannot access other users' claude configurations
- Validate that image pull operations respect Docker daemon security settings
- No additional authentication required - leverage existing Docker access controls

### Data Validation & Sanitization
- Sanitize Docker image references to prevent injection attacks
- Validate image names against Docker registry naming conventions
- Prevent path traversal attacks in image reference parsing
- Validate image digests and tags for proper format

### Potential Vulnerabilities
- **Malicious Images**: Users could specify images with malware or security vulnerabilities
  - *Mitigation*: Clear documentation about image source responsibility, optional image scanning integration
- **Privilege Escalation**: Custom images might attempt to gain elevated privileges
  - *Mitigation*: Run containers with same security constraints as standard variants
- **Resource Exhaustion**: Large custom images could consume excessive disk/memory
  - *Mitigation*: Implement size warnings and resource monitoring
- **Registry Authentication**: Private registry credentials could be exposed
  - *Mitigation*: Use Docker daemon's credential management, never store in claude-reactor

## 7. Rollout & Deployment
### Feature Flags
Feature flag name: `CUSTOM_IMAGE_SUPPORT`
Default state: `enabled` (feature will be stable on release)
Environment variable: `CLAUDE_REACTOR_ENABLE_CUSTOM_IMAGES` (default: `true`)

### Deployment Steps
1. Update Go binary with new CLI flag and validation logic
2. Update documentation with custom image usage examples
3. No additional infrastructure changes required
4. Feature is backward compatible - existing users unaffected

### Rollback Plan
1. Set environment variable `CLAUDE_REACTOR_ENABLE_CUSTOM_IMAGES=false`
2. Remove `--image` flag from CLI help output
3. Existing containers continue running; new ones use standard variants only
4. No data loss - configuration gracefully degrades to variant selection

## 8. Open Questions & Assumptions

### Open Questions
1. **Image Size Limits**: Should we impose maximum size limits for custom images? What's reasonable?
2. **Registry Authentication**: How should we handle private registry authentication? Use Docker daemon creds?
3. **Image Caching Strategy**: Should we implement local caching for frequently used custom images?
4. **Validation Depth**: How deep should compatibility validation go? Just basic tools or comprehensive testing?
5. **Multi-architecture Handling**: Should we auto-select architecture-appropriate images or require explicit specification?
6. **Update Notifications**: Should we notify users when their custom images have updates available?
7. **Preset Custom Images**: Should we provide a curated list of "recommended" custom images?
8. **Integration with DevContainers**: How should this interact with existing devcontainer functionality?

### Assumptions
- Users have appropriate Docker permissions for pulling and running custom images
- Custom images will generally be based on common Linux distributions
- Users understand Docker security implications of running arbitrary images
- Majority of use cases will be extending standard base images (ubuntu, alpine) rather than specialized proprietary images
- Image validation can be performed quickly enough to not significantly impact startup time
- Docker registry connectivity and authentication are handled by user's Docker daemon configuration