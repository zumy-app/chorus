# Installation Guide for Chorus

## Prerequisites Installation

### 1. Install Go (Required for Backend)

1. Download Go 1.21 or later from https://go.dev/dl/
2. Run the installer
3. Verify installation:
   ```powershell
   go version
   ```

### 2. Install Node.js (Required for Frontend)

1. Download Node.js 18 LTS or later from https://nodejs.org/
2. Run the installer (includes npm)
3. Verify installation:
   ```powershell
   node --version
   npm --version
   ```

### 3. Install PostgreSQL (Required for Database)

1. Download PostgreSQL 15 from https://www.postgresql.org/download/windows/
2. Run the installer
3. Remember the password you set for the postgres user
4. Verify installation:
   ```powershell
   psql --version
   ```

5. Create the database:
   ```powershell
   # Connect to PostgreSQL
   psql -U postgres

   # Run these commands in psql
   CREATE DATABASE messenger_dev;
   CREATE USER messenger WITH PASSWORD 'password';
   GRANT ALL PRIVILEGES ON DATABASE messenger_dev TO messenger;
   \q
   ```

### 4. Install Redis (Required for Caching)

**Option A: Using WSL (Recommended)**
```powershell
# Enable WSL if not already enabled
wsl --install

# Install Redis in WSL
wsl
sudo apt update
sudo apt install redis-server
sudo service redis-server start
redis-cli ping  # Should return PONG
```

**Option B: Using Memurai (Windows Native)**
1. Download Memurai from https://www.memurai.com/
2. Run the installer
3. Start Memurai service
4. Verify:
   ```powershell
   redis-cli ping  # Should return PONG
   ```

**Option C: Using Docker**
```powershell
docker run -d -p 6379:6379 redis:7-alpine
```

### 5. Install Docker (Optional, for containerized deployment)

1. Download Docker Desktop from https://www.docker.com/products/docker-desktop/
2. Run the installer
3. Restart your computer
4. Verify installation:
   ```powershell
   docker --version
   docker-compose --version
   ```

## After Installing Prerequisites

### Setup Backend

```powershell
cd C:\dev\chorus\backend

# Download Go dependencies
go mod download

# Copy environment file
Copy-Item .env.example .env

# Edit .env file with your settings if needed

# Run the backend
go run cmd/server/main.go
```

The backend will:
- Connect to PostgreSQL
- Run database migrations automatically
- Start on http://localhost:8080

### Setup Frontend

```powershell
cd C:\dev\chorus\frontend

# Install dependencies
npm install

# Start development server
npm run dev
```

The frontend will start on http://localhost:3000

## Using Docker (Easiest Method)

If you have Docker installed, you can skip individual installations:

```powershell
cd C:\dev\chorus

# Start everything
docker-compose up -d

# View logs
docker-compose logs -f

# Stop everything
docker-compose down
```

This will automatically set up:
- PostgreSQL database
- Redis cache
- Go backend
- React frontend

Access the app at http://localhost:3000

## Verify Installation

1. Backend health check: http://localhost:8080/health
   - Should return: `{"status":"healthy","version":"1.0.0"}`

2. Frontend: http://localhost:3000
   - Should show the login page

3. Create a test account and start messaging!

## Troubleshooting

### Go not found
- Add Go to your PATH: `C:\Go\bin`
- Restart your terminal/PowerShell

### PostgreSQL connection failed
- Check if PostgreSQL service is running
- Verify credentials in .env file
- Check if database exists: `psql -U messenger -d messenger_dev`

### Redis connection failed
- Check if Redis/Memurai service is running
- For WSL: `wsl sudo service redis-server start`
- For Memurai: Check Services → Memurai

### Port already in use
- Backend (8080): `netstat -ano | findstr :8080`
- Frontend (3000): `netstat -ano | findstr :3000`
- Kill the process: `taskkill /PID <PID> /F`

## Next Steps

After installation, see [QUICK_START.md](QUICK_START.md) for:
- Creating test users
- Starting the application
- Common workflows
- Development tips
