# Quick Start Scripts for Chorus

## Start Development Environment (Windows PowerShell)

### Start all services with Docker
```powershell
docker-compose up -d
```

### Start services manually

#### Terminal 1 - PostgreSQL (if not using Docker)
```powershell
# Start PostgreSQL service
# On Windows: Services → PostgreSQL → Start
# Or if using installed postgres
pg_ctl -D "C:\Program Files\PostgreSQL\15\data" start
```

#### Terminal 2 - Redis (if not using Docker)
```powershell
# Start Redis
redis-server
```

#### Terminal 3 - Backend
```powershell
cd backend
go run cmd/server/main.go
```

#### Terminal 4 - Frontend
```powershell
cd frontend
npm run dev
```

## Access the Application

- Frontend: http://localhost:3000
- Backend API: http://localhost:8080
- Health Check: http://localhost:8080/health

## Create Test Users

Use the registration page at http://localhost:3000/register or use the API directly:

```powershell
# User 1 (English → Spanish)
curl -X POST http://localhost:8080/api/v1/auth/register `
  -H "Content-Type: application/json" `
  -d '{
    "username": "alice",
    "email": "alice@example.com",
    "password": "password123",
    "displayName": "Alice",
    "nativeLanguage": "en",
    "targetLanguages": ["es"]
  }'

# User 2 (Spanish → English)
curl -X POST http://localhost:8080/api/v1/auth/register `
  -H "Content-Type: application/json" `
  -d '{
    "username": "bob",
    "email": "bob@example.com",
    "password": "password123",
    "displayName": "Bob",
    "nativeLanguage": "es",
    "targetLanguages": ["en"]
  }'
```

## Stop Services

### Docker
```powershell
docker-compose down
```

### Manual
Press Ctrl+C in each terminal window

## Reset Database

```powershell
# Stop all services first

# Drop and recreate database
psql -U postgres
DROP DATABASE messenger_dev;
CREATE DATABASE messenger_dev;
GRANT ALL PRIVILEGES ON DATABASE messenger_dev TO messenger;
\q

# Restart backend (migrations run automatically)
cd backend
go run cmd/server/main.go
```

## View Logs

### Docker
```powershell
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f backend
docker-compose logs -f frontend
```

### Manual
Logs appear in the terminal windows

## Build for Production

### Backend
```powershell
cd backend
go build -o chorus.exe ./cmd/server
```

### Frontend
```powershell
cd frontend
npm run build
```

### Docker
```powershell
docker-compose build
```

## Common Issues

### Port already in use
```powershell
# Find process using port 8080
netstat -ano | findstr :8080

# Kill the process (replace PID with actual process ID)
taskkill /PID <PID> /F
```

### Database connection failed
- Ensure PostgreSQL is running
- Check DATABASE_URL in .env file
- Verify database exists: `psql -U messenger -d messenger_dev`

### Redis connection failed
- Ensure Redis is running: `redis-cli ping`
- Should respond with "PONG"
