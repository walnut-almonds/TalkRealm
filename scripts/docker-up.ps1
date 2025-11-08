# Docker 管理腳本 - 啟動資料庫服務
# Usage: .\scripts\docker-up.ps1

Write-Host "🚀 Starting TalkRealm database services..." -ForegroundColor Green

# 檢查 Docker 是否安裝
try {
    $dockerVersion = docker --version
    Write-Host "✓ Docker found: $dockerVersion" -ForegroundColor Cyan
} catch {
    Write-Host "❌ Docker is not installed or not in PATH" -ForegroundColor Red
    Write-Host "Please install Docker Desktop from: https://www.docker.com/products/docker-desktop" -ForegroundColor Yellow
    exit 1
}

# 檢查 Docker 是否正在運行
$dockerInfo = docker info 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Docker daemon is not running" -ForegroundColor Red
    Write-Host "Please start Docker Desktop first" -ForegroundColor Yellow
    exit 1
}

# 啟動服務
Write-Host "`n📦 Starting PostgreSQL and Redis containers..." -ForegroundColor Yellow
docker-compose up -d

if ($LASTEXITCODE -eq 0) {
    Write-Host "`n✅ Services started successfully!" -ForegroundColor Green
    Write-Host "`nService information:" -ForegroundColor Cyan
    Write-Host "  PostgreSQL: localhost:5432" -ForegroundColor White
    Write-Host "    - Database: talkrealm" -ForegroundColor Gray
    Write-Host "    - Username: talkrealm" -ForegroundColor Gray
    Write-Host "    - Password: talkrealm_password" -ForegroundColor Gray
    Write-Host "`n  Redis: localhost:6379" -ForegroundColor White
    Write-Host "    - Password: talkrealm_redis_password" -ForegroundColor Gray
    
    Write-Host "`n🔍 Checking container status..." -ForegroundColor Yellow
    Start-Sleep -Seconds 3
    docker-compose ps
    
    Write-Host "`n💡 Useful commands:" -ForegroundColor Cyan
    Write-Host "  View logs:    docker-compose logs -f" -ForegroundColor Gray
    Write-Host "  Stop services: .\scripts\docker-down.ps1" -ForegroundColor Gray
    Write-Host "  Restart:      docker-compose restart" -ForegroundColor Gray
} else {
    Write-Host "`n❌ Failed to start services" -ForegroundColor Red
    exit 1
}
