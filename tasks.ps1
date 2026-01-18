# MCP Scooter Tasks Helper
# Usage: ./tasks.ps1 <command> [args]

param (
    [Parameter(Position=0)]
    [string]$Command = "help",

    [Parameter(Position=1)]
    [string]$TestName = ""
)

# Ensure UTF-8 output for better visibility of icons
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$RootDir = Get-Location

function Show-Help {
    Write-Host "MCP Scooter Task Runner" -ForegroundColor Green
    Write-Host "========================"
    Write-Host ""
    Write-Host "Usage: ./tasks.ps1 <command>"
    Write-Host ""
    Write-Host "Build & Run:"
    Write-Host "  all               - Validate registry and build"
    Write-Host "  build             - Build the scooter binary"
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
    Write-Host "  release           - Tag and push a stable release"
    Write-Host "  release-beta      - Tag and push a beta release"
    Write-Host ""
    Write-Host "Helper:"
    Write-Host "  test-run <name>   - Run specific test by name"
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
        ./validate-registry.exe appdata/registry
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✓ Registry is valid." -ForegroundColor Green
        } else {
            Write-Host "✗ Registry validation failed." -ForegroundColor Red
        }
    }

    "validate-strict" {
        & ./tasks.ps1 build-validator
        ./validate-registry.exe -strict appdata/registry
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
        $Version = Read-Host "Enter version (e.g., 1.0.0)"
        git tag -a "v$Version" -m "Release v$Version"
        git push origin "v$Version"
        Write-Host "GitHub Action will now build and release v$Version" -ForegroundColor Green
    }

    "release-beta" {
        $Version = Read-Host "Enter beta version (e.g., 1.0.0-beta.1)"
        git tag -a "v$Version" -m "Beta release v$Version"
        git push origin "v$Version"
        Write-Host "GitHub Action will now build and release v$Version" -ForegroundColor Green
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
