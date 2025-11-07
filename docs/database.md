# 資料庫設定指南

## PostgreSQL 安裝

### Windows
1. 下載 PostgreSQL: https://www.postgresql.org/download/windows/
2. 執行安裝程式並記住設定的密碼

### macOS
```bash
brew install postgresql
brew services start postgresql
```

### Linux (Ubuntu/Debian)
```bash
sudo apt update
sudo apt install postgresql postgresql-contrib
sudo systemctl start postgresql
```

## 建立資料庫

連線到 PostgreSQL：
```bash
psql -U postgres
```

建立資料庫和使用者：
```sql
-- 建立資料庫
CREATE DATABASE talkrealm;

-- 建立使用者（可選）
CREATE USER talkrealm_user WITH PASSWORD 'your_secure_password';

-- 授予權限
GRANT ALL PRIVILEGES ON DATABASE talkrealm TO talkrealm_user;

-- 離開
\q
```

## 設定應用程式

1. 複製設定檔範例：
```bash
cp configs/config.example.yaml configs/config.yaml
```

2. 編輯 `configs/config.yaml`，更新資料庫設定：
```yaml
database:
  host: localhost
  port: 5432
  user: postgres  # 或你建立的使用者
  password: your_password
  dbname: talkrealm
  sslmode: disable
  max_idle_conns: 10
  max_open_conns: 100
  conn_max_lifetime: 60
  log_mode: false  # 開發時可設為 true 查看 SQL
```

## 執行資料庫遷移

### 方法 1：使用遷移腳本（推薦）
```bash
# 執行遷移
go run scripts/migrate.go

# 重置資料庫（會刪除所有資料！）
go run scripts/migrate.go -drop
```

### 方法 2：自動遷移（啟動服務時）
服務啟動時會自動執行遷移（在 `main.go` 中已設定）

## 資料表結構

遷移後會建立以下資料表：

- **users** - 使用者資料
  - id, username, email, password, nickname, avatar, status, created_at, updated_at

- **guilds** - 社群/伺服器資料
  - id, name, description, icon, owner_id, created_at, updated_at

- **channels** - 頻道資料
  - id, guild_id, name, type (text/voice), topic, position, created_at, updated_at

- **messages** - 訊息資料
  - id, channel_id, user_id, content, type (text/image/file), created_at, updated_at

- **guild_members** - 社群成員資料
  - id, guild_id, user_id, nickname, role (owner/admin/member), joined_at, created_at, updated_at

## 驗證資料庫連線

啟動服務後，可以在日誌中看到：
```
Database connected successfully
Database migrations completed successfully
```

## 常見問題

### 連線失敗
- 檢查 PostgreSQL 服務是否正在執行
- 確認資料庫名稱、使用者名稱和密碼是否正確
- 檢查防火牆設定

### 權限錯誤
```sql
-- 重新授予權限
GRANT ALL PRIVILEGES ON DATABASE talkrealm TO your_user;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO your_user;
```

### 重置資料庫
```bash
# 使用遷移腳本
go run scripts/migrate.go -drop

# 或手動刪除
psql -U postgres -d talkrealm -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
```
