# Feature: Hot Reload & File Watching

## 1. Description
Implement intelligent file watching and hot reload capabilities that automatically detect code changes, rebuild applications, restart services, and refresh development environments within claude-reactor containers. This eliminates manual restart cycles and provides instant feedback loops for rapid development, supporting multiple languages and frameworks with optimized rebuild strategies.

## Goal statement
To provide developers with instant feedback on code changes through intelligent file watching, automatic rebuilding, and service restarting that works seamlessly across all supported languages and frameworks within claude-reactor containers.

## Project Analysis & current state

### Technology & architecture
- **Current claude-reactor**: Go-based CLI with 5 container variants supporting multiple languages and frameworks
- **Container Integration**: Docker SDK-based container management with mount point system for live file sharing
- **Language Support**: Go, Rust, Node.js, Python, Java across different container variants
- **Build Systems**: Existing support for various build tools (go build, cargo, npm, pip, maven)
- **File System**: Host directory mounting enables file watching from both host and container perspectives
- **Process Management**: Container lifecycle management through Docker SDK
- **Key Files**: `internal/docker/manager.go`, container variants in Dockerfile, mount management system

### current state
**Current Development Workflow:**
1. Developer makes code changes in host editor/IDE
2. Changes are visible in container through mounted volumes
3. Developer must manually rebuild/restart services inside container
4. Manual testing and validation of changes
5. Repeat cycle for each change iteration

**Missing Hot Reload Integration:**
- No automatic detection of file changes
- No intelligent rebuild triggering based on file types and project structure
- No automatic service restart capabilities
- No development server integration (webpack-dev-server, cargo watch, etc.)
- Manual process creates slow feedback loops and interrupts development flow

## context & problem definition

### problem statement
**Who**: Developers using claude-reactor for active development who need rapid iteration and feedback loops
**What**: Currently face slow development cycles due to manual rebuild/restart processes, breaking development flow and reducing productivity
**Why**: Manual rebuild cycles create friction that slows development velocity, especially for compiled languages and complex applications

**Current Pain Points:**
- **Slow Feedback Loops**: 30+ seconds between code change and running application
- **Manual Process**: Developers must remember to rebuild/restart after changes
- **Context Switching**: Breaking focus to manually trigger rebuilds and restarts
- **Language-Specific Complexity**: Different rebuild processes for different languages and frameworks
- **Service Coordination**: Complex applications require coordinated service restarts

### success criteria
- [ ] **Instant Feedback**: Code changes reflected in running application within 5 seconds
- [ ] **Intelligent Detection**: Only rebuild/restart when necessary based on changed file types and dependency analysis
- [ ] **Multi-Language Support**: Works across Go, Rust, Node.js, Python, Java, and hybrid projects
- [ ] **Framework Integration**: Native integration with development servers and hot reload tools
- [ ] **Resource Efficiency**: Minimal CPU and memory overhead during file watching
- [ ] **Configurable**: Customizable watch patterns, ignore rules, and rebuild strategies

## technical requirements

### functional requirements
- [ ] **File System Monitoring**: Efficient file watching across mounted volumes with debouncing and filtering
- [ ] **Language-Specific Rebuild Logic**: Intelligent rebuild strategies for each supported language and build system
- [ ] **Service Management**: Automatic process restart and service coordination within containers
- [ ] **Framework Integration**: Native integration with hot reload tools (webpack, nodemon, cargo-watch, air, etc.)
- [ ] **Configuration System**: Flexible configuration for watch patterns, ignore rules, and custom build commands
- [ ] **Development Server Integration**: Automatic proxy and port forwarding for development servers
- [ ] **Dependency Analysis**: Smart rebuilding based on dependency graphs and file change impact
- [ ] **Multi-Service Orchestration**: Coordinated rebuilds and restarts for microservice applications
- [ ] **Build Caching**: Intelligent caching to minimize rebuild times
- [ ] **Error Handling**: Graceful handling of build failures with clear error reporting
- [ ] Comprehensive documentation and operational tooling via Makefile
- [ ] Developer onboarding experience <10 minutes from clone to running

### non-functional requirements
- **Performance**: File change detection and rebuild trigger within 1 second
- **Resource Usage**: <100MB additional memory overhead, <5% CPU usage during idle watching
- **Reliability**: 99.9% file change detection accuracy across different file systems
- **Scalability**: Support for projects with 10,000+ files without performance degradation
- **Compatibility**: Works with all claude-reactor variants and supported development environments
- **Configurability**: Easy customization for different project types and development workflows
- **Operations**: All common tasks accessible via single Makefile commands
- **Documentation**: Comprehensive docs/ structure with setup, deployment, troubleshooting guides
- **Developer Experience**: <10 minutes from git clone to running locally

### Technical Constraints
- Must work within existing container mount point architecture
- File watching must be efficient across different host file systems (macOS, Linux, Windows)
- Cannot break existing development workflows or container lifecycle management
- Must integrate with existing Go CLI architecture and Docker SDK
- Language-specific tools must be available in appropriate container variants
- Must handle file system events reliably across container boundaries
- Resource usage must be minimal to maintain laptop development feasibility

## Data & Database changes

### Data model updates
**File Watching Configuration:**
```go
type WatchConfig struct {
    Enabled       bool
    WatchPatterns []string
    IgnorePatterns []string
    DebounceMs    int
    Language      string
    BuildCommand  string
    RestartCommand string
    PreBuildHooks []string
    PostBuildHooks []string
}

type FileWatcher struct {
    Config       WatchConfig
    EventChannel chan FileEvent
    Builder      LanguageBuilder
    ServiceManager ProcessManager
}
```

### Data migration plan
N/A - New feature, existing `.claude-reactor` config will be extended with optional watch configuration.

## API & Backend changes

### Data access pattern
File system monitoring through Go's `fsnotify` package and container filesystem access.

### server actions
N/A - Client-side container feature, no server-side components.

### Database queries
N/A - No database interaction required.

### API Routes
N/A - Container-based feature, no API endpoints.

## frontend changes

### New components
N/A - CLI and container-based tool with no web frontend components.

### Page updates
N/A - Command-line interface and container integration only.

## Implementation plan

### phase 1 - Core File Watching Infrastructure
**Goal**: Implement efficient, cross-platform file system monitoring with intelligent filtering
- [ ] Research and implement file system watching using Go fsnotify package
- [ ] Create debouncing and event filtering system to handle rapid file changes
- [ ] Implement configurable watch patterns and ignore rules (.git, node_modules, target/, etc.)
- [ ] Add file change event categorization (source code, config, dependencies)
- [ ] Create efficient file system traversal for large projects
- [ ] Test file watching performance across different host operating systems

### phase 2 - Language-Specific Build Integration
**Goal**: Implement intelligent rebuild strategies for each supported language and build system
- [ ] **Go Integration**: Integrate with `air` for Go hot reloading with custom build commands
- [ ] **Rust Integration**: Use `cargo-watch` for efficient Rust rebuilds with selective compilation
- [ ] **Node.js Integration**: Support `nodemon`, webpack-dev-server, and custom npm script watching
- [ ] **Python Integration**: Implement Python service restart with dependency change detection
- [ ] **Java Integration**: Add Maven/Gradle incremental compilation with selective rebuilds
- [ ] Create plugin architecture for extensible language support

### phase 3 - CLI and Configuration Integration
**Goal**: Integrate hot reload capabilities with claude-reactor CLI and configuration system
- [ ] Add `--watch` flag to `claude-reactor run` command for automatic hot reload activation
- [ ] Implement `claude-reactor watch` command for standalone file watching
- [ ] Create configuration system for watch patterns, build commands, and restart strategies
- [ ] Add auto-detection of framework-specific hot reload tools (webpack config, Cargo.toml, etc.)
- [ ] Integrate with existing project detection logic for automatic watch configuration
- [ ] Add interactive configuration setup for custom watch patterns

### phase 4 - Service Management and Process Coordination
**Goal**: Implement robust service management and coordination for complex applications
- [ ] Create process manager for automatic service restart within containers
- [ ] Implement graceful shutdown and restart sequences for running services
- [ ] Add dependency-aware restart logic (restart dependent services when shared libraries change)
- [ ] Create service health checking and failure recovery mechanisms
- [ ] Add support for multi-service coordination (microservices, full-stack applications)
- [ ] Implement port management and proxy configuration for development servers

### phase 5 - Advanced Features and Optimization
**Goal**: Add intelligent caching, performance optimization, and advanced development features
- [ ] Implement build artifact caching to minimize rebuild times
- [ ] Add dependency graph analysis for selective rebuilds
- [ ] Create intelligent ignore pattern generation based on project analysis
- [ ] Add real-time build status reporting and error notifications
- [ ] Implement change impact analysis to optimize rebuild scope
- [ ] Add integration with IDE/editor notifications for build status

### phase 6 - Testing and Documentation
**Goal**: Comprehensive testing across languages and platforms with complete documentation
- [ ] Create comprehensive test suite covering all supported languages and frameworks
- [ ] Test file watching performance and reliability across different host systems
- [ ] Validate hot reload functionality with real-world development scenarios
- [ ] Create troubleshooting guide for file watching and hot reload issues
- [ ] Document configuration options and customization patterns
- [ ] Validate <10 minute developer onboarding experience with hot reload enabled
- [ ] Create performance benchmarking and optimization guidelines

## 5. Testing Strategy

### Unit Tests
- **File System Monitoring**: Test event detection, filtering, and debouncing with mock file systems
- **Build Command Integration**: Test language-specific build command execution and error handling
- **Configuration Management**: Test watch configuration parsing, validation, and defaults
- **Process Management**: Test service restart logic, graceful shutdown, and health checking
- **Event Processing**: Test file change categorization and rebuild decision logic

### Integration Tests
- **End-to-End File Watching**: Test complete file change → rebuild → restart workflow for each language
- **Multi-Language Projects**: Test hot reload in projects with multiple languages and build systems
- **Container Integration**: Test file watching across container mount boundaries
- **Framework Integration**: Test integration with existing hot reload tools and development servers
- **Performance**: Test file watching performance with large projects and rapid change scenarios

### End-to-End (E2E) Tests
- **Complete Development Workflow**: Code change → automatic rebuild → service restart → testing
- **Multi-Service Applications**: Test coordinated rebuilds and restarts for complex applications
- **Cross-Platform**: Test file watching behavior across macOS, Linux, and Windows (WSL2)
- **Framework-Specific Scenarios**: Test with real-world frameworks (React, Express, Django, etc.)
- **Error Recovery**: Test behavior during build failures, service crashes, and file system issues

## 6. Security Considerations

### Authentication & Authorization
- File watching operates within existing container security boundaries
- No additional authentication required beyond current claude-reactor permissions
- File system access limited to existing mount points and container privileges

### Data Validation & Sanitization
- **File Path Validation**: Ensure watched file paths cannot escape container boundaries
- **Command Injection Prevention**: Sanitize custom build commands and prevent shell injection
- **Process Management Security**: Ensure spawned processes maintain container security context
- **Configuration Validation**: Validate watch patterns and ignore rules to prevent malicious configurations

### Potential Vulnerabilities
- **Resource Exhaustion**: Excessive file watching could consume system resources - mitigation: resource limits and efficient algorithms
- **Command Injection**: Custom build commands could execute malicious code - mitigation: command sanitization and safe execution
- **File System Traversal**: Malicious watch patterns could access unauthorized files - mitigation: path validation and sandboxing
- **Process Escape**: Managed processes could attempt container escape - mitigation: maintain container security boundaries

## 7. Rollout & Deployment

### Feature Flags
Hot reload will be opt-in via command-line flags (`--watch`) to maintain backward compatibility.

### Deployment Steps
1. **Core Infrastructure**: Implement file watching and build integration
2. **Language Support**: Add language-specific hot reload capabilities
3. **CLI Integration**: Add watch commands and flags to claude-reactor CLI
4. **Documentation**: Complete setup guides and configuration documentation
5. **Beta Testing**: Test with development teams across different project types
6. **Public Release**: Announce hot reload capabilities with usage examples

### Rollback Plan
- **Opt-in Feature**: Hot reload is disabled by default, no impact on existing workflows
- **Command Removal**: Remove watch-related commands and flags if needed
- **Full Backwards Compatibility**: All existing claude-reactor functionality remains unchanged
- **Resource Cleanup**: Automatic cleanup of file watching processes and resources

## 8. Open Questions & Assumptions

### Open Questions
1. Should file watching be enabled by default or require explicit opt-in?
2. How do we handle very large projects (10,000+ files) efficiently?
3. What's the optimal balance between rebuild frequency and resource usage?
4. Should we integrate with IDE/editor file watching or maintain independent watching?
5. How do we handle network file systems and remote development scenarios?
6. What level of build customization should be supported vs opinionated defaults?
7. Should we support custom notification systems (desktop notifications, Slack, etc.)?

### Assumptions
- Developers prefer automatic rebuilds over manual control once properly configured
- File watching performance overhead is acceptable for development environments
- Container mount points provide sufficient file system event propagation
- Language-specific hot reload tools are available in appropriate container variants
- Developers will configure ignore patterns to exclude unnecessary file types
- Build times are fast enough to provide meaningful hot reload experience