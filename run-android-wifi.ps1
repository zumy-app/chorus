<#
.SYNOPSIS
  Run the Chorus mobile app on a physical Android device over Wi-Fi ADB.

.DESCRIPTION
  0. Preflight: verifies adb/node exist, syncs mobile/node_modules when stale
     (missing react-native-dotenv 500s every Metro bundle), and smoke-loads
     metro.config.js before touching the device.
  1. Finds/connects the Wi-Fi ADB device: reuses an already-connected one,
     else tries mDNS discovery, else uses -DeviceIp (with optional -AdbPort
     and one-time -PairPort/-PairCode pairing).
  2. Ensures the Chorus backend is running (starts it if needed; auto-falls
     back to port 8090 when 8080 is taken by another service).
  3. Sets up `adb reverse` so the phone reaches the host backend + Metro over
     the ADB transport (no LAN IP / firewall needed — seamless across networks).
  4. Starts Metro with EXPO_PUBLIC_API_URL pointing at localhost (via reverse).
  5. Builds + installs + launches the debug APK on the phone.

  Phone setup (once): Settings > System > Developer options > Wireless
  debugging ON. Read the "IP address & Port" shown there (e.g.
  192.168.1.155:37851). First time only: "Pair device with pairing code"
  gives an IP:pair-port + 6-digit code.

.EXAMPLE
  ./run-android-wifi.ps1
  ./run-android-wifi.ps1 -DeviceIp 192.168.1.155 -AdbPort 37851
  ./run-android-wifi.ps1 -DeviceIp 192.168.1.155 -PairPort 41165 -PairCode 123456
#>
param(
  [string]$DeviceId = "",
  [string]$DeviceIp = "",
  [int]$AdbPort = 0,
  [int]$PairPort = 0,
  [string]$PairCode = "",
  [int]$Port = 8080,
  [switch]$ResetCache,
  [switch]$SkipDepsCheck
)

$ErrorActionPreference = "Stop"
$RepoRoot = $PSScriptRoot
$MobileDir = Join-Path $RepoRoot "mobile"
$BackendDir = Join-Path $RepoRoot "backend"
$LogDir = Join-Path $MobileDir "logs"
New-Item -ItemType Directory -Force -Path $LogDir | Out-Null

function Test-ChorusHealth([string]$Url) {
  try {
    $r = curl.exe -s -m 3 "$Url/health" 2>$null
    return ($r -match '"status"\s*:\s*"healthy"')
  } catch { return $false }
}

function Get-WifiDevices {
  # Wi-Fi ADB devices show as <ip>:<port> (emulators excluded).
  # NOTE: @(...) forces an array — without it a single device stays a scalar
  # string and $devices[0] returns its first CHARACTER.
  return @((adb devices) | Select-String "^\S+\s+device$" | ForEach-Object { ($_ -split "\s+")[0] } |
    Where-Object { $_ -match ":\d+$" -and $_ -notmatch "^emulator-" })
}

# --- 0. Preflight: toolchain + JS deps in sync ---
# A present-but-stale node_modules is a silent killer: e.g. a missing
# react-native-dotenv aborts every Babel transform and Metro serves HTTP 500
# to the phone. Fail fast here instead of after the gradle build.
if (-not (Get-Command adb -ErrorAction SilentlyContinue)) { throw "adb not found on PATH. Install Android platform-tools (ANDROID_HOME=$env:ANDROID_HOME) and re-run." }
if (-not (Get-Command node -ErrorAction SilentlyContinue)) { throw "node not found on PATH. Install Node.js LTS and re-run." }
if (-not $SkipDepsCheck) {
  $sharedIndex = Join-Path $RepoRoot "packages\shared\src\index.ts"
  Push-Location $MobileDir
  try {
    $dotenvOk = $false
    try { $null = & node -e "require.resolve('react-native-dotenv')" 2>$null; $dotenvOk = ($LASTEXITCODE -eq 0) } catch { $dotenvOk = $false }
    $metroOk = $false
    try { $null = & node -e "require('./metro.config.js')" 2>$null; $metroOk = ($LASTEXITCODE -eq 0) } catch { $metroOk = $false }
    $needInstall = (-not $dotenvOk) -or (-not $metroOk) -or (-not (Test-Path $sharedIndex)) -or (-not (Test-Path (Join-Path $MobileDir "node_modules")))
    if ($needInstall) {
      Write-Host "==> JS deps out of sync (dotenv:$dotenvOk metro-config:$metroOk shared-src:$(Test-Path $sharedIndex)) - running npm install ..." -ForegroundColor Cyan
      npm install 2>&1 | ForEach-Object { Write-Host "  $_" }
      if ($LASTEXITCODE -ne 0) { throw "npm install failed (exit $LASTEXITCODE)" }
      try { $null = & node -e "require.resolve('react-native-dotenv'); require('./metro.config.js')" 2>$null; $stillOk = ($LASTEXITCODE -eq 0) } catch { $stillOk = $false }
      if (-not $stillOk) { throw "Deps still broken after npm install (dotenv/metro.config.js won't load). Fix mobile/package.json and re-run." }
      Write-Host "==> JS deps synced" -ForegroundColor Green
    } else {
      Write-Host "==> JS deps OK (dotenv + metro.config.js load)" -ForegroundColor Green
    }
  } finally { Pop-Location }
} else {
  Write-Host "==> Skipped deps check (SkipDepsCheck)" -ForegroundColor Yellow
}

# --- 1. Find / connect the Wi-Fi ADB device ---
$wifi = @(Get-WifiDevices)
if ($DeviceId -ne "") {
  if ($wifi -notcontains $DeviceId) { throw "Device '$DeviceId' not connected via adb. Attached Wi-Fi: $($wifi -join ', ')" }
} elseif ($wifi.Count -eq 1 -and $DeviceIp -eq "") {
  $DeviceId = $wifi[0]
  Write-Host "==> Reusing connected Wi-Fi device: $DeviceId" -ForegroundColor Green
} else {
  if ($wifi.Count -gt 1 -and $DeviceIp -eq "") { throw "Multiple Wi-Fi devices attached ($($wifi -join ', ')). Re-run with -DeviceId IP colon PORT, or with -DeviceIp IP." }
  # Best effort: mDNS discovery (only works when the phone + PC are on the
  # same subnet and multicast isn't blocked).
  $mdnsTargets = @()
  try {
    $mdnsOut = adb mdns services 2>$null
    $mdnsTargets = @($mdnsOut | Select-String "(\d+\.\d+\.\d+\.\d+:\d+)" | ForEach-Object { $_.Matches[0].Groups[1].Value } | Select-Object -Unique)
  } catch { }
  if ($DeviceIp -eq "" -and $mdnsTargets.Count -gt 0) {
    Write-Host "==> mDNS found $($mdnsTargets.Count) candidate(s): $($mdnsTargets -join ', ')" -ForegroundColor Cyan
  }
  $targets = @()
  if ($DeviceIp -ne "") {
    if ($AdbPort -gt 0) { $targets = @("${DeviceIp}:${AdbPort}") }
    elseif ($DeviceIp -match ":\d+$") { $targets = @($DeviceIp) }
    else { throw "Wireless debugging uses a dynamic port. Read it from the Wireless debugging screen on the phone and pass -AdbPort PORT (example: -DeviceIp 192.168.1.155 -AdbPort 37851)." }
    # One-time pairing (only needed if this PC was never paired with the phone).
    if ($PairPort -gt 0) {
      if ($PairCode -eq "") { throw "Pairing needs -PairCode with the 6-digit code from the Pair device screen on the phone." }
      Write-Host "==> Pairing with ${DeviceIp}:${PairPort} ..." -ForegroundColor Cyan
      $PairCode | adb pair "${DeviceIp}:${PairPort}" 2>&1 | ForEach-Object { Write-Host "  $_" }
    }
  } elseif ($mdnsTargets.Count -gt 0) {
    $targets = $mdnsTargets
  } else {
    # Last resort before giving up: retry the last-known address. Pairing keys
    # persist (%USERPROFILE%\.android\adbkey), so only the dynamic port can go
    # stale — a failed connect here is fast and harmless.
    $cached = ""
    $cacheFile = Join-Path $LogDir ".last-wifi-device"
    if (Test-Path $cacheFile) { $cached = (Get-Content $cacheFile -Raw).Trim() }
    if ($cached -ne "") {
      Write-Host "==> mDNS quiet - retrying last-known Wi-Fi device: $cached ..." -ForegroundColor Cyan
      $targets = @($cached)
    } else {
      throw "No Wi-Fi ADB device found. On the phone enable Wireless debugging, read its IP address and Port, then re-run with -DeviceIp IP -AdbPort PORT. mDNS discovery found nothing (same subnet and unblocked multicast required)."
    }
  }
  $connected = $false
  foreach ($t in $targets) {
    Write-Host "==> adb connect $t ..." -ForegroundColor Cyan
    adb connect $t 2>&1 | ForEach-Object { Write-Host "  $_" }
    $wifi = @(Get-WifiDevices)
    if ($wifi -contains $t) { $DeviceId = $t; $connected = $true; break }
  }
  if (-not $connected) {
    $wifi = @(Get-WifiDevices)
    if ($wifi.Count -eq 1) { $DeviceId = $wifi[0]; $connected = $true }
  }
  if (-not $connected) { throw "Could not connect to a Wi-Fi device (tried: $($targets -join ', ')). If the phone shows 'Pair device with pairing code', re-run with -PairPort <port> -PairCode <code>." }
}
Write-Host "==> Wi-Fi device: $DeviceId" -ForegroundColor Green
# Remember for next time: pairing keys persist, so a zero-arg re-run can
# reconnect without manual steps (unless the phone rotated its dynamic port).
try { Set-Content (Join-Path $LogDir ".last-wifi-device") $DeviceId -NoNewline } catch { }
try {
  $model = adb -s $DeviceId shell getprop ro.product.model 2>$null
  if ($model) { Write-Host "==> Model: $($model.Trim())" -ForegroundColor Green }
} catch { }

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
    $outLog = Join-Path $LogDir "backend-wifi.log"
    $errLog = Join-Path $LogDir "backend-wifi.err.log"
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

# --- 3. adb reverse: phone -> host backend + Metro over the ADB transport ---
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

# --- 4. Metro (with API URL baked for the Wi-Fi path) ---
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
& npx react-native run-android --device $DeviceId
if ($LASTEXITCODE -ne 0) { throw "run-android failed (exit $LASTEXITCODE)" }

Write-Host ""
Write-Host "==> App launched on $DeviceId. Backend :$BackendPort | Metro :8081" -ForegroundColor Green
Write-Host "    Logs:  adb -s $DeviceId logcat | Select-String 'ReactNativeJS'"
