<#
.SYNOPSIS
    Start Chorus development environment with hot-reload for all services.
.DESCRIPTION
    Ensures Docker Desktop is running, launches Docker services (PostgreSQL,
    Redis, LibreTranslate, Ollama), then starts the Go backend (with air
    auto-reload) and React frontend (Vite HMR) in their own terminal windows.

    Run this from the repository root:
        .\start-dev.ps1
#>

$RootDir = $PSScriptRoot
$BackendDir = Join-Path $RootDir "backend"
$FrontendDir = Join-Path $RootDir "frontend"
$DevCompose = Join-Path $RootDir "docker-compose.dev.yml"

# Helper: log with colour
function Log($Msg, $Color = "White") { Write-Host "$Msg" -ForegroundColor $Color }
function Ok($Msg)  { Write-Host "  ✓ $Msg" -ForegroundColor Green }
function Warn($Msg) { Write-Host "  ⚠ $Msg" -ForegroundColor Yellow }
function Fail($Msg) { Write-Host "  ✘ $Msg" -ForegroundColor Red }

# ──────────────────────────────────────────────
# Header
# ──────────────────────────────────────────────
Log "╔══════════════════════════════════════════════════════════╗" Cyan
Log "║   Chorus Dev Environment — Starting all services...     ║" Cyan
Log "╚══════════════════════════════════════════════════════════╝" Cyan
Log ""

# ──────────────────────────────────────────────
# 0. Ensure Docker Desktop is running
# ──────────────────────────────────────────────
Log "▶ [0/5] Checking Docker Desktop..." Yellow

# Test if the Docker CLI responds
$dockerOk = $false
for ($attempt = 0; $attempt -lt 20; $attempt++) {
    $null = docker ps 2>$null
    if ($LASTEXITCODE -eq 0) { $dockerOk = $true; break }

    if ($attempt -eq 0) {
        Warn "Docker CLI not responding. Attempting to start Docker Desktop..."
        $dockerPath = "C:\Program Files\Docker\Docker\Docker Desktop.exe"
        if (Test-Path $dockerPath) {
            Start-Process -FilePath $dockerPath -WindowStyle Hidden
            Log "  Waiting for Docker Desktop to start (up to 60s)..." Yellow
        } else {
            Warn "Docker Desktop not found at '$dockerPath'."
            Log "  Please start Docker Desktop manually and re-run this script." Yellow
        }
    }
    Start-Sleep -Seconds 3
}

if (-not $dockerOk) {
    Fail "Docker Desktop is not running. Start it manually, then re-run this script."
    Log ""
    Log "  Try: & 'C:\Program Files\Docker\Docker\Docker Desktop.exe'" Yellow
    exit 1
}
Ok "Docker Desktop is running"
Log ""

# ──────────────────────────────────────────────
# 1. Start Docker services (detached)
# ──────────────────────────────────────────────
Log "▶ [1/5] Starting Docker services (PostgreSQL, Redis, LibreTranslate, Ollama)..." Yellow

# Check for port conflicts before starting
$PortsToCheck = @{5433 = "postgres-dev"; 6380 = "redis-dev"; 5002 = "libretranslate-dev"; 11435 = "ollama-dev"}
$portBlockers = @()
foreach ($port in $PortsToCheck.Keys) {
    $processInfo = netstat -ano -p tcp 2>$null | Select-String ":$port\s" | Select-String "LISTENING"
    if ($processInfo) {
        $processInfo | ForEach-Object {
            if ($_ -match "(\d+)$") {
                $foundPid = $Matches[1]
                $proc = Get-Process -Id $foundPid -ErrorAction SilentlyContinue
                $procName = if ($proc) { $proc.ProcessName } else { "PID $foundPid" }
                $portBlockers += @{Port = $port; Service = $PortsToCheck[$port]; Process = $procName; PID = $foundPid }
            }
        }
    }
}

if ($portBlockers.Count -gt 0) {
    Warn "Port conflict(s) detected:"
    $portBlockers | ForEach-Object {
        Log "    Port $($_.Port) ($($_.Service)) is used by '$($_.Process)' (PID $($_.PID))" Yellow
        # Try to stop it if it's a Docker container
        $containerName = docker ps --format "{{.Names}}" --filter "publish=$($_.Port)" 2>$null
        if ($containerName) {
            Log "    -> Stopping conflicting Docker container '$containerName'..." Yellow
            docker stop $containerName 2>$null | Out-Null
            docker rm $containerName 2>$null | Out-Null
        }
    }
    # Re-check
    Start-Sleep -Seconds 1
    $stillBlocked = $portBlockers | Where-Object {
        $pInfo = netstat -ano -p tcp 2>$null | Select-String ":$($_.Port)\s" | Select-String "LISTENING"
        $pInfo -ne $null
    }
    if ($stillBlocked) {
        Warn "Some ports are still in use. Docker services that conflict may fail to start."
        Warn "Close the conflicting applications manually and re-run, or the script will continue anyway."
    }
    Log ""
}

# Start Docker services (with --remove-orphans to clean up stale containers)
docker-compose -f $DevCompose up -d --remove-orphans postgres-dev redis-dev libretranslate-dev ollama-dev 2>&1 | ForEach-Object {
    $line = $_.ToString()
    # Suppress the version warning
    if ($line -notmatch "the attribute .version. is obsolete") {
        Write-Host "  $line"
    }
}

if ($LASTEXITCODE -ne 0) {
    Warn "Docker compose had some issues. Checking what started successfully..."
} else {
    Ok "Docker services started"
}
Log ""

# ──────────────────────────────────────────────
# 2. Wait for PostgreSQL to be healthy
# ──────────────────────────────────────────────
Log "▶ [2/5] Waiting for PostgreSQL to be healthy..." Yellow
$pgReady = $false
for ($i = 0; $i -lt 30; $i++) {
    $status = docker inspect --format='{{.State.Health.Status}}' chorus-dev-postgres 2>$null
    if ($status -eq "healthy") { $pgReady = $true; break }
    if ($i % 5 -eq 0 -and $i -gt 0) { Write-Host "  ...still waiting ($i sec)" }
    Start-Sleep -Seconds 2
}
if (-not $pgReady) {
    Warn "PostgreSQL not yet healthy — backend may retry connection."
} else {
    Ok "PostgreSQL healthy"
}
Log ""

# ──────────────────────────────────────────────
# 3. Create .env for backend with dev ports
# ──────────────────────────────────────────────
Log "▶ [3/5] Setting up backend .env for dev Docker services..." Yellow

$EnvFile = Join-Path $BackendDir ".env"
$DevEnvContent = @'
ENVIRONMENT=development

# Dev Docker services (different ports from production defaults)
DATABASE_URL=postgres://chorus_dev:dev_password_123@localhost:5433/chorus_dev?sslmode=disable
REDIS_URL=localhost:6380

# JWT
JWT_SECRET=dev-jwt-secret-key-for-testing-only

# Translation — Phase 1 (LibreTranslate on dev port 5002)
TRANSLATOR_ENGINE_URL=http://localhost:5002
LIBRETRANSLATE_URL=http://localhost:5002

# Translation — Phase 2 (Ollama on dev port 11435)
OLLAMA_URL=http://localhost:11435
OLLAMA_MODEL=qwen2.5:3b

# Server
PORT=8080
'@

if (-not (Test-Path $EnvFile)) {
    $DevEnvContent | Set-Content -Path $EnvFile -Encoding UTF8
    Ok "Created backend\.env with dev Docker service ports"
} else {
    Ok "backend\.env already exists (using existing)"
}
Log ""

# ──────────────────────────────────────────────
# 4. Start Go backend with air (new window)
# ──────────────────────────────────────────────
Log "▶ [4/5] Starting Go backend with air (hot-reload)..." Yellow

$airInstalled = (Get-Command air -ErrorAction SilentlyContinue) -ne $null
if (-not $airInstalled) {
    Warn "'air' not found. Installing..."
    go install github.com/air-verse/air@latest
    if ($LASTEXITCODE -ne 0) {
        Warn "Failed to install 'air'. Starting backend with 'go run' instead."
        Start-Process powershell -ArgumentList @(
            "-NoExit",
            "-Command", "cd '$BackendDir'; Write-Host 'Backend starting with go run (no hot-reload)...' -ForegroundColor Cyan; go run ./cmd/server"
        ) -WindowStyle Normal
    } else {
        Start-Process powershell -ArgumentList @(
            "-NoExit",
            "-Command", "cd '$BackendDir'; Write-Host 'Backend starting with air (hot-reload)...' -ForegroundColor Cyan; air"
        ) -WindowStyle Normal
    }
} else {
    Start-Process powershell -ArgumentList @(
        "-NoExit",
        "-Command", "cd '$BackendDir'; Write-Host 'Backend starting with air (hot-reload)...' -ForegroundColor Cyan; air"
    ) -WindowStyle Normal
}
Ok "Backend starting in new window (hot-reload on port 8080)"
Log ""

# ──────────────────────────────────────────────
# 5. Start frontend with Vite HMR (new window)
# ──────────────────────────────────────────────
Log "▶ [5/5] Starting frontend with Vite HMR..." Yellow

Start-Process powershell -ArgumentList @(
    "-NoExit",
    "-Command", "cd '$FrontendDir'; if (-not (Test-Path node_modules)) { Write-Host 'Installing npm dependencies...' -ForegroundColor Yellow; npm install }; Write-Host 'Frontend starting with Vite HMR...' -ForegroundColor Cyan; npm run dev"
) -WindowStyle Normal

Ok "Frontend starting in new window (Vite HMR on port 3000)"
Log ""

# ──────────────────────────────────────────────
# 6. Summary
# ──────────────────────────────────────────────
Log "╔══════════════════════════════════════════════════════════╗" Cyan
Log "║   ✓ All services starting!                               ║" Cyan
Log "║                                                          ║" Cyan
Log "║   Frontend (HMR):   http://localhost:3000                ║" Cyan
Log "║   Backend  (API):   http://localhost:8080                ║" Cyan
Log "║   Backend  (health): http://localhost:8080/health       ║" Cyan
Log "║   PostgreSQL:       localhost:5433                       ║" Cyan
Log "║   Redis:            localhost:6380                       ║" Cyan
Log "║   LibreTranslate:   http://localhost:5002                ║" Cyan
Log "║   Ollama:           http://localhost:11435               ║" Cyan
Log "║                                                          ║" Cyan
Log "║   Close the backend/frontend windows or press Ctrl+C     ║" Cyan
Log "║   to stop. Run the following to stop Docker services:    ║" Cyan
Log "║     docker-compose -f docker-compose.dev.yml down       ║" Cyan
Log "╚══════════════════════════════════════════════════════════╝" Cyan
