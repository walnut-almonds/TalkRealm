# Guild API 快速測試腳本
$baseUrl = "http://localhost:8080/api/v1"
$ContentType = "application/json; charset=utf-8"

Write-Host "=== Guild API 快速測試 ===" -ForegroundColor Cyan

# 1. 註冊並登入
Write-Host "`n1. 註冊使用者..." -ForegroundColor Yellow
$registerBody = @{
    username = "testowner"
    email = "testowner@test.com"
    password = "password123"
    nickname = "Test Owner"
} | ConvertTo-Json

try {
    $null = Invoke-RestMethod -Uri "$baseUrl/auth/register" -Method Post -Body $registerBody -ContentType $ContentType -ErrorAction Stop
    Write-Host "✓ 註冊成功" -ForegroundColor Green
} catch {
    Write-Host "⚠ 使用者可能已存在" -ForegroundColor Yellow
}

Write-Host "`n2. 登入..." -ForegroundColor Yellow
$loginBody = @{
    username = "testowner"
    password = "password123"
} | ConvertTo-Json

$loginResponse = Invoke-RestMethod -Uri "$baseUrl/auth/login" -Method Post -Body $loginBody -ContentType $ContentType
$token = $loginResponse.token
Write-Host "✓ 登入成功" -ForegroundColor Green

$headers = @{
    "Authorization" = "Bearer $token"
    "Content-Type" = $ContentType
}

# 3. 建立社群
Write-Host "`n3. 建立社群..." -ForegroundColor Yellow
$createGuildBody = @{
    name = "測試社群"
    description = "這是測試"
    icon = ""
} | ConvertTo-Json

$guild = Invoke-RestMethod -Uri "$baseUrl/guilds" -Method Post -Body $createGuildBody -Headers $headers
Write-Host "✓ 社群建立成功: $($guild.name) (ID: $($guild.id))" -ForegroundColor Green
$guildId = $guild.id

# 4. 取得社群
Write-Host "`n4. 取得社群詳情..." -ForegroundColor Yellow
$guildDetail = Invoke-RestMethod -Uri "$baseUrl/guilds/$guildId" -Method Get -Headers $headers
Write-Host "✓ 取得成功: $($guildDetail.name)" -ForegroundColor Green

# 5. 列出社群
Write-Host "`n5. 列出使用者社群..." -ForegroundColor Yellow
$guilds = Invoke-RestMethod -Uri "$baseUrl/guilds" -Method Get -Headers $headers
Write-Host "✓ 共有 $($guilds.Count) 個社群" -ForegroundColor Green

# 6. 更新社群
Write-Host "`n6. 更新社群..." -ForegroundColor Yellow
$updateBody = @{
    name = "已更新的社群"
    description = "更新後的描述"
} | ConvertTo-Json

$updatedGuild = Invoke-RestMethod -Uri "$baseUrl/guilds/$guildId" -Method Put -Body $updateBody -Headers $headers
Write-Host "✓ 更新成功: $($updatedGuild.name)" -ForegroundColor Green

# 7. 列出成員
Write-Host "`n7. 列出社群成員..." -ForegroundColor Yellow
$members = Invoke-RestMethod -Uri "$baseUrl/guilds/$guildId/members" -Method Get -Headers $headers
Write-Host "✓ 共有 $($members.Count) 位成員" -ForegroundColor Green

# 8. 刪除社群
Write-Host "`n8. 刪除社群..." -ForegroundColor Yellow
$deleteResponse = Invoke-RestMethod -Uri "$baseUrl/guilds/$guildId" -Method Delete -Headers $headers
Write-Host "✓ 刪除成功: $($deleteResponse.message)" -ForegroundColor Green

Write-Host "`n=== 測試完成 ===" -ForegroundColor Cyan
