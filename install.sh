#!/usr/bin/env bash

# Claude-Reactor Installer
# Downloads and installs the latest claude-reactor binary for your platform

set -euo pipefail

# Configuration
GITHUB_REPO="dyluth/claude-reactor"
VERSION="${VERSION:-v0.1.0}"  # TODO: Auto-detect latest when using tags
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
BASE_URL="https://github.com/${GITHUB_REPO}/releases/download"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Platform detection
detect_platform() {
    local os arch

    # Detect OS
    case "$(uname -s)" in
        Linux*)     os="linux" ;;
        Darwin*)    os="darwin" ;;
        *)
            log_error "Unsupported operating system: $(uname -s)"
            log_error "Claude-reactor currently supports Linux and macOS only"
            exit 1
            ;;
    esac

    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64)   arch="amd64" ;;
        arm64|aarch64)  arch="arm64" ;;
        *)
            log_error "Unsupported architecture: $(uname -m)"
            log_error "Claude-reactor currently supports x86_64/amd64 and arm64/aarch64 only"
            exit 1
            ;;
    esac

    echo "${os}-${arch}"
}

# Check dependencies
check_dependencies() {
    local missing_deps=()

    # Check for required commands
    for cmd in curl sha256sum; do
        if ! command -v "$cmd" &> /dev/null; then
            missing_deps+=("$cmd")
        fi
    done

    if [ ${#missing_deps[@]} -ne 0 ]; then
        log_error "Missing required dependencies: ${missing_deps[*]}"
        log_error "Please install the missing commands and try again"
        exit 1
    fi

    # Check for Docker (warn but don't fail)
    if ! command -v docker &> /dev/null; then
        log_warning "Docker not found in PATH"
        log_warning "Claude-reactor requires Docker for container functionality"
        log_warning "Please install Docker: https://docs.docker.com/get-docker/"
        echo
        read -p "Continue installation anyway? [y/N] " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Installation cancelled"
            exit 0
        fi
    else
        log_success "Docker found: $(docker --version)"
    fi
}

# Download and verify binary
download_binary() {
    local platform="$1"
    local binary_name="claude-reactor-${VERSION}-${platform}"
    local binary_url="${BASE_URL}/${VERSION}/${binary_name}"
    local checksum_url="${BASE_URL}/${VERSION}/${binary_name}.sha256"
    local temp_dir

    temp_dir=$(mktemp -d)
    trap "rm -rf '$temp_dir'" EXIT

    log_info "Downloading claude-reactor ${VERSION} for ${platform}..."
    log_info "Download URL: ${binary_url}"

    # Download binary
    if ! curl -fsSL -o "${temp_dir}/${binary_name}" "${binary_url}"; then
        log_error "Failed to download binary from ${binary_url}"
        log_error "Please check your internet connection and try again"
        exit 1
    fi

    # Download checksum
    log_info "Downloading and verifying checksum..."
    if ! curl -fsSL -o "${temp_dir}/${binary_name}.sha256" "${checksum_url}"; then
        log_error "Failed to download checksum from ${checksum_url}"
        log_error "Proceeding without checksum verification (not recommended)"
    else
        # Verify checksum
        if ! (cd "$temp_dir" && sha256sum -c "${binary_name}.sha256" --quiet); then
            log_error "Checksum verification failed!"
            log_error "The downloaded binary may be corrupted or tampered with"
            exit 1
        fi
        log_success "Checksum verification passed"
    fi

    echo "${temp_dir}/${binary_name}"
}

# Install binary
install_binary() {
    local binary_path="$1"
    local install_path="${INSTALL_DIR}/claude-reactor"

    # Create install directory if it doesn't exist
    if [ ! -d "$INSTALL_DIR" ]; then
        log_info "Creating install directory: $INSTALL_DIR"
        mkdir -p "$INSTALL_DIR"
    fi

    # Check if binary already exists
    if [ -f "$install_path" ]; then
        local current_version
        current_version=$("$install_path" --version 2>/dev/null || echo "unknown")
        log_warning "Existing installation found: $current_version"
        echo
        read -p "Replace existing installation? [Y/n] " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Nn]$ ]]; then
            log_info "Installation cancelled"
            exit 0
        fi
    fi

    # Install binary
    log_info "Installing claude-reactor to $install_path"
    cp "$binary_path" "$install_path"
    chmod +x "$install_path"

    log_success "Claude-reactor installed successfully!"
}

# Check if installed location is in PATH
check_path() {
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        log_warning "$INSTALL_DIR is not in your PATH"
        log_info "To use claude-reactor from anywhere, add this line to your shell profile:"
        echo
        echo "    export PATH=\"\$PATH:$INSTALL_DIR\""
        echo
        log_info "Common shell profile files:"
        echo "  - ~/.bashrc (bash)"
        echo "  - ~/.zshrc (zsh)"
        echo "  - ~/.config/fish/config.fish (fish)"
        echo
    fi
}

# Test installation
test_installation() {
    local install_path="${INSTALL_DIR}/claude-reactor"

    log_info "Testing installation..."

    if [ -x "$install_path" ]; then
        local version_output
        version_output=$("$install_path" --version 2>/dev/null || echo "version check failed")
        log_success "Installation test passed: $version_output"
        echo
        log_info "Try running: claude-reactor --help"
    else
        log_error "Installation test failed: binary not executable"
        exit 1
    fi
}

# Show usage information
show_usage() {
    cat << EOF
Claude-Reactor Installer

USAGE:
    $0 [OPTIONS]

OPTIONS:
    -h, --help              Show this help message
    -v, --version VERSION   Install specific version (default: $VERSION)
    -d, --dir DIRECTORY     Install directory (default: $INSTALL_DIR)
    --force                 Skip confirmation prompts

ENVIRONMENT VARIABLES:
    VERSION                 Version to install (default: $VERSION)
    INSTALL_DIR            Installation directory (default: $INSTALL_DIR)

EXAMPLES:
    # Install latest version
    $0

    # Install specific version
    $0 --version v0.2.0

    # Install to custom directory
    $0 --dir /usr/local/bin

    # Non-interactive installation
    $0 --force

EOF
}

# Main installation function
main() {
    local force=false

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_usage
                exit 0
                ;;
            -v|--version)
                VERSION="$2"
                shift 2
                ;;
            -d|--dir)
                INSTALL_DIR="$2"
                shift 2
                ;;
            --force)
                force=true
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done

    # Display banner
    echo "=================================="
    echo "   Claude-Reactor Installer"
    echo "=================================="
    echo

    log_info "Installing claude-reactor ${VERSION} to ${INSTALL_DIR}"

    # Detect platform
    local platform
    platform=$(detect_platform)
    log_info "Detected platform: $platform"

    # Check dependencies
    check_dependencies

    # Confirm installation
    if [ "$force" = false ]; then
        echo
        read -p "Continue with installation? [Y/n] " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Nn]$ ]]; then
            log_info "Installation cancelled"
            exit 0
        fi
    fi

    # Download and install
    local binary_path
    binary_path=$(download_binary "$platform")
    install_binary "$binary_path"

    # Post-installation checks
    check_path
    test_installation

    # Success message
    echo
    log_success "Claude-reactor installation completed successfully!"
    log_info "Run 'claude-reactor --help' to get started"
}

# Run main function
main "$@"