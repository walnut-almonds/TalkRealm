# 簡單的 API 測試
Write-Host "測試 API..." -ForegroundColor Cyan

# 健康檢查
Write-Host "`n1. 健康檢查" -ForegroundColor Yellow
$response = Invoke-RestMethod -Uri "http://localhost:8080/health"
Write-Host "✅ $($response | ConvertTo-Json)" -ForegroundColor Green

# 註冊
Write-Host "`n2. 註冊使用者" -ForegroundColor Yellow
$registerData = @{
    username = "alice"
    email = "alice@example.com"
    password = "password123"
    nickname = "Alice Wang"
} | ConvertTo-Json

try {
    $response = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/auth/register" -Method Post -Body $registerData -ContentType "application/json"
    Write-Host "✅ 註冊成功: User ID = $($response.user.id)" -ForegroundColor Green
} catch {
    Write-Host "⚠️  $($_.ErrorDetails.Message)" -ForegroundColor Yellow
}

# 登入
Write-Host "`n3. 登入" -ForegroundColor Yellow
$loginData = @{
    email = "alice@example.com"
    password = "password123"
} | ConvertTo-Json

$response = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/auth/login" -Method Post -Body $loginData -ContentType "application/json"
Write-Host "✅ 登入成功！Token: $($response.token.Substring(0,30))..." -ForegroundColor Green
$token = $response.token

# 獲取使用者資訊
Write-Host "`n4. 獲取使用者資訊（需認證）" -ForegroundColor Yellow
$headers = @{ "Authorization" = "Bearer $token" }
$response = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/users/me" -Headers $headers
Write-Host "✅ 使用者: $($response.user.nickname) (@$($response.user.username))" -ForegroundColor Green

# 更新使用者
Write-Host "`n5. 更新使用者資訊" -ForegroundColor Yellow
$updateData = @{
    nickname = "Alice Updated"
    status = "online"
} | ConvertTo-Json
$response = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/users/me" -Method Patch -Headers $headers -Body $updateData -ContentType "application/json"
Write-Host "✅ 更新成功: $($response.user.nickname) - $($response.user.status)" -ForegroundColor Green

Write-Host "`n✨ 所有測試完成！" -ForegroundColor Cyan
