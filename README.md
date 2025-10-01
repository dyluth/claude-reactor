# Claude-Reactor

[![Build Status](https://github.com/dyluth/claude-reactor/workflows/Build%20and%20Push%20Docker%20Images/badge.svg)](https://github.com/dyluth/claude-reactor/actions)
[![codecov](https://codecov.io/gh/dyluth/claude-reactor/branch/main/graph/badge.svg)](https://codecov.io/gh/dyluth/claude-reactor)
[![Go Report Card](https://goreportcard.com/badge/github.com/dyluth/claude-reactor)](https://goreportcard.com/report/github.com/dyluth/claude-reactor)

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

## Installation

### Quick Install (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/dyluth/claude-reactor/main/install.sh | bash
```

The installer will:
- Auto-detect your platform (Linux/macOS, x86_64/ARM64)
- Download the latest binary with checksum verification
- Install to `~/.local/bin/claude-reactor`
- Check for Docker dependency

### Manual Installation

1. **Download Binary**: Get the latest release for your platform from [GitHub Releases](https://github.com/dyluth/claude-reactor/releases)

2. **Install Binary**:
   ```bash
   # Download (replace with your platform)
   curl -fsSL -O https://github.com/dyluth/claude-reactor/releases/download/v0.1.0/claude-reactor-v0.1.0-linux-amd64

   # Make executable and install
   chmod +x claude-reactor-v0.1.0-linux-amd64
   mv claude-reactor-v0.1.0-linux-amd64 ~/.local/bin/claude-reactor

   # Add to PATH (if needed)
   export PATH="$PATH:$HOME/.local/bin"
   ```

3. **Verify Installation**:
   ```bash
   claude-reactor --version
   ```

### Prerequisites

- **Docker**: Required for container functionality
  - Install from [docker.com](https://docs.docker.com/get-docker/)
  - Ensure Docker daemon is running
- **Platform**: Linux or macOS (x86_64 or ARM64)

## Quick Start

### Basic Usage

```bash
# Start Claude CLI (auto-detects project type and creates appropriate container)
claude-reactor

# Use specific container images
claude-reactor run --image go           # Go development tools
claude-reactor run --image full         # Go + Rust + Java + databases
claude-reactor run --image cloud        # Full + AWS/GCP/Azure CLIs
claude-reactor run --image k8s          # Full + enhanced Kubernetes tools

# Use custom Docker image
claude-reactor run --image python:3.11  # Custom Python image (with validation)
```

### Account Management

Claude-Reactor provides complete account isolation for teams and personal use:

```bash
# Default account (uses ~/.claude.json)
claude-reactor

# Work account (isolated config and containers)
claude-reactor run --account work

# Personal account
claude-reactor run --account personal  

# Check current configuration
claude-reactor config show
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
claude-reactor run --image jupyter/scipy-notebook

# Custom enterprise image
claude-reactor run --image myregistry.com/dev-env:latest

# Image validation provides compatibility feedback:
# ✅ Linux platform detected
# ✅ Claude CLI installation verified
# ⚠️ Missing recommended tools: git, curl
# 📦 Package analysis: 8/10 recommended tools available
```

## Configuration Management
