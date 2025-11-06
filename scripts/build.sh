#!/bin/bash

# Build script for TalkRealm

echo "Building TalkRealm..."

# Set build variables
APP_NAME="talkrealm"
OUTPUT_DIR="bin"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GO_VERSION=$(go version | awk '{print $3}')

# Create output directory
mkdir -p ${OUTPUT_DIR}

# Build for current platform
echo "Building for current platform..."
go build -ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GoVersion=${GO_VERSION}" \
  -o ${OUTPUT_DIR}/${APP_NAME} \
  cmd/server/main.go

echo "Build complete: ${OUTPUT_DIR}/${APP_NAME}"

# Optional: Build for multiple platforms
# Uncomment the following lines if you want cross-platform builds

# echo "Building for Linux..."
# GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=${VERSION}" \
#   -o ${OUTPUT_DIR}/${APP_NAME}-linux-amd64 \
#   cmd/server/main.go

# echo "Building for Windows..."
# GOOS=windows GOARCH=amd64 go build -ldflags "-X main.Version=${VERSION}" \
#   -o ${OUTPUT_DIR}/${APP_NAME}-windows-amd64.exe \
#   cmd/server/main.go

# echo "Building for macOS..."
# GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.Version=${VERSION}" \
#   -o ${OUTPUT_DIR}/${APP_NAME}-darwin-amd64 \
#   cmd/server/main.go
