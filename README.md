# TalkRealm

一個開源的即時通訊平台，提供文字聊天、語音通話、視訊分享等功能。

[![Go](https://img.shields.io/badge/Go-1.26.1-00ADD8?logo=go)](https://go.dev/)
[![Vue](https://img.shields.io/badge/Vue-3-4FC08D?logo=vue.js)](https://vuejs.org/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

## 功能特色

| 功能 | 狀態 |
|------|------|
| 📝 即時文字聊天（WebSocket） | ✅ 完成 |
| 🏠 社群（Guild）建立與管理 | ✅ 完成 |
| 📁 文字 / 語音頻道分類 | ✅ 完成 |
| 👥 成員系統（角色/踢人/邀請連結） | ✅ 完成 |
| 🔔 正在輸入提示 / 線上狀態 | ✅ 完成 |
| 📎 圖片 / 檔案上傳（Minio） | ✅ 完成 |
| 🎤 即時語音（LiveKit WebRTC SFU） | ✅ 完成 |
| 📷 攝影機 & 螢幕分享 | ✅ 完成 |
| 🖥️ 視訊視窗（固定、音量、畫質控制） | ✅ 完成 |
| 🎮 單字學習遊戲（釋義填字 / 字母盤 / 每日挑戰 + 排行榜） | ✅ 完成 |
| 🔐 Google OAuth 登入 | ✅ 完成 |
| 🔑 JWT 身份驗證（存取 + 刷新 token） | ✅ 完成 |

## 技術棧

### 後端
- **語言 / 框架**：Go 1.26.1 + [Gin](https://github.com/gin-gonic/gin) v1.10.0
- **ORM**：[GORM](https://gorm.io/) + PostgreSQL
- **快取**：Redis
- **WebSocket**：[gorilla/websocket](https://github.com/gorilla/websocket)
- **語音 / 視訊**：[LiveKit](https://livekit.io/) SFU（WebRTC）
- **物件儲存**：[Minio](https://min.io/)（S3-compatible，Pre-signed URL 上傳）
- **認證**：JWT (`golang-jwt/jwt/v5`) + Google OAuth
- **日誌**：[Zap](https://github.com/uber-go/zap)
- **設定**：Viper

### 前端
- **框架**：[Vue 3](https://vuejs.org/)（Composition API + `<script setup>`）
- **狀態管理**：[Pinia](https://pinia.vuejs.org/)
- **建置工具**：[Vite](https://vitejs.dev/)
- **語音 SDK**：[livekit-client](https://github.com/livekit/client-sdk-js)
- **樣式**：自訂 CSS（Dark Theme）

## 系統架構

```
瀏覽器（Vue 3 SPA）
  │── REST  /api/v1/*   ──▶  Gin HTTP Server
  │── WS    /api/v1/ws  ──▶  WebSocket Manager（即時訊息 / 語音狀態）
  └── WebRTC             ──▶  LiveKit SFU（語音 / 視訊串流）

Gin HTTP Server
  ├── Handler Layer      (internal/handler/)
  ├── Service Layer      (internal/service/)
  ├── Repository Layer   (internal/repository/)
  └── Model Layer        (internal/model/)

基礎設施
  ├── PostgreSQL  — 主要資料儲存
  ├── Redis       — 快取 / Refresh Token 黑名單
  ├── Minio       — 檔案物件儲存
  └── LiveKit     — WebRTC SFU 語音/視訊
```

## 專案結構

```
TalkRealm/
├── cmd/server/           # 主程式入口
├── internal/
│   ├── handler/          # HTTP 請求處理器
│   ├── service/          # 業務邏輯
│   ├── repository/       # 資料存取層
│   ├── model/            # GORM 資料模型
│   ├── middleware/        # JWT 認證、Rate Limit 中介軟體
│   ├── websocket/        # WebSocket 管理器 & 客戶端
│   └── server/           # DI 組裝 & 路由設定
├── pkg/
│   ├── auth/             # JWT 工具
│   ├── config/           # 設定解析（Viper）
│   ├── database/         # GORM 連線初始化
│   ├── redis/            # Redis 客戶端
│   ├── storage/          # Minio 客戶端
│   ├── voice/            # LiveKit token 產生
│   └── logger/           # Zap 日誌
├── web/
│   └── src/
│       ├── components/   # Vue 元件（聊天室、頻道列表、語音列 …）
│       ├── composables/  # useVoice、useWebSocket …
│       ├── stores/       # Pinia stores（useAppStore、useVoiceStore …）
│       ├── api/          # API 用戶端封裝
│       └── styles/       # 全域 CSS
├── configs/              # 設定範本（config.example.yaml）
├── docs/                 # 架構 & 資料庫文件
├── docker-compose.yml        # 開發環境（含 LiveKit、Minio）
├── docker-compose.prod.yml   # 生產環境
└── Makefile
```

## 快速開始

### 前置需求

- Go 1.26+
- Docker & Docker Compose
- Node.js 20+（前端開發）

### 使用 Docker Compose 啟動（推薦）

```bash
# 1. 複製專案
git clone https://github.com/walnut-almonds/TalkRealm.git
cd TalkRealm

# 2. 啟動基礎設施（PostgreSQL、Redis、Minio、LiveKit）
docker compose up -d

# 3. 複製設定並調整
cp configs/config.example.yaml configs/config.yaml
# 編輯 configs/config.yaml：資料庫、Redis、JWT secret、Minio、LiveKit 等

# 4. 資料庫遷移
go run scripts/migrate.go

# 5. 啟動後端
go run cmd/server/main.go

# 6. 啟動前端（另一個終端）
cd web && npm install && npm run dev
```

前端：`http://localhost:5173`　後端 API：`http://localhost:8080`

### 只需後端（已有資料庫）

```bash
cp configs/config.example.yaml configs/config.yaml
# 填入 DB / Redis 連線資訊

go run scripts/migrate.go
go run cmd/server/main.go
```

## 設定說明

`configs/config.yaml` 主要設定項目：

```yaml
server:
  port: 8080

database:
  host: localhost
  user: talkrealm
  password: YOUR_PASSWORD
  dbname: talkrealm

redis:
  host: localhost
  password: YOUR_REDIS_PASSWORD

jwt:
  secret: YOUR_JWT_SECRET      # 請務必更換

minio:
  endpoint: localhost:9000
  access_key: minioadmin
  secret_key: minioadmin
  bucket: talkrealm
  public_read: true            # true = 公開讀取（搭配 nginx/CDN）

livekit:
  host: ws://localhost:7880
  api_key: devkey
  api_secret: secret

oauth:
  google:
    client_id: YOUR_CLIENT_ID
    client_secret: YOUR_CLIENT_SECRET
    redirect_url: http://localhost:8080/api/v1/auth/google/callback
```

## 開發指令

```bash
make check          # lint + build + test 全套檢查
make fmt            # 自動格式化
make fix            # golangci-lint --fix
make lint           # 只執行 lint
go test ./...       # 執行測試
go build -o bin/talkrealm cmd/server/main.go  # 建置
```

## API 文件

啟動後端後可存取：

- **Swagger UI**：`http://localhost:8080/swagger/index.html`
- **API 指南**：[api/API_GUIDE.md](api/API_GUIDE.md)
- **線上預覽**：[OpenAPI Viewer](https://min0625.github.io/openapi-viewer/?url=https://raw.githubusercontent.com/walnut-almonds/TalkRealm/main/docs/openapi/swagger.json)

## 生產部署

使用 `docker-compose.prod.yml`，需提供 `.env`：

```env
POSTGRES_PASSWORD=your_strong_password
REDIS_PASSWORD=your_redis_password
MINIO_ACCESS_KEY=your_minio_key
MINIO_SECRET_KEY=your_minio_secret
MINIO_PUBLIC_ENDPOINT=https://media.example.com
LIVEKIT_API_KEY=your_livekit_key
LIVEKIT_API_SECRET=your_livekit_secret
```

```bash
docker compose -f docker-compose.prod.yml --env-file .env up -d
```

詳細說明：[docs/docker.md](docs/docker.md)

## 相關文件

- [架構設計](docs/architecture.md)
- [資料庫設計](docs/database.md)
- [Docker 部署指南](docs/docker.md)
- [測試說明](docs/test.md)

## 貢獻

歡迎 Issue 與 Pull Request！

1. Fork 本專案
2. 建立功能分支：`git checkout -b feature/my-feature`
3. 提交：`git commit -m 'feat: add my feature'`
4. 推送並開 PR

## 授權

[Apache License 2.0](LICENSE)

---

專案連結：[https://github.com/walnut-almonds/TalkRealm](https://github.com/walnut-almonds/TalkRealm)
