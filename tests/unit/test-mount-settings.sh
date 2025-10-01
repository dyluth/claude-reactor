#!/usr/bin/env bash

# Unit tests for Claude mount settings generation
# Tests the update_claude_settings_for_mounts function

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TEST_DIR="$SCRIPT_DIR/../fixtures/unit/mount-settings"

# Import test functions
if [[ -n "${log_info:-}" ]]; then
    # Functions are available from test-runner
    true
else
    # Define basic test functions for standalone execution
    log_info() { echo "[INFO] $1"; }
    log_success() { echo "[PASS] $1"; }
    log_failure() { echo "[FAIL] $1"; }
    run_test() {
        local test_name="$1"
        local test_func="$2"
        echo "Running: $test_name"
        if $test_func; then
            log_success "$test_name"
        else
            log_failure "$test_name"
            exit 1
        fi
    }
fi

# Setup test environment
setup_test_env() {
    rm -rf "$TEST_DIR"
    mkdir -p "$TEST_DIR"
    cd "$TEST_DIR"
    
    # Source the functions we need to test from the old claude-reactor (legacy)
    # Extract just the function without executing the script
    sed -n '/^update_claude_settings_for_mounts()/,/^}$/p' "$PROJECT_ROOT/old/claude-reactor" > test-functions.sh
    source test-functions.sh
    
    # Mock log functions
    log_verbose() { echo "[VERBOSE] $*"; }
    log_warning() { echo "[WARNING] $*"; }
    log_info() { echo "[INFO] $*"; }
}

# Test: Create new settings file
test_create_new_settings() {
    setup_test_env
    
    # Set up test data
    MOUNT_PATHS=("test-dir" "../other-dir")
    
    # Run the function
    update_claude_settings_for_mounts
    
    # Check that settings file was created
    if [ ! -f ".claude/settings.local.json" ]; then
        echo "Settings file was not created"
        return 1
    fi
    
    # Check content using jq (sort both since unique sorts)
    local expected='["/mnt/other-dir","/mnt/test-dir"]'
    local actual=$(jq -c '.additionalDirectories | sort' .claude/settings.local.json)
    
    if [ "$actual" != "$expected" ]; then
        echo "Expected: $expected"
        echo "Actual: $actual"
        return 1
    fi
    
    return 0
}

# Test: Merge with existing settings
test_merge_existing_settings() {
    setup_test_env
    
    # Create existing settings file with other properties
    mkdir -p .claude
    cat > .claude/settings.local.json << 'EOF'
{
  "existingSetting": "preserved",
  "additionalDirectories": ["/existing/path"],
  "otherArray": ["item1", "item2"]
}
EOF
    
    # Set up test data
    MOUNT_PATHS=("new-mount")
    
    # Run the function
    update_claude_settings_for_mounts
    
    # Check that all settings are preserved
    local existing_setting=$(jq -r '.existingSetting' .claude/settings.local.json)
    if [ "$existing_setting" != "preserved" ]; then
        echo "Existing setting was not preserved: $existing_setting"
        return 1
    fi
    
    local other_array=$(jq -c '.otherArray' .claude/settings.local.json)
    if [ "$other_array" != '["item1","item2"]' ]; then
        echo "Other array was not preserved: $other_array"
        return 1
    fi
    
    # Check that directories were merged and deduplicated
    local dirs=$(jq -c '.additionalDirectories | sort' .claude/settings.local.json)
    local expected='["/existing/path","/mnt/new-mount"]'
    if [ "$dirs" != "$expected" ]; then
        echo "Directory merge failed. Expected: $expected, Got: $dirs"
        return 1
    fi
    
    return 0
}

# Test: Handle duplicate mounts
test_deduplicate_mounts() {
    setup_test_env
    
    # Create existing settings
    mkdir -p .claude
    echo '{"additionalDirectories": ["/mnt/existing"]}' > .claude/settings.local.json
    
    # Set up test data with duplicate
    MOUNT_PATHS=("existing" "new-mount")
    
    # Run the function
    update_claude_settings_for_mounts
    
    # Check that duplicates are removed
    local dirs=$(jq -c '.additionalDirectories | sort' .claude/settings.local.json)
    local expected='["/mnt/existing","/mnt/new-mount"]'
    if [ "$dirs" != "$expected" ]; then
        echo "Deduplication failed. Expected: $expected, Got: $dirs"
        return 1
    fi
    
    return 0
}

# Test: No mounts (should not modify settings)
test_no_mounts() {
    setup_test_env
    
    # Create existing settings
    mkdir -p .claude
    echo '{"existingSetting": "test"}' > .claude/settings.local.json
    local original_content=$(cat .claude/settings.local.json)
    
    # Set up test data with no mounts
    MOUNT_PATHS=()
    
    # Run the function
    update_claude_settings_for_mounts
    
    # Check that file was not modified
    local current_content=$(cat .claude/settings.local.json)
    if [ "$current_content" != "$original_content" ]; then
        echo "Settings were modified when no mounts provided"
        echo "Original: $original_content"
        echo "Current: $current_content"
        return 1
    fi
    
    return 0
}

# Test: Invalid JSON handling
test_invalid_json_handling() {
    setup_test_env
    
    # Create invalid JSON file
    mkdir -p .claude
    echo 'invalid json{' > .claude/settings.local.json
    
    # Set up test data
    MOUNT_PATHS=("test-mount")
    
    # Run the function
    update_claude_settings_for_mounts
    
    # Check that new valid file was created
    local dirs=$(jq -c '.additionalDirectories' .claude/settings.local.json)
    local expected='["/mnt/test-mount"]'
    if [ "$dirs" != "$expected" ]; then
        echo "Invalid JSON not handled correctly. Expected: $expected, Got: $dirs"
        return 1
    fi
    
    return 0
}

# Run all tests
main() {
    log_info "Running mount settings unit tests..."
    
    run_test "Create new settings file" test_create_new_settings
    run_test "Merge with existing settings" test_merge_existing_settings  
    run_test "Deduplicate mount paths" test_deduplicate_mounts
    run_test "No mounts should not modify settings" test_no_mounts
    run_test "Invalid JSON handling" test_invalid_json_handling
    
    log_info "Mount settings unit tests completed successfully"
}

# Run tests if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi