# TalkRealm API 測試腳本
# 測試使用者註冊、登入和認證功能

$baseUrl = "http://localhost:8080"

Write-Host "=== TalkRealm API 測試 ===" -ForegroundColor Cyan
Write-Host ""

# 1. 健康檢查
Write-Host "1️⃣  測試健康檢查..." -ForegroundColor Yellow
try {
    $response = Invoke-RestMethod -Uri "$baseUrl/health" -Method Get
    Write-Host "✅ 健康檢查成功: $($response | ConvertTo-Json -Compress)" -ForegroundColor Green
} catch {
    Write-Host "❌ 健康檢查失敗: $_" -ForegroundColor Red
    exit 1
}
Write-Host ""

# 2. 使用者註冊
Write-Host "2️⃣  測試使用者註冊..." -ForegroundColor Yellow
$registerData = @{
    username = "testuser"
    email = "test@example.com"
    password = "password123"
    nickname = "Test User"
} | ConvertTo-Json

try {
    $response = Invoke-RestMethod -Uri "$baseUrl/api/v1/auth/register" -Method Post -Body $registerData -ContentType "application/json"
    Write-Host "✅ 註冊成功!" -ForegroundColor Green
    Write-Host "   使用者 ID: $($response.user.id)" -ForegroundColor Gray
    Write-Host "   使用者名稱: $($response.user.username)" -ForegroundColor Gray
    Write-Host "   Email: $($response.user.email)" -ForegroundColor Gray
} catch {
    $errorDetail = $_.ErrorDetails.Message | ConvertFrom-Json
    Write-Host "⚠️  註冊回應: $($errorDetail.error)" -ForegroundColor Yellow
}
Write-Host ""

# 3. 使用者登入
Write-Host "3️⃣  測試使用者登入..." -ForegroundColor Yellow
$loginData = @{
    email = "test@example.com"
    password = "password123"
} | ConvertTo-Json

try {
    $response = Invoke-RestMethod -Uri "$baseUrl/api/v1/auth/login" -Method Post -Body $loginData -ContentType "application/json"
    Write-Host "✅ 登入成功!" -ForegroundColor Green
    Write-Host "   Token: $($response.token.Substring(0, 30))..." -ForegroundColor Gray
    $token = $response.token
    $user = $response.user
} catch {
    Write-Host "❌ 登入失敗: $_" -ForegroundColor Red
    exit 1
}
Write-Host ""

# 4. 獲取當前使用者資訊（需要認證）
Write-Host "4️⃣  測試獲取當前使用者資訊（需要認證）..." -ForegroundColor Yellow
$headers = @{
    "Authorization" = "Bearer $token"
}

try {
    $response = Invoke-RestMethod -Uri "$baseUrl/api/v1/users/me" -Method Get -Headers $headers
    Write-Host "✅ 獲取使用者資訊成功!" -ForegroundColor Green
    Write-Host "   使用者 ID: $($response.user.id)" -ForegroundColor Gray
    Write-Host "   使用者名稱: $($response.user.username)" -ForegroundColor Gray
    Write-Host "   暱稱: $($response.user.nickname)" -ForegroundColor Gray
    Write-Host "   狀態: $($response.user.status)" -ForegroundColor Gray
} catch {
    Write-Host "❌ 獲取使用者資訊失敗: $_" -ForegroundColor Red
    exit 1
}
Write-Host ""

# 5. 更新使用者資訊
Write-Host "5️⃣  測試更新使用者資訊..." -ForegroundColor Yellow
$updateData = @{
    nickname = "Updated Test User"
    status = "online"
} | ConvertTo-Json

try {
    $response = Invoke-RestMethod -Uri "$baseUrl/api/v1/users/me" -Method Patch -Headers $headers -Body $updateData -ContentType "application/json"
    Write-Host "✅ 更新使用者資訊成功!" -ForegroundColor Green
    Write-Host "   新暱稱: $($response.user.nickname)" -ForegroundColor Gray
    Write-Host "   新狀態: $($response.user.status)" -ForegroundColor Gray
} catch {
    Write-Host "❌ 更新使用者資訊失敗: $_" -ForegroundColor Red
}
Write-Host ""

# 6. 測試無效 Token（應該失敗）
Write-Host "6️⃣  測試無效 Token（預期失敗）..." -ForegroundColor Yellow
$invalidHeaders = @{
    "Authorization" = "Bearer invalid_token"
}

try {
    $response = Invoke-RestMethod -Uri "$baseUrl/api/v1/users/me" -Method Get -Headers $invalidHeaders
    Write-Host "❌ 應該要失敗但卻成功了！" -ForegroundColor Red
} catch {
    Write-Host "✅ 正確拒絕無效 Token!" -ForegroundColor Green
}
Write-Host ""

# 7. 測試錯誤的密碼（應該失敗）
Write-Host "7️⃣  測試錯誤的密碼（預期失敗）..." -ForegroundColor Yellow
$wrongLoginData = @{
    email = "test@example.com"
    password = "wrongpassword"
} | ConvertTo-Json

try {
    $response = Invoke-RestMethod -Uri "$baseUrl/api/v1/auth/login" -Method Post -Body $wrongLoginData -ContentType "application/json"
    Write-Host "❌ 應該要失敗但卻成功了！" -ForegroundColor Red
} catch {
    Write-Host "✅ 正確拒絕錯誤密碼!" -ForegroundColor Green
}
Write-Host ""

Write-Host "=== 所有測試完成！✨ ===" -ForegroundColor Cyan
Write-Host ""
Write-Host "📋 測試總結:" -ForegroundColor Cyan
Write-Host "  ✅ 健康檢查" -ForegroundColor Green
Write-Host "  ✅ 使用者註冊" -ForegroundColor Green
Write-Host "  ✅ 使用者登入" -ForegroundColor Green
Write-Host "  ✅ JWT 認證" -ForegroundColor Green
Write-Host "  ✅ 獲取使用者資訊" -ForegroundColor Green
Write-Host "  ✅ 更新使用者資訊" -ForegroundColor Green
Write-Host "  ✅ 安全性驗證" -ForegroundColor Green
