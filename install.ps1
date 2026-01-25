# MCP Scooter Installer for Windows
# Usage: irm https://mcp-scooter.com/install.ps1 | iex

$ErrorActionPreference = 'Stop'

Write-Host "
   __  ___  __________  ____  _  _  _____  ____  ____  ____ 
  (  \/  ) / __)(  _ \( ___)( \/ )(  _  )(  _ \(_  _)(  __)
   )    ( ( (__  ) __/ )__)  \  /  )(_)(  )   /  )(   ) _) 
  (_/\/\_) \___)(__)  (____)  \/  (_____)(__\_) (__) (____)
" -ForegroundColor Green

Write-Host "Installing MCP Scooter..." -ForegroundColor Cyan

# 1. Setup Installation Directory
$InstallDir = Join-Path $env:USERPROFILE ".scooter"
$BinDir = Join-Path $InstallDir "bin"

if (!(Test-Path $BinDir)) {
    New-Item -ItemType Directory -Force -Path $BinDir | Out-Null
}

# 2. Determine Download URL (Simulated for now - points to GitHub Releases)
# In production, this would query the GitHub API for the latest tag
$RepoOwner = "mcp-scooter"
$RepoName = "scooter"
$Asset = "scooter_windows_amd64.exe"
$DownloadUrl = "https://github.com/$RepoOwner/$RepoName/releases/latest/download/$Asset"

# 3. Download Binary
$OutputFile = Join-Path $BinDir "scooter.exe"

Write-Host "Downloading latest release..." -ForegroundColor Gray
try {
    # Note: This will fail until actual releases are published. 
    # For now, we'll check if we are in the dev repo and copy the local binary if it exists.
    if (Test-Path ".\scooter.exe") {
        Write-Host "Development mode: Copying local binary..." -ForegroundColor Yellow
        Copy-Item ".\scooter.exe" -Destination $OutputFile -Force
    } else {
        # Fallback to web download (commented out until releases exist to prevent errors in testing)
        # Invoke-WebRequest -Uri $DownloadUrl -OutFile $OutputFile
        Write-Warning "No release found on GitHub yet. Please build locally using 'go build -o scooter.exe ./cmd/scooter'"
    }
} catch {
    Write-Error "Failed to download MCP Scooter. Please check your internet connection."
    exit 1
}

# 4. Add to PATH
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$BinDir*") {
    Write-Host "Adding $BinDir to PATH..." -ForegroundColor Cyan
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$BinDir", "User")
    $env:PATH = "$BinDir;$env:PATH"
}

# 5. Create Config Directory
$ConfigDir = Join-Path $InstallDir "config"
if (!(Test-Path $ConfigDir)) {
    New-Item -ItemType Directory -Force -Path $ConfigDir | Out-Null
    # Create default profiles.yaml
    $DefaultConfig = @"
settings:
  gateway_api_key: "sk-scooter-local-dev"
profiles:
  - id: default
    name: "Default Profile"
    env: {}
    allow_tools: []
"@
    Set-Content -Path (Join-Path $ConfigDir "profiles.yaml") -Value $DefaultConfig
}

Write-Host "
----------------------------------------------------------------
SUCCESS! MCP Scooter has been installed.

Location: $BinDir
Config:   $ConfigDir

Run 'scooter' to start the daemon.
----------------------------------------------------------------
" -ForegroundColor Green
