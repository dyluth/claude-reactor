#!/usr/bin/env bash

# Test runner for claude-reactor functionality
# Runs unit tests, integration tests, and demonstrates all features

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TEST_RESULTS_DIR="$SCRIPT_DIR/results"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test tracking
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

log_failure() {
    echo -e "${RED}[FAIL]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

run_test() {
    local test_name="$1"
    local test_command="$2"
    
    echo ""
    log_info "Running test: $test_name"
    ((TESTS_RUN++))
    
    if eval "$test_command"; then
        log_success "$test_name"
        ((TESTS_PASSED++))
        return 0
    else
        log_failure "$test_name"
        ((TESTS_FAILED++))
        return 1
    fi
}

show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Test runner for claude-reactor functionality.

OPTIONS:
    --unit          Run only unit tests
    --integration   Run only integration tests  
    --quick         Skip Docker builds (faster testing)
    --verbose       Enable verbose output
    --clean         Clean up test artifacts before running
    --help, -h      Show this help message

EXAMPLES:
    $0                      # Run all tests
    $0 --unit               # Run only unit tests
    $0 --integration --quick # Run integration tests without building images
    $0 --clean --verbose    # Clean setup with detailed output

EOF
}

# Parse arguments
UNIT_ONLY=false
INTEGRATION_ONLY=false
QUICK=false
VERBOSE=false
CLEAN=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --unit)
            UNIT_ONLY=true
            shift
            ;;
        --integration)
            INTEGRATION_ONLY=true
            shift
            ;;
        --quick)
            QUICK=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --clean)
            CLEAN=true
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

# Create results directory
mkdir -p "$TEST_RESULTS_DIR"

# Main test execution
main() {
    log_info "Starting claude-reactor test suite"
    log_info "Project root: $PROJECT_ROOT"
    
    # Clean up if requested
    if [ "$CLEAN" = true ]; then
        log_info "Cleaning up test artifacts..."
        rm -rf "$TEST_RESULTS_DIR"/*
        find "$SCRIPT_DIR" -name ".claude-reactor" -delete 2>/dev/null || true
        find "$SCRIPT_DIR" -name "go.mod" -delete 2>/dev/null || true
        find "$SCRIPT_DIR" -name "package.json" -delete 2>/dev/null || true
        find "$SCRIPT_DIR" -name "Cargo.toml" -delete 2>/dev/null || true
    fi
    
    # Set verbose mode
    if [ "$VERBOSE" = true ]; then
        set -x
    fi
    
    # Run unit tests
    if [ "$INTEGRATION_ONLY" = false ]; then
        log_info "=== Running Unit Tests ==="
        if "$SCRIPT_DIR/unit/test-functions.sh"; then
            TESTS_RUN=$((TESTS_RUN + 1))
            TESTS_PASSED=$((TESTS_PASSED + 1))
            log_success "Unit tests completed"
        else
            TESTS_RUN=$((TESTS_RUN + 1))
            TESTS_FAILED=$((TESTS_FAILED + 1))
            log_failure "Unit tests failed"
        fi
    fi
    
    # Run integration tests  
    if [ "$UNIT_ONLY" = false ]; then
        log_info "=== Running Integration Tests ==="
        if [ "$QUICK" = true ]; then
            log_warning "Skipping Docker builds (--quick mode)"
            if SKIP_BUILDS=true "$SCRIPT_DIR/integration/test-variants.sh"; then
                TESTS_RUN=$((TESTS_RUN + 1))
                TESTS_PASSED=$((TESTS_PASSED + 1))
                log_success "Integration tests completed (quick mode)"
            else
                TESTS_RUN=$((TESTS_RUN + 1))
                TESTS_FAILED=$((TESTS_FAILED + 1))
                log_failure "Integration tests failed"
            fi
        else
            if "$SCRIPT_DIR/integration/test-variants.sh"; then
                TESTS_RUN=$((TESTS_RUN + 1))
                TESTS_PASSED=$((TESTS_PASSED + 1))
                log_success "Integration tests completed"
            else
                TESTS_RUN=$((TESTS_RUN + 1))
                TESTS_FAILED=$((TESTS_FAILED + 1))
                log_failure "Integration tests failed"
            fi
        fi
    fi
    
    # Show summary
    echo ""
    echo "==============================================="
    log_info "Test Summary:"
    echo "  Tests Run:    $TESTS_RUN"
    echo "  Tests Passed: $TESTS_PASSED"
    echo "  Tests Failed: $TESTS_FAILED"
    
    if [ $TESTS_FAILED -eq 0 ]; then
        log_success "All tests passed! 🎉"
        return 0
    else
        log_failure "Some tests failed. Check the output above."
        return 1
    fi
}

# Export functions for use in test scripts
export -f log_info log_success log_failure log_warning run_test
export TESTS_RUN TESTS_PASSED TESTS_FAILED
export PROJECT_ROOT TEST_RESULTS_DIR VERBOSE

# Run main function and exit with its return code
main "$@"
exit $?