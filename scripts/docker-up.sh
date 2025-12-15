#!/bin/bash
# 啟動資料庫服務

set -e

echo "🚀 啟動 TalkRealm 資料庫服務..."

# 檢查 Docker
if ! command -v docker &> /dev/null; then
    echo "❌ Docker 未安裝"
    exit 1
fi

if ! docker info &> /dev/null; then
    echo "❌ Docker 未運行，請先啟動 Docker"
    exit 1
fi

# 啟動服務
docker-compose up -d

echo ""
echo "✅ 服務已啟動！"
echo ""
echo "PostgreSQL: localhost:5432"
echo "Redis:      localhost:6379"
echo ""
echo "查看日誌: docker-compose logs -f"
