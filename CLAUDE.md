# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Docker containerization project for the Claude CLI tool. The project creates a secure, isolated sandbox environment that allows developers to run Claude CLI commands within a Docker container, with optional Kubernetes access.

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

## Key File Structure (to be created)

```
.
├── .devcontainer/
│   └── devcontainer.json
└── Dockerfile
```

## Authentication Methods

The project supports two authentication approaches:
- **API Key**: Via environment file (`~/.env` with `ANTHROPIC_API_KEY`)
- **Interactive UI**: Direct login through Claude CLI (requires removing `runArgs` from devcontainer.json)

## Container Usage Patterns

**Recommended: claude-reactor script (automated management):**
```bash
# First time in a Go project - auto-detects and saves preference
./claude-reactor --variant go

# Subsequent runs - uses saved preference
./claude-reactor

# Override for specific needs
./claude-reactor --variant cloud --danger

# List available variants
./claude-reactor --list-variants

# Show current configuration
./claude-reactor --show-config
```

**Manual Docker approach (advanced users):**
```bash
# Build specific variant
docker build --target go -t claude-runner-go .

# Run with variant-specific naming
docker run -d --name claude-agent-go -v "$(pwd)":/app -v "${HOME}/.kube:/home/claude/.kube" -v "${HOME}/.claude:/home/claude/.claude" --env-file ~/.env claude-runner-go
```

## Configuration Files

### `.claude-reactor` (Project-specific settings)
```bash
variant=go
danger=true
```

This file is automatically created when you use `--variant` or `--danger` flags and stores your preferences per project directory.

## Important Notes

- **Architecture**: ARM64-compatible (M1 Macs, but works on x86_64 too)
- **Security**: Non-root `claude` user with sudo access inside container
- **Persistence**: Claude config, git settings, and Kubernetes config mounted from host
- **Isolation**: Complete separation from host system while maintaining access to necessary configs
- **Auto-detection**: Intelligently selects appropriate variant based on project files
- **Modularity**: Choose only the tools you need to keep container size manageable