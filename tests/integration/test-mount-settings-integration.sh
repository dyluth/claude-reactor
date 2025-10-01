#!/usr/bin/env bash

# Integration test for mount settings generation with actual claude-reactor script
# Tests the complete flow from command line to settings file creation

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TEST_DIR="$(mktemp -d)"
CLAUDE_REACTOR="$PROJECT_ROOT/claude-reactor"

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
            return 1
        fi
    }
fi

# Cleanup function
cleanup() {
    cd /
    rm -rf "$TEST_DIR"
}
trap cleanup EXIT

# Test: End-to-end mount settings generation
test_e2e_mount_settings() {
    cd "$TEST_DIR"
    
    # Create test mount directory
    mkdir -p test-data
    echo "test content" > test-data/file.txt
    
    # Create basic config file
    echo "variant=base" > .claude-reactor
    
    # Run claude-reactor with mount (using --show-config to avoid container creation)
    export MOUNT_PATHS_TEST="test-data"
    
    # We need to extract just the settings generation part since full container tests
    # require Docker images. Let's test the mount processing logic instead.
    
    # Create a mock script that calls the mount settings function
    cat > test-mount-function.sh << 'EOF'
#!/usr/bin/env bash
set -e

# Source the claude-reactor functions
source /dev/stdin << 'FUNCTIONS_EOF'
FUNCTIONS_CONTENT_HERE
FUNCTIONS_EOF

# Set up test environment
MOUNT_PATHS=("test-data" "../other-dir")

# Mock log functions
log_verbose() { echo "[VERBOSE] $*"; }
log_warning() { echo "[WARNING] $*"; }  
log_info() { echo "[INFO] $*"; }

# Run the function
update_claude_settings_for_mounts
EOF
    
    # Extract the function from claude-reactor and inject it
    function_content=$(sed -n '/^update_claude_settings_for_mounts()/,/^}$/p' "$CLAUDE_REACTOR")
    sed "s/FUNCTIONS_CONTENT_HERE/$function_content/" test-mount-function.sh > test-mount-function-filled.sh
    chmod +x test-mount-function-filled.sh
    
    # Run the test
    ./test-mount-function-filled.sh
    
    # Verify settings file was created
    if [ ! -f ".claude/settings.local.json" ]; then
        echo "Settings file was not created by integration test"
        return 1
    fi
    
    # Verify content
    local dirs=$(jq -c '.additionalDirectories | sort' .claude/settings.local.json)
    local expected='["/mnt/other-dir","/mnt/test-data"]'
    
    if [ "$dirs" != "$expected" ]; then
        echo "Integration test failed. Expected: $expected, Got: $dirs"
        return 1
    fi
    
    return 0
}

# Test: Settings persist with existing project structure
test_settings_persistence() {
    cd "$TEST_DIR"
    
    # Create existing .claude directory with settings
    mkdir -p .claude
    cat > .claude/settings.local.json << 'EOF'
{
  "projectName": "test-project",
  "theme": "dark", 
  "additionalDirectories": ["/existing/mount"]
}
EOF
    
    # Create the mount function test
    cat > test-persistence.sh << 'EOF'
#!/usr/bin/env bash
set -e

# Source the function (content will be replaced)
FUNCTION_CONTENT_PLACEHOLDER

# Set up test
MOUNT_PATHS=("new-data")
log_verbose() { echo "[VERBOSE] $*"; }
log_warning() { echo "[WARNING] $*"; } 
log_info() { echo "[INFO] $*"; }

# Run function
update_claude_settings_for_mounts
EOF

    # Inject the actual function
    function_content=$(sed -n '/^update_claude_settings_for_mounts()/,/^}$/p' "$CLAUDE_REACTOR")
    echo "$function_content" > extracted-function.sh
    echo "" >> extracted-function.sh  # Add newline
    cat test-persistence.sh >> extracted-function.sh
    sed -i '' 's/FUNCTION_CONTENT_PLACEHOLDER//g' extracted-function.sh
    chmod +x extracted-function.sh
    
    # Run the test
    ./extracted-function.sh
    
    # Check that original settings were preserved
    local project_name=$(jq -r '.projectName' .claude/settings.local.json)
    if [ "$project_name" != "test-project" ]; then
        echo "Project name not preserved: $project_name"
        return 1
    fi
    
    local theme=$(jq -r '.theme' .claude/settings.local.json)
    if [ "$theme" != "dark" ]; then
        echo "Theme not preserved: $theme" 
        return 1
    fi
    
    # Check that directories were merged
    local dirs=$(jq -c '.additionalDirectories | sort' .claude/settings.local.json)
    local expected='["/existing/mount","/mnt/new-data"]'
    
    if [ "$dirs" != "$expected" ]; then
        echo "Directory merge failed in persistence test. Expected: $expected, Got: $dirs"
        return 1
    fi
    
    return 0
}

# Main test execution
main() {
    log_info "Running mount settings integration tests..."
    
    run_test "End-to-end mount settings generation" test_e2e_mount_settings
    run_test "Settings persistence with existing project" test_settings_persistence
    
    log_info "Mount settings integration tests completed successfully"
}

# Run tests if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi