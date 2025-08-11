# Claude-Reactor Deployment & Release Guide

This document provides comprehensive guidance for deploying, releasing, and maintaining Claude-Reactor with GitHub Container Registry integration.

## 🚀 Quick Deployment

### For Users (Pull from Registry)
```bash
# Clone and run - automatically pulls from registry
git clone https://github.com/dyluth/claude-reactor-backup.git
cd claude-reactor-backup
./claude-reactor
```

### For Developers (Local Development)
```bash
# Clone and build locally
git clone https://github.com/dyluth/claude-reactor-backup.git
cd claude-reactor-backup
./claude-reactor --dev  # Force local build
```

## 🔧 GitHub Container Registry Setup

### Repository Configuration

The project is configured to use:
- **Registry**: `ghcr.io/dyluth/claude-reactor`
- **Variants**: base, go, full, cloud, k8s
- **Architectures**: linux/amd64, linux/arm64
- **Tags**: latest, v0.1.0, dev

### Required GitHub Secrets

No additional secrets required! The GitHub Actions workflow uses the built-in `GITHUB_TOKEN` which automatically has `packages:write` permission.

### First-Time Registry Setup

1. **Enable GitHub Container Registry** (if not already enabled):
   - Go to GitHub → Settings → Developer settings → Personal access tokens
   - Ensure packages scope is available (should be by default)

2. **Repository Settings**:
   - Go to repository → Settings → Actions → General
   - Ensure "Read and write permissions" is enabled for GITHUB_TOKEN
   - This allows the workflow to push to ghcr.io

## 📦 Release Workflow

### Automated Release (Recommended)

1. **Create and Push Tag**:
   ```bash
   # Update version
   echo "v0.1.0" > VERSION
   git add VERSION
   git commit -m "Release v0.1.0: Add registry integration and CI/CD"
   
   # Create and push tag
   git tag v0.1.0
   git push origin main
   git push origin v0.1.0
   ```

2. **GitHub Actions Automatically**:
   - Builds all 5 variants for both architectures (10 images total)
   - Pushes to `ghcr.io/dyluth/claude-reactor-*:v0.1.0`
   - Pushes to `ghcr.io/dyluth/claude-reactor-*:latest`
   - Runs security scans
   - Creates GitHub release (if configured)

### Manual Release (If Needed)

1. **Setup Authentication**:
   ```bash
   # Generate GitHub Personal Access Token with packages:write scope
   echo $GITHUB_TOKEN | docker login ghcr.io -u dyluth --password-stdin
   ```

2. **Build and Push**:
   ```bash
   # Build all variants locally
   make build-extended
   
   # Push to registry
   make push-extended
   ```

## 🔄 CI/CD Pipeline

### GitHub Actions Triggers

The `.github/workflows/build-and-push.yml` workflow triggers on:

1. **Push to main**: Builds and pushes `latest` tags
2. **Create tag `v*`**: Builds and pushes version tags
3. **Pull request**: Builds and tests (no push)
4. **Manual dispatch**: Supports dev builds

### Workflow Features

- **Multi-Architecture**: Builds for AMD64 and ARM64
- **Caching**: Uses GitHub Actions cache for efficiency
- **Security**: Trivy vulnerability scanning
- **Testing**: Integration tests with pulled images
- **Metadata**: Proper OCI labels and versioning

## 🏗️ Architecture Overview

### Registry Structure
```
ghcr.io/dyluth/claude-reactor-base:latest
ghcr.io/dyluth/claude-reactor-base:v0.1.0
ghcr.io/dyluth/claude-reactor-go:latest
ghcr.io/dyluth/claude-reactor-go:v0.1.0
ghcr.io/dyluth/claude-reactor-full:latest
ghcr.io/dyluth/claude-reactor-full:v0.1.0
ghcr.io/dyluth/claude-reactor-cloud:latest
ghcr.io/dyluth/claude-reactor-cloud:v0.1.0
ghcr.io/dyluth/claude-reactor-k8s:latest
ghcr.io/dyluth/claude-reactor-k8s:v0.1.0
```

### Image Naming Convention
- **Format**: `ghcr.io/dyluth/claude-reactor-{variant}:{tag}`
- **Variants**: base, go, full, cloud, k8s
- **Tags**: latest, v{semver}, dev
- **Multi-arch**: Single manifest supports both AMD64 and ARM64

## 🧪 Testing Strategy

### Pre-Release Testing
```bash
# Run complete test suite
make test

# Test registry integration
CLAUDE_REACTOR_USE_REGISTRY=true ./claude-reactor --variant base --verbose

# Test dev mode fallback
./claude-reactor --dev --variant go --verbose
```

### Post-Release Verification
```bash
# Test pull from registry
docker pull ghcr.io/dyluth/claude-reactor-base:v0.1.0

# Test automated pulling
./claude-reactor --pull-latest --variant base
```

## 📊 Monitoring & Maintenance

### Registry Management

**View Registry Images**:
- Go to GitHub → Packages tab
- View `claude-reactor-*` packages
- Check download statistics

**Clean Old Images** (if needed):
```bash
# List images
gh api user/packages/container/claude-reactor-base/versions

# Delete specific version (if needed)
gh api -X DELETE user/packages/container/claude-reactor-base/versions/VERSION_ID
```

### Size Optimization

**Monitor Image Sizes**:
```bash
make benchmark  # Shows local image sizes
```

**Expected Sizes**:
- base: ~500MB
- go: ~800MB  
- full: ~1.2GB
- cloud: ~1.5GB
- k8s: ~1.4GB

## 🔐 Security Considerations

### Automated Scanning
- Trivy security scanning runs on every build
- Results uploaded to GitHub Security tab
- Scans core variants (base, go, full)

### Manual Security Review
```bash
# Run security scan locally
make security-scan

# Check for vulnerabilities
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
  aquasec/trivy image ghcr.io/dyluth/claude-reactor-base:latest
```

## 🚨 Troubleshooting

### Common Issues

**Registry Pull Fails**:
- Fallback to local build is automatic
- Check internet connection
- Verify registry URL in logs

**Build Failures**:
- Check GitHub Actions logs
- Verify multi-stage Dockerfile syntax
- Test locally with `make build-all`

**Permission Issues**:
- Ensure GITHUB_TOKEN has packages:write
- Check repository settings → Actions → Permissions

### Debug Commands
```bash
# Show detailed registry configuration
./claude-reactor --show-config --verbose

# Test registry connectivity
docker pull ghcr.io/dyluth/claude-reactor-base:latest

# Force local build for debugging
./claude-reactor --dev --rebuild --verbose
```

## 📈 Version Management

### Semantic Versioning
- Follow semantic versioning (e.g., v0.1.0, v0.2.0, v1.0.0)
- Update VERSION file before releases
- Tag format: `v{major}.{minor}.{patch}`

### Release Notes Template
```markdown
## Claude-Reactor v0.1.0

### 🚀 New Features
- GitHub Container Registry integration
- Multi-architecture support (AMD64/ARM64)
- Automated CI/CD pipeline

### 🔧 Improvements
- Registry-first behavior with local fallback
- Enhanced Makefile with 30+ targets
- Comprehensive documentation updates

### 🐛 Bug Fixes
- None in this release

### 📦 Container Images
- ghcr.io/dyluth/claude-reactor-*:v0.1.0
- All 5 variants available for both architectures
```

This deployment guide ensures reliable, automated releases with proper monitoring and troubleshooting procedures! 🎯