# TalkRealm

一個類似 Discord 的即時通訊平台，支援文字聊天和語音通話功能。

## 專案簡介

TalkRealm 是一個開源的即時通訊解決方案，提供以下核心功能：

- 🏠 **伺服器/社群管理**：建立和管理多個社群空間
- 💬 **文字聊天**：即時文字訊息傳送與接收
- 🎤 **語音聊天室**：高品質的即時語音通訊
- 👥 **使用者系統**：註冊、登入、個人資料管理
- 🔐 **權限管理**：角色與權限控制系統
- 📁 **頻道系統**：文字頻道和語音頻道分類

## 技術棧

- **後端框架**: [Gin](https://github.com/gin-gonic/gin) - 高效能 HTTP Web 框架
- **WebSocket**: [Gorilla WebSocket](https://github.com/gorilla/websocket) - 即時雙向通訊
- **資料庫**: PostgreSQL + Redis
- **身份驗證**: JWT (JSON Web Tokens)
- **語音處理**: WebRTC
- **日誌**: Zap - 高效能結構化日誌

## 📐 系統架構

查看完整的系統設計圖：[架構文件](docs/architecture.md)

- 系統架構圖
- 資料模型關係圖
- API 請求流程圖
- WebSocket 即時通訊流程
- 部署架構圖

## 專案結構

```
TalkRealm/
├── cmd/
│   └── server/          # 主程式入口
├── internal/            # 私有應用程式碼
│   ├── server/         # HTTP 伺服器設定
│   ├── handler/        # HTTP 處理器
│   ├── service/        # 業務邏輯層
│   ├── repository/     # 資料存取層
│   ├── model/          # 資料模型
│   └── middleware/     # 中介軟體
├── pkg/                # 可重用的公共庫
│   ├── logger/         # 日誌工具
│   └── config/         # 配置管理
├── api/                # API 定義檔案 (OpenAPI/Swagger)
├── configs/            # 配置檔案
├── scripts/            # 建置和部署腳本
├── web/                # 前端靜態資源 (可選)
├── go.mod              # Go 模組定義
└── README.md           # 專案說明文件
```

## 快速開始

### 前置需求

- Go 1.25.5 或更高版本
- Docker & Docker Compose（推薦）或
- PostgreSQL 14+ 和 Redis 6+（手動安裝）

### 方法一：使用 Docker（推薦）⭐

**最快速的開始方式！**

1. 克隆專案
```bash
git clone https://github.com/walnut-almonds/TalkRealm.git
cd TalkRealm
```

2. 啟動資料庫服務
```powershell
# Windows
.\scripts\docker-up.ps1

# Linux/macOS
chmod +x scripts/*.sh
./scripts/docker-up.sh
```

3. 準備配置並安裝依賴
```bash
cp configs/config.docker.yaml configs/config.yaml
go mod download
```

4. 執行資料庫遷移
```bash
go run scripts/migrate.go
```

5. 啟動服務
```bash
go run cmd/server/main.go
```

伺服器將在 `http://localhost:8080` 啟動

📖 **詳細說明**: 查看 [Docker 指南](docs/docker.md)

### 方法二：手動安裝資料庫

1. 克隆專案
```bash
git clone https://github.com/walnut-almonds/TalkRealm.git
cd TalkRealm
```

2. 安裝依賴
```bash
go mod download
```

3. 配置環境變數
```bash
cp configs/config.example.yaml configs/config.yaml
# 編輯 config.yaml 設定資料庫連線等資訊
```

4. 執行資料庫遷移
```bash
go run scripts/migrate.go
```

5. 啟動服務
```bash
go run cmd/server/main.go
```

伺服器將在 `http://localhost:8080` 啟動

## API 文件

啟動服務後，可以訪問以下端點查看 API 文件：
- Swagger UI: `http://localhost:8080/swagger/index.html`
- [線上預覽（OpenAPI Viewer）](https://min0625.github.io/openapi-viewer/?url=https://raw.githubusercontent.com/walnut-almonds/TalkRealm/main/docs/openapi/swagger.json)

## 開發

### 執行測試
```bash
go test ./...
```

### 建置
```bash
go build -o bin/talkrealm cmd/server/main.go
```

### 程式碼規範
本專案遵循 Go 官方的程式碼規範，請在提交前執行：
```bash
go fmt ./...
go vet ./...
golangci-lint run
```

## 貢獻指南

歡迎提交 Issue 和 Pull Request！

1. Fork 本專案
2. 建立你的特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交你的更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 開啟一個 Pull Request

## 授權

本專案採用 Apache License 2.0 授權 - 詳見 [LICENSE](LICENSE) 檔案

## 聯繫方式

專案連結: [https://github.com/walnut-almonds/TalkRealm](https://github.com/walnut-almonds/TalkRealm)

## 致謝

感謝所有為本專案做出貢獻的開發者！
