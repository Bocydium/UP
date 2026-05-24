#!/bin/bash
set -e

REPO="github.com/aapollo/up"
INSTALL_DIR="/usr/local/bin"
LOCAL_INSTALL_DIR="$HOME/.local/bin"

# Colors
CYAN='\033[36m'
GREEN='\033[32m'
YELLOW='\033[33m'
RED='\033[31m'
RESET='\033[0m'

header() {
    echo -e "${CYAN}==>${RESET} $1"
}

step() {
    echo -e "  ${YELLOW}->${RESET} $1"
}

success() {
    echo -e "  ${GREEN}✓${RESET} $1"
}

error() {
    echo -e "  ${RED}✗${RESET} $1"
}

# Check for Go
if ! command -v go &> /dev/null; then
    error "Go is not installed. Please install Go first: https://go.dev/dl/"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
step "Found Go $GO_VERSION"

# Determine install location
if [ -w "$INSTALL_DIR" ] || [ "$EUID" -eq 0 ]; then
    TARGET_DIR="$INSTALL_DIR"
    USE_SUDO=false
else
    TARGET_DIR="$LOCAL_INSTALL_DIR"
    USE_SUDO=false
    step "No write access to $INSTALL_DIR, installing to $TARGET_DIR"
    mkdir -p "$TARGET_DIR"
fi

# Check if target dir is in PATH
if [[ ":$PATH:" != *":$TARGET_DIR:"* ]]; then
    error "$TARGET_DIR is not in your PATH"
    echo "    Add this to your ~/.bashrc or ~/.zshrc:"
    echo "    export PATH=\"$TARGET_DIR:\$PATH\""
fi

header "Building up..."

# Create temp build dir
BUILD_DIR=$(mktemp -d)
trap "rm -rf $BUILD_DIR" EXIT

# Clone or use local source
if [ -d "cmd/up" ] && [ -f "go.mod" ]; then
    step "Using local source..."
    cp -r . "$BUILD_DIR/"
    cd "$BUILD_DIR"
else
    step "Cloning $REPO..."
    git clone --depth 1 "https://$REPO.git" "$BUILD_DIR/up" 2>/dev/null || true
    cd "$BUILD_DIR/up"
fi

# Build
step "Building binary..."
go build -ldflags="-s -w" -o up ./cmd/up

# Install
step "Installing to $TARGET_DIR..."
if [ "$USE_SUDO" = true ]; then
    sudo install -Dm755 up "$TARGET_DIR/up"
else
    install -Dm755 up "$TARGET_DIR/up"
fi

success "up installed to $TARGET_DIR/up"

# Verify
if command -v up &> /dev/null; then
    INSTALLED_VERSION=$(up --help | head -1)
    success "$INSTALLED_VERSION"
else
    error "up is not in your PATH. Restart your shell or run:"
    echo "    export PATH=\"$TARGET_DIR:\$PATH\""
fi

echo ""
echo -e "${GREEN}Installation complete!${RESET}"
echo ""
echo "Quick start:"
echo "  up inst neovim       # Install a package"
echo "  up plan              # Dry-run pending updates"
echo "  up upda              # Update everything"
echo "  up cache             # Show cache size"
echo "  up cache clean       # Clean old builds"
echo "  up --help            # Show all commands"
