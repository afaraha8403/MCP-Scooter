# ==============================================================================
# MCP Scooter Tasks Helper
# ==============================================================================
#
# A PowerShell task runner for common development operations.
#
# USAGE:
#   ./tasks.ps1 <command> [args]
#
# EXAMPLES:
#   ./tasks.ps1 help              # Show all available commands
#   ./tasks.ps1 dev               # Run in development mode
#   ./tasks.ps1 build             # Build the scooter binary
#   ./tasks.ps1 test              # Run all tests
#   ./tasks.ps1 release 1.0.0     # Create a stable release
#
# ==============================================================================

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

# ==============================================================================
# HELPER FUNCTIONS
# ==============================================================================

# ------------------------------------------------------------------------------
# Update-Version
# ------------------------------------------------------------------------------
# Updates the version number in all configuration files that need to stay in sync.
#
# FILES UPDATED:
#   - desktop/src-tauri/tauri.conf.json  (Tauri app version)
#   - desktop/package.json               (npm package version)
#   - desktop/src-tauri/Cargo.toml       (Rust crate version)
#
# IMPORTANT: These three files MUST have matching versions for the build to work
# correctly. The GitHub Actions release workflow reads the version from
# tauri.conf.json to create the release tag.
# ------------------------------------------------------------------------------
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
    
    # Update Cargo.toml (simple regex replace for the package version only)
    # Note: This regex targets the first version field which is the package version
    $cargoToml = Get-Content "desktop/src-tauri/Cargo.toml" -Raw
    $cargoToml = $cargoToml -replace '(^\[package\][\s\S]*?version = ")[^"]*(")', "`${1}$NewVersion`${2}"
    Set-Content "desktop/src-tauri/Cargo.toml" $cargoToml -NoNewline
    
    Write-Host "Version updated to $NewVersion" -ForegroundColor Green
}

# ------------------------------------------------------------------------------
# Show-Help
# ------------------------------------------------------------------------------
# Displays the help menu with all available commands and their descriptions.
# ------------------------------------------------------------------------------
function Show-Help {
    Write-Host ""
    Write-Host "  MCP Scooter Task Runner" -ForegroundColor Green
    Write-Host "  =======================" -ForegroundColor Green
    Write-Host ""
    Write-Host "  Usage: ./tasks.ps1 <command> [args]"
    Write-Host ""
    Write-Host "  BUILD & RUN" -ForegroundColor Yellow
    Write-Host "  -----------"
    Write-Host "    all               - Validate registry and build"
    Write-Host "    build             - Build the scooter Go binary"
    Write-Host "    build-installer   - Build Windows MSI/NSIS installers (Tauri)"
    Write-Host "    dev               - Run in development mode"
    Write-Host "    clean             - Clean build artifacts"
    Write-Host "    deps              - Install all dependencies (Go + npm)"
    Write-Host ""
    Write-Host "  TESTING" -ForegroundColor Yellow
    Write-Host "  -------"
    Write-Host "    test              - Run all tests"
    Write-Host "    test-unit         - Run all unit tests (verbose)"
    Write-Host "    test-registry     - Test registry validation logic"
    Write-Host "    test-discovery    - Test discovery engine"
    Write-Host "    test-profile      - Test profile management"
    Write-Host "    test-api          - Test API/SSE server"
    Write-Host "    test-integration  - Test client integrations"
    Write-Host "    test-coverage     - Generate HTML coverage report"
    Write-Host "    test-meta-mcp     - Meta-MCP lifecycle tests"
    Write-Host "    test-run <name>   - Run specific test by name pattern"
    Write-Host ""
    Write-Host "  VALIDATION" -ForegroundColor Yellow
    Write-Host "  ----------"
    Write-Host "    validate          - Validate registry JSON files"
    Write-Host "    validate-strict   - Validate with warnings as errors"
    Write-Host ""
    Write-Host "  CI & CODE QUALITY" -ForegroundColor Yellow
    Write-Host "  -----------------"
    Write-Host "    ci                - Run standard CI checks"
    Write-Host "    ci-full           - Run full CI checks with coverage"
    Write-Host "    pre-commit        - Quick check before committing"
    Write-Host "    fmt               - Format code (Go + frontend)"
    Write-Host "    lint              - Lint Go code"
    Write-Host ""
    Write-Host "  RELEASE" -ForegroundColor Yellow
    Write-Host "  -------"
    Write-Host "    release [version]      - Create a stable release (v1.0.0)"
    Write-Host "    release-beta [version] - Create a beta release (v1.0.0-beta.1)"
    Write-Host "    set-version <version>  - Update version only (no commit/tag)"
    Write-Host ""
    Write-Host "  SIGNING KEYS (for auto-updater)" -ForegroundColor Yellow
    Write-Host "  --------------------------------"
    Write-Host "    generate-keys     - Generate Tauri signing keys for updates"
    Write-Host "    show-pubkey       - Display the public key to add to config"
    Write-Host ""
    Write-Host "  EXAMPLES" -ForegroundColor Cyan
    Write-Host "  --------"
    Write-Host "    ./tasks.ps1 release 1.0.0"
    Write-Host "    ./tasks.ps1 release-beta 1.0.0-beta.2"
    Write-Host "    ./tasks.ps1 set-version 1.0.0"
    Write-Host "    ./tasks.ps1 test-run TestDiscovery"
    Write-Host ""
    Write-Host "  FIRST-TIME SETUP" -ForegroundColor Magenta
    Write-Host "  -----------------"
    Write-Host "  Before creating your first release, you need to set up signing keys:"
    Write-Host ""
    Write-Host "    1. ./tasks.ps1 generate-keys    # Creates signing keypair"
    Write-Host "    2. ./tasks.ps1 show-pubkey      # Copy this to tauri.conf.json"
    Write-Host "    3. Add secrets to GitHub:       # Settings > Secrets > Actions"
    Write-Host "       - TAURI_SIGNING_PRIVATE_KEY"
    Write-Host "       - TAURI_SIGNING_PRIVATE_KEY_PASSWORD"
    Write-Host ""
}

# ==============================================================================
# COMMAND ROUTER
# ==============================================================================

switch ($Command) {
    # ==========================================================================
    # BUILD COMMANDS
    # ==========================================================================
    
    "all" {
        # Run validation and build in sequence
        & ./tasks.ps1 validate
        & ./tasks.ps1 build
    }

    "build" {
        # Build the Go backend binary
        Write-Host "Building Scooter..." -ForegroundColor Cyan
        go build -o scooter.exe ./cmd/scooter
    }

    "build-installer" {
        # ----------------------------------------------------------------------
        # Build Windows Installer
        # ----------------------------------------------------------------------
        # This builds the full Tauri desktop application including:
        #   1. Go backend (as a sidecar binary)
        #   2. React frontend (bundled into the app)
        #   3. Tauri shell (Rust WebView wrapper)
        #
        # OUTPUT:
        #   - desktop/src-tauri/target/release/bundle/msi/   (MSI installer)
        #   - desktop/src-tauri/target/release/bundle/nsis/  (NSIS installer)
        #
        # NOTE: This is for local development builds. Release builds are done
        # by GitHub Actions (.github/workflows/release.yml) which also handles
        # signing and cross-platform compilation.
        # ----------------------------------------------------------------------
        Write-Host "Building Windows Installer..." -ForegroundColor Cyan
        
        # Step 1: Build Go backend for Tauri bundle
        # The binary MUST be named with the target triple suffix for Tauri's
        # externalBin feature to find it. See: https://v2.tauri.app/develop/sidecar/
        Write-Host "  [1/3] Building Go backend..." -ForegroundColor Gray
        $env:GOOS = "windows"
        $env:GOARCH = "amd64"
        
        # Ensure binaries directory exists
        if (-not (Test-Path "desktop/src-tauri/binaries")) {
            New-Item -ItemType Directory -Path "desktop/src-tauri/binaries" -Force | Out-Null
        }
        
        # Build with target triple suffix (REQUIRED by Tauri sidecar)
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
        # This compiles the Rust shell, bundles the frontend, and creates installers
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
            Write-Host ""
            Write-Host "Common issues:" -ForegroundColor Yellow
            Write-Host "  - Missing signing keys: Run './tasks.ps1 generate-keys'" -ForegroundColor Gray
            Write-Host "  - Empty pubkey: Run './tasks.ps1 show-pubkey' and update tauri.conf.json" -ForegroundColor Gray
            Write-Host "  - Missing Rust: Install from https://rustup.rs" -ForegroundColor Gray
            exit 1
        }
    }

    "build-validator" {
        # Build the registry validation tool
        go build -o validate-registry.exe ./cmd/validate-registry
    }

    # ==========================================================================
    # TEST COMMANDS
    # ==========================================================================

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

    # ==========================================================================
    # VALIDATION COMMANDS
    # ==========================================================================

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

    # ==========================================================================
    # CODE QUALITY COMMANDS
    # ==========================================================================

    "fmt" {
        go fmt ./...
        Set-Location desktop
        npm run format 2>$null
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

    # ==========================================================================
    # CI COMMANDS
    # ==========================================================================

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

    # ==========================================================================
    # RELEASE COMMANDS
    # ==========================================================================

    "release" {
        # ----------------------------------------------------------------------
        # Create a Stable Release
        # ----------------------------------------------------------------------
        # This command:
        #   1. Updates version in all config files
        #   2. Commits the version bump
        #   3. Creates and pushes a version tag (e.g., v1.0.0)
        #   4. Triggers GitHub Actions to build and publish the release
        #
        # PREREQUISITES:
        #   - Signing keys must be generated: ./tasks.ps1 generate-keys
        #   - Public key must be in tauri.conf.json
        #   - Private key must be in GitHub Secrets
        # ----------------------------------------------------------------------
        if (-not $Version) {
            $Version = Read-Host "Enter version (e.g., 1.0.0)"
        }
        
        # Validate version format
        if (-not ($Version -match '^\d+\.\d+\.\d+$')) {
            Write-Host "Error: Invalid version format. Use semantic versioning (e.g., 1.0.0)" -ForegroundColor Red
            exit 1
        }
        
        # Update version in all config files
        Update-Version $Version
        
        # Commit version bump
        git add desktop/src-tauri/tauri.conf.json desktop/package.json desktop/src-tauri/Cargo.toml
        git commit -m "chore: bump version to $Version"
        git push origin main
        
        # Create and push tag (this triggers the release workflow)
        git tag -a "v$Version" -m "Release v$Version"
        git push origin "v$Version"
        
        Write-Host ""
        Write-Host "✓ Release v$Version initiated!" -ForegroundColor Green
        Write-Host ""
        Write-Host "GitHub Actions will now:" -ForegroundColor Cyan
        Write-Host "  1. Run tests"
        Write-Host "  2. Build installers for Windows, macOS, and Linux"
        Write-Host "  3. Sign update artifacts"
        Write-Host "  4. Create a draft release"
        Write-Host ""
        Write-Host "Monitor progress at: https://github.com/afaraha8403/MCP-Scooter/actions" -ForegroundColor Gray
    }

    "release-beta" {
        # ----------------------------------------------------------------------
        # Create a Beta Release
        # ----------------------------------------------------------------------
        # Beta releases use tags like v1.0.0-beta.1, v1.0.0-alpha.2, v1.0.0-rc.1
        # They are marked as pre-releases on GitHub and distributed through
        # a separate update channel.
        # ----------------------------------------------------------------------
        if (-not $Version) {
            $Version = Read-Host "Enter beta version (e.g., 1.0.0-beta.1)"
        }
        
        # Validate beta version format
        if (-not ($Version -match '^\d+\.\d+\.\d+-(alpha|beta|rc)\.\d+$')) {
            Write-Host "Error: Invalid beta version format." -ForegroundColor Red
            Write-Host "Use: X.Y.Z-beta.N, X.Y.Z-alpha.N, or X.Y.Z-rc.N" -ForegroundColor Yellow
            exit 1
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
        
        Write-Host ""
        Write-Host "✓ Beta release v$Version initiated!" -ForegroundColor Green
        Write-Host ""
        Write-Host "This release will be marked as a PRE-RELEASE on GitHub." -ForegroundColor Yellow
    }

    "set-version" {
        # ----------------------------------------------------------------------
        # Update Version Without Releasing
        # ----------------------------------------------------------------------
        # Use this when you want to change the version number without creating
        # a release. Useful for development or preparing for a future release.
        # ----------------------------------------------------------------------
        if (-not $Version) {
            $Version = Read-Host "Enter version (e.g., 1.0.0)"
        }
        Update-Version $Version
        Write-Host ""
        Write-Host "✓ Version updated to $Version in all config files." -ForegroundColor Green
        Write-Host ""
        Write-Host "Files updated:" -ForegroundColor Gray
        Write-Host "  - desktop/src-tauri/tauri.conf.json"
        Write-Host "  - desktop/package.json"
        Write-Host "  - desktop/src-tauri/Cargo.toml"
        Write-Host ""
        Write-Host "Note: Changes are NOT committed. Run 'git add' and 'git commit' manually." -ForegroundColor Yellow
    }

    # ==========================================================================
    # SIGNING KEY COMMANDS
    # ==========================================================================

    "generate-keys" {
        # ----------------------------------------------------------------------
        # Generate Tauri Signing Keys
        # ----------------------------------------------------------------------
        # The auto-updater requires cryptographic signatures to verify that
        # updates are authentic. This command generates a keypair:
        #
        #   - PRIVATE KEY: Used to sign updates during CI builds
        #                  Must be stored as a GitHub Secret
        #                  NEVER commit this to the repository!
        #
        #   - PUBLIC KEY:  Used by the app to verify signatures
        #                  Must be added to tauri.conf.json
        #                  Safe to commit to the repository
        #
        # STORAGE LOCATION:
        #   Keys are saved to: ~/.tauri/mcp-scooter.key (private)
        #                      ~/.tauri/mcp-scooter.key.pub (public)
        #
        # NEXT STEPS after running this command:
        #   1. Run './tasks.ps1 show-pubkey' to get the public key
        #   2. Add the public key to desktop/src-tauri/tauri.conf.json
        #   3. Add the private key to GitHub Secrets as TAURI_SIGNING_PRIVATE_KEY
        #   4. Add the password to GitHub Secrets as TAURI_SIGNING_PRIVATE_KEY_PASSWORD
        # ----------------------------------------------------------------------
        Write-Host ""
        Write-Host "  Generating Tauri Signing Keys" -ForegroundColor Green
        Write-Host "  =============================" -ForegroundColor Green
        Write-Host ""
        Write-Host "  This will create a keypair for signing auto-updates." -ForegroundColor Gray
        Write-Host "  You will be prompted to enter a password for the private key." -ForegroundColor Gray
        Write-Host ""
        
        # Ensure the .tauri directory exists
        $tauriDir = "$env:USERPROFILE\.tauri"
        if (-not (Test-Path $tauriDir)) {
            New-Item -ItemType Directory -Path $tauriDir -Force | Out-Null
        }
        
        # Check if keys already exist
        $keyPath = "$tauriDir\mcp-scooter.key"
        if (Test-Path $keyPath) {
            Write-Host "  WARNING: Keys already exist at $keyPath" -ForegroundColor Yellow
            $confirm = Read-Host "  Overwrite existing keys? (y/N)"
            if ($confirm -ne "y" -and $confirm -ne "Y") {
                Write-Host "  Aborted." -ForegroundColor Gray
                exit 0
            }
        }
        
        # Generate keys using Tauri CLI
        # Note: We use npx directly because `npm run tauri ... -- -w` incorrectly
        # interprets -w as npm's --workspace flag instead of passing it to tauri
        Set-Location desktop
        npx tauri signer generate -w "$keyPath"
        $result = $LASTEXITCODE
        Set-Location $RootDir
        
        if ($result -eq 0) {
            Write-Host ""
            Write-Host "  ✓ Keys generated successfully!" -ForegroundColor Green
            Write-Host ""
            Write-Host "  Key files:" -ForegroundColor Cyan
            Write-Host "    Private: $keyPath" -ForegroundColor Gray
            Write-Host "    Public:  $keyPath.pub" -ForegroundColor Gray
            Write-Host ""
            Write-Host "  NEXT STEPS:" -ForegroundColor Yellow
            Write-Host "  -----------" -ForegroundColor Yellow
            Write-Host "  1. Run: ./tasks.ps1 show-pubkey" -ForegroundColor White
            Write-Host "     Copy the output and update tauri.conf.json" -ForegroundColor Gray
            Write-Host ""
            Write-Host "  2. Add GitHub Secrets (Settings > Secrets > Actions):" -ForegroundColor White
            Write-Host "     - TAURI_SIGNING_PRIVATE_KEY = contents of $keyPath" -ForegroundColor Gray
            Write-Host "     - TAURI_SIGNING_PRIVATE_KEY_PASSWORD = your password" -ForegroundColor Gray
            Write-Host ""
            Write-Host "  IMPORTANT: Never commit the private key to the repository!" -ForegroundColor Red
            Write-Host ""
        } else {
            Write-Host "  ✗ Key generation failed" -ForegroundColor Red
            Write-Host "  Make sure Tauri CLI is installed: npm install -g @tauri-apps/cli" -ForegroundColor Yellow
            exit 1
        }
    }

    "show-pubkey" {
        # ----------------------------------------------------------------------
        # Display the Public Key
        # ----------------------------------------------------------------------
        # Shows the public key that needs to be added to tauri.conf.json.
        # The key is displayed in a format ready to copy-paste.
        # ----------------------------------------------------------------------
        $keyPath = "$env:USERPROFILE\.tauri\mcp-scooter.key.pub"
        
        if (-not (Test-Path $keyPath)) {
            Write-Host ""
            Write-Host "  ✗ Public key not found at: $keyPath" -ForegroundColor Red
            Write-Host ""
            Write-Host "  Run './tasks.ps1 generate-keys' first to create a keypair." -ForegroundColor Yellow
            Write-Host ""
            exit 1
        }
        
        $pubkey = Get-Content $keyPath -Raw
        $pubkey = $pubkey.Trim()
        
        Write-Host ""
        Write-Host "  Your Public Key" -ForegroundColor Green
        Write-Host "  ===============" -ForegroundColor Green
        Write-Host ""
        Write-Host "  Copy this key and paste it into:" -ForegroundColor Gray
        Write-Host "  desktop/src-tauri/tauri.conf.json > plugins > updater > pubkey" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  ┌──────────────────────────────────────────────────────────────┐" -ForegroundColor Cyan
        Write-Host "  │ $pubkey" -ForegroundColor White
        Write-Host "  └──────────────────────────────────────────────────────────────┘" -ForegroundColor Cyan
        Write-Host ""
        
        # Also copy to clipboard if possible
        try {
            $pubkey | Set-Clipboard
            Write-Host "  ✓ Copied to clipboard!" -ForegroundColor Green
        } catch {
            Write-Host "  (Could not copy to clipboard automatically)" -ForegroundColor Gray
        }
        Write-Host ""
    }

    # ==========================================================================
    # HELP
    # ==========================================================================

    "help" {
        Show-Help
    }

    default {
        Write-Host ""
        Write-Host "  Unknown command: $Command" -ForegroundColor Red
        Write-Host "  Run './tasks.ps1 help' for available commands." -ForegroundColor Gray
        Write-Host ""
        exit 1
    }
}
