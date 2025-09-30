# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Claude-Reactor** is a simple, safe way to run Claude CLI in Docker containers with account isolation. It provides multiple pre-built container variants for different development needs while maintaining security and simplicity.

**Primary Use Case**: Secure, isolated Claude CLI execution with proper account separation and container-based development environments.

**Key Value Propositions**:
- **Complete Account Isolation**: Each Claude account gets separate credentials, sessions, and containers
- **Persistent Authentication**: No re-login required when restarting containers
- **Project-Specific Sessions**: Isolated conversation history per project/account combination
- **Smart Container Management**: Intelligent reuse vs recreation based on command arguments
- **Multiple Variants**: Pre-built images for different development stacks (base, go, full, cloud, k8s)
- **Security First**: Non-root execution and proper permission handling
- **Simple Configuration**: Minimal setup with sensible defaults and auto-detection
- **Comprehensive Management**: List and clean commands for complete visibility and control

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
- **Account Defaults**: Uses `$USER` environment variable as default account, fallback to "user"
- **Project Sessions**: Each project gets isolated session directory with conversation history
- **Smart Container Lifecycle**: Reuse containers when no args, recreate when args provided
- **Persistent preferences**: Configuration saved in project-specific session directories
- **Registry-first**: Attempts to pull from ghcr.io, falls back to local builds
- **Development mode**: `--dev` flag forces local builds for development

## Current Project Structure

```
claude-reactor/
├── cmd/                       # Application entrypoints
│   ├── claude-reactor/        # Main claude-reactor application
│   └── reactor-fabric/        # Distributed MCP orchestrator (Phase 0+)
├── internal/                  # Private application logic
│   ├── reactor/               # Claude-reactor specific implementation
│   └── fabric/                # Reactor-fabric orchestrator implementation
├── pkg/                       # Shared data structures and utilities
├── Dockerfile                 # Multi-stage container definitions
├── Makefile                   # Professional build automation (25+ targets)
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
├── ai-prompts/                # Implementation specifications
│   └── 6-distributed-mcp-orchestration-system.md
└── .claude-reactor            # Auto-generated project configuration
```

## Account Isolation and Authentication

The project provides complete account isolation with persistent authentication and project-specific sessions:

### **Account Structure**
Each Claude account gets completely isolated configuration and session data:

```
~/.claude-reactor/
├── {account}/                              # Account-specific session data
│   ├── {project-name}-{project-hash}/      # Project-specific sessions
│   │   ├── .claude-reactor                 # Project config
│   │   ├── projects/                       # Claude conversation history
│   │   ├── shell-snapshots/               # Shell session data
│   │   └── todos/                         # Project todos
│   └── another-project-{hash}/
├── .{account}-claude.json                  # Account-specific Claude config
├── .claude-reactor-{account}-env           # Account-specific API keys (optional)
└── .default-claude.json                   # Default account config
```

### **Default Account Logic**
- **Default account**: Uses `$USER` environment variable (e.g., "john", "alice")
- **Fallback**: Uses "user" if `$USER` is not available
- **Container naming**: `claude-reactor-{variant}-{arch}-{project-hash}-{account}`
- **Session isolation**: Each project/account combo gets separate conversation history

### **Authentication Methods**
- **OAuth (Recommended)**: Persistent authentication across container restarts
- **API Key**: Via account-specific environment files (`claude-reactor --apikey YOUR_KEY`)
- **Interactive Login**: Use `--interactive-login` for first-time setup

### **Authentication Persistence**
✅ **Fixed**: Authentication now persists across container restarts
- Account-specific Claude config files are properly mounted
- No re-login required when restarting containers for same account
- Each account maintains separate OAuth tokens and project settings

## Build Operations

**⚠️ IMPORTANT: Use the Makefile for all build and development operations**

Claude-reactor follows standard practice by using a Makefile for build automation instead of CLI commands. For all build, test, and development operations, use the comprehensive Makefile targets:

```bash
# Build operations
make help                    # Show all available targets
make build                   # Build the claude-reactor binary
make build-all              # Build all container variants
make test                   # Run complete test suite
make test-unit              # Run unit tests only
make clean                  # Clean build artifacts
make clean-all              # Complete cleanup

# Container management
make run                    # Quick container startup (delegates to claude-reactor)
make demo                   # Interactive feature demonstration

# Development workflow
make dev-setup              # Prepare environment for contributions
```

The Makefile provides professional build automation with 25+ targets covering all development needs. CLI commands focus solely on runtime container management, not build operations.

## Development Workflows

### **Smart Container Management (New)**
Claude-reactor now features intelligent container lifecycle management:

```bash
# Smart reuse - containers are automatically reused when no arguments passed
claude-reactor                      # Reuses existing container for this project/account
claude-reactor run                  # Same as above - reuses existing container

# Force recreation - any arguments force container recreation  
claude-reactor run --image go       # Recreates container with go variant
claude-reactor run --shell          # Recreates container, launches shell instead of Claude
claude-reactor run --danger         # Recreates container with danger mode
claude-reactor run --account work   # Recreates container for work account

# Container conflict resolution is automatic - no more "container name already in use" errors
```

### **Multi-Account Workflows**

#### **Default Account (Automatic)**
```bash
# Uses your username as account (e.g., if $USER=john, account=john)
claude-reactor                    # Uses account "john", isolated session/auth
cd ~/my-project && claude-reactor  # Separate session for this project under "john" account
```

#### **Named Account Usage**
```bash
# Work account setup
claude-reactor --account work     # Sets up work account isolation
claude-reactor config show       # Shows current account and project info

# Personal account setup  
claude-reactor --account personal # Completely separate from work account

# Account-specific API key setup
claude-reactor --account work --apikey sk-ant-xxx  # Account-specific credentials

# Switch between accounts across projects
cd ~/work-project
claude-reactor --account work     # Work account, work project session

cd ~/personal-project  
claude-reactor --account personal # Personal account, personal project session
```

#### **Project Session Isolation**
Each project/account combination gets completely isolated sessions:
```bash
# Project A with work account
cd ~/frontend-project
claude-reactor --account work     # Session: ~/.claude-reactor/work/frontend-project-a1b2c3d4/

# Project B with work account (separate session)
cd ~/backend-project  
claude-reactor --account work     # Session: ~/.claude-reactor/work/backend-project-e5f6g7h8/

# Same project with personal account (separate session)
cd ~/frontend-project
claude-reactor --account personal # Session: ~/.claude-reactor/personal/frontend-project-a1b2c3d4/
```

### **Project Management Commands**

#### **List All Accounts and Projects**
```bash
# View all accounts, projects, and containers
claude-reactor list               # Flat table view
claude-reactor list --json        # JSON output for scripting

# Example output:
# ACCOUNT    PROJECT         HASH     CTR LAST USED     SESSION DIR
# john       claude-reactor  f7894af8 1   2h ago        ~/.claude-reactor/john/claude-reactor-f7894af8
# work       frontend-app    a1b2c3d4 0   1d ago        ~/.claude-reactor/work/frontend-app-a1b2c3d4
```

#### **Granular Cleanup Options**
```bash
# Containers only (default, backward compatible)
claude-reactor clean              

# Containers + session data (conversation history)
claude-reactor clean --sessions   

# Containers + session data + authentication
claude-reactor clean --auth       

# Everything including global cache
claude-reactor clean --all        

# Additional options
claude-reactor clean --sessions --images  # Also remove Docker images
claude-reactor clean --force              # Skip confirmation prompts
```

**Container Images:**
- **Built-in variants**: `base`, `go`, `full`, `cloud`, `k8s` (auto-built and validated)
- **Custom Docker images**: Any Docker Hub or registry image (e.g. `ubuntu:22.04`, `node:18-alpine`)
- **Auto-detection**: Automatically selects best variant based on project files
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
make run                           # or claude-reactor

# 3. Work in container with full Go toolchain
# 4. Run tests and validation
make test-unit                     # Quick validation (5 seconds)

# 5. Clean up when done
make clean-containers              # or claude-reactor --clean
```

### **Registry-Enabled Workflows**
```bash
# Standard usage (registry-first, local fallback)
claude-reactor                   # Pulls from ghcr.io/dylutclaude-reactor automatically

# Development workflows
claude-reactor --dev             # Force local build for development/testing
claude-reactor --pull-latest     # Ensure you have the newest image

# CI/CD and maintenance
make pull-all                      # Pull all variants from registry
make push-all                      # Build and push to registry (requires auth)
make registry-login                # Login to GitHub Container Registry
```

### **Advanced Usage**
```bash
# Force specific configurations
claude-reactor --image cloud --danger    # Cloud tools + skip permissions
claude-reactor --rebuild                 # Force image rebuild

# Registry management
claude-reactor --dev                       # Force local build (disable registry)
claude-reactor --registry-off              # Disable registry completely
claude-reactor --pull-latest               # Force pull latest from registry

# Manual Docker control (rarely needed)
docker build --target go -t claude-reactor-go .
docker run -d --name claude-agent-go -v "$(pwd)":/app claude-reactor-go
```

## Configuration Files

### **Project Session Configuration (New Structure)**
Configuration is now stored in account/project-specific session directories:

**Location**: `~/.claude-reactor/{account}/{project-name}-{project-hash}/.claude-reactor`

```bash
variant=go
account=work
danger=true
host_docker=false
session_persistence=true
```

**Configuration Options:**
- `variant=` - Container variant (base, go, full, cloud, k8s, or custom image)
- `account=` - Claude account name (defaults to $USER, fallback to "user")
- `danger=` - Enable danger mode (true/false)
- `host_docker=` - Enable host Docker access (true/false)
- `host_docker_timeout=` - Timeout for Docker operations (e.g., "5m", "0" for unlimited)
- `session_persistence=` - Enable session persistence (true/false)

**Key Changes:**
- ✅ **Configuration moved** from local project directory to session directory
- ✅ **Account isolation** - each account has separate config storage
- ✅ **Project isolation** - each project gets unique session directory
- ✅ **Persistent sessions** - conversation history maintained per project/account

### **Container Registry Configuration**

Claude-Reactor automatically pulls pre-built images from GitHub Container Registry for faster startup times.

**Registry Settings:**
```bash
# Environment variables (optional)
export CLAUDE_REACTOR_REGISTRY="ghcr.io/dylutclaude-reactor"  # Default registry
export CLAUDE_REACTOR_USE_REGISTRY=true                        # Enable registry (default: true)
export CLAUDE_REACTOR_TAG=latest                               # Image tag (default: latest)
```

**Registry Behavior:**
- **Default**: Attempts to pull from `ghcr.io/dylutclaude-reactor` first
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
├── {account}/                           # Account session directories
│   ├── {project-name}-{project-hash}/   # Project-specific sessions
│   │   ├── .claude-reactor              # Project configuration
│   │   ├── projects/                    # Claude conversation history
│   │   └── shell-snapshots/            # Shell session data
├── .{account}-claude.json               # Account-specific Claude credentials
├── .claude-reactor-{account}-env        # Account-specific API keys (optional)
└── .default-claude.json                # Default account credentials
```

**Examples:**
```bash
~/.claude-reactor/
├── john/                               # Default account (from $USER)
│   ├── claude-reactor-f7894af8/        # This project session
│   └── frontend-app-a1b2c3d4/          # Another project session  
├── work/                               # Work account sessions
│   └── backend-api-e5f6g7h8/
├── .john-claude.json                   # John's Claude credentials
├── .work-claude.json                   # Work account credentials
└── .claude-reactor-work-env            # Work account API key
```

**Automatic Setup:**
- First time using an account: Config auto-copied from `~/.claude.json`
- Each account gets isolated OAuth tokens and session data
- Container naming: `claude-reactor-{variant}-{arch}-{project-hash}-{account}`
- Persistent authentication eliminates re-login requirements

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
# 2. Push to ghcr.io/dylutclaude-reactor-*:v0.2.0
# 3. Push to ghcr.io/dylutclaude-reactor-*:latest
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
CLAUDE_REACTOR_TAG=v0.1.0 claude-reactor --pull-latest
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

## Troubleshooting Account Isolation and Authentication

### **Authentication Issues**

#### **Problem: Container requires re-login every restart**
**Symptoms**: Claude CLI asks for authentication every time container is restarted
**Solution**: 
```bash
# Check if account-specific config exists
ls -la ~/.claude-reactor/.{your-account}-claude.json

# If missing, authenticate once and config will be auto-created
claude-reactor --account your-account --interactive-login

# Verify persistent authentication
claude-reactor list  # Should show your account and projects
```

#### **Problem: "Authentication config not found for account"**
**Symptoms**: Error when trying to use specific account
**Solution**:
```bash
# First-time account setup - auto-copies from main Claude config
claude-reactor --account work --interactive-login

# Or setup with API key
claude-reactor --account work --apikey sk-ant-your-key

# Verify account was created
claude-reactor list
```

#### **Problem: Cannot switch between accounts**
**Symptoms**: Always using same account despite `--account` flag
**Solution**:
```bash
# Check current configuration
claude-reactor config show

# Force account recreation with different account
claude-reactor run --account personal  # Forces recreation for personal account

# Verify account isolation
claude-reactor list  # Should show separate accounts
```

### **Container Management Issues**

#### **Problem: "Container name already in use" error (Fixed)**
**Symptoms**: `Error response from daemon: Conflict. The container name is already in use`
**Solution**: This is now automatically resolved by smart container lifecycle management
```bash
# This now works automatically - container is reused
claude-reactor run

# Force recreation if needed
claude-reactor run --image go  # Any argument forces recreation
```

#### **Problem: Wrong container being reused**
**Symptoms**: Container reused when you want a fresh one
**Solution**: Pass any argument to force recreation
```bash
# These force container recreation:
claude-reactor run --shell
claude-reactor run --image go
claude-reactor run --account work
claude-reactor run --danger

# This reuses existing container:
claude-reactor run  # (no arguments)
```

### **Session and Project Issues**

#### **Problem: Lost conversation history**
**Symptoms**: Previous Claude conversations not available
**Solution**:
```bash
# Check if session directory exists
ls -la ~/.claude-reactor/{account}/{project-name}-{hash}/

# List all projects to find your sessions
claude-reactor list

# Check project hash matches (8 characters from project path)
echo "/your/project/path" | sha256sum | cut -c1-8
```

#### **Problem: Session data mixed between projects**
**Symptoms**: Conversation history appearing in wrong project
**Solution**: Each project/account combo now gets isolated sessions automatically
```bash
# Verify isolation
claude-reactor list  # Should show separate sessions per project

# Clean up mixed sessions if needed
claude-reactor clean --sessions --auth  # Nuclear option - removes all sessions
```

### **Account Directory Issues**

#### **Problem: Permission denied accessing ~/.claude-reactor/**
**Symptoms**: Cannot read or write account configuration
**Solution**:
```bash
# Check permissions
ls -la ~/.claude-reactor/

# Fix permissions if needed
chmod 755 ~/.claude-reactor/
chmod 600 ~/.claude-reactor/.*.json
chmod 600 ~/.claude-reactor/.claude-reactor-*-env

# Verify access
claude-reactor list
```

#### **Problem: Default account not using username**
**Symptoms**: Account shows as "user" instead of your username
**Solution**:
```bash
# Check USER environment variable
echo $USER

# If empty, set it:
export USER=$(whoami)

# Verify default account logic
claude-reactor config show
```

### **List and Clean Command Issues**

#### **Problem: `claude-reactor list` shows no projects**
**Symptoms**: Empty list despite having used claude-reactor before
**Solution**:
```bash
# Check if directory structure exists
ls -la ~/.claude-reactor/

# Check for old structure (pre-isolation)
ls -la .claude-reactor  # Old project-local config

# If you have old configs, they won't be migrated automatically
# Use claude-reactor normally to create new structure
claude-reactor run
```

#### **Problem: Clean command not removing expected data**
**Symptoms**: Data still present after clean operation
**Solution**:
```bash
# Understand clean levels:
claude-reactor clean --help

# Use appropriate level:
claude-reactor clean              # Containers only
claude-reactor clean --sessions   # + session data
claude-reactor clean --auth       # + authentication
claude-reactor clean --all        # Everything

# Force without confirmation
claude-reactor clean --auth --force
```

### **Common Account Isolation Patterns**

#### **Development Team Setup**
```bash
# Each team member uses their own default account
# john: ~/.claude-reactor/john/
# alice: ~/.claude-reactor/alice/

# Shared project, separate sessions
cd ~/team-project
john$ claude-reactor  # Uses john account
alice$ claude-reactor  # Uses alice account
```

#### **Work/Personal Separation**
```bash
# Work projects
cd ~/work-project
claude-reactor --account work

# Personal projects  
cd ~/personal-project
claude-reactor --account personal

# Each maintains separate authentication and session history
```

#### **Multiple Client Projects**
```bash
# Client A project
cd ~/client-a-project
claude-reactor --account client-a

# Client B project
cd ~/client-b-project  
claude-reactor --account client-b

# Complete isolation between client projects
```

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
claude-reactor --image node:18-alpine     # Multi-arch (preferred)
claude-reactor --image --platform linux/arm64 node:18-alpine

# Force AMD64 if needed (slower on M1):
claude-reactor --image --platform linux/amd64 your-image:tag
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
claude-reactor --image node:18-alpine    # Multi-arch
claude-reactor --image python:3.11-slim  # Multi-arch

# Language-specific optimized images
claude-reactor --image golang:1.21-alpine     # Go development
claude-reactor --image rust:1.75-slim-bullseye # Rust development
claude-reactor --image openjdk:21-slim        # Java development
```

#### **Registry Selection**
```bash
# Use geographically closer registries
claude-reactor --image gcr.io/image:tag      # Google Container Registry
claude-reactor --image ghcr.io/user/image:tag # GitHub Container Registry

# Use cached layers when possible
claude-reactor --image ubuntu:22.04  # Likely cached
claude-reactor --image alpine:latest # Small and common
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
- ✅ **Registry integration**: Automatic pulls from GitHub Container Registry with local fallback
- ✅ **Language ecosystem support**: Go, Rust, Java, Python, Node.js, cloud tools
- ✅ **Professional automation**: 30+ Makefile targets including registry management
- ✅ **CI/CD pipeline**: Multi-architecture builds, security scanning, automated releases
- ✅ **Comprehensive testing**: Unit, integration, and demo validation
- ✅ **Smart persistence**: Remembers preferences without manual configuration
- ✅ **Production architecture**: Multi-stage builds, security best practices, efficient resource usage

**Ready for**: Personal projects, team development, educational use, enterprise development workflows, and public distribution via container registry.
