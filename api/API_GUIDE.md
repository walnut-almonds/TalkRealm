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
1. `scripts/test-api.ps1` - 完整 API 測試
2. `scripts/quick-test.ps1` - 快速 API 測試

---

## ✅ 測試結果

所有測試通過！✨

- ✅ 健康檢查
- ✅ 使用者註冊
- ✅ 使用者登入
- ✅ JWT Token 生成
- ✅ JWT 認證中間件
- ✅ 獲取使用者資訊
- ✅ 更新使用者資訊
- ✅ 安全性驗證（拒絕無效 Token）

---

## 🔜 下一步建議

1. **社群功能 (Guild)**
   - 建立社群 Service 和 Handler
   - 社群成員管理
   - 權限控制

2. **頻道功能 (Channel)**
   - 文字頻道
   - 語音頻道
   - 頻道訊息

3. **WebSocket 即時通訊**
   - WebSocket 連接管理
   - 訊息廣播
   - 線上狀態同步

4. **測試與文件**
   - 單元測試
   - Swagger API 文件
   - 錯誤處理完善

---

## 📞 聯繫方式

如有問題，請查看專案 README 或提交 issue。
