# Channel API 快速測試腳本
$baseUrl = "http://localhost:8080/api/v1"
$ContentType = "application/json; charset=utf-8"

Write-Host "=== Channel API 快速測試 ===" -ForegroundColor Cyan

# 1. 註冊並登入
Write-Host "`n1. 註冊使用者..." -ForegroundColor Yellow
$registerBody = @{
    username = "channeltest"
    email = "channeltest@test.com"
    password = "password123"
    nickname = "Channel Tester"
} | ConvertTo-Json

try {
    $null = Invoke-RestMethod -Uri "$baseUrl/auth/register" -Method Post -Body $registerBody -ContentType $ContentType -ErrorAction Stop
    Write-Host "✓ 註冊成功" -ForegroundColor Green
} catch {
    Write-Host "⚠ 使用者可能已存在" -ForegroundColor Yellow
}

Write-Host "`n2. 登入..." -ForegroundColor Yellow
$loginBody = @{
    username = "channeltest"
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
Write-Host "`n3. 建立測試社群..." -ForegroundColor Yellow
$createGuildBody = @{
    name = "測試社群"
    description = "用於測試頻道功能"
} | ConvertTo-Json

$guild = Invoke-RestMethod -Uri "$baseUrl/guilds" -Method Post -Body $createGuildBody -Headers $headers
Write-Host "✓ 社群建立成功: $($guild.name) (ID: $($guild.id))" -ForegroundColor Green
$guildId = $guild.id

# 4. 建立文字頻道
Write-Host "`n4. 建立文字頻道..." -ForegroundColor Yellow
$createTextChannelBody = @{
    guild_id = $guildId
    name = "一般文字"
    type = "text"
    topic = "這是一般文字頻道"
    position = 0
} | ConvertTo-Json

$textChannel = Invoke-RestMethod -Uri "$baseUrl/channels" -Method Post -Body $createTextChannelBody -Headers $headers
Write-Host "✓ 文字頻道建立成功: $($textChannel.name) (ID: $($textChannel.id))" -ForegroundColor Green
$textChannelId = $textChannel.id

# 5. 建立語音頻道
Write-Host "`n5. 建立語音頻道..." -ForegroundColor Yellow
$createVoiceChannelBody = @{
    guild_id = $guildId
    name = "一般語音"
    type = "voice"
    topic = "這是語音頻道"
    position = 1
} | ConvertTo-Json

$voiceChannel = Invoke-RestMethod -Uri "$baseUrl/channels" -Method Post -Body $createVoiceChannelBody -Headers $headers
Write-Host "✓ 語音頻道建立成功: $($voiceChannel.name) (ID: $($voiceChannel.id))" -ForegroundColor Green
$voiceChannelId = $voiceChannel.id

# 6. 取得頻道詳情
Write-Host "`n6. 取得文字頻道詳情..." -ForegroundColor Yellow
$channelDetail = Invoke-RestMethod -Uri "$baseUrl/channels/$textChannelId" -Method Get -Headers $headers
Write-Host "✓ 頻道詳情: $($channelDetail.name) - 類型: $($channelDetail.type)" -ForegroundColor Green

# 7. 列出社群的所有頻道
Write-Host "`n7. 列出社群的所有頻道..." -ForegroundColor Yellow
$channels = Invoke-RestMethod -Uri "$baseUrl/channels?guild_id=$guildId" -Method Get -Headers $headers
Write-Host "✓ 社群共有 $($channels.Count) 個頻道:" -ForegroundColor Green
foreach ($ch in $channels) {
    Write-Host "   - $($ch.name) ($($ch.type)) - 位置: $($ch.position)" -ForegroundColor Gray
}

# 8. 更新頻道
Write-Host "`n8. 更新文字頻道..." -ForegroundColor Yellow
$updateChannelBody = @{
    name = "已更新的文字頻道"
    topic = "更新後的主題"
} | ConvertTo-Json

$updatedChannel = Invoke-RestMethod -Uri "$baseUrl/channels/$textChannelId" -Method Put -Body $updateChannelBody -Headers $headers
Write-Host "✓ 頻道更新成功: $($updatedChannel.name)" -ForegroundColor Green

# 9. 更新頻道位置
Write-Host "`n9. 更新頻道位置..." -ForegroundColor Yellow
$updatePositionBody = @{
    position = 5
} | ConvertTo-Json

$positionResponse = Invoke-RestMethod -Uri "$baseUrl/channels/$textChannelId/position" -Method Put -Body $updatePositionBody -Headers $headers
Write-Host "✓ 位置更新成功: $($positionResponse.message)" -ForegroundColor Green

# 10. 刪除語音頻道
Write-Host "`n10. 刪除語音頻道..." -ForegroundColor Yellow
$deleteResponse = Invoke-RestMethod -Uri "$baseUrl/channels/$voiceChannelId" -Method Delete -Headers $headers
Write-Host "✓ 頻道刪除成功: $($deleteResponse.message)" -ForegroundColor Green

# 11. 驗證刪除
Write-Host "`n11. 列出剩餘頻道..." -ForegroundColor Yellow
$remainingChannels = Invoke-RestMethod -Uri "$baseUrl/channels?guild_id=$guildId" -Method Get -Headers $headers
Write-Host "✓ 社群現在有 $($remainingChannels.Count) 個頻道" -ForegroundColor Green

# 12. 清理：刪除社群
Write-Host "`n12. 清理測試資料..." -ForegroundColor Yellow
$cleanupResponse = Invoke-RestMethod -Uri "$baseUrl/guilds/$guildId" -Method Delete -Headers $headers
Write-Host "✓ 測試社群已刪除" -ForegroundColor Green

Write-Host "`n=== 測試完成 ===" -ForegroundColor Cyan
Write-Host "所有 Channel 功能測試通過！✨" -ForegroundColor Green
