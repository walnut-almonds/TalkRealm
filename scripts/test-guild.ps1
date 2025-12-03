# Guild API 測試腳本
# 測試所有 Guild 管理功能

$baseUrl = "http://localhost:8080/api/v1"
$ContentType = "application/json; charset=utf-8"

Write-Host "=== Guild API 測試開始 ===" -ForegroundColor Cyan
Write-Host ""

# 1. 註冊測試使用者
Write-Host "1. 註冊測試使用者..." -ForegroundColor Yellow
$registerBody = @{
    username = "guildowner"
    email = "owner@test.com"
    password = "password123"
    nickname = "Guild Owner"
} | ConvertTo-Json -Compress

try {
    $registerResponse = Invoke-RestMethod -Uri "$baseUrl/auth/register" -Method Post -Body $registerBody -ContentType $ContentType
    Write-Host "✓ 使用者註冊成功: $($registerResponse.username)" -ForegroundColor Green
} catch {
    Write-Host "✗ 註冊失敗（可能已存在）: $($_.Exception.Message)" -ForegroundColor Yellow
}

# 2. 登入取得 Token
Write-Host "`n2. 使用者登入..." -ForegroundColor Yellow
$loginBody = @{
    username = "guildowner"
    password = "password123"
} | ConvertTo-Json -Compress

$loginResponse = Invoke-RestMethod -Uri "$baseUrl/auth/login" -Method Post -Body $loginBody -ContentType $ContentType
$token = $loginResponse.token
Write-Host "✓ 登入成功，Token: $($token.Substring(0, 20))..." -ForegroundColor Green

$headers = @{
    "Authorization" = "Bearer $token"
    "Content-Type" = $ContentType
}

# 3. 建立社群
Write-Host "`n3. 建立社群..." -ForegroundColor Yellow
$createGuildBody = @{
    name = "測試社群"
    description = "這是一個測試社群"
    icon = "https://example.com/icon.png"
} | ConvertTo-Json -Compress

$guild = Invoke-RestMethod -Uri "$baseUrl/guilds" -Method Post -Body $createGuildBody -Headers $headers
$guildId = $guild.id
Write-Host "✓ 社群建立成功: $($guild.name) (ID: $guildId)" -ForegroundColor Green

# 4. 取得社群詳情
Write-Host "`n4. 取得社群詳情..." -ForegroundColor Yellow
$guildDetail = Invoke-RestMethod -Uri "$baseUrl/guilds/$guildId" -Method Get -Headers $headers
Write-Host "✓ 社群詳情: $($guildDetail.name) - $($guildDetail.description)" -ForegroundColor Green

# 5. 列出使用者的社群
Write-Host "`n5. 列出使用者的社群..." -ForegroundColor Yellow
$userGuilds = Invoke-RestMethod -Uri "$baseUrl/guilds" -Method Get -Headers $headers
Write-Host "✓ 使用者擁有 $($userGuilds.Count) 個社群" -ForegroundColor Green

# 6. 更新社群
Write-Host "`n6. 更新社群..." -ForegroundColor Yellow
$updateGuildBody = @{
    name = "測試社群（已更新）"
    description = "這是更新後的描述"
} | ConvertTo-Json -Compress

$updatedGuild = Invoke-RestMethod -Uri "$baseUrl/guilds/$guildId" -Method Put -Body $updateGuildBody -Headers $headers
Write-Host "✓ 社群更新成功: $($updatedGuild.name)" -ForegroundColor Green

# 7. 註冊第二個使用者
Write-Host "`n7. 註冊第二個使用者..." -ForegroundColor Yellow
$register2Body = @{
    username = "member1"
    email = "member1@test.com"
    password = "password123"
    nickname = "Member One"
} | ConvertTo-Json -Compress

try {
    $register2Response = Invoke-RestMethod -Uri "$baseUrl/auth/register" -Method Post -Body $register2Body -ContentType $ContentType
    Write-Host "✓ 第二個使用者註冊成功: $($register2Response.username)" -ForegroundColor Green
} catch {
    Write-Host "✗ 註冊失敗（可能已存在）: $($_.Exception.Message)" -ForegroundColor Yellow
}

# 8. 第二個使用者登入
Write-Host "`n8. 第二個使用者登入..." -ForegroundColor Yellow
$login2Body = @{
    username = "member1"
    password = "password123"
} | ConvertTo-Json -Compress

$login2Response = Invoke-RestMethod -Uri "$baseUrl/auth/login" -Method Post -Body $login2Body -ContentType $ContentType
$token2 = $login2Response.token
Write-Host "✓ 第二個使用者登入成功" -ForegroundColor Green

$headers2 = @{
    "Authorization" = "Bearer $token2"
    "Content-Type" = $ContentType
}

# 9. 第二個使用者加入社群
Write-Host "`n9. 第二個使用者加入社群..." -ForegroundColor Yellow
$joinResponse = Invoke-RestMethod -Uri "$baseUrl/guilds/$guildId/join" -Method Post -Headers $headers2
Write-Host "✓ 加入社群成功: $($joinResponse.message)" -ForegroundColor Green

# 10. 列出社群成員
Write-Host "`n10. 列出社群成員..." -ForegroundColor Yellow
$members = Invoke-RestMethod -Uri "$baseUrl/guilds/$guildId/members" -Method Get -Headers $headers
Write-Host "✓ 社群有 $($members.Count) 位成員" -ForegroundColor Green
foreach ($member in $members) {
    Write-Host "   - User ID: $($member.user_id), Role: $($member.role)" -ForegroundColor Gray
}

# 11. 更新成員角色
Write-Host "`n11. 更新成員角色..." -ForegroundColor Yellow
$member1Id = $members | Where-Object { $_.role -eq "member" } | Select-Object -First 1 -ExpandProperty user_id
$updateRoleBody = @{
    role = "moderator"
} | ConvertTo-Json -Compress

$updateRoleResponse = Invoke-RestMethod -Uri "$baseUrl/guilds/$guildId/members/$member1Id/role" -Method Put -Body $updateRoleBody -Headers $headers
Write-Host "✓ 成員角色更新成功: $($updateRoleResponse.message)" -ForegroundColor Green

# 12. 測試非擁有者更新社群（應該失敗）
Write-Host "`n12. 測試非擁有者更新社群（應該失敗）..." -ForegroundColor Yellow
$badUpdateBody = @{
    name = "試圖非法更新"
} | ConvertTo-Json -Compress

try {
    Invoke-RestMethod -Uri "$baseUrl/guilds/$guildId" -Method Put -Body $badUpdateBody -Headers $headers2
    Write-Host "✗ 非擁有者竟然可以更新社群！" -ForegroundColor Red
} catch {
    Write-Host "✓ 正確阻止非擁有者更新: $($_.Exception.Response.StatusCode)" -ForegroundColor Green
}

# 13. 測試第二個使用者離開社群
Write-Host "`n13. 第二個使用者離開社群..." -ForegroundColor Yellow
$leaveResponse = Invoke-RestMethod -Uri "$baseUrl/guilds/$guildId/leave" -Method Post -Headers $headers2
Write-Host "✓ 離開社群成功: $($leaveResponse.message)" -ForegroundColor Green

# 14. 第二個使用者重新加入
Write-Host "`n14. 第二個使用者重新加入..." -ForegroundColor Yellow
$rejoinResponse = Invoke-RestMethod -Uri "$baseUrl/guilds/$guildId/join" -Method Post -Headers $headers2
Write-Host "✓ 重新加入成功: $($rejoinResponse.message)" -ForegroundColor Green

# 15. 擁有者踢出成員
Write-Host "`n15. 擁有者踢出成員..." -ForegroundColor Yellow
$kickResponse = Invoke-RestMethod -Uri "$baseUrl/guilds/$guildId/members/$member1Id" -Method Delete -Headers $headers
Write-Host "✓ 踢出成員成功: $($kickResponse.message)" -ForegroundColor Green

# 16. 驗證成員已被踢出
Write-Host "`n16. 驗證成員已被踢出..." -ForegroundColor Yellow
$membersAfterKick = Invoke-RestMethod -Uri "$baseUrl/guilds/$guildId/members" -Method Get -Headers $headers
Write-Host "✓ 社群現在有 $($membersAfterKick.Count) 位成員（應該只剩擁有者）" -ForegroundColor Green

# 17. 測試擁有者離開（應該失敗）
Write-Host "`n17. 測試擁有者離開（應該失敗）..." -ForegroundColor Yellow
try {
    Invoke-RestMethod -Uri "$baseUrl/guilds/$guildId/leave" -Method Post -Headers $headers
    Write-Host "✗ 擁有者竟然可以離開社群！" -ForegroundColor Red
} catch {
    Write-Host "✓ 正確阻止擁有者離開: $($_.Exception.Response.StatusCode)" -ForegroundColor Green
}

# 18. 刪除社群
Write-Host "`n18. 刪除社群..." -ForegroundColor Yellow
$deleteResponse = Invoke-RestMethod -Uri "$baseUrl/guilds/$guildId" -Method Delete -Headers $headers
Write-Host "✓ 社群刪除成功: $($deleteResponse.message)" -ForegroundColor Green

# 19. 驗證社群已被刪除
Write-Host "`n19. 驗證社群已被刪除..." -ForegroundColor Yellow
try {
    Invoke-RestMethod -Uri "$baseUrl/guilds/$guildId" -Method Get -Headers $headers
    Write-Host "✗ 社群刪除後仍然存在！" -ForegroundColor Red
} catch {
    Write-Host "✓ 社群已正確刪除: $($_.Exception.Response.StatusCode)" -ForegroundColor Green
}

Write-Host "`n=== Guild API 測試完成 ===" -ForegroundColor Cyan
Write-Host "所有測試均已執行，請檢查上方結果" -ForegroundColor Cyan
