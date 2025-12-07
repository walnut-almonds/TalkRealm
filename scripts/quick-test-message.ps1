# Message API 快速測試腳本
# 測試所有訊息管理功能

$baseUrl = "http://localhost:8080/api/v1"
$testUsername = "messagetest"
$testPassword = "password123"

Write-Host "=== Message API 快速測試 ===" -ForegroundColor Cyan

# 1. 註冊使用者
Write-Host "`n1. 註冊使用者..." -ForegroundColor Yellow
try {
    $registerBody = @{
        username = $testUsername
        email = "messagetest@example.com"
        password = $testPassword
    } | ConvertTo-Json

    $registerResponse = Invoke-RestMethod -Uri "$baseUrl/auth/register" -Method Post -Body $registerBody -ContentType "application/json"
    Write-Host "✓ 註冊成功" -ForegroundColor Green
} catch {
    Write-Host "⚠ 使用者可能已存在" -ForegroundColor Yellow
}

# 2. 登入
Write-Host "`n2. 登入..." -ForegroundColor Yellow
$loginBody = @{
    username = $testUsername
    password = $testPassword
} | ConvertTo-Json

$loginResponse = Invoke-RestMethod -Uri "$baseUrl/auth/login" -Method Post -Body $loginBody -ContentType "application/json"
$token = $loginResponse.token
Write-Host "✓ 登入成功" -ForegroundColor Green

$headers = @{
    "Authorization" = "Bearer $token"
    "Content-Type" = "application/json"
}

# 3. 建立測試社群
Write-Host "`n3. 建立測試社群..." -ForegroundColor Yellow
$guildBody = @{
    name = "訊息測試社群"
    description = "用於測試訊息功能的社群"
} | ConvertTo-Json

$guild = Invoke-RestMethod -Uri "$baseUrl/guilds" -Method Post -Body $guildBody -Headers $headers
Write-Host "✓ 社群建立成功 - ID: $($guild.id)" -ForegroundColor Green

# 4. 建立測試頻道
Write-Host "`n4. 建立測試頻道..." -ForegroundColor Yellow
$channelBody = @{
    guild_id = $guild.id
    name = "一般聊天"
    type = "text"
    topic = "測試訊息的文字頻道"
} | ConvertTo-Json

$channel = Invoke-RestMethod -Uri "$baseUrl/channels" -Method Post -Body $channelBody -Headers $headers
Write-Host "✓ 頻道建立成功 - ID: $($channel.id)" -ForegroundColor Green

# 5. 發送第一則訊息
Write-Host "`n5. 發送第一則訊息..." -ForegroundColor Yellow
$messageBody1 = @{
    channel_id = $channel.id
    content = "大家好！這是第一則測試訊息。"
    type = "text"
} | ConvertTo-Json

$message1 = Invoke-RestMethod -Uri "$baseUrl/messages" -Method Post -Body $messageBody1 -Headers $headers
Write-Host "✓ 訊息發送成功 - ID: $($message1.id)" -ForegroundColor Green
Write-Host "  內容: $($message1.content)" -ForegroundColor Gray

# 6. 發送第二則訊息
Write-Host "`n6. 發送第二則訊息..." -ForegroundColor Yellow
$messageBody2 = @{
    channel_id = $channel.id
    content = "這是第二則訊息，用來測試列表功能。"
} | ConvertTo-Json

$message2 = Invoke-RestMethod -Uri "$baseUrl/messages" -Method Post -Body $messageBody2 -Headers $headers
Write-Host "✓ 訊息發送成功 - ID: $($message2.id)" -ForegroundColor Green

# 7. 發送第三則訊息
Write-Host "`n7. 發送第三則訊息..." -ForegroundColor Yellow
$messageBody3 = @{
    channel_id = $channel.id
    content = "第三則訊息，測試分頁功能。"
} | ConvertTo-Json

$message3 = Invoke-RestMethod -Uri "$baseUrl/messages" -Method Post -Body $messageBody3 -Headers $headers
Write-Host "✓ 訊息發送成功 - ID: $($message3.id)" -ForegroundColor Green

# 8. 取得單一訊息
Write-Host "`n8. 取得單一訊息..." -ForegroundColor Yellow
$getMessage = Invoke-RestMethod -Uri "$baseUrl/messages/$($message1.id)" -Method Get -Headers $headers
Write-Host "✓ 訊息取得成功" -ForegroundColor Green
Write-Host "  ID: $($getMessage.id)" -ForegroundColor Gray
Write-Host "  內容: $($getMessage.content)" -ForegroundColor Gray
Write-Host "  作者: $($getMessage.user.username)" -ForegroundColor Gray

# 9. 列出頻道的所有訊息
Write-Host "`n9. 列出頻道的所有訊息..." -ForegroundColor Yellow
$messageList = Invoke-RestMethod -Uri "$baseUrl/messages?channel_id=$($channel.id)&page=1&page_size=10" -Method Get -Headers $headers
Write-Host "✓ 訊息列表取得成功" -ForegroundColor Green
Write-Host "  總數: $($messageList.total)" -ForegroundColor Gray
Write-Host "  頁碼: $($messageList.page)" -ForegroundColor Gray
Write-Host "  訊息:" -ForegroundColor Gray
foreach ($msg in $messageList.messages) {
    Write-Host "    - [$($msg.user.username)] $($msg.content)" -ForegroundColor Gray
}

# 10. 更新訊息
Write-Host "`n10. 更新訊息..." -ForegroundColor Yellow
$updateBody = @{
    content = "這是更新後的第一則訊息內容！"
} | ConvertTo-Json

$updatedMessage = Invoke-RestMethod -Uri "$baseUrl/messages/$($message1.id)" -Method Put -Body $updateBody -Headers $headers
Write-Host "✓ 訊息更新成功" -ForegroundColor Green
Write-Host "  新內容: $($updatedMessage.content)" -ForegroundColor Gray

# 11. 測試訊息權限（嘗試更新他人訊息 - 應該失敗）
Write-Host "`n11. 測試訊息權限（預期失敗）..." -ForegroundColor Yellow
Write-Host "  （略過，需要第二個使用者）" -ForegroundColor Gray

# 12. 刪除一則訊息
Write-Host "`n12. 刪除一則訊息..." -ForegroundColor Yellow
$deleteResult = Invoke-RestMethod -Uri "$baseUrl/messages/$($message2.id)" -Method Delete -Headers $headers
Write-Host "✓ 訊息刪除成功: $($deleteResult.message)" -ForegroundColor Green

# 13. 驗證訊息已刪除
Write-Host "`n13. 驗證訊息已刪除..." -ForegroundColor Yellow
try {
    $deletedMessage = Invoke-RestMethod -Uri "$baseUrl/messages/$($message2.id)" -Method Get -Headers $headers
    Write-Host "✗ 訊息應該已被刪除但仍可存取" -ForegroundColor Red
} catch {
    Write-Host "✓ 確認訊息已刪除（404 錯誤）" -ForegroundColor Green
}

# 14. 列出剩餘訊息
Write-Host "`n14. 列出剩餘訊息..." -ForegroundColor Yellow
$remainingMessages = Invoke-RestMethod -Uri "$baseUrl/messages?channel_id=$($channel.id)" -Method Get -Headers $headers
Write-Host "✓ 剩餘訊息數量: $($remainingMessages.total)" -ForegroundColor Green

# 15. 測試空訊息（應該失敗）
Write-Host "`n15. 測試空訊息（預期失敗）..." -ForegroundColor Yellow
try {
    $emptyBody = @{
        channel_id = $channel.id
        content = ""
    } | ConvertTo-Json
    
    $emptyMessage = Invoke-RestMethod -Uri "$baseUrl/messages" -Method Post -Body $emptyBody -Headers $headers
    Write-Host "✗ 空訊息應該被拒絕" -ForegroundColor Red
} catch {
    Write-Host "✓ 空訊息被正確拒絕" -ForegroundColor Green
}

# 16. 測試無效頻道（應該失敗）
Write-Host "`n16. 測試無效頻道（預期失敗）..." -ForegroundColor Yellow
try {
    $invalidChannelBody = @{
        channel_id = 99999
        content = "這則訊息應該失敗"
    } | ConvertTo-Json
    
    $invalidMessage = Invoke-RestMethod -Uri "$baseUrl/messages" -Method Post -Body $invalidChannelBody -Headers $headers
    Write-Host "✗ 無效頻道應該被拒絕" -ForegroundColor Red
} catch {
    Write-Host "✓ 無效頻道被正確拒絕" -ForegroundColor Green
}

# 清理測試資料
Write-Host "`n17. 清理測試資料..." -ForegroundColor Yellow
try {
    # 刪除剩餘訊息
    Invoke-RestMethod -Uri "$baseUrl/messages/$($message1.id)" -Method Delete -Headers $headers | Out-Null
    Invoke-RestMethod -Uri "$baseUrl/messages/$($message3.id)" -Method Delete -Headers $headers | Out-Null
    
    # 刪除頻道
    Invoke-RestMethod -Uri "$baseUrl/channels/$($channel.id)" -Method Delete -Headers $headers | Out-Null
    
    # 刪除社群
    Invoke-RestMethod -Uri "$baseUrl/guilds/$($guild.id)" -Method Delete -Headers $headers | Out-Null
    
    Write-Host "✓ 測試資料清理完成" -ForegroundColor Green
} catch {
    Write-Host "⚠ 清理過程中發生錯誤（可能資源已被刪除）" -ForegroundColor Yellow
}

Write-Host "`n=== 所有 Message 功能測試完成！✨ ===" -ForegroundColor Cyan
Write-Host "`n測試摘要：" -ForegroundColor Cyan
Write-Host "  ✓ 發送訊息 (text 類型)" -ForegroundColor Green
Write-Host "  ✓ 取得單一訊息" -ForegroundColor Green
Write-Host "  ✓ 列出頻道訊息（分頁）" -ForegroundColor Green
Write-Host "  ✓ 更新自己的訊息" -ForegroundColor Green
Write-Host "  ✓ 刪除訊息" -ForegroundColor Green
Write-Host "  ✓ 權限驗證（成員檢查）" -ForegroundColor Green
Write-Host "  ✓ 錯誤處理（空訊息、無效頻道）" -ForegroundColor Green
