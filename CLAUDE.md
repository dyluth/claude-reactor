# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Claude-Reactor** is a professional, modular Docker containerization system for Claude CLI development workflows. It transforms the basic Claude CLI into a comprehensive development environment with intelligent automation, multi-language support, and production-ready tooling.

**Primary Use Case**: Personal and team development workflows that require isolated, reproducible development environments with smart tooling automation.

**Key Value Propositions**:
- **Zero Configuration**: Auto-detects project type and sets up appropriate development environment
- **Language Agnostic**: Supports Go, Rust, Java, Python, Node.js, and cloud development workflows  
- **Professional Automation**: Makefile + script integration for both development and CI/CD
- **Persistent Intelligence**: Remembers your preferences and configurations per project
- **Comprehensive Testing**: Unit tests, integration tests, and interactive demonstrations

## Project Architecture

This project creates a modular Docker containerization system for Claude CLI with multiple specialized variants:

### **Container Variants:**
- **base**: Node.js, Python, basic development tools (smallest, ~500MB)
- **go**: Base + Go toolchain and development utilities (~800MB)  
- **full**: Go + Rust, Java, database clients (~1.2GB)
- **cloud**: Full + AWS/GCP/Azure CLIs (~1.5GB)
- **k8s**: Full + Enhanced Kubernetes tools (helm, k9s, stern) (~1.4GB)

### **Multi-stage Dockerfile:**
- Uses Docker multi-stage builds for efficiency
- Each variant builds upon the previous stage
- ARM64-compatible for M1 Macs
- Non-root `claude` user for security

### **Smart Configuration:**
- **Auto-detection**: Detects project type (go.mod → go variant, etc.)
- **Persistent preferences**: `.claude-reactor` file saves variant choice
- **Danger mode persistence**: Remembers `--danger` flag preference per project

## Current Project Structure

```
claude-reactor/
├── claude-reactor              # Main script - intelligent container management
├── Dockerfile                  # Multi-stage container definitions
├── Makefile                   # Professional build automation (25+ targets)
├── CLAUDE.md                  # This file - project guidance
├── WORKFLOW.md                # Tool responsibilities and usage patterns
├── ROADMAP.md                 # Future enhancements and prioritization
├── tests/                     # Comprehensive test suite
│   ├── test-runner.sh         # Main test orchestrator
│   ├── demo.sh                # Interactive feature demonstration
│   ├── unit/                  # Unit tests for all core functionality
│   │   └── test-functions.sh
│   ├── integration/           # Docker integration testing
│   │   └── test-variants.sh
│   └── README.md              # Test documentation
└── .claude-reactor            # Auto-generated project configuration
```

## Authentication Methods

The project supports two authentication approaches:
- **API Key**: Via environment file (`~/.env` with `ANTHROPIC_API_KEY`)
- **Interactive UI**: Direct login through Claude CLI (requires removing `runArgs` from devcontainer.json)

## Development Workflows

### **Primary Development Workflow (Recommended)**
```bash
# Smart container management - auto-detects project type
./claude-reactor                    # Launch Claude CLI directly (uses saved config)
./claude-reactor --variant go       # Set specific variant and save preference
./claude-reactor --shell            # Launch bash shell instead of Claude CLI
./claude-reactor --danger           # Launch Claude CLI with --dangerously-skip-permissions
./claude-reactor --show-config      # Check current configuration
./claude-reactor --list-variants    # See all available options
```

### **Build and Test Automation**
```bash
# Professional build system
make help                          # Show all available targets with examples
make build-all                     # Build core container variants (base, go, full)
make test                          # Run complete test suite (unit + integration)
make demo                          # Interactive feature demonstration
make run-go                        # Quick container startup (delegates to claude-reactor)
make clean-all                     # Complete cleanup
```

### **Typical Development Session**
```bash
# 1. Navigate to project directory
cd my-go-project

# 2. Start development container (auto-detects Go, saves preference)
make run                           # or ./claude-reactor

# 3. Work in container with full Go toolchain
# 4. Run tests and validation
make test-unit                     # Quick validation (5 seconds)

# 5. Clean up when done
make clean-containers              # or ./claude-reactor --clean
```

### **Advanced Usage**
```bash
# Force specific configurations
./claude-reactor --variant cloud --danger    # Cloud tools + skip permissions
./claude-reactor --rebuild                   # Force image rebuild

# Manual Docker control (rarely needed)
docker build --target go -t claude-reactor-go .
docker run -d --name claude-agent-go -v "$(pwd)":/app claude-reactor-go
```

## Configuration Files

### `.claude-reactor` (Project-specific settings)
```bash
variant=go
danger=true
```

This file is automatically created when you use `--variant` or `--danger` flags and stores your preferences per project directory.

## Development Philosophy & Best Practices

### **Collaborative Development Approach**
This project demonstrates effective Claude-Human collaboration patterns:
- **Iterative Enhancement**: Build core functionality first, then systematically add advanced features
- **Test-Driven Development**: Comprehensive test suite (unit + integration + demo) validates all functionality
- **Professional Standards**: Makefile automation, proper documentation, and production-ready workflows
- **Future-Oriented**: ROADMAP.md tracks enhancements prioritized by real-world value

### **Technical Excellence Principles**
- **Zero-Configuration Experience**: Auto-detection eliminates manual setup friction
- **Smart Persistence**: System learns and remembers user preferences per project
- **Modular Architecture**: Choose only the tools needed, avoiding bloat
- **Professional Automation**: Makefile + script integration supports both development and CI/CD
- **Comprehensive Testing**: All functionality validated through automated tests

### **Usage Guidelines for Claude Code**

**When working on this project:**
1. **Run tests first**: `make test-unit` (5 seconds) validates core functionality
2. **Use the demo**: `make demo-quick` showcases all features interactively
3. **Follow the workflow**: Reference WORKFLOW.md for tool responsibilities
4. **Check the roadmap**: ROADMAP.md contains future enhancements prioritized by value
5. **Maintain quality**: All changes should include appropriate tests

**For new features:**
1. **Test existing functionality** to understand current capabilities
2. **Reference ROADMAP.md** to align with planned enhancements
3. **Follow established patterns** from existing code structure
4. **Add comprehensive tests** for new functionality
5. **Update documentation** to reflect changes

**Architecture Notes:**
- **ARM64-optimized**: Primary target is M1 Macs (but supports x86_64)
- **Security-first**: Non-root `claude` user with minimal necessary privileges
- **Host Integration**: Mounts Claude config, git settings, and Kubernetes config
- **Complete Isolation**: Full separation from host while maintaining development workflow
- **Container Variants**: 5 specialized environments from minimal (base) to comprehensive (cloud/k8s)

## Testing and Validation

**Quick Validation**: `make test-unit` - Validates all core logic in ~5 seconds  
**Full Testing**: `make test` - Complete test suite including Docker integration  
**Interactive Demo**: `make demo` - Guided tour of all features  
**Development Setup**: `make dev-setup` - Prepare environment for contributions

**Test Coverage**: Unit tests for configuration management, auto-detection, variant validation, and persistence logic. Integration tests for Docker builds, container lifecycle, and tool availability.

## Project Maturity & Status

**Current Status**: Production-ready for personal and team development workflows  
**Version**: 2.0 (Major rewrite with modular architecture)  
**Last Updated**: August 2025  
**Maintenance**: Actively maintained with roadmap-driven enhancements  

**Key Achievements**:
- ✅ **Zero-friction setup**: Auto-detection eliminates configuration overhead
- ✅ **Language ecosystem support**: Go, Rust, Java, Python, Node.js, cloud tools
- ✅ **Professional automation**: 25+ Makefile targets for all workflows
- ✅ **Comprehensive testing**: Unit, integration, and demo validation
- ✅ **Smart persistence**: Remembers preferences without manual configuration
- ✅ **Production architecture**: Multi-stage builds, security best practices, efficient resource usage

**Ready for**: Personal projects, team development, educational use, and foundation for enterprise development workflows.