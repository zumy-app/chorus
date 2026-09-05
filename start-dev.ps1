<#
.SYNOPSIS
    Start Chorus development environment with hot-reload.

.DESCRIPTION
    Starts Docker infra (PostgreSQL, Redis), the Go backend (air hot-reload),
    and the React frontend (Vite HMR).  Use -Mobile to also set up the
    Android emulator, Metro bundler, and install the app on the AVD.

    Run from repo root:
        .\start-dev.ps1              # web + backend
        .\start-dev.ps1 -Mobile      # web + backend + Android emulator

    Open separate terminals for frontend/backend logs by using:
        .\start-dev.ps1 -SplitWindows
#>

param(
    [switch]$SplitWindows,
    [switch]$Mobile
)

$RootDir = $PSScriptRoot
$BackendDir = Join-Path $RootDir "backend"
$FrontendDir = Join-Path $RootDir "frontend"
$MobileDir = Join-Path $RootDir "mobile"
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
if ($Mobile) { Log "║   Mode: Web + Backend + Mobile (Android)       ║" Cyan }
else         { Log "║   Mode: Web + Backend                          ║" Cyan }
Log "╚════════════════════════════════════════════════╝" Cyan
Log ""

# ──────────────────────────────────────────────
# 1. Ensure Docker is running
# ──────────────────────────────────────────────
Log "▶ [1/7] Checking Docker..." Yellow
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
Log "▶ [2/7] Starting Docker services (PostgreSQL, Redis)..." Yellow

docker compose -f $ComposeFile up -d --remove-orphans --force-recreate  postgres redis  2>&1 | ForEach-Object {
    $line = $_.ToString().Trim()
    if ($line -ne "") { Write-Host "  $line" }
}
if ($LASTEXITCODE -eq 0) { Ok "Docker services started" } else { Warn "Some services may have issues" }
Log ""

# ──────────────────────────────────────────────
# 3. Wait for PostgreSQL & Redis healthy
# ──────────────────────────────────────────────
Log "▶ [3/7] Waiting for services to be healthy..." Yellow
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
Log ""

# ──────────────────────────────────────────────
# 4. Run backend with air (hot-reload)
# ──────────────────────────────────────────────
Log "▶ [4/7] Starting Go backend (air hot-reload)..." Yellow

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
    Log "▶ [5/7] Starting frontend with Vite HMR..." Yellow

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
# 6. Mobile: Android emulator + Metro + app
# ──────────────────────────────────────────────
if ($Mobile) {
    Log "▶ [6/7] Setting up Android mobile dev..." Yellow

    # --- 6a. Check Android SDK ---
    $androidHome = $env:ANDROID_HOME
    if (-not $androidHome) {
        $androidHome = "$env:LOCALAPPDATA\Android\Sdk"
        if (-not (Test-Path $androidHome)) {
            $androidHome = "$env:USERPROFILE\AppData\Local\Android\Sdk"
        }
    }

    if (-not $androidHome -or -not (Test-Path $androidHome)) {
        Fail "Android SDK not found. Set ANDROID_HOME or install Android Studio."
        Log "  See ANDROID_SETUP.md for instructions." Yellow
    } else {
        Ok "Android SDK: $androidHome"
        $env:ANDROID_HOME = $androidHome

        # Ensure platform-tools and emulator are on PATH
        $platformTools = Join-Path $androidHome "platform-tools"
        $emulatorDir = Join-Path $androidHome "emulator"
        if ($env:PATH -notlike "*$platformTools*") { $env:PATH = "$platformTools;$env:PATH" }
        if ($env:PATH -notlike "*$emulatorDir*") { $env:PATH = "$emulatorDir;$env:PATH" }

        # --- 6b. Check for running emulator ---
        $devices = adb devices 2>$null | Select-String "device$" | Measure-Object
        if ($devices.Count -gt 0) {
            Ok "Android device/emulator connected"
        } else {
            # --- 6c. Start emulator ---
            Log "  No emulator running. Starting AVD..." Yellow
            $avds = emulator -list-avds 2>$null
            if ($avds -and $avds.Count -gt 0) {
                $avdName = $avds[0].ToString().Trim()
                Log "  Using AVD: $avdName" Yellow
                Start-Process emulator -ArgumentList "-avd", $avdName -WindowStyle Normal
                Log "  Waiting for emulator to boot (may take 30-60s)..." Yellow
                $bootOk = $false
                for ($i = 0; $i -lt 60; $i++) {
                    $bootAnim = adb shell getprop sys.boot_completed 2>$null
                    if ($bootAnim -and $bootAnim.Trim() -eq "1") { $bootOk = $true; break }
                    Start-Sleep -Seconds 2
                    if ($i -gt 0 -and $i % 10 -eq 0) { Write-Host "  ...still booting ($($i*2)s)" }
                }
                if ($bootOk) { Ok "Emulator booted" } else { Warn "Emulator may still be booting" }
            } else {
                Fail "No AVDs found. Create one in Android Studio > Device Manager."
                Log "  See ANDROID_SETUP.md for instructions." Yellow
            }
        }

        # --- 6d. Install npm deps ---
        Log "  Checking mobile npm dependencies..." Yellow
        Push-Location $MobileDir
        if (-not (Test-Path "node_modules")) {
            Log "  Installing npm dependencies..." Yellow
            npm install 2>&1 | ForEach-Object { Write-Host "  $_" }
            if ($LASTEXITCODE -ne 0) { Fail "npm install failed in mobile/"; Pop-Location; exit 1 }
            Ok "npm dependencies installed"
        } else {
            Ok "npm dependencies already installed"
        }
        Pop-Location

        # --- 6e. Start Metro bundler ---
        Log "  Starting Metro bundler..." Yellow
        $metroRunning = netstat -ano 2>$null | Select-String "LISTENING" | Select-String ":8081\s"
        if ($metroRunning) {
            Ok "Metro already running on :8081"
        } else {
            $metroScript = @"
Set-Location '$MobileDir'
npx react-native start --reset-cache
"@
            $metroScriptPath = Join-Path $env:TEMP "start-metro.ps1"
            Set-Content -Path $metroScriptPath -Value $metroScript -Force
            $metroProc = Start-Process powershell -ArgumentList @("-NoExit", "-File", $metroScriptPath) -WindowStyle Normal -PassThru
            Ok "Metro bundler starting in new window (PID $($metroProc.Id))"
            # Wait for Metro to be ready
            Log "  Waiting for Metro to be ready..." Yellow
            for ($i = 0; $i -lt 30; $i++) {
                try {
                    $r = Invoke-WebRequest -Uri "http://localhost:8081/status" -TimeoutSec 2 -ErrorAction SilentlyContinue
                    if ($r.StatusCode -eq 200) { Ok "Metro ready"; break }
                } catch {}
                Start-Sleep -Seconds 2
                if ($i -gt 0 -and $i % 5 -eq 0) { Write-Host "  ...waiting ($($i*2)s)" }
            }
        }

        # --- 6f. Build & install app on emulator ---
        Log "  Building and installing app on emulator..." Yellow
        Push-Location $MobileDir
        npx react-native run-android --no-packager 2>&1 | ForEach-Object { Write-Host "  $_" }
        if ($LASTEXITCODE -eq 0) {
            Ok "App installed and launched on emulator"
        } else {
            Warn "App build/install had issues — check output above"
        }
        Pop-Location
    }
    Log ""
} else {
    Log "▶ [6/7] Skipping mobile (use -Mobile to enable)" DarkGray
    Log ""
}

# ──────────────────────────────────────────────
# 7. Summary
# ──────────────────────────────────────────────
Log "▶ [7/7] Done!" Yellow
Log ""
Log "╔════════════════════════════════════════════════╗" Cyan
Log "║   ✓ All services starting!                     ║" Cyan
Log "║                                                ║" Cyan
Log "║   Frontend (HMR):  http://localhost:3000       ║" Cyan
Log "║   Backend  (API):  http://localhost:8080       ║" Cyan
Log "║   Health check:    http://localhost:8080/health║" Cyan
Log "║   PostgreSQL:      localhost:5432              ║" Cyan
Log "║   Redis:           localhost:6379              ║" Cyan
if ($Mobile) {
Log "║   Android app:     Running on emulator         ║" Cyan
Log "║   Metro bundler:   http://localhost:8081       ║" Cyan
}
Log "║                                                ║" Cyan
Log "║   Stop infra:  docker compose down             ║" Cyan
Log "║   Stop all:    docker compose down -v          ║" Cyan
Log "╚════════════════════════════════════════════════╝" Cyan
