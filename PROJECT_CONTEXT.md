# PROJECT_CONTEXT.md

## Project Purpose & Philosophy

**Claude-Reactor** is a secure, professional Docker containerization system for Claude CLI that prioritizes **account isolation**, **zero-configuration experience**, and **enterprise-grade development workflows**. 

The project exists to solve a fundamental problem: enabling safe, isolated Claude CLI usage in containerized environments with complete separation between different users, projects, and accounts, while maintaining the ease and familiarity of local development.

## Core Design Principles

### 1. **Security by Default**
- **Account Isolation**: Complete separation of credentials, sessions, and containers between Claude accounts
- **Read-Only Mounts**: SSH keys, configs, and sensitive files mounted read-only when possible  
- **No Private Key Exposure**: SSH agent forwarding preferred over key copying
- **Minimal Privileges**: Non-root container execution with least-privilege access

### 2. **Zero-Configuration Experience**
- **Intelligent Auto-Detection**: Project type, architecture, and optimal container variant detected automatically
- **Smart Defaults**: Sensible fallbacks that work for 90% of use cases without manual configuration
- **Progressive Enhancement**: Basic functionality works immediately, advanced features available when needed

### 3. **Professional Standards**
- **Interface-Driven Design**: Clear contracts between components via `pkg/interfaces.go`
- **Comprehensive Testing**: Minimum 35% coverage with unit, integration, and E2E tests
- **CI/CD Automation**: GitHub Actions for multi-architecture builds, testing, and security scanning
- **Makefile Automation**: All development, testing, and deployment tasks accessible via single commands

### 4. **Smart Persistence & Reuse**
- **Configuration Memory**: User preferences persisted per project/account combination
- **Intelligent Container Lifecycle**: Automatic reuse vs recreation based on argument presence
- **Session Continuity**: Conversation history and development state maintained across container restarts

### 5. **Modular & Extensible Architecture**
- **Variant System**: Multiple specialized container environments (base, go, full, cloud, k8s)
- **Plugin Architecture**: Mount system, authentication providers, and validation components designed for extension
- **Registry Integration**: Automatic image pulling with local build fallback

## Architecture Overview

### **Layered Design**
```
┌─────────────────────────────────────────────────────────────┐
│                     CLI Interface (cmd/)                    │
├─────────────────────────────────────────────────────────────┤
│                Business Logic (internal/reactor/)           │
├─────────────────────────────────────────────────────────────┤
│              Shared Interfaces (pkg/interfaces.go)         │
├─────────────────────────────────────────────────────────────┤
│           Docker Engine / Container Runtime                 │
└─────────────────────────────────────────────────────────────┘
```

### **Core Components**
- **Configuration Management**: YAML-based persistence with account/project isolation
- **Docker Abstraction**: Clean interface over Docker SDK with lifecycle management
- **Authentication System**: Account-specific credential management with OAuth support
- **Mount Management**: Secure, validated file system mounting with permission control including subagent directories
- **Image Validation**: Custom Docker image compatibility and security checking

### **Multi-Variant Container System**
Each variant builds upon the previous, providing specialized development environments:
- `base` → `go` → `full` → `cloud` / `k8s`
- ARM64-first design with AMD64 compatibility
- Registry-distributed with local build fallback

## Key Files & Navigation

### **Essential Understanding** (read these first)
- **`pkg/interfaces.go`** - Core system contracts and data structures
- **`CLAUDE.md`** - Project instructions and conventions for AI collaborators  
- **`cmd/claude-reactor/commands/run.go`** - Main application logic and container lifecycle
- **`Makefile`** - Professional build automation with 25+ targets

### **Architecture Implementation**
- **`internal/reactor/`** - Core business logic implementation
  - `config/` - Configuration management and persistence
  - `docker/` - Container lifecycle and Docker operations
  - `auth/` - Account isolation and authentication  
  - `mount/` - Secure file system mounting
- **`cmd/claude-reactor/commands/`** - CLI command implementations
- **`pkg/`** - Shared data structures and interfaces

### **Container Definitions**
- **`Dockerfile`** - Multi-stage container builds with security hardening
- **`entrypoint.sh`** - Container initialization and Claude CLI setup

### **Quality Assurance**
- **`.github/workflows/`** - CI/CD automation and multi-architecture builds
- **`tests/`** - Comprehensive test suite with unit, integration, and demo validation
- **`.github/COVERAGE.md`** - Coverage guidelines and improvement strategies

### **Documentation Structure**
- **`docs/`** - Comprehensive user and developer documentation
- **`ai-prompts/`** - Feature specifications and implementation context
- **`examples/`** - Usage patterns and configuration examples

## Development Philosophy

### **Interface-First Design**
All major components defined as interfaces in `pkg/interfaces.go` before implementation. This enables:
- Clear contracts between layers
- Easy testing with mocks
- Modularity and extensibility
- Dependency injection patterns

### **Configuration Over Code**
User preferences, container settings, and behavioral options controlled through:
- Project-specific `.claude-reactor` files
- Account-specific session directories  
- Environment-based overrides
- Smart defaults with explicit control

### **Security-Conscious Development**
Every feature evaluated through security lens:
- Principle of least privilege
- Input validation and sanitization  
- Read-only mounts where possible
- Account isolation as first-class feature

### **Professional Operations**
Development workflow optimized for team collaboration:
- Makefile-driven automation
- Comprehensive CI/CD pipeline
- Multi-architecture support
- Registry-based distribution

## Technology Choices & Rationale

### **Go Language**
- **Type Safety**: Interfaces and strong typing prevent runtime errors
- **Container Ecosystem**: Native Docker SDK integration and tooling
- **Cross-Platform**: Single binary deployment across architectures
- **Performance**: Fast startup and low resource usage

### **Docker as Foundation**
- **Industry Standard**: Ubiquitous container runtime with broad compatibility
- **Security Features**: Namespace isolation, resource limits, and privilege dropping
- **Ecosystem**: Rich registry ecosystem and multi-architecture support
- **Development Familiarity**: Most developers already understand Docker concepts

### **YAML Configuration**
- **Human Readable**: Easy editing and version control
- **Structured**: Type safety with Go struct marshaling
- **Standard**: Widely adopted in DevOps and container ecosystems
- **Extensible**: Simple addition of new configuration options

### **Cobra CLI Framework**
- **Professional**: Industry standard for Go CLI applications
- **Feature Rich**: Built-in help, completion, and flag management
- **Extensible**: Easy addition of new commands and subcommands
- **Consistent**: Familiar patterns for Go developers

## Quality Standards & Expectations

### **Testing Requirements**
- **Minimum 35% Coverage**: Enforced via CI/CD pipeline
- **Multiple Test Types**: Unit, integration, and end-to-end validation
- **Platform Testing**: Cross-platform compatibility verification
- **Mock-Based Testing**: Interface-driven testing with dependency injection

### **Code Quality**
- **Interface Contracts**: All major components must implement defined interfaces
- **Error Handling**: Comprehensive error messages with actionable guidance
- **Logging**: Structured logging with appropriate verbosity levels
- **Documentation**: Inline documentation for all public interfaces

### **Security Standards**
- **Input Validation**: All user inputs validated and sanitized
- **Permission Checks**: File and directory access properly validated
- **Credential Handling**: No secrets in logs, proper credential isolation
- **Container Security**: Non-root execution, minimal container privileges

### **User Experience**
- **Progressive Disclosure**: Simple commands work immediately, complexity available when needed
- **Helpful Errors**: Error messages include diagnosis and resolution guidance
- **Smart Defaults**: Zero configuration required for standard use cases
- **Consistent Patterns**: Similar operations work the same way across commands

## Current Implementation Status

### ✅ **Completed Features**
- **Account Isolation System**: Complete separation of credentials, sessions, and containers
- **Subagent Support**: Automatic mounting of global (~/.claude/agents/) and project-specific (.claude/agents/) subagents
- **VS Code Integration**: Automatic dev container generation with project detection
- **Project Templates**: Interactive scaffolding with built-in language templates
- **Custom Image Support**: Docker image validation and compatibility checking
- **Registry Integration**: Multi-architecture container distribution with local fallback
- **Smart Container Management**: Intelligent reuse vs recreation based on arguments
- **Professional CLI**: Comprehensive Cobra-based interface with 25+ commands
- **CI/CD Pipeline**: Multi-architecture builds, testing, and security scanning

### 🚧 **In Development** 
- **SSH Support**: Secure key forwarding and Git integration (Phase 1)
- **Enhanced Validation**: Improved container image compatibility testing
- **Documentation Consolidation**: Streamlined user and developer documentation

### 📋 **Planned Enhancements** (see ai-prompts/ for detailed specs)
- **Hot Reload & File Watching**: Real-time code synchronization
- **Package Manager Integration**: Unified dependency management across languages  
- **Multi-Architecture Support**: Enhanced cross-platform deployment
- **Environment Management**: Advanced configuration and secrets handling

### 🎯 **Development Patterns**
When implementing new features, follow these established patterns:
1. **Interface-First**: Define contracts in `pkg/interfaces.go` before implementation
2. **Test-Driven**: Minimum 35% coverage with unit + integration tests
3. **CLI Integration**: Use Cobra commands with comprehensive help and examples
4. **Configuration Management**: YAML-based with smart defaults and validation
5. **Professional Quality**: Error handling, logging, and user feedback

## Project Maturity & Context

**Current State**: Production-ready system with comprehensive feature set
**Architecture**: Stable interface-based design with extensible components  
**Quality**: 35% test coverage with CI/CD automation and security scanning
**Distribution**: Multi-architecture containers with registry automation
**Documentation**: Complete user documentation in README.md, developer context in this file

This project represents a mature, professionally-designed system that balances ease of use with enterprise security requirements while maintaining extensibility for future enhancements through the ai-prompts/ specification system.