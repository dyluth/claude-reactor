#!/usr/bin/env bash

# Unit tests for claude-reactor script functions
# Tests the core logic without requiring Docker

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TEST_DIR="$SCRIPT_DIR/../fixtures/unit"

# Source the claude-reactor script to access its functions
# We'll create a test-safe version that doesn't execute main logic
CLAUDE_REACTOR="$PROJECT_ROOT/claude-reactor"

# Create test environment
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

# Import test functions if available
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
        local test_command="$2"
        echo "Running: $test_name"
        if eval "$test_command"; then
            log_success "$test_name"
            return 0
        else
            log_failure "$test_name"
            return 1
        fi
    }
fi

# Create a test-safe version of claude-reactor functions
create_test_functions() {
    cat > test-functions.sh << 'EOF'
#!/usr/bin/env bash

# Test-safe versions of claude-reactor functions
CONFIG_FILE=".claude-reactor"
VERBOSE=false

log_verbose() {
    if [ "$VERBOSE" = true ]; then
        echo "[VERBOSE] $1"
    fi
}

save_config() {
    local variant="$1"
    local danger_mode="$2"
    
    {
        echo "variant=$variant"
        if [ "$danger_mode" = true ]; then
            echo "danger=true"
        fi
    } > "$CONFIG_FILE"
    
    log_verbose "Saved configuration: variant=$variant, danger=$danger_mode"
}

load_config() {
    if [ -f "$CONFIG_FILE" ]; then
        log_verbose "Loading configuration from $CONFIG_FILE"
        
        # Load variant
        SAVED_VARIANT=$(grep "^variant=" "$CONFIG_FILE" 2>/dev/null | cut -d'=' -f2)
        
        # Load danger setting
        SAVED_DANGER=$(grep "^danger=" "$CONFIG_FILE" 2>/dev/null | cut -d'=' -f2)
        if [ "$SAVED_DANGER" = "true" ]; then
            SAVED_DANGER=true
        else
            SAVED_DANGER=false
        fi
        
        log_verbose "Loaded: variant=$SAVED_VARIANT, danger=$SAVED_DANGER"
    else
        log_verbose "No configuration file found"
        SAVED_VARIANT=""
        SAVED_DANGER=false
    fi
}

auto_detect_variant() {
    log_verbose "Auto-detecting project variant..."
    
    if [ -f "go.mod" ] || [ -f "go.sum" ]; then
        echo "go"
    elif [ -f "Cargo.toml" ] || [ -f "Cargo.lock" ]; then
        echo "full"
    elif [ -f "pom.xml" ] || [ -f "build.gradle" ] || [ -f "build.gradle.kts" ]; then
        echo "full"
    elif [ -f "requirements.txt" ] || [ -f "pyproject.toml" ] || [ -f "Pipfile" ]; then
        echo "base"
    elif [ -f "package.json" ]; then
        echo "base"
    else
        echo "base"
    fi
}

determine_variant() {
    if [ -n "$EXPLICIT_VARIANT" ]; then
        echo "$EXPLICIT_VARIANT"
    elif [ -n "$SAVED_VARIANT" ]; then
        log_verbose "Using saved variant: $SAVED_VARIANT"
        echo "$SAVED_VARIANT"
    else
        local detected=$(auto_detect_variant)
        log_verbose "Auto-detected variant: $detected"
        echo "$detected"
    fi
}

validate_variant() {
    local variant="$1"
    case "$variant" in
        base|go|full|cloud|k8s)
            return 0
            ;;
        *)
            echo "Invalid variant: $variant" >&2
            return 1
            ;;
    esac
}
EOF
    chmod +x test-functions.sh
}

# Test configuration save/load functionality
test_config_save_load() {
    local test_dir="config_test_$$"
    mkdir -p "$test_dir"
    cd "$test_dir"
    
    # Create test functions
    create_test_functions
    source test-functions.sh
    
    # Test saving configuration
    save_config "go" true
    
    # Check file was created
    [ -f ".claude-reactor" ] || return 1
    
    # Check file contents
    grep -q "variant=go" ".claude-reactor" || return 1
    grep -q "danger=true" ".claude-reactor" || return 1
    
    # Test loading configuration
    SAVED_VARIANT=""
    SAVED_DANGER=false
    load_config
    
    [ "$SAVED_VARIANT" = "go" ] || return 1
    [ "$SAVED_DANGER" = true ] || return 1
    
    cd ..
    rm -rf "$test_dir"
    return 0
}

# Test auto-detection functionality
test_auto_detection() {
    local test_dir="detection_test_$$"
    mkdir -p "$test_dir"
    cd "$test_dir"
    
    create_test_functions
    source test-functions.sh
    
    # Test Go detection
    echo 'module test' > go.mod
    local detected=$(auto_detect_variant)
    [ "$detected" = "go" ] || return 1
    rm go.mod
    
    # Test Rust detection
    echo '[package]' > Cargo.toml
    detected=$(auto_detect_variant)
    [ "$detected" = "full" ] || return 1
    rm Cargo.toml
    
    # Test Node.js detection
    echo '{}' > package.json
    detected=$(auto_detect_variant)
    [ "$detected" = "base" ] || return 1
    rm package.json
    
    # Test default fallback
    detected=$(auto_detect_variant)
    [ "$detected" = "base" ] || return 1
    
    cd ..
    rm -rf "$test_dir"
    return 0
}

# Test variant validation
test_variant_validation() {
    local test_dir="validation_test_$$"
    mkdir -p "$test_dir"
    cd "$test_dir"
    
    create_test_functions
    source test-functions.sh
    
    # Test valid variants
    validate_variant "base" || return 1
    validate_variant "go" || return 1
    validate_variant "full" || return 1
    validate_variant "cloud" || return 1
    validate_variant "k8s" || return 1
    
    # Test invalid variants (redirect stderr to suppress expected error messages)
    ! validate_variant "invalid" 2>/dev/null || return 1
    ! validate_variant "" 2>/dev/null || return 1
    ! validate_variant "GO" 2>/dev/null || return 1  # Case sensitive
    
    cd ..
    rm -rf "$test_dir"
    return 0
}

# Test variant determination logic
test_variant_determination() {
    local test_dir="determination_test_$$"
    mkdir -p "$test_dir"
    cd "$test_dir"
    
    create_test_functions
    source test-functions.sh
    
    # Test explicit variant takes precedence
    EXPLICIT_VARIANT="cloud"
    SAVED_VARIANT="go"
    echo 'module test' > go.mod  # Would normally detect as 'go'
    
    local determined=$(determine_variant)
    [ "$determined" = "cloud" ] || return 1
    
    # Test saved variant takes precedence over auto-detection
    EXPLICIT_VARIANT=""
    determined=$(determine_variant)
    [ "$determined" = "go" ] || return 1
    
    # Test auto-detection when no explicit or saved variant
    SAVED_VARIANT=""
    determined=$(determine_variant)
    [ "$determined" = "go" ] || return 1  # Should detect from go.mod
    
    cd ..
    rm -rf "$test_dir"
    return 0
}

# Test configuration persistence across runs
test_config_persistence() {
    local test_dir="persistence_test_$$"
    mkdir -p "$test_dir"
    cd "$test_dir"
    
    create_test_functions
    source test-functions.sh
    
    # First "run" - save config
    save_config "full" false
    
    # Second "run" - load config (simulate fresh start)
    SAVED_VARIANT=""
    SAVED_DANGER=false
    load_config
    
    [ "$SAVED_VARIANT" = "full" ] || return 1
    [ "$SAVED_DANGER" = false ] || return 1
    
    # Third "run" - update danger mode
    save_config "full" true
    
    # Fourth "run" - verify both settings persist
    SAVED_VARIANT=""
    SAVED_DANGER=false
    load_config
    
    [ "$SAVED_VARIANT" = "full" ] || return 1
    [ "$SAVED_DANGER" = true ] || return 1
    
    cd ..
    rm -rf "$test_dir"
    return 0
}

# Run all unit tests
run_all_unit_tests() {
    log_info "Running unit tests for claude-reactor functions..."
    
    run_test "Configuration save/load" "test_config_save_load"
    run_test "Auto-detection logic" "test_auto_detection" 
    run_test "Variant validation" "test_variant_validation"
    run_test "Variant determination priority" "test_variant_determination"
    run_test "Configuration persistence" "test_config_persistence"
    
    log_info "Unit tests completed"
}

# Run tests if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    run_all_unit_tests
else
    # Called from test-runner, just run the tests
    run_all_unit_tests
fi