<#
.SYNOPSIS
  Run the Chorus mobile app on an Android emulator (AVD).

.DESCRIPTION
  1. Boots the AVD if needed (default: Chorus_Test) and waits for boot.
  2. Ensures the Chorus backend is running (starts it if needed; auto-falls
     back to port 8090 when 8080 is taken by another service).
  3. Starts Metro (emulator reaches the host via 10.0.2.2, so no
     EXPO_PUBLIC_API_URL override and no adb reverse are needed).
  4. Builds + installs + launches the debug APK on the emulator.

  NOTE: the emulator is CPU-heavy on this PC. If it stutters, prefer
  ../run-android-usb.ps1 with a physical phone over USB.

.EXAMPLE
  ./run-android-avd.ps1
  ./run-android-avd.ps1 -Avd Chorus_Test -ResetCache
#>
param(
  [string]$Avd = "Chorus_Test",
  [int]$Port = 8080,
  [switch]$ResetCache
)

$ErrorActionPreference = "Stop"
$MobileDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = Split-Path -Parent $MobileDir
$BackendDir = Join-Path $RepoRoot "backend"
$LogDir = Join-Path $MobileDir "logs"
New-Item -ItemType Directory -Force -Path $LogDir | Out-Null

$Sdk = $env:ANDROID_HOME
if (-not $Sdk) { $Sdk = Join-Path $env:LOCALAPPDATA "Android\Sdk" }
$EmulatorExe = Join-Path $Sdk "emulator\emulator.exe"
if (-not (Test-Path $EmulatorExe)) { throw "emulator.exe not found at $EmulatorExe (set ANDROID_HOME)" }

function Test-ChorusHealth([string]$Url) {
  try {
    $r = curl.exe -s -m 3 "$Url/health" 2>$null
    return ($r -match '"status"\s*:\s*"healthy"')
  } catch { return $false }
}

function Get-EmulatorId {
  $ids = @((adb devices) | Select-String "^emulator-\d+\s+device$" | ForEach-Object { ($_ -split "\s+")[0] })
  if ($ids.Count -eq 0) { return $null }
  return $ids[0]
}

# --- 1. Boot AVD if needed ---
$emu = Get-EmulatorId
if (-not $emu) {
  Write-Host "==> Booting AVD '$Avd' (heavy on CPU; use -Avd to pick another) ..." -ForegroundColor Cyan
  Start-Process -FilePath $EmulatorExe -ArgumentList "-avd", $Avd -WindowStyle Minimized
  $booted = $false
  for ($i = 0; $i -lt 150; $i++) {
    Start-Sleep -Seconds 4
    $emu = Get-EmulatorId
    if ($emu) {
      try {
        $b = adb -s $emu shell getprop sys.boot_completed 2>$null
        if ($b -match "1") { $booted = $true; break }
      } catch {}
    }
  }
  if (-not $booted) { throw "Emulator '$Avd' did not finish booting in ~10 min." }
}
Write-Host "==> Emulator: $emu" -ForegroundColor Green

# --- 2. Ensure Chorus backend ---
$BackendPort = $Port
if (-not (Test-ChorusHealth "http://localhost:$BackendPort")) {
  $fallback = 8090
  if (Test-ChorusHealth "http://localhost:$fallback") {
    Write-Host "==> Port $BackendPort is taken by another service; reusing Chorus backend on $fallback." -ForegroundColor Yellow
    $BackendPort = $fallback
  } else {
    try { $null = curl.exe -s -m 2 "http://localhost:$BackendPort/health" 2>$null; $occupied = $true } catch { $occupied = $false }
    if ($occupied) {
      Write-Host "==> Port $BackendPort is taken by a non-Chorus service; starting backend on $fallback." -ForegroundColor Yellow
      $BackendPort = $fallback
    }
    Write-Host "==> Starting Chorus backend on :$BackendPort ..." -ForegroundColor Cyan
    # NOTE: -port flag (not $env:PORT): backend/.env is loaded with
    # godotenv.Overload and would clobber inherited env.
    if (-not $env:DATABASE_URL) { $env:DATABASE_URL = "postgres://messenger:password@localhost:5432/messenger_dev?sslmode=disable" }
    if (-not $env:REDIS_URL) { $env:REDIS_URL = "localhost:6379" }
    if (-not $env:JWT_SECRET) { $env:JWT_SECRET = "dev-local-secret-change-me" }
    $outLog = Join-Path $LogDir "backend-avd.log"
    $errLog = Join-Path $LogDir "backend-avd.err.log"
    Start-Process -FilePath "go" -ArgumentList "run ./cmd/server -port $BackendPort" -WorkingDirectory $BackendDir `
      -RedirectStandardOutput $outLog -RedirectStandardError $errLog -WindowStyle Hidden
    $ready = $false
    for ($i = 0; $i -lt 30; $i++) {
      Start-Sleep -Seconds 2
      if (Test-ChorusHealth "http://localhost:$BackendPort") { $ready = $true; break }
    }
    if (-not $ready) { throw "Backend did not become healthy on :$BackendPort. See $outLog and $errLog" }
  }
}
Write-Host "==> Chorus backend healthy on :$BackendPort" -ForegroundColor Green

# --- 3a. Pin API URL in mobile/.env (authoritative for babel) ---
# Emulator reaches the host via 10.0.2.2 (NOT localhost). react-native-dotenv
# only inlines keys present in mobile/.env — shell env alone is not enough.
$EnvFile = Join-Path $MobileDir ".env"
$ApiLine = "EXPO_PUBLIC_API_URL=http://10.0.2.2:$BackendPort"
if (Test-Path $EnvFile) {
  $content = Get-Content $EnvFile -Raw
  if ($content -match "(?m)^EXPO_PUBLIC_API_URL=.*$") {
    $content = $content -replace "(?m)^EXPO_PUBLIC_API_URL=.*$", $ApiLine
  } else {
    $content = $content.TrimEnd() + "`n$ApiLine`n"
  }
  Set-Content $EnvFile $content -NoNewline:$false
} else {
  Set-Content $EnvFile "$ApiLine`n"
}
Write-Host "==> mobile/.env API URL: $ApiLine" -ForegroundColor Green

# --- 3. Metro (emulator uses 10.0.2.2 origin from .env) ---
$metroUp = $false
try { $null = curl.exe -s -m 2 "http://localhost:8081/status" 2>$null; $metroUp = $true } catch { $metroUp = $false }
if (-not $metroUp) {
  Write-Host "==> Starting Metro ..." -ForegroundColor Cyan
  $metroArgs = ""
  if ($ResetCache) { $metroArgs = "--reset-cache" }
  # NOTE: `npx react-native start --reset-cache`, NOT `npm start -- --reset-cache`
  # (the extra `--` swallows the flag and the stale transform cache survives).
  Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$MobileDir'; npx react-native start $metroArgs" -WorkingDirectory $MobileDir
  for ($i = 0; $i -lt 30 -and -not $metroUp; $i++) {
    Start-Sleep -Seconds 2
    try { $null = curl.exe -s -m 2 "http://localhost:8081/status" 2>$null; $metroUp = $true } catch {}
  }
  if (-not $metroUp) { throw "Metro did not come up on :8081" }
} else {
  Write-Host "==> Metro already running on :8081 (reusing)" -ForegroundColor Yellow
}

# --- 4. Build + install + launch on the emulator ---
# Non-default backend ports need an explicit origin on AVD too: the emulator
# cannot use the phone's adb-reverse path, so map host port explicitly.
if ($BackendPort -ne 8080) {
  adb -s $emu reverse tcp:$BackendPort tcp:$BackendPort | Out-Null
  Write-Host "==> Non-default backend port: adb reverse + EXPO_PUBLIC_API_URL=http://localhost:$BackendPort required." -ForegroundColor Yellow
  Write-Host "    Restart Metro with `$env:EXPO_PUBLIC_API_URL='http://localhost:$BackendPort' then re-run." -ForegroundColor Yellow
}
Write-Host "==> Building + installing on $emu (gradle assembleDebug, first run takes minutes) ..." -ForegroundColor Cyan
Set-Location $MobileDir
& npx react-native run-android --deviceId $emu
if ($LASTEXITCODE -ne 0) { throw "run-android failed (exit $LASTEXITCODE)" }

Write-Host ""
Write-Host "==> App launched on $emu. Backend :$BackendPort | Metro :8081" -ForegroundColor Green
