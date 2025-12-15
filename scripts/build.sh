#!/bin/bash
# 編譯專案

set -e

APP_NAME="talkrealm"
OUTPUT_DIR="bin"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')

echo "🔨 編譯 TalkRealm..."

mkdir -p ${OUTPUT_DIR}

go build -ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}" \
  -o ${OUTPUT_DIR}/${APP_NAME} \
  cmd/server/main.go

echo "✅ 編譯完成: ${OUTPUT_DIR}/${APP_NAME}"
