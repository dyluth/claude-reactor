# Installation Guide

## Automated Installation

### Quick Install (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/dyluth/claude-reactor/main/install.sh | bash
```

### Custom Installation Options

```bash
# Install specific version
curl -fsSL https://raw.githubusercontent.com/dyluth/claude-reactor/main/install.sh | bash -s -- --version v0.2.0

# Install to custom directory
curl -fsSL https://raw.githubusercontent.com/dyluth/claude-reactor/main/install.sh | bash -s -- --dir /usr/local/bin

# Non-interactive installation
curl -fsSL https://raw.githubusercontent.com/dyluth/claude-reactor/main/install.sh | bash -s -- --force
```

## Manual Installation

### 1. Download Binary

Visit the [Releases page](https://github.com/dyluth/claude-reactor/releases) and download the appropriate binary for your platform:

- **Linux x86_64**: `claude-reactor-v0.1.0-linux-amd64`
- **Linux ARM64**: `claude-reactor-v0.1.0-linux-arm64`
- **macOS x86_64**: `claude-reactor-v0.1.0-darwin-amd64`
- **macOS ARM64**: `claude-reactor-v0.1.0-darwin-arm64`

### 2. Install Binary

```bash
# Download (example for Linux x86_64)
curl -fsSL -O https://github.com/dyluth/claude-reactor/releases/download/v0.1.0/claude-reactor-v0.1.0-linux-amd64

# Verify checksum (recommended)
curl -fsSL -O https://github.com/dyluth/claude-reactor/releases/download/v0.1.0/claude-reactor-v0.1.0-linux-amd64.sha256
sha256sum -c claude-reactor-v0.1.0-linux-amd64.sha256

# Make executable and install
chmod +x claude-reactor-v0.1.0-linux-amd64
mv claude-reactor-v0.1.0-linux-amd64 ~/.local/bin/claude-reactor
```

### 3. Add to PATH

Ensure `~/.local/bin` is in your PATH:

```bash
# Add to your shell profile (~/.bashrc, ~/.zshrc, etc.)
export PATH="$PATH:$HOME/.local/bin"

# Reload your shell or run:
source ~/.bashrc  # or ~/.zshrc
```

### 4. Verify Installation

```bash
claude-reactor --version
claude-reactor --help
```

## Prerequisites

### Docker (Required)

Claude-reactor requires Docker for container functionality:

```bash
# Install Docker Desktop
# Visit: https://docs.docker.com/get-docker/

# Verify Docker installation
docker --version
docker run hello-world
```

### Platform Support

- **Operating Systems**: Linux, macOS
- **Architectures**: x86_64 (amd64), ARM64 (aarch64)
- **Shell**: bash, zsh, fish (any POSIX-compatible shell)

## Troubleshooting

### Common Issues

**1. Command not found**
```bash
# Check if binary is in PATH
echo $PATH | tr ':' '\n' | grep -q "$HOME/.local/bin" && echo "In PATH" || echo "Not in PATH"

# Add to PATH temporarily
export PATH="$PATH:$HOME/.local/bin"
```

**2. Permission denied**
```bash
# Make binary executable
chmod +x ~/.local/bin/claude-reactor
```

**3. Docker not found**
```bash
# Install Docker Desktop
# macOS: brew install --cask docker
# Linux: Follow distribution-specific instructions
```

**4. Certificate/TLS errors**
```bash
# Update CA certificates
# Ubuntu/Debian: sudo apt-get update && sudo apt-get install ca-certificates
# macOS: brew install ca-certificates
```

### Getting Help

- **Documentation**: [README.md](../README.md)
- **Issues**: [GitHub Issues](https://github.com/dyluth/claude-reactor/issues)
- **Discussions**: [GitHub Discussions](https://github.com/dyluth/claude-reactor/discussions)

## Development Installation

For development or building from source:

```bash
# Clone repository
git clone https://github.com/dyluth/claude-reactor.git
cd claude-reactor

# Build and install
make build-local
sudo mv claude-reactor /usr/local/bin/

# Or install to user directory
mv claude-reactor ~/.local/bin/
```

## Uninstallation

```bash
# Remove binary
rm ~/.local/bin/claude-reactor

# Remove configuration (optional)
rm -rf ~/.claude-reactor

# Remove containers (optional)
docker ps -a --format '{{.Names}}' | grep '^claude-agent' | xargs -r docker rm -f
docker images --format '{{.Repository}}:{{.Tag}}' | grep '^claude-reactor' | xargs -r docker rmi -f
```