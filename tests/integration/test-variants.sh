#!/usr/bin/env bash

# Integration tests for claude-reactor container variants
# Tests actual Docker functionality and container behavior

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TEST_DIR="$SCRIPT_DIR/../fixtures/integration"

# Create test environment
mkdir -p "$TEST_DIR"

# Import test functions if available
if [[ -n "${log_info:-}" ]]; then
    # Functions are available from test-runner
    true
else
    # Define basic test functions for standalone execution
    log_info() { echo "[INFO] $1"; }
    log_success() { echo "[PASS] $1"; }
    log_failure() { echo "[FAIL] $1"; }
    log_warning() { echo "[WARN] $1"; }
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

# Test if Docker is available
check_docker() {
    if ! command -v docker &> /dev/null; then
        log_failure "Docker is not installed or not in PATH"
        return 1
    fi
    
    if ! docker info &> /dev/null; then
        log_failure "Docker daemon is not running"
        return 1
    fi
    
    return 0
}

# Detect architecture for consistent naming
detect_architecture() {
    local arch_raw=$(uname -m)
    case "$arch_raw" in
        x86_64|amd64)
            echo "amd64"
            ;;
        arm64|aarch64)
            echo "arm64"
            ;;
        *)
            echo "unknown"
            return 1
            ;;
    esac
}

# Test building a specific variant
test_build_variant() {
    local variant="$1"
    local architecture=$(detect_architecture)
    local image_name="claude-reactor-$variant-$architecture"
    
    log_info "Building variant: $variant (architecture: $architecture)"
    
    cd "$PROJECT_ROOT"
    
    # Build the variant with architecture detection
    if docker build --target "$variant" -t "$image_name" . > /dev/null 2>&1; then
        log_info "Build successful for variant: $variant"
        return 0
    else
        log_failure "Build failed for variant: $variant"
        return 1
    fi
}

# Test that a container can be created and started
test_container_lifecycle() {
    local variant="$1"
    local architecture=$(detect_architecture)
    local image_name="claude-reactor-$variant-$architecture"
    local container_name="test-claude-agent-$variant-$architecture-$$"
    
    log_info "Testing container lifecycle for variant: $variant"
    
    # Create and start container
    if ! docker run -d --name "$container_name" "$image_name" > /dev/null 2>&1; then
        log_failure "Failed to create container for variant: $variant"
        return 1
    fi
    
    # Wait for container to be ready
    sleep 2
    
    # Check if container is running
    if ! docker ps --format '{{.Names}}' | grep -q "^${container_name}$"; then
        log_failure "Container not running for variant: $variant"
        docker rm -f "$container_name" > /dev/null 2>&1 || true
        return 1
    fi
    
    # Test basic container responsiveness
    if ! docker exec "$container_name" echo "Container is responsive" > /dev/null 2>&1; then
        log_failure "Container not responsive for variant: $variant"
        docker rm -f "$container_name" > /dev/null 2>&1 || true
        return 1
    fi
    
    # Clean up
    docker rm -f "$container_name" > /dev/null 2>&1 || true
    
    return 0
}

# Test that expected tools are available in variant
test_variant_tools() {
    local variant="$1"
    local architecture=$(detect_architecture)
    local image_name="claude-reactor-$variant-$architecture"
    local container_name="test-tools-$variant-$architecture-$$"
    
    log_info "Testing tools availability for variant: $variant"
    
    # Start container
    if ! docker run -d --name "$container_name" "$image_name" > /dev/null 2>&1; then
        return 1
    fi
    
    sleep 2
    
    local success=true
    
    # Test base tools (should be in all variants)
    docker exec "$container_name" which node > /dev/null 2>&1 || success=false
    docker exec "$container_name" which python3 > /dev/null 2>&1 || success=false
    docker exec "$container_name" which python > /dev/null 2>&1 || success=false
    docker exec "$container_name" which pip3 > /dev/null 2>&1 || success=false
    docker exec "$container_name" which pip > /dev/null 2>&1 || success=false
    docker exec "$container_name" which uv > /dev/null 2>&1 || success=false
    docker exec "$container_name" which uvx > /dev/null 2>&1 || success=false
    docker exec "$container_name" which git > /dev/null 2>&1 || success=false
    docker exec "$container_name" which ripgrep > /dev/null 2>&1 || success=false
    
    # Test variant-specific tools
    case "$variant" in
        go|full|cloud|k8s)
            docker exec "$container_name" which go > /dev/null 2>&1 || success=false
            docker exec "$container_name" go version > /dev/null 2>&1 || success=false
            ;;
    esac
    
    case "$variant" in
        full|cloud|k8s)
            docker exec "$container_name" which cargo > /dev/null 2>&1 || success=false
            docker exec "$container_name" which java > /dev/null 2>&1 || success=false
            docker exec "$container_name" which mysql > /dev/null 2>&1 || success=false
            ;;
    esac
    
    case "$variant" in
        cloud)
            docker exec "$container_name" which aws > /dev/null 2>&1 || success=false
            docker exec "$container_name" which gcloud > /dev/null 2>&1 || success=false
            docker exec "$container_name" which az > /dev/null 2>&1 || success=false
            ;;
    esac
    
    case "$variant" in
        k8s)
            docker exec "$container_name" which helm > /dev/null 2>&1 || success=false
            docker exec "$container_name" which k9s > /dev/null 2>&1 || success=false
            ;;
    esac
    
    # Clean up
    docker rm -f "$container_name" > /dev/null 2>&1 || true
    
    if [ "$success" = true ]; then
        return 0
    else
        log_failure "Some tools missing in variant: $variant"
        return 1
    fi
}

# Test claude-reactor script with different options
test_script_options() {
    local test_dir="script_test_$$"
    mkdir -p "$TEST_DIR/$test_dir"
    cd "$TEST_DIR/$test_dir"
    
    local reactor_script="$PROJECT_ROOT/claude-reactor"
    
    # Test --list-variants
    if ! "$reactor_script" --list-variants > /dev/null 2>&1; then
        log_failure "Failed to list variants"
        cd "$TEST_DIR"
        rm -rf "$test_dir"
        return 1
    fi
    
    # Test config show (should work even without config file)
    if ! "$reactor_script" config show > /dev/null 2>&1; then
        log_failure "Failed to show config"
        cd "$TEST_DIR" 
        rm -rf "$test_dir"
        return 1
    fi
    
    # Test variant validation (should fail for invalid variant)
    if "$reactor_script" run --variant invalid --help > /dev/null 2>&1; then
        # This should succeed because --help doesn't validate the variant
        # Let's test build with invalid variant instead which does validate
        if "$reactor_script" build invalid > /dev/null 2>&1; then
            log_failure "Script should reject invalid variant"
            cd "$TEST_DIR"
            rm -rf "$test_dir"
            return 1
        fi
    fi
    
    cd "$TEST_DIR"
    rm -rf "$test_dir"
    return 0
}

# Test configuration file creation and reading
test_config_file_integration() {
    local test_dir="config_integration_$$"
    mkdir -p "$TEST_DIR/$test_dir"
    cd "$TEST_DIR/$test_dir"
    
    local reactor_script="$PROJECT_ROOT/claude-reactor"
    
    # Create a go.mod to test auto-detection
    echo 'module test' > go.mod
    
    # Run script to show config (should auto-detect 'go')
    local output=$("$reactor_script" config show 2>&1 || true)
    if ! echo "$output" | grep -q "go"; then
        log_failure "Auto-detection not working in integration"
        cd "$TEST_DIR"
        rm -rf "$test_dir"
        return 1
    fi
    
    # Set explicit variant using legacy compatibility flag and verify config file creation
    "$reactor_script" --variant full > /dev/null 2>&1 || true
    
    if [ ! -f ".claude-reactor" ]; then
        log_failure "Configuration file not created"
        cd "$TEST_DIR"
        rm -rf "$test_dir"
        return 1
    fi
    
    if ! grep -q "variant=full" ".claude-reactor"; then
        log_failure "Configuration not saved correctly"
        cd "$TEST_DIR"
        rm -rf "$test_dir" 
        return 1
    fi
    
    cd "$TEST_DIR"
    rm -rf "$test_dir"
    return 0
}

# Main test execution
run_integration_tests() {
    log_info "Running integration tests for claude-reactor..."
    
    # Check prerequisites
    run_test "Docker availability" "check_docker"
    
    # Test script functionality (doesn't require builds)
    run_test "Script options" "test_script_options"
    run_test "Configuration file integration" "test_config_file_integration"
    
    # Skip Docker builds if requested
    if [[ "${SKIP_BUILDS:-false}" == "true" ]]; then
        log_warning "Skipping Docker build tests (SKIP_BUILDS=true)"
        log_info "Integration tests completed (partial)"
        return
    fi
    
    # Test each variant
    local variants=("base" "go" "full")
    
    # Only test cloud and k8s if we have time (they're large)
    if [[ "${QUICK:-false}" != "true" ]]; then
        variants+=("cloud" "k8s")
    fi
    
    for variant in "${variants[@]}"; do
        run_test "Build variant: $variant" "test_build_variant $variant"
        run_test "Container lifecycle: $variant" "test_container_lifecycle $variant"
        run_test "Tools availability: $variant" "test_variant_tools $variant"
    done
    
    log_info "Integration tests completed"
}

# Clean up any leftover test containers
cleanup_test_containers() {
    log_info "Cleaning up test containers..."
    docker ps -a --format '{{.Names}}' | grep "test-claude-agent\|test-tools" | xargs -r docker rm -f > /dev/null 2>&1 || true
}

# Trap to ensure cleanup on exit
trap cleanup_test_containers EXIT

# Run tests if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    run_integration_tests
else
    # Called from test-runner
    run_integration_tests
fi