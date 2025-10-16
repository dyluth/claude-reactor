#!/usr/bin/env bash

# Integration test for mount settings
# Tests that the Go CLI correctly handles mount configurations

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

log_info() { echo "[INFO] $1"; }
log_success() { echo "[PASS] $1"; }
log_failure() { echo "[FAIL] $1"; }

# Test the mount settings functionality with the new Go CLI
test_mount_settings_integration() {
    local test_dir=$(mktemp -d)
    cd "$test_dir"
    
    # Create test directories
    mkdir -p test-mount-dir
    echo "test" > test-mount-dir/file.txt
    mkdir -p other-dir
    echo "other" > other-dir/file.txt
    
    # Create a simple Go project to trigger Go variant
    echo 'module test-project

go 1.21' > go.mod
    
    # Test that the Go CLI can show config (should work without errors)
    if ! "$PROJECT_ROOT/claude-reactor" config show > /dev/null 2>&1; then
        echo "Failed to run config show command"
        cd / && rm -rf "$test_dir"
        return 1
    fi
    
    # Test that the Go CLI can validate config 
    if ! "$PROJECT_ROOT/claude-reactor" config validate > /dev/null 2>&1; then
        echo "Failed to run config validate command"
        cd / && rm -rf "$test_dir"
        return 1
    fi
    
    # Test that mounts can be specified via command line (this would be used at runtime)
    # We test this by checking that the CLI accepts mount flags without errors
    if ! timeout 5 "$PROJECT_ROOT/claude-reactor" run --variant base --mount "$test_dir/test-mount-dir" --no-mounts --help > /dev/null 2>&1; then
        echo "Failed to parse mount flags correctly"
        cd / && rm -rf "$test_dir"
        return 1
    fi
    
    cd / && rm -rf "$test_dir"
    return 0
}

# Run the test
main() {
    log_info "Running mount settings integration test..."
    
    if test_mount_settings_integration; then
        log_success "Mount settings integration test"
    else
        log_failure "Mount settings integration test"
        exit 1
    fi
    
    log_info "Integration test completed successfully"
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi