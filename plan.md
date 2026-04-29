# TalkRealm — 架構重設計計畫

> 基於 2026-04-30 架構圖，從現有 monolith 拆分為微服務架構。

---

## 一、架構總覽

```
Client (Browser / App)
    │
    ├── wss://ws.talkrealm.com  ──►  WebSocket Gateway
    │                                     │
    │                                     ├── Chat Server A ─┐
    │                                     └── Chat Server B  ├──► Redis (user→server mapping)
    │                                                         └──► MQ (Kafka / NATS)
    │                                                                  │
    │                                                         ┌────────┴─────────┐
    │                                                         │                  │
    │                                                  topic:serverB       topic:record
    │                                                         │                  │
    │                                              (Chat Server B)   Message Persistence Service
    │                                                                       │
    │                                                                      DB (PostgreSQL)
    │
    └── https://api.talkrealm.com ──►  RestfulAPI Gateway
                                             │
                                             ├── Auth Service ──► DB
                                             ├── File Access Service ──► DB + Minio
                                             └── (其他 RESTful 資源)

Voice:
    User F / G ──► WebRTC ──► LiveKit Voice Server A
```

---

## 二、服務清單與職責

| 服務 | 技術 | 職責 |
|------|------|------|
| **WebSocket Gateway** | Go + gorilla/websocket | 維護長連線、路由 WS 訊息到 Chat Server |
| **Chat Server** | Go | 處理即時訊息路由、跨 server 轉發（透過 MQ）|
| **Auth Service** | Go + JWT | 註冊、登入、Token 驗證 |
| **RestfulAPI Gateway** | Go + Gin | 統一 REST 入口、轉發到後端服務 |
| **Message Persistence Service** | Go | 消費 MQ `topic:record`，寫入 DB |
| **File Access Service** | Go | Pre-signed URL 生成、檔案 metadata 管理 |
| **File Server** | Minio (S3-compatible) | 實際存放檔案 |
| **LiveKit Voice Server** | LiveKit | WebRTC 語音/視訊 |
| **Redis** | Redis | 記錄 userID → serverID 映射、Session 快取 |
| **Message Queue** | NATS JetStream / Kafka | 跨 Chat Server 訊息路由 + 持久化任務佇列 |

---

## 三、核心流程

### 3.1 跨 Server 訊息傳遞（User A → User C，不同 Chat Server）

```
1. User A 透過 WebSocket 送訊息 → Chat Server A
2. Chat Server A 查 Redis: GET user:{userC_id}:server → "serverB"
3. Chat Server A publish MQ topic:serverB  { from, to, content, channelID }
4. Chat Server B subscribe topic:serverB → 收到後透過 WebSocket push 給 User C
5. Chat Server A 同時 publish MQ topic:record → Message Persistence Service 寫 DB
```

### 3.2 User 連線 / 斷線（Redis 記錄）

```
連線: SET user:{userID}:server {serverID}  EX 86400
斷線: DEL user:{userID}:server
```

### 3.3 Pre-signed 檔案上傳流程

```
1. User D  POST /api/v1/files/presign  { filename, size, mime_type }
2. File Access Service → Minio SDK 生成 Pre-signed Upload URL（有效期 15 min）
3. 回傳 { upload_url, file_id, expires_at }
4. User D  PUT {upload_url}  直接上傳到 Minio，不經過 API Server
5. 上傳完成後，User D 透過 Chat Server WebSocket 發送 file 類型訊息 { file_id }
6. Chat Server 廣播給頻道成員
7. 接收方從訊息中取得 file_id，再呼叫 GET /api/v1/files/{id}/url 取得下載 URL
```

### 3.4 語音頻道（WebRTC via LiveKit）

```
1. User 呼叫 POST /api/v1/channels/{id}/voice/token
2. Auth Service 驗證權限後，LiveKit SDK 生成 Room Token
3. 回傳 { token, livekit_url }
4. 前端 LiveKit SDK 使用 token 連接 LiveKit Server
```

---

## 四、各服務 API 介面定義

### 4.1 Auth Service（`api.talkrealm.com/api/v1/auth`）

#### POST /api/v1/auth/register
```json
// Request
{
  "username": "alice",
  "email": "alice@example.com",
  "password": "P@ssw0rd"
}

// Response 201
{
  "id": 1,
  "username": "alice",
  "email": "alice@example.com",
  "avatar_url": null,
  "created_at": "2026-04-30T00:00:00Z"
}

// Error 409 - user already exists
{ "error": "user_already_exists" }
```

#### POST /api/v1/auth/login
```json
// Request
{ "email": "alice@example.com", "password": "P@ssw0rd" }

// Response 200
{
  "access_token": "<JWT>",
  "token_type": "Bearer",
  "expires_in": 86400,
  "user": { "id": 1, "username": "alice", "email": "alice@example.com" }
}
```

#### POST /api/v1/auth/refresh
```json
// Request Header: Authorization: Bearer <refresh_token>
// Response 200
{ "access_token": "<new_JWT>", "expires_in": 86400 }
```

#### POST /api/v1/auth/logout
```
// Request Header: Authorization: Bearer <token>
// Response 204 No Content
```

---

### 4.2 User Service（`/api/v1/users`）

#### GET /api/v1/users/me
```json
// Response 200
{
  "id": 1,
  "username": "alice",
  "email": "alice@example.com",
  "avatar_url": "https://cdn.talkrealm.com/avatars/1.png",
  "status": "online",
  "created_at": "2026-04-30T00:00:00Z"
}
```

#### PATCH /api/v1/users/me
```json
// Request
{ "username": "alice2", "avatar_url": "https://..." }
// Response 200 — 回傳更新後 User 物件
```

#### GET /api/v1/users/{id}
```json
// Response 200 — 公開資料（不含 email）
{ "id": 2, "username": "bob", "avatar_url": null, "status": "offline" }
```

---

### 4.3 Guild（Server）Service（`/api/v1/guilds`）

#### POST /api/v1/guilds
```json
// Request
{ "name": "My Server", "description": "Hello", "icon_url": null }
// Response 201
{
  "id": 10,
  "name": "My Server",
  "owner_id": 1,
  "icon_url": null,
  "created_at": "..."
}
```

#### GET /api/v1/guilds/me
```json
// Response 200 — 當前使用者加入的所有 guild
{ "guilds": [ { "id": 10, "name": "My Server", "icon_url": null, "role": "owner" } ] }
```

#### GET /api/v1/guilds/{id}
```json
// Response 200
{
  "id": 10,
  "name": "My Server",
  "owner_id": 1,
  "member_count": 5,
  "channels": [ { "id": 1, "name": "general", "type": "text" } ]
}
```

#### PATCH /api/v1/guilds/{id}
```json
// Request（需 owner 或 admin 權限）
{ "name": "New Name", "description": "...", "icon_url": "..." }
// Response 200
```

#### DELETE /api/v1/guilds/{id}
```
// 需 owner 權限
// Response 204
```

#### POST /api/v1/guilds/{id}/join
```json
// Request（可選 invite code）
{ "invite_code": "abc123" }
// Response 200
{ "guild_id": 10, "user_id": 2, "role": "member", "joined_at": "..." }
```

#### POST /api/v1/guilds/{id}/leave
```
// Response 204
```

#### GET /api/v1/guilds/{id}/members
```json
// Query: ?page=1&page_size=50
// Response 200
{
  "members": [
    { "user_id": 1, "username": "alice", "role": "owner", "joined_at": "..." }
  ],
  "total": 1, "page": 1
}
```

#### DELETE /api/v1/guilds/{id}/members/{userId}
```
// 需 admin 以上權限（kick member）
// Response 204
```

#### PUT /api/v1/guilds/{id}/members/{userId}/role
```json
// Request
{ "role": "admin" }   // role: owner | admin | member
// Response 200
```

#### POST /api/v1/guilds/{id}/invites
```json
// 生成邀請碼
// Response 201
{ "code": "abc123", "guild_id": 10, "expires_at": "2026-05-07T00:00:00Z", "max_uses": 100 }
```

---

### 4.4 Channel Service（`/api/v1/channels`, `/api/v1/guilds/{id}/channels`）

#### GET /api/v1/guilds/{id}/channels
```json
// Response 200
{
  "channels": [
    { "id": 1, "name": "general", "type": "text", "position": 0 },
    { "id": 2, "name": "voice-1", "type": "voice", "position": 1 }
  ]
}
```

#### POST /api/v1/guilds/{id}/channels
```json
// Request（需 admin 以上）
{ "name": "announcements", "type": "text", "position": 2 }
// Response 201
{ "id": 3, "name": "announcements", "type": "text", "guild_id": 10, "position": 2 }
```

#### GET /api/v1/channels/{id}
```json
// Response 200
{ "id": 1, "name": "general", "type": "text", "guild_id": 10, "position": 0 }
```

#### PATCH /api/v1/channels/{id}
```json
// Request（需 admin 以上）
{ "name": "new-name" }
// Response 200
```

#### DELETE /api/v1/channels/{id}
```
// 需 admin 以上
// Response 204
```

#### PUT /api/v1/channels/{id}/position
```json
// Request
{ "position": 3 }
// Response 200
```

#### GET /api/v1/channels/{id}/voice/token  *(新增)*
```json
// 取得 LiveKit Room Token（語音頻道）
// Response 200
{
  "token": "<livekit_jwt>",
  "livekit_url": "wss://livekit.talkrealm.com",
  "room": "channel-2"
}
```

---

### 4.5 Message Service（`/api/v1/channels/{id}/messages`, `/api/v1/messages`）

> **注意**：訊息的「發送」走 WebSocket，REST API 僅用於歷史訊息查詢、編輯、刪除。

#### GET /api/v1/channels/{id}/messages
```json
// Query: ?before={message_id}&limit=50  (cursor-based pagination)
// Response 200
{
  "messages": [
    {
      "id": 100,
      "channel_id": 1,
      "author": { "id": 1, "username": "alice" },
      "content": "Hello!",
      "type": "text",
      "attachments": [],
      "created_at": "...",
      "updated_at": null,
      "is_edited": false
    }
  ],
  "has_more": true
}
```

#### GET /api/v1/messages/{id}
```json
// Response 200 — 單一訊息詳情
```

#### PATCH /api/v1/messages/{id}
```json
// Request（僅訊息作者）
{ "content": "edited content" }
// Response 200 — 回傳更新後訊息
```

#### DELETE /api/v1/messages/{id}
```
// 訊息作者 或 channel admin 可刪
// Response 204
```

---

### 4.6 File Access Service（`/api/v1/files`）  *(全新)*

#### POST /api/v1/files/presign
```json
// Request
{
  "filename": "photo.png",
  "size": 204800,
  "mime_type": "image/png",
  "channel_id": 1   // 關聯頻道（可選）
}

// Response 201
{
  "file_id": "uuid-xxx",
  "upload_url": "https://minio.talkrealm.com/bucket/uuid-xxx?X-Amz-Signature=...",
  "expires_at": "2026-04-30T00:15:00Z"
}
```

#### GET /api/v1/files/{id}
```json
// Response 200
{
  "id": "uuid-xxx",
  "filename": "photo.png",
  "mime_type": "image/png",
  "size": 204800,
  "url": "https://cdn.talkrealm.com/uuid-xxx",   // public URL（若公開）
  "uploaded_by": 1,
  "created_at": "..."
}
```

#### GET /api/v1/files/{id}/url  *(取得下載 Pre-signed URL，私有檔案用)*
```json
// Response 200
{
  "url": "https://minio.talkrealm.com/bucket/uuid-xxx?X-Amz-Signature=...",
  "expires_at": "2026-04-30T01:00:00Z"
}
```

#### DELETE /api/v1/files/{id}
```
// 僅上傳者或 admin
// Response 204
```

---

### 4.7 WebSocket Gateway & Chat Server Protocol

#### 連線
```
wss://ws.talkrealm.com/ws?token=<JWT>
```

#### Client → Server 訊息格式
```json
{
  "op": "send_message",       // op code
  "d": {                       // data
    "channel_id": 1,
    "content": "Hello",
    "type": "text",             // text | file
    "file_id": null             // 若 type=file 則填 file_id
  },
  "nonce": "client-uuid"       // 用於 ACK 對應
}
```

**Op Codes（Client → Server）**

| op | 說明 |
|----|------|
| `identify` | 連線後認證（帶 token）|
| `heartbeat` | 定期 ping |
| `send_message` | 發送訊息到頻道 |
| `subscribe_channel` | 訂閱頻道事件 |
| `unsubscribe_channel` | 取消訂閱 |
| `typing_start` | 正在輸入 |
| `voice_state_update` | 語音狀態更新 |

**Op Codes（Server → Client）**

| op | 說明 |
|----|------|
| `hello` | 連線成功，帶 heartbeat_interval |
| `heartbeat_ack` | pong |
| `ready` | identify 成功，帶初始化資料 |
| `message_create` | 新訊息 |
| `message_update` | 訊息被編輯 |
| `message_delete` | 訊息被刪除 |
| `typing_start` | 有人正在輸入 |
| `presence_update` | 使用者狀態變更 |
| `channel_create` | 頻道建立 |
| `channel_update` | 頻道更新 |
| `channel_delete` | 頻道刪除 |
| `error` | 錯誤通知 |

#### Server → Client 訊息範例
```json
{
  "op": "message_create",
  "d": {
    "id": 100,
    "channel_id": 1,
    "author": { "id": 1, "username": "alice" },
    "content": "Hello!",
    "type": "text",
    "attachments": [],
    "created_at": "2026-04-30T00:00:00Z"
  },
  "t": 1746000000000
}
```

---

## 五、Redis Schema

```
# User 所在的 Chat Server
user:{userID}:server     STRING   → "chat-server-a"   TTL: 86400s

# Chat Server 對應的 MQ topic
server:{serverID}:topic  STRING   → "topic:chat-server-a"  (可省略，topic == serverID)

# Online 使用者集合（Guild 維度，用於 presence）
guild:{guildID}:online   SET      → { userID, ... }

# Rate limiting
ratelimit:{userID}:msg   COUNTER  TTL: 1s
```

---

## 六、Message Queue Topic 設計

使用 **NATS JetStream**（或 Kafka）：

| Topic | 生產者 | 消費者 | 說明 |
|-------|--------|--------|------|
| `chat.server.{serverID}` | 任何 Chat Server | 對應 Chat Server | 跨 server 訊息路由 |
| `chat.record` | 所有 Chat Server | Message Persistence Service | 持久化訊息到 DB |
| `chat.event` | 所有 Chat Server | 推送服務 / 通知服務 | 事件廣播 |

---

## 七、資料庫 Schema（主要表格）

### users
```sql
id BIGSERIAL PK, username TEXT UNIQUE, email TEXT UNIQUE,
password_hash TEXT, avatar_url TEXT, status TEXT DEFAULT 'offline',
created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ
```

### guilds
```sql
id BIGSERIAL PK, name TEXT, description TEXT, icon_url TEXT,
owner_id BIGINT FK users, created_at TIMESTAMPTZ
```

### guild_members
```sql
guild_id BIGINT FK, user_id BIGINT FK, role TEXT DEFAULT 'member',
joined_at TIMESTAMPTZ
PRIMARY KEY (guild_id, user_id)
```

### guild_invites *(新增)*
```sql
id BIGSERIAL PK, guild_id BIGINT FK, code TEXT UNIQUE,
creator_id BIGINT FK, max_uses INT, uses INT DEFAULT 0,
expires_at TIMESTAMPTZ, created_at TIMESTAMPTZ
```

### channels
```sql
id BIGSERIAL PK, guild_id BIGINT FK, name TEXT, type TEXT,
position INT DEFAULT 0, created_at TIMESTAMPTZ
```

### messages
```sql
id BIGSERIAL PK, channel_id BIGINT FK, author_id BIGINT FK,
content TEXT, type TEXT DEFAULT 'text', is_edited BOOL DEFAULT FALSE,
created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ, deleted_at TIMESTAMPTZ
```

### message_attachments *(新增)*
```sql
id BIGSERIAL PK, message_id BIGINT FK, file_id TEXT,
filename TEXT, mime_type TEXT, size BIGINT
```

### files *(新增)*
```sql
id TEXT PK (UUID), filename TEXT, mime_type TEXT, size BIGINT,
bucket TEXT, object_key TEXT, uploaded_by BIGINT FK,
channel_id BIGINT FK (nullable), is_public BOOL DEFAULT FALSE,
created_at TIMESTAMPTZ
```

---

## 八、現有程式碼 Gap 分析

| 項目 | 現狀 | 目標 | 優先級 |
|------|------|------|--------|
| WebSocket 跨 server 路由 | ❌ 單一 in-process manager | Redis + MQ 路由 | 🔴 高 |
| MQ 整合（NATS/Kafka）| ❌ 無 | NATS JetStream | 🔴 高 |
| Message Persistence 分離 | ❌ 同步寫入 DB | 異步 MQ consumer | 🟡 中 |
| 邀請碼系統 | ❌ 無 | guild_invites 表 + API | 🟡 中 |
| Pre-signed 檔案上傳 | ❌ 無 | Minio + File Access Service | 🟡 中 |
| Presence 系統 | ❌ 無 | Redis SET + WS event | 🟡 中 |
| Typing Indicator | ❌ 無 | WS op:typing_start | 🟡 中 |
| LiveKit 語音整合 | ❌ 無 | LiveKit SDK + token API | 🟠 低 |
| Refresh Token | ❌ 無 | DB 存 refresh token | 🟠 低 |
| Rate Limiting | ❌ 無 | Redis counter | 🟠 低 |
| RBAC 權限系統 | ❌ 只有 role 欄位 | 完整 role permission check | 🟡 中 |
| Cursor-based pagination | ❌ offset pagination | before/after message_id | 🟡 中 |

---

## 九、遷移路徑

### Phase 1：強化現有 Monolith（短期）
- 修正 WS Manager：加入 `channelID` → `[]*Client` 索引，避免廣播給無關 client
- 加入 Redis client（連線追蹤、session）
- 實作邀請碼 API
- 實作 Presence（上線/下線通知）
- Typing indicator WS event
- Cursor-based message pagination
- 完整 RBAC middleware

### Phase 2：接入 MQ（中期）
- 引入 NATS JetStream
- Chat Server publish `chat.record` → Message Persistence Consumer
- Chat Server publish `chat.server.{id}` → 跨 server 路由
- 將 WebSocket 連線管理與 HTTP handler 拆為兩個 binary（ws-gateway / api-server）

### Phase 3：File Service & Voice（中期）
- 接入 Minio，實作 File Access Service
- Pre-signed URL API
- LiveKit 整合，實作語音頻道 token API

### Phase 4：微服務拆分（長期）
- Auth Service 獨立部署
- Message Persistence Service 獨立部署
- API Gateway（可用 Nginx / Traefik / 自研 gateway）
- 各服務 Kubernetes Deployment 分離

---

## 十、技術選型補充

| 需求 | 選擇 | 備選 |
|------|------|------|
| MQ | NATS JetStream | Kafka（更重） |
| 物件儲存 | Minio（自建）| AWS S3 |
| 語音 | LiveKit | mediasoup |
| API Gateway | Nginx / custom Go | Kong |
| Service Mesh | 暫不引入 | Istio（Phase 4+）|
| Metrics | Prometheus + Grafana | — |
| Tracing | OpenTelemetry | — |
