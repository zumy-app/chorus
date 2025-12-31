# Test Script for Chorus Messenger Integration
Write-Host "Testing Chorus Messenger Integration..." -ForegroundColor Green

# Test 1: Health check
Write-Host "`n1. Testing Backend Health..." -ForegroundColor Yellow
$health = Invoke-RestMethod -Uri "http://localhost:8080/health" -Method Get
Write-Host "   Health Status: $($health.status)" -ForegroundColor Cyan

# Test 2: Register a user
Write-Host "`n2. Registering a test user..." -ForegroundColor Yellow
$randomId = Get-Random -Maximum 10000
$registerBody = @{
    username = "testuser_$randomId"
    email = "test$randomId@example.com"
    password = "TestPassword123!"
    displayName = "Test User"
    nativeLanguage = "en"
    targetLanguages = @("es", "fr")
} | ConvertTo-Json

try {
    $registerResponse = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/auth/register" `
        -Method Post `
        -Body $registerBody `
        -ContentType "application/json"
    Write-Host "   User registered successfully!" -ForegroundColor Green
    Write-Host "   User ID: $($registerResponse.user.id)" -ForegroundColor Cyan
    $token = $registerResponse.tokens.accessToken
    $username = "testuser_$randomId"
} catch {
    Write-Host "   Registration failed: $_" -ForegroundColor Red
    exit 1
}

# Test 3: Login
Write-Host "`n3. Testing login..." -ForegroundColor Yellow
$loginBody = @{
    username = $username
    password = "TestPassword123!"
} | ConvertTo-Json

try {
    $loginResponse = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/auth/login" `
        -Method Post `
        -Body $loginBody `
        -ContentType "application/json"
    Write-Host "   Login successful!" -ForegroundColor Green
    $token = $loginResponse.tokens.accessToken
} catch {
    Write-Host "   Login failed: $_" -ForegroundColor Red
    exit 1
}

# Test 4: Get user profile
Write-Host "`n4. Getting user profile..." -ForegroundColor Yellow
try {
    $headers = @{
        Authorization = "Bearer $token"
    }
    $profile = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/users/me" `
        -Method Get `
        -Headers $headers
    Write-Host "   Profile retrieved successfully!" -ForegroundColor Green
    Write-Host "   Username: $($profile.username)" -ForegroundColor Cyan
    Write-Host "   Display Name: $($profile.displayName)" -ForegroundColor Cyan
} catch {
    Write-Host "   Failed to get profile: $_" -ForegroundColor Red
    exit 1
}

# Test 5: List chats (should be empty)
Write-Host "`n5. Listing chats..." -ForegroundColor Yellow
try {
    $chats = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/chats" `
        -Method Get `
        -Headers $headers
    Write-Host "   Chats retrieved successfully!" -ForegroundColor Green
    Write-Host "   Number of chats: $($chats.Count)" -ForegroundColor Cyan
} catch {
    Write-Host "   Failed to list chats: $_" -ForegroundColor Red
}

# Test 6: Check database connection
Write-Host "`n6. Verifying database..." -ForegroundColor Yellow
$dbStatus = docker exec chorus-postgres pg_isready -U messenger 2>&1
if ($LASTEXITCODE -eq 0) {
    Write-Host "   PostgreSQL is ready!" -ForegroundColor Green
} else {
    Write-Host "   PostgreSQL check failed" -ForegroundColor Red
}

# Test 7: Check Redis connection
Write-Host "`n7. Verifying Redis..." -ForegroundColor Yellow
$redisStatus = docker exec chorus-redis redis-cli ping 2>&1
if ($redisStatus -match "PONG") {
    Write-Host "   Redis is ready!" -ForegroundColor Green
} else {
    Write-Host "   Redis check failed" -ForegroundColor Red
}

Write-Host "`n========================================" -ForegroundColor Green
Write-Host "Integration Tests Completed!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host "`nServices Running:" -ForegroundColor Yellow
Write-Host "  Frontend: http://localhost:3000" -ForegroundColor Cyan
Write-Host "  Backend:  http://localhost:8080" -ForegroundColor Cyan
Write-Host "  PostgreSQL: localhost:5432" -ForegroundColor Cyan
Write-Host "  Redis: localhost:6379" -ForegroundColor Cyan
Write-Host "`nAppwrite Configuration:" -ForegroundColor Yellow
Write-Host "  Endpoint: https://appwrite.zumy.app/v1" -ForegroundColor Cyan
Write-Host "  Project ID: 6955733a001c10e7c020" -ForegroundColor Cyan
Write-Host "  Database ID: 6955749200181b13d9d6" -ForegroundColor Cyan
