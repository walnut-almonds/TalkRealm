# TalkRealm API 使用文件

## 🎉 已完成的功能

### ✅ 使用者認證系統
- 使用者註冊
- 使用者登入 (JWT Token)
- JWT 認證中間件
- 密碼加密 (bcrypt)

### ✅ 使用者管理
- 獲取當前使用者資訊
- 更新使用者資訊
- 使用者狀態管理

### ✅ 社群管理 (Guild)
- 建立社群
- 獲取社群詳情
- 列出使用者社群
- 更新社群資訊
- 刪除社群

### ✅ 社群成員管理
- 加入社群
- 離開社群
- 列出社群成員
- 踢出成員
- 更新成員角色

### ✅ 頻道管理 (Channel)
- 建立文字頻道和語音頻道
- 獲取頻道詳情
- 列出社群的所有頻道
- 更新頻道資訊
- 刪除頻道
- 更新頻道位置

### ✅ 訊息管理 (Message)
- 發送訊息 (文字、圖片、檔案)
- 取得訊息詳情
- 列出頻道訊息 (分頁)
- 更新訊息內容
- 刪除訊息
- 權限控制 (擁有者、管理員)

---

## 📚 API 端點說明

### 基礎 URL
```
http://localhost:8080
```

---

## 🔓 公開 API（無需認證）

### 1. 健康檢查
檢查服務是否正常運行。

**請求**
```http
GET /health
```

**回應**
```json
{
  "status": "ok",
  "service": "talkrealm"
}
```

---

### 2. 使用者註冊
建立新的使用者帳號。

**請求**
```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "username": "alice",
  "email": "alice@example.com",
  "password": "password123",
  "nickname": "Alice Wang"
}
```

**欄位說明**
- `username` (必填): 使用者名稱，3-32 字元
- `email` (必填): 電子郵件，需符合 email 格式
- `password` (必填): 密碼，6-128 字元
- `nickname` (選填): 暱稱，最多 64 字元（未提供則使用 username）

**成功回應 (201 Created)**
```json
{
  "message": "user registered successfully",
  "user": {
    "id": 1,
    "username": "alice",
    "email": "alice@example.com",
    "nickname": "Alice Wang",
    "avatar": "",
    "status": "offline",
    "created_at": "2025-11-16T20:00:00Z",
    "updated_at": "2025-11-16T20:00:00Z"
  }
}
```

**錯誤回應 (409 Conflict)**
```json
{
  "error": "user already exists"
}
```

---

### 3. 使用者登入
使用 email 和密碼登入，獲取 JWT Token。

**請求**
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "alice@example.com",
  "password": "password123"
}
```

**成功回應 (200 OK)**
```json
{
  "message": "login successful",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "username": "alice",
    "email": "alice@example.com",
    "nickname": "Alice Wang",
    "avatar": "",
    "status": "online",
    "created_at": "2025-11-16T20:00:00Z",
    "updated_at": "2025-11-16T20:00:00Z"
  }
}
```

**錯誤回應 (401 Unauthorized)**
```json
{
  "error": "invalid email or password"
}
```

---

## 🔒 需要認證的 API

所有以下 API 都需要在 HTTP Header 中包含 JWT Token：

```http
Authorization: Bearer <your_jwt_token>
```

---

### 4. 獲取當前使用者資訊
取得已登入使用者的詳細資訊。

**請求**
```http
GET /api/v1/users/me
Authorization: Bearer <token>
```

**成功回應 (200 OK)**
```json
{
  "user": {
    "id": 1,
    "username": "alice",
    "email": "alice@example.com",
    "nickname": "Alice Wang",
    "avatar": "",
    "status": "online",
    "created_at": "2025-11-16T20:00:00Z",
    "updated_at": "2025-11-16T20:00:00Z"
  }
}
```

---

### 5. 更新使用者資訊
更新當前使用者的暱稱、頭像或狀態。

**請求**
```http
PATCH /api/v1/users/me
Authorization: Bearer <token>
Content-Type: application/json

{
  "nickname": "Alice Updated",
  "avatar": "https://example.com/avatar.jpg",
  "status": "online"
}
```

**欄位說明**
- `nickname` (選填): 新的暱稱，最多 64 字元
- `avatar` (選填): 頭像 URL，最多 256 字元
- `status` (選填): 狀態，可選值: `online`, `offline`, `busy`, `away`

**成功回應 (200 OK)**
```json
{
  "message": "user updated successfully",
  "user": {
    "id": 1,
    "username": "alice",
    "email": "alice@example.com",
    "nickname": "Alice Updated",
    "avatar": "https://example.com/avatar.jpg",
    "status": "online",
    "created_at": "2025-11-16T20:00:00Z",
    "updated_at": "2025-11-16T20:08:00Z"
  }
}
```

---

## 🧪 測試方式

### 使用 PowerShell 測試
我們提供了測試腳本：

```powershell
# 啟動伺服器（在新視窗）
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd D:\GoProject\TalkRealm; go run cmd\server\main.go"

# 等待幾秒後執行測試
Start-Sleep -Seconds 5
.\scripts\quick-test.ps1
```

### 使用 curl 測試

```bash
# 健康檢查
curl http://localhost:8080/health

# 註冊
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"bob","email":"bob@example.com","password":"password123"}'

# 登入
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"bob@example.com","password":"password123"}'

# 獲取使用者資訊（需要 token）
curl http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

---

## 🗄️ 資料庫設定

### 執行資料庫遷移
```powershell
go run scripts\migrate.go
```

### 重置資料庫（刪除所有資料）
```powershell
go run scripts\migrate.go -drop
```

---

## 🔐 安全性特性

1. **密碼加密**: 使用 bcrypt 加密，成本因子為預設值
2. **JWT Token**: 
   - 過期時間: 24 小時（可在 config.yaml 調整）
   - 包含使用者 ID、username、email
3. **認證中間件**: 自動驗證 Bearer Token
4. **CORS 支援**: 允許跨域請求

---

## 📝 配置文件

編輯 `configs/config.yaml`:

```yaml
jwt:
  secret: "your-secret-key-change-in-production"
  expiration_hours: 24

server:
  port: 8080
  mode: debug  # 或 release

database:
  host: localhost
  port: 5432
  user: talkrealm
  password: talkrealm_password
  dbname: talkrealm
```

---

## 🚀 啟動服務

### 開發模式
```powershell
go run cmd\server\main.go
```

### 編譯後執行
```powershell
go build -o bin\talkrealm.exe cmd\server\main.go
.\bin\talkrealm.exe
```

---

## 📦 已建立的檔案

### 新增的核心檔案
1. `pkg/auth/jwt.go` - JWT 工具函數
2. `internal/service/user_service.go` - 使用者業務邏輯
3. `internal/handler/user_handler.go` - 使用者 API 處理器
4. `internal/middleware/middleware.go` - 認證中間件（已更新）

### 測試腳本
1. `scripts/test-api.ps1` - 使用者 API 完整測試
2. `scripts/quick-test.ps1` - 使用者 API 快速測試
3. `scripts/test-guild.ps1` - Guild API 完整測試
4. `scripts/quick-test-guild.ps1` - Guild API 快速測試
5. `scripts/quick-test-channel.ps1` - Channel API 快速測試

---

## 🏰 社群管理 API（需要認證）

### 1. 建立社群
建立一個新的社群，建立者自動成為擁有者。

**請求**
```http
POST /api/v1/guilds
Authorization: Bearer {token}
Content-Type: application/json

{
  "name": "我的社群",
  "description": "這是一個很棒的社群",
  "icon": "https://example.com/icon.png"
}
```

**回應** (201 Created)
```json
{
  "id": 1,
  "name": "我的社群",
  "description": "這是一個很棒的社群",
  "icon": "https://example.com/icon.png",
  "owner_id": 1,
  "created_at": "2024-12-03T15:30:00Z",
  "updated_at": "2024-12-03T15:30:00Z"
}
```

---

### 2. 取得社群詳情
獲取指定社群的詳細資訊。

**請求**
```http
GET /api/v1/guilds/{id}
Authorization: Bearer {token}
```

**回應** (200 OK)
```json
{
  "id": 1,
  "name": "我的社群",
  "description": "這是一個很棒的社群",
  "icon": "https://example.com/icon.png",
  "owner_id": 1,
  "created_at": "2024-12-03T15:30:00Z",
  "updated_at": "2024-12-03T15:30:00Z"
}
```

---

### 3. 列出使用者的社群
列出當前使用者所屬的所有社群。

**請求**
```http
GET /api/v1/guilds
Authorization: Bearer {token}
```

**回應** (200 OK)
```json
[
  {
    "id": 1,
    "name": "我的社群",
    "description": "這是一個很棒的社群",
    "icon": "https://example.com/icon.png",
    "owner_id": 1,
    "created_at": "2024-12-03T15:30:00Z",
    "updated_at": "2024-12-03T15:30:00Z"
  }
]
```

---

### 4. 更新社群
更新社群資訊（僅擁有者可操作）。

**請求**
```http
PUT /api/v1/guilds/{id}
Authorization: Bearer {token}
Content-Type: application/json

{
  "name": "更新後的社群名稱",
  "description": "更新後的描述",
  "icon": "https://example.com/new-icon.png"
}
```

**回應** (200 OK)
```json
{
  "id": 1,
  "name": "更新後的社群名稱",
  "description": "更新後的描述",
  "icon": "https://example.com/new-icon.png",
  "owner_id": 1,
  "created_at": "2024-12-03T15:30:00Z",
  "updated_at": "2024-12-03T15:35:00Z"
}
```

**錯誤回應** (403 Forbidden)
```json
{
  "error": "only owner can update guild"
}
```

---

### 5. 刪除社群
刪除社群（僅擁有者可操作）。

**請求**
```http
DELETE /api/v1/guilds/{id}
Authorization: Bearer {token}
```

**回應** (200 OK)
```json
{
  "message": "guild deleted successfully"
}
```

**錯誤回應** (403 Forbidden)
```json
{
  "error": "only owner can delete guild"
}
```

---

## 👥 社群成員管理 API（需要認證）

### 1. 加入社群
使用者加入指定社群。

**請求**
```http
POST /api/v1/guilds/{id}/join
Authorization: Bearer {token}
```

**回應** (200 OK)
```json
{
  "message": "joined guild successfully"
}
```

**錯誤回應** (400 Bad Request)
```json
{
  "error": "already in guild"
}
```

---

### 2. 離開社群
使用者離開社群（擁有者需先轉移所有權）。

**請求**
```http
POST /api/v1/guilds/{id}/leave
Authorization: Bearer {token}
```

**回應** (200 OK)
```json
{
  "message": "left guild successfully"
}
```

**錯誤回應** (403 Forbidden)
```json
{
  "error": "owner cannot leave, transfer ownership first"
}
```

---

### 3. 列出社群成員
列出社群的所有成員。

**請求**
```http
GET /api/v1/guilds/{id}/members
Authorization: Bearer {token}
```

**回應** (200 OK)
```json
[
  {
    "id": 1,
    "guild_id": 1,
    "user_id": 1,
    "nickname": "",
    "role": "owner",
    "joined_at": "2024-12-03T15:30:00Z",
    "created_at": "2024-12-03T15:30:00Z",
    "updated_at": "2024-12-03T15:30:00Z"
  },
  {
    "id": 2,
    "guild_id": 1,
    "user_id": 2,
    "nickname": "",
    "role": "member",
    "joined_at": "2024-12-03T15:32:00Z",
    "created_at": "2024-12-03T15:32:00Z",
    "updated_at": "2024-12-03T15:32:00Z"
  }
]
```

---

### 4. 踢出成員
擁有者踢出社群成員。

**請求**
```http
DELETE /api/v1/guilds/{id}/members/{userId}
Authorization: Bearer {token}
```

**回應** (200 OK)
```json
{
  "message": "member kicked successfully"
}
```

**錯誤回應** (403 Forbidden)
```json
{
  "error": "only owner can kick members"
}
```

---

### 5. 更新成員角色
擁有者更新成員角色。

**請求**
```http
PUT /api/v1/guilds/{id}/members/{userId}/role
Authorization: Bearer {token}
Content-Type: application/json

{
  "role": "moderator"
}
```

**可用角色**
- `owner` - 擁有者
- `admin` - 管理員
- `moderator` - 版主
- `member` - 普通成員

**回應** (200 OK)
```json
{
  "message": "member role updated successfully"
}
```

**錯誤回應** (403 Forbidden)
```json
{
  "error": "only owner can update member roles"
}
```

---

## 📺 頻道管理 API（需要認證）

### 1. 建立頻道
在社群中建立新的文字或語音頻道（僅擁有者或管理員）。

**請求**
```http
POST /api/v1/channels
Authorization: Bearer {token}
Content-Type: application/json

{
  "guild_id": 1,
  "name": "一般文字",
  "type": "text",
  "topic": "歡迎來到一般文字頻道",
  "position": 0
}
```

**頻道類型**
- `text` - 文字頻道
- `voice` - 語音頻道

**回應** (201 Created)
```json
{
  "id": 1,
  "guild_id": 1,
  "name": "一般文字",
  "type": "text",
  "topic": "歡迎來到一般文字頻道",
  "position": 0,
  "created_at": "2024-12-07T19:30:00Z",
  "updated_at": "2024-12-07T19:30:00Z"
}
```

---

### 2. 取得頻道詳情
獲取指定頻道的詳細資訊（需為社群成員）。

**請求**
```http
GET /api/v1/channels/{id}
Authorization: Bearer {token}
```

**回應** (200 OK)
```json
{
  "id": 1,
  "guild_id": 1,
  "name": "一般文字",
  "type": "text",
  "topic": "歡迎來到一般文字頻道",
  "position": 0,
  "created_at": "2024-12-07T19:30:00Z",
  "updated_at": "2024-12-07T19:30:00Z"
}
```

---

### 3. 列出社群的頻道
列出指定社群的所有頻道（需為社群成員）。

**請求**
```http
GET /api/v1/channels?guild_id={guild_id}
Authorization: Bearer {token}
```

**回應** (200 OK)
```json
[
  {
    "id": 1,
    "guild_id": 1,
    "name": "一般文字",
    "type": "text",
    "topic": "歡迎來到一般文字頻道",
    "position": 0,
    "created_at": "2024-12-07T19:30:00Z",
    "updated_at": "2024-12-07T19:30:00Z"
  },
  {
    "id": 2,
    "guild_id": 1,
    "name": "語音聊天",
    "type": "voice",
    "topic": "語音頻道",
    "position": 1,
    "created_at": "2024-12-07T19:31:00Z",
    "updated_at": "2024-12-07T19:31:00Z"
  }
]
```

---

### 4. 更新頻道
更新頻道資訊（僅擁有者或管理員）。

**請求**
```http
PUT /api/v1/channels/{id}
Authorization: Bearer {token}
Content-Type: application/json

{
  "name": "更新後的頻道名稱",
  "topic": "更新後的主題",
  "position": 2
}
```

**回應** (200 OK)
```json
{
  "id": 1,
  "guild_id": 1,
  "name": "更新後的頻道名稱",
  "type": "text",
  "topic": "更新後的主題",
  "position": 2,
  "created_at": "2024-12-07T19:30:00Z",
  "updated_at": "2024-12-07T19:35:00Z"
}
```

---

### 5. 刪除頻道
刪除頻道（僅擁有者或管理員）。

**請求**
```http
DELETE /api/v1/channels/{id}
Authorization: Bearer {token}
```

**回應** (200 OK)
```json
{
  "message": "channel deleted successfully"
}
```

---

### 6. 更新頻道位置
更新頻道在列表中的位置（僅擁有者或管理員）。

**請求**
```http
PUT /api/v1/channels/{id}/position
Authorization: Bearer {token}
Content-Type: application/json

{
  "position": 5
}
```

**回應** (200 OK)
```json
{
  "message": "channel position updated successfully"
}
```

---

## 💬 訊息管理 API（需要認證）

### 1. 發送訊息

**端點**: `POST /api/v1/messages`

**描述**: 在指定頻道中發送新訊息

**請求**
```http
POST /api/v1/messages
Authorization: Bearer {token}
Content-Type: application/json

{
  "channel_id": 1,
  "content": "大家好！這是一則測試訊息。",
  "type": "text"
}
```

**參數說明**:
- `channel_id` (required): 頻道 ID
- `content` (required): 訊息內容（不可為空）
- `type` (optional): 訊息類型，可選值：`text`、`image`、`file`（預設：`text`）

**回應** (201 Created)
```json
{
  "id": 1,
  "channel_id": 1,
  "user_id": 1,
  "content": "大家好！這是一則測試訊息。",
  "type": "text",
  "created_at": "2024-12-07T10:30:00Z",
  "updated_at": "2024-12-07T10:30:00Z",
  "user": {
    "id": 1,
    "username": "testuser",
    "nickname": "測試使用者",
    "avatar": ""
  },
  "channel": {
    "id": 1,
    "guild_id": 1,
    "name": "一般聊天",
    "type": "text"
  }
}
```

### 2. 取得訊息

**端點**: `GET /api/v1/messages/{id}`

**描述**: 取得指定 ID 的訊息詳情

**請求**
```http
GET /api/v1/messages/1
Authorization: Bearer {token}
```

**回應** (200 OK)
```json
{
  "id": 1,
  "channel_id": 1,
  "user_id": 1,
  "content": "大家好！這是一則測試訊息。",
  "type": "text",
  "created_at": "2024-12-07T10:30:00Z",
  "updated_at": "2024-12-07T10:30:00Z",
  "user": {
    "id": 1,
    "username": "testuser",
    "nickname": "測試使用者"
  },
  "channel": {
    "id": 1,
    "name": "一般聊天"
  }
}
```

### 3. 列出頻道訊息

**端點**: `GET /api/v1/messages?channel_id={id}&page={page}&page_size={size}`

**描述**: 列出指定頻道的所有訊息（支援分頁）

**請求**
```http
GET /api/v1/messages?channel_id=1&page=1&page_size=50
Authorization: Bearer {token}
```

**參數說明**:
- `channel_id` (required): 頻道 ID
- `page` (optional): 頁碼，預設 1
- `page_size` (optional): 每頁數量，預設 50，最大 100

**回應** (200 OK)
```json
{
  "messages": [
    {
      "id": 3,
      "channel_id": 1,
      "user_id": 1,
      "content": "最新的訊息",
      "type": "text",
      "created_at": "2024-12-07T10:32:00Z",
      "user": {
        "id": 1,
        "username": "testuser"
      }
    },
    {
      "id": 2,
      "channel_id": 1,
      "user_id": 2,
      "content": "第二則訊息",
      "type": "text",
      "created_at": "2024-12-07T10:31:00Z",
      "user": {
        "id": 2,
        "username": "user2"
      }
    },
    {
      "id": 1,
      "channel_id": 1,
      "user_id": 1,
      "content": "大家好！",
      "type": "text",
      "created_at": "2024-12-07T10:30:00Z",
      "user": {
        "id": 1,
        "username": "testuser"
      }
    }
  ],
  "total": 3,
  "page": 1,
  "page_size": 50,
  "total_pages": 1
}
```

**注意**: 訊息按建立時間降序排列（最新的在前）

### 4. 更新訊息

**端點**: `PUT /api/v1/messages/{id}`

**描述**: 更新自己發送的訊息內容

**權限**: 只有訊息擁有者可以更新

**請求**
```http
PUT /api/v1/messages/1
Authorization: Bearer {token}
Content-Type: application/json

{
  "content": "這是更新後的訊息內容"
}
```

**回應** (200 OK)
```json
{
  "id": 1,
  "channel_id": 1,
  "user_id": 1,
  "content": "這是更新後的訊息內容",
  "type": "text",
  "created_at": "2024-12-07T10:30:00Z",
  "updated_at": "2024-12-07T10:35:00Z",
  "user": {
    "id": 1,
    "username": "testuser"
  }
}
```

### 5. 刪除訊息

**端點**: `DELETE /api/v1/messages/{id}`

**描述**: 刪除訊息

**權限**: 
- 訊息擁有者可以刪除自己的訊息
- 社群擁有者和管理員可以刪除任何訊息

**請求**
```http
DELETE /api/v1/messages/1
Authorization: Bearer {token}
```

**回應** (200 OK)
```json
{
  "message": "message deleted successfully"
}
```

### 訊息類型說明

- **text**: 純文字訊息
- **image**: 圖片訊息（未來實作檔案上傳功能）
- **file**: 檔案訊息（未來實作檔案上傳功能）

### 權限說明

1. **發送訊息**: 只有社群成員可以發送訊息
2. **查看訊息**: 只有社群成員可以查看訊息
3. **更新訊息**: 只有訊息擁有者可以更新
4. **刪除訊息**: 訊息擁有者、社群擁有者、社群管理員可以刪除

### 錯誤回應

**400 Bad Request** - 請求參數錯誤
```json
{
  "error": "message content cannot be empty"
}
```

**403 Forbidden** - 權限不足
```json
{
  "error": "you are not a member of this channel's guild"
}
```

**404 Not Found** - 訊息不存在
```json
{
  "error": "message not found"
}
```

---

### 測試腳本
1. `scripts/test-api.ps1` - 完整 API 測試
2. `scripts/quick-test.ps1` - 快速 API 測試
3. `scripts/quick-test-channel.ps1` - 頻道功能測試
4. `scripts/quick-test-message.ps1` - 訊息功能測試

---

## ✅ 測試結果

### 使用者認證系統測試
所有測試通過！✨

- ✅ 健康檢查
- ✅ 使用者註冊
- ✅ 使用者登入
- ✅ JWT Token 生成
- ✅ JWT 認證中間件
- ✅ 獲取使用者資訊
- ✅ 更新使用者資訊
- ✅ 安全性驗證（拒絕無效 Token）

### 社群管理系統測試
所有測試通過！✨

- ✅ 建立社群
- ✅ 取得社群詳情
- ✅ 列出使用者社群
- ✅ 更新社群資訊
- ✅ 刪除社群
- ✅ 加入社群
- ✅ 離開社群
- ✅ 列出社群成員
- ✅ 踢出成員
- ✅ 更新成員角色
- ✅ 權限控制（非擁有者無法更新/刪除）
- ✅ 擁有者無法離開社群驗證

### 頻道管理系統測試
所有測試通過！✨

- ✅ 建立文字頻道
- ✅ 建立語音頻道
- ✅ 取得頻道詳情
- ✅ 列出社群頻道
- ✅ 更新頻道資訊
- ✅ 更新頻道位置
- ✅ 刪除頻道
- ✅ 權限控制（擁有者/管理員）
- ✅ 成員權限驗證（非成員無法查看頻道）

### 訊息管理系統測試
所有測試通過！✨

- ✅ 發送文字訊息
- ✅ 取得單一訊息
- ✅ 列出頻道訊息（分頁）
- ✅ 更新自己的訊息
- ✅ 刪除訊息
- ✅ 管理員刪除他人訊息
- ✅ 權限驗證（成員檢查）
- ✅ 錯誤處理（空訊息、無效頻道）

---
- ✅ 建立語音頻道
- ✅ 取得頻道詳情
- ✅ 列出社群的所有頻道
- ✅ 更新頻道資訊
- ✅ 更新頻道位置
- ✅ 刪除頻道
- ✅ 權限控制（只有擁有者或管理員可管理頻道）
- ✅ 成員權限驗證（非成員無法查看頻道）

---

## 🔜 下一步建議

1. **訊息附件功能**
   - 圖片上傳
   - 檔案上傳
   - 附件管理

2. **WebSocket 即時通訊**
   - WebSocket 連接管理
   - 即時訊息推送
   - 線上狀態同步
   - 打字狀態顯示

3. **進階功能**
   - 訊息反應 (Emoji Reactions)
   - 訊息回覆/引用
   - 訊息搜尋
   - 訊息固定

4. **測試與文件**
   - 單元測試
   - 整合測試
   - Swagger API 文件
   - 錯誤處理完善

---

## 📞 聯繫方式

如有問題，請查看專案 README 或提交 issue。
