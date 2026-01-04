#!/bin/bash

# TalkRealm 快速啟動腳本

echo "🚀 啟動 TalkRealm..."
echo ""

# 檢查 Go 是否已安裝
if ! command -v go &> /dev/null; then
    echo "❌ 錯誤: 未安裝 Go"
    echo "請先安裝 Go: https://golang.org/dl/"
    exit 1
fi

# 檢查資料庫是否運行
echo "📦 檢查資料庫連接..."

# 進入專案目錄
cd "$(dirname "$0")/.."

# 確保依賴已安裝
echo "📥 確認依賴..."
go mod download

# 執行資料庫遷移
echo "🗄️  執行資料庫遷移..."
go run scripts/migrate.go

# 建置伺服器
echo "🔨 建置伺服器..."
go build -o bin/server cmd/server/main.go

# 啟動伺服器
echo ""
echo "✅ 啟動伺服器..."
echo "📡 API 伺服器: http://localhost:8080"
echo "🌐 前端介面: http://localhost:8080"
echo "📚 API 文件: http://localhost:8080/health"
echo ""
echo "按 Ctrl+C 停止伺服器"
echo ""

./bin/server
