# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Claude-Reactor** is a professional, modular Docker containerization system for Claude CLI development workflows. It transforms the basic Claude CLI into a comprehensive development environment with intelligent automation, multi-language support, and production-ready tooling.

**Primary Use Case**: Personal and team development workflows that require isolated, reproducible development environments with smart tooling automation.

**Key Value Propositions**:
- **Zero Configuration**: Auto-detects project type and sets up appropriate development environment
- **Registry Integration**: Automatically pulls pre-built images from GitHub Container Registry for instant startup
- **Language Agnostic**: Supports Go, Rust, Java, Python, Node.js, and cloud development workflows  
- **Professional Automation**: Makefile + script integration for both development and CI/CD
- **Persistent Intelligence**: Remembers your preferences and configurations per project
- **Comprehensive Testing**: Unit tests, integration tests, and interactive demonstrations

## Project Architecture

This project creates a modular Docker containerization system for Claude CLI with multiple specialized variants:

### **Container Variants:**
- **base**: Node.js, Python (with pip + uv), basic development tools (smallest, ~500MB)
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
- **Registry-first**: Attempts to pull from ghcr.io, falls back to local builds
- **Development mode**: `--dev` flag forces local builds for development

## Current Project Structure

```
claude-reactor/
├── claude-reactor              # Main script - intelligent container management
├── Dockerfile                  # Multi-stage container definitions
├── Makefile                   # Professional build automation (30+ targets)
├── VERSION                    # Semantic version file (v0.1.0)
├── CLAUDE.md                  # This file - project guidance
├── WORKFLOW.md                # Tool responsibilities and usage patterns
├── ROADMAP.md                 # Future enhancements and prioritization
├── .github/                   # CI/CD automation
│   └── workflows/
│       └── build-and-push.yml # Multi-architecture builds and registry pushes
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

The project supports multiple authentication approaches with account isolation:

### **Account-Specific Authentication (Recommended)**
Each Claude account gets its own isolated configuration:
- **Default account**: When no account is specified in `.claude-reactor`
  - Uses: `~/.claude-reactor/.default-claude.json`
  - Container: `claude-reactor-*-default`
  - API key: `.claude-reactor-env` (if used)

- **Named accounts**: When `account=name` is specified in `.claude-reactor`
  - Uses: `~/.claude-reactor/.name-claude.json`
  - Container: `claude-reactor-*-name`
  - API key: `.claude-reactor-name-env` (if used)

### **Authentication Methods**
- **OAuth (Recommended)**: Uses existing Claude CLI authentication from config files
- **API Key**: Via project-specific environment files (`claude-reactor --apikey YOUR_KEY`)
- **Interactive UI**: Direct login through Claude CLI (use `--interactive-login`)

## Development Workflows

### **Primary Development Workflow (Recommended)**
```bash
# Smart container management - auto-detects project type and pulls from registry
./claude-reactor                    # Launch Claude CLI directly (registry-first, local fallback)
./claude-reactor --variant go       # Set specific variant and save preference
./claude-reactor --shell            # Launch bash shell instead of Claude CLI
./claude-reactor --danger           # Launch Claude CLI with --dangerously-skip-permissions
./claude-reactor --show-config      # Check current configuration
./claude-reactor --list-variants    # See all available options

# Registry control
./claude-reactor --dev              # Force local build (disable registry)
./claude-reactor --pull-latest      # Force pull latest from registry
./claude-reactor --registry-off     # Disable registry completely
```

### **Multi-Account Workflow**
```bash
# Default account usage (no account specified)
./claude-reactor                    # Uses ~/.claude-reactor/.default-claude.json

# Work account usage
./claude-reactor --account work     # Sets account=work, uses ~/.claude-reactor/.work-claude.json
./claude-reactor --show-config      # Shows: Account: work

# Personal account usage  
./claude-reactor --account personal # Sets account=personal, uses ~/.claude-reactor/.personal-claude.json

# Account-specific API key (optional)
./claude-reactor --account work --apikey sk-ant-xxx  # Creates .claude-reactor-work-env

# Switch between accounts in different projects
cd ~/work-project && ./claude-reactor  # Uses work account if saved in .claude-reactor
cd ~/personal-project && ./claude-reactor  # Uses personal account if saved in .claude-reactor
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

# Registry management
make push-all                      # Build and push core variants to registry
make pull-all                      # Pull core variants from registry
make registry-login                # Log in to container registry
```

### **Typical Development Session**
```bash
# 1. Navigate to project directory
cd my-go-project

# 2. Start development container (auto-detects Go, pulls from registry if available)
make run                           # or ./claude-reactor

# 3. Work in container with full Go toolchain
# 4. Run tests and validation
make test-unit                     # Quick validation (5 seconds)

# 5. Clean up when done
make clean-containers              # or ./claude-reactor --clean
```

### **Registry-Enabled Workflows**
```bash
# Standard usage (registry-first, local fallback)
./claude-reactor                   # Pulls from ghcr.io/dyluth/claude-reactor automatically

# Development workflows
./claude-reactor --dev             # Force local build for development/testing
./claude-reactor --pull-latest     # Ensure you have the newest image

# CI/CD and maintenance
make pull-all                      # Pull all variants from registry
make push-all                      # Build and push to registry (requires auth)
make registry-login                # Login to GitHub Container Registry
```

### **Advanced Usage**
```bash
# Force specific configurations
./claude-reactor --variant cloud --danger    # Cloud tools + skip permissions
./claude-reactor --rebuild                   # Force image rebuild

# Registry management
./claude-reactor --dev                       # Force local build (disable registry)
./claude-reactor --registry-off              # Disable registry completely
./claude-reactor --pull-latest               # Force pull latest from registry

# Manual Docker control (rarely needed)
docker build --target go -t claude-reactor-go .
docker run -d --name claude-agent-go -v "$(pwd)":/app claude-reactor-go
```

## Configuration Files

### `.claude-reactor` (Project-specific settings)
```bash
variant=go
danger=true
account=work
```

This file is automatically created when you use `--variant`, `--danger`, or `--account` flags and stores your preferences per project directory.

**Configuration Options:**
- `variant=` - Container variant (base, go, full, cloud, k8s)
- `danger=` - Enable danger mode (true/false)
- `account=` - Claude account to use (creates isolated authentication)

### **Container Registry Configuration**

Claude-Reactor automatically pulls pre-built images from GitHub Container Registry for faster startup times.

**Registry Settings:**
```bash
# Environment variables (optional)
export CLAUDE_REACTOR_REGISTRY="ghcr.io/dyluth/claude-reactor"  # Default registry
export CLAUDE_REACTOR_USE_REGISTRY=true                        # Enable registry (default: true)
export CLAUDE_REACTOR_TAG=latest                               # Image tag (default: latest)
```

**Registry Behavior:**
- **Default**: Attempts to pull from `ghcr.io/dyluth/claude-reactor` first
- **Fallback**: Builds locally if registry pull fails
- **Development**: Use `--dev` flag to force local builds
- **Public Images**: No authentication required for pulls
- **Multi-arch**: Supports both ARM64 (M1 Macs) and AMD64 architectures
- **Versioning**: Supports `latest`, `v0.1.0`, and `dev` tags
- **CI/CD Integration**: Automatic builds on git push and tags

### **Account-Specific Authentication Files**
The system creates separate Claude configuration files for each account:

```bash
~/.claude-reactor/
├── .default-claude.json       # Default account (when account= is not set)
├── .work-claude.json          # Work account (when account=work)
├── .personal-claude.json      # Personal account (when account=personal)
└── .unitary-claude.json       # Unitary account (when account=unitary)
```

**Automatic Setup:**
- First time using an account: Config is auto-copied from `~/.claude.json`
- Each account gets isolated OAuth tokens and project settings
- Containers are named with account: `claude-reactor-*-work`, `claude-reactor-*-personal`

## CI/CD and Release Management

### **GitHub Actions Integration**
Claude-Reactor includes comprehensive CI/CD automation through GitHub Actions:

**Automatic Triggers:**
- **Push to main**: Builds and pushes `latest` images
- **Create tag `v*`**: Builds and pushes version-tagged images (e.g., `v0.1.0`)
- **Pull requests**: Builds and tests without pushing
- **Manual dispatch**: Supports `dev` tag builds

**Multi-Architecture Builds:**
- Builds for both `linux/amd64` and `linux/arm64`
- Uses Docker Buildx for efficient cross-platform compilation
- Leverages GitHub Actions build cache for speed

**Security and Quality:**
- Trivy security scanning on core variants
- Integration tests with registry images
- SARIF upload to GitHub Security tab

### **Release Workflow**
```bash
# Create and push a new release
echo "v0.2.0" > VERSION
git add VERSION
git commit -m "Release v0.2.0"
git tag v0.2.0
git push origin main
git push origin v0.2.0

# GitHub Actions will automatically:
# 1. Build all variants for both architectures
# 2. Push to ghcr.io/dyluth/claude-reactor-*:v0.2.0
# 3. Push to ghcr.io/dyluth/claude-reactor-*:latest
# 4. Run security scans
# 5. Create GitHub release (if configured)
```

### **Manual Registry Management**
```bash
# Login to GitHub Container Registry
echo $GITHUB_TOKEN | docker login ghcr.io -u dyluth --password-stdin

# Build and push manually (if needed)
make push-all                      # Core variants (base, go, full)
make push-extended                 # All variants including cloud/k8s

# Pull specific versions
CLAUDE_REACTOR_TAG=v0.1.0 ./claude-reactor --pull-latest
```

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
- ✅ **Registry integration**: Automatic pulls from GitHub Container Registry with local fallback
- ✅ **Language ecosystem support**: Go, Rust, Java, Python, Node.js, cloud tools
- ✅ **Professional automation**: 30+ Makefile targets including registry management
- ✅ **CI/CD pipeline**: Multi-architecture builds, security scanning, automated releases
- ✅ **Comprehensive testing**: Unit, integration, and demo validation
- ✅ **Smart persistence**: Remembers preferences without manual configuration
- ✅ **Production architecture**: Multi-stage builds, security best practices, efficient resource usage

**Ready for**: Personal projects, team development, educational use, enterprise development workflows, and public distribution via container registry.