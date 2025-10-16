# Claude-Reactor

[![Build Status](https://github.com/dyluth/claude-reactor/workflows/Build%20and%20Push%20Docker%20Images/badge.svg)](https://github.com/dyluth/claude-reactor/actions)
[![codecov](https://codecov.io/gh/dyluth/claude-reactor/branch/main/graph/badge.svg)](https://codecov.io/gh/dyluth/claude-reactor)
[![Go Report Card](https://goreportcard.com/badge/github.com/dyluth/claude-reactor)](https://goreportcard.com/report/github.com/dyluth/claude-reactor)

**A secure, professional Docker containerization system for Claude CLI that prioritizes account isolation, zero-configuration experience, and enterprise-grade development workflows.**

## ✨ Key Features

- **🔒 Complete Account Isolation**: Separate credentials, sessions, and containers between Claude accounts
- **⚡ Zero Configuration**: Auto-detects project type and sets up appropriate development environment  
- **🎯 Smart Container Management**: Intelligent reuse vs recreation based on command arguments
- **🛠️ Language Agnostic**: Go, Rust, Java, Python, Node.js, and cloud development support
- **💼 VS Code Integration**: Automatic dev container generation with project-specific extensions
- **📝 Project Templates**: Interactive scaffolding for common project types
- **🎨 Custom Images**: Support for any Docker image with compatibility validation
- **📦 Registry Integration**: Automatic image pulling with local build fallback

## 🚀 Quick Start

### Installation

**Option 1: Build from Source**
```bash
git clone https://github.com/dyluth/claude-reactor.git
cd claude-reactor
make build
```

**Option 2: Download Binary** (when releases are available)
```bash
# Will be available from GitHub Releases
curl -fsSL https://raw.githubusercontent.com/dyluth/claude-reactor/main/install.sh | bash
```

### Prerequisites
- **Docker**: Required for container functionality ([install guide](https://docs.docker.com/get-docker/))
- **Platform**: Linux or macOS (x86_64 or ARM64)

### Basic Usage

```bash
# Start Claude CLI (auto-detects project type)
./claude-reactor run

# Use specific container variants
./claude-reactor run --image go           # Go development tools
./claude-reactor run --image full         # Multi-language environment
./claude-reactor run --image cloud        # Cloud development tools
./claude-reactor run --image k8s          # Kubernetes tools

# Account isolation for teams
./claude-reactor run --account work       # Work account
./claude-reactor run --account personal   # Personal account
```

## 🏗️ Container Variants

Built-in variants optimized for different development needs:

| Variant | Size | Tools | Use Case |
|---------|------|-------|----------|
| `base` | ~500MB | Node.js, Python, Git | Lightweight development |
| `go` | ~800MB | Base + Go toolchain | Go development |
| `full` | ~1.2GB | Go + Rust, Java, Databases | Multi-language projects |
| `cloud` | ~1.5GB | Full + AWS/GCP/Azure CLIs | Cloud development |
| `k8s` | ~1.4GB | Full + Enhanced Kubernetes tools | Kubernetes workflows |

## 👥 Account Isolation

Complete separation between different Claude accounts and projects:

```bash
# Default account (uses your username from $USER)
./claude-reactor run

# Work account (completely isolated)
./claude-reactor run --account work

# Check current configuration  
./claude-reactor config show

# View all accounts and projects
./claude-reactor list
```

**Account Structure:**
- **Credentials**: `~/.claude-reactor/.{account}-claude.json`
- **Sessions**: `~/.claude-reactor/{account}/{project-hash}/`
- **Containers**: `claude-reactor-{variant}-{arch}-{project-hash}-{account}`

**Benefits:**
- ✅ Persistent authentication across container restarts
- ✅ Separate conversation history per project/account
- ✅ Team collaboration without credential conflicts
- ✅ Project-specific configuration and preferences

## 🎨 VS Code Integration

Automatic dev container generation with intelligent project detection:

```bash
# Generate .devcontainer for current project
./claude-reactor devcontainer generate

# Force specific variant
./claude-reactor devcontainer generate --image go

# Show project detection details
./claude-reactor devcontainer info
```

**Features:**
- **Automatic Project Detection**: Detects Go, Rust, Node.js, Python, Java with confidence scoring
- **Extension Installation**: Language-specific VS Code extensions automatically included
- **Professional Setup**: Complete IDE environment ready in 30 seconds
- **Team Consistency**: Identical development environments across all machines

## 📝 Project Templates

Interactive project scaffolding with built-in templates:

```bash
# Interactive project creation wizard
./claude-reactor template init

# Create from specific template
./claude-reactor template new go-api my-service

# List available templates
./claude-reactor template list
```

**Built-in Templates:**
- **Go**: REST API (Gorilla Mux), CLI application (Cobra)
- **Rust**: CLI application (clap), library with testing
- **Node.js**: Express + TypeScript API, React + TypeScript app
- **Python**: FastAPI service, Click CLI application
- **Java**: Spring Boot REST API

## 🔧 Advanced Usage

### Custom Docker Images

Use any Linux-based Docker image with validation:

```bash
# Python scientific computing
./claude-reactor run --image jupyter/scipy-notebook

# Custom enterprise image
./claude-reactor run --image myregistry.com/dev-env:latest

# Image validation provides compatibility feedback
# ✅ Linux platform detected
# ✅ Claude CLI installation verified  
# ⚠️ Missing recommended tools: git, curl
```

### Registry Management

Automatic image pulling with local build fallback:

```bash
# Standard usage (pulls from registry automatically)
./claude-reactor run

# Force local build for development
./claude-reactor run --dev

# Force pull latest from registry
./claude-reactor run --pull-latest

# Disable registry completely
./claude-reactor run --registry-off
```

### Container Management

```bash
# List all accounts, projects, and containers
./claude-reactor list

# Clean up containers and optionally sessions
./claude-reactor clean
./claude-reactor clean --sessions    # Also remove conversation history
./claude-reactor clean --auth        # Also remove authentication
./claude-reactor clean --all         # Complete cleanup

# Configuration management
./claude-reactor config show         # Current configuration
./claude-reactor config show --verbose  # Detailed system info
```

## 🛠️ Development Workflow

### For Contributors

```bash
# Setup development environment
make dev-setup

# Run tests (recommended before any changes)
make test-unit                       # Quick validation (5 seconds)
make test                           # Complete test suite

# Build and test
make build                          # Build Go binary
make build-all                      # Build all container variants

# Interactive demo
make demo                           # Guided tour of features
```

### Build Automation

Professional Makefile with 25+ targets:

```bash
make help                           # Show all available targets
make ci-full                        # Complete CI pipeline
make clean-all                      # Complete cleanup
make registry-login                 # Login to container registry
make push-all                       # Build and push to registry
```

## 📚 Documentation

- **[CLAUDE.md](CLAUDE.md)** - Instructions for AI collaborators
- **[PROJECT_CONTEXT.md](PROJECT_CONTEXT.md)** - Complete project context and architecture
- **[docs/CUSTOM-IMAGES.md](docs/CUSTOM-IMAGES.md)** - Custom Docker image usage guide
- **[docs/CONTAINER_STRATEGIES.md](docs/CONTAINER_STRATEGIES.md)** - Container architecture details
- **[tests/README.md](tests/README.md)** - Testing documentation and examples

## 🎯 Project Goals

Claude-Reactor solves fundamental problems in containerized Claude CLI usage:

1. **Security**: Complete account isolation with proper credential management
2. **Simplicity**: Zero-configuration setup with intelligent auto-detection
3. **Flexibility**: Support for custom images while maintaining compatibility
4. **Professional Quality**: Enterprise-grade automation and development workflows
5. **Team Collaboration**: Consistent environments with proper session isolation

## 🏆 Project Status

**Current State**: Production-ready with comprehensive feature set

**Key Achievements:**
- ✅ Complete account isolation system
- ✅ VS Code dev container integration  
- ✅ Project template scaffolding
- ✅ Multi-architecture container registry
- ✅ Comprehensive test suite (35+ % coverage)
- ✅ Professional CI/CD pipeline
- ✅ Custom Docker image support with validation

## 🤝 Contributing

1. **Read the project context**: [PROJECT_CONTEXT.md](PROJECT_CONTEXT.md)
2. **Understand the workflow**: [CLAUDE.md](CLAUDE.md) 
3. **Run tests first**: `make test-unit` validates core functionality
4. **Follow established patterns**: Interface-based design with comprehensive testing
5. **Check future plans**: [ai-prompts/](ai-prompts/) directory for implementation specs

## 📄 License

[License information to be determined]

---

*Claude-Reactor: Professional Docker containerization for Claude CLI development workflows*