# Feature: Go-Based Architecture Restructure

## 1. Description
Transform the monolithic `claude-reactor` bash script into a robust, maintainable Go-based CLI tool with strong typing, comprehensive testing, and superior error handling. This restructure leverages Go's strengths for Docker integration, cross-platform distribution, and modular architecture while maintaining the zero-dependency installation experience through single-binary distribution.

## Goal statement
To eliminate technical debt and brittleness in the claude-reactor codebase by creating a modular architecture that improves maintainability, testability, and extensibility while preserving the existing user experience and installation simplicity.

## Project Analysis & current state

### Technology & architecture
- **Current**: Single monolithic bash script (`claude-reactor`) with ~900 lines handling all concerns
- **Target**: Go-based modular CLI with clean architecture and strong typing
- **Docker Integration**: Multi-stage Dockerfile with architecture-aware builds + Go Docker SDK
- **Build System**: Go modules + Makefile with cross-compilation for multiple platforms
- **Testing**: Go's built-in testing framework with mocks and table-driven tests
- **Configuration**: YAML configuration files with struct validation
- **Core Dependencies**: Go 1.21+, Docker (runtime only)

### current state
The current system works well functionally but has grown organically into a brittle monolithic structure:
- All logic (config, Docker ops, auth, architecture detection) in single 900-line script
- Mixed concerns make testing difficult and error-prone
- Complex Dockerfile with interleaved user switching and installation logic  
- Hard to extend with new variants or features without touching multiple unrelated sections
- Debugging and troubleshooting requires deep script knowledge

## context & problem definition

### problem statement
**Who**: Developers and maintainers working on claude-reactor, and indirectly end-users affected by bugs
**What**: The monolithic architecture has become difficult to maintain, test, and extend safely
**Why**: As features accumulated, the single-script approach created tight coupling, making changes risky and time-consuming

Current pain points:
- **Maintainability**: Changes require understanding the entire 900-line script
- **Testing**: Hard to test individual components in isolation
- **Extensibility**: Adding new variants/features touches multiple unrelated concerns
- **Debugging**: Complex interdependencies make issue isolation difficult
- **Onboarding**: New contributors face steep learning curve

### success criteria
- [ ] **CLI Excellence**: Modern, intuitive CLI interface with improved UX over bash version
- [ ] **Maintainability**: Each Go package <500 lines, clear interfaces, single responsibility  
- [ ] **Testability**: >90% unit test coverage with comprehensive mocking
- [ ] **Build Time**: Cross-compilation for all platforms <30 seconds
- [ ] **Distribution**: Single-binary per platform maintains zero-dependency installation
- [ ] **Performance**: Startup time ≤ 100ms, memory usage <50MB
- [ ] **Developer Experience**: Standard Go tooling, hot reload, comprehensive error messages
- [ ] **Type Safety**: Compile-time validation of configuration and parameters
- [ ] **Migration Path**: Clear migration guide and tooling from bash version

## technical requirements

### functional requirements
- [ ] **Go Package Structure**: Clean architecture with separate packages (config, docker, auth, architecture)
- [ ] **Cross-Platform Binaries**: Build native binaries for linux/darwin on amd64/arm64
- [ ] **Modern CLI Interface**: Improved UX with subcommands, better help, and intuitive workflows
- [ ] **Configuration Migration**: Tooling to migrate from .claude-reactor to new YAML format
- [ ] **YAML Configuration**: Strongly-typed configuration with validation and schema
- [ ] **Docker SDK Integration**: Native Go Docker client with fallback to CLI where needed
- [ ] **Comprehensive Testing**: Unit tests with mocks, integration tests, benchmarks
- [ ] **Structured Logging**: Configurable log levels with structured output and debug modes
- [ ] **Error Handling**: Wrapped errors with context, suggestions, and actionable error messages
- [ ] **Progress Indicators**: Progress bars and status updates for long-running operations
- [ ] Comprehensive documentation and operational tooling via Makefile
- [ ] Developer onboarding experience <10 minutes from clone to running

### non-functional requirements
- **Performance**: Startup time ≤100ms, memory usage <50MB at runtime
- **Binary Size**: Go binary <20MB per platform (reasonable for single-file distribution)
- **Reliability**: No functional regressions, comprehensive error handling with recovery
- **Maintainability**: Interface-driven design, dependency injection, clear package boundaries
- **Extensibility**: Plugin architecture for new variants, authentication providers, etc.
- **Observability**: Structured logging, metrics collection, debug modes
- **Operations**: All common tasks accessible via single Makefile commands
- **Documentation**: Comprehensive docs/ structure with setup, deployment, troubleshooting guides
- **Developer Experience**: <10 minutes from git clone to running locally

### Technical Constraints
- **Go Version**: Minimum Go 1.21 for development, no runtime Go dependency
- **Zero Runtime Dependencies**: Single static binary, no external dependencies at runtime
- **Docker Integration**: Use Docker SDK for Go where possible, fallback to CLI for complex operations
- **Cross-Platform**: Support macOS (darwin) and Linux on amd64/arm64 architectures
- **Build Dependencies**: Only Go toolchain required for building, optional tools for development
- **Memory Limits**: Designed for developer laptops, reasonable memory usage expectations

## Data & Database changes

### Data model updates
N/A - No database involved, only file-based configuration.

### Data migration plan
**Configuration Migration Strategy**:
- Migration tool to convert existing `.claude-reactor` files to new YAML format
- Clear migration guide with before/after examples
- One-time migration on first run with user confirmation
- Breaking change acceptable - users will adapt to improved configuration

## API & Backend changes

### Data access pattern
N/A - This is a CLI tool restructure, no API changes.

### server actions
N/A - No server-side components.

### Database queries
N/A - No database integration.

### API Routes
N/A - CLI tool only.

## frontend changes

### New components
N/A - Command-line interface only.

### Page updates
N/A - No UI components.

## Implementation plan

### phase 1 - Go Project Setup and Core Architecture
**Goal**: Establish Go project structure with clean architecture and core interfaces
- [ ] Initialize Go module with proper project structure (`cmd/`, `internal/`, `pkg/`, `configs/`)
- [ ] Design core interfaces: `ConfigManager`, `DockerClient`, `ArchDetector`, `AuthManager`
- [ ] Set up dependency injection container/framework (wire or fx)
- [ ] Implement `internal/architecture/` - host architecture detection and Docker platform mapping
- [ ] Implement `internal/config/` - YAML configuration loading with validation
- [ ] Implement `internal/logging/` - structured logging with configurable levels
- [ ] Set up Cobra CLI framework with command structure
- [ ] Create basic `cmd/claude-reactor/main.go` with version and help commands

### phase 2 - Docker Integration and Container Management  
**Goal**: Implement Docker operations using Go SDK with fallback to CLI
- [ ] Implement `internal/docker/` package with Docker SDK integration
- [ ] Container lifecycle management (build, start, stop, remove)
- [ ] Image management with architecture-aware building
- [ ] Mount point handling and volume management
- [ ] Network configuration and port exposure
- [ ] Health checks and container status monitoring
- [ ] Graceful fallback to Docker CLI for complex operations
- [ ] Comprehensive error handling with actionable messages

### phase 3 - Authentication and Account Management
**Goal**: Claude authentication handling with multi-account support
- [ ] Implement `internal/auth/` package for Claude account management
- [ ] OAuth token handling and refresh logic
- [ ] Account isolation and configuration management
- [ ] API key management with secure storage
- [ ] Interactive login flows with proper UX
- [ ] Account switching and profile management
- [ ] Migration from existing `.claude.json` files

### phase 4 - Configuration System and Migration
**Goal**: Robust YAML-based configuration with validation and migration
- [ ] Design configuration schema with struct tags for validation
- [ ] Implement variant definitions in `configs/variants.yaml`
- [ ] Implement tool specifications in `configs/tools.yaml`
- [ ] Configuration validation with detailed error messages
- [ ] Migration tool for existing `.claude-reactor` files
- [ ] Configuration merging (defaults < user config < CLI flags)
- [ ] Hot-reloading configuration for development

### phase 5 - CLI Interface and Commands
**Goal**: Intuitive CLI interface with improved UX over bash version
- [ ] Implement all core commands (`run`, `build`, `config`, `clean`)
- [ ] Rich help system with examples and troubleshooting
- [ ] Progress bars and status indicators for long operations
- [ ] Interactive prompts for configuration and account setup
- [ ] Shell completion for bash/zsh/fish
- [ ] Configuration management commands (`config edit`, `config validate`)
- [ ] Debug commands for troubleshooting (`debug info`, `debug logs`)

### phase 6 - Cross-Platform Building and Distribution
**Goal**: Multi-platform binary distribution with automated releases
- [ ] Cross-compilation setup for linux/darwin on amd64/arm64
- [ ] Automated build pipeline with version embedding
- [ ] Binary optimization and size reduction techniques
- [ ] Installation scripts for different platforms
- [ ] GitHub Actions for automated releases
- [ ] Package manager integration (Homebrew, etc.)
- [ ] Update mechanism with version checking

### phase 7 - Testing and Quality Assurance
**Goal**: Comprehensive testing strategy with high coverage
- [ ] Unit tests for all packages with mocking (testify/mock)
- [ ] Integration tests with real Docker (testcontainers-go)
- [ ] Table-driven tests for complex scenarios
- [ ] Benchmarks for performance-critical operations
- [ ] End-to-end tests with real Claude CLI interaction
- [ ] Cross-platform testing in CI/CD
- [ ] Performance regression testing

### phase 8 - Documentation and Developer Experience
**Goal**: Comprehensive documentation and smooth developer onboarding
- [ ] API documentation with examples (godoc)
- [ ] Architecture documentation with diagrams
- [ ] User migration guide from bash version
- [ ] Developer setup guide with prerequisites
- [ ] Contributing guidelines and code standards
- [ ] Troubleshooting guide with common issues
- [ ] Performance tuning and configuration optimization guide

## 5. Testing Strategy

### Unit Tests
- **Package Functions**: Test each exported function with table-driven tests and mocked dependencies
- **Interface Mocking**: Use testify/mock for Docker client, filesystem, and external service mocks
- **Configuration Validation**: Test YAML parsing, validation, and error scenarios
- **Architecture Detection**: Test detection across different platforms with mocked `runtime` calls
- **Error Handling**: Validate error wrapping, context, and user-friendly messages
- **Concurrent Operations**: Test goroutine safety and concurrent Docker operations

### Integration Tests  
- **Docker SDK Integration**: Test real Docker operations with testcontainers-go
- **Configuration Loading**: Test YAML config loading with real files and edge cases
- **Cross-Package Integration**: Test interactions between config, docker, auth packages
- **CLI Command Integration**: Test cobra commands with real flag parsing and validation
- **File System Operations**: Test configuration file reading/writing with temporary directories
- **Migration Testing**: Test conversion from old .claude-reactor format to new YAML

### End-to-End (E2E) Tests
- **Complete Workflows**: Full CLI scenarios from initialization to container cleanup
- **Cross-Platform Testing**: Automated testing on Linux and macOS in CI/CD
- **Container Variant Testing**: Test each variant builds and runs correctly
- **Account Management**: Test multi-account workflows with real Claude authentication
- **Performance Testing**: Benchmark startup time, memory usage, and Docker operations
- **Error Recovery**: Test graceful handling of Docker daemon failures, network issues

## 6. Security Considerations

### Authentication & Authorization
- Maintain existing Claude authentication patterns without changes
- Preserve account isolation mechanisms
- Ensure no regression in API key handling security

### Data Validation & Sanitization
- **Configuration Validation**: Strict validation of config files and user inputs
- **Path Injection Prevention**: Sanitize all file paths and mount points
- **Command Injection**: Validate all parameters passed to Docker and system commands
- **Module Boundaries**: Ensure modules can't access data outside their scope

### Potential Vulnerabilities
- **Bundle Tampering**: Risk of modified bundled scripts - mitigation: checksums and signing
- **Module Path Injection**: Malicious module paths - mitigation: strict module loading validation
- **Configuration Injection**: Malicious config values - mitigation: input sanitization and validation
- **Docker Escape**: Enhanced Docker operations - mitigation: maintain current security boundaries

## 7. Rollout & Deployment

### Feature Flags
No feature flags needed - this is an internal restructure maintaining identical external behavior.

### Deployment Steps
1. **Development Phase**: Work in `src/` directory with modular structure
2. **Bundle Generation**: `make build` creates single-file distribution in `dist/`
3. **Testing Phase**: Validate bundled version matches modular behavior
4. **Documentation**: Update all guides and examples
5. **Release**: Replace monolithic script with generated bundle

### Rollback Plan
- **Immediate Rollback**: Keep current monolithic script as `claude-reactor.legacy`
- **Automated Fallback**: If bundle fails, automatically fall back to legacy version
- **User Override**: `--use-legacy` flag to force monolithic version
- **Git Rollback**: Simple git revert restores monolithic structure

## 8. Open Questions & Assumptions

### **DECISIONS MADE**
1. **Technology Stack**: ✅ **Go-Based Architecture** - Rewrite from bash to Go for superior maintainability
2. **Backwards Compatibility**: ✅ **No Backwards Compatibility** - Breaking changes acceptable for better design
3. **Configuration Format**: ✅ **YAML Configuration** - Strongly-typed YAML with migration tooling
4. **Build System**: ✅ **Cross-Compilation** - Native binaries for each platform, no bundling complexity
5. **Testing Strategy**: ✅ **Comprehensive Go Testing** - Unit tests with mocks, integration tests, benchmarks
6. **CLI Interface**: ✅ **Modern CLI** - Cobra-based CLI with improved UX over bash version

### **ARCHITECTURAL DECISIONS FINALIZED**

#### **1. Installation and Distribution** ✅ **Option A**
- **Decision**: GitHub releases with platform-specific binaries + well-documented install script
- **Implementation**: Create `install.sh` that detects platform and downloads appropriate binary
- **Documentation**: Clear installation docs with troubleshooting for all platforms

#### **2. CLI Interface Design** ✅ **Modern Subcommands**
- **Decision**: Modern subcommand interface with Cobra framework
- **Interface**: `claude-reactor run --variant=go --danger --account=work`
- **Commands**: `run`, `build`, `clean`, `config`, `version`, `debug`

#### **3. Configuration Migration Strategy** ✅ **Scorched Earth**
- **Decision**: No migration - complete replacement of existing bash version
- **Approach**: Start fresh with new YAML configuration format
- **Impact**: Clean slate allows optimal design without legacy constraints

#### **4. Docker Integration Approach** ✅ **Docker SDK Only**
- **Decision**: Use Docker SDK for Go exclusively for professional, robust implementation
- **Benefits**: Better error handling, type safety, no shell command complexity
- **Trade-off**: More complex but superior reliability and maintainability

#### **5. Go Module Dependencies** ✅ **Standard Go Ecosystem**
- **Decision**: Use well-maintained community libraries (Cobra, Viper, Docker SDK, logrus)
- **Philosophy**: No vendoring, embrace Go ecosystem best practices
- **Dependencies**: Minimal but high-quality, well-maintained packages only

#### **6. Development Build Requirements** ✅ **Go Required**
- **Decision**: Go 1.21+ required for all development work
- **Justification**: Professional development setup, standard Go tooling
- **Impact**: Clear development requirements, no complexity from supporting multiple environments

#### **7. Performance and Resource Requirements** ✅ **Track but Don't Optimize**
- **Decision**: Implement performance tracking for startup time, memory, binary size
- **Targets**: Monitor <100ms startup, <50MB RAM, <20MB binary
- **Approach**: Benchmark tests to track performance, optimize if needed

#### **8. Error Handling and User Experience** ✅ **Structured Errors with Suggestions**
- **Decision**: Implement structured errors with context and actionable suggestions
- **Implementation**: Wrapped errors with troubleshooting links and next steps
- **Future**: Add interactive recovery for common issues if patterns emerge

### Assumptions
- Go 1.21+ acceptable as development requirement
- Breaking changes acceptable for long-term benefits  
- Users willing to adapt to improved CLI interface
- Single atomic release preferred over gradual migration
- Cross-platform binary distribution feasible
- Docker SDK integration provides sufficient functionality