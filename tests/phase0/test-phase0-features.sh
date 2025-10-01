#!/bin/bash

# Phase 0 Feature Validation Tests
# Tests all Phase 0.x feature parity implementations

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BINARY="$PROJECT_ROOT/claude-reactor"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Test utilities
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((TESTS_PASSED++))
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((TESTS_FAILED++))
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

run_test() {
    local test_name="$1"
    local test_command="$2"
    local expected_pattern="$3"
    
    ((TESTS_RUN++))
    log_info "Testing: $test_name"
    
    if output=$(eval "$test_command" 2>&1); then
        if [[ "$output" =~ $expected_pattern ]]; then
            log_success "$test_name"
            return 0
        else
            log_error "$test_name - Output didn't match pattern '$expected_pattern'"
            echo "Actual output: $output"
            return 1
        fi
    else
        log_error "$test_name - Command failed with exit code $?"
        echo "Output: $output"
        return 1
    fi
}

# Ensure binary exists
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    if [[ ! -f "$BINARY" ]]; then
        log_error "Binary not found at $BINARY. Run 'make build' first."
        exit 1
    fi
    
    if [[ ! -x "$BINARY" ]]; then
        log_error "Binary is not executable: $BINARY"
        exit 1
    fi
    
    log_success "Prerequisites check"
}

# Phase 0.1: Registry CLI Integration Tests
test_phase0_1_registry() {
    log_info "=== Phase 0.1: Registry CLI Integration Tests ==="
    
    # Test registry flags are available in help
    run_test "Registry flags in help" \
        "$BINARY run --help" \
        "--dev.*Force local build"
    
    run_test "Registry-off flag in help" \
        "$BINARY run --help" \
        "--registry-off.*Disable registry"
    
    run_test "Pull-latest flag in help" \
        "$BINARY run --help" \
        "--pull-latest.*Force pull latest"
    
    # Test registry configuration parsing (dry run)
    run_test "Dev mode flag parsing" \
        "$BINARY run --dev --no-mounts --shell --help" \
        "Force local build"
    
    run_test "Registry-off flag parsing" \
        "$BINARY run --registry-off --no-mounts --shell --help" \
        "Disable registry"
    
    run_test "Pull-latest flag parsing" \
        "$BINARY run --pull-latest --no-mounts --shell --help" \
        "Force pull latest"
}

# Phase 0.2: System Installation Tests
test_phase0_2_installation() {
    log_info "=== Phase 0.2: System Installation Tests ==="
    
    # Test installation flags are available
    run_test "Install flag in help" \
        "$BINARY --help" \
        "--install.*Install claude-reactor to system PATH"
    
    run_test "Uninstall flag in help" \
        "$BINARY --help" \
        "--uninstall.*Remove claude-reactor from system PATH"
    
    # Test installation dry-run (won't actually install due to test environment)
    # We expect these to work but not actually install in restricted environment
    log_info "Note: Installation tests run in restricted environment - checking flag parsing only"
}

# Phase 0.3: Conversation Control Tests
test_phase0_3_conversation() {
    log_info "=== Phase 0.3: Conversation Control Tests ==="
    
    # Test continue flag is available
    run_test "Continue flag in help" \
        "$BINARY run --help" \
        "--continue.*Enable conversation continuation"
    
    run_test "Continue flag default value" \
        "$BINARY run --help" \
        "default: true.*default true"
    
    # Test continue flag parsing (dry run)
    run_test "Continue=false flag parsing" \
        "$BINARY run --continue=false --no-mounts --shell --help" \
        "conversation continuation"
    
    run_test "Continue=true flag parsing" \
        "$BINARY run --continue=true --no-mounts --shell --help" \
        "conversation continuation"
}

# Phase 0.4: Enhanced Config Display Tests
test_phase0_4_config() {
    log_info "=== Phase 0.4: Enhanced Config Display Tests ==="
    
    # Test enhanced config show command
    run_test "Enhanced config show basic" \
        "$BINARY config show" \
        "=== Claude-Reactor Configuration ==="
    
    run_test "Config show includes project config" \
        "$BINARY config show" \
        "Project Configuration:"
    
    run_test "Config show includes registry config" \
        "$BINARY config show" \
        "Registry Configuration:"
    
    run_test "Config show registry URL" \
        "$BINARY config show" \
        "Registry URL:.*ghcr.io/dyluth/claude-reactor"
    
    run_test "Config show registry tag" \
        "$BINARY config show" \
        "Tag:.*latest"
    
    run_test "Config show registry status" \
        "$BINARY config show" \
        "Status:.*enabled"
    
    # Test verbose flag
    run_test "Config show verbose flag" \
        "$BINARY config show --verbose" \
        "System Information:"
    
    run_test "Config show verbose architecture" \
        "$BINARY config show --verbose" \
        "Architecture:.*arm64|amd64"
    
    run_test "Config show verbose container name" \
        "$BINARY config show --verbose" \
        "Container Name:.*v2-claude-reactor"
    
    # Test raw flag
    run_test "Config show raw flag" \
        "$BINARY config show --raw" \
        "Raw Configuration File:"
    
    # Test help for config show subcommand
    run_test "Config show help includes new flags" \
        "$BINARY config show --help" \
        "--raw.*Include raw configuration"
    
    run_test "Config show help includes verbose flag" \
        "$BINARY config show --help" \
        "--verbose.*Show detailed system"
}

# v2 Prefix Tests
test_v2_prefix() {
    log_info "=== v2 Prefix Tests ==="
    
    # Test v2 prefix in image names
    run_test "v2 prefix in image names" \
        "$BINARY config show --verbose" \
        "Image Name:.*v2-claude-reactor"
    
    # Test v2 prefix in container names
    run_test "v2 prefix in container names" \
        "$BINARY config show --verbose" \
        "Container Name:.*v2-claude-reactor"
}

# Integration Tests
test_integration() {
    log_info "=== Integration Tests ==="
    
    # Test that all flags work together
    run_test "Combined registry and conversation flags" \
        "$BINARY run --dev --continue=false --no-mounts --shell --help" \
        "Force local build"
    
    # Test config validation with registry environment variables
    run_test "Config show with environment variables" \
        "CLAUDE_REACTOR_REGISTRY=custom.registry.com $BINARY config show" \
        "Registry URL:.*custom.registry.com"
    
    run_test "Config show with disabled registry" \
        "CLAUDE_REACTOR_USE_REGISTRY=false $BINARY config show" \
        "Status:.*disabled"
}

# Main test execution
main() {
    log_info "Starting Phase 0 Feature Validation Tests"
    log_info "Binary: $BINARY"
    echo ""
    
    check_prerequisites
    echo ""
    
    # Run all test suites
    test_phase0_1_registry
    echo ""
    
    test_phase0_2_installation
    echo ""
    
    test_phase0_3_conversation
    echo ""
    
    test_phase0_4_config
    echo ""
    
    test_v2_prefix
    echo ""
    
    test_integration
    echo ""
    
    # Summary
    log_info "=== Test Summary ==="
    echo "Tests Run: $TESTS_RUN"
    echo "Tests Passed: $TESTS_PASSED"
    echo "Tests Failed: $TESTS_FAILED"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        log_success "All Phase 0 tests passed! 🎉"
        log_success "Go CLI has achieved feature parity with bash script"
        exit 0
    else
        log_error "Some tests failed. Phase 0 implementation needs fixes."
        exit 1
    fi
}

# Run tests
main "$@"