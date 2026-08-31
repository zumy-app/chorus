<#
.SYNOPSIS
    Start Chorus Android development environment.

.DESCRIPTION
    Sets up Android SDK/AVD, starts emulator, starts Docker infra
    (PostgreSQL, Redis), starts Go backend with hot-reload, and runs
    Expo for Android. Skips steps that are already satisfied.

    Run from repo root:
        .\start-android.ps1

    Environment variables (optional):
        ANDROID_SDK_ROOT  - path to Android SDK (default: %LOCALAPPDATA%\Android\Sdk)
        AVD_NAME          - AVD name (default: Pixel_6a_API_35)
        SYSTEM_IMAGE      - system image (default: system-images;android-35;google_apis_x86_64)
#>

param(
    [string]$AvdName,
    [string]$SystemImage
)

$ErrorActionPreference = "Stop"
$RootDir = $PSScriptRoot

# ---- Helpers ----
function Log($Msg, $Color = "White") { Write-Host $Msg -ForegroundColor $Color }
function Ok($Msg)  { Write-Host "  OK: $Msg" -ForegroundColor Green }
function Warn($Msg) { Write-Host "  WARN: $Msg" -ForegroundColor Yellow }
function Fail($Msg) { Write-Host "  FAIL: $Msg" -ForegroundColor Red }

Log "============================================" Cyan
Log "  Chorus Android Dev Environment" Cyan
Log "============================================" Cyan
Log ""

# ──────────────────────────────────────────────
# 1. Android SDK / platform-tools
# ──────────────────────────────────────────────
Log "[1/8] Checking Android SDK..." Yellow

if ($env:ANDROID_SDK_ROOT) {
    $SdkRoot = $env:ANDROID_SDK_ROOT
} else {
    $SdkRoot = Join-Path $env:LOCALAPPDATA "Android\Sdk"
}

if (-not (Test-Path $SdkRoot)) {
    Fail "Android SDK not found at $SdkRoot"
    Fail "Install Android Studio or set ANDROID_SDK_ROOT environment variable."
    exit 1
}

$platformTools = Join-Path $SdkRoot "platform-tools"
if (-not (Test-Path $platformTools)) {
    Fail "platform-tools not found at $platformTools"
    Fail "Open Android Studio -> SDK Manager -> install Platform-Tools."
    exit 1
}

# Add platform-tools and emulator to PATH for this session
$userPath = [environment]::GetEnvironmentVariable("PATH", "User")
$needsPathUpdate = $false
if (-not $userPath.Split(';').Contains($platformTools)) {
    [environment]::SetEnvironmentVariable("PATH", "$userPath;$platformTools", "User")
    $env:PATH = "$env:PATH;$platformTools"
    $needsPathUpdate = $true
}

$emulatorDir = Join-Path $SdkRoot "emulator"
if (-not (Test-Path $emulatorDir)) {
    $emulatorDir = $null
} elseif (-not $userPath.Split(';').Contains($emulatorDir)) {
    [environment]::SetEnvironmentVariable("PATH", "$userPath;$platformTools;$emulatorDir", "User")
    $env:PATH = "$env:PATH;$emulatorDir"
}

# Verify adb is available
$adbExe = Join-Path $platformTools "adb.exe"
if (-not (Get-Command adb -ErrorAction SilentlyContinue)) {
    if (Test-Path $adbExe) {
        Set-Alias -Name adb -Value $adbExe -Force
    } else {
        Fail "adb.exe not found at $adbExe"
        exit 1
    }
}

# Verify emulator is available
$emulatorExe = Join-Path $SdkRoot "emulator\emulator.exe"
if (-not (Test-Path $emulatorExe)) {
    Fail "emulator.exe not found at $emulatorExe"
    Fail "Open Android Studio -> SDK Manager -> install Android Emulator."
    exit 1
}

Ok "Android SDK ready at $SdkRoot"

# Ensure Gradle can find the Android SDK regardless of local.properties.
$env:ANDROID_HOME = $SdkRoot
$env:ANDROID_SDK_ROOT = $SdkRoot
Log ""

# ──────────────────────────────────────────────
# 2. System image & AVD
# ──────────────────────────────────────────────
Log "[2/8] Checking AVD..." Yellow

if (-not $AvdName) {
    if ($env:AVD_NAME) { $AvdName = $env:AVD_NAME } else { $AvdName = "Pixel_6a_API_35" }
}

# Find avdmanager
$avdmanagerExe = $null
$candidatePaths = @(
    (Join-Path $SdkRoot "cmdline-tools\latest\bin\avdmanager.bat"),
    (Join-Path $SdkRoot "cmdline-tools\latest\bin\avdmanager.exe"),
    (Join-Path $SdkRoot "tools\bin\avdmanager.bat"),
    (Join-Path $SdkRoot "tools\bin\avdmanager.exe")
)
foreach ($p in $candidatePaths) {
    if (Test-Path $p) { $avdmanagerExe = $p; break }
}
if (-not $avdmanagerExe) {
    $found = Get-ChildItem -Path $SdkRoot -Recurse -Filter "avdmanager*" -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($found) { $avdmanagerExe = $found.FullName }
}
if ($avdmanagerExe -and -not (Get-Command avdmanager -ErrorAction SilentlyContinue)) {
    Set-Alias -Name avdmanager -Value $avdmanagerExe -Force
}

# Find sdkmanager
$sdkmanagerExe = $null
foreach ($p in @(
    (Join-Path $SdkRoot "cmdline-tools\latest\bin\sdkmanager.bat"),
    (Join-Path $SdkRoot "cmdline-tools\latest\bin\sdkmanager.exe"),
    (Join-Path $SdkRoot "tools\bin\sdkmanager.bat"),
    (Join-Path $SdkRoot "tools\bin\sdkmanager.exe")
)) {
    if (Test-Path $p) { $sdkmanagerExe = $p; break }
}
if (Get-Command sdkmanager -ErrorAction SilentlyContinue) { $sdkmanagerExe = (Get-Command sdkmanager).Source }

# List existing AVDs
$avds = @()
try {
    $avdListOutput = & emulator -list-avds 2>$null
    if ($avdListOutput) { $avds = @($avdListOutput | ForEach-Object { $_.Trim() } | Where-Object { $_ -ne "" }) }
} catch { }

# Auto-detect an installed x86_64 system image (google_apis preferred)
if (-not $SystemImage) {
    if ($env:SYSTEM_IMAGE) { $SystemImage = $env:SYSTEM_IMAGE }
}
$detectedImage = $null
if (-not $SystemImage -and $sdkmanagerExe) {
    try {
        $inst = & $sdkmanagerExe --list_installed 2>$null
        $cands = $inst |
            Where-Object { $_ -match '^\s*system-images;' -and $_ -match 'x86_64' } |
            Where-Object { $_ -notmatch 'arm64' } |
            ForEach-Object { ($_ -split '\|')[0].Trim() } |
            Where-Object { $_ -ne "" }
        $ga = @($cands | Where-Object { $_ -match 'google_apis' -and $_ -notmatch 'playstore' })
        $play = @($cands | Where-Object { $_ -match 'playstore' })
        $pref = if ($ga.Count -gt 0) { $ga } else { $play }
        if ($pref.Count -gt 0) { $detectedImage = $pref | Select-Object -Last 1 }
    } catch { }
    if ($detectedImage) {
        $SystemImage = $detectedImage
        Log "  Using installed system image: $SystemImage" Gray
    }
}
if (-not $SystemImage) {
    $SystemImage = "system-images;android-36.1;google_apis_playstore;x86_64"
    Log "  No system image detected, defaulting to $SystemImage" Gray
}

# Determine the AVD to boot
$useAvd = $AvdName
$needCreate = $avds -notcontains $AvdName

if ($needCreate) {
    Warn "AVD '$AvdName' not found - creating..."
    if ($sdkmanagerExe) {
        Log "  Installing system image (this may take a while)..." Yellow
        try {
            $imgOut = & $sdkmanagerExe --install $SystemImage 2>&1
            $imgOut | ForEach-Object { Write-Host "    $_" }
        } catch {
            Warn "sdkmanager reported: $($_.Exception.Message)"
        }
    }
    if (Get-Command avdmanager -ErrorAction SilentlyContinue) {
        try {
            $createOut = 'no' | & avdmanager create avd -n $AvdName -k $SystemImage --force 2>&1
            $createOut | ForEach-Object { Write-Host "    $_" }
        } catch {
            Warn "avdmanager reported: $($_.Exception.Message)"
        }
    } else {
        Warn "avdmanager not found - will try to use an existing AVD"
    }
    # Verify the AVD now exists
    $updatedAvds = @( & emulator -list-avds 2>$null | ForEach-Object { $_.Trim() } | Where-Object { $_ -ne "" } )
    if ($updatedAvds -contains $AvdName) {
        Ok "AVD '$AvdName' created"
        $needCreate = $false
    } else {
        # Fall back to an existing (preferably non-ARM) AVD
        $candidate = $updatedAvds | Where-Object { $_ -notmatch 'arm64' } | Select-Object -First 1
        if (-not $candidate) { $candidate = $updatedAvds | Select-Object -First 1 }
        if ($candidate) {
            Warn "Could not create '$AvdName'. Using existing AVD '$candidate' instead."
            $useAvd = $candidate
            $needCreate = $false
        } else {
            Fail "No AVD is available. Create one in Android Studio (Device Manager) and re-run."
            exit 1
        }
    }
}

if (-not $needCreate -and $useAvd -eq $AvdName) {
    Ok "AVD '$AvdName' ready"
} elseif (-not $needCreate) {
    Ok "AVD '$useAvd' ready"
}
Log ""

# ──────────────────────────────────────────────
# 3. Start emulator if not running
# ──────────────────────────────────────────────
Log "[3/8] Checking emulator..." Yellow

$deviceState = ""
try { $deviceState = & adb get-state 2>$null } catch { }

if ($deviceState -ne "device") {
    Log "  Starting emulator for '$useAvd'..." Magenta

    # Run WITH a visible window (drop -no-window) so the app is actually visible;
    # use software GPU (swiftshader) for host compatibility.
    $emulatorArgs = "-avd $useAvd -gpu swiftshader_indirect -no-audio -no-snapshot"
    Start-Process -FilePath $emulatorExe -ArgumentList $emulatorArgs -PassThru | Out-Null

    # Wait for emulator to appear in adb devices first (up to 3 minutes)
    Log "  Waiting for emulator to connect to adb (up to 3 minutes)..." Yellow
    $appearTimeout = (Get-Date).AddMinutes(3)
    while ((Get-Date) -lt $appearTimeout) {
        $prevEAP5 = $ErrorActionPreference
        $ErrorActionPreference = "SilentlyContinue"
        $deviceState = & adb get-state 2>$null
        $ErrorActionPreference = $prevEAP5
        if ($deviceState -eq "device") { break }
        Start-Sleep -Seconds 3
    }

    if ($deviceState -eq "device") {
        Ok "Emulator connected to adb"
        # Now wait for the OS to fully boot
        Log "  Waiting for Android OS to boot (up to 5 minutes)..." Yellow
        $timeout = (Get-Date).AddMinutes(5)
        $booted = $false
        while ((Get-Date) -lt $timeout) {
            try {
                $prevEAP6 = $ErrorActionPreference
                $ErrorActionPreference = "SilentlyContinue"
                $boot = & adb shell getprop sys.boot_completed 2>$null
                $ErrorActionPreference = $prevEAP6
                if ($boot -eq "1") { $booted = $true; break }
            } catch { }
            Start-Sleep -Seconds 3
        }

        if ($booted) {
            Ok "Emulator fully booted"
        } else {
            Warn "Emulator boot timeout - backend and Expo will start, press 'a' in Expo once device is ready"
        }
    } else {
        Warn "Emulator did not connect to adb - backend and Expo will start, connect device manually"
    }
} else {
    Ok "Emulator already running"
}
Log ""

# ──────────────────────────────────────────────
# 4. Ensure Docker is running
# ──────────────────────────────────────────────
Log "[4/8] Checking Docker..." Yellow
$dockerOk = $false
for ($attempt = 0; $attempt -lt 20; $attempt++) {
    $prevEAP4 = $ErrorActionPreference
    $ErrorActionPreference = "SilentlyContinue"
    $null = docker ps 2>$null
    $ErrorActionPreference = $prevEAP4
    if ($LASTEXITCODE -eq 0) { $dockerOk = $true; break }
    if ($attempt -eq 0) {
        Warn "Docker CLI not responding. Starting Docker Desktop..."
        $dp = "C:\Program Files\Docker\Docker\Docker Desktop.exe"
        if (Test-Path $dp) { Start-Process -FilePath $dp -WindowStyle Hidden }
        Log "  Waiting up to 60s..." Yellow
    }
    Start-Sleep -Seconds 3
}
if (-not $dockerOk) {
    Fail "Docker Desktop not running. Start Docker Desktop and retry."
    exit 1
}
Ok "Docker Desktop ready"
Log ""

# ──────────────────────────────────────────────
# 5. Start Docker services (PostgreSQL, Redis)
# ──────────────────────────────────────────────
Log "[5/8] Starting Docker services..." Yellow

$ComposeFile = Join-Path $RootDir "docker-compose.yml"
if (Test-Path $ComposeFile) {
    # docker compose writes status to stderr even on success; suppress stderr to avoid false errors
    $prevEAP = $ErrorActionPreference
    $ErrorActionPreference = "SilentlyContinue"
    $dcOut = docker compose -f $ComposeFile up -d --remove-orphans postgres redis 2>&1
    $ErrorActionPreference = $prevEAP
    if ($dcOut) {
        foreach ($line in $dcOut) {
            $txt = $line.ToString().Trim()
            if ($txt -ne "") { Write-Host "  $txt" }
        }
    }
    Ok "PostgreSQL and Redis starting"
} else {
    Warn "docker-compose.yml not found - skipping Docker services"
}
Log ""

# ──────────────────────────────────────────────
# 6. Wait for PostgreSQL & Redis to be healthy
# ──────────────────────────────────────────────
Log "[6/8] Waiting for services..." Yellow

$pgReady = $false
for ($i = 0; $i -lt 30; $i++) {
    $prevEAP2 = $ErrorActionPreference
    $ErrorActionPreference = "SilentlyContinue"
    $status = docker inspect --format='{{.State.Health.Status}}' chorus-postgres 2>$null
    $ErrorActionPreference = $prevEAP2
    if ($status -eq "healthy") { $pgReady = $true; break }
    if ($i -gt 0 -and $i % 5 -eq 0) { Log "  Waiting for PostgreSQL ($($i * 2)s)..." Yellow }
    Start-Sleep -Seconds 2
}
if ($pgReady) { Ok "PostgreSQL healthy" } else { Warn "PostgreSQL not healthy yet - backend may retry" }

$redisReady = $false
for ($i = 0; $i -lt 15; $i++) {
    $prevEAP3 = $ErrorActionPreference
    $ErrorActionPreference = "SilentlyContinue"
    $status = docker inspect --format='{{.State.Health.Status}}' chorus-redis 2>$null
    $ErrorActionPreference = $prevEAP3
    if ($status -eq "healthy") { $redisReady = $true; break }
    Start-Sleep -Seconds 2
}
if ($redisReady) { Ok "Redis healthy" } else { Warn "Redis not healthy yet" }
Log ""

# ──────────────────────────────────────────────
# 7. Start Go backend with air (hot-reload)
# ──────────────────────────────────────────────
Log "[7/8] Starting Go backend..." Yellow

$BackendDir = Join-Path $RootDir "backend"

# Kill any existing processes on port 8080
$portPid = netstat -ano | Select-String ":8080\s" | ForEach-Object { $_.ToString().Trim().Split()[-1] } | Select-Object -First 1
if ($portPid -and $portPid -match '^\d+$') {
    Warn "Killing existing process (PID $portPid) on port 8080..."
    Stop-Process -Id ([int]$portPid) -Force -ErrorAction SilentlyContinue
    Start-Sleep -Seconds 2
}

# Set environment for backend
$env:ENVIRONMENT = "development"
$env:DATABASE_URL = "postgres://messenger:password@localhost:5432/messenger_dev?sslmode=disable"
$env:REDIS_URL = "localhost:6379"
$env:JWT_SECRET = "dev-jwt-secret-key-for-testing-only"
$env:PORT = "8080"

# Check if air is installed
$airInstalled = Get-Command air -ErrorAction SilentlyContinue
if (-not $airInstalled) {
    Log "  Installing air (Go hot-reload tool)..." Yellow
    go install github.com/air-verse/air@latest 2>&1 | ForEach-Object { Write-Host "  $_" }
    if ($LASTEXITCODE -ne 0) {
        Fail "Failed to install air. Install Go and run: go install github.com/air-verse/air@latest"
    } else {
        Ok "air installed"
    }
}

# Build first to catch errors
Log "  Building backend..." Yellow
Push-Location $BackendDir
go build -o ./tmp/main.exe ./cmd/server 2>&1 | ForEach-Object { Write-Host "  $_" }
if ($LASTEXITCODE -ne 0) {
    Fail "Build failed. Fix errors before starting."
    Pop-Location
} else {
    Ok "Build successful"
    Pop-Location

    # Start backend with air in a new window
    Start-Process powershell -ArgumentList @(
        "-NoExit", "-Command", @"
            Set-Location '$BackendDir'
            `$env:ENVIRONMENT = 'development'
            `$env:DATABASE_URL = 'postgres://messenger:password@localhost:5432/messenger_dev?sslmode=disable'
            `$env:REDIS_URL = 'localhost:6379'
            `$env:JWT_SECRET = 'dev-jwt-secret-key-for-testing-only'
            `$env:PORT = '8080'
            Write-Host 'Backend starting with air (hot-reload)...' -ForegroundColor Cyan
            air -c .air.toml
"@
    ) -WindowStyle Normal | Out-Null
    Ok "Backend starting in new window (hot-reload on :8080)"
}
Log ""

# ──────────────────────────────────────────────
# 8. Start Metro and install the app on the emulator
# ──────────────────────────────────────────────
Log "[8/8] Starting Metro and installing app on emulator..." Yellow

$MobileDir = Join-Path $RootDir "mobile"
if (-not (Test-Path (Join-Path $MobileDir "package.json"))) {
    Fail "mobile/package.json not found"
    exit 1
}

# Install deps if needed
if (-not (Test-Path (Join-Path $MobileDir "node_modules"))) {
    Log "  Installing mobile dependencies..." Yellow
    Push-Location $MobileDir
    npm install 2>&1 | ForEach-Object { Write-Host "  $_" }
    Pop-Location
}

# Determine host IP for the app to reach the backend (Expo inlines EXPO_PUBLIC_*;
# a bare React Native dev build normally uses the emulator's 10.0.2.2 host alias,
# which resolveApiConfig in @chorus/shared auto-selects when no origin is set).
$ip = $null
try {
    $ip = (Get-NetIPAddress -AddressFamily IPv4 -ErrorAction SilentlyContinue |
        Where-Object {
            $_.InterfaceAlias -match "Wi-Fi|Ethernet" -and
            $_.IPAddress -ne "127.0.0.1" -and
            $_.IPAddress -notmatch "^169\.254\." -and
            $_.IPAddress -notmatch "^172\.(1[6-9]|2[0-9]|3[01])\." -and
            $_.IPAddress -ne "0.0.0.0"
        } |
        Sort-Object -Property InterfaceIndex | Select-Object -First 1).IPAddress
} catch { }

if (-not $ip) {
    $ipCandidates = ipconfig | Select-String "IPv4.*?:\s+(\d+\.\d+\.\d+\.\d+)" |
        ForEach-Object { $_.Matches[0].Groups[1].Value } |
        Where-Object { $_ -ne "127.0.0.1" -and $_ -notmatch "^169\.254\." }
    if ($ipCandidates) { $ip = $ipCandidates | Select-Object -First 1 }
}
if (-not $ip) {
    $ip = "10.0.2.2"
    Warn "Could not detect host IP, using emulator host alias: $ip"
}

$backendPort = "8080"
$expoUrl = "http://${ip}:${backendPort}"
$env:EXPO_PUBLIC_API_URL = $expoUrl
Ok "Backend URL: ${expoUrl} (emulator uses http://10.0.2.2:${backendPort})"

# Start Metro bundler in a new window
Start-Process powershell -ArgumentList @(
    "-NoExit", "-Command", @"
        Set-Location '$MobileDir'
        `$env:EXPO_PUBLIC_API_URL = '$expoUrl'
        Write-Host 'Starting Metro Bundler...' -ForegroundColor Cyan
        Write-Host "Backend URL: $expoUrl" -ForegroundColor Green
        npx react-native start
"@
) -WindowStyle Normal | Out-Null
Ok "Metro bundler starting in a new window (ensure it reaches 'Bundler ready')"

# Build the native app and install it on the connected emulator/device.
Log "  Building and installing app on the device (first build may take several minutes)..." Yellow
Push-Location $MobileDir

# Warm the included @react-native/gradle-plugin build once. On a cold checkout the
# first `settings.gradle` evaluation can fail resolving com.facebook.react.settings
# before the included build has compiled; a no-op gradle invocation bootstraps it.
& (Join-Path $MobileDir "android\gradlew.bat") help --quiet 2>&1 | ForEach-Object { Write-Host "  $_" }

# Now build & install. Retry once after a warmup in case the cold settings-phase
# failed the very first attempt.
npx react-native run-android --no-packager 2>&1 | ForEach-Object { Write-Host "  $_" }
if ($LASTEXITCODE -ne 0) {
    Warn "First install attempt failed - warming Gradle build and retrying once..."
    & (Join-Path $MobileDir "android\gradlew.bat") :app:help --quiet 2>&1 | ForEach-Object { Write-Host "  $_" }
    npx react-native run-android --no-packager 2>&1 | ForEach-Object { Write-Host "  $_" }
}
Pop-Location
Ok "App build/install finished (see output above; app should now be on the emulator)"

Log ""
Log "============================================" Cyan
Log "  All services starting!" Cyan
Log "============================================" Cyan
Log "  Backend API:    http://localhost:8080" White
Log "  Backend URL:    $expoUrl (emulator: http://10.0.2.2:8080)" White
Log "  PostgreSQL:     localhost:5432" White
Log "  Redis:          localhost:6379" White
Log "  Metro bundler:  http://localhost:8081" White
Log ""
Log "  Stop emulator:  adb emu kill" Gray
Log "  Stop backend:   close the backend PowerShell window" Gray
Log "  Stop infra:     docker compose down" Gray
Log "============================================" Cyan
