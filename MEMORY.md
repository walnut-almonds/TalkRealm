# TalkRealm — Agent Memory

## Quick Facts
- Module: `github.com/walnut-almonds/talkrealm`
- Go version: 1.25.5
- Web framework: Gin v1.10.0
- ORM: GORM + PostgreSQL (`gorm.io/driver/postgres`)
- WebSocket: `gorilla/websocket`
- Auth: `golang-jwt/jwt/v5`
- Config: Viper
- Logger: `go.uber.org/zap`
- 目前是 **monolith**，架構目標是漸進拆分為微服務（見 `plan.md`）

## Commands
```bash
make check        # 全部檢查（lint + build + test）
```

## Architecture Notes
- `internal/server/server.go`：DI 組裝、路由設定的主入口
- `internal/websocket/manager.go`：channel 訂閱索引（`channelSubscriptions map[uint]map[*Client]bool`），O(1) 廣播；jwtManager 注入用於 identify op
- WS 協議：client→server op: `identify`, `heartbeat`, `subscribe`, `unsubscribe`, `typing_start`；server→client op: `hello`, `ready`, `heartbeat_ack`, `message_create`, `typing_start`, `presence_update`, `error`
- WS 端點：`GET /api/v1/ws`（無需 JWT 中間件，由 identify op 驗證）
- identify flow：client 連線 → server 送 `hello`（heartbeat_interval=30000ms）→ client 送 `identify`（token + channels[]）→ server 驗證 JWT，送 `ready` + 廣播 `presence_update online`
- `pkg/auth/jwt.go`：JWTManager，sign / verify token
- `pkg/database/database.go`：GORM DB singleton
- REST API 路由前綴：`/api/v1/`
- WebSocket 端點：`GET /api/v1/ws`（token 透過 identify op 傳遞，不再放 query string）
- 目前訊息分頁是 offset，計畫改為 cursor-based（before message_id）

## Pitfalls
- WS Manager 已有 channel subscription index（Phase 1 完成）；Presence 系統目前無 Redis（狀態不持久化）
- `message_service.go` 中 WS Manager 以 interface 注入（避免循環依賴），需 `SetWebSocketManager()` 設定
- handler.go 仍有 TODO stub functions（已被 user_handler.go 等各自的 handler 取代）

## Decisions
- MQ 選擇 NATS JetStream（輕量，適合小團隊），備選 Kafka
- 物件儲存選 Minio（self-hosted S3-compatible），生產可換 AWS S3
- 語音選 LiveKit（WebRTC SFU）
- 檔案上傳採 Pre-signed URL 模式，API Server 不處理 binary

## Last Updated
2026-04-30 — 實作 WebSocket 改善（Phase 1）：channel index、identify/hello/ready、typing_start、presence_update、heartbeat/heartbeat_ack；go.mod 更新至 go1.26.1。
