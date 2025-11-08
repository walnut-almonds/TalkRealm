# Docker 快速開始指南

本專案使用 Docker Compose 來管理 PostgreSQL 和 Redis 服務，讓你無需手動安裝資料庫即可開始開發。

## 前置需求

### Windows
1. 下載並安裝 [Docker Desktop for Windows](https://www.docker.com/products/docker-desktop)
2. 啟動 Docker Desktop（確保系統托盤中有 Docker 圖示）

### macOS
```bash
brew install --cask docker
# 或從官網下載: https://www.docker.com/products/docker-desktop
```

### Linux
```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install docker.io docker-compose

# 啟動 Docker
sudo systemctl start docker
sudo systemctl enable docker
```

## 快速開始

### 1. 啟動資料庫服務

**Windows (PowerShell):**
```powershell
.\scripts\docker-up.ps1
```

**Linux/macOS:**
```bash
chmod +x scripts/*.sh
./scripts/docker-up.sh
```

**或直接使用 Docker Compose:**
```bash
docker-compose up -d
```

啟動後你會看到：
```
✅ Services started successfully!

Service information:
  PostgreSQL: localhost:5432
    - Database: talkrealm
    - Username: talkrealm
    - Password: talkrealm_password

  Redis: localhost:6379
    - Password: talkrealm_redis_password
```

### 2. 準備配置檔

```bash
# 複製 Docker 開發配置
cp configs/config.docker.yaml configs/config.yaml
```

或手動編輯 `configs/config.yaml`：
```yaml
database:
  host: localhost
  port: 5432
  user: talkrealm
  password: talkrealm_password
  dbname: talkrealm
  sslmode: disable

redis:
  host: localhost
  port: 6379
  password: talkrealm_redis_password
```

### 3. 執行資料庫遷移

```bash
# 建立資料表
go run scripts/migrate.go
```

### 4. 啟動應用程式

```bash
go run cmd/server/main.go
```

## Docker 管理命令

### 查看服務狀態
```bash
docker-compose ps
```

### 查看日誌
```powershell
# Windows
.\scripts\docker-logs.ps1          # 查看所有日誌
.\scripts\docker-logs.ps1 -Follow  # 即時跟蹤日誌
.\scripts\docker-logs.ps1 -Service postgres  # 僅查看 PostgreSQL 日誌
```

```bash
# Linux/macOS
docker-compose logs           # 查看所有日誌
docker-compose logs -f        # 即時跟蹤日誌
docker-compose logs postgres  # 僅查看 PostgreSQL 日誌
```

### 停止服務
```powershell
# Windows
.\scripts\docker-down.ps1
```

```bash
# Linux/macOS
./scripts/docker-down.sh
```

### 重啟服務
```bash
docker-compose restart
docker-compose restart postgres  # 僅重啟 PostgreSQL
```

### 完全重置（刪除所有資料）
```powershell
# Windows - 會提示確認
.\scripts\docker-reset.ps1
```

```bash
# Linux/macOS
docker-compose down -v
```

## 連接到資料庫

### 使用 psql (命令列)
```bash
# 需要先安裝 PostgreSQL 客戶端工具
docker exec -it talkrealm-postgres psql -U talkrealm -d talkrealm
```

### 使用 GUI 工具

#### pgAdmin
- Host: `localhost`
- Port: `5432`
- Database: `talkrealm`
- Username: `talkrealm`
- Password: `talkrealm_password`

#### DBeaver / DataGrip
配置相同的連線資訊即可。

### 使用 Redis CLI
```bash
docker exec -it talkrealm-redis redis-cli -a talkrealm_redis_password
```

## 服務細節

### PostgreSQL
- **Image:** `postgres:15-alpine`
- **Port:** 5432
- **Volume:** 資料持久化於 Docker volume `postgres_data`
- **Health Check:** 每 10 秒檢查一次

### Redis
- **Image:** `redis:7-alpine`
- **Port:** 6379
- **Volume:** 資料持久化於 Docker volume `redis_data`
- **Persistence:** AOF 模式（Append Only File）

## 常見問題

### 1. 無法連接到 PostgreSQL
```bash
# 檢查容器狀態
docker-compose ps

# 查看日誌
docker-compose logs postgres

# 確認端口未被佔用
netstat -ano | findstr :5432  # Windows
lsof -i :5432                 # macOS/Linux
```

### 2. 端口已被佔用
編輯 `docker-compose.yml`，修改端口映射：
```yaml
ports:
  - "5433:5432"  # 改用 5433
```

然後更新 `configs/config.yaml` 中的端口。

### 3. 清除所有資料重新開始
```bash
# 停止並刪除所有容器和資料
docker-compose down -v

# 重新啟動
docker-compose up -d

# 重新執行遷移
go run scripts/migrate.go
```

### 4. Docker Desktop 未啟動
**Windows:** 在開始選單搜尋並啟動 "Docker Desktop"

**macOS:** 在 Applications 中啟動 Docker

**Linux:** 
```bash
sudo systemctl start docker
```

### 5. 權限問題 (Linux)
```bash
# 將使用者加入 docker 群組
sudo usermod -aG docker $USER

# 登出後重新登入使權限生效
```

## 資料備份與還原

### 備份資料庫
```bash
docker exec talkrealm-postgres pg_dump -U talkrealm talkrealm > backup.sql
```

### 還原資料庫
```bash
cat backup.sql | docker exec -i talkrealm-postgres psql -U talkrealm -d talkrealm
```

## 生產環境注意事項

⚠️ **請勿在生產環境使用這些預設密碼！**

生產環境建議：
1. 使用強密碼（至少 32 字元）
2. 使用 Docker secrets 或環境變數管理敏感資訊
3. 配置防火牆規則限制資料庫存取
4. 啟用 SSL/TLS 連線
5. 定期備份資料
6. 使用受管理的資料庫服務（如 AWS RDS、Google Cloud SQL）

## 更多資訊

- [Docker 官方文件](https://docs.docker.com/)
- [Docker Compose 參考](https://docs.docker.com/compose/)
- [PostgreSQL Docker Hub](https://hub.docker.com/_/postgres)
- [Redis Docker Hub](https://hub.docker.com/_/redis)
