#!/bin/bash
set -e

# termiflow installer script

INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
REPO="termiflow/termiflow"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

case $OS in
    linux|darwin)
        ;;
    mingw*|msys*|cygwin*)
        OS="windows"
        ;;
    *)
        echo "Unsupported OS: $OS"
        exit 1
        ;;
esac

BINARY="termiflow-${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then
    BINARY="${BINARY}.exe"
fi

echo "Installing termiflow for ${OS}/${ARCH}..."

# Check if Go is installed
if command -v go &> /dev/null; then
    echo "Go found, installing via go install..."
    go install github.com/${REPO}/cmd/termiflow@latest
    echo "termiflow installed successfully!"
    echo "Run 'termiflow config init' to get started."
    exit 0
fi

# Otherwise try to download pre-built binary
echo "Go not found, attempting to download pre-built binary..."

# Get latest release URL
LATEST_URL="https://github.com/${REPO}/releases/latest/download/${BINARY}"

# Download binary
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

echo "Downloading from ${LATEST_URL}..."
if command -v curl &> /dev/null; then
    curl -fsSL "$LATEST_URL" -o "${TEMP_DIR}/termiflow"
elif command -v wget &> /dev/null; then
    wget -q "$LATEST_URL" -O "${TEMP_DIR}/termiflow"
else
    echo "Neither curl nor wget found. Please install one of them."
    exit 1
fi

# Make executable
chmod +x "${TEMP_DIR}/termiflow"

# Install
echo "Installing to ${INSTALL_DIR}/termiflow..."
if [ -w "$INSTALL_DIR" ]; then
    mv "${TEMP_DIR}/termiflow" "${INSTALL_DIR}/termiflow"
else
    sudo mv "${TEMP_DIR}/termiflow" "${INSTALL_DIR}/termiflow"
fi

echo "termiflow installed successfully!"
echo "Run 'termiflow config init' to get started."
