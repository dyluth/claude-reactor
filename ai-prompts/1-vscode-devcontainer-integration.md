# Feature: VS Code Dev Container Integration

## 1. Description
Create comprehensive `.devcontainer/` configurations that automatically detect project type, select appropriate claude-reactor variant, install relevant VS Code extensions, and mount necessary directories for a seamless one-click "Reopen in Container" workflow. This eliminates the "works on my machine" problem entirely and provides the IDE integration that modern developers expect.

## Goal statement
To enable developers to instantly create isolated, fully-configured development environments with a single click in VS Code, automatically selecting the optimal claude-reactor variant based on project type and including all necessary tools and extensions.

## Project Analysis & current state

### Technology & architecture
- **Current claude-reactor**: Go-based CLI with 5 container variants (base, go, full, cloud, k8s)
- **Docker Integration**: Multi-stage Dockerfile with architecture-aware builds
- **Project Detection**: Auto-detection logic for Go, Rust, Node.js, Python, Java, K8s, Cloud projects
- **VS Code Dev Containers**: JSON configuration format with customization options
- **Mount Management**: Existing mount logic for Claude config, project files, and additional directories
- **Account Isolation**: Support for multiple Claude accounts with isolated configurations
- **Key Files**: `Dockerfile`, `claude-reactor` CLI, project detection in `internal/config/`

### current state
**Current Workflow**:
1. Developer manually runs `./claude-reactor run` or `make run`
2. Container is built/started with auto-detected variant
3. Developer must manually attach to container via docker exec or separate terminal
4. No IDE integration - developers lose VS Code extensions, IntelliSense, debugging, etc.
5. Context switching between host IDE and container terminal creates friction

**Missing Integration**:
- No `.devcontainer/` configurations exist
- No automatic VS Code extension installation in containers
- No seamless IDE experience with container-based development
- Manual container attachment process interrupts developer flow

## context & problem definition

### problem statement
**Who**: Developers using claude-reactor who want modern IDE integration and seamless container-based development workflows
**What**: Currently face context switching friction between VS Code on host and claude-reactor containers, losing IDE features like IntelliSense, debugging, extensions, and integrated terminals
**Why**: Manual container attachment and lack of IDE integration creates friction that discourages container-based development adoption, especially for complex projects

**Current Pain Points**:
- **Context Switching**: Developers lose IDE features when working in containers
- **Manual Setup**: Must remember container attachment commands and workflow
- **Extension Loss**: VS Code extensions don't work with container-based tools
- **Debugging Difficulty**: No integrated debugging experience with containerized environments
- **Team Inconsistency**: Different developers use different attachment methods

### success criteria
- [ ] **One-Click Setup**: Complete development environment ready within 30 seconds of "Reopen in Container"
- [ ] **Automatic Variant Detection**: Correctly selects variant (go, rust, cloud, k8s) based on project files
- [ ] **Extension Auto-Install**: Language-specific extensions automatically installed and configured
- [ ] **Full IDE Integration**: IntelliSense, debugging, terminal, Git integration all work seamlessly
- [ ] **Account Isolation**: Respects claude-reactor account configuration without manual setup
- [ ] **Performance**: Container startup and VS Code connection under 30 seconds
- [ ] **Team Consistency**: Same environment for all team members regardless of host OS

## technical requirements

### functional requirements
- [ ] **Dynamic devcontainer.json Generation**: Template system that generates configurations based on detected project type
- [ ] **Variant-Specific Configurations**: Separate devcontainer configs for each claude-reactor variant (base, go, full, cloud, k8s)
- [ ] **Extension Auto-Selection**: Automatic installation of relevant VS Code extensions based on project type
- [ ] **Mount Point Integration**: Seamless integration with claude-reactor's mount management (Claude config, project files, additional mounts)
- [ ] **Account Configuration**: Automatic detection and mounting of correct Claude account configuration
- [ ] **Environment Variable Passthrough**: Support for environment variables and secrets in dev containers
- [ ] **Port Forwarding**: Automatic forwarding of common development ports (3000, 8080, etc.)
- [ ] **Git Integration**: Proper Git configuration and SSH key forwarding
- [ ] **Shell Customization**: Pre-configured shell with aliases and development tools
- [ ] Comprehensive documentation and operational tooling via Makefile
- [ ] Developer onboarding experience <10 minutes from clone to running

### non-functional requirements
- **Performance**: Container ready and VS Code connected within 30 seconds
- **Reliability**: 99% successful container startup rate across different project types
- **Compatibility**: Works with VS Code on macOS, Linux, and Windows (with WSL2)
- **Resource Usage**: Container overhead <500MB RAM, reasonable for developer laptops
- **Maintainability**: Template-driven system that's easy to update and extend
- **Extensibility**: Plugin architecture for custom project types and extension sets
- **Operations**: All common tasks accessible via single Makefile commands
- **Documentation**: Comprehensive docs/ structure with setup, deployment, troubleshooting guides
- **Developer Experience**: <10 minutes from git clone to running locally

### Technical Constraints
- Must work with existing claude-reactor architecture and Docker containers
- VS Code Dev Containers extension required on developer machines
- Docker Desktop or compatible Docker environment required
- Must maintain claude-reactor's zero-configuration philosophy
- Cannot break existing `./claude-reactor run` workflows
- Must support all 5 claude-reactor variants without modification
- Template system must be maintainable without VS Code expertise

## Data & Database changes

### Data model updates
N/A - No database changes required. Configuration stored in `.devcontainer/` directory.

### Data migration plan
N/A - New feature, no existing data to migrate.

## API & Backend changes

### Data access pattern
N/A - This is a client-side development environment feature.

### server actions
N/A - No server-side components involved.

### Database queries
N/A - No database interaction required.

### API Routes
N/A - No API endpoints needed.

## frontend changes

### New components
N/A - VS Code extension ecosystem handles UI components.

### Page updates
N/A - No web UI changes required.

## Implementation plan

### phase 1 - Project Detection Integration
**Goal**: Integrate with claude-reactor's existing project detection for automatic variant selection
- [ ] Study existing auto-detection logic in `internal/config/manager.go`
- [ ] Create mapping from detected project types to VS Code extensions
- [ ] Implement detection result caching for devcontainer generation
- [ ] Add devcontainer-specific project detection (look for existing `.devcontainer/`)
- [ ] Test detection accuracy across various project types

### phase 2 - Devcontainer Template System
**Goal**: Create flexible template system for generating variant-specific devcontainer configurations
- [ ] Design template structure for each claude-reactor variant (base, go, full, cloud, k8s)
- [ ] Implement template engine for dynamic devcontainer.json generation
- [ ] Create variant-specific extension mappings (Go → Go extension, Rust → rust-analyzer, etc.)
- [ ] Implement mount point translation from claude-reactor to devcontainer format
- [ ] Add environment variable and secrets handling
- [ ] Create port forwarding configuration for common development servers

### phase 3 - CLI Integration
**Goal**: Integrate devcontainer generation with claude-reactor CLI workflow
- [ ] Add `claude-reactor devcontainer generate` command
- [ ] Implement `--devcontainer` flag for run command to auto-generate configs
- [ ] Add devcontainer validation and troubleshooting commands
- [ ] Integrate with existing configuration management (accounts, variants, etc.)
- [ ] Add VS Code workspace settings generation for optimal container experience
- [ ] Implement automatic devcontainer updates when claude-reactor version changes

### phase 4 - Extension and Tool Configuration
**Goal**: Optimize VS Code experience with pre-configured extensions and development tools
- [ ] Create extension recommendation system based on project analysis
- [ ] Implement automatic language server configuration (gopls, rust-analyzer, etc.)
- [ ] Add debugging configuration for each variant and language
- [ ] Configure integrated terminal with development aliases and tools
- [ ] Set up Git integration with proper user configuration
- [ ] Add container-specific VS Code settings for optimal development experience

### phase 5 - Testing and Quality Assurance
**Goal**: Comprehensive testing across project types, variants, and platforms
- [ ] Create test projects for each supported project type (Go, Rust, Node.js, Python, Java, K8s)
- [ ] Test devcontainer generation and startup for each claude-reactor variant
- [ ] Validate extension installation and functionality in containers
- [ ] Test cross-platform compatibility (macOS, Linux, Windows WSL2)
- [ ] Performance testing for container startup and VS Code connection times
- [ ] Create automated testing workflow for devcontainer validation

### phase 6 - Documentation and Developer Experience
**Goal**: Comprehensive documentation and smooth onboarding experience
- [ ] Create comprehensive VS Code + claude-reactor setup guide
- [ ] Document devcontainer customization options and advanced configurations
- [ ] Create troubleshooting guide for common VS Code Dev Container issues
- [ ] Add video tutorials for setup and common workflows
- [ ] Document team collaboration workflows with shared devcontainers
- [ ] Create migration guide for teams transitioning from local development
- [ ] Validate <10 minute developer onboarding experience

## 5. Testing Strategy

### Unit Tests
- **Project Detection**: Test detection logic with various project structures and edge cases
- **Template Generation**: Test devcontainer.json generation with different variants and options
- **Extension Mapping**: Validate correct extensions are selected for each project type
- **Mount Configuration**: Test translation of claude-reactor mounts to devcontainer format
- **Configuration Validation**: Test generated configurations against VS Code devcontainer schema

### Integration Tests
- **End-to-End Generation**: Test complete workflow from project detection to devcontainer creation
- **VS Code Integration**: Test actual container startup and VS Code connection (where possible)
- **Claude-reactor Integration**: Test devcontainer workflow with existing claude-reactor commands
- **Account Integration**: Test proper mounting and configuration of Claude account isolation
- **Cross-Platform**: Test on macOS and Linux development environments

### End-to-End (E2E) Tests
- **Complete Developer Workflow**: Fresh project → detection → devcontainer generation → VS Code reopen → development
- **Multi-Variant Testing**: Test each claude-reactor variant (base, go, full, cloud, k8s) with appropriate projects
- **Team Collaboration**: Test shared devcontainer configurations across different developer machines
- **Performance Validation**: Measure and validate startup times and resource usage
- **Extension Functionality**: Validate that installed extensions work correctly in container environment

## 6. Security Considerations

### Authentication & Authorization
- Maintain claude-reactor's existing authentication patterns without changes
- Ensure Claude account isolation is preserved in devcontainer environment
- Properly handle SSH key forwarding for Git operations without exposing host keys

### Data Validation & Sanitization
- **Template Input Validation**: Sanitize all inputs to template generation to prevent injection attacks
- **Extension Validation**: Validate VS Code extension IDs against known safe extensions
- **Mount Path Validation**: Ensure mount paths cannot escape intended directories
- **Environment Variable Filtering**: Filter sensitive environment variables from devcontainer configs

### Potential Vulnerabilities
- **Container Escape**: Enhanced VS Code integration increases container attack surface - mitigation: maintain container security boundaries
- **Extension Malware**: Automatically installed extensions could be compromised - mitigation: use only well-known, verified extensions
- **Configuration Injection**: Malicious project files could inject harmful devcontainer configs - mitigation: strict template validation
- **Secrets Exposure**: Development environment might expose more secrets - mitigation: careful secrets management and documentation

## 7. Rollout & Deployment

### Feature Flags
No feature flags needed - this is an additive feature that doesn't change existing workflows.

### Deployment Steps
1. **CLI Integration**: Add devcontainer commands to claude-reactor CLI
2. **Template Creation**: Create and test devcontainer templates for all variants
3. **Documentation**: Complete setup guides and troubleshooting documentation  
4. **Team Testing**: Beta test with development teams for feedback
5. **Public Release**: Announce VS Code integration capability

### Rollback Plan
- **No Impact Rollback**: Feature is additive - simply don't use devcontainer commands
- **Template Removal**: Remove `.devcontainer/` directory to disable VS Code integration
- **CLI Rollback**: Remove devcontainer-related commands from CLI if needed
- **Full Backwards Compatibility**: All existing claude-reactor workflows remain unchanged

## 8. Open Questions & Assumptions

### Open Questions
1. Should devcontainer generation be automatic (always create .devcontainer/) or opt-in (flag/command)?
2. How do we handle teams with mixed preferences (some want VS Code integration, others prefer terminal)?
3. Should we support multiple devcontainer configurations per project (e.g., different variants)?
4. How do we handle custom extensions and settings that teams want to add?
5. What's the best approach for handling secrets and API keys in devcontainers?
6. Should we integrate with VS Code Profiles for different development contexts?

### Assumptions
- VS Code Dev Containers extension is acceptable requirement for this feature
- Developers are willing to adopt container-based development for improved consistency
- 30-second startup time is acceptable for the benefits provided
- Teams will prefer shared devcontainer configurations over individual setups
- Claude-reactor's existing variant system maps well to VS Code development needs
- Docker Desktop performance is sufficient for daily development workflows