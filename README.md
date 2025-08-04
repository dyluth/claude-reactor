# Claude CLI in Docker

A secure, isolated Docker environment for running the Claude CLI with Kubernetes access support.

## Overview

This project provides a containerized environment for the Claude CLI tool that:
- Runs in complete isolation from your host machine
- Supports both API key and interactive authentication methods
- Includes kubectl for Kubernetes cluster management
- Integrates seamlessly with VS Code Dev Containers

## Components

- **Dockerfile**: ARM64-compatible container with Debian, Node.js 22.17.0, Claude CLI, and kubectl
- **Dev Container**: VS Code integration with automatic extension installation and directory mounting

## Quick Start

### Prerequisites

- Docker Desktop installed and running
- VS Code with Dev Containers extension (for Option B)

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
Remove the `runArgs` section from `.devcontainer/devcontainer.json`. The file should look like:
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

```bash
# Stop and remove container
docker stop claude-agent && docker rm claude-agent

# Remove image
docker rmi claude-runner
```