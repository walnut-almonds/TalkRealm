# Docker 管理腳本 - 停止資料庫服務
# Usage: .\scripts\docker-down.ps1

Write-Host "🛑 Stopping TalkRealm database services..." -ForegroundColor Yellow

# 停止服務
docker-compose down

if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Services stopped successfully!" -ForegroundColor Green
    Write-Host "`n💡 To remove all data volumes, run:" -ForegroundColor Cyan
    Write-Host "  docker-compose down -v" -ForegroundColor Gray
} else {
    Write-Host "❌ Failed to stop services" -ForegroundColor Red
    exit 1
}
