# Chorus Setup Helper
# Run this script to check prerequisites and setup the project

Write-Host "==================================" -ForegroundColor Cyan
Write-Host "Chorus - Multilingual Messenger" -ForegroundColor Cyan
Write-Host "Phase 1 Setup Helper" -ForegroundColor Cyan
Write-Host "==================================" -ForegroundColor Cyan
Write-Host ""

# Check prerequisites
Write-Host "Checking prerequisites..." -ForegroundColor Yellow
Write-Host ""

# Check Go
Write-Host "Checking Go..." -NoNewline
try {
    $goVersion = go version 2>$null
    if ($goVersion) {
        Write-Host " ✓ Found: $goVersion" -ForegroundColor Green
    } else {
        Write-Host " ✗ Not found" -ForegroundColor Red
        Write-Host "  Install from: https://go.dev/dl/" -ForegroundColor Yellow
    }
} catch {
    Write-Host " ✗ Not found" -ForegroundColor Red
    Write-Host "  Install from: https://go.dev/dl/" -ForegroundColor Yellow
}

# Check Node.js
Write-Host "Checking Node.js..." -NoNewline
try {
    $nodeVersion = node --version 2>$null
    if ($nodeVersion) {
        Write-Host " ✓ Found: $nodeVersion" -ForegroundColor Green
    } else {
        Write-Host " ✗ Not found" -ForegroundColor Red
        Write-Host "  Install from: https://nodejs.org/" -ForegroundColor Yellow
    }
} catch {
    Write-Host " ✗ Not found" -ForegroundColor Red
    Write-Host "  Install from: https://nodejs.org/" -ForegroundColor Yellow
}

# Check PostgreSQL
Write-Host "Checking PostgreSQL..." -NoNewline
try {
    $pgVersion = psql --version 2>$null
    if ($pgVersion) {
        Write-Host " ✓ Found: $pgVersion" -ForegroundColor Green
    } else {
        Write-Host " ✗ Not found" -ForegroundColor Red
        Write-Host "  Install from: https://www.postgresql.org/download/" -ForegroundColor Yellow
    }
} catch {
    Write-Host " ✗ Not found" -ForegroundColor Red
    Write-Host "  Install from: https://www.postgresql.org/download/" -ForegroundColor Yellow
}

# Check Redis
Write-Host "Checking Redis..." -NoNewline
try {
    $redisPing = redis-cli ping 2>$null
    if ($redisPing -eq "PONG") {
        Write-Host " ✓ Running" -ForegroundColor Green
    } else {
        Write-Host " ⚠ Not running" -ForegroundColor Yellow
        Write-Host "  Start Redis or install from: https://github.com/microsoftarchive/redis/releases" -ForegroundColor Yellow
    }
} catch {
    Write-Host " ⚠ Not found or not running" -ForegroundColor Yellow
    Write-Host "  Install Memurai (Windows): https://www.memurai.com/" -ForegroundColor Yellow
    Write-Host "  Or use Docker: docker run -d -p 6379:6379 redis:7-alpine" -ForegroundColor Yellow
}

# Check Docker (optional)
Write-Host "Checking Docker (optional)..." -NoNewline
try {
    $dockerVersion = docker --version 2>$null
    if ($dockerVersion) {
        Write-Host " ✓ Found: $dockerVersion" -ForegroundColor Green
    } else {
        Write-Host " ⚠ Not found (optional)" -ForegroundColor Yellow
    }
} catch {
    Write-Host " ⚠ Not found (optional)" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "==================================" -ForegroundColor Cyan
Write-Host "Setup Options" -ForegroundColor Cyan
Write-Host "==================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Choose your setup method:" -ForegroundColor Yellow
Write-Host "1. Docker Compose (Easiest - requires Docker)" -ForegroundColor White
Write-Host "2. Manual Setup (requires Go, Node.js, PostgreSQL, Redis)" -ForegroundColor White
Write-Host "3. Install Prerequisites First" -ForegroundColor White
Write-Host "4. Exit" -ForegroundColor White
Write-Host ""

$choice = Read-Host "Enter your choice (1-4)"

switch ($choice) {
    "1" {
        Write-Host ""
        Write-Host "Starting Docker Compose setup..." -ForegroundColor Green
        Write-Host ""
        
        # Check if Docker is available
        try {
            docker --version | Out-Null
            Write-Host "Building and starting containers..." -ForegroundColor Yellow
            docker-compose up -d
            
            Write-Host ""
            Write-Host "==================================" -ForegroundColor Green
            Write-Host "✓ Setup Complete!" -ForegroundColor Green
            Write-Host "==================================" -ForegroundColor Green
            Write-Host ""
            Write-Host "Your application is starting up..." -ForegroundColor Cyan
            Write-Host "Frontend: http://localhost:3000" -ForegroundColor White
            Write-Host "Backend:  http://localhost:8080" -ForegroundColor White
            Write-Host "Health:   http://localhost:8080/health" -ForegroundColor White
            Write-Host ""
            Write-Host "View logs: docker-compose logs -f" -ForegroundColor Yellow
            Write-Host "Stop:      docker-compose down" -ForegroundColor Yellow
        } catch {
            Write-Host "Error: Docker not found or not running" -ForegroundColor Red
            Write-Host "Please install Docker Desktop: https://www.docker.com/products/docker-desktop/" -ForegroundColor Yellow
        }
    }
    
    "2" {
        Write-Host ""
        Write-Host "Starting Manual Setup..." -ForegroundColor Green
        Write-Host ""
        
        # Check if .env exists
        if (-not (Test-Path "backend\.env")) {
            Write-Host "Creating backend/.env from template..." -ForegroundColor Yellow
            Copy-Item "backend\.env.example" "backend\.env"
            Write-Host "✓ Created backend/.env" -ForegroundColor Green
        }
        
        # Setup backend
        Write-Host ""
        Write-Host "Setting up backend..." -ForegroundColor Yellow
        Push-Location backend
        
        try {
            Write-Host "Downloading Go dependencies..." -ForegroundColor Cyan
            go mod download
            Write-Host "✓ Backend dependencies ready" -ForegroundColor Green
        } catch {
            Write-Host "✗ Failed to download Go dependencies" -ForegroundColor Red
            Write-Host "Make sure Go is installed: https://go.dev/dl/" -ForegroundColor Yellow
        }
        
        Pop-Location
        
        # Setup frontend
        Write-Host ""
        Write-Host "Setting up frontend..." -ForegroundColor Yellow
        Push-Location frontend
        
        try {
            Write-Host "Installing Node.js dependencies..." -ForegroundColor Cyan
            npm install
            Write-Host "✓ Frontend dependencies ready" -ForegroundColor Green
        } catch {
            Write-Host "✗ Failed to install npm dependencies" -ForegroundColor Red
            Write-Host "Make sure Node.js is installed: https://nodejs.org/" -ForegroundColor Yellow
        }
        
        Pop-Location
        
        Write-Host ""
        Write-Host "==================================" -ForegroundColor Green
        Write-Host "✓ Setup Complete!" -ForegroundColor Green
        Write-Host "==================================" -ForegroundColor Green
        Write-Host ""
        Write-Host "Next steps:" -ForegroundColor Cyan
        Write-Host "1. Ensure PostgreSQL is running with database 'messenger_dev'" -ForegroundColor White
        Write-Host "2. Ensure Redis is running" -ForegroundColor White
        Write-Host "3. Start backend:  cd backend; go run cmd/server/main.go" -ForegroundColor White
        Write-Host "4. Start frontend: cd frontend; npm run dev" -ForegroundColor White
        Write-Host ""
        Write-Host "See INSTALLATION.md for detailed setup instructions" -ForegroundColor Yellow
    }
    
    "3" {
        Write-Host ""
        Write-Host "Please install the following prerequisites:" -ForegroundColor Yellow
        Write-Host ""
        Write-Host "1. Go 1.21+:         https://go.dev/dl/" -ForegroundColor White
        Write-Host "2. Node.js 18+:      https://nodejs.org/" -ForegroundColor White
        Write-Host "3. PostgreSQL 15+:   https://www.postgresql.org/download/" -ForegroundColor White
        Write-Host "4. Redis/Memurai:    https://www.memurai.com/" -ForegroundColor White
        Write-Host "5. Docker (optional):https://www.docker.com/products/docker-desktop/" -ForegroundColor White
        Write-Host ""
        Write-Host "After installation, run this script again." -ForegroundColor Cyan
        Write-Host ""
        Write-Host "See INSTALLATION.md for detailed instructions" -ForegroundColor Yellow
    }
    
    "4" {
        Write-Host "Exiting..." -ForegroundColor Yellow
        exit
    }
    
    default {
        Write-Host "Invalid choice. Exiting..." -ForegroundColor Red
    }
}

Write-Host ""
Write-Host "For more information, see:" -ForegroundColor Cyan
Write-Host "- README.md for overview" -ForegroundColor White
Write-Host "- INSTALLATION.md for prerequisites" -ForegroundColor White
Write-Host "- QUICK_START.md for development guide" -ForegroundColor White
Write-Host "- IMPLEMENTATION_SUMMARY.md for technical details" -ForegroundColor White
Write-Host ""
