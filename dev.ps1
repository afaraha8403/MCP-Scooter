# MCP Scooter Development Helper Script
# Usage: .\dev.ps1

$RootDir = Get-Location

# 0. Cleanup existing processes
Write-Host "--- Cleaning up existing processes ---" -ForegroundColor Gray
Get-Process "main" -ErrorAction SilentlyContinue | Stop-Process -Force
Get-Process "scooter" -ErrorAction SilentlyContinue | Stop-Process -Force

# Ensure Cargo is in the PATH for this session
if (!(Get-Command "cargo" -ErrorAction SilentlyContinue)) {
    $CargoBin = Join-Path $env:USERPROFILE ".cargo\bin"
    if (Test-Path $CargoBin) {
        Write-Host "Adding $CargoBin to PATH..." -ForegroundColor Gray
        $env:PATH = "$CargoBin;$env:PATH"
    } else {
        Write-Host "Warning: Cargo not found. Tauri may fail to build." -ForegroundColor Yellow
    }
}

Write-Host "--- MCP Scooter Dev Startup ---" -ForegroundColor Green

# 1. Start Go Backend in the background
Write-Host "[1/2] Starting Go Backend (Control Server on :6200)..." -ForegroundColor Cyan
$BackendJob = Start-Job -ScriptBlock {
    param($path)
    Set-Location $path
    go run ./cmd/scooter/main.go
} -ArgumentList $RootDir

# Wait a moment for the backend to initialize
Start-Sleep -Seconds 3

# Check if backend started
$JobState = Get-Job -Id $BackendJob.Id
if ($JobState.State -eq "Failed") {
    Write-Host "Error: Backend job failed to start." -ForegroundColor Red
    Receive-Job -Job $BackendJob
    Read-Host "Press Enter to exit..."
    exit 1
}

# 2. Start Tauri Frontend
Write-Host "[2/2] Starting Tauri Frontend..." -ForegroundColor Cyan
Set-Location -Path "$RootDir\desktop"

try {
    npm run tauri dev
} catch {
    Write-Host "Error: Failed to start Tauri frontend." -ForegroundColor Red
    Write-Host $_
} finally {
    # Cleanup backend when frontend is closed
    Write-Host "--- Shutting down ---" -ForegroundColor Yellow
    Stop-Job -Job $BackendJob
    Remove-Job -Job $BackendJob
    Set-Location $RootDir
    Write-Host "Dev session ended." -ForegroundColor Gray
    Read-Host "Press Enter to exit..."
}
