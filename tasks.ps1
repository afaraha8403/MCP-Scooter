# MCP Scooter Tasks Helper
# Usage: ./tasks.ps1 <command> [args]

param (
    [Parameter(Position=0)]
    [string]$Command = "help",

    [Parameter(Position=1)]
    [string]$Arg1 = "",

    [Parameter(Position=2)]
    [string]$Arg2 = ""
)

# Alias for backwards compatibility
$TestName = $Arg1
$Version = $Arg1

# Ensure UTF-8 output for better visibility of icons
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$RootDir = Get-Location

function Update-Version {
    param([string]$NewVersion)
    
    Write-Host "Updating version to $NewVersion..." -ForegroundColor Cyan
    
    # Update tauri.conf.json
    $tauriConf = Get-Content "desktop/src-tauri/tauri.conf.json" -Raw | ConvertFrom-Json
    $tauriConf.version = $NewVersion
    $tauriConf | ConvertTo-Json -Depth 10 | Set-Content "desktop/src-tauri/tauri.conf.json"
    
    # Update package.json
    $packageJson = Get-Content "desktop/package.json" -Raw | ConvertFrom-Json
    $packageJson.version = $NewVersion
    $packageJson | ConvertTo-Json -Depth 10 | Set-Content "desktop/package.json"
    
    # Update Cargo.toml (simple regex replace)
    $cargoToml = Get-Content "desktop/src-tauri/Cargo.toml" -Raw
    $cargoToml = $cargoToml -replace 'version = "[^"]*"', "version = `"$NewVersion`""
    Set-Content "desktop/src-tauri/Cargo.toml" $cargoToml -NoNewline
    
    Write-Host "Version updated to $NewVersion" -ForegroundColor Green
}

function Show-Help {
    Write-Host "MCP Scooter Task Runner" -ForegroundColor Green
    Write-Host "========================"
    Write-Host ""
    Write-Host "Usage: ./tasks.ps1 <command>"
    Write-Host ""
    Write-Host "Build & Run:"
    Write-Host "  all               - Validate registry and build"
    Write-Host "  build             - Build the scooter binary"
    Write-Host "  build-installer   - Build Windows MSI/NSIS installers"
    Write-Host "  dev               - Run in development mode"
    Write-Host "  clean             - Clean build artifacts"
    Write-Host "  deps              - Install dependencies"
    Write-Host ""
    Write-Host "Testing (Levels 1-2):"
    Write-Host "  test              - Run all tests"
    Write-Host "  test-unit         - Run all unit tests (verbose)"
    Write-Host "  test-registry     - Test registry validation logic"
    Write-Host "  test-discovery    - Test discovery engine"
    Write-Host "  test-profile      - Test profile management"
    Write-Host "  test-api          - Test API/SSE server"
    Write-Host "  test-integration  - Test client integrations"
    Write-Host "  test-coverage     - Generate HTML coverage report"
    Write-Host ""
    Write-Host "Testing (Levels 5):"
    Write-Host "  test-meta-mcp     - Meta-MCP lifecycle tests"
    Write-Host ""
    Write-Host "Validation:"
    Write-Host "  validate          - Validate registry JSON files"
    Write-Host "  validate-strict   - Validate with warnings as errors"
    Write-Host ""
    Write-Host "CI & Git:"
    Write-Host "  ci                - Run standard CI checks"
    Write-Host "  ci-full           - Run full CI checks with coverage"
    Write-Host "  pre-commit        - Quick check before committing"
    Write-Host "  fmt               - Format code"
    Write-Host "  lint              - Lint code"
    Write-Host ""
    Write-Host "Release:"
    Write-Host "  release [version]      - Update version, tag, and push a stable release"
    Write-Host "  release-beta [version] - Update version, tag, and push a beta release"
    Write-Host "  set-version <version>  - Update version in config files only (no commit/tag)"
    Write-Host ""
    Write-Host "Helper:"
    Write-Host "  test-run <name>   - Run specific test by name"
    Write-Host ""
    Write-Host "Examples:"
    Write-Host "  ./tasks.ps1 release 1.0.0"
    Write-Host "  ./tasks.ps1 release-beta 1.0.0-beta.2"
    Write-Host "  ./tasks.ps1 set-version 1.0.0"
    Write-Host ""
}

switch ($Command) {
    "all" {
        & ./tasks.ps1 validate
        & ./tasks.ps1 build
    }

    "build" {
        Write-Host "Building Scooter..." -ForegroundColor Cyan
        go build -o scooter.exe ./cmd/scooter
    }

    "build-installer" {
        Write-Host "Building Windows Installer..." -ForegroundColor Cyan
        
        # Step 1: Build Go backend for Tauri bundle
        Write-Host "  [1/3] Building Go backend..." -ForegroundColor Gray
        $env:GOOS = "windows"
        $env:GOARCH = "amd64"
        go build -o desktop/src-tauri/binaries/scooter-x86_64-pc-windows-msvc.exe ./cmd/scooter
        if ($LASTEXITCODE -ne 0) {
            Write-Host "✗ Go build failed" -ForegroundColor Red
            exit 1
        }
        
        # Step 2: Install frontend dependencies if needed
        Write-Host "  [2/3] Checking frontend dependencies..." -ForegroundColor Gray
        if (-not (Test-Path "desktop/node_modules")) {
            Set-Location desktop
            npm install
            Set-Location $RootDir
        }
        
        # Step 3: Build Tauri app
        Write-Host "  [3/3] Building Tauri installer..." -ForegroundColor Gray
        Set-Location desktop
        npm run tauri build
        $buildResult = $LASTEXITCODE
        Set-Location $RootDir
        
        if ($buildResult -eq 0) {
            Write-Host ""
            Write-Host "✓ Build complete!" -ForegroundColor Green
            Write-Host ""
            Write-Host "Installers created:" -ForegroundColor Cyan
            Write-Host "  MSI:  desktop/src-tauri/target/release/bundle/msi/" -ForegroundColor Gray
            Write-Host "  NSIS: desktop/src-tauri/target/release/bundle/nsis/" -ForegroundColor Gray
        } else {
            Write-Host "✗ Tauri build failed" -ForegroundColor Red
            exit 1
        }
    }

    "build-validator" {
        go build -o validate-registry.exe ./cmd/validate-registry
    }

    "test" {
        Write-Host "--- Running All Tests ---" -ForegroundColor Cyan
        go test ./...
        if ($LASTEXITCODE -eq 0) {
            Write-Host "`n✓ SUCCESS: All test suites passed." -ForegroundColor Green
        } else {
            Write-Host "`n✗ FAILURE: Some tests failed. Run './tasks.ps1 test-unit' for details." -ForegroundColor Red
        }
    }

    "test-unit" {
        Write-Host "--- Running Unit Tests (Verbose) ---" -ForegroundColor Cyan
        go test ./... -v
    }

    "test-registry" {
        go test ./internal/domain/registry/... -v
    }

    "test-discovery" {
        go test ./internal/domain/discovery/... -v
    }

    "test-profile" {
        go test ./internal/domain/profile/... -v
    }

    "test-api" {
        go test ./internal/api/... -v
    }

    "test-integration" {
        go test ./internal/domain/integration/... -v
    }

    "test-coverage" {
        go test ./internal/... -coverprofile=coverage.out
        go tool cover -html=coverage.out -o coverage.html
        Write-Host "Coverage report generated: coverage.html" -ForegroundColor Green
    }

    "test-meta-mcp" {
        go test ./internal/domain/discovery/... -v -run "Engine"
    }

    "test-run" {
        if (-not $TestName) {
            Write-Host "Error: Please provide a test name pattern." -ForegroundColor Red
            Write-Host "Usage: ./tasks.ps1 test-run TestMyFunction"
            exit 1
        }
        go test ./... -v -run $TestName
    }

    "validate" {
        Write-Host "--- Validating MCP Registry ---" -ForegroundColor Cyan
        & ./tasks.ps1 build-validator
        ./validate-registry.exe appdata/registry/official
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✓ Registry is valid." -ForegroundColor Green
        } else {
            Write-Host "✗ Registry validation failed." -ForegroundColor Red
        }
    }

    "validate-strict" {
        & ./tasks.ps1 build-validator
        ./validate-registry.exe -strict appdata/registry/official
    }

    "fmt" {
        go fmt ./...
        Set-Location desktop
        npm run format
        Set-Location $RootDir
    }

    "lint" {
        go vet ./...
    }

    "deps" {
        go mod download
        Set-Location desktop
        npm install
        Set-Location $RootDir
    }

    "dev" {
        go run ./cmd/scooter
    }

    "clean" {
        Remove-Item -Path "scooter.exe", "validate-registry.exe", "coverage.out", "coverage.html" -ErrorAction SilentlyContinue
        Remove-Item -Path "desktop/dist" -Recurse -Force -ErrorAction SilentlyContinue
        Write-Host "Cleaned build artifacts." -ForegroundColor Gray
    }

    "ci" {
        & ./tasks.ps1 fmt
        & ./tasks.ps1 lint
        & ./tasks.ps1 validate
        & ./tasks.ps1 test
        & ./tasks.ps1 build
        Write-Host "CI checks passed!" -ForegroundColor Green
    }

    "ci-full" {
        & ./tasks.ps1 fmt
        & ./tasks.ps1 lint
        & ./tasks.ps1 validate-strict
        & ./tasks.ps1 test-coverage
        & ./tasks.ps1 build
        Write-Host "Full CI checks passed!" -ForegroundColor Green
    }

    "pre-commit" {
        & ./tasks.ps1 fmt
        & ./tasks.ps1 lint
        & ./tasks.ps1 validate
        & ./tasks.ps1 test
        Write-Host "Pre-commit checks passed!" -ForegroundColor Green
    }

    "release" {
        if (-not $Version) {
            $Version = Read-Host "Enter version (e.g., 1.0.0)"
        }
        # Update version in all config files
        Update-Version $Version
        # Commit version bump
        git add desktop/src-tauri/tauri.conf.json desktop/package.json desktop/src-tauri/Cargo.toml
        git commit -m "chore: bump version to $Version"
        git push origin main
        # Create and push tag
        git tag -a "v$Version" -m "Release v$Version"
        git push origin "v$Version"
        Write-Host "GitHub Action will now build and release v$Version" -ForegroundColor Green
    }

    "release-beta" {
        if (-not $Version) {
            $Version = Read-Host "Enter beta version (e.g., 1.0.0-beta.1)"
        }
        # Extract base version (e.g., 1.0.0 from 1.0.0-beta.1)
        $BaseVersion = $Version -replace '-.*$', ''
        # Update version in all config files (use base version for files)
        Update-Version $BaseVersion
        # Commit version bump
        git add desktop/src-tauri/tauri.conf.json desktop/package.json desktop/src-tauri/Cargo.toml
        git commit -m "chore: bump version to $BaseVersion for $Version release"
        git push origin main
        # Create and push tag
        git tag -a "v$Version" -m "Beta release v$Version"
        git push origin "v$Version"
        Write-Host "GitHub Action will now build and release v$Version" -ForegroundColor Green
    }

    "set-version" {
        if (-not $Version) {
            $Version = Read-Host "Enter version (e.g., 1.0.0)"
        }
        Update-Version $Version
        Write-Host "Version updated to $Version in all config files." -ForegroundColor Green
        Write-Host "Files updated:" -ForegroundColor Gray
        Write-Host "  - desktop/src-tauri/tauri.conf.json"
        Write-Host "  - desktop/package.json"
        Write-Host "  - desktop/src-tauri/Cargo.toml"
    }

    "help" {
        Show-Help
    }

    default {
        Write-Host "Unknown command: $Command" -ForegroundColor Red
        Show-Help
        exit 1
    }
}
