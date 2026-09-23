<#
.SYNOPSIS
  Run the Chorus mobile app on a physical Android device over USB/ADB.

.DESCRIPTION
  1. Picks the attached USB device (or -DeviceId).
  2. Ensures the Chorus backend is running (starts it if needed; auto-falls
     back to port 8090 when 8080 is taken by another service).
  3. Sets up `adb reverse` so the phone reaches the host backend + Metro over
     USB (no Wi-Fi / firewall needed).
  4. Starts Metro with EXPO_PUBLIC_API_URL pointing at localhost (via reverse).
  5. Builds + installs + launches the debug APK on the phone.

.EXAMPLE
  ./run-android-usb.ps1
  ./run-android-usb.ps1 -DeviceId R5CXC2YATLA -ResetCache
#>
param(
  [string]$DeviceId = "",
  [int]$Port = 8080,
  [switch]$ResetCache
)

$ErrorActionPreference = "Stop"
$MobileDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = Split-Path -Parent $MobileDir
$BackendDir = Join-Path $RepoRoot "backend"
$LogDir = Join-Path $MobileDir "logs"
New-Item -ItemType Directory -Force -Path $LogDir | Out-Null

function Test-ChorusHealth([string]$Url) {
  try {
    $r = curl.exe -s -m 3 "$Url/health" 2>$null
    return ($r -match '"status"\s*:\s*"healthy"')
  } catch { return $false }
}

# --- 1. Pick USB device (skip emulators) ---
# NOTE: @(...) forces an array — without it a single device stays a scalar
# string and $devices[0] returns its first CHARACTER.
$devices = @((adb devices) | Select-String "^\S+\s+device$" | ForEach-Object { ($_ -split "\s+")[0] } |
  Where-Object { $_ -notmatch "^emulator-" })
if ($DeviceId -ne "") {
  if ($devices -notcontains $DeviceId) { throw "Device '$DeviceId' not found via adb. Attached: $($devices -join ', ')" }
} else {
  if ($devices.Count -eq 0) { throw "No USB device found. Plug in the phone, enable USB debugging, accept the RSA prompt." }
  if ($devices.Count -gt 1) { throw "Multiple devices attached ($($devices -join ', ')). Re-run with -DeviceId <id>." }
  $DeviceId = $devices[0]
}
Write-Host "==> USB device: $DeviceId" -ForegroundColor Green

# --- 2. Ensure Chorus backend (8080 may belong to another project) ---
$BackendPort = $Port
if (-not (Test-ChorusHealth "http://localhost:$BackendPort")) {
  $fallback = 8090
  if (Test-ChorusHealth "http://localhost:$fallback") {
    Write-Host "==> Port $BackendPort is taken by another service; reusing Chorus backend on $fallback." -ForegroundColor Yellow
    $BackendPort = $fallback
  } else {
    if ($BackendPort -ne $Port) { $BackendPort = $Port }
    try { $null = curl.exe -s -m 2 "http://localhost:$BackendPort/health" 2>$null; $occupied = $true } catch { $occupied = $false }
    if ($occupied -and -not (Test-ChorusHealth "http://localhost:$BackendPort")) {
      Write-Host "==> Port $BackendPort is taken by a non-Chorus service; starting backend on $fallback." -ForegroundColor Yellow
      $BackendPort = $fallback
    }
    Write-Host "==> Starting Chorus backend on :$BackendPort ..." -ForegroundColor Cyan
    # NOTE: -port flag (not $env:PORT): backend/.env is loaded with
    # godotenv.Overload and would clobber inherited env.
    if (-not $env:DATABASE_URL) { $env:DATABASE_URL = "postgres://messenger:password@localhost:5432/messenger_dev?sslmode=disable" }
    if (-not $env:REDIS_URL) { $env:REDIS_URL = "localhost:6379" }
    if (-not $env:JWT_SECRET) { $env:JWT_SECRET = "dev-local-secret-change-me" }
    $outLog = Join-Path $LogDir "backend-usb.log"
    $errLog = Join-Path $LogDir "backend-usb.err.log"
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

# --- 3. adb reverse: phone -> host backend + Metro over USB ---
adb -s $DeviceId reverse tcp:$BackendPort tcp:$BackendPort | Out-Null
adb -s $DeviceId reverse tcp:8081 tcp:8081 | Out-Null
Write-Host "==> adb reverse set (device localhost:$BackendPort -> host, :8081 Metro)" -ForegroundColor Green

# --- 3b. Pin API URL in mobile/.env (authoritative for babel) ---
# react-native-dotenv only inlines keys present in mobile/.env — shell env
# alone is NOT enough (verified: bundle kept the 10.0.2.2 fallback without it).
$EnvFile = Join-Path $MobileDir ".env"
$ApiLine = "EXPO_PUBLIC_API_URL=http://localhost:$BackendPort"
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

# --- 4. Metro (with API URL baked for the USB path) ---
$metroUp = $false
try { $null = curl.exe -s -m 2 "http://localhost:8081/status" 2>$null; $metroUp = $true } catch { $metroUp = $false }
if (-not $metroUp) {
  Write-Host "==> Starting Metro ..." -ForegroundColor Cyan
  $env:EXPO_PUBLIC_API_URL = "http://localhost:$BackendPort"
  $metroArgs = ""
  if ($ResetCache) { $metroArgs = "--reset-cache" }
  # Own window so bundler logs stay visible and Ctrl+C is easy.
  # NOTE: `npx react-native start --reset-cache`, NOT `npm start -- --reset-cache`
  # (the extra `--` swallows the flag and the stale transform cache survives).
  Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$MobileDir'; `$env:EXPO_PUBLIC_API_URL='http://localhost:$BackendPort'; npx react-native start $metroArgs" -WorkingDirectory $MobileDir
  for ($i = 0; $i -lt 30 -and -not $metroUp; $i++) {
    Start-Sleep -Seconds 2
    try { $null = curl.exe -s -m 2 "http://localhost:8081/status" 2>$null; $metroUp = $true } catch {}
  }
  if (-not $metroUp) { throw "Metro did not come up on :8081" }
} else {
  Write-Host "==> Metro already running on :8081 (reusing; ensure it was started with EXPO_PUBLIC_API_URL=http://localhost:$BackendPort)" -ForegroundColor Yellow
}

# --- 5. Build + install + launch on the phone ---
Write-Host "==> Building + installing on $DeviceId (gradle assembleDebug, first run takes minutes) ..." -ForegroundColor Cyan
Set-Location $MobileDir
& npx react-native run-android --deviceId $DeviceId
if ($LASTEXITCODE -ne 0) { throw "run-android failed (exit $LASTEXITCODE)" }

Write-Host ""
Write-Host "==> App launched on $DeviceId. Backend :$BackendPort | Metro :8081" -ForegroundColor Green
Write-Host "    Logs:  adb -s $DeviceId logcat | Select-String 'ReactNativeJS'"
