# TalkRealm — Agent Memory

## Quick Facts
- Module: `github.com/walnut-almonds/talkrealm`
- Go version: 1.26.1
- Web framework: Gin v1.10.0
- ORM: GORM + PostgreSQL (`gorm.io/driver/postgres`)
- WebSocket: `gorilla/websocket`
- Auth: `golang-jwt/jwt/v5`
- OAuth: `golang.org/x/oauth2` (Google OAuth 已實作)
- Config: Viper
- Logger: `go.uber.org/zap`
- 目前是 **monolith**，架構目標是漸進拆分為微服務（見 `plan.md`）

## Commands
```bash
make check        # 全部檢查（lint + build + test）
```

## Architecture Notes
- `internal/server/server.go`：DI 組裝、路由設定的主入口
- `internal/websocket/manager.go`：channel 訂閱索引（`channelSubscriptions map[uint]map[*Client]bool`）+ guild 訂閱索引（`guildSubscriptions map[uint]map[*Client]bool`），O(1) 廣播；jwtManager 注入用於 identify op；identify 後自動呼叫 `SubscribeClientToUserGuilds` 訂閱所有 guild
- WS 協議：client→server op: `identify`, `heartbeat`, `subscribe`, `unsubscribe`, `typing_start`, `send_message`, `voice_state_update`；server→client op: `hello`, `ready`, `heartbeat_ack`, `message_create`, `message_update`, `message_delete`, `typing_start`, `presence_update`, `error`, `guild_update`, `guild_delete`, `guild_member_add`, `guild_member_remove`, `guild_member_update`, `channel_create`, `channel_update`, `channel_delete`, `voice_state_update`
- WS 端點：`GET /api/v1/ws`（無需 JWT 中間件，由 identify op 驗證）
- identify flow：client 連線 → server 送 `hello`（heartbeat_interval=30000ms）→ client 送 `identify`（token + channels[]）→ server 驗證 JWT，送 `ready` + 廣播 `presence_update online`
- `pkg/auth/jwt.go`：JWTManager，sign / verify token
- `pkg/database/database.go`：GORM DB singleton
- REST API 路由前綴：`/api/v1/`
- WebSocket 端點：`GET /api/v1/ws`（token 透過 identify op 傳遞，不再放 query string）
- 目前訊息分頁是 offset，計畫改為 cursor-based（before message_id）

## Pitfalls
- **File routes 404**：`/api/v1/files/*` 路由只在 Minio 初始化成功時才掛載。Minio 未設定或連線失敗會導致 `fileHandler == nil`，所有 file API 回傳 404 而非 503。已改為無條件掛載路由，Minio 不可用時回傳 503。若遇 404，先確認 Minio 容器是否正常運行及環境變數（`MINIO_ACCESS_KEY`、`MINIO_SECRET_KEY`、`MINIO_BUCKET`）是否設定正確。
- WS Manager 已有 channel subscription index（Phase 1 完成）；Presence 系統目前無 Redis（狀態不持久化）
- `message_service.go` 中 WS Manager 以 interface 注入（避免循環依賴），需 `SetWebSocketManager()` 設定；另有 `CreateMessageWS()` 供 WS `send_message` op 呼叫（`MessageSender` interface 注入到 Manager）
- handler.go 仍有 TODO stub functions（已被 user_handler.go 等各自的 handler 取代）
- 部署到 VPS 使用 `docker-compose.prod.yml` 時，`POSTGRES_PASSWORD` / `REDIS_PASSWORD` 需由同目錄 `.env` 或 `--env-file` 提供；否則 Compose 會以空字串替代，造成 postgres healthcheck 失敗。
- `golangci-lint --fix` + `whole-files: true` 坑：修改 `mocks.go` 會曝露所有既有的 nilnil 問題。已用 `//nolint:nilnil` 全部標記。新增 mock 方法必須一同加標記。
- `golangci-lint --fix` 會重新格式化 oauth_handler.go，造成 `NewRequestWithContext` 行號改變；`wsl_v5` 需在 `if err != nil { c.JSON(); return }` 的 return 前加空行。
- 前端拖曳檔案判斷不可只用 `e.dataTransfer.types.includes('Files')`：Safari/部分瀏覽器 `types` 是 `DOMStringList`，需改用 `types.contains('Files')` 或 `Array.from(types).includes('Files')`；另外要在 `window.dragover` `preventDefault()`，避免瀏覽器直接開啟拖入檔案。
- 若部署使用 `docker-compose.prod.yml`，必須包含 `livekit` service（`livekit:7880` 供 nginx upstream 轉發）。缺少該容器會導致 `wss://voice.../rtc/v1` 連線失敗，前端可能同時看到 `/rtc/v1/validate` CORS 錯誤（實際上常是 upstream 不可達）。
- LiveKit `--keys` 參數格式必須是 **`"key: secret"`**（冒號後必須有空白）。在 compose 建議整段 `command` 用單引號包住，避免 YAML 把 `:` 誤判為 mapping。

## Decisions
- MQ 選擇 NATS JetStream（輕量，適合小團隊），備選 Kafka
- 物件儲存選 Minio（self-hosted S3-compatible），生產可換 AWS S3
- 語音選 LiveKit（WebRTC SFU）
- 檔案上傳採 Pre-signed URL 模式，API Server 不處理 binary

## Last Updated
2026-05-08 — Voice 連線修正（prod 部署）：
- **`docker-compose.prod.yml`**：新增 `livekit` service（keys 讀取 `${LIVEKIT_API_KEY}:${LIVEKIT_API_SECRET}`、expose `7880`、開放 `7881` 與 `50100-50200/udp`）
- **`docker-compose.prod.yml`**：`nginx` 增加 `depends_on: livekit`
- **`nginx/nginx.conf`**：`location /rtc` CORS header 改為穩定 always 模式（含 `Vary: Origin`、`X-Livekit-*` headers），並保留 `OPTIONS -> 204`

2026-05-08 — LiveKit 語音整合（Phase 3）實作完成：
- **`pkg/voice/token.go`**：`Manager` 封裝 LiveKit token 生成；`GenerateRoomToken(channelID, userID, username)` 以 `channel:{id}` 作為 room name，回傳 `RoomTokenResponse{Token, URL, RoomName, Identity}`
- **`pkg/config/config.go`**：新增 `LiveKitConfig{APIKey, APISecret, URL, PublicURL, TokenTTL}`；`--dev` 模式預設 key=devkey / secret=secret
- **`internal/handler/voice_handler.go`**：`GET /api/v1/channels/:id/voice/token`（需認證）；LiveKit 未設定時回 503
- **WS `voice_state_update` op**（client→server / server→channel）：payload `{channel_id, action: "join"|"leave"}`；廣播格式 `{channel_id, user_id, username, action}`；實作在 `handleVoiceStateUpdate()` 方法
- **`internal/websocket/client.go`**：refactored `handleMessage` — `send_message` 邏輯提取為 `handleSendMessage()`，降低 cognitive complexity；新增 `handleVoiceStateUpdate()`
- **`docker-compose.yml`**：加入 `livekit/livekit-server:latest`（`--dev` 模式）；port 7880/7881/50100-50200(udp)
- **`configs/config.example.yaml`**：新增 livekit 區段（api_key=devkey, api_secret=secret, url=ws://livekit:7880, public_url=ws://localhost:7880, token_ttl=3600）
- **依賴新增**：`github.com/livekit/server-sdk-go/v2`（token 生成用 `github.com/livekit/protocol/auth`）
- **nolint**：`client.go` readPump/writePump 的 `conn.Close/SetReadDeadline/SetWriteDeadline/WriteMessage` 加 `//nolint:errcheck,gosec`（pre-existing pattern）

2026-05-04 — File Access Service（Phase 3）實作完成：
- **前端檔案上傳**：`+` 按鈕 → `<input type="file">` → `handleFileSelected()` → `uploadFile()` (presign→PUT→confirm) → chip 預覽 → `sendMessage()` 帶 `file_ids`
- **`api.js`**：新增 `presignUpload`, `uploadToMinio` (XHR PUT), `confirmUpload`, `getFileDownloadUrl`, `deleteFile`；`sendMessage` 新增 `fileIds` 參數
- **`app.js`**：`appState.pendingFileIds`、`handleFileSelected`、`uploadFile`、chip CRUD、`renderAttachments`（圖片縮圖/檔案下載）、`openAttachment`、`loadAttachmentImage`
- **`CreateMessageRequest`**：新增 `FileIDs []uint`；允許空 Content（有附件時）
- **`MessageService`**：新增 `SetFileService(FileService)`；`CreateMessage` 後建立 `MessageAttachment` 記錄
- **`minio.go` 陷阱**：曾出現重複 `package storage` 宣告（garbled）；用 `/tmp/write_xxx.go + go run` 修復（heredoc 會觸發 tab-completion）
- **`PresignPutURL`**：簽名改為 `(key, contentType string, expiry int)`
- **`internal/model/user.go`**：新增 `File`（含 `status`, `expires_at`, `last_accessed_at`）、`MessageAttachment` 模型
- **`pkg/config/config.go`**：新增 `MinioConfig`（endpoint/bucket/ssl/presign_expiry/max_file_size_mb/daily_upload_max/daily_bytes_max_mb/file_ttl_days/lru_evict_enabled）
- **`internal/repository/file_repository.go`**：CRUD + `CountByUserToday`, `SumBytesByUserToday`, `FindExpired`, `FindLRUByUser`, `TouchLastAccessed`, attachment CRUD
- **`internal/service/file_service.go`**：副檔名白名單驗證、單檔大小限制、每日 quota（Redis pipeline 優先 + DB fallback）、Pre-signed upload 流程（presign→pending→confirm→active）、TTL CleanupExpired
- **`internal/handler/file_handler.go`**：`PresignUpload`, `ConfirmUpload`, `GetFile`, `GetDownloadURL`, `DeleteFile`
- **Routes**：`POST /presign`, `POST /{id}/confirm`, `GET /{id}`, `GET /{id}/url`, `DELETE /{id}` 掛於 `/api/v1/files`（need Auth middleware）
- **DB AutoMigrate**：`File`, `MessageAttachment` 加入 `pkg/database/database.go`
- **`configs/config.example.yaml`**：新增完整 minio 區段
- **依賴新增**：`github.com/minio/minio-go/v7`, `github.com/google/uuid`
- **未完成**：WS `type: "file"` send_message、background LRU cleanup goroutine
- **新增 Notification Service**：獨立服務，消費 `topic:notification`，呼叫 Push Gateway (FCM/APNs)，寫 DB
- **新增 Translation Service**：獨立服務，由 Message Persistence Service 非同步派發，DeepL/GPT-4o，結果存 **Cassandra**（非 PostgreSQL 欄位）
- **MQ Topic 命名統一**：`topic:record`、`topic:server.{id}`、`topic:notification`（對齊圖示；實作選 NATS JetStream，圖示標 Kafka，等效替代）
- **Message Persistence Service** 寫 DB 後需非同步派發翻譯任務給 Translation Service
- plan.md 第七節新增 Cassandra `translation_messages` schema 與 PostgreSQL `notifications` schema
- WS 連線 URL 修正：token 透過 identify op 傳遞，不放 query string
- `index.html`：新增 **Guild Settings modal**（編輯社群 + 邀請碼 + 刪除）、**Join by Invite modal**、guilds sidebar 加入「透過邀請碼加入」按鈕
- `app.js`：實作 `showGuildSettings()`、`handleUpdateGuild()`、`handleDeleteGuild()`、`handleCreateInvite()`、`copyInviteCode()`、`showJoinByInviteModal()`、`handleJoinByInvite()`、`handleKickMember()`、`handleUpdateMemberRole()`
- `renderMembers()` 現在顯示角色 badge（owner/admin/moderator）及管理員操作按鈕（踢人/改角色），透過 `ROLE_LEVEL` 比較決定可見性
- `styles.css`：新增 `.role-badge`、`.btn-danger`、`.btn-icon-sm`、`.invite-code-box`、`.settings-section`、`.member-actions`

2026-04-30 — 前後端對接修正（Phase 1）：
- `config.js`：補齊所有 Phase 1 端點（REFRESH, LOGOUT, PUBLIC_USER, KICK_MEMBER, UPDATE_MEMBER_ROLE, CREATE_INVITE, GET_INVITE, JOIN_BY_INVITE）及 `REFRESH_TOKEN` storage key
- `api.js`：新增 `setRefreshToken/getRefreshToken`；login 儲存 refresh_token；`request()` 自動 401 → refresh → retry；新增 `refreshToken/logout/getPublicUser/kickMember/updateMemberRole/createInvite/getInvite/joinByInvite`
- `api.js` `createChannel` 改送 `topic`（後端 `CreateChannelRequest.Topic`），不再送 `description`
- `app.js`：`handleLogout()` 改為 async，先呼叫 `api.logout()` 再清除狀態，並清除 `REFRESH_TOKEN`；`updateChannelHeader()` 改用 `channel.topic`
- Channel 模型 JSON 欄位為 `topic`（非 `description`）；Guild 模型用 `description`

2026-04-30 — RBAC 權限系統（Phase 1 完成）：
- `middleware.RequireGuildRole(minRole, guildMemberRepo)` 封裝角色驗證，從 URL param `:id` 取得 guild ID，角色階層 member<moderator<admin<owner，並將 `guild_member` 存入 context
- `middleware.HasMinRole(userRole, minRole)` 公開函式，供 service/handler 重用
- `guildRoleLevel` map 與 `hasMinRole()` 在 `guild_service.go` 內部維護
- `ErrInsufficientPermission` 新增至 guild_service errors
- `KickMember`：admin 可踢低階成員，不能踢同級或高階
- `UpdateMemberRole`：admin 可設定 moderator/member；只有 owner 可設定 admin
- 路由層 middleware 已套用：`DELETE /guilds/:id/members/:userId`、`PUT /guilds/:id/members/:userId/role`、`POST /guilds/:id/channels` 均需 `RequireGuildRole("admin")`
- `Server` struct 新增 `guildMemberRepo repository.GuildMemberRepository` 欄位
- 訊息刪除（`DeleteMessage`）已支援 admin 刪除他人訊息（service 層已處理）

2026-04-30 — API 補全（Phase 1）：
- Cursor-based pagination 已實作（`GetByChannelIDCursor` + `before`/`limit` query params）
- Guild 邀請碼系統：`GuildInvite` model、`guild_invite_repository.go`、`guild_invite_service.go`；routes: `POST /guilds/:id/invites`、`GET /invites/:code`、`POST /guilds/join-by-invite`
- Refresh Token：`RefreshToken` model、`refresh_token_repository.go`；`UserService` 新增 `RefreshAccessToken`/`RevokeRefreshToken`；token rotation 機制（舊 token 撤銷，發新 token）；routes: `POST /auth/refresh`、`POST /auth/logout`
- User 公開資料 API：`GET /api/v1/users/:id`（回傳 `PublicUser`，不含 email）
- Message Attachments：`Message.Attachments []string`（`gorm:"-"`），AfterFind hook 確保序列化為 `[]` 而非 `null`
- `NewUserService` 需傳入 `RefreshTokenRepository`；`NewGuildHandler` 需傳入 `GuildInviteService`
