#!/bin/bash

# Template System Unit Tests
# Tests template management, scaffolding, and CLI integration in isolated temporary directories

set -e

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
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Test output
TEST_OUTPUT=""

# Create temporary directory for all tests
TEMP_BASE_DIR=""
cleanup_temp_dir() {
    if [[ -n "$TEMP_BASE_DIR" && -d "$TEMP_BASE_DIR" ]]; then
        echo -e "${BLUE}[CLEANUP]${NC} Removing temporary test directory: $TEMP_BASE_DIR"
        rm -rf "$TEMP_BASE_DIR"
    fi
}

# Set up cleanup trap
trap cleanup_temp_dir EXIT

# Setup function
setup_test_env() {
    # Create temporary directory for all tests
    TEMP_BASE_DIR=$(mktemp -d -t claude-reactor-template-tests-XXXXXX)
    echo -e "${BLUE}[SETUP]${NC} Created temporary test directory: $TEMP_BASE_DIR"
    
    # Build binary if it doesn't exist
    if [[ ! -f "$BINARY" ]]; then
        echo -e "${BLUE}[SETUP]${NC} Building claude-reactor binary..."
        cd "$PROJECT_ROOT"
        go build -o claude-reactor ./cmd/claude-reactor
        cd - > /dev/null
    fi
    
    # Verify binary exists and is executable
    if [[ ! -x "$BINARY" ]]; then
        echo -e "${RED}[ERROR]${NC} Binary not found or not executable: $BINARY"
        exit 1
    fi
    
    echo -e "${GREEN}[SETUP]${NC} Test environment ready"
}

# Test helper functions
run_test() {
    local test_name="$1"
    local test_function="$2"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -e "\n${BLUE}[TEST $TOTAL_TESTS]${NC} $test_name"
    
    # Create isolated temp directory for this test
    local test_temp_dir="$TEMP_BASE_DIR/test_$TOTAL_TESTS"
    mkdir -p "$test_temp_dir"
    
    # Run test in isolated environment
    if (cd "$test_temp_dir" && $test_function "$test_temp_dir"); then
        echo -e "${GREEN}[PASS]${NC} $test_name"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}[FAIL]${NC} $test_name"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        TEST_OUTPUT="${TEST_OUTPUT}FAILED: $test_name\n"
    fi
}

assert_file_exists() {
    local file="$1"
    local description="$2"
    
    if [[ -f "$file" ]]; then
        echo -e "  ${GREEN}✓${NC} File exists: $description ($file)"
        return 0
    else
        echo -e "  ${RED}✗${NC} File missing: $description ($file)"
        return 1
    fi
}

assert_dir_exists() {
    local dir="$1"
    local description="$2"
    
    if [[ -d "$dir" ]]; then
        echo -e "  ${GREEN}✓${NC} Directory exists: $description ($dir)"
        return 0
    else
        echo -e "  ${RED}✗${NC} Directory missing: $description ($dir)"
        return 1
    fi
}

assert_file_contains() {
    local file="$1"
    local pattern="$2"
    local description="$3"
    
    if [[ -f "$file" ]] && grep -q "$pattern" "$file"; then
        echo -e "  ${GREEN}✓${NC} File contains pattern: $description"
        return 0
    else
        echo -e "  ${RED}✗${NC} File missing pattern: $description (pattern: $pattern)"
        return 1
    fi
}

assert_command_success() {
    local cmd="$1"
    local description="$2"
    
    if eval "$cmd" >/dev/null 2>&1; then
        echo -e "  ${GREEN}✓${NC} Command succeeded: $description"
        return 0
    else
        echo -e "  ${RED}✗${NC} Command failed: $description (cmd: $cmd)"
        return 1
    fi
}

assert_command_output_contains() {
    local cmd="$1"
    local pattern="$2"
    local description="$3"
    
    local output
    output=$(eval "$cmd" 2>&1)
    
    if echo "$output" | grep -q "$pattern"; then
        echo -e "  ${GREEN}✓${NC} Command output contains: $description"
        return 0
    else
        echo -e "  ${RED}✗${NC} Command output missing: $description (pattern: $pattern)"
        echo -e "    Output was: $output"
        return 1
    fi
}

# Test Functions

test_template_list_command() {
    local test_dir="$1"
    echo "  Testing template list command..."
    
    # Test basic list command
    assert_command_success "'$BINARY' template list" "template list command" &&
    assert_command_output_contains "'$BINARY' template list" "Available project templates" "template list shows header" &&
    assert_command_output_contains "'$BINARY' template list" "go-api" "template list shows go-api template" &&
    assert_command_output_contains "'$BINARY' template list" "📦" "template list shows devcontainer indicators"
}

test_template_show_command() {
    local test_dir="$1"
    echo "  Testing template show command..."
    
    # Test showing specific template
    assert_command_success "'$BINARY' template show go-api" "template show go-api command" &&
    assert_command_output_contains "'$BINARY' template show go-api" "Go REST API" "show command displays description" &&
    assert_command_output_contains "'$BINARY' template show go-api" "gorilla/mux" "show command displays framework" &&
    assert_command_output_contains "'$BINARY' template show go-api" "Template Variables" "show command displays variables" &&
    assert_command_output_contains "'$BINARY' template show go-api" "PORT" "show command shows PORT variable"
}

test_template_new_go_api() {
    local test_dir="$1"
    echo "  Testing Go API template creation..."
    
    local project_name="test-go-api"
    local project_path="$test_dir/$project_name"
    
    # Create project from template
    assert_command_success "'$BINARY' template new go-api $project_name" "create go-api project" &&
    
    # Verify project structure
    assert_dir_exists "$project_path" "project directory" &&
    assert_file_exists "$project_path/main.go" "main.go file" &&
    assert_file_exists "$project_path/go.mod" "go.mod file" &&
    assert_file_exists "$project_path/README.md" "README.md file" &&
    assert_file_exists "$project_path/Dockerfile" "Dockerfile" &&
    assert_file_exists "$project_path/.gitignore" ".gitignore file" &&
    assert_file_exists "$project_path/.claude-reactor" ".claude-reactor config" &&
    
    # Verify devcontainer integration
    assert_dir_exists "$project_path/.devcontainer" ".devcontainer directory" &&
    assert_file_exists "$project_path/.devcontainer/devcontainer.json" "devcontainer.json" &&
    
    # Verify git initialization
    assert_dir_exists "$project_path/.git" ".git directory" &&
    
    # Verify template variable substitution
    assert_file_contains "$project_path/main.go" "$project_name" "main.go contains project name" &&
    assert_file_contains "$project_path/go.mod" "module $project_name" "go.mod has correct module name" &&
    assert_file_contains "$project_path/README.md" "# $project_name" "README has project title"
}

test_template_new_rust_cli() {
    local test_dir="$1"
    echo "  Testing Rust CLI template creation..."
    
    local project_name="test-rust-cli"
    local project_path="$test_dir/$project_name"
    
    # Create project from template
    assert_command_success "'$BINARY' template new rust-cli $project_name" "create rust-cli project" &&
    
    # Verify project structure
    assert_dir_exists "$project_path" "project directory" &&
    assert_file_exists "$project_path/Cargo.toml" "Cargo.toml file" &&
    assert_file_exists "$project_path/src/main.rs" "src/main.rs file" &&
    assert_file_exists "$project_path/README.md" "README.md file" &&
    assert_file_exists "$project_path/.gitignore" ".gitignore file" &&
    
    # Verify template variable substitution
    assert_file_contains "$project_path/Cargo.toml" "name = \"$project_name\"" "Cargo.toml has correct name" &&
    assert_file_contains "$project_path/src/main.rs" "$project_name" "main.rs contains project name"
}

test_template_new_node_api() {
    local test_dir="$1"
    echo "  Testing Node.js API template creation..."
    
    local project_name="test-node-api"
    local project_path="$test_dir/$project_name"
    
    # Create project from template
    assert_command_success "'$BINARY' template new node-api $project_name" "create node-api project" &&
    
    # Verify project structure
    assert_dir_exists "$project_path" "project directory" &&
    assert_file_exists "$project_path/package.json" "package.json file" &&
    assert_file_exists "$project_path/src/index.ts" "src/index.ts file" &&
    assert_file_exists "$project_path/tsconfig.json" "tsconfig.json file" &&
    assert_file_exists "$project_path/README.md" "README.md file" &&
    
    # Verify template variable substitution
    assert_file_contains "$project_path/package.json" "\"name\": \"$project_name\"" "package.json has correct name" &&
    assert_file_contains "$project_path/src/index.ts" "$project_name" "index.ts contains project name"
}

test_template_new_python_api() {
    local test_dir="$1"
    echo "  Testing Python API template creation..."
    
    local project_name="test-python-api"
    local project_path="$test_dir/$project_name"
    
    # Create project from template
    assert_command_success "'$BINARY' template new python-api $project_name" "create python-api project" &&
    
    # Verify project structure
    assert_dir_exists "$project_path" "project directory" &&
    assert_file_exists "$project_path/main.py" "main.py file" &&
    assert_file_exists "$project_path/requirements.txt" "requirements.txt file" &&
    assert_file_exists "$project_path/README.md" "README.md file" &&
    
    # Verify template variable substitution
    assert_file_contains "$project_path/main.py" "$project_name" "main.py contains project name" &&
    assert_file_contains "$project_path/main.py" "FastAPI" "main.py uses FastAPI"
}

test_template_with_variables() {
    local test_dir="$1"
    echo "  Testing template creation with custom variables..."
    
    local project_name="test-with-vars"
    local project_path="$test_dir/$project_name"
    local custom_port="3000"
    
    # Create project with custom variables
    assert_command_success "'$BINARY' template new go-api $project_name --var PORT=$custom_port" "create project with custom PORT" &&
    
    # Verify custom variable was used (this would require the template to use the PORT variable)
    assert_file_exists "$project_path/main.go" "main.go file exists"
    # Note: The current template doesn't use PORT in file content, but this tests the variable parsing
}

test_template_force_overwrite() {
    local test_dir="$1"
    echo "  Testing template force overwrite..."
    
    local project_name="test-overwrite"
    local project_path="$test_dir/$project_name"
    
    # Create project first time
    assert_command_success "'$BINARY' template new go-api $project_name" "create initial project" &&
    
    # Try to create again without force (should fail)
    if "$BINARY" template new go-api "$project_name" 2>/dev/null; then
        echo -e "  ${RED}✗${NC} Expected command to fail without --force flag"
        return 1
    else
        echo -e "  ${GREEN}✓${NC} Command correctly failed without --force flag"
    fi &&
    
    # Create again with force (should succeed)
    assert_command_success "'$BINARY' template new go-api $project_name --force" "create project with --force"
}

test_template_validate_command() {
    local test_dir="$1"
    echo "  Testing template validation..."
    
    # Test validating built-in template
    assert_command_success "'$BINARY' template validate go-api" "validate go-api template" &&
    assert_command_output_contains "'$BINARY' template validate go-api" "Template.*is valid" "validation shows success message"
}

test_devcontainer_integration() {
    local test_dir="$1"
    echo "  Testing devcontainer integration..."
    
    local project_name="test-devcontainer"
    local project_path="$test_dir/$project_name"
    
    # Create project
    assert_command_success "'$BINARY' template new go-api $project_name" "create project for devcontainer test" &&
    
    # Verify devcontainer configuration
    assert_file_exists "$project_path/.devcontainer/devcontainer.json" "devcontainer.json" &&
    assert_file_contains "$project_path/.devcontainer/devcontainer.json" "Claude Reactor Go" "devcontainer name" &&
    assert_file_contains "$project_path/.devcontainer/devcontainer.json" "golang.Go" "Go extension" &&
    assert_file_contains "$project_path/.devcontainer/devcontainer.json" "workspaces" "workspace folder" &&
    
    # Test devcontainer validation using existing command
    cd "$project_path" &&
    assert_command_success "'$BINARY' devcontainer validate" "devcontainer validate command"
}

test_config_file_creation() {
    local test_dir="$1"
    echo "  Testing .claude-reactor config file creation..."
    
    local project_name="test-config"
    local project_path="$test_dir/$project_name"
    
    # Create project
    assert_command_success "'$BINARY' template new go-api $project_name" "create project for config test" &&
    
    # Verify config file
    assert_file_exists "$project_path/.claude-reactor" "claude-reactor config file" &&
    assert_file_contains "$project_path/.claude-reactor" "variant=go" "config contains go variant"
}

test_git_initialization() {
    local test_dir="$1"
    echo "  Testing git repository initialization..."
    
    local project_name="test-git"
    local project_path="$test_dir/$project_name"
    
    # Create project
    assert_command_success "'$BINARY' template new go-api $project_name" "create project for git test" &&
    
    # Verify git initialization
    assert_dir_exists "$project_path/.git" ".git directory" &&
    assert_file_exists "$project_path/.gitignore" ".gitignore file" &&
    assert_file_contains "$project_path/.gitignore" "*.exe" "gitignore contains Go patterns"
}

test_template_error_handling() {
    local test_dir="$1"
    echo "  Testing error handling..."
    
    # Test invalid template name
    if "$BINARY" template new invalid-template test-project 2>/dev/null; then
        echo -e "  ${RED}✗${NC} Expected command to fail with invalid template"
        return 1
    else
        echo -e "  ${GREEN}✓${NC} Command correctly failed with invalid template"
    fi &&
    
    # Test invalid project name
    if "$BINARY" template new go-api "invalid project name with spaces" 2>/dev/null; then
        echo -e "  ${RED}✗${NC} Expected command to fail with invalid project name"
        return 1
    else
        echo -e "  ${GREEN}✓${NC} Command correctly failed with invalid project name"
    fi
}

# Main test execution
main() {
    echo -e "${BLUE}=== Claude-Reactor Template System Tests ===${NC}"
    echo -e "${BLUE}Testing template management and project scaffolding${NC}\n"
    
    # Setup test environment
    setup_test_env
    
    # Run all tests
    run_test "Template List Command" test_template_list_command
    run_test "Template Show Command" test_template_show_command
    run_test "Go API Template Creation" test_template_new_go_api
    run_test "Rust CLI Template Creation" test_template_new_rust_cli
    run_test "Node.js API Template Creation" test_template_new_node_api
    run_test "Python API Template Creation" test_template_new_python_api
    run_test "Template with Custom Variables" test_template_with_variables
    run_test "Template Force Overwrite" test_template_force_overwrite
    run_test "Template Validation" test_template_validate_command
    run_test "DevContainer Integration" test_devcontainer_integration
    run_test "Config File Creation" test_config_file_creation
    run_test "Git Initialization" test_git_initialization
    run_test "Error Handling" test_template_error_handling
    
    # Print results
    echo -e "\n${BLUE}=== Test Results ===${NC}"
    echo -e "Total tests: $TOTAL_TESTS"
    echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
    echo -e "${RED}Failed: $FAILED_TESTS${NC}"
    
    if [[ $FAILED_TESTS -gt 0 ]]; then
        echo -e "\n${RED}Failed tests:${NC}"
        echo -e "$TEST_OUTPUT"
        exit 1
    else
        echo -e "\n${GREEN}All tests passed! 🎉${NC}"
        exit 0
    fi
}

# Run main function
main "$@"