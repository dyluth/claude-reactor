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
# Smart container management - auto-detects project type
./claude-reactor                    # Launch Claude CLI directly (uses saved config)
./claude-reactor --image go         # Set specific container image and save preference
./claude-reactor --shell            # Launch bash shell instead of Claude CLI
./claude-reactor --danger           # Launch Claude CLI with --dangerously-skip-permissions
./claude-reactor --show-config      # Check current configuration
./claude-reactor --list-variants    # See all available options
```

**Container Images:**
- **Built-in variants**: `base`, `go`, `full`, `cloud`, `k8s` (auto-built and validated)
- **Custom Docker images**: Any Docker Hub or registry image (e.g. `ubuntu:22.04`, `node:18-alpine`)
- **Auto-detection**: Automatically selects best variant based on project files

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
./claude-reactor --image cloud --danger    # Cloud tools + skip permissions
./claude-reactor --rebuild                 # Force image rebuild

# Manual Docker control (rarely needed)
docker build --target go -t claude-reactor-go .
docker run -d --name claude-agent-go -v "$(pwd)":/app claude-reactor-go
```

### **Custom Docker Images**
Use any Docker image with claude-reactor for specialized development environments beyond the built-in variants:

```bash
# Quick start with custom images
./claude-reactor --image ubuntu:22.04      # Ubuntu-based development
./claude-reactor --image node:18-alpine    # Node.js development
./claude-reactor --image python:3.11       # Python development

# Advanced custom image usage
./claude-reactor --image myregistry/custom-image:latest --account work
./claude-reactor --image nvidia/cuda:11.8-devel  # GPU development
./claude-reactor --image ghcr.io/devcontainers/python:3.11  # VS Code Dev Container
```

**Requirements for Custom Images:**
- **Platform**: Must be Linux-based (`linux/amd64` or `linux/arm64`)
- **Claude CLI**: Must have Claude CLI installed and accessible via `claude --version`
- **Base system**: Should have basic shell utilities (`sh`, `bash`, or compatible)

**Recommended Tools (High Priority):**
- `git` - Version control operations
- `curl` - Network operations and downloads
- `make` - Build automation
- `nano` or `vim` - Text editing

**Optional Tools (Enhance Experience):**
- `wget` - Alternative download tool
- `jq` - JSON processing
- `zip`/`unzip` - Archive handling
- `ssh` - Remote access capabilities
- `ripgrep` (`rg`) - Fast text search
- `yq` - YAML processing

**Validation Process:**
Claude-reactor automatically validates custom images before use:

1. **Platform Check**: Verifies Linux compatibility
2. **Claude CLI Detection**: Tests `claude --version` command
3. **Package Analysis**: Scans for 10 recommended development tools
4. **Smart Warnings**: Shows missing high-priority tools (once per session)
5. **Result Caching**: Stores validation results for 30+ days (immutable image digests)

**Common Custom Image Patterns:**

```bash
# Language-specific development
./claude-reactor --image rust:1.75          # Latest Rust toolchain
./claude-reactor --image golang:1.21        # Go development
./claude-reactor --image openjdk:21         # Java development
./claude-reactor --image php:8.3-cli        # PHP development

# Specialized environments
./claude-reactor --image tensorflow/tensorflow:latest-gpu  # ML/AI with GPU
./claude-reactor --image cypress/browsers:latest          # Browser testing
./claude-reactor --image hashicorp/terraform:latest       # Infrastructure as code

# Registry-specific images
./claude-reactor --image ghcr.io/user/project:latest      # GitHub Container Registry
./claude-reactor --image us-central1-docker.pkg.dev/project/repo/image:tag  # Google Artifact Registry
./claude-reactor --image registry.gitlab.com/group/project:latest           # GitLab Container Registry
```

**Custom Image Best Practices:**

1. **Start with minimal base**: Use Alpine Linux or distroless images when possible
2. **Install Claude CLI**: Add to your Dockerfile: `curl -fsSL https://claude.ai/install.sh | sh`
3. **Include development essentials**: `git`, `curl`, `make`, `nano`
4. **Test compatibility**: Run `claude-reactor --image your-image:tag --shell` first
5. **Version pinning**: Use specific tags rather than `latest` for reproducibility

### **Custom Image Examples**

#### **Data Science & Machine Learning**
```bash
# Python data science stack
./claude-reactor --image jupyter/scipy-notebook:latest
./claude-reactor --image tensorflow/tensorflow:latest-gpu
./claude-reactor --image pytorch/pytorch:2.0.1-cuda11.7-cudnn8-devel

# R development
./claude-reactor --image rocker/tidyverse:latest
./claude-reactor --image rocker/rstudio:latest
```

#### **Web Development**
```bash
# Modern JavaScript/TypeScript
./claude-reactor --image node:18-alpine          # Minimal Node.js
./claude-reactor --image node:20-bullseye        # Full Debian base
./claude-reactor --image denoland/deno:alpine    # Deno runtime

# PHP development
./claude-reactor --image php:8.3-cli-alpine      # CLI-focused
./claude-reactor --image php:8.3-fpm-bullseye    # Full web stack
```

#### **Systems Programming**
```bash
# Rust development
./claude-reactor --image rust:1.75-alpine        # Latest Rust
./claude-reactor --image rust:1.75-slim-bullseye # Rust with more tools

# C/C++ development
./claude-reactor --image gcc:latest               # GCC compiler
./claude-reactor --image clang:17                 # LLVM/Clang
```

#### **Cloud & DevOps**
```bash
# AWS development
./claude-reactor --image amazon/aws-cli:latest
./claude-reactor --image amazon/aws-sam-cli-build-image-python3.11

# Google Cloud
./claude-reactor --image gcr.io/google.com/cloudsdktool/cloud-sdk:alpine
./claude-reactor --image gcr.io/google.com/cloudsdktool/cloud-sdk:latest

# Azure development  
./claude-reactor --image mcr.microsoft.com/azure-cli:latest
./claude-reactor --image mcr.microsoft.com/dotnet/sdk:8.0

# Kubernetes tooling
./claude-reactor --image bitnami/kubectl:latest
./claude-reactor --image alpine/helm:latest
```

#### **Database Development**
```bash
# Database clients and tools
./claude-reactor --image postgres:16-alpine      # PostgreSQL client
./claude-reactor --image mysql:8.0               # MySQL client
./claude-reactor --image mongo:7                 # MongoDB tools
./claude-reactor --image redis:7-alpine          # Redis tools
```

#### **Scientific Computing**
```bash
# Jupyter environments
./claude-reactor --image jupyter/datascience-notebook:latest
./claude-reactor --image jupyter/tensorflow-notebook:latest
./claude-reactor --image quay.io/jupyter/scipy-notebook:latest

# Specialized scientific tools
./claude-reactor --image continuumio/anaconda3:latest
./claude-reactor --image condaforge/mambaforge:latest
```

#### **Custom Registry Images**
```bash
# GitHub Container Registry
./claude-reactor --image ghcr.io/devcontainers/python:3.11-bullseye
./claude-reactor --image ghcr.io/microsoft/vscode-dev-containers/typescript-node:0-18-bullseye

# Custom enterprise registries
./claude-reactor --image registry.company.com/dev/nodejs:18
./claude-reactor --image us-central1-docker.pkg.dev/project/repo/custom-dev:latest

# GitLab Container Registry
./claude-reactor --image registry.gitlab.com/group/project/dev-env:latest
```

#### **Creating Custom Images**

**Simple Claude-Compatible Image:**
```dockerfile
FROM ubuntu:22.04

# Install Claude CLI
RUN curl -fsSL https://claude.ai/install.sh | sh

# Install essential development tools
RUN apt-get update && apt-get install -y \
    git \
    curl \
    make \
    nano \
    wget \
    jq \
    zip \
    unzip \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN useradd -m -s /bin/bash claude
USER claude
WORKDIR /home/claude

# Set up shell environment
RUN echo 'alias ll="ls -la"' >> ~/.bashrc
```

**Language-Specific Custom Image:**
```dockerfile
FROM golang:1.21-bullseye

# Install Claude CLI
RUN curl -fsSL https://claude.ai/install.sh | sh

# Install additional Go tools
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && \
    go install github.com/swaggo/swag/cmd/swag@latest && \
    go install github.com/air-verse/air@latest

# Install development essentials
RUN apt-get update && apt-get install -y \
    git \
    curl \
    make \
    nano \
    ripgrep \
    && rm -rf /var/lib/apt/lists/*

# Create claude user
RUN useradd -m -s /bin/bash claude && \
    usermod -aG sudo claude
USER claude
WORKDIR /workspace
```

**Build and test custom image:**
```bash
# Build the image
docker build -t my-custom-dev:latest .

# Test compatibility
claude-reactor debug image my-custom-dev:latest

# Use the custom image
claude-reactor --image my-custom-dev:latest
```

## Configuration Files

### `.claude-reactor` (Project-specific settings)
```bash
# Built-in variant or custom Docker image
variant=go                                  # Built-in variants: base, go, full, cloud, k8s
variant=ubuntu:22.04                        # Custom Docker image
variant=ghcr.io/user/project:latest         # Registry image

# Additional configuration
danger=true                                 # Enable danger mode
account=work                               # Account isolation
```

This file is automatically created when you use `--image`, `--danger`, or `--account` flags and stores your preferences per project directory.

**Configuration Options:**
- `variant=` - Container image (built-in variants: `base`, `go`, `full`, `cloud`, `k8s` or custom Docker image)
- `danger=` - Enable danger mode (true/false)  
- `account=` - Claude account to use (creates isolated authentication)

**Note**: The config file uses `variant=` for backward compatibility, but the CLI uses `--image` flag.

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

## Troubleshooting Custom Images

### **Common Issues and Solutions**

#### **1. "Claude CLI not found" Error**
**Problem**: Custom image fails validation with "claude --version command failed"

**Solutions**:
```bash
# Check if Claude CLI is installed
claude-reactor debug image your-image:tag --shell
# Inside container:
which claude  # Should show Claude CLI path
claude --version  # Should show version info

# If Claude CLI is missing, add to your Dockerfile:
RUN curl -fsSL https://claude.ai/install.sh | sh
# Or manually install:
RUN wget https://github.com/anthropics/claude-cli/releases/latest/download/claude-linux-amd64 -O /usr/local/bin/claude && \
    chmod +x /usr/local/bin/claude
```

#### **2. Platform Architecture Mismatch**
**Problem**: "Unsupported platform" or slow performance on M1 Macs

**Solutions**:
```bash
# Check image architecture
docker inspect your-image:tag | grep Architecture

# For M1 Macs, prefer ARM64 images:
./claude-reactor --image node:18-alpine     # Multi-arch (preferred)
./claude-reactor --image --platform linux/arm64 node:18-alpine

# Force AMD64 if needed (slower on M1):
./claude-reactor --image --platform linux/amd64 your-image:tag
```

#### **3. Missing Development Tools**
**Problem**: Warnings about missing high-priority tools (git, curl, make, nano)

**Solutions**:
```bash
# Test what tools are available
claude-reactor debug image your-image:tag

# Add essential tools to your Dockerfile:
RUN apt-get update && apt-get install -y \
    git \
    curl \
    make \
    nano \
    wget \
    && rm -rf /var/lib/apt/lists/*

# For Alpine-based images:
RUN apk add --no-cache \
    git \
    curl \
    make \
    nano \
    wget
```

#### **4. Permission and User Issues**
**Problem**: Container runs as root or has permission issues

**Solutions**:
```dockerfile
# Create non-root user in your Dockerfile
RUN useradd -m -s /bin/bash claude
USER claude
WORKDIR /home/claude

# Or use existing user
USER node  # For Node.js images
USER 1000  # Use numeric UID

# Fix permissions for mounted directories
RUN chown -R claude:claude /workspace
```

#### **5. Image Pull/Access Issues**
**Problem**: "Unable to pull image" or authentication errors

**Solutions**:
```bash
# Check if image exists
docker pull your-image:tag

# For private registries, login first:
docker login registry.company.com
echo "password" | docker login registry.company.com -u username --password-stdin

# For GitHub Container Registry:
echo "$GITHUB_TOKEN" | docker login ghcr.io -u USERNAME --password-stdin

# Test with public image first:
claude-reactor --image ubuntu:22.04
```

#### **6. Large Image Size / Slow Startup**
**Problem**: Image is very large or takes long to pull

**Solutions**:
```bash
# Check image size
docker images | grep your-image

# Use smaller base images:
FROM alpine:latest        # ~5MB
FROM ubuntu:22.04         # ~80MB  
FROM debian:bullseye-slim # ~80MB

# Multi-stage builds to reduce size:
FROM golang:1.21 AS builder
# ... build steps ...

FROM alpine:latest
COPY --from=builder /app/binary /usr/local/bin/
```

#### **7. Container Startup Failures**
**Problem**: Container exits immediately or fails to start

**Solutions**:
```bash
# Check container logs
docker logs container-name

# Test container manually
docker run -it your-image:tag /bin/bash
# Or for minimal images:
docker run -it your-image:tag /bin/sh

# Ensure proper entry point
ENTRYPOINT ["/bin/bash"]
# Or keep container running:
CMD ["tail", "-f", "/dev/null"]
```

#### **8. Package Manager Issues**
**Problem**: apt, apk, or other package managers fail

**Solutions**:
```dockerfile
# Update package lists first
RUN apt-get update && apt-get install -y package-name
# Clean up to reduce size
RUN apt-get update && apt-get install -y \
    package1 \
    package2 \
    && rm -rf /var/lib/apt/lists/*

# For Alpine:
RUN apk update && apk add --no-cache package-name

# For CentOS/RHEL:
RUN yum update -y && yum install -y package-name
```

### **Validation Cache Issues**

#### **Clear Validation Cache**
```bash
# Clear all cached validation results
claude-reactor debug cache clear

# Show cache info
claude-reactor debug cache info

# Manual cache cleanup
rm -rf ~/.claude-reactor/image-cache/
```

#### **Force Re-validation**
```bash
# Clear session warnings to see them again
claude-reactor debug cache clear

# Pull latest image version
docker pull your-image:tag

# Test with fresh validation
claude-reactor debug image your-image:tag
```

### **Performance Optimization**

#### **Image Selection Tips**
```bash
# Prefer multi-architecture images
./claude-reactor --image node:18-alpine    # Multi-arch
./claude-reactor --image python:3.11-slim  # Multi-arch

# Language-specific optimized images
./claude-reactor --image golang:1.21-alpine     # Go development
./claude-reactor --image rust:1.75-slim-bullseye # Rust development
./claude-reactor --image openjdk:21-slim        # Java development
```

#### **Registry Selection**
```bash
# Use geographically closer registries
./claude-reactor --image gcr.io/image:tag      # Google Container Registry
./claude-reactor --image ghcr.io/user/image:tag # GitHub Container Registry

# Use cached layers when possible
./claude-reactor --image ubuntu:22.04  # Likely cached
./claude-reactor --image alpine:latest # Small and common
```

### **Getting Help**

#### **Debug Commands**
```bash
# System information
claude-reactor debug info

# Test image compatibility
claude-reactor debug image your-image:tag

# Check cache status
claude-reactor debug cache info

# Show detailed logs
claude-reactor --verbose --log-level debug run --image your-image:tag
```

#### **Community Resources**
- **GitHub Issues**: Report bugs and compatibility issues
- **Documentation**: Check WORKFLOW.md and ROADMAP.md for latest features
- **Examples**: See custom image examples above for working configurations

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