#!/bin/bash
# Docker 管理腳本 - 啟動資料庫服務 (Linux/macOS)
# Usage: ./scripts/docker-up.sh

echo "🚀 Starting TalkRealm database services..."

# 檢查 Docker 是否安裝
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed"
    echo "Please install Docker from: https://docs.docker.com/get-docker/"
    exit 1
fi

# 檢查 Docker 是否正在運行
if ! docker info &> /dev/null; then
    echo "❌ Docker daemon is not running"
    echo "Please start Docker first"
    exit 1
fi

# 啟動服務
echo ""
echo "📦 Starting PostgreSQL and Redis containers..."
docker-compose up -d

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ Services started successfully!"
    echo ""
    echo "Service information:"
    echo "  PostgreSQL: localhost:5432"
    echo "    - Database: talkrealm"
    echo "    - Username: talkrealm"
    echo "    - Password: talkrealm_password"
    echo ""
    echo "  Redis: localhost:6379"
    echo "    - Password: talkrealm_redis_password"
    
    echo ""
    echo "🔍 Checking container status..."
    sleep 3
    docker-compose ps
    
    echo ""
    echo "💡 Useful commands:"
    echo "  View logs:    docker-compose logs -f"
    echo "  Stop services: ./scripts/docker-down.sh"
    echo "  Restart:      docker-compose restart"
else
    echo ""
    echo "❌ Failed to start services"
    exit 1
fi
