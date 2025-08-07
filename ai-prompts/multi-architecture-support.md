# Feature: Multi-Architecture Support

## 1. Description
Add automatic multi-architecture support to Claude-Reactor, enabling seamless operation across ARM64 (Apple Silicon, modern ARM servers) and x86_64 (Intel/AMD) platforms. The system will automatically detect the host architecture and build the appropriate Docker images without requiring user configuration, maintaining the project's zero-configuration philosophy while expanding platform compatibility.

## Goal statement
To automatically detect and build architecture-appropriate Docker containers for ARM64 and x86_64 platforms, ensuring Claude-Reactor works seamlessly across all major hardware platforms without user intervention.

## Project Analysis & current state

### Technology & architecture
- **Docker Multi-stage Builds**: Current Dockerfile uses multi-stage builds with 5 variants (base, go, full, cloud, k8s)
- **Shell Scripting**: Main logic in `claude-reactor` script handles container management and configuration
- **Makefile Automation**: 25+ targets in Makefile for build and test automation  
- **Configuration Management**: `.claude-reactor` file stores user preferences per project
- **Host Integration**: Mounts host directories and configuration files into containers
- **Key Files**: `Dockerfile`, `claude-reactor` script, `Makefile`, `tests/integration/test-variants.sh`

### current state
**CRITICAL FINDING**: The current Dockerfile hardcodes ARM64-specific URLs, making it incompatible with x86_64 systems. Key issues:

- **Dockerfile line 25**: `kubectl` download hardcoded to `linux/arm64` 
- **Dockerfile line 115**: Go binary hardcoded to `linux-arm64.tar.gz`
- **Dockerfile line 191**: AWS CLI hardcoded to `awscliv2-exe-linux-aarch64.zip`
- **Dockerfile lines 231, 235, 243**: K8s tools hardcoded to `arm64` binaries
- **Makefile line 12**: `DOCKER_PLATFORM ?= linux/arm64` forces ARM64 platform

This means the current implementation will **fail to build on x86_64 systems** and represents a blocking issue for multi-architecture support.

## context & problem definition

### problem statement
**Who**: Developers using Claude-Reactor across mixed architecture environments (ARM64 MacBooks, x86_64 Linux servers, Intel Macs)
**What**: Currently face architecture compatibility issues, slow container performance, and potential build failures when containers don't match host architecture
**Why**: No automatic architecture detection means containers may be built for wrong platform, leading to emulation overhead and compatibility issues

### success criteria
- [ ] 100% automatic architecture detection with zero user configuration required
- [ ] Native performance on both ARM64 and x86_64 platforms (no emulation)
- [ ] Backward compatibility maintained - existing workflows unchanged
- [ ] Build time improvement of 50%+ on ARM64 systems (eliminating x86_64 emulation)
- [ ] Cross-platform team collaboration without architecture-specific issues

## technical requirements

### functional requirements
- [ ] Automatic detection of host architecture (ARM64/x86_64) at container build time
- [ ] Architecture-specific Docker image tags and management
- [ ] Seamless operation across all 5 container variants (base, go, full, cloud, k8s)
- [ ] Architecture information display in `--show-config` output
- [ ] Backward compatibility with existing `.claude-reactor` configurations
- [ ] Architecture-aware container naming to prevent conflicts
- [ ] Integration with existing Makefile build targets
- [ ] Comprehensive documentation and operational tooling via Makefile
- [ ] Developer onboarding experience <10 minutes from clone to running

### non-functional requirements
- **Performance**: Native architecture performance without emulation overhead
- **Compatibility**: 100% backward compatibility with existing workflows
- **Reliability**: Consistent behavior across architecture transitions  
- **Maintainability**: Clean integration with existing codebase patterns
- **Usability**: Zero configuration required from users
- **Operations**: All common tasks accessible via single Makefile commands
- **Documentation**: Comprehensive docs/ structure with setup, deployment, troubleshooting guides
- **Developer Experience**: <10 minutes from git clone to running locally

### Technical Constraints
- **CRITICAL**: Current Dockerfile hardcodes ARM64 binaries - must be fixed before multi-arch support
- Must work with existing Docker installation (no buildx requirement initially)
- No cross-compilation - native builds only  
- Local builds only (no registry requirements)
- Must maintain current container size optimization
- Shell script compatibility across macOS and Linux
- **Architecture mapping**: `uname -m` outputs (arm64, aarch64, x86_64, amd64) to Docker platform names
- **Binary URL patterns**: Each tool has different URL patterns for different architectures

## Data & Database changes

### Data model updates
N/A - No database changes required.

### Data migration plan
N/A - No data migration needed.

## API & Backend changes

### Data access pattern
N/A - This is a containerization feature, no backend API changes.

### server actions
N/A - No server-side changes required.

### Database queries
N/A - No database queries involved.

### API Routes
N/A - No API endpoints affected.

## frontend changes

### New components
N/A - This is a CLI tool with no frontend components.

### Page updates
N/A - No UI changes required.

## Implementation plan

### phase 1 - Architecture Detection & Core Logic
- [ ] Add architecture detection function to `claude-reactor` script using `uname -m`
- [ ] Map architecture names to standardized values (arm64, amd64)
- [ ] Update container naming scheme to include architecture suffix
- [ ] Modify image build logic to use architecture-aware tags
- [ ] Add architecture info to `--show-config` output

### phase 2 - Docker Integration (CRITICAL - Fix Hardcoded Architecture)
- [ ] **URGENT**: Replace all hardcoded ARM64 URLs with architecture-aware logic in Dockerfile
- [ ] Fix kubectl download URL (lines 25-26) to use detected architecture
- [ ] Fix Go binary download (line 115) to use correct architecture suffix
- [ ] Fix AWS CLI download (line 191) to use correct architecture
- [ ] Fix all K8s tools downloads (lines 231, 235, 243) to use detected architecture
- [ ] Update container cleanup logic to handle architecture-specific names
- [ ] Test image builds on both ARM64 and x86_64 platforms

### phase 3 - Makefile & Build System Updates
- [ ] Update Makefile targets to pass architecture information
- [ ] Modify `build-all`, `build-base`, `build-go`, etc. to handle architectures
- [ ] Update test targets to validate architecture-specific containers
- [ ] Add architecture-aware cleanup targets

### phase 4 - Testing & Validation
- [ ] Extend unit tests to cover architecture detection logic
- [ ] Update integration tests to validate both architectures
- [ ] Add cross-platform test scenarios to test suite
- [ ] Validate performance improvements on ARM64 systems

### phase 5 - Documentation & Operations
- [ ] Update CLAUDE.md with multi-architecture capabilities
- [ ] Document architecture detection behavior in README
- [ ] Add troubleshooting guide for architecture-related issues
- [ ] Update WORKFLOW.md with architecture considerations
- [ ] Validate <10 minute developer onboarding experience across platforms

## 5. Testing Strategy
### Unit Tests
- Architecture detection function with various `uname -m` outputs
- Container naming logic with architecture suffixes
- Configuration parsing and architecture storage
- Error handling for unsupported architectures

### Integration Tests
- Container builds on both ARM64 and x86_64 platforms
- All 5 variants building successfully with correct architecture
- Container startup and Claude CLI functionality
- Cross-platform configuration file compatibility

### End-to-End (E2E) Tests
- Complete workflow test on ARM64: detect → build → run → cleanup
- Complete workflow test on x86_64: detect → build → run → cleanup
- Project sharing scenario: ARM64 user creates .claude-reactor, x86_64 user uses same project
- Architecture transition: same user moving from ARM64 to x86_64 machine

## 6. Security Considerations
### Authentication & Authorization
No changes to authentication model - existing Claude CLI authentication methods maintained.

### Data Validation & Sanitization
- Validate architecture detection output against known values (arm64, amd64)
- Sanitize architecture strings used in container names to prevent injection
- Ensure architecture detection cannot be manipulated maliciously

### Potential Vulnerabilities
- Container name injection through architecture detection manipulation
- **Mitigation**: Strict allowlist of valid architecture strings
- Path traversal through malicious architecture values in container mounting
- **Mitigation**: Validate architecture strings against regex pattern

## 7. Rollout & Deployment
### Feature Flags
No feature flag needed - architecture detection will be always-on with graceful fallback to current behavior if detection fails.

### Deployment Steps
1. Update `claude-reactor` script with architecture detection
2. Test on both ARM64 and x86_64 development machines
3. Update Makefile and test suite
4. Update documentation

### Rollback Plan
- Architecture detection failure falls back to current behavior (no architecture suffix)
- Users can force rebuild without architecture detection if needed
- Existing containers and configurations remain functional

## 8. Open Questions & Assumptions

### Open Questions
1. Should we store detected architecture in `.claude-reactor` config file for consistency?
2. How should we handle edge cases like running x86_64 Docker on ARM64 host?
3. Should architecture be user-overrideable via command line flag?
4. **CRITICAL**: How should we handle the immediate compatibility issue with x86_64 systems?
5. Should we implement build-time architecture detection in Dockerfile or runtime detection in script?
6. How do we map different `uname -m` outputs (aarch64 vs arm64, x86_64 vs amd64) to consistent naming?

### Assumptions
- Docker is configured to run native containers (not emulated)
- `uname -m` provides reliable architecture detection on target platforms  
- Users prefer automatic detection over manual configuration
- Current container variants work well on both architectures without modification
- Performance improvement will be significant enough to justify the complexity