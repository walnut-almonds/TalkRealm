# TalkRealm TODO List

> 基於 2026-04-30 架構重設計，詳細規劃見 [`plan.md`](./plan.md)。

---

## 🔴 Phase 1 — 強化現有 Monolith（高優先）

### WebSocket 改善
- [x] **WS Channel Index**：Manager 加入 `channelSubscriptions map[uint]map[*Client]bool`，只推訊息給訂閱該頻道的 client，O(1) 查找
- [x] **WS Identify Op**：連線後 server 發 `hello`（帶 heartbeat_interval），client 發 `identify`（帶 JWT），server 回 `ready`，並訂閱初始頻道列表
- [x] **Typing Indicator**：實作 `typing_start` op（WS Client → Server → 同頻道廣播，排除發送者）
- [x] **Presence 系統**：identify 後廣播 `presence_update` online，斷線後廣播 offline（Redis 版本留待 Redis 整合後補充）
- [x] **WS Heartbeat / Reconnect**：標準化 `heartbeat` / `heartbeat_ack`，前端自動重連邏輯

### Redis 整合
- [x] **引入 Redis Client**：在 `pkg/redis/` 封裝連線，config 加入 Redis 設定
- [x] **User Server Mapping**：Chat Server 連線時 `SET user:{userID}:server {serverID} EX 86400`
- [x] **Guild Online Set**：`SADD guild:{guildID}:online {userID}` / `SREM` on disconnect
- [x] **Rate Limiting Middleware**：`INCR ratelimit:{userID}:msg` TTL 1s，超過 10 則回 429

### API 補全
- [x] **Cursor-based Pagination**：`GET /api/v1/channels/{id}/messages` 改為 `?before={id}&limit=50`，移除 offset 分頁
- [x] **Guild 邀請碼系統**
  - [x] DB migration：`guild_invites` 表
  - [x] `POST /api/v1/guilds/{id}/invites`：生成邀請碼
  - [x] `GET /api/v1/invites/{code}`：查詢邀請碼資訊
  - [x] `POST /api/v1/guilds/join-by-invite`：透過邀請碼加入
- [x] **Refresh Token**
  - [x] DB migration：`refresh_tokens` 表
  - [x] `POST /api/v1/auth/refresh`：換發 access token
  - [x] `POST /api/v1/auth/logout`：revoke refresh token
- [x] **User 公開資料 API**：`GET /api/v1/users/{id}`（不含 email）
- [x] **訊息 Attachment 欄位**：Message 回應加入 `attachments []` 陣列

### 權限系統（RBAC）
- [x] **Permission 中介層**：封裝 `RequireGuildRole(minRole)` middleware
- [x] **頻道操作權限**：create / update / delete channel 需 `admin` 以上
- [x] **成員管理權限**：kick / update role 需 `admin` 以上
- [x] **訊息刪除權限**：允許 channel admin 刪除他人訊息

---

## 🟡 Phase 2 — MQ 整合（中期）

### NATS JetStream
- [ ] **引入 NATS client**：`pkg/mq/` 封裝 Publish / Subscribe
- [ ] **`chat.record` Stream**：Chat Server 發送訊息時 publish，供 Message Persistence Consumer 消費寫 DB
- [ ] **`chat.server.{id}` Topic**：Chat Server 收到目標 user 在另一 server 時，publish 到對應 topic
- [ ] **Message Persistence Consumer**：獨立 goroutine（或獨立 binary），consume `chat.record` → 批次寫入 DB
- [ ] **跨 Server 訊息路由**：Chat Server subscribe 自身 topic，收到後透過 WS Manager 推給 client

### 服務拆分（Binary 層）
- [ ] **ws-gateway binary**：僅處理 WebSocket 連線管理，與 api-server 分開
- [ ] **api-server binary**：僅處理 REST API，不含 WS 邏輯
- [ ] Docker Compose 分別啟動兩個服務，設定不同 port

---

## 🟠 Phase 3 — File Service & Voice（中期）

### File Access Service（`/api/v1/files`）
- [ ] **Minio 整合**：`pkg/storage/` 封裝 Minio SDK，config 加入 Minio 設定
- [ ] **DB migration**：`files` 表、`message_attachments` 表
- [ ] `POST /api/v1/files/presign`：生成 Upload Pre-signed URL（15 min 有效）
- [ ] `GET /api/v1/files/{id}`：取得檔案 metadata
- [ ] `GET /api/v1/files/{id}/url`：生成 Download Pre-signed URL（私有檔案）
- [ ] `DELETE /api/v1/files/{id}`：刪除檔案（Minio + DB）
- [ ] **WS file 訊息類型**：`send_message` op 支援 `type: "file"`，帶 `file_id`

### LiveKit 語音整合
- [ ] **引入 LiveKit SDK**：`pkg/voice/` 封裝 token 生成
- [ ] `GET /api/v1/channels/{id}/voice/token`：回傳 LiveKit Room Token + URL
- [ ] **WS `voice_state_update` op**：廣播使用者加入/離開語音頻道事件
- [ ] Docker Compose 加入 LiveKit Server

---

## ⚪ Phase 4 — 微服務拆分（長期）

- [ ] **Auth Service 獨立**：獨立 repo / binary，其他服務透過 gRPC 或 JWT 驗證
- [ ] **Message Persistence Service 獨立**：獨立部署，只消費 MQ
- [ ] **File Access Service 獨立**：獨立 Deployment
- [ ] **API Gateway**：Nginx / Traefik 路由 `ws.talkrealm.com` 和 `api.talkrealm.com`
- [ ] **K8s HPA**：Chat Server 水平擴展，依連線數 autoscale
- [ ] **Observability**：Prometheus metrics + Grafana dashboard + OpenTelemetry tracing

---

## 🧪 測試 & 品質

- [ ] **Unit Tests**：Handler / Service / Repository 各層覆蓋率 > 70%
- [ ] **WS Integration Test**：`gorilla/websocket` 模擬 client，測試 identify → send_message → message_create 流程
- [ ] **Load Test**：k6 模擬 1000 concurrent WS 連線
- [ ] **E2E Test**：docker-compose 啟動全服務，API 端到端測試

---

## 🏗️ 基礎架構 & 部署

- [x] Dockerfile 與服務打包
- [x] Docker Compose 本地環境（Postgres + Redis）
- [x] Kubernetes Kustomize Base & Overlays
- [ ] **Docker Compose 加入 NATS**：`nats:latest` with JetStream enabled
- [ ] **Docker Compose 加入 Minio**：`minio/minio` latest
- [ ] **Docker Compose 加入 LiveKit**：`livekit/livekit-server`
- [ ] **CI/CD**：GitHub Actions — lint + test + build + push Docker image
- [ ] **K8s Secrets 管理**：使用 Sealed Secrets 或 External Secrets Operator

---

## 📚 文件

- [x] OpenAPI / Swagger 定義（`docs/openapi/`）
- [ ] **更新 Swagger**：補充所有新 API（files、invites、voice token、cursor pagination）
- [ ] **WebSocket Protocol 文件**：所有 op codes 的 request/response schema
- [ ] **架構圖更新**：用 draw.io 或 Mermaid 繪製新架構圖並放入 `docs/architecture.md`
- [ ] **SDK 生成**：嘗試 [microsoft/kiota](https://github.com/microsoft/kiota) 從 OpenAPI 生成 Go / TS SDK

---

## ✅ 已完成

- [x] 使用者系統（註冊、登入、JWT）
- [x] Guild CRUD + 成員管理
- [x] Channel CRUD
- [x] 基礎訊息 CRUD（REST）
- [x] WebSocket 連線與基礎廣播
- [x] Docker Compose 本地開發環境
- [x] Kubernetes 部署配置
- [x] OpenAPI 基礎定義
