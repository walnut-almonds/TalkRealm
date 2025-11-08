# Docker 管理腳本 - 查看日誌
# Usage: .\scripts\docker-logs.ps1

param(
    [string]$Service = "",
    [switch]$Follow
)

Write-Host "📋 Viewing TalkRealm service logs..." -ForegroundColor Cyan

if ($Follow) {
    if ($Service) {
        Write-Host "Following logs for: $Service" -ForegroundColor Yellow
        docker-compose logs -f $Service
    } else {
        Write-Host "Following all service logs..." -ForegroundColor Yellow
        docker-compose logs -f
    }
} else {
    if ($Service) {
        Write-Host "Showing logs for: $Service" -ForegroundColor Yellow
        docker-compose logs --tail=100 $Service
    } else {
        Write-Host "Showing all service logs (last 100 lines)..." -ForegroundColor Yellow
        docker-compose logs --tail=100
    }
}
