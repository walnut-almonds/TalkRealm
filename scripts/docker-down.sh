#!/bin/bash
# 停止資料庫服務

set -e

echo "⏸️  停止資料庫服務..."
docker-compose down
echo "✅ 服務已停止"
echo ""
echo "💡 若要同時刪除資料: docker-compose down -v"
