#!/bin/bash
# TalkRealm 開發腳本集合
# 方便快速執行各種常用操作

set -e

# 顏色定義
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
RED='\033[0;31m'
NC='\033[0m'

# 顯示使用說明
show_help() {
  echo -e "${CYAN}TalkRealm 開發工具${NC}"
  echo ""
  echo "使用方法: ./scripts/dev.sh [command]"
  echo ""
  echo "可用指令:"
  echo "  up          - 啟動 Docker 資料庫服務"
  echo "  down        - 停止 Docker 資料庫服務"
  echo "  logs        - 查看 Docker 日誌"
  echo "  reset       - 重置資料庫 (清空資料)"
  echo "  migrate     - 執行資料庫遷移"
  echo "  build       - 編譯專案"
  echo "  run         - 啟動伺服器"
  echo "  test        - 執行 API 測試"
  echo "  clean       - 清理編譯檔案"
  echo "  help        - 顯示此說明"
  echo ""
  echo "範例:"
  echo "  ./scripts/dev.sh up      # 啟動資料庫"
  echo "  ./scripts/dev.sh run     # 啟動伺服器"
  echo "  ./scripts/dev.sh test    # 測試 API"
}

# Docker 啟動
docker_up() {
  echo -e "${CYAN}🚀 啟動資料庫服務...${NC}"
  docker-compose up -d
  echo -e "${GREEN}✅ 資料庫服務已啟動${NC}"
  echo "PostgreSQL: localhost:5432"
  echo "Redis: localhost:6379"
}

# Docker 停止
docker_down() {
  echo -e "${YELLOW}⏸️  停止資料庫服務...${NC}"
  docker-compose down
  echo -e "${GREEN}✅ 資料庫服務已停止${NC}"
}

# Docker 日誌
docker_logs() {
  echo -e "${CYAN}📋 查看 Docker 日誌...${NC}"
  docker-compose logs -f
}

# 重置資料庫
docker_reset() {
  echo -e "${RED}⚠️  警告: 這將刪除所有資料！${NC}"
  read -p "確定要繼續嗎？ (y/N): " -n 1 -r
  echo
  if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}重置資料庫...${NC}"
    docker-compose down -v
    docker-compose up -d
    sleep 3
    go run scripts/migrate.go
    echo -e "${GREEN}✅ 資料庫已重置${NC}"
  else
    echo "已取消"
  fi
}

# 資料庫遷移
migrate() {
  echo -e "${CYAN}📦 執行資料庫遷移...${NC}"
  go run scripts/migrate.go
  echo -e "${GREEN}✅ 遷移完成${NC}"
}

# 編譯專案
build() {
  echo -e "${CYAN}🔨 編譯專案...${NC}"
  ./scripts/build.sh
  echo -e "${GREEN}✅ 編譯完成${NC}"
}

# 啟動伺服器
run() {
  echo -e "${CYAN}🚀 啟動伺服器...${NC}"
  go run cmd/server/main.go
}

# 執行測試
test_api() {
  echo -e "${CYAN}🧪 執行 API 測試...${NC}"
  ./scripts/test.sh
}

# 清理編譯檔案
clean() {
  echo -e "${YELLOW}🧹 清理編譯檔案...${NC}"
  rm -rf bin/
  echo -e "${GREEN}✅ 清理完成${NC}"
}

# 主程式
case "${1:-help}" in
  up)
    docker_up
    ;;
  down)
    docker_down
    ;;
  logs)
    docker_logs
    ;;
  reset)
    docker_reset
    ;;
  migrate)
    migrate
    ;;
  build)
    build
    ;;
  run)
    run
    ;;
  test)
    test_api
    ;;
  clean)
    clean
    ;;
  help|--help|-h)
    show_help
    ;;
  *)
    echo -e "${RED}❌ 未知指令: $1${NC}"
    echo ""
    show_help
    exit 1
    ;;
esac
