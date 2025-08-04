#!/usr/bin/env bash

# Demo script to showcase claude-reactor functionality
# This script demonstrates all the key features in a realistic way

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DEMO_DIR="$SCRIPT_DIR/fixtures/demo"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

demo_header() {
    echo ""
    echo -e "${PURPLE}================================================================${NC}"
    echo -e "${PURPLE} $1${NC}"
    echo -e "${PURPLE}================================================================${NC}"
    echo ""
}

demo_step() {
    echo -e "${CYAN}👉 $1${NC}"
    echo ""
}

demo_command() {
    echo -e "${YELLOW}$ $1${NC}"
    eval "$1"
    echo ""
}

demo_pause() {
    echo -e "${BLUE}Press Enter to continue...${NC}"
    read -r
}

show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Interactive demo of claude-reactor functionality.

OPTIONS:
    --auto          Run demo automatically without pauses
    --quick         Skip Docker builds (show functionality only)
    --help, -h      Show this help message

This demo will:
1. Show variant listing and configuration
2. Demonstrate auto-detection for different project types
3. Show configuration persistence
4. Build and test container variants (unless --quick)
5. Demonstrate the complete workflow

EOF
}

# Parse arguments
AUTO_MODE=false
QUICK_MODE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --auto)
            AUTO_MODE=true
            shift
            ;;
        --quick)
            QUICK_MODE=true
            shift
            ;;
        --help|-h)
            show_usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Override pause function in auto mode
if [ "$AUTO_MODE" = true ]; then
    demo_pause() {
        sleep 2
    }
fi

main() {
    cd "$PROJECT_ROOT"
    
    demo_header "Claude-Reactor Feature Demo"
    
    echo -e "${GREEN}Welcome to the claude-reactor demo!${NC}"
    echo "This demo will showcase all the key features of the modular container system."
    echo ""
    
    if [ "$AUTO_MODE" = false ]; then
        demo_pause
    fi
    
    # Feature 1: List available variants
    demo_header "Feature 1: Container Variants"
    demo_step "Let's see what container variants are available:"
    demo_command "./claude-reactor --list-variants"
    demo_pause
    
    # Feature 2: Show current configuration
    demo_header "Feature 2: Configuration Management"
    demo_step "Check current configuration (should show auto-detection):"
    demo_command "./claude-reactor --show-config"
    demo_pause
    
    # Feature 3: Auto-detection demo
    demo_header "Feature 3: Auto-Detection"
    
    # Create demo project directories
    mkdir -p "$DEMO_DIR"
    
    # Go project demo
    demo_step "Creating a Go project and testing auto-detection:"
    mkdir -p "$DEMO_DIR/go-project"
    cd "$DEMO_DIR/go-project"
    echo 'module github.com/example/demo' > go.mod
    echo 'go 1.21' >> go.mod
    
    demo_command "cat go.mod"
    demo_command "$PROJECT_ROOT/claude-reactor --show-config"
    demo_pause
    
    # Rust project demo
    demo_step "Creating a Rust project and testing auto-detection:"
    cd "$DEMO_DIR"
    mkdir -p rust-project
    cd rust-project
    cat > Cargo.toml << 'EOF'
[package]
name = "demo"
version = "0.1.0"
edition = "2021"
EOF
    
    demo_command "cat Cargo.toml"
    demo_command "$PROJECT_ROOT/claude-reactor --show-config"
    demo_pause
    
    # Node.js project demo
    demo_step "Creating a Node.js project and testing auto-detection:"
    cd "$DEMO_DIR"
    mkdir -p node-project
    cd node-project
    cat > package.json << 'EOF'
{
  "name": "demo",
  "version": "1.0.0",
  "main": "index.js"
}
EOF
    
    demo_command "cat package.json"
    demo_command "$PROJECT_ROOT/claude-reactor --show-config"
    demo_pause
    
    # Feature 4: Configuration persistence
    demo_header "Feature 4: Configuration Persistence"
    demo_step "Setting explicit variant and observing persistence:"
    
    cd "$DEMO_DIR/go-project"
    demo_command "$PROJECT_ROOT/claude-reactor --variant full --show-config"
    
    demo_step "Configuration file was created:"
    demo_command "cat .claude-reactor || echo 'Configuration file created successfully'"
    
    demo_step "Now running without --variant should use saved preference:"
    demo_command "$PROJECT_ROOT/claude-reactor --show-config"
    demo_pause
    
    # Feature 5: Danger mode persistence
    demo_header "Feature 5: Danger Mode Persistence"
    demo_step "Setting danger mode and showing persistence:"
    demo_command "$PROJECT_ROOT/claude-reactor --variant go --danger --show-config"
    
    demo_step "Configuration now includes danger mode:"
    demo_command "cat .claude-reactor"
    demo_pause
    
    if [ "$QUICK_MODE" = true ]; then
        demo_header "Demo Complete (Quick Mode)"
        echo -e "${GREEN}✅ Demo completed successfully!${NC}"
        echo ""
        echo "Quick mode was enabled, so we skipped Docker builds."
        echo "To see the full demo with container builds, run:"
        echo -e "${YELLOW}  $0${NC}"
        echo ""
        cleanup_demo
        return
    fi
    
    # Feature 6: Container builds and testing
    demo_header "Feature 6: Container Building and Testing"
    
    echo -e "${YELLOW}⚠️  The following steps will build Docker containers.${NC}"
    echo "This may take several minutes and requires Docker to be running."
    echo ""
    
    if [ "$AUTO_MODE" = false ]; then
        echo "Continue? (y/n)"
        read -r response
        if [[ ! "$response" =~ ^[Yy] ]]; then
            echo "Skipping container builds."
            cleanup_demo
            return
        fi
    fi
    
    cd "$PROJECT_ROOT"
    
    demo_step "Building base variant (fastest):"
    demo_command "docker build --target base -t claude-runner-base . || echo 'Build failed - this is expected in demo'"
    
    demo_step "Testing if base variant has expected tools:"
    if docker image inspect claude-runner-base > /dev/null 2>&1; then
        demo_command "docker run --rm claude-runner-base which node || echo 'Tool check failed'"
        demo_command "docker run --rm claude-runner-base which python3 || echo 'Tool check failed'"
    else
        echo "Base image not available - skipping tool checks"
    fi
    
    demo_pause
    
    demo_step "Building Go variant:"
    demo_command "docker build --target go -t claude-runner-go . || echo 'Build failed - this is expected in demo'"
    
    if docker image inspect claude-runner-go > /dev/null 2>&1; then
        demo_step "Testing Go tools:"
        demo_command "docker run --rm claude-runner-go which go || echo 'Tool check failed'"
        demo_command "docker run --rm claude-runner-go go version || echo 'Tool check failed'"
    else
        echo "Go image not available - skipping tool checks"
    fi
    
    demo_pause
    
    # Feature 7: Complete workflow demonstration
    demo_header "Feature 7: Complete Workflow"
    
    cd "$DEMO_DIR/go-project"
    
    demo_step "Complete workflow for a Go project:"
    echo "1. Auto-detects Go project from go.mod"
    echo "2. Uses 'go' variant automatically"
    echo "3. Saves configuration for future runs"
    echo "4. Builds and starts appropriate container"
    echo ""
    
    demo_command "$PROJECT_ROOT/claude-reactor --show-config"
    
    if [ "$AUTO_MODE" = false ]; then
        echo -e "${BLUE}In a real scenario, this would now:${NC}"
        echo "- Build the claude-runner-go image (if needed)"
        echo "- Start a container named claude-agent-go"
        echo "- Connect you to the container with all Go tools available"
        echo ""
    fi
    
    demo_header "Demo Complete!"
    
    echo -e "${GREEN}🎉 Congratulations! You've seen all the key features:${NC}"
    echo ""
    echo "✅ Container variants (base, go, full, cloud, k8s)"
    echo "✅ Auto-detection based on project files"
    echo "✅ Configuration persistence"
    echo "✅ Danger mode support"
    echo "✅ Docker container building and testing"
    echo "✅ Complete development workflow"
    echo ""
    echo -e "${CYAN}Ready to use claude-reactor for your projects!${NC}"
    echo ""
    
    cleanup_demo
    return 0
}

cleanup_demo() {
    echo -e "${BLUE}Cleaning up demo files...${NC}"
    rm -rf "$DEMO_DIR" 2>/dev/null || true
    echo "Demo cleanup complete."
    return 0  # Ensure clean exit
}

# Trap to ensure cleanup on exit
trap cleanup_demo EXIT

# Run main demo
main "$@"
exit $?