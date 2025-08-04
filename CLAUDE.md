# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Docker containerization project for the Claude CLI tool. The project creates a secure, isolated sandbox environment that allows developers to run Claude CLI commands within a Docker container, with optional Kubernetes access.

## Project Architecture

This project is designed to create two main components:

1. **Dockerfile**: Defines an ARM64-compatible container environment with:
   - Debian bullseye-slim base image
   - Node.js installed via nvm (version 22.17.1)
   - Claude CLI (`@anthropic-ai/claude-cli`) installed globally
   - kubectl for Kubernetes cluster access
   - Working directory set to `/app`

2. **Dev Container Configuration**: Enables VS Code integration with:
   - Docker container mounting
   - Kubernetes tools extension
   - Local `.kube` directory mounting for cluster access
   - Environment variable injection for API key authentication

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

**Manual Docker approach:**
```bash
# Build image
docker build -t claude-runner .

# Run with API key and persistent configs
docker run -d --name claude-agent -v "$(pwd)":/app -v "${HOME}/.kube:/root/.kube" -v "${HOME}/.claude:/root/.claude" -v "${HOME}/.gitconfig:/root/.gitconfig" --env-file ~/.env claude-runner

# Run for interactive login with persistent configs
docker run -d --name claude-agent -v "$(pwd)":/app -v "${HOME}/.kube:/root/.kube" -v "${HOME}/.claude:/root/.claude" -v "${HOME}/.gitconfig:/root/.gitconfig" claude-runner

# Connect to container
docker exec -it claude-agent /bin/bash
```

**VS Code Dev Containers approach:**
- Use "Reopen in Container" functionality
- Automatic environment setup with extensions
- Claude authentication and git config persist between container restarts via mounted directories

## Important Notes

- This project targets ARM64 architecture (M1 Macs)
- The container includes both Claude CLI and kubectl for Kubernetes operations
- Local Kubernetes config is mounted to enable cluster access from within the container
- The setup ensures complete isolation from the host machine