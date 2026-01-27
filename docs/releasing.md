# Releasing MCP Scooter

This guide explains how to create releases for MCP Scooter using the automated GitHub Actions workflow.

## Prerequisites: Signing Keys Setup

Before your first release, you must set up signing keys for the auto-updater. The updater requires cryptographic signatures to verify update integrity and authenticity.

### 1. Generate Signing Keys

```powershell
# Windows PowerShell
./tasks.ps1 generate-keys
```

```bash
# macOS/Linux
cd desktop && npx tauri signer generate -w ~/.tauri/mcp-scooter.key
```

You'll be prompted to enter a password. **Remember this password!**

### 2. Add Public Key to Configuration

```powershell
# Display the public key
./tasks.ps1 show-pubkey
```

Copy the output and paste it into `desktop/src-tauri/tauri.conf.json`:

```json
{
  "plugins": {
    "updater": {
      "pubkey": "dW50cnVzdGVkIGNvbW1lbnQ6IG1pbm..."
    }
  }
}
```

### 3. Add GitHub Secrets

Go to your repository's **Settings → Secrets and variables → Actions** and add:

| Secret Name | Value |
|-------------|-------|
| `TAURI_SIGNING_PRIVATE_KEY` | Content of `~/.tauri/mcp-scooter.key` |
| `TAURI_SIGNING_PRIVATE_KEY_PASSWORD` | The password you set during generation |

> ⚠️ **Security Notes:**
> - Never commit the private key to the repository
> - Store a backup of the private key securely
> - If you lose the private key, users won't be able to receive updates signed with it

---

## Release Channels

MCP Scooter supports two release channels:

| Channel | Tag Format | Example | Who Gets It |
|---------|------------|---------|-------------|
| **Stable** | `vX.Y.Z` | `v1.0.0`, `v1.2.3` | All users |
| **Beta** | `vX.Y.Z-beta.N` | `v1.0.0-beta.1` | Users with "Include Beta Releases" enabled |

Other prerelease formats also work: `v1.0.0-alpha.1`, `v1.0.0-rc.1`

## Quick Reference

### Using Task Runner (Recommended)

The task runner handles version bumping, committing, tagging, and pushing automatically:

```powershell
# Windows PowerShell

# Release a stable version (e.g., 1.0.0)
./tasks.ps1 release 1.0.0

# Release a beta version (e.g., 1.0.0-beta.1)
./tasks.ps1 release-beta 1.0.0-beta.1

# Just update version numbers without releasing
./tasks.ps1 set-version 1.0.0
```

```bash
# macOS/Linux
make release          # Interactive prompt for stable version
make release-beta     # Interactive prompt for beta version
```

### Manual Release (Advanced)

If you prefer manual control:

```bash
# 1. Update version in config files (all three must match!)
#    - desktop/src-tauri/tauri.conf.json
#    - desktop/package.json  
#    - desktop/src-tauri/Cargo.toml

# 2. Commit and push
git add -A && git commit -m "chore: bump version to X.Y.Z"
git push origin main

# 3. Create and push the tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

## What Happens After You Push a Tag

1. **GitHub Actions triggers** the Release workflow
2. **Tests run** (Go tests + registry validation)
3. **Builds start** in parallel for:
   - Windows (x64)
   - macOS (Intel x64)
   - macOS (Apple Silicon ARM64)
   - Linux (x64)
4. **Draft release created** with all installers attached
5. **Updater manifests updated** (`latest.json` for stable, `beta.json` for beta)

## After the Build Completes

1. Go to [GitHub Releases](https://github.com/mcp-scooter/scooter/releases)
2. Find your **draft** release
3. Review the release notes and installers
4. Click **"Publish release"** to make it public

## Version Numbering

Follow [Semantic Versioning](https://semver.org/):

- **MAJOR** (1.0.0 → 2.0.0): Breaking changes
- **MINOR** (1.0.0 → 1.1.0): New features, backwards compatible
- **PATCH** (1.0.0 → 1.0.1): Bug fixes

For prereleases:
- `v1.0.0-alpha.1` → Early development, unstable
- `v1.0.0-beta.1` → Feature complete, testing phase
- `v1.0.0-rc.1` → Release candidate, final testing

## Updating Version Numbers

Before creating a release tag, update the version in these files:

1. `desktop/src-tauri/tauri.conf.json` → `"version": "X.Y.Z"`
2. `desktop/package.json` → `"version": "X.Y.Z"`
3. `desktop/src-tauri/Cargo.toml` → `version = "X.Y.Z"`

Or use the Makefile shortcut (if available):
```bash
make release-beta  # Interactive prompt for version
make release       # Interactive prompt for stable version
```

## Troubleshooting

### Build Failed

1. Check the [Actions tab](https://github.com/mcp-scooter/scooter/actions) for error logs
2. Common issues:
   - Go tests failing → Fix tests first
   - Missing dependencies → Check workflow file
   - Rust compilation errors → Check `desktop/src-tauri/`

### Signing Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `Error signing` or `Missing comment in secret key` | Private key not set or malformed | Verify `TAURI_SIGNING_PRIVATE_KEY` secret contains the full key content |
| `incorrect updater private key password` | Wrong password | Check `TAURI_SIGNING_PRIVATE_KEY_PASSWORD` secret |
| `pubkey must not be empty` | Empty pubkey in config | Run `./tasks.ps1 show-pubkey` and add to `tauri.conf.json` |
| Build works locally but fails in CI | Missing GitHub secrets | Add both `TAURI_SIGNING_PRIVATE_KEY` and `TAURI_SIGNING_PRIVATE_KEY_PASSWORD` secrets |

### Delete a Tag (if needed)

```bash
# Delete local tag
git tag -d v0.0.1-beta.1

# Delete remote tag
git push origin --delete v0.0.1-beta.1
```

### Re-run a Failed Build

1. Go to the failed workflow run in GitHub Actions
2. Click "Re-run all jobs"

## Auto-Updates

Users receive updates based on their settings:

- **"Include Beta Releases" OFF** → Only stable releases (`latest.json`)
- **"Include Beta Releases" ON** → All releases including beta (`beta.json`)

The app checks for updates on startup and periodically while running.

## Files Involved

| File | Purpose |
|------|---------|
| `.github/workflows/release.yml` | Release automation workflow (builds, signs, publishes) |
| `desktop/src-tauri/tauri.conf.json` | App version, updater config, and **public key** |
| `desktop/package.json` | npm package version (must match tauri.conf.json) |
| `desktop/src-tauri/Cargo.toml` | Rust crate version (must match tauri.conf.json) |
| `desktop/src-tauri/src/lib.rs` | Update checking logic |
| `tasks.ps1` | Windows task runner with release commands |
| `Makefile` | macOS/Linux task runner with release commands |

### Signing Key Locations (Local Machine Only)

| File | Purpose |
|------|---------|
| `~/.tauri/mcp-scooter.key` | **Private key** - Used to sign updates (KEEP SECRET!) |
| `~/.tauri/mcp-scooter.key.pub` | **Public key** - Copy to `tauri.conf.json` |
