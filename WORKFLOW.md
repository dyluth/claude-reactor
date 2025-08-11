# Claude-Reactor Workflow Guide

This document explains the improved division of responsibilities between the Makefile and claude-reactor script.

## 🎯 **Clear Separation of Concerns**

### **claude-reactor** (Runtime Development Tool)
**Purpose**: Daily development workflow and smart container management

**Responsibilities**:
- Container variant selection and auto-detection
- Registry-first image management with local fallback
- Configuration persistence (`.claude-reactor` file)
- Interactive container connection
- Smart state management
- Development-focused user interface

**Usage**:
```bash
# Smart development workflow
./claude-reactor                    # Auto-detect variant, pull from registry
./claude-reactor --variant go       # Set and save Go preference
./claude-reactor --danger           # Quick dangerous mode
./claude-reactor --dev              # Force local build (development)
./claude-reactor --pull-latest      # Force fresh registry pull
./claude-reactor --show-config      # Check current settings
```

### **Makefile** (Build Automation & CI)
**Purpose**: Build automation, testing, CI/CD, and project management

**Responsibilities**:
- Docker image building
- Container registry operations (push/pull)
- Test orchestration
- Code quality (linting, formatting)
- CI/CD pipelines
- Project maintenance tasks
- Delegating to claude-reactor when appropriate

**Usage**:
```bash
# Build and test automation
make build-all                      # Build all variants
make test                           # Run test suite
make ci-full                        # Complete CI pipeline
make clean-all                      # Complete cleanup

# Registry operations
make pull-all                       # Pull core variants from registry
make push-all                       # Build and push to registry
make registry-login                 # Login to GitHub Container Registry
```

## 🔄 **Smart Delegation**

The Makefile now **delegates** to claude-reactor for tasks where the script's intelligence is valuable:

### **Container Management** → claude-reactor
```bash
make run-go          # Calls: ./claude-reactor --variant go
make run             # Calls: ./claude-reactor (smart detection)
make config          # Calls: ./claude-reactor --show-config
make variants        # Calls: ./claude-reactor --list-variants
make clean-containers # Calls: ./claude-reactor --clean
```

### **Pure Build Tasks** → Makefile
```bash
make build-base      # Pure Docker build
make build-all       # Build multiple variants
make test-unit       # Test orchestration
make lint            # Code quality

# Registry management
make pull-all        # Pull from registry
make push-all        # Push to registry  
make registry-login  # Registry authentication
```

## 📋 **Typical Workflows**

### **Developer Daily Workflow**
```bash
# Option 1: Direct script usage (recommended for development)
./claude-reactor --variant go       # First time setup (pulls from registry)
./claude-reactor                    # Subsequent runs (registry-first, local fallback)
./claude-reactor --dev              # Force local build for development
./claude-reactor --pull-latest      # Get newest images

# Option 2: Make targets for convenience
make run-go                         # One-time variant selection
make run                            # Use auto-detection/saved config
make config                         # Check what will be used
```

### **Project Setup & Testing**
```bash
# Setup and validation
make dev-setup                      # Prepare development environment
make test                           # Run complete test suite
make demo                           # Interactive feature demo

# Build pipeline
make build-all                      # Build core variants
make test-full                      # Build + comprehensive testing
make clean-all                      # Complete cleanup
```

### **CI/CD Pipeline**
```bash
# Continuous Integration
make ci-build                       # Build containers for CI
make ci-test                        # Run CI-appropriate tests
make ci-full                        # Complete CI pipeline

# Registry management
make pull-all                       # Pull latest from registry
make push-all                       # Build and push (requires auth)
make registry-login                 # GitHub Container Registry auth

# Advanced analysis
make benchmark                      # Container size analysis
make security-scan                  # Security vulnerability scanning
```

## 🎉 **Benefits of This Approach**

### **No Duplication**
- Container management logic stays in claude-reactor
- Build automation stays in Makefile
- Each tool has a clear, focused purpose

### **Best of Both Worlds**
- **Make users** get familiar, standardized commands
- **Script users** get intelligent, stateful behavior
- **CI systems** get reproducible, stateless builds
- **Developers** get smart auto-detection and persistence

### **Flexibility**
- Use claude-reactor directly for development
- Use Makefile for build automation and CI
- Mix and match as needed
- Both tools complement each other

### **Maintainability**
- Single source of truth for container logic (claude-reactor)
- Single source of truth for build automation (Makefile)
- Clear interface boundaries
- Easy to extend either tool independently

## 📚 **Quick Reference**

| Task | Use | Command |
|------|-----|---------|
| Daily development | claude-reactor | `./claude-reactor` |
| Force specific variant | either | `./claude-reactor --variant go` or `make run-go` |
| Force local build | claude-reactor | `./claude-reactor --dev` |
| Pull latest images | claude-reactor | `./claude-reactor --pull-latest` |
| Check configuration | either | `./claude-reactor --show-config` or `make config` |
| Build containers | Makefile | `make build-all` |
| Pull from registry | Makefile | `make pull-all` |
| Push to registry | Makefile | `make push-all` |
| Registry login | Makefile | `make registry-login` |
| Run tests | Makefile | `make test` |
| Complete cleanup | Makefile | `make clean-all` |
| Interactive demo | either | `./tests/demo.sh` or `make demo` |
| CI pipeline | Makefile | `make ci-full` |

This design provides the **professional polish of Make** with the **intelligence of claude-reactor**, eliminating duplication while maximizing the strengths of each tool! 🚀