# Custom Docker Images Guide

Claude-reactor supports using any Docker image as your development environment. This guide provides practical examples and troubleshooting tips.

## Quick Start

```bash
# Try a custom image
claude-reactor run --image ubuntu:22.04

# Use with specific account
claude-reactor run --image node:18-alpine --account work

# Check what's available in your image
claude-reactor run --image python:3.11 --shell
```

## Common Use Cases

### Language-Specific Development

```bash
# Node.js development
claude-reactor run --image node:18-alpine
claude-reactor run --image node:20

# Python development  
claude-reactor run --image python:3.11
claude-reactor run --image python:3.11-slim

# Ubuntu-based development
claude-reactor run --image ubuntu:22.04
claude-reactor run --image ubuntu:24.04

# Alpine-based (minimal)
claude-reactor run --image alpine:3.18
```

### Specialized Development Environments

```bash
# GPU development (NVIDIA CUDA)
claude-reactor run --image nvidia/cuda:11.8-devel-ubuntu22.04

# Database development
claude-reactor run --image postgres:15
claude-reactor run --image mysql:8.0

# Machine Learning
claude-reactor run --image tensorflow/tensorflow:latest-gpu
claude-reactor run --image pytorch/pytorch:2.0.1-cuda11.7-cudnn8-devel

# Custom registry images
claude-reactor run --image ghcr.io/your-org/dev-env:latest
claude-reactor run --image your-registry.com/team/dev-image:v1.2
```

## Image Requirements

### Essential Requirements
- **Linux-based**: Must run on linux/amd64 or linux/arm64
- **Claude CLI**: Must have Claude CLI installed and working
  ```bash
  # Test in your image:
  claude --version  # Should succeed
  ```

### Recommended Tools
These tools enhance the development experience (warnings shown if missing):
- **High Priority**: `git`, `curl`
- **Medium Priority**: `gh`, `wget`, `jq`, `make`, `nano`
- **Low Priority**: `yq`, `docker`, `vim`

## Building Custom Images

### Basic Dockerfile Example

```dockerfile
FROM ubuntu:22.04

# Install essential packages
RUN apt-get update && apt-get install -y \
    curl \
    git \
    build-essential \
    nano \
    && rm -rf /var/lib/apt/lists/*

# Install Claude CLI
RUN curl -fsSL https://claude.ai/install.sh | bash

# Add your development tools
RUN apt-get update && apt-get install -y \
    python3 \
    python3-pip \
    nodejs \
    npm

# Set working directory
WORKDIR /app

# Default command
CMD ["bash"]
```

### Multi-stage Build Example

```dockerfile
# Builder stage
FROM golang:1.21-alpine AS builder
RUN apk add --no-cache git curl
RUN curl -fsSL https://claude.ai/install.sh | bash

# Final stage
FROM alpine:3.18
COPY --from=builder /usr/local/bin/claude /usr/local/bin/claude
RUN apk add --no-cache git curl make nano
WORKDIR /app
CMD ["bash"]
```

## Validation Process

When you use a custom image, claude-reactor automatically:

1. **Pulls the image** (if not available locally)
2. **Validates compatibility**:
   - Checks Linux platform (linux/amd64 or linux/arm64)
   - Tests Claude CLI availability (`claude --version`)
3. **Analyzes packages**:
   - Tests 10 recommended development tools
   - Shows warnings for missing high-priority tools
   - Caches results for 30 days (images with same digest are immutable)
4. **Provides feedback**:
   - Success: Shows validation summary and package analysis
   - Failure: Shows specific errors with suggestions

## Troubleshooting

### Common Issues

#### "Custom image validation failed"
```bash
# Check what's wrong with verbose output
claude-reactor run --image your-image --verbose

# Test Claude CLI manually
docker run --rm -it your-image claude --version
```

#### "Claude CLI not found or not functional"
Your image needs Claude CLI installed:
```bash
# Add to your Dockerfile:
RUN curl -fsSL https://claude.ai/install.sh | bash
```

#### "Unsupported platform" 
Your image must be Linux-based:
```bash
# Check image platform
docker inspect your-image | grep Architecture
```

#### Image pull failures
```bash
# Try pulling manually first
docker pull your-image

# Use registry authentication if needed
docker login your-registry.com
claude-reactor run --image your-registry.com/image:tag
```

### Debug Commands

```bash
# Check Docker connectivity
claude-reactor debug info

# View detailed logs
claude-reactor run --image your-image --verbose --log-level debug

# Clear validation cache
rm -rf ~/.claude-reactor/image-cache/

# Test image manually
docker run --rm -it your-image /bin/bash
```

### Getting Help

- Run `claude-reactor run --help` for comprehensive usage information
- Use `claude-reactor debug info` to check system status
- Check validation cache: `~/.claude-reactor/image-cache/`
- Enable verbose logging: `--verbose --log-level debug`

## Best Practices

### Performance
- **Use image tags**: Specify exact versions (`ubuntu:22.04` not `ubuntu:latest`)
- **Long-term caching**: Validation results cached for 30 days (Docker digests are immutable)
- **Minimize layers**: Use multi-stage builds for smaller images

### Security  
- **Use official images**: Start with official base images when possible
- **Keep updated**: Regularly update your custom images
- **Scan for vulnerabilities**: Use `docker scout` or similar tools

### Development Workflow
```bash
# Set up project-specific configuration
cd my-project
claude-reactor run --image my-custom:latest --account work
# Config saved to .claude-reactor file

# Later runs use saved config
claude-reactor run  # Uses my-custom:latest with work account
```

## Examples by Use Case

### Web Development
```bash
# Frontend (Node.js)
claude-reactor run --image node:18-alpine

# Full-stack (with databases)
claude-reactor run --image node:18 --account fullstack
```

### Data Science
```bash  
# Python data science
claude-reactor run --image jupyter/datascience-notebook

# R development
claude-reactor run --image rocker/tidyverse
```

### DevOps
```bash
# Infrastructure as Code
claude-reactor run --image hashicorp/terraform:1.6

# Kubernetes development
claude-reactor run --image bitnami/kubectl
```

### Mobile Development
```bash
# Android development
claude-reactor run --image androidsdk/android-30

# React Native
claude-reactor run --image reactnativecommunity/react-native-android
```