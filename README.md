# Claude CLI in Docker

A secure, isolated Docker environment for running the Claude CLI with Kubernetes access support and automated container registry integration.

## Overview

This project provides a containerized environment for the Claude CLI tool that:
- Runs in complete isolation from your host machine
- **Automatically pulls pre-built images from GitHub Container Registry**
- Supports both API key and interactive authentication methods
- Includes kubectl for Kubernetes cluster management
- Integrates seamlessly with VS Code Dev Containers
- Uses the `claude` user (non-root) for enhanced security
- Includes comprehensive development tools (ripgrep, jq, git, Node.js, Python)
- **5 specialized variants**: base, go, full, cloud, k8s (auto-detected)

## Components

- **Multi-stage Dockerfile**: ARM64/AMD64-compatible containers with specialized variants
- **GitHub Actions CI/CD**: Automated multi-architecture builds and registry publishing
- **claude-reactor script**: Intelligent container management with registry-first behavior
- **Comprehensive Makefile**: 30+ targets for development, testing, and registry operations
- **Dev Container**: VS Code integration with automatic extension installation and directory mounting

## Quick Start

### Prerequisites

- Docker Desktop installed and running
- VS Code with Dev Containers extension (for VS Code integration)
- Internet connection (for registry pulls - optional, builds locally as fallback)

### Using claude-reactor (Recommended)

The `claude-reactor` script provides the easiest way to manage the containerized Claude CLI environment:

```bash
# First time setup - automatically pulls from registry or builds locally
./claude-reactor

# Variant selection (auto-detected based on project files)
./claude-reactor --variant go           # Force Go development environment
./claude-reactor --variant full         # Full stack (Go + Rust + Java + DBs)
./claude-reactor --variant cloud        # Cloud tools (AWS/GCP/Azure CLIs)
./claude-reactor --variant k8s          # Kubernetes tools (helm, k9s, stern)

# Registry management
./claude-reactor --dev                  # Force local build (disable registry)
./claude-reactor --registry-off         # Disable registry completely
./claude-reactor --pull-latest          # Force pull latest from registry

# Other options
./claude-reactor --help
./claude-reactor --verbose              # Show detailed output
./claude-reactor --rebuild              # Force rebuild (auto-cleans old container)
./claude-reactor --clean                # Remove existing container
./claude-reactor --interactive-login    # Force interactive UI login
./claude-reactor --danger              # Launch directly into Claude CLI
```

### Container Variants

| Variant | Contents | Size | Use Case |
|---------|----------|------|----------|
| **base** | Node.js, Python, Claude CLI, basic tools | ~500MB | JavaScript/Python projects |
| **go** | Base + Go toolchain & utilities | ~800MB | Go development |
| **full** | Go + Rust, Java, database clients | ~1.2GB | Multi-language projects |
| **cloud** | Full + AWS/GCP/Azure CLIs | ~1.5GB | Cloud infrastructure |
| **k8s** | Full + Kubernetes tools (helm, k9s) | ~1.4GB | Kubernetes development |

**Important:** On first run with `--danger` mode, you'll need to accept the Claude CLI trust dialog once. This acceptance persists via the mounted `~/.claude` directory.

## Registry Integration

### How It Works

Claude-Reactor uses a **registry-first, local-fallback** approach:

1. **Registry Check**: First attempts to pull from `ghcr.io/dyluth/claude-reactor`
2. **Multi-Architecture**: Automatically selects ARM64 or AMD64 based on your system
3. **Local Fallback**: If registry pull fails, builds the image locally
4. **Development Override**: Use `--dev` flag to force local builds

### Registry Behavior

| Scenario | Action |
|----------|--------|
| **First run** | Pulls from registry, falls back to local build if needed |
| **Image exists locally** | Uses local image (fast startup) |
| **`--pull-latest`** | Forces fresh pull from registry |
| **`--dev` mode** | Skips registry, builds locally |
| **No internet** | Automatically falls back to local build |

### Environment Variables

```bash
# Registry configuration (optional)
export CLAUDE_REACTOR_REGISTRY="ghcr.io/dyluth/claude-reactor"  # Default
export CLAUDE_REACTOR_USE_REGISTRY=true                        # Enable registry
export CLAUDE_REACTOR_TAG=latest                               # Image tag
```

### Option A: Manual Docker Usage

1. **Build the image:**
   ```bash
   docker build -t claude-reactor .
   ```

2. **Run the container:**
   
   **For API Key Authentication:**
   ```bash
   docker run -d --name claude-agent -v "$(pwd)":/app -v "${HOME}/.kube:/root/.kube" --env-file ~/.env claude-reactor
   ```
   
   **For Interactive UI Login:**
   ```bash
   docker run -d --name claude-agent -v "$(pwd)":/app -v "${HOME}/.kube:/root/.kube" claude-reactor
   ```

3. **Connect to the container:**
   ```bash
   docker exec -it claude-agent /bin/bash
   ```

4. **Use the tools:**
   ```bash
   claude --version
   kubectl version --client
   ```

### Option B: VS Code Dev Containers (Recommended)

1. **Choose authentication method:**
   - **API Key**: Ensure `~/.env` contains `ANTHROPIC_API_KEY=your_key_here`
   - **Interactive**: Remove the `runArgs` block from `.devcontainer/devcontainer.json`

2. **Open in VS Code:**
   - Open this project folder in VS Code
   - Click "Reopen in Container" when prompted
   - Or use Command Palette: "Dev Containers: Reopen in Container"

3. **Start working:**
   - Open a new terminal in VS Code
   - Claude CLI and kubectl are ready to use
   - Kubernetes extension provides UI-based cluster management

## Authentication Setup

### API Key Method
Create `~/.env` on your host machine:
```
ANTHROPIC_API_KEY=your_api_key_here
```

### Interactive Method
Use `./claude-reactor --interactive-login` to force interactive authentication, or remove the `runArgs` section from `.devcontainer/devcontainer.json`. The file should look like:
```json
{
	"name": "Claude Interactive Environment",
	"dockerFile": "../Dockerfile",
	"customizations": {
		"vscode": {
			"extensions": [
				"dbaeumer.vscode-eslint",
				"ms-kubernetes-tools.vscode-kubernetes-tools"
			]
		}
	},
	"mounts": [
		"source=${localWorkspaceFolder},target=/app,type=bind,consistency=cached",
		"source=${localEnv:HOME}/.kube,target=/root/.kube,type=bind,consistency=cached"
	]
}
```

## First-Time Setup

### Claude CLI Trust Dialog
When using Claude CLI for the first time in the container (especially with `--danger` mode), you'll need to accept the trust dialog once:

1. Run `./claude-reactor --danger`
2. Accept the trust dialog when prompted
3. This acceptance is saved to your mounted `~/.claude` directory and persists across container restarts

### Git Configuration
Your host's `~/.gitconfig` is automatically mounted, so git operations will use your existing configuration.

## Kubernetes Access

The container automatically mounts your local `~/.kube` directory, providing access to your Kubernetes clusters. Test with:
```bash
kubectl get pods
```

## Verified Versions

- **Node.js**: 22.17.0 (LTS)
- **Claude CLI**: 1.0.67
- **kubectl**: 1.33.3
- **Base Image**: debian:bullseye-slim (ARM64 compatible)

## Advanced Usage

### Makefile Automation

The project includes comprehensive Makefile automation:

```bash
# Container management
make run                    # Start with auto-detected variant
make run-go                 # Start with Go variant
make stop-all               # Stop all containers
make clean-all              # Complete cleanup

# Registry operations
make pull-all               # Pull core variants from registry
make pull-extended          # Pull all variants from registry
make push-all               # Build and push core variants (requires auth)
make registry-login         # Login to GitHub Container Registry

# Development
make test                   # Run complete test suite
make test-unit              # Quick unit tests
make demo                   # Interactive feature demo
make help                   # Show all available targets
```

## Cleanup

Using claude-reactor:
```bash
./claude-reactor --clean    # Remove container only
./claude-reactor --rebuild  # Rebuild image (auto-cleans container)
```

Using Makefile:
```bash
make clean-containers       # Remove all containers
make clean-images          # Remove all images
make clean-all             # Complete cleanup
```

Manual cleanup:
```bash
# Stop and remove container
docker stop claude-agent && docker rm claude-agent

# Remove image
docker rmi claude-reactor
```

## Features

- **Registry Integration**: Automatic pulls from GitHub Container Registry with local fallback
- **Multi-Architecture**: Native support for ARM64 (M1 Macs) and AMD64 systems
- **5 Specialized Variants**: base, go, full, cloud, k8s (auto-detected based on project)
- **CI/CD Pipeline**: Automated builds, testing, and publishing via GitHub Actions
- **Security**: Runs as non-root `claude` user with sudo access
- **Development Tools**: Includes ripgrep, jq, fzf, vim, nano, git, Python, Node.js tools
- **Git Integration**: Git-aware prompt showing branch and dirty status
- **Configuration Persistence**: Claude CLI, git, and kubectl configs persist via mounted directories
- **Professional Automation**: 30+ Makefile targets for all development workflows
- **Account Isolation**: Separate authentication and containers per Claude account