# Claude-Reactor

A professional, modular Docker containerization system for Claude CLI development workflows.

## Overview

Claude-Reactor transforms the basic Claude CLI into a comprehensive development environment with intelligent automation, multi-language support, and production-ready tooling.

**Key Features:**
- **Zero Configuration**: Auto-detects project type and sets up appropriate development environment
- **Language Agnostic**: Supports Go, Rust, Java, Python, Node.js, and cloud development workflows  
- **Professional Automation**: Comprehensive CLI with intelligent container management
- **Account Isolation**: Complete separation between different Claude accounts
- **Custom Images**: Support for custom Docker images with compatibility validation
- **VS Code Integration**: Automatic dev container generation with project-specific extensions

## Quick Start

### Prerequisites

- Docker Desktop installed and running
- Claude-reactor binary (see Installation section below)

### Basic Usage

```bash
# Start Claude CLI (auto-detects project type and creates appropriate container)
./claude-reactor

# Use specific container images
./claude-reactor run --image go           # Go development tools
./claude-reactor run --image full         # Go + Rust + Java + databases
./claude-reactor run --image cloud        # Full + AWS/GCP/Azure CLIs
./claude-reactor run --image k8s          # Full + enhanced Kubernetes tools

# Use custom Docker image
./claude-reactor run --image python:3.11  # Custom Python image (with validation)
```

### Account Management

Claude-Reactor provides complete account isolation for teams and personal use:

```bash
# Default account (uses ~/.claude.json)
./claude-reactor                      

# Work account (isolated config and containers)
./claude-reactor run --account work      

# Personal account 
./claude-reactor run --account personal  

# Check current configuration
./claude-reactor config show
```

**Account-Specific Files:**
- **Config**: `~/.claude-reactor/.work-claude.json`, `~/.claude-reactor/.personal-claude.json`
- **Containers**: `claude-reactor-go-work`, `claude-reactor-go-personal`
- **Settings**: Preferences saved per project in `.claude-reactor`

## Authentication Setup

Claude-Reactor automatically uses your existing Claude CLI authentication. For new accounts, authentication happens once and is remembered:

### First-time Setup for New Accounts

```bash
# For work account with API key
./claude-reactor run --account work --apikey sk-ant-your-key-here

# For interactive authentication
./claude-reactor run --account work --interactive-login

# After setup, just use the account name
./claude-reactor run --account work
```

**How it Works:**
- **Default account**: Uses your existing `~/.claude.json` file
- **Named accounts**: Creates isolated configs in `~/.claude-reactor/.{account}-claude.json`
- **One-time setup**: Authentication is configured once per account and remembered
- **Container isolation**: Each account gets separate containers with independent authentication

## Advanced Usage

### Container Variants

Built-in variants optimized for different development needs:

| Variant | Size | Tools | Use Case |
|---------|------|-------|----------|
| `base` | ~500MB | Node.js, Python, Git | Lightweight development |
| `go` | ~800MB | Base + Go toolchain | Go development |
| `full` | ~1.2GB | Go + Rust, Java, Databases | Multi-language projects |
| `cloud` | ~1.5GB | Full + AWS/GCP/Azure CLIs | Cloud development |
| `k8s` | ~1.4GB | Full + Enhanced Kubernetes tools | Kubernetes workflows |

### VS Code Integration

Automatic dev container generation with project-specific setup:

```bash
# Generate .devcontainer for current project
./claude-reactor devcontainer generate

# Force specific image
./claude-reactor devcontainer generate --image go

# Auto-detects project type and includes relevant extensions:
# - Go projects: Go extension, Delve debugger
# - Node.js: ESLint, Prettier, Node.js debugger
# - Python: Python extension, Pylance, debugger
# - Multi-language: Includes all relevant tooling
```

### Custom Docker Images

Use any Linux-based Docker image with validation:

```bash
# Python scientific computing
./claude-reactor run --image jupyter/scipy-notebook

# Custom enterprise image  
./claude-reactor run --image myregistry.com/dev-env:latest

# Image validation provides compatibility feedback:
# ✅ Linux platform detected
# ✅ Claude CLI installation verified
# ⚠️ Missing recommended tools: git, curl
# 📦 Package analysis: 8/10 recommended tools available
```

## Configuration Management

### Project-Specific Settings
Each project directory can have custom preferences saved in `.claude-reactor`:

```bash
# Example .claude-reactor file (auto-created)
image=go
danger=true
account=work
```

### Global Account Configuration
Account configurations stored in `~/.claude-reactor/`:

```
~/.claude-reactor/
├── .default-claude.json      # Default account OAuth config
├── .work-claude.json         # Work account OAuth config  
├── .personal-claude.json     # Personal account OAuth config
├── .default-claude/          # Default account Claude CLI state
├── .work-claude/             # Work account Claude CLI state
└── image-cache/              # Docker image validation cache
```

## Common Workflows

### Team Development Setup

```bash
# Set up work environment
cd /path/to/work/project
./claude-reactor run --account work --image go

# Set up personal environment  
cd /path/to/personal/project
./claude-reactor run --account personal --image base

# Each has isolated containers, configs, and authentication
```

### Multi-Language Project

```bash
cd /path/to/fullstack/project

# Use full image for comprehensive tooling
./claude-reactor run --image full

# Container includes: Go, Rust, Java, Node.js, Python, databases
# Auto-detects project type and provides relevant tooling
```

### Custom Enterprise Environment

```bash
# Use company-specific development image
./claude-reactor run --image enterprise-registry.com/dev-env:v2.1

# System validates compatibility and shows tool analysis:
# ✅ Compatible with claude-reactor
# 📦 Available tools: git, curl, jq, docker
# ⚠️ Missing: ripgrep, fzf (non-critical)
```

## Project Templates and Scaffolding

Generate new projects with intelligent defaults:

```bash
# Interactive project creation
./claude-reactor template init

# Direct project creation from templates
./claude-reactor template new go-api my-api
./claude-reactor template new rust-cli my-cli
./claude-reactor template new node-webapp my-webapp
```

## Command Reference

### Core Commands

```bash
# Container management
./claude-reactor                         # Start Claude CLI (smart defaults)
./claude-reactor run --shell             # Launch bash shell instead
./claude-reactor run --danger            # Skip Claude CLI permissions dialog
./claude-reactor config show             # Display current configuration

# Image and account management  
./claude-reactor run --image go          # Use Go development image
./claude-reactor run --account work      # Switch to work account
./claude-reactor clean                   # Remove current project container
./claude-reactor clean --all             # Remove all containers

# Development tools
./claude-reactor devcontainer generate   # Create VS Code dev container
./claude-reactor template init           # Interactive project scaffolding
./claude-reactor debug cache info        # Show image validation cache
```

## Installation

### Option 1: Pre-built Binaries (Recommended)

```bash
# Build and auto-install (detects best method for your OS)
make build && ./INSTALL

# Specific installation methods:
make build && ./INSTALL --local    # ~/bin (recommended for macOS)  
make build && ./INSTALL --system   # /usr/local/bin (good for Linux)

# Or use Makefile shortcut
make build install                  # Auto-detection
```

### Option 2: Build from Source

```bash
# Clone and build
git clone <repository>
cd claude-reactor
make build

# Install to system PATH (optional)
./INSTALL

# Or use directly without installing
./dist/claude-reactor-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m) --help
```

**The INSTALL script will:**
- **Auto-detect** the best installation method for your OS
- **macOS**: Install to `~/bin` (avoids Gatekeeper security issues)
- **Linux**: Install to `/usr/local/bin` (system-wide access)  
- Make it executable and handle macOS security restrictions
- Provide PATH setup instructions when needed

**Supported Platforms:**
- Linux: x86_64, arm64
- macOS: x86_64, arm64 (Apple Silicon)

**To uninstall:** `./INSTALL --uninstall`

## Troubleshooting

### Common Issues

**Authentication Problems:**
```bash
# Force interactive login for new account
./claude-reactor run --account newaccount --interactive-login

# Check authentication status
./claude-reactor config show
```

**Container Issues:**
```bash
# Clean and rebuild
./claude-reactor clean && ./claude-reactor build go

# Check Docker daemon
docker info
```

**Custom Image Problems:**
```bash
# Check image validation details
./claude-reactor debug image your-image:tag

# Clear validation cache
./claude-reactor debug cache clear
```

## Architecture

Claude-Reactor uses a modular Go architecture with:

- **Account Isolation**: Complete separation of Claude configurations and containers
- **Image Validation**: Automatic compatibility checking for custom Docker images  
- **Project Detection**: Smart detection of project types for optimal tooling
- **Mount Management**: Secure mounting of configs, projects, and additional directories
- **Template System**: Language-specific project scaffolding with best practices

**File Structure:**
- Configuration: `.claude-reactor` (per project), `~/.claude-reactor/` (global)
- Containers: Named with account and project isolation
- Validation Cache: `~/.claude-reactor/image-cache/` (digest-based, 30-day expiry)