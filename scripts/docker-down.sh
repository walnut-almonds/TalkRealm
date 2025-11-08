#!/bin/bash
# Docker 管理腳本 - 停止資料庫服務 (Linux/macOS)
# Usage: ./scripts/docker-down.sh

echo "🛑 Stopping TalkRealm database services..."

docker-compose down

if [ $? -eq 0 ]; then
    echo "✅ Services stopped successfully!"
    echo ""
    echo "💡 To remove all data volumes, run:"
    echo "  docker-compose down -v"
else
    echo "❌ Failed to stop services"
    exit 1
fi
