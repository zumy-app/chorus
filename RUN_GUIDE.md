# Chorus - Complete Run Guide

This guide covers how to start all services and test the application.

## 🚀 Quick Start (Docker - Recommended)

The easiest way to run Chorus is with Docker Compose, which sets up PostgreSQL, Redis, Backend, and Frontend automatically.

### Prerequisites
- Docker Desktop installed and running

### Start Services
```powershell
cd C:\dev\chorus
docker-compose up -d
```

### Verify Services
```powershell
# Check all containers are running
docker-compose ps

# You should see 4 containers with status "Up":
# - chorus-postgres
# - chorus-redis
# - chorus-backend
# - chorus-frontend
```

### Access the Application
- **Frontend**: http://localhost:3000
- **Backend API**: http://localhost:8080
- **Health Check**: http://localhost:8080/health
- **WebSocket**: ws://localhost:8080/ws

### Stop Services
```powershell
docker-compose down
```

---

## 🛠️ Manual Setup (4 Separate Terminals)

For more control or if you prefer not to use Docker, follow this terminal-by-terminal guide.

### Prerequisites
- Go 1.23+
- Node.js 18+ (20+ recommended)
- PostgreSQL 15+
- Redis 7+

---

## Terminal 1: PostgreSQL

### First Time Setup
```powershell
# Connect to PostgreSQL as admin
psql -U postgres

# In psql, create database and user:
CREATE DATABASE messenger_dev;
CREATE USER messenger WITH PASSWORD 'password';
GRANT ALL PRIVILEGES ON DATABASE messenger_dev TO messenger;
\q
```

### Start PostgreSQL Service
```powershell
# Option A: Start via Windows Services
# Services → PostgreSQL → Right-click → Start

# Option B: Via Command Line
pg_ctl -D "C:\Program Files\PostgreSQL\15\data" start

# Option C: If installed via Chocolatey/Scoop
pg_ctl start

# Verify connection
psql -U messenger -d messenger_dev -c "SELECT 1"
# Output: Returns 1 if successful
```

**Keep this terminal open.** PostgreSQL runs in the background.

---

## Terminal 2: Redis

### Start Redis
```powershell
# Option A: WSL (Windows Subsystem for Linux)
wsl
sudo service redis-server start
redis-cli ping
# Output: PONG

# Option B: Memurai (Windows Native)
# Services → Memurai → Right-click → Start
redis-cli ping
# Output: PONG

# Option C: Docker
docker run -d -p 6379:6379 --name chorus-redis redis:7-alpine
docker logs -f chorus-redis
```

**Keep this terminal open.** Redis runs in the background.

---

## Terminal 3: Go Backend

```powershell
cd C:\dev\chorus\backend

# First time only: Download Go dependencies
go mod download

# First time only: Create .env from example
Copy-Item .env.example .env

# Optional: Edit .env if you have Google Translate API key
# Otherwise, it will use mock translations

# Run the backend
go run cmd/server/main.go

# Expected output:
# No .env file found, using system environment variables
# Database connected successfully
# Redis connected successfully
# Server starting on port 8080 (Phase 2 & 3 features enabled)
```

**Keep this terminal open.** The backend will stay running and show request logs.

---

## Terminal 4: React Frontend

```powershell
cd C:\dev\chorus\frontend

# First time only: Install npm dependencies
npm install

# Start Vite development server
npm run dev

# Expected output:
# VITE v5.x.x  ready in xxx ms
# ➜  Local:   http://localhost:5173/
# ➜  Press h to show help

# Or if configured for port 3000:
# ➜  Local:   http://localhost:3000/
```

**Keep this terminal open.** The frontend will reload automatically when you edit files.

---

## 🧪 Testing Login & Features

### Test 1: Health Check
```powershell
# Verify backend is running
curl http://localhost:8080/health

# Expected response: {"status":"healthy","version":"2.0.0"}
```

### Test 2: Registration & Login (UI Method)

1. **Open** http://localhost:3000 in browser
2. **Click "Register"** link
3. **Form is pre-filled** with:
   - Email: `uhsarp@gmail.com`
   - Password: `Demor@cer1`
   - Display Name: `Prashanth`
   - Native Language: `en`
   - Target Languages: `es` (Spanish)
4. **Click "Register"**
5. **Auto-login** → Redirected to `/chat`
6. **Verify** you see the chat interface with user info

### Test 3: Registration via API (PowerShell)

**Create User 1 (English → Spanish):**
```powershell
$body = @{
    email = "alice@example.com"
    password = "Password123!"
    displayName = "Alice"
    nativeLanguage = "en"
    targetLanguages = @("es")
} | ConvertTo-Json

$response = Invoke-WebRequest -Uri "http://localhost:8080/api/v1/auth/register" `
  -Method Post `
  -ContentType "application/json" `
  -Body $body

$response.Content | ConvertFrom-Json

# Save the accessToken and refreshToken for next steps
```

**Create User 2 (Spanish → English):**
```powershell
$body = @{
    email = "bob@example.com"
    password = "Password123!"
    displayName = "Bob"
    nativeLanguage = "es"
    targetLanguages = @("en")
} | ConvertTo-Json

$response = Invoke-WebRequest -Uri "http://localhost:8080/api/v1/auth/register" `
  -Method Post `
  -ContentType "application/json" `
  -Body $body

$response.Content | ConvertFrom-Json
```

### Test 4: Login via API

```powershell
$body = @{
    username = "alice@example.com"
    password = "Password123!"
} | ConvertTo-Json

$response = Invoke-WebRequest -Uri "http://localhost:8080/api/v1/auth/login" `
  -Method Post `
  -ContentType "application/json" `
  -Body $body

$data = $response.Content | ConvertFrom-Json
$data | ConvertTo-Json -Depth 10

# Extract tokens for next tests
$accessToken = $data.tokens.accessToken
$refreshToken = $data.tokens.refreshToken

Write-Host "Access Token: $accessToken"
Write-Host "Refresh Token: $refreshToken"
```

### Test 5: Get Current User (Protected Endpoint)

```powershell
$headers = @{
    Authorization = "Bearer $accessToken"
}

$response = Invoke-WebRequest -Uri "http://localhost:8080/api/v1/users/me" `
  -Headers $headers

$response.Content | ConvertFrom-Json
```

### Test 6: Load Chats

```powershell
$headers = @{
    Authorization = "Bearer $accessToken"
}

$response = Invoke-WebRequest -Uri "http://localhost:8080/api/v1/chats" `
  -Headers $headers

$response.Content | ConvertFrom-Json
```

### Test 7: Create Direct Chat

```powershell
$headers = @{
    Authorization = "Bearer $accessToken"
}

$body = @{
    type = "direct"
    participants = @("bob@example.com")  # or use the actual Bob's user ID
} | ConvertTo-Json

$response = Invoke-WebRequest -Uri "http://localhost:8080/api/v1/chats" `
  -Method Post `
  -Headers $headers `
  -ContentType "application/json" `
  -Body $body

$data = $response.Content | ConvertFrom-Json
$chatId = $data.id

Write-Host "Chat Created: $chatId"
```

### Test 8: Send Message

```powershell
$headers = @{
    Authorization = "Bearer $accessToken"
}

$body = @{
    text = "Hello, how are you?"
} | ConvertTo-Json

$response = Invoke-WebRequest -Uri "http://localhost:8080/api/v1/chats/$chatId/messages" `
  -Method Post `
  -Headers $headers `
  -ContentType "application/json" `
  -Body $body

$response.Content | ConvertFrom-Json
```

### Test 9: Get Messages

```powershell
$headers = @{
    Authorization = "Bearer $accessToken"
}

$response = Invoke-WebRequest -Uri "http://localhost:8080/api/v1/chats/$chatId/messages" `
  -Headers $headers

$response.Content | ConvertFrom-Json
```

---

## 📊 Monitoring Services

### View Backend Logs
```powershell
# Terminal 3 where backend is running shows live logs:
# [GIN] | 200 | GET    /api/v1/users/me
# [GIN] | 201 | POST   /api/v1/chats
```

### View Frontend Console
```
Browser → F12 → Console tab
Look for:
- "WebSocket connected"
- Any error messages
```

### Docker Logs
```powershell
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f backend
docker-compose logs -f frontend
docker-compose logs -f postgres

# Recent 50 lines
docker-compose logs --tail=50 backend
```

### Redis Status
```powershell
redis-cli ping
# Output: PONG

redis-cli info server
# Shows Redis version, uptime, etc.
```

### PostgreSQL Status
```powershell
psql -U messenger -d messenger_dev -c "SELECT version();"
```

---

## 🔧 Troubleshooting

### Port Already in Use

**Port 3000/5173 in use (Frontend):**
```powershell
# Find what's using the port
netstat -ano | findstr :3000

# Kill the process (replace PID with actual number)
taskkill /PID <PID> /F

# Or change Vite port in frontend/vite.config.ts
```

**Port 8080 in use (Backend):**
```powershell
# Find what's using the port
netstat -ano | findstr :8080

# Kill the process
taskkill /PID <PID> /F

# Or change port in backend or set environment variable
$env:PORT = "8081"
```

### PostgreSQL Connection Error

```powershell
# Verify PostgreSQL service is running
Get-Service PostgreSQL*

# If not running, start it
Start-Service PostgreSQL*

# Test connection
psql -U messenger -d messenger_dev -c "SELECT 1"

# Check credentials in .env or docker-compose.yml
```

### Redis Connection Error

```powershell
# Test Redis
redis-cli ping
# Should return: PONG

# If fails, ensure Redis is running
# WSL: wsl && sudo service redis-server status
# Memurai: Services → Memurai → Verify running
```

### Backend Won't Connect to Database

```powershell
# Check DATABASE_URL environment variable
echo $env:DATABASE_URL

# Manually test connection
psql -U messenger -d messenger_dev -h localhost

# If using Docker, ensure postgres service is healthy
docker-compose ps postgres
# Status should show "Up (healthy)"
```

### WebSocket Connection Fails

```
Browser Console (F12):
- Check for errors like "WebSocket is closed"
- Verify backend is running on 8080
- Check CORS policy in backend/cmd/server/main.go
```

### Translation Shows Original Text

This is normal if Google Translate API key is not configured. The app uses mock translations instead.

**To enable real translations:**
1. Get API key from Google Cloud Console
2. Set environment variable: `GOOGLE_TRANSLATE_API_KEY=your-key`
3. Restart backend

---

## 🗑️ Cleanup & Reset

### Reset Database (Deletes All Data)

```powershell
# Stop all services first

# Option 1: Docker
docker-compose down -v  # -v removes volumes too

# Option 2: Manual
psql -U postgres
DROP DATABASE messenger_dev;
CREATE DATABASE messenger_dev;
GRANT ALL PRIVILEGES ON DATABASE messenger_dev TO messenger;
\q

# Restart backend (migrations run automatically)
cd C:\dev\chorus\backend
go run cmd/server/main.go
```

### Clear Frontend Cache

```powershell
# Delete node_modules (if needed)
cd C:\dev\chorus\frontend
rm -r node_modules
npm install

# Or in PowerShell
Remove-Item -Recurse -Force node_modules
npm install
```

---

## 📝 Environment Variables

### Backend (.env or docker-compose.yml)

| Variable | Default | Description |
|----------|---------|-------------|
| `ENVIRONMENT` | development | development or production |
| `DATABASE_URL` | postgres://... | PostgreSQL connection string |
| `REDIS_URL` | localhost:6379 | Redis address |
| `JWT_SECRET` | your-secret-key... | JWT signing secret |
| `GOOGLE_TRANSLATE_API_KEY` | (empty) | Optional Google Translate API key |
| `PORT` | 8080 | Server port |

### Frontend (src/services/api.ts)

Auto-detects platform:
- Web: `/api/v1`
- Android Emulator: `http://10.0.2.2:8080/api/v1`
- iOS Emulator: `http://localhost:8080/api/v1`

---

## ✅ Verification Checklist

After starting services, verify each step:

```
☐ Docker containers running (or manual services started)
☐ PostgreSQL: psql -U messenger -d messenger_dev -c "SELECT 1"
☐ Redis: redis-cli ping → PONG
☐ Backend: curl http://localhost:8080/health → {"status":"healthy",...}
☐ Frontend loads: http://localhost:3000
☐ Can register: Enter form → Click Register
☐ Auto-login: Redirected to /chat page
☐ See user info in sidebar
☐ Can create chat: Click "New Chat"
☐ Can send message: Type → Click send
☐ WebSocket connected: Browser console shows "WebSocket connected"
```

---

## 🎯 Next Steps

Once you have services running:

1. **Test the UI** following [Test 2: Registration & Login](#test-2-registration--login-ui-method)
2. **Test the API** following tests 3-9 above
3. **Read** [ARCHITECTURE.md](ARCHITECTURE.md) to understand the system design
4. **Explore** [API_ENDPOINTS.md](API_ENDPOINTS.md) for complete API reference
5. **Check** backend logs for any errors or performance insights

---

## 💡 Tips

- **Hot reload**: Frontend hot-reloads on file changes; backend requires restart
- **Debug WebSocket**: Browser → F12 → Network → ws:// tab
- **Debug API calls**: Browser → F12 → Network → XHR/Fetch tab
- **Backend debug**: Add `log.Printf(...)` to services, run backend in terminal
- **Database schema**: Check [backend/internal/database/postgres.go](backend/internal/database/postgres.go) for full schema

