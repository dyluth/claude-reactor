# Claude CLI in Docker

A secure, isolated Docker environment for running the Claude CLI with Kubernetes access support.

## Overview

This project provides a containerized environment for the Claude CLI tool that:
- Runs in complete isolation from your host machine
- Supports both API key and interactive authentication methods
- Includes kubectl for Kubernetes cluster management
- Integrates seamlessly with VS Code Dev Containers
- Uses the `claude` user (non-root) for enhanced security
- Includes comprehensive development tools (ripgrep, jq, git, Node.js, Python)

## Components

- **Dockerfile**: ARM64-compatible container with Debian, Node.js 22.17.0, Claude CLI, and kubectl
- **Dev Container**: VS Code integration with automatic extension installation and directory mounting

## Quick Start

### Prerequisites

- Docker Desktop installed and running
- VS Code with Dev Containers extension (for Option B)

### Using claude-reactor (Recommended)

The `claude-reactor` script provides the easiest way to manage the containerized Claude CLI environment:

```bash
# First time setup - build image and create container
./claude-reactor

# Available options
./claude-reactor --help
./claude-reactor --verbose              # Show detailed output
./claude-reactor --rebuild              # Force rebuild (auto-cleans old container)
./claude-reactor --clean                # Remove existing container
./claude-reactor --interactive-login    # Force interactive UI login
./claude-reactor --danger              # Launch directly into Claude CLI
```

**Important:** On first run with `--danger` mode, you'll need to accept the Claude CLI trust dialog once. This acceptance persists via the mounted `~/.claude` directory.

### Option A: Manual Docker Usage

1. **Build the image:**
   ```bash
   docker build -t claude-runner .
   ```

2. **Run the container:**
   
   **For API Key Authentication:**
   ```bash
   docker run -d --name claude-agent -v "$(pwd)":/app -v "${HOME}/.kube:/root/.kube" --env-file ~/.env claude-runner
   ```
   
   **For Interactive UI Login:**
   ```bash
   docker run -d --name claude-agent -v "$(pwd)":/app -v "${HOME}/.kube:/root/.kube" claude-runner
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

## Cleanup

Using claude-reactor:
```bash
./claude-reactor --clean    # Remove container only
./claude-reactor --rebuild  # Rebuild image (auto-cleans container)
```

Manual cleanup:
```bash
# Stop and remove container
docker stop claude-agent && docker rm claude-agent

# Remove image
docker rmi claude-runner
```

## Features

- **Security**: Runs as non-root `claude` user with sudo access
- **Development Tools**: Includes ripgrep, jq, fzf, vim, nano, git, Python, Node.js tools
- **Git Integration**: Git-aware prompt showing branch and dirty status
- **Configuration Persistence**: Claude CLI, git, and kubectl configs persist via mounted directories
- **Easy Management**: Single script handles all container lifecycle operations