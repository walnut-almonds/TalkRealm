# Docker 管理腳本 - 重置資料庫（刪除所有資料）
# Usage: .\scripts\docker-reset.ps1

Write-Host "⚠️  WARNING: This will DELETE ALL DATA in the database!" -ForegroundColor Red
$confirmation = Read-Host "Are you sure you want to continue? (yes/no)"

if ($confirmation -ne "yes") {
    Write-Host "Operation cancelled." -ForegroundColor Yellow
    exit 0
}

Write-Host "`n🗑️  Stopping and removing containers with volumes..." -ForegroundColor Yellow
docker-compose down -v

if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ All data removed successfully!" -ForegroundColor Green
    Write-Host "`n💡 To start fresh, run:" -ForegroundColor Cyan
    Write-Host "  .\scripts\docker-up.ps1" -ForegroundColor Gray
} else {
    Write-Host "❌ Failed to reset database" -ForegroundColor Red
    exit 1
}
