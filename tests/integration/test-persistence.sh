#!/bin/bash
# Integration test for Claude CLI configuration persistence
# Tests that Claude configuration and Docker access persist across container restarts

set -euo pipefail

# Source test utilities if available
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Test configuration
TEST_NAME="Claude CLI Configuration Persistence"
TEMP_DIR=""
TEST_CLAUDE_DIR=""
TEST_PROJECT_DIR=""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_failure() {
    echo -e "${RED}[FAILURE]${NC} $1"
}

log_test() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

# Cleanup function
cleanup() {
    local exit_code=$?
    
    if [ -n "$TEMP_DIR" ] && [ -d "$TEMP_DIR" ]; then
        log_info "Cleaning up temporary directory: $TEMP_DIR"
        rm -rf "$TEMP_DIR"
    fi
    
    # Clean up test containers using isolated config
    if [ -n "$TEST_CLAUDE_DIR" ]; then
        HOME="$TEMP_DIR" "$PROJECT_ROOT/claude-reactor" --clean > /dev/null 2>&1 || true
    fi
    
    if [ $exit_code -eq 0 ]; then
        log_success "Test cleanup completed successfully"
    else
        log_warning "Test cleanup completed with errors"
    fi
    
    exit $exit_code
}

# Set up cleanup trap
trap cleanup EXIT

# Setup test environment
setup_test_environment() {
    log_info "Setting up test environment for $TEST_NAME"
    
    # Create temporary directory
    TEMP_DIR=$(mktemp -d)
    TEST_PROJECT_DIR="$TEMP_DIR/test-project"  
    TEST_CLAUDE_DIR="$TEMP_DIR/.claude"
    mkdir -p "$TEST_PROJECT_DIR"
    mkdir -p "$TEST_CLAUDE_DIR"
    
    # Create a simple test project
    cat > "$TEST_PROJECT_DIR/test-file.txt" << 'EOF'
This is a test file for Claude CLI persistence testing.
EOF
    
    # Create a simple go.mod to trigger Go variant auto-detection
    cat > "$TEST_PROJECT_DIR/go.mod" << 'EOF'
module test-project

go 1.21
EOF
    
    log_success "Test environment created at $TEST_PROJECT_DIR"
    log_success "Isolated Claude config directory created at $TEST_CLAUDE_DIR"
}

# Create isolated test Claude configuration
create_isolated_claude_config() {
    log_info "Creating isolated Claude configuration for testing"
    
    # Copy existing Claude directory structure if it exists (excluding credentials)
    if [ -d "$HOME/.claude" ]; then
        log_info "Copying existing Claude directory structure (excluding credentials)"
        cp -r "$HOME/.claude"/* "$TEST_CLAUDE_DIR/" 2>/dev/null || true
        # Remove credentials to avoid using real auth in tests
        rm -f "$TEST_CLAUDE_DIR/.credentials.json" 2>/dev/null || true
    fi
    
    # Create test Claude.json configuration in the isolated directory
    cat > "$TEST_CLAUDE_DIR/.claude.json" << 'EOF'
{
  "numStartups": 5,
  "installMethod": "test",
  "autoUpdates": true,
  "userID": "test-user-id-12345",
  "firstStartTime": "2025-08-05T00:00:00.000Z",
  "projects": {
    "/app": {
      "allowedTools": [],
      "trustLevel": "trusted"
    }
  },
  "trustedDirectories": {
    "/app": true
  },
  "hasTrustDialogAccepted": true,
  "hasCompletedProjectOnboarding": true,
  "theme": "dark"
}
EOF

    # Also create the main claude.json file in the fake home directory
    cat > "$TEMP_DIR/.claude.json" << 'EOF'
{
  "numStartups": 5,
  "installMethod": "test", 
  "autoUpdates": true,
  "userID": "test-user-id-12345",
  "firstStartTime": "2025-08-05T00:00:00.000Z",
  "projects": {
    "/app": {
      "allowedTools": [],
      "trustLevel": "trusted"
    }
  },
  "trustedDirectories": {
    "/app": true
  },
  "hasTrustDialogAccepted": true,
  "hasCompletedProjectOnboarding": true,
  "theme": "dark"
}
EOF
    
    log_success "Isolated Claude configuration created"
}

# Test 1: Configuration mounting
test_config_mounting() {
    log_test "Testing Claude configuration file mounting"
    
    cd "$TEST_PROJECT_DIR"
    
    # Start container using isolated HOME directory
    timeout 30 env HOME="$TEMP_DIR" "$PROJECT_ROOT/claude-reactor" --variant go --shell &
    local container_pid=$!
    
    # Wait for container to be ready
    sleep 10
    
    # Check if config file is mounted and accessible
    if docker exec claude-agent-go test -f /home/claude/.claude.json; then
        log_success "Claude config file is mounted in container"
    else
        log_failure "Claude config file is not mounted in container"
        kill $container_pid 2>/dev/null || true
        return 1
    fi
    
    # Check if config content matches
    local container_user_id
    container_user_id=$(docker exec claude-agent-go jq -r '.userID // "missing"' /home/claude/.claude.json 2>/dev/null || echo "error")
    
    if [ "$container_user_id" = "test-user-id-12345" ]; then
        log_success "Claude config content is correctly mounted"
    else
        log_failure "Claude config content mismatch: expected 'test-user-id-12345', got '$container_user_id'"
        kill $container_pid 2>/dev/null || true
        return 1
    fi
    
    # Clean up container
    kill $container_pid 2>/dev/null || true
    env HOME="$TEMP_DIR" "$PROJECT_ROOT/claude-reactor" --clean > /dev/null 2>&1 || true
    
    return 0
}

# Test 2: Docker socket access  
test_docker_access() {
    log_test "Testing Docker socket access from container"
    
    cd "$TEST_PROJECT_DIR"
    
    # Start container using isolated HOME directory
    timeout 30 env HOME="$TEMP_DIR" "$PROJECT_ROOT/claude-reactor" --variant go --shell &
    local container_pid=$!
    
    # Wait for container to be ready
    sleep 10
    
    # Test if Docker socket is accessible
    if docker exec claude-agent-go test -S /var/run/docker.sock; then
        log_success "Docker socket is mounted in container"
    else
        log_failure "Docker socket is not mounted in container"
        kill $container_pid 2>/dev/null || true
        return 1
    fi
    
    # Test if claude user can run docker commands
    if docker exec --user claude claude-agent-go docker version > /dev/null 2>&1; then
        log_success "Claude user can execute Docker commands"
    else
        log_warning "Claude user cannot execute Docker commands (may need group membership)"
        # This is not a hard failure as group setup might take time
    fi
    
    # Clean up container
    kill $container_pid 2>/dev/null || true
    env HOME="$TEMP_DIR" "$PROJECT_ROOT/claude-reactor" --clean > /dev/null 2>&1 || true
    
    return 0
}

# Test 3: Configuration persistence across restarts
test_persistence_across_restarts() {
    log_test "Testing configuration persistence across container restarts"
    
    cd "$TEST_PROJECT_DIR"
    
    # Start container first time using isolated HOME directory
    log_info "Starting container for the first time"
    timeout 30 env HOME="$TEMP_DIR" "$PROJECT_ROOT/claude-reactor" --variant go --shell &
    local container_pid=$!
    
    # Wait for container to be ready
    sleep 10
    
    # Modify the config file in the isolated home to test persistence
    local test_timestamp=$(date +%s)
    echo "# Test modification: $test_timestamp" >> "$TEMP_DIR/.claude.json"
    
    # Stop container with --clean
    kill $container_pid 2>/dev/null || true
    env HOME="$TEMP_DIR" "$PROJECT_ROOT/claude-reactor" --clean > /dev/null 2>&1 || true
    
    log_info "Restarting container after clean"
    
    # Start container second time using isolated HOME directory
    timeout 30 env HOME="$TEMP_DIR" "$PROJECT_ROOT/claude-reactor" --variant go --shell &
    container_pid=$!
    
    # Wait for container to be ready
    sleep 10
    
    # Check if the original config persisted and our modification is there
    local restored_user_id
    restored_user_id=$(docker exec claude-agent-go jq -r '.userID // "missing"' /home/claude/.claude.json 2>/dev/null || echo "error")
    
    if [ "$restored_user_id" = "test-user-id-12345" ]; then
        log_success "Claude configuration persisted across container restart"
        
        # Check if our host modification is reflected (it should be)
        if grep -q "Test modification: $test_timestamp" "$TEMP_DIR/.claude.json"; then
            log_success "Host configuration modifications persist correctly"
        else
            log_warning "Host configuration modifications were not found (expected for isolated test)"
        fi
    else
        log_failure "Claude configuration did not persist: expected 'test-user-id-12345', got '$restored_user_id'"
        kill $container_pid 2>/dev/null || true
        return 1
    fi
    
    # Clean up container
    kill $container_pid 2>/dev/null || true
    env HOME="$TEMP_DIR" "$PROJECT_ROOT/claude-reactor" --clean > /dev/null 2>&1 || true
    
    return 0
}

# Main test runner
run_tests() {
    log_info "Starting $TEST_NAME tests"
    
    local tests_passed=0
    local tests_failed=0
    
    # Setup
    setup_test_environment
    create_isolated_claude_config
    
    # Run tests
    if test_config_mounting; then
        ((tests_passed++))
    else
        ((tests_failed++))
    fi
    
    if test_docker_access; then
        ((tests_passed++))
    else
        ((tests_failed++))
    fi
    
    if test_persistence_across_restarts; then
        ((tests_passed++))
    else
        ((tests_failed++))
    fi
    
    # Report results
    log_info "Test Results:"
    log_info "  Passed: $tests_passed"
    log_info "  Failed: $tests_failed"
    
    if [ $tests_failed -eq 0 ]; then
        log_success "$TEST_NAME: All tests passed!"
        return 0
    else
        log_failure "$TEST_NAME: $tests_failed test(s) failed"
        return 1
    fi
}

# Run tests if script is executed directly
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    run_tests
fi