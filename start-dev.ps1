<#
.SYNOPSIS
    Start Chorus development environment with hot-reload.

.DESCRIPTION
    Starts Docker infra (PostgreSQL, Redis), then runs the Go backend
    (with file-watch auto-restart) and React frontend (Vite HMR).

    Run from repo root:
        .\start-dev.ps1

    Open separate terminals for frontend/backend logs by using:
        .\start-dev.ps1 -SplitWindows
#>

param([switch]$SplitWindows)

$RootDir = $PSScriptRoot
$BackendDir = Join-Path $RootDir "backend"
$FrontendDir = Join-Path $RootDir "frontend"
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

docker compose -f $ComposeFile up -d --remove-orphans postgres redis 2>&1 | ForEach-Object {
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

# ──────────────────────────────────────────────
# 4. Run backend with file-watch auto-restart
# ──────────────────────────────────────────────
Log "▶ [4/5] Starting Go backend (file-watch enabled)..." Yellow

$env:ENVIRONMENT = "development"
$env:DATABASE_URL = "postgres://messenger:password@localhost:5432/messenger_dev?sslmode=disable"
$env:REDIS_URL = "localhost:6379"
$env:JWT_SECRET = "dev-jwt-secret-key-for-testing-only"
$env:PORT = "8080"
$env:TRANSLATION_PROVIDER_NAME = "opencode"

if ($SplitWindows) {
    Start-Process powershell -ArgumentList @(
        "-NoExit", "-Command", @"
            cd '$BackendDir'
            `$env:ENVIRONMENT = 'development'
            `$env:DATABASE_URL = 'postgres://messenger:password@localhost:5432/messenger_dev?sslmode=disable'
            `$env:REDIS_URL = 'localhost:6379'
            `$env:JWT_SECRET = 'dev-jwt-secret-key-for-testing-only'
            `$env:PORT = '8080'
            `$env:TRANSLATION_PROVIDER_NAME = 'opencode'
            Write-Host 'Backend starting...' -ForegroundColor Cyan
            # Watch for .go file changes and restart
            `$watcher = New-Object System.IO.FileSystemWatcher
            `$watcher.Path = '$BackendDir'
            `$watcher.Filter = '*.go'
            `$watcher.IncludeSubdirectories = `$true
            `$watcher.EnableRaisingEvents = `$true
            `$lastRestart = Get-Date
            `$proc = $null
            function Start-Backend {
                if (`$proc -and -not `$proc.HasExited) { `$proc.Kill() }
                Start-Sleep 1
                `$script:proc = Start-Process -FilePath 'go' -ArgumentList 'run','./cmd/server' -NoNewWindow -PassThru -RedirectStandardOutput "stdout.txt" -RedirectStandardError "stderr.txt"
            }
            Start-Backend
            Register-ObjectEvent `$watcher "Changed" -Action {
                if (((Get-Date) - `$script:lastRestart).TotalSeconds -gt 2) {
                    Write-Host "`n  File changed. Restarting backend..." -ForegroundColor Yellow
                    `$script:lastRestart = Get-Date
                    Start-Backend
                }
            } | Out-Null
            # Tail output
            while (`$true) {
                if (Test-Path "stdout.txt") { Get-Content "stdout.txt" -Tail 0 -Wait }
                Start-Sleep 0.5
            }
"@
    ) -WindowStyle Normal
} else {
    Log "  Starting backend in this terminal. Files in backend/ will auto-restart." Yellow
    Log "  --- backend output ---" Cyan

    $watcher = New-Object System.IO.FileSystemWatcher
    $watcher.Path = $BackendDir
    $watcher.Filter = '*.go'
    $watcher.IncludeSubdirectories = $true
    $watcher.EnableRaisingEvents = $true

    $global:lastRestart = Get-Date
    $global:backendProc = $null

    function Start-Backend {
        if ($global:backendProc -and -not $global:backendProc.HasExited) {
            try { $global:backendProc.Kill() } catch {}
            Start-Sleep 1
        }
        Push-Location $BackendDir
        $global:backendProc = Start-Process -FilePath "go" -ArgumentList "run","./cmd/server" -NoNewWindow -PassThru -RedirectStandardOutput "$BackendDir\.dev-stdout.txt" -RedirectStandardError "$BackendDir\.dev-stderr.txt"
        Pop-Location
    }

    Start-Backend

    Register-ObjectEvent $watcher "Changed" -Action {
        if (((Get-Date) - $global:lastRestart).TotalSeconds -gt 2) {
            Write-Host "`n  File changed. Restarting backend..." -ForegroundColor Yellow
            $global:lastRestart = Get-Date
            Start-Backend
        }
    } | Out-Null

    # Start frontend BEFORE entering tail loop
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

    # Tail backend output
    $stdoutFile = Join-Path $BackendDir ".dev-stdout.txt"
    $stderrFile = Join-Path $BackendDir ".dev-stderr.txt"
    try {
        while ($true) {
            if (Test-Path $stdoutFile) {
                Get-Content $stdoutFile -Tail 0 -Wait -ErrorAction SilentlyContinue
            }
            if (Test-Path $stderrFile) {
                Get-Content $stderrFile -Tail 0 -Wait -ErrorAction SilentlyContinue
            }
            Start-Sleep 0.5
        }
    } finally {
        if ($global:backendProc -and -not $global:backendProc.HasExited) {
            $global:backendProc.Kill()
        }
    }
}
Log ""

# ──────────────────────────────────────────────
# 5. Start frontend with Vite HMR (SplitWindows)
# ──────────────────────────────────────────────
if ($SplitWindows) {
    Log "▶ [5/5] Starting frontend with Vite HMR..." Yellow

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
