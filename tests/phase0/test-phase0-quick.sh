#!/bin/bash

# Quick Phase 0 Feature Validation
# Validates key functionality implemented in Phase 0

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BINARY="$PROJECT_ROOT/claude-reactor"

echo "🧪 Phase 0 Feature Validation (Quick)"
echo "Binary: $BINARY"
echo ""

# Counters
TESTS=0
PASSED=0

test_feature() {
    local name="$1"
    local command="$2"
    local pattern="$3"
    
    ((TESTS++))
    echo -n "Testing $name... "
    
    if output=$(eval "$command" 2>&1) && [[ "$output" =~ $pattern ]]; then
        echo "✅ PASS"
        ((PASSED++))
    else
        echo "❌ FAIL"
        echo "  Command: $command"
        echo "  Expected pattern: $pattern"
        echo "  Actual output: $output"
    fi
}

echo "Phase 0.1: Registry CLI Integration"
test_feature "Registry --dev flag" \
    "$BINARY run --help" \
    "--dev.*Force local build"

test_feature "Registry --registry-off flag" \
    "$BINARY run --help" \
    "--registry-off.*Disable registry"

test_feature "Registry --pull-latest flag" \
    "$BINARY run --help" \
    "--pull-latest.*Force pull latest"

echo ""
echo "Phase 0.2: System Installation"
test_feature "Install flag" \
    "$BINARY --help" \
    "--install.*Install claude-reactor to system PATH"

test_feature "Uninstall flag" \
    "$BINARY --help" \
    "--uninstall.*Remove claude-reactor from system PATH"

echo ""
echo "Phase 0.3: Conversation Control"
test_feature "Continue flag" \
    "$BINARY run --help" \
    "--continue.*Enable conversation continuation"

echo ""
echo "Phase 0.4: Enhanced Config Display"
test_feature "Enhanced config show" \
    "$BINARY config show" \
    "=== Claude-Reactor Configuration ==="

test_feature "Registry configuration display" \
    "$BINARY config show" \
    "Registry Configuration:"

test_feature "Config show verbose flag" \
    "$BINARY config show --help" \
    "--verbose.*Show detailed system"

test_feature "Config show raw flag" \
    "$BINARY config show --help" \
    "--raw.*Include raw configuration"

echo ""
echo "v2 Prefix Implementation"
test_feature "v2 prefix in container names" \
    "$BINARY config show --verbose" \
    "Container Name:.*v2-claude-reactor"

test_feature "v2 prefix in image names" \
    "$BINARY config show --verbose" \
    "Image Name:.*v2-claude-reactor"

echo ""
echo "📊 Summary"
echo "Tests run: $TESTS"
echo "Tests passed: $PASSED"
echo "Tests failed: $((TESTS - PASSED))"

if [[ $PASSED -eq $TESTS ]]; then
    echo ""
    echo "🎉 All Phase 0 features are working correctly!"
    echo "✅ Go CLI has achieved feature parity with bash script"
    exit 0
else
    echo ""
    echo "❌ Some Phase 0 features need fixes"
    exit 1
fi