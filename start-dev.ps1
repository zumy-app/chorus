<#
.SYNOPSIS
    Start Chorus development environment with hot-reload.

.DESCRIPTION
    Starts Docker infra (PostgreSQL, Redis), then runs the Go backend
    (with air hot-reload) and React frontend (Vite HMR).

    Run from repo root:
        .\start-dev.ps1

    Open separate terminals for frontend/backend logs by using:
        .\start-dev.ps1 -SplitWindows
#>

param([switch]$SplitWindows)

$RootDir = $PSScriptRoot
$BackendDir = Join-Path $RootDir "backend"
$FrontendDir = Join-Path $RootDir "frontend"

# ──────────────────────────────────────────────
# Clean stale provider env vars from previous runs
# (godotenv.Load does NOT override existing env vars)
# ──────────────────────────────────────────────
Get-ChildItem Env:PROVIDER_*, Env:TRANSLATION_PROVIDER_ORDER, Env:GRAMMAR_PROVIDER_ORDER -ErrorAction SilentlyContinue |
    ForEach-Object { Remove-Item "Env:$($_.Name)" -ErrorAction SilentlyContinue }
$ComposeFile = Join-Path $RootDir "docker-compose.yml"

function Log($Msg, $Color = "White") { Write-Host "$Msg" -ForegroundColor $Color }
function Ok($Msg)  { Write-Host "  ✓ $Msg" -ForegroundColor Green }
function Warn($Msg) { Write-Host "  ⚠ $Msg" -ForegroundColor Yellow }
function Fail($Msg) { Write-Host "  ✘ $Msg" -ForegroundColor Red }

# ──────────────────────────────────────────────
# 0. Header
# ──────────────────────────────────────────────
Log "╔════════════════════════════════════════════════╗" Cyan
Log "║   Chorus Dev Environment                       ║" Cyan
Log "╚════════════════════════════════════════════════╝" Cyan
Log ""

# ──────────────────────────────────────────────
# 1. Ensure Docker is running
# ──────────────────────────────────────────────
Log "▶ [1/5] Checking Docker..." Yellow
$dockerOk = $false
for ($attempt = 0; $attempt -lt 20; $attempt++) {
    $null = docker ps 2>$null
    if ($LASTEXITCODE -eq 0) { $dockerOk = $true; break }
    if ($attempt -eq 0) {
        Warn "Docker CLI not responding. Starting Docker Desktop..."
        $dp = "C:\Program Files\Docker\Docker\Docker Desktop.exe"
        if (Test-Path $dp) { Start-Process -FilePath $dp -WindowStyle Hidden }
        Log "  Waiting up to 60s..." Yellow
    }
    Start-Sleep -Seconds 3
}
if (-not $dockerOk) { Fail "Docker Desktop not running."; exit 1 }
Ok "Docker Desktop ready"
Log ""

# ──────────────────────────────────────────────
# 2. Start Docker services (infra only)
# ──────────────────────────────────────────────
Log "▶ [2/5] Starting Docker services (PostgreSQL, Redis)..." Yellow

docker compose -f $ComposeFile up -d --remove-orphans --force-recreate ollama postgres redis 2>&1 | ForEach-Object {
    $line = $_.ToString().Trim()
    if ($line -ne "") { Write-Host "  $line" }
}
if ($LASTEXITCODE -eq 0) { Ok "Docker services started" } else { Warn "Some services may have issues" }
Log ""

# ──────────────────────────────────────────────
# 3. Wait for PostgreSQL & Redis healthy
# ──────────────────────────────────────────────
Log "▶ [3/5] Waiting for services to be healthy..." Yellow
$pgReady = $false
for ($i = 0; $i -lt 30; $i++) {
    $status = docker inspect --format='{{.State.Health.Status}}' chorus-postgres 2>$null
    if ($status -eq "healthy") { $pgReady = $true; break }
    if ($i -gt 0 -and $i % 10 -eq 0) { Write-Host "  ...waiting for PostgreSQL ($($i*2)s)" }
    Start-Sleep -Seconds 2
}
if ($pgReady) { Ok "PostgreSQL healthy" } else { Warn "PostgreSQL not healthy yet" }

$redisReady = $false
for ($i = 0; $i -lt 15; $i++) {
    $status = docker inspect --format='{{.State.Health.Status}}' chorus-redis 2>$null
    if ($status -eq "healthy") { $redisReady = $true; break }
    Start-Sleep -Seconds 2
}
if ($redisReady) { Ok "Redis healthy" } else { Warn "Redis not healthy yet" }

# Wait for Ollama to be healthy (docker healthcheck), then ensure the model exists
Log "  Waiting for Ollama to be healthy..." Yellow
$ollamaReady = $false
for ($i = 0; $i -lt 30; $i++) {
    $status = docker inspect --format='{{.State.Health.Status}}' chorus-ollama 2>$null
    if ($status -eq "healthy") { $ollamaReady = $true; break }
    if ($i -gt 0 -and $i % 10 -eq 0) { Write-Host "  ...waiting for Ollama ($($i*2)s)" }
    Start-Sleep -Seconds 2
}
if ($ollamaReady) {
    # Check if the model exists, if not pull it
    $modelList = docker exec chorus-ollama ollama list 2>$null | Out-String
    if ($modelList -notmatch "llama3.2:3b-instruct-q4_K_M") {
        Warn "Model not found. Pulling llama3.2:3b-instruct-q4_K_M (may take 2-5 min)..."
        docker exec chorus-ollama ollama pull llama3.2:3b-instruct-q4_K_M
        if ($LASTEXITCODE -eq 0) {
            Ok "Model ready (llama3.2:3b-instruct-q4_K_M)"
        } else {
            Warn "Failed to pull model. Check if Ollama has internet access."
        }
    } else {
        Ok "Ollama healthy (llama3.2:3b-instruct-q4_K_M ready)"
    }
} else {
    Warn "Ollama not ready yet - backend will fall through to other providers"
}

# ──────────────────────────────────────────────
# 4. Run backend with air (hot-reload)
# ──────────────────────────────────────────────
Log "▶ [4/5] Starting Go backend (air hot-reload)..." Yellow

# Kill any existing processes on port 8080
$portPid = netstat -ano | Select-String ":8080\s" | ForEach-Object { $_.ToString().Trim().Split()[-1] } | Select-Object -First 1
if ($portPid -and $portPid -match '^\d+$') {
    Warn "Killing existing process (PID $portPid) on port 8080..."
    Stop-Process -Id $portPid -Force -ErrorAction SilentlyContinue
    Start-Sleep -Seconds 3
}

# Also kill any lingering main.exe or air processes
Get-Process "main", "air" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 1

$env:ENVIRONMENT = "development"
$env:DATABASE_URL = "postgres://messenger:password@localhost:5432/messenger_dev?sslmode=disable"
$env:REDIS_URL = "localhost:6379"
$env:JWT_SECRET = "dev-jwt-secret-key-for-testing-only"
$env:PORT = "8080"

# Check if air is installed, install if not
$airInstalled = Get-Command air -ErrorAction SilentlyContinue
if (-not $airInstalled) {
    Log "  Installing air (Go hot-reload tool)..." Yellow
    go install github.com/air-verse/air@latest
    if ($LASTEXITCODE -ne 0) {
        Fail "Failed to install air. Please install manually: go install github.com/air-verse/air@latest"
        exit 1
    }
    Ok "air installed"
}

# Clean build first to ensure no stale binaries
Log "  Running clean build..." Yellow
Push-Location $BackendDir
go build -o ./tmp/main.exe ./cmd/server 2>&1 | ForEach-Object { Write-Host "  $_" }
if ($LASTEXITCODE -ne 0) {
    Fail "Build failed. Fix errors before starting."
    Pop-Location
    exit 1
}
Ok "Build successful"
Pop-Location

if ($SplitWindows) {
    Start-Process powershell -ArgumentList @(
        "-NoExit", "-Command", @"
            cd '$BackendDir'
            `$env:ENVIRONMENT = 'development'
            `$env:DATABASE_URL = 'postgres://messenger:password@localhost:5432/messenger_dev?sslmode=disable'
            `$env:REDIS_URL = 'localhost:6379'
            `$env:JWT_SECRET = 'dev-jwt-secret-key-for-testing-only'
            `$env:PORT = '8080'
            Write-Host 'Backend starting with air (hot-reload)...' -ForegroundColor Cyan
            air -c .air.toml
"@
    ) -WindowStyle Normal
} else {
    Log "  Starting backend with air (hot-reload on .go file changes)..." Yellow

    # Start frontend in a new window (unless already running on :3000)
    $frontendRunning = netstat -ano 2>$null | Select-String "LISTENING" | Select-String ":3000\s"
    if ($frontendRunning) {
        Log "  Frontend already running on http://localhost:3000, skipping" Yellow
    } else {
        $frontendScript = @"
Set-Location '$FrontendDir'
if (-not (Test-Path node_modules)) { npm install }
Write-Host 'Frontend (Vite HMR) starting...' -ForegroundColor Cyan
npm run dev
"@
        $frontendScriptPath = Join-Path $env:TEMP "start-frontend.ps1"
        Set-Content -Path $frontendScriptPath -Value $frontendScript -Force
        $frontendProc = Start-Process powershell -ArgumentList @("-NoExit", "-File", $frontendScriptPath) -WindowStyle Normal -PassThru
        $frontendPid = $frontendProc.Id
        $parentPid = $PID
        Ok "Frontend starting in new window (PID $frontendPid, Vite HMR on http://localhost:3000)"
        Log ""

        # Launch a hidden watcher that kills the frontend when this script exits
        $watcherScript = @"
`$parentPid = $parentPid
`$frontendPid = $frontendPid
while (Get-Process -Id `$parentPid -ErrorAction SilentlyContinue) {
    Start-Sleep -Seconds 2
}
Start-Sleep -Seconds 1
Stop-Process -Id `$frontendPid -Force -ErrorAction SilentlyContinue
"@
        $watcherPath = Join-Path $env:TEMP "watch-frontend.ps1"
        Set-Content -Path $watcherPath -Value $watcherScript -Force
        Start-Process powershell -ArgumentList @("-WindowStyle", "Hidden", "-File", $watcherPath) -WindowStyle Hidden
    }

    # Run air in the current terminal
    Push-Location $BackendDir
    try {
        air -c .air.toml
    } finally {
        Pop-Location
    }
}
Log ""

# ──────────────────────────────────────────────
# 5. Start frontend with Vite HMR (SplitWindows)
# ──────────────────────────────────────────────
if ($SplitWindows) {
    Log "▶ [5/5] Starting frontend with Vite HMR..." Yellow

    $frontendRunning = netstat -ano 2>$null | Select-String "LISTENING" | Select-String ":3000\s"
    if ($frontendRunning) {
        Log "  Frontend already running on http://localhost:3000, skipping" Yellow
    } else {
        $frontendScript = @"
Set-Location '$FrontendDir'
if (-not (Test-Path node_modules)) { npm install }
Write-Host 'Frontend (Vite HMR) starting...' -ForegroundColor Cyan
npm run dev
"@
        $scriptPath = Join-Path $env:TEMP "start-frontend.ps1"
        Set-Content -Path $scriptPath -Value $frontendScript -Force
        Start-Process powershell -ArgumentList @("-NoExit", "-File", $scriptPath) -WindowStyle Normal
        Ok "Frontend starting in new window (Vite HMR on http://localhost:3000)"
        Log ""
    }
}

# ──────────────────────────────────────────────
# Summary
# ──────────────────────────────────────────────
Log "╔════════════════════════════════════════════════╗" Cyan
Log "║   ✓ All services starting!                     ║" Cyan
Log "║                                                ║" Cyan
Log "║   Frontend (HMR):  http://localhost:3000       ║" Cyan
Log "║   Backend  (API):  http://localhost:8080       ║" Cyan
Log "║   Health check:    http://localhost:8080/health║" Cyan
Log "║   PostgreSQL:      localhost:5432              ║" Cyan
Log "║   Redis:           localhost:6379              ║" Cyan
Log "║                                                ║" Cyan
Log "║   Stop infra:  docker compose down             ║" Cyan
Log "║   Stop all:    docker compose down -v          ║" Cyan
Log "╚════════════════════════════════════════════════╝" Cyan
