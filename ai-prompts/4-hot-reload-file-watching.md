# Feature: Hot Reload & File Watching

## 1. Description
Implement intelligent file watching and hot reload capabilities that automatically detect code changes, rebuild applications, restart services, and refresh development environments within claude-reactor containers. This eliminates manual restart cycles and provides instant feedback loops for rapid development, supporting multiple languages and frameworks with optimized rebuild strategies.

## Goal statement (Updated)
**Original Goal**: To provide developers with instant feedback on code changes through intelligent file watching, automatic rebuilding, and service restarting that works seamlessly across all supported languages and frameworks within claude-reactor containers.

**Current Reality**: To provide sophisticated file change monitoring with comprehensive CLI tooling and session management, laying the foundation for future automated build and restart capabilities.

**Gap**: The core value proposition of "instant feedback" and "automatic rebuilding" is not delivered by the current implementation.

## **Current Implementation Status** ⚠️ **CRITICAL UPDATE - January 2025**

**Implementation Level: Partial (Phase 1 Complete, Core Functionality Missing)**

### **✅ What Currently Works:**
- **CLI Interface**: Comprehensive `claude-reactor hotreload` commands (start, stop, status, list, config)
- **Project Detection**: Accurate detection of Go, Node.js, Python, Rust, Java projects with framework identification
- **File Watching**: Efficient file system monitoring using fsnotify with debouncing and pattern filtering
- **Session Management**: Multi-session tracking with metrics, activity logs, and status reporting
- **Configuration System**: Flexible watch patterns, ignore rules, and debounce settings

### **❌ What's NOT Implemented (Critical Gaps):**
- **Build Execution**: File changes detected but builds are NOT triggered (placeholder code at `watcher.go:355`)
- **Container Sync**: Session framework exists but no actual file/artifact synchronization (`sync.go:352`)
- **Service Management**: No automatic process restart or service coordination
- **Framework Integration**: No integration with air, nodemon, cargo-watch, webpack, etc.
- **Make/Custom Build Support**: Hard-coded build commands only (`go build .`), no Makefile detection

### **Current User Experience:**
```bash
# This works - shows file changes detected
claude-reactor hotreload start
🔥 Starting hot reload... ✅
📁 Project Type: go (95% confidence) ✅
👀 File watching: active ✅
🔨 Build: idle ❌ (detected changes but no builds executed)
📋 Recent Activity: "File change detected: main.go" ✅ (but no follow-up build)
```

**Translation**: Currently provides sophisticated file monitoring and status reporting, but **does NOT deliver the core value proposition** of automated builds and rapid feedback loops.

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
**Current Development Workflow (Updated January 2025):**
1. Developer makes code changes in host editor/IDE ✅
2. Changes are visible in container through mounted volumes ✅
3. **HotReload detects file changes and logs them** ✅ (NEW)
4. **Developer must still manually rebuild/restart services** ❌ (UNCHANGED - core gap)
5. Manual testing and validation of changes ❌ (UNCHANGED)
6. Repeat cycle for each change iteration ❌ (UNCHANGED)

**Partially Implemented Hot Reload Integration:**
- ✅ **File Change Detection**: Sophisticated file watching with fsnotify, debouncing, and pattern matching
- ✅ **Project Intelligence**: Auto-detection of Go, Node.js, Python, Rust, Java projects and frameworks
- ✅ **Session Management**: Multi-project tracking with comprehensive status reporting
- ❌ **Build Execution**: File changes trigger detection but **NO actual build execution** (critical gap)
- ❌ **Service Restart**: No automatic process management or service coordination
- ❌ **Framework Integration**: No integration with webpack-dev-server, cargo watch, air, nodemon, etc.
- ❌ **Container Sync**: No synchronization of build artifacts to running containers

**Real Impact**: HotReload provides **enhanced monitoring and status reporting** but **does not eliminate manual rebuild cycles** - the core value proposition remains undelivered.

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

### success criteria (Updated January 2025)
- [ ] **Instant Feedback**: Code changes reflected in running application within 5 seconds ❌ **NOT ACHIEVED** - no build execution
- [x] **Intelligent Detection**: Only rebuild/restart when necessary based on changed file types and dependency analysis ✅ **PARTIALLY** - detection works, no rebuilds
- [x] **Multi-Language Support**: Works across Go, Rust, Node.js, Python, Java, and hybrid projects ✅ **PARTIALLY** - detection only, no build execution
- [ ] **Framework Integration**: Native integration with development servers and hot reload tools ❌ **NOT ACHIEVED** - no air, nodemon, webpack integration
- [x] **Resource Efficiency**: Minimal CPU and memory overhead during file watching ✅ **ACHIEVED** - efficient fsnotify implementation
- [x] **Configurable**: Customizable watch patterns, ignore rules, and rebuild strategies ✅ **ACHIEVED** - comprehensive configuration system

**New Critical Requirements (Missing from Original Spec):**
- [ ] **Make/Build System Support**: Auto-detect and use Makefile, custom build scripts, and project-specific build commands ❌ **NOT ADDRESSED**
- [ ] **Build Output Integration**: Capture and display build results, errors, and warnings in HotReload status ❌ **NOT IMPLEMENTED**
- [ ] **Cross-Platform Build Command Detection**: Support different build systems beyond hard-coded language defaults ❌ **NOT IMPLEMENTED**
- [ ] **Framework Dev Server Integration**: Support for webpack-dev-server, air, nodemon, cargo-watch ❌ **NOT IMPLEMENTED**
- [ ] **Error Recovery and Reporting**: Handle build failures gracefully with actionable feedback ❌ **NOT IMPLEMENTED**

**Current Score: 3/11 criteria achieved (27% complete)**

**Development Progress Assessment**:
- **Phase 1 (File Watching)**: ✅ **100% Complete** - Production-ready, efficient, configurable
- **Phase 2 (Build Integration)**: ❌ **0% Complete** - Core execution logic is placeholder code
- **Overall HotReload Value Delivery**: ❌ **~15% Complete** - Monitoring without automation

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

## Implementation plan (Updated January 2025)

### ✅ phase 1 - Core File Watching Infrastructure **COMPLETED**
**Goal**: Implement efficient, cross-platform file system monitoring with intelligent filtering
- [x] Research and implement file system watching using Go fsnotify package ✅ **DONE**
- [x] Create debouncing and event filtering system to handle rapid file changes ✅ **DONE**
- [x] Implement configurable watch patterns and ignore rules (.git, node_modules, target/, etc.) ✅ **DONE**
- [x] Add file change event categorization (source code, config, dependencies) ✅ **DONE**
- [x] Create efficient file system traversal for large projects ✅ **DONE**
- [x] Test file watching performance across different host operating systems ✅ **DONE**

**Status**: ✅ **PHASE 1 COMPLETE** - File watching infrastructure is solid and production-ready

### ❌ phase 2 - Language-Specific Build Integration **CRITICAL GAP**
**Goal**: Implement intelligent rebuild strategies for each supported language and build system
**Current Status**: ⚠️ **DETECTION ONLY - NO EXECUTION** (see `watcher.go:355` placeholder)

- [ ] **Go Integration**: ❌ **BLOCKED** - No air integration, hard-coded `go build .` commands
- [ ] **Rust Integration**: ❌ **BLOCKED** - No cargo-watch integration
- [ ] **Node.js Integration**: ❌ **BLOCKED** - No nodemon, webpack-dev-server integration
- [ ] **Python Integration**: ❌ **BLOCKED** - No service restart implementation
- [ ] **Java Integration**: ❌ **BLOCKED** - No Maven/Gradle execution
- [ ] **Make/Build System Support**: ❌ **NEW REQUIREMENT** - Not in original spec, critically needed
- [ ] Create plugin architecture for extensible language support ❌ **BLOCKED**

**Blocker**: Core build execution logic is placeholder code - needs complete implementation

### ✅/❌ phase 3 - CLI and Configuration Integration **PARTIALLY COMPLETE**
**Goal**: Integrate hot reload capabilities with claude-reactor CLI and configuration system
**Current Status**: ⚠️ **CLI COMPLETE, INTEGRATION MISSING**

- [ ] Add `--watch` flag to `claude-reactor run` command ❌ **NOT IMPLEMENTED** - Different command structure used
- [x] Implement `claude-reactor hotreload` command for standalone file watching ✅ **DONE** - Comprehensive command set
- [x] Create configuration system for watch patterns, build commands, and restart strategies ✅ **DONE** - Flexible configuration
- [x] Add auto-detection of framework-specific hot reload tools (webpack config, Cargo.toml, etc.) ✅ **PARTIALLY** - Detection only
- [x] Integrate with existing project detection logic for automatic watch configuration ✅ **DONE** - Works well
- [ ] Add interactive configuration setup for custom watch patterns ❌ **NOT IMPLEMENTED**

**Status**: ✅ **CLI INFRASTRUCTURE COMPLETE** - Excellent command interface, missing core execution

### 🆕 **NEW** phase 2a - Build System Integration **CRITICAL PRIORITY**
**Goal**: Implement comprehensive build system support beyond hard-coded language defaults
**Priority**: ⚠️ **CRITICAL** - Must be implemented before Phase 2 language integrations

- [ ] **Makefile Detection**: Auto-detect and use Makefile targets (make build, make test, make run)
- [ ] **Custom Build Script Support**: Support project-specific build scripts and commands
- [ ] **Build System Priority**: Prefer Makefile/custom scripts over hard-coded language defaults
- [ ] **Build Command Configuration**: Allow manual override of detected build commands
- [ ] **Build Output Capture**: Implement actual command execution with output capture and error handling
- [ ] **Build Result Integration**: Display build success/failure and output in hotreload status
- [ ] **Working Directory Management**: Execute builds in correct project directories

**Implementation Note**: This addresses the core execution gap at `watcher.go:355` and enables real build automation.

### ❌ phase 4 - Service Management and Process Coordination **NOT STARTED**
**Goal**: Implement robust service management and coordination for complex applications
**Current Status**: ⚠️ **COMPLETELY MISSING** - No service coordination or process management exists
**Goal**: Implement robust service management and coordination for complex applications
- [ ] **Process Manager**: ❌ **NOT IMPLEMENTED** - No automatic service restart capabilities
- [ ] **Graceful Shutdown**: ❌ **NOT IMPLEMENTED** - No restart sequence management
- [ ] **Dependency Logic**: ❌ **NOT IMPLEMENTED** - No dependency-aware restart capabilities
- [ ] **Health Checking**: ❌ **NOT IMPLEMENTED** - No service health monitoring
- [ ] **Multi-Service Support**: ❌ **NOT IMPLEMENTED** - No microservice coordination
- [ ] **Port Management**: ❌ **NOT IMPLEMENTED** - No development server integration

**Blocker**: Requires Phase 2a build execution foundation before service management becomes relevant.

### ❌ phase 5 - Advanced Features and Optimization **NOT STARTED**
**Goal**: Add intelligent caching, performance optimization, and advanced development features
**Current Status**: ⚠️ **FUTURE PHASE** - Depends on core build execution being implemented first

- [ ] **Build Caching**: ❌ **NOT IMPLEMENTED** - No artifact caching exists
- [ ] **Dependency Analysis**: ❌ **NOT IMPLEMENTED** - No selective rebuild logic
- [ ] **Smart Ignore Patterns**: ❌ **NOT IMPLEMENTED** - Uses basic hardcoded patterns only
- [ ] **Real-time Reporting**: ❌ **NOT IMPLEMENTED** - Status shows detection only, no build results
- [ ] **Impact Analysis**: ❌ **NOT IMPLEMENTED** - No change scope optimization
- [ ] **IDE Integration**: ❌ **NOT IMPLEMENTED** - No editor notification system

**Blocker**: All advanced features require working build execution from Phase 2a.

### ❌ phase 6 - Testing and Documentation **NOT STARTED**
**Goal**: Comprehensive testing across languages and platforms with complete documentation
**Current Status**: ⚠️ **PARTIAL** - File watching tests exist, build execution tests missing

- [x] **File Watching Tests**: ✅ **EXISTS** - Unit tests for fsnotify and pattern matching in place
- [ ] **Language Framework Tests**: ❌ **MISSING** - No end-to-end build tests since builds don't execute
- [ ] **Performance Testing**: ❌ **MISSING** - File watching performance not comprehensively tested
- [ ] **Real-world Scenarios**: ❌ **IMPOSSIBLE** - Cannot test without working build execution
- [ ] **Troubleshooting Guide**: ❌ **MISSING** - No docs for build failures since builds don't run
- [ ] **Configuration Documentation**: ❌ **MISSING** - No docs for build command customization
- [ ] **Onboarding Experience**: ❌ **BROKEN** - Users expect builds to work but they don't
- [ ] **Performance Benchmarks**: ❌ **MISSING** - No build performance metrics available

**Critical Gap**: Testing strategy assumes working build execution, but core functionality is placeholder code.

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