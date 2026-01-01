#!/bin/bash
set -e

VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
COMMIT=${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo "none")}
DATE=$(date -u +%Y-%m-%d)

LDFLAGS="-X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.Date=${DATE}"

echo "Building termiflow ${VERSION}..."

# Create dist directory
mkdir -p dist

# Build for current platform
echo "Building for current platform..."
go build -ldflags "${LDFLAGS}" -o dist/termiflow ./cmd/termiflow

# Cross-compile if --all flag is passed
if [ "$1" = "--all" ]; then
    echo "Building for Linux (amd64)..."
    GOOS=linux GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o dist/termiflow-linux-amd64 ./cmd/termiflow

    echo "Building for Linux (arm64)..."
    GOOS=linux GOARCH=arm64 go build -ldflags "${LDFLAGS}" -o dist/termiflow-linux-arm64 ./cmd/termiflow

    echo "Building for macOS (amd64)..."
    GOOS=darwin GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o dist/termiflow-darwin-amd64 ./cmd/termiflow

    echo "Building for macOS (arm64)..."
    GOOS=darwin GOARCH=arm64 go build -ldflags "${LDFLAGS}" -o dist/termiflow-darwin-arm64 ./cmd/termiflow

    echo "Building for Windows (amd64)..."
    GOOS=windows GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o dist/termiflow-windows-amd64.exe ./cmd/termiflow
fi

echo "Build complete!"
ls -la dist/
