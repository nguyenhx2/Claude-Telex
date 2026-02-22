# Claude TELEX — Installer for Windows
# Usage: irm https://raw.githubusercontent.com/nguyenhx2/claude-telex/main/install.ps1 | iex
#
# Downloads the latest release from GitHub, installs to %LOCALAPPDATA%\claude-telex,
# adds to PATH, and optionally enables autostart.

$ErrorActionPreference = "Stop"
$repo = "nguyenhx2/claude-telex"
$installDir = "$env:LOCALAPPDATA\claude-telex"
$exeName = "claude-telex.exe"

Write-Host ""
Write-Host "  ╔══════════════════════════════════════╗" -ForegroundColor Green
Write-Host "  ║    Claude TELEX — Vietnamese IME Fix    ║" -ForegroundColor Green
Write-Host "  ╚══════════════════════════════════════╝" -ForegroundColor Green
Write-Host ""

# Get latest release info
Write-Host "[1/4] Fetching latest release..." -ForegroundColor Cyan
$releaseUrl = "https://api.github.com/repos/$repo/releases/latest"
try {
    $release = Invoke-RestMethod -Uri $releaseUrl -Headers @{ Accept = "application/vnd.github.v3+json" }
} catch {
    Write-Host "  Error: Could not fetch release info. Check your internet connection." -ForegroundColor Red
    exit 1
}

$version = $release.tag_name
Write-Host "  Latest version: $version" -ForegroundColor Gray

# Determine architecture
$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
$assetName = "claude-telex_*_windows_$arch.zip"
$asset = $release.assets | Where-Object { $_.name -like $assetName } | Select-Object -First 1

if (-not $asset) {
    Write-Host "  Error: No matching asset found for windows/$arch" -ForegroundColor Red
    exit 1
}

# Download
Write-Host "[2/4] Downloading $($asset.name)..." -ForegroundColor Cyan
$tempZip = Join-Path $env:TEMP "claude-telex-download.zip"
Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $tempZip

# Extract
Write-Host "[3/4] Installing to $installDir..." -ForegroundColor Cyan
New-Item -ItemType Directory -Path $installDir -Force | Out-Null
Expand-Archive -Path $tempZip -DestinationPath $installDir -Force
Remove-Item $tempZip -ErrorAction SilentlyContinue

# Verify
$exePath = Join-Path $installDir $exeName
if (-not (Test-Path $exePath)) {
    Write-Host "  Error: $exeName not found after extraction" -ForegroundColor Red
    exit 1
}

# Add to user PATH if not already there
$userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($userPath -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable("PATH", "$userPath;$installDir", "User")
    Write-Host "  Added to PATH (restart terminal to use 'claude-telex' command)" -ForegroundColor Gray
}

# Enable autostart via registry
Write-Host "[4/4] Enabling autostart..." -ForegroundColor Cyan
$regPath = "HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Run"
Set-ItemProperty -Path $regPath -Name "ClaudeTelex" -Value "`"$exePath`""
Write-Host "  Autostart enabled" -ForegroundColor Gray

Write-Host ""
Write-Host "  ✓ Claude TELEX $version installed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "  Run:  claude-telex" -ForegroundColor White
Write-Host "  Or:   $exePath" -ForegroundColor Gray
Write-Host ""

# Launch immediately
Write-Host "Launching Claude TELEX..." -ForegroundColor Cyan
Start-Process -FilePath $exePath
